package parity

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/nelsonwerd/countershape/internal/canon"
	programv1 "github.com/nelsonwerd/countershape/internal/emit/node/program/v1"
)

const copiedManifestParserDriver = `
const chunks = [];
try {
  for await (const chunk of process.stdin) chunks.push(Buffer.from(chunk));
  const manifest = parseManifestEnvelope(Buffer.concat(chunks));
  process.stdout.write(JSON.stringify(manifest) + "\n");
} catch {
  process.exitCode = 1;
}
`

// TestContractParityManifestMatchesCopiedEntrypointParser is deliberately not
// another call through the parity runner. It copies the private parser prefix
// from the exact embedded contract entrypoint and appends test-only stdin
// instrumentation. This keeps the production asset closed while making the
// corpus compare three independently wired paths: Go, parity Node, and the
// parser the generated contract actually carries.
func TestContractParityManifestMatchesCopiedEntrypointParser(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	source, err := programv1.Bytes(programv1.ContractTestPath)
	if err != nil {
		t.Fatal(err)
	}
	marker := []byte("\nfunction rawDigest(bytes) {")
	if bytes.Count(source, marker) != 1 {
		t.Fatalf("contract entrypoint has %d rawDigest markers; want exactly one", bytes.Count(source, marker))
	}
	prefixEnd := bytes.Index(source, marker) + 1
	if prefixEnd < 1 ||
		bytes.Count(source[:prefixEnd], []byte("class CanonicalManifestParser")) != 1 ||
		bytes.Count(source[:prefixEnd], []byte("function parseManifestEnvelope(bytes) {")) != 1 ||
		bytes.Contains(source[:prefixEnd], []byte("\ntest(")) {
		t.Fatal("contract entrypoint parser prefix is not instrumentable")
	}
	instrumented := append(append([]byte(nil), source[:prefixEnd]...), copiedManifestParserDriver...)
	directory := t.TempDir()
	path := filepath.Join(directory, "copied-manifest-parser.mjs")
	if err := os.WriteFile(path, instrumented, 0o600); err != nil {
		t.Fatal(err)
	}

	manifestVectors := 0
	for _, vector := range loadCorpus(t) {
		if vector.operation != ParseManifestEnvelope {
			continue
		}
		manifestVectors++
		vector := vector
		t.Run(vector.id, func(t *testing.T) {
			encoded, present := vector.input.LookupMember("bytes_base64")
			if !present {
				t.Fatal("manifest vector has no bytes_base64 input")
			}
			envelope, ok := canonicalBase64Bytes(encoded)
			if !ok {
				t.Fatal("manifest vector does not carry exact standard base64")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, node, path)
			command.Dir = directory
			command.Env = []string{"LANG=C", "LC_ALL=C", "NO_COLOR=1", "TZ=UTC"}
			command.Stdin = bytes.NewReader(envelope)
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			runErr := command.Run()

			want, accepted := expectedManifestFrame(t, vector.expected)
			if accepted {
				if ctx.Err() != nil || runErr != nil || stderr.Len() != 0 || !bytes.Equal(stdout.Bytes(), want) {
					t.Fatalf("copied parser = err=%v stdout=%q stderr=%q; want exact %q", runErr, stdout.Bytes(), stderr.Bytes(), want)
				}
				return
			}
			var exitError *exec.ExitError
			if ctx.Err() != nil || !errors.As(runErr, &exitError) || exitError.ExitCode() != 1 || stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatalf("copied parser refusal = err=%v stdout=%q stderr=%q; want nonzero and empty streams", runErr, stdout.Bytes(), stderr.Bytes())
			}
		})
	}
	if manifestVectors != corpusOperationCounts[ParseManifestEnvelope] {
		t.Fatalf("copied parser exercised %d manifest vectors; want %d", manifestVectors, corpusOperationCounts[ParseManifestEnvelope])
	}
}

func expectedManifestFrame(t testing.TB, expected []byte) ([]byte, bool) {
	t.Helper()
	value, err := canon.Parse(expected)
	if err != nil {
		t.Fatal(err)
	}
	fields, ok := exactObject(value, "status", "value")
	if !ok {
		fields, ok := exactObject(value, "code", "status")
		if !ok {
			t.Fatal("manifest expectation is outside the closed OK/REFUSED roster")
		}
		code, codeOK := fields[0].Text()
		status, statusOK := fields[1].Text()
		if !codeOK || code != "INVALID_MANIFEST" || !statusOK || status != "REFUSED" {
			t.Fatal("manifest refusal expectation is not INVALID_MANIFEST/REFUSED")
		}
		return nil, false
	}
	status, ok := fields[0].Text()
	if !ok || status != "OK" {
		t.Fatal("manifest OK expectation has an invalid status")
	}
	manifest, present := fields[1].LookupMember("manifest")
	if !present {
		t.Fatal("manifest OK expectation has no manifest value")
	}
	exact, err := manifest.CanonicalChecked()
	if err != nil {
		t.Fatal(err)
	}
	return append(exact, '\n'), true
}

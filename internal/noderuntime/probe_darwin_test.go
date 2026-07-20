//go:build darwin && arm64 && cgo

package noderuntime

import (
	"bytes"
	"strings"
	"testing"
)

func TestC3NodeRuntimeProbeParserIsClosed(t *testing.T) {
	path := "/opt/homebrew/Cellar/node/25.2.1/bin/node"
	valid := []byte(`{"architecture":"arm64","measured_process_exec_path":"/opt/homebrew/Cellar/node/25.2.1/bin/node","platform":"darwin","version":"v25.2.1"}` + "\n")
	result, err := parseProbeOutput(valid, path)
	if err != nil || !result.valid(path) {
		t.Fatalf("valid exact probe output refused: %v", err)
	}
	maxVersion := "v1." + strings.Repeat("0", 125)
	maxOutput := bytes.Replace(valid, []byte("v25.2.1"), []byte(maxVersion), 1)
	if result, err := parseProbeOutput(maxOutput, path); err != nil || !result.valid(path) || len(result.version) != 128 {
		t.Fatalf("128-byte exact version was not admitted: result=%+v err=%v", result, err)
	}
	for name, value := range map[string][]byte{
		"no lf":              bytes.TrimSuffix(valid, []byte{'\n'}),
		"stderr form":        append(append([]byte(nil), valid...), 'x'),
		"unknown":            []byte(`{"architecture":"arm64","extra":"x","measured_process_exec_path":"/opt/homebrew/Cellar/node/25.2.1/bin/node","platform":"darwin","version":"v25.2.1"}` + "\n"),
		"duplicate":          []byte(`{"architecture":"arm64","architecture":"arm64","measured_process_exec_path":"/opt/homebrew/Cellar/node/25.2.1/bin/node","platform":"darwin","version":"v25.2.1"}` + "\n"),
		"wrong path":         bytes.Replace(valid, []byte("25.2.1/bin/node"), []byte("25.2.2/bin/node"), 1),
		"wrong arch":         bytes.Replace(valid, []byte("arm64"), []byte("x64"), 1),
		"wrong platform":     bytes.Replace(valid, []byte("darwin"), []byte("linux"), 1),
		"crlf":               bytes.Replace(valid, []byte{'\n'}, []byte{'\r', '\n'}, 1),
		"missing minor":      bytes.Replace(valid, []byte("v25.2.1"), []byte("v25"), 1),
		"empty minor":        bytes.Replace(valid, []byte("v25.2.1"), []byte("v25."), 1),
		"nonnumeric minor":   bytes.Replace(valid, []byte("v25.2.1"), []byte("v25.x"), 1),
		"zero major":         bytes.Replace(valid, []byte("v25.2.1"), []byte("v0.2.1"), 1),
		"leading zero major": bytes.Replace(valid, []byte("v25.2.1"), []byte("v025.2.1"), 1),
		"major above safe":   bytes.Replace(valid, []byte("v25.2.1"), []byte("v9007199254740992.0"), 1),
		"version over bound": bytes.Replace(valid, []byte("v25.2.1"), []byte("v1."+strings.Repeat("0", 126)), 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseProbeOutput(value, path); err == nil {
				t.Fatal("hostile probe output was accepted")
			}
		})
	}
}

func TestC3NodeRuntimeProbeWritersDrainAfterBounds(t *testing.T) {
	writer := &boundedBuffer{limit: 4}
	input := []byte("abcdefgh")
	if count, err := writer.Write(input); err != nil || count != len(input) || !writer.overflow || writer.buffer.String() != "abcd" {
		t.Fatalf("bounded writer did not drain and mark overflow: count=%d err=%v state=%+v", count, err, writer)
	}
}

func TestC3NodeRuntimeProbeProgramDigestIsExact(t *testing.T) {
	const exactProgram = `process.stdout.write(JSON.stringify({architecture:process.arch,measured_process_exec_path:process.execPath,platform:process.platform,version:process.version})+"\n");`
	const exactDigest = "sha256:6d01062a7082e3302c3bf5a74470644ff151ca055ed8e920897d8b4af27f8a7f"
	if nodeProbeProgram != exactProgram || !nodeProbeDigest().Valid() || nodeProbeDigest().String() != exactDigest {
		t.Fatalf("probe program or typed digest differs: program=%q digest=%s", nodeProbeProgram, nodeProbeDigest())
	}
}

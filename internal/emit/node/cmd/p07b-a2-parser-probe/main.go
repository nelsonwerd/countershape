// Command p07b-a2-parser-probe recovers one inert bundle in a fresh process.
// It receives only the compiler probe's two-line digest/body frame on stdin;
// it has no compiler input, corpus path, or repository lookup.
package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/emit/node/model"
)

type recoveredFile struct {
	Path          string `json:"path"`
	Mode          string `json:"mode"`
	ByteCount     int    `json:"byte_count"`
	ByteSHA256    string `json:"byte_sha256"`
	ContentBase64 string `json:"content_base64"`
}

type recoveredBundle struct {
	SchemaVersion        string          `json:"schema_version"`
	Kind                 string          `json:"kind"`
	BundleDigest         string          `json:"bundle_digest"`
	PortableSourceDigest string          `json:"portable_source_digest"`
	PortableSourceBase64 string          `json:"portable_source_base64"`
	Files                []recoveredFile `json:"files"`
}

func main() {
	exact, expected, err := readFrame()
	if err != nil {
		fail(err)
	}
	bundle, err := model.ParseContractBundle(exact, expected)
	if err != nil || !bundle.Valid() {
		fail(fmt.Errorf("strict bundle recovery failed: %w", err))
	}
	files := bundle.Files()
	recovered := recoveredBundle{
		SchemaVersion:        "countershape.p07b-a2.parser-probe/v1",
		Kind:                 "RecoveredContractBundle",
		BundleDigest:         bundle.Digest().String(),
		PortableSourceDigest: bundle.PortableSource().Digest().String(),
		PortableSourceBase64: base64.StdEncoding.EncodeToString(bundle.PortableSource().CanonicalBytes()),
		Files:                make([]recoveredFile, len(files)),
	}
	for index, file := range files {
		recovered.Files[index] = recoveredFile{
			Path: file.Path(), Mode: file.Mode(), ByteCount: file.ByteCount(),
			ByteSHA256: file.ByteSHA256().String(), ContentBase64: base64.StdEncoding.EncodeToString(file.Content()),
		}
	}
	canonical, err := canon.CanonicalizeTyped(recovered)
	if err != nil {
		fail(err)
	}
	if _, err := os.Stdout.Write(append(canonical, '\n')); err != nil {
		fail(err)
	}
}

func readFrame() ([]byte, domain.Digest, error) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64<<10), canon.MaxInputBytes*2)
	lines := make([]string, 0, 2)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > 2 {
			return nil, domain.Digest(""), fmt.Errorf("probe frame has extra lines")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, domain.Digest(""), err
	}
	if len(lines) != 2 || strings.TrimSpace(lines[0]) != lines[0] || strings.TrimSpace(lines[1]) != lines[1] {
		return nil, domain.Digest(""), fmt.Errorf("probe frame must contain exactly two canonical lines")
	}
	digest, err := domain.ParseDigest(lines[0])
	if err != nil {
		return nil, domain.Digest(""), err
	}
	exact, err := base64.StdEncoding.Strict().DecodeString(lines[1])
	if err != nil || base64.StdEncoding.EncodeToString(exact) != lines[1] {
		return nil, domain.Digest(""), fmt.Errorf("probe body is not canonical base64: %w", err)
	}
	return exact, digest, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

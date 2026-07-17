// Package program contains the fixed, checked-in Node-core source assets used
// by the v1 contract compiler. Assets are inert bytes; loading them performs no
// filesystem, process, environment, clock, randomness, or network operation.
package program

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"unicode/utf8"
)

const (
	ContractTestPath = "contract.test.mjs"
	HarnessPath      = "harness.mjs"

	contractTestReviewedRawSHA256 = "sha256:0ded06ad24d9b218fef7776835b26124483d788f2eb49f69d0ffb9806c85e021"
	harnessReviewedRawSHA256      = "sha256:51eb70171f044cd77776d36dceed8101e45c774c3b5f70327fe144061b08bb34"
)

var (
	//go:embed contract.test.mjs
	contractTestSource []byte

	//go:embed harness.mjs
	harnessSource []byte
)

// Bytes returns a defensive copy of one fixed source asset.
func Bytes(path string) ([]byte, error) {
	source, _, err := validatedAsset(path)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), source...), nil
}

// RawSHA256 returns the independently reviewed lowercase raw SHA-256 only
// after proving that the exact embedded bytes still match that fixed pin.
func RawSHA256(path string) (string, error) {
	_, reviewed, err := validatedAsset(path)
	if err != nil {
		return "", err
	}
	return reviewed, nil
}

func validatedAsset(path string) ([]byte, string, error) {
	var source []byte
	var reviewed string
	switch path {
	case ContractTestPath:
		source = contractTestSource
		reviewed = contractTestReviewedRawSHA256
	case HarnessPath:
		source = harnessSource
		reviewed = harnessReviewedRawSHA256
	default:
		return nil, "", fmt.Errorf("unknown Node program asset %q", path)
	}
	if err := validateTextAsset(source); err != nil {
		return nil, "", fmt.Errorf("invalid fixed Node program asset %q: %w", path, err)
	}
	if err := verifyReviewedDigest(source, reviewed); err != nil {
		return nil, "", fmt.Errorf("fixed Node program asset %q: %w", path, err)
	}
	return source, reviewed, nil
}

func verifyReviewedDigest(source []byte, reviewed string) error {
	digest := sha256.Sum256(source)
	actual := "sha256:" + hex.EncodeToString(digest[:])
	if actual != reviewed {
		return fmt.Errorf("differs from its reviewed raw SHA-256")
	}
	return nil
}

func validateTextAsset(source []byte) error {
	if len(source) == 0 || !utf8.Valid(source) || bytes.HasPrefix(source, []byte{0xef, 0xbb, 0xbf}) ||
		bytes.IndexByte(source, 0) >= 0 || bytes.IndexByte(source, '\r') >= 0 || source[len(source)-1] != '\n' {
		return fmt.Errorf("asset is not nonempty BOM-free UTF-8 LF text")
	}
	if len(source) > 1 && source[len(source)-2] == '\n' {
		return fmt.Errorf("asset does not end in exactly one LF")
	}
	return nil
}

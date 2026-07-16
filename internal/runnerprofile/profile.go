// Package runnerprofile owns the cycle-free identity of the closed physical
// runner lineages admitted by portable source reconstruction and execution.
// A digest from this package is structural identity, never evidence that the
// runner was invoked.
package runnerprofile

import (
	"strings"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	NodeToolConstraintV1        = "executed-major-only"
	CLIAdapterVersionV1         = "cli/v1"
	HTTPAdapterVersionV1        = "http/v1"
	CLIRunnerKindV1             = "CLIStudyRunner"
	CLIRunnerLineageV1          = "world.ExecuteCLI/U3/opaque-binding/v1"
	HTTPLegacyRunnerKindV1      = "HTTPStudyRunner"
	HTTPLegacyRunnerLineageV1   = "world.ExecuteHTTP/U4/opaque-binding/inherited-listener/v1"
	HTTPPortableRunnerKindV1    = "HTTPStudyRunner"
	HTTPPortableRunnerLineageV1 = "world.ExecuteHTTP/P07B/opaque-binding/child-bind-pipe-ready/v1"
)

func CLIDigest() (domain.Digest, error) {
	return digest(CLIRunnerKindV1, CLIRunnerLineageV1)
}

func HTTPLegacyDigest() (domain.Digest, error) {
	return digest(HTTPLegacyRunnerKindV1, HTTPLegacyRunnerLineageV1)
}

func HTTPPortableDigest() (domain.Digest, error) {
	return digest(HTTPPortableRunnerKindV1, HTTPPortableRunnerLineageV1)
}

// ValidRepositoryNodeEntrypoint is the exact ASCII grammar shared by the
// portable start authority, source reconstruction, and bundle schema.
func ValidRepositoryNodeEntrypoint(value string) bool {
	if value == "" || len(value) > 4096 || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "-") ||
		strings.Contains(value, "\\") {
		return false
	}
	segments := strings.Split(value, "/")
	for _, segment := range segments {
		if segment == "" || !entrypointFirstByte(segment[0]) {
			return false
		}
		for index := 1; index < len(segment); index++ {
			if !entrypointByte(segment[index]) {
				return false
			}
		}
	}
	return strings.HasSuffix(value, ".js") || strings.HasSuffix(value, ".mjs") || strings.HasSuffix(value, ".cjs")
}

func entrypointFirstByte(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' ||
		value >= '0' && value <= '9' || value == '_' || value == '-'
}

func entrypointByte(value byte) bool {
	return entrypointFirstByte(value) || value == '.'
}

func digest(kind, lineage string) (domain.Digest, error) {
	raw, err := canon.DigestBytes(kind, []byte(lineage))
	if err != nil {
		return "", err
	}
	return domain.ParseDigest(raw.String())
}

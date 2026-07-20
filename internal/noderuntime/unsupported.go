//go:build !darwin || !arm64 || !cgo

package noderuntime

import (
	"context"

	"github.com/nelsonwerd/countershape/internal/domain"
)

func measureExecutable(string) (runtimeSnapshot, error) { return runtimeSnapshot{}, unsupported() }
func resolvePrivateProbeParent(string) (string, probeParentIdentity, error) {
	return "", probeParentIdentity{}, unsupported()
}
func runOwnedProbe(context.Context, string, string) (probeResult, error) {
	return probeResult{}, unsupported()
}
func nodeProbeDigest() domain.Digest { return "" }

func unsupported() error {
	return refuse(CodeUnsupportedPlatform, "Node runtime admission requires Darwin arm64 with cgo", nil)
}

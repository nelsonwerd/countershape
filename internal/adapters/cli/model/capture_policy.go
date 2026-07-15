package model

import (
	"bytes"

	"github.com/nelsonwerd/countershape/internal/domain"
)

const maxCaptureBytes = 16 << 20

type CLICapturePolicyConfig struct {
	StdoutBytes int64
	StderrBytes int64
}

// CLICapturePolicy is cycle-free resolved execution authority. Keeping the
// concrete policy beside CLIExecutionBinding lets the impure world edge prove
// that the plan's digest and numeric caps resolve before any process effect.
type CLICapturePolicy struct {
	digest         domain.Digest
	canonicalBytes []byte
	stdoutBytes    int64
	stderrBytes    int64
}

type capturePolicyChannelIdentity struct {
	Name     string `json:"name"`
	MaxBytes int64  `json:"max_bytes"`
}

type capturePolicyIdentity struct {
	SchemaVersion string                         `json:"schema_version"`
	Kind          string                         `json:"kind"`
	Adapter       string                         `json:"adapter"`
	Version       string                         `json:"version"`
	Channels      []capturePolicyChannelIdentity `json:"channels"`
}

func NewCLICapturePolicy(config CLICapturePolicyConfig) (CLICapturePolicy, error) {
	if config.StdoutBytes < 1 || config.StdoutBytes > maxCaptureBytes ||
		config.StderrBytes < 1 || config.StderrBytes > maxCaptureBytes {
		return CLICapturePolicy{}, refuse(CodeInvalidCapturePolicy, "stdout and stderr limits must be within the closed v1 range")
	}
	identity := capturePolicyIdentity{
		SchemaVersion: domain.SchemaVersion,
		Kind:          "CLICapturePolicy",
		Adapter:       string(domain.AdapterCLI),
		Version:       "cli-capture/v1",
		Channels: []capturePolicyChannelIdentity{
			{Name: "stdout", MaxBytes: config.StdoutBytes},
			{Name: "stderr", MaxBytes: config.StderrBytes},
		},
	}
	digest, canonicalBytes, err := digestTyped("CLICapturePolicy", identity)
	if err != nil {
		return CLICapturePolicy{}, err
	}
	return CLICapturePolicy{
		digest: digest, canonicalBytes: canonicalBytes,
		stdoutBytes: config.StdoutBytes, stderrBytes: config.StderrBytes,
	}, nil
}

func (p CLICapturePolicy) Valid() bool {
	rebuilt, err := NewCLICapturePolicy(CLICapturePolicyConfig{StdoutBytes: p.stdoutBytes, StderrBytes: p.stderrBytes})
	return err == nil && rebuilt.digest == p.digest && bytes.Equal(rebuilt.canonicalBytes, p.canonicalBytes)
}

func (p CLICapturePolicy) Digest() domain.Digest  { return p.digest }
func (p CLICapturePolicy) CanonicalBytes() []byte { return append([]byte(nil), p.canonicalBytes...) }
func (p CLICapturePolicy) StdoutBytes() int64     { return p.stdoutBytes }
func (p CLICapturePolicy) StderrBytes() int64     { return p.stderrBytes }

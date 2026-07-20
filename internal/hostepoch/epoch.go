// Package hostepoch owns the closed Darwin boot-session measurement edge used
// by contract-execution targets. It exports no constructor from raw epoch
// material: authority can only be obtained by measuring the host again.
package hostepoch

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	CodeInvalidMeasurement  = "INVALID_HOST_EPOCH_MEASUREMENT"
	CodeUnstableMeasurement = "UNSTABLE_HOST_EPOCH_MEASUREMENT"
	CodeUnsupportedPlatform = "UNSUPPORTED_HOST_EPOCH_PLATFORM"
)

// Error is a stable refusal. Detail never contains the measured UUID.
type Error struct {
	Code   string
	Detail string
	Cause  error
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Detail)
}

func (e *Error) Unwrap() error { return e.Cause }

func IsCode(err error, code string) bool {
	var target *Error
	return errors.As(err, &target) && target.Code == code
}

type epochSeal struct{ marker byte }

var measuredEpochSeal = &epochSeal{marker: 1}

// Epoch is an opaque result of two matching live measurements. It retains no
// raw UUID; only the domain-separated identity digest crosses this boundary.
type Epoch struct {
	digest domain.Digest
	seal   *epochSeal
}

// Measure reads the fixed Darwin source exactly twice and refuses disagreement.
func Measure(ctx context.Context) (Epoch, error) {
	return measureWithSource(ctx, readBootSessionSample)
}

// Revalidate performs a new two-sample measurement and returns fresh authority
// only when it joins this exact epoch.
func (e Epoch) Revalidate(ctx context.Context) (Epoch, error) {
	return revalidateWithSource(ctx, e, readBootSessionSample)
}

func revalidateWithSource(ctx context.Context, retained Epoch, source epochSource) (Epoch, error) {
	if !retained.Valid() {
		return Epoch{}, refuse(CodeInvalidMeasurement, "epoch authority is zero or forged", nil)
	}
	fresh, err := measureWithSource(ctx, source)
	if err != nil {
		return Epoch{}, err
	}
	if fresh.digest != retained.digest {
		return Epoch{}, refuse(CodeUnstableMeasurement, "host epoch differs from the retained authority", nil)
	}
	return fresh, nil
}

func (e Epoch) Valid() bool { return e.seal == measuredEpochSeal && e.digest.Valid() }

func (e Epoch) Digest() domain.Digest {
	if !e.Valid() {
		return ""
	}
	return e.digest
}

type epochSource func() ([]byte, error)

func measureWithSource(ctx context.Context, source epochSource) (Epoch, error) {
	if ctx == nil || source == nil {
		return Epoch{}, refuse(CodeInvalidMeasurement, "context and fixed measurement source are required", nil)
	}
	if err := ctx.Err(); err != nil {
		return Epoch{}, refuse(CodeInvalidMeasurement, "measurement context is closed", err)
	}
	firstRaw, err := source()
	if err != nil {
		return Epoch{}, refuseSource(err)
	}
	first, err := canonicalUUID(firstRaw)
	if err != nil {
		return Epoch{}, err
	}
	if err := ctx.Err(); err != nil {
		return Epoch{}, refuse(CodeInvalidMeasurement, "measurement context closed between samples", err)
	}
	secondRaw, err := source()
	if err != nil {
		return Epoch{}, refuseSource(err)
	}
	second, err := canonicalUUID(secondRaw)
	if err != nil {
		return Epoch{}, err
	}
	if err := ctx.Err(); err != nil {
		return Epoch{}, refuse(CodeInvalidMeasurement, "measurement context closed after samples", err)
	}
	if !bytes.Equal(first[:], second[:]) {
		return Epoch{}, refuse(CodeUnstableMeasurement, "consecutive host-epoch samples disagree", nil)
	}
	digest, err := canon.DigestBytes("DarwinBootSessionIdentity", first[:])
	if err != nil {
		return Epoch{}, refuse(CodeInvalidMeasurement, "typed host-epoch digest failed", err)
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return Epoch{}, refuse(CodeInvalidMeasurement, "typed host-epoch digest is invalid", err)
	}
	return Epoch{digest: parsed, seal: measuredEpochSeal}, nil
}

func refuseSource(err error) error {
	if IsCode(err, CodeUnsupportedPlatform) {
		return err
	}
	return refuse(CodeInvalidMeasurement, "fixed host-epoch source failed", err)
}

func canonicalUUID(raw []byte) ([36]byte, error) {
	var canonical [36]byte
	if len(raw) != len(canonical) {
		return canonical, refuse(CodeInvalidMeasurement, "boot-session UUID has the wrong byte length", nil)
	}
	nonzero := false
	for index, value := range raw {
		hyphen := index == 8 || index == 13 || index == 18 || index == 23
		if hyphen {
			if value != '-' {
				return [36]byte{}, refuse(CodeInvalidMeasurement, "boot-session UUID hyphen positions differ", nil)
			}
			canonical[index] = value
			continue
		}
		switch {
		case value >= '0' && value <= '9':
			canonical[index] = value
		case value >= 'a' && value <= 'f':
			canonical[index] = value
		case value >= 'A' && value <= 'F':
			canonical[index] = value + ('a' - 'A')
		default:
			return [36]byte{}, refuse(CodeInvalidMeasurement, "boot-session UUID contains a nonhex byte", nil)
		}
		if canonical[index] != '0' {
			nonzero = true
		}
	}
	if !nonzero {
		return [36]byte{}, refuse(CodeInvalidMeasurement, "all-zero boot-session UUID is invalid", nil)
	}
	return canonical, nil
}

func refuse(code, detail string, cause error) error {
	return &Error{Code: code, Detail: detail, Cause: cause}
}

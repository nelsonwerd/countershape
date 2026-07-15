package model

import (
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func digestTyped(kind string, value any) (domain.Digest, []byte, error) {
	digest, canonicalBytes, err := canon.DigestTyped(kind, value)
	if err != nil {
		return "", nil, err
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return "", nil, err
	}
	return parsed, append([]byte(nil), canonicalBytes...), nil
}

func digestExactBytes(kind string, value []byte) (domain.Digest, error) {
	digest, err := canon.DigestBytes(kind, value)
	if err != nil {
		return "", err
	}
	return domain.ParseDigest(digest.String())
}

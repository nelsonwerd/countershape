package canon

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	digestDomainPrefix  = "countershape/v1/"
	digestKindSeparator = byte(0)
)

// Digest is a comparable, immutable SHA-256 value. Its textual form is always
// "sha256:" followed by 64 lowercase hexadecimal digits.
type Digest struct {
	sum [sha256.Size]byte
}

// DigestBytes hashes exact bytes in the named Countershape v1 domain. Callers
// passing semantic JSON should normally use DigestValue so canonicality is
// guaranteed by construction.
func DigestBytes(kind string, exactBytes []byte) (Digest, error) {
	if err := validateDigestKind(kind); err != nil {
		return Digest{}, err
	}
	h := sha256.New()
	_, _ = h.Write([]byte(digestDomainPrefix))
	_, _ = h.Write([]byte(kind))
	_, _ = h.Write([]byte{digestKindSeparator}) // MUTANT_U1_CANON_REMOVE_KIND_SEPARATOR
	_, _ = h.Write(exactBytes)
	var digest Digest
	copy(digest.sum[:], h.Sum(nil))
	return digest, nil
}

func DigestValue(kind string, value Value) (Digest, error) {
	canonical, err := value.CanonicalChecked()
	if err != nil {
		return Digest{}, err
	}
	return DigestBytes(kind, canonical)
}

func validateDigestKind(kind string) error {
	if kind == "" {
		return refusal(CodeInvalidKind, UnknownOffset, "digest kind must be nonempty")
	}
	if offset := firstInvalidUTF8([]byte(kind)); offset >= 0 {
		return refusal(CodeInvalidKind, offset, "digest kind is not valid UTF-8")
	}
	if offset := strings.IndexByte(kind, 0); offset >= 0 {
		return refusal(CodeInvalidKind, offset, "digest kind contains the domain separator")
	}
	for offset, r := range kind {
		if r < 0x20 {
			return refusal(CodeInvalidKind, offset, "digest kind contains a control character")
		}
	}
	return nil
}

func (d Digest) String() string {
	return "sha256:" + hex.EncodeToString(d.sum[:])
}

func (d Digest) Bytes() []byte {
	return append([]byte(nil), d.sum[:]...)
}

func (d Digest) Equal(other Digest) bool {
	return d == other
}

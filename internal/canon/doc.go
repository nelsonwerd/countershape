// Package canon is Countershape's sole identity-bearing JSON authority.
//
// Parse first scans bytes into lossless project-owned tokens. It rejects input
// that ordinary JSON decoders commonly erase or coerce: duplicate object names,
// invalid UTF-8, lone UTF-16 surrogates, negative zero, non-integral number
// spellings, and integers outside the exact JavaScript interoperability range.
// Values are immutable after construction and encode to one deterministic byte
// representation. Object names are ordered by their decoded UTF-8 bytes; no
// Unicode normalization is performed.
package canon

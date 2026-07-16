// Package authority exposes the opaque publication value without exposing its
// issuer. Only packages below internal/confirmation can import the issuer's Go
// internal path.
package authority

import "github.com/nelsonwerd/countershape/internal/confirmation/internal/publication"

type Publication = publication.Authority

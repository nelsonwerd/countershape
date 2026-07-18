// Package authority exposes node-emitter publication values without exposing
// their issuer outside the node emitter subtree.
package authority

import "github.com/nelsonwerd/countershape/internal/emit/node/internal/publication"

type Publication = publication.Authority

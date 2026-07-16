// Package httpfixture provides the dependency-free Node candidates used by
// the U4 cross-tenant invoice study. It is test infrastructure, not product
// execution authority.
package httpfixture

import (
	_ "embed"
	"fmt"

	"github.com/nelsonwerd/countershape/testkit/gitrepo"
)

//go:embed server.mjs
var fixtureProgram []byte

//go:embed server_child_bind.mjs
var portableFixtureProgram []byte

// CandidateRole names one deliberately incompatible invoice authorization
// behavior. The byte-identical server program reads only this immutable role
// file to select the candidate behavior.
type CandidateRole string

const (
	Forbidden          CandidateRole = "forbidden"
	ConcealNotFound    CandidateRole = "conceal-not-found"
	MetadataDisclosure CandidateRole = "metadata-disclosure"
	Alternating        CandidateRole = "alternating"
)

const (
	Entrypoint                = "fixture/server.mjs"
	PortableEntrypoint        = "fixture/server_child_bind.mjs"
	SeedFilename              = "invoice-seed.json"
	InvocationReceiptFilename = "http-invocation.json"
	ContaminationFilename     = "authz-cache.json"
	FixtureVersion            = "countershape-http-invoice-fixture/v1"
)

// Roles returns the closed v1 study roster in display order. Semantic map
// identity must not depend on this order.
func Roles() []CandidateRole {
	return []CandidateRole{Forbidden, ConcealNotFound, MetadataDisclosure, Alternating}
}

func (r CandidateRole) Valid() bool {
	switch r {
	case Forbidden, ConcealNotFound, MetadataDisclosure, Alternating:
		return true
	default:
		return false
	}
}

// CandidateFiles returns one complete selected Git tree. The role file is
// candidate identity; server.mjs is byte-identical across the four trees.
func CandidateFiles(role CandidateRole) ([]gitrepo.File, error) {
	return candidateFiles(role, Entrypoint, fixtureProgram)
}

// PortableCandidateFiles returns the distinct P07B child-bind fixture tree.
// CandidateFiles and server.mjs remain byte-identical to the sealed U4 input.
func PortableCandidateFiles(role CandidateRole) ([]gitrepo.File, error) {
	return candidateFiles(role, PortableEntrypoint, portableFixtureProgram)
}

func candidateFiles(role CandidateRole, entrypoint string, program []byte) ([]gitrepo.File, error) {
	if !role.Valid() {
		return nil, fmt.Errorf("unsupported HTTP fixture role %q", role)
	}
	roleJSON := []byte(fmt.Sprintf("{\"role\":%q}", string(role)))
	return []gitrepo.File{
		{Path: "candidate-role.json", Mode: "100644", Content: roleJSON},
		{Path: entrypoint, Mode: "100755", Content: append([]byte(nil), program...)},
	}, nil
}

// Program returns a defensive copy for direct fixture tests.
func Program() []byte {
	return append([]byte(nil), fixtureProgram...)
}

// PortableProgram returns the distinct child-bind fixture bytes.
func PortableProgram() []byte {
	return append([]byte(nil), portableFixtureProgram...)
}

// SeedJSON is the exact declarative seed installed into every fresh fixture
// root. No setup command or package installation is needed.
func SeedJSON() []byte {
	return []byte(`{"invoice_id":"inv-204","metadata":{"amount_cents":4200,"currency":"USD","owner_tenant":"tenant-b"},"owner_tenant":"tenant-b"}`)
}

// TenantlessSeedJSON is the one closed shape-trap neighbor reserved for the
// reducer falsification study. It deliberately removes both tenant-bearing
// seed fields while retaining an otherwise valid invoice. The fixture treats
// this as an application condition after readiness, not malformed setup.
func TenantlessSeedJSON() []byte {
	return []byte(`{"invoice_id":"inv-204","metadata":{"amount_cents":4200,"currency":"USD"}}`)
}

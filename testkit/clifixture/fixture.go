// Package clifixture provides dependency-free Node candidate trees for the
// U3 CLI precedence study. It is test infrastructure, not product execution
// authority.
package clifixture

import (
	_ "embed"
	"fmt"

	"github.com/nelsonwerd/countershape/testkit/gitrepo"
)

//go:embed fixture.mjs
var fixtureProgram []byte

// CandidateRole selects one deliberately incompatible precedence rule.
type CandidateRole string

const (
	ConfigFirst      CandidateRole = "config-first"
	EnvironmentFirst CandidateRole = "env-first"
	ArgvFirst        CandidateRole = "argv-first"
)

// Roles returns the closed v1 study roster in display order. Semantic map
// identity must not depend on this order.
func Roles() []CandidateRole {
	return []CandidateRole{ConfigFirst, EnvironmentFirst, ArgvFirst}
}

func (r CandidateRole) Valid() bool {
	return r == ConfigFirst || r == EnvironmentFirst || r == ArgvFirst
}

// CandidateFiles returns a complete selected Git tree. The role file is part
// of candidate identity, while fixture.mjs is byte-identical across roles.
func CandidateFiles(role CandidateRole) ([]gitrepo.File, error) {
	if !role.Valid() {
		return nil, fmt.Errorf("unsupported CLI fixture role %q", role)
	}
	roleJSON := []byte(fmt.Sprintf("{\"precedence\":%q}", string(role)))
	return []gitrepo.File{
		{Path: "candidate-role.json", Mode: "100644", Content: roleJSON},
		{Path: "fixture.mjs", Mode: "100755", Content: append([]byte(nil), fixtureProgram...)},
	}, nil
}

// Program returns a defensive copy for tests that exercise the fixture
// without constructing a Git repository.
func Program() []byte {
	return append([]byte(nil), fixtureProgram...)
}

// ConfigJSON returns the exact closed fixture shape consumed by fixture.mjs.
func ConfigJSON(mode string) []byte {
	return []byte(fmt.Sprintf("{\"mode\":%q}", mode))
}

const (
	Entrypoint                = "fixture.mjs"
	InvocationReceiptFilename = "cli-invocation.json"
)

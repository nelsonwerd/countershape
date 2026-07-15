package httpfixture

import (
	"bytes"
	"testing"
)

func TestCandidateFilesAreClosedDistinctAndShareOneProgram(t *testing.T) {
	t.Parallel()
	seenRoles := map[string]struct{}{}
	for _, role := range Roles() {
		files, err := CandidateFiles(role)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 2 || files[0].Path != "candidate-role.json" || files[1].Path != Entrypoint {
			t.Fatalf("unexpected files for %s: %#v", role, files)
		}
		if files[0].Mode != "100644" || files[1].Mode != "100755" ||
			!bytes.Equal(files[1].Content, Program()) {
			t.Fatalf("candidate %s did not retain the closed program profile", role)
		}
		roleIdentity := string(files[0].Content)
		if _, duplicate := seenRoles[roleIdentity]; duplicate {
			t.Fatalf("duplicate role identity %q", roleIdentity)
		}
		seenRoles[roleIdentity] = struct{}{}
	}
	if _, err := CandidateFiles("unknown"); err == nil {
		t.Fatal("unknown HTTP fixture role was accepted")
	}
}

func TestSeedIsDefensiveAndCanonicalByConstruction(t *testing.T) {
	t.Parallel()
	for name, seed := range map[string]func() []byte{
		"reference":             SeedJSON,
		"tenantless-shape-trap": TenantlessSeedJSON,
	} {
		first := seed()
		second := seed()
		if !bytes.Equal(first, second) || len(first) == 0 {
			t.Fatalf("%s invoice seed changed between calls", name)
		}
		first[0] = '['
		if bytes.Equal(first, seed()) {
			t.Fatalf("%s invoice seed returned shared mutable bytes", name)
		}
	}
	if bytes.Equal(SeedJSON(), TenantlessSeedJSON()) {
		t.Fatal("tenantless shape-trap seed collapsed to the reference seed")
	}
}

package httpfixture

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

const (
	legacyProgramBytes  = 18343
	legacyProgramSHA256 = "d476c70a9c8e1605fc65335044afddaa9f56f007d1fcbc70ea87c0b747c72caa"
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

func TestPortableCandidateFilesAreDistinctWithoutRewritingLegacyFixture(t *testing.T) {
	t.Parallel()
	legacy := Program()
	portable := PortableProgram()
	legacyDigest := sha256.Sum256(legacy)
	if len(legacy) != legacyProgramBytes || hex.EncodeToString(legacyDigest[:]) != legacyProgramSHA256 {
		t.Fatalf("sealed U4 server.mjs changed: bytes=%d sha256=%x", len(legacy), legacyDigest)
	}
	if bytes.Equal(legacy, portable) || len(portable) == 0 || Entrypoint == PortableEntrypoint {
		t.Fatal("portable fixture collapsed onto the sealed legacy fixture")
	}
	for _, role := range Roles() {
		legacyFiles, err := CandidateFiles(role)
		if err != nil {
			t.Fatal(err)
		}
		portableFiles, err := PortableCandidateFiles(role)
		if err != nil {
			t.Fatal(err)
		}
		if len(portableFiles) != 2 || portableFiles[1].Path != PortableEntrypoint ||
			portableFiles[1].Mode != "100755" || !bytes.Equal(portableFiles[1].Content, portable) ||
			len(legacyFiles) != 2 || legacyFiles[1].Path != Entrypoint ||
			!bytes.Equal(legacyFiles[1].Content, legacy) ||
			!bytes.Equal(legacyFiles[0].Content, portableFiles[0].Content) {
			t.Fatalf("legacy/portable fixture roster mismatch for %s", role)
		}
	}
	legacy[0] ^= 0xff
	portable[0] ^= 0xff
	if bytes.Equal(legacy, Program()) || bytes.Equal(portable, PortableProgram()) {
		t.Fatal("fixture program accessors returned shared mutable bytes")
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

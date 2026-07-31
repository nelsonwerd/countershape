package scope

import (
	"crypto/sha256"

	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
)

const (
	referenceProgramBytes  = 19264
	referenceProgramSHA256 = "8cb5dfc1d5276e9f6157c127339e6e6c6404f525a3e8f9bec4f3eb1ad84a9932"
)

var referenceRoleDigests = map[int64]string{
	20: "400cf03fbcae3a4c74fc8c988f9dc31b266ddc1a88a778ccd87b97d22396b816",
	22: "8e4fbe13c07ffee401698aafe024e6eaf9a28b95ca4806ca1fdd5f9543745c8c",
	28: "05574ed6c775737393d5d11345eb0a0fa650e8c351bf62dd4583ffa54d3ad8fc",
	30: "362707a076cfcefefaf86d9fcd05b43ccfc89bb5974758f73ee1bc75d23fc0b3",
}

type Entry struct {
	Path   string
	Mode   string
	Size   int64
	SHA256 string
}

type Inventory struct {
	entries []Entry
	digest  string
}

func (value Inventory) Valid() bool {
	return len(value.entries) > 0 && len(value.digest) == sha256.Size*2
}

func (value Inventory) Digest() string {
	if !value.Valid() {
		return ""
	}
	return value.digest
}

func (value Inventory) EntryCount() int {
	if !value.Valid() {
		return 0
	}
	return len(value.entries)
}

func (value Inventory) Entries() []Entry {
	if !value.Valid() {
		return nil
	}
	return append([]Entry(nil), value.entries...)
}

func (value Inventory) Equal(other Inventory) bool {
	if !value.Valid() || !other.Valid() || value.digest != other.digest || len(value.entries) != len(other.entries) {
		return false
	}
	for index := range value.entries {
		if value.entries[index] != other.entries[index] {
			return false
		}
	}
	return true
}

func (value Inventory) ReferenceHTTPFixture() bool {
	if !value.Valid() || len(value.entries) != 3 {
		return false
	}
	role, directory, program := value.entries[0], value.entries[1], value.entries[2]
	roleDigest, roleKnown := referenceRoleDigests[role.Size]
	return roleKnown &&
		role.Path == "candidate-role.json" && role.Mode == "100644" && role.SHA256 == roleDigest &&
		directory.Path == "fixture" && directory.Mode == "040700" && directory.Size == 0 && directory.SHA256 == "" &&
		program.Path == "fixture/server_child_bind.mjs" && program.Mode == "100755" &&
		program.Size == referenceProgramBytes && program.SHA256 == referenceProgramSHA256
}

func (value Inventory) TargetViolation() (contractmodel.ScopeViolation, bool) {
	source := false
	dependency := false
	for _, entry := range value.entries {
		switch entry.Path {
		case "countershape-runtime.mjs":
			source = true
		case "node_modules/countershape/index.mjs":
			dependency = true
		}
	}
	switch {
	case source && dependency:
		return contractmodel.ViolationTargetSourceAndDependency, true
	case source:
		return contractmodel.ViolationTargetSourcePresent, true
	case dependency:
		return contractmodel.ViolationTargetDependencyPresent, true
	default:
		return "", false
	}
}

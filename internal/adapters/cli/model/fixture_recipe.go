package model

import (
	"bytes"

	"github.com/nelsonwerd/countershape/internal/domain"
)

const CLIFixtureOverlayAuthority = "U3_PRIVATE_FIXTURE_ROOT_EXCLUSIVE_REOPEN_REHASH_V1"

// CLIFixtureRecipe is the fixed U3 materialization capability for stimulus
// files. It is algorithm identity, not the stimulus file set itself.
type CLIFixtureRecipe struct {
	digest         domain.Digest
	canonicalBytes []byte
}

func NewCLIFixtureRecipe() (CLIFixtureRecipe, error) {
	identity := struct {
		SchemaVersion string   `json:"schema_version"`
		Kind          string   `json:"kind"`
		Version       string   `json:"version"`
		Authority     string   `json:"authority"`
		RootPolicy    string   `json:"root_policy"`
		PathPolicy    string   `json:"path_policy"`
		Modes         []string `json:"regular_file_modes"`
		WritePolicy   string   `json:"write_policy"`
		VerifyPolicy  string   `json:"verify_policy"`
	}{
		domain.SchemaVersion, "CLIFixtureRecipe", "cli-fixture-recipe/v1", CLIFixtureOverlayAuthority,
		"NEW_PRIVATE_FIXTURE_ROOT", "CLEAN_RELATIVE_STRICT_ORDER_NO_ALIAS_PREFIX_COLLISION_OR_GIT_METADATA",
		[]string{string(FixtureMode0644)}, "EXCLUSIVE_NO_OVERWRITE_SYNCED",
		"REOPEN_REGULAR_MODE_SIZE_BYTES_AND_DOMAIN_DIGEST",
	}
	digest, canonicalBytes, err := digestTyped("CLIFixtureRecipe", identity)
	if err != nil {
		return CLIFixtureRecipe{}, err
	}
	return CLIFixtureRecipe{digest: digest, canonicalBytes: canonicalBytes}, nil
}

func (r CLIFixtureRecipe) Valid() bool {
	rebuilt, err := NewCLIFixtureRecipe()
	return err == nil && rebuilt.digest == r.digest && bytes.Equal(rebuilt.canonicalBytes, r.canonicalBytes)
}

func (r CLIFixtureRecipe) Digest() domain.Digest  { return r.digest }
func (r CLIFixtureRecipe) CanonicalBytes() []byte { return append([]byte(nil), r.canonicalBytes...) }

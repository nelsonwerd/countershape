package package_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseProfileIsNarrowAndLocal(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "packaging", "release-profile.json"))
	if err != nil {
		t.Fatal(err)
	}
	var value struct {
		SchemaVersion  string   `json:"schema_version"`
		Platform       string   `json:"platform"`
		Architecture   string   `json:"architecture"`
		RuntimeClaim   string   `json:"runtime_claim"`
		LinuxClaim     string   `json:"linux_claim"`
		Signing        string   `json:"signing"`
		Distribution   string   `json:"distribution"`
		EmbeddedAssets []string `json:"embedded_assets"`
	}
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	if value.SchemaVersion != "countershape/release-profile/v1" || value.Platform != "darwin" || value.Architecture != "arm64" || value.RuntimeClaim != "EXACT_NATIVE_DARWIN_HOST_ONLY" || value.LinuxClaim != "CROSS_COMPILE_ONLY_UNRECEIPTED" || value.Signing != "UNSIGNED_LOCAL_REFERENCE_PACKAGE" || value.Distribution != "LOCAL_ONLY_NOT_PUBLISHED" || len(value.EmbeddedAssets) != 3 {
		t.Fatal("release profile overclaimed or drifted")
	}
}

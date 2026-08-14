//go:build darwin

package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStudioCLIAdmitsOnlyTheClosedTestAndPresentationSurface(t *testing.T) {
	t.Setenv(testLaunchAuthority, "")
	if _, err := parseCLI([]string{"--launch-file", "/tmp/forbidden"}); err == nil {
		t.Fatal("ordinary CLI admitted the private test launch channel")
	}
	if _, err := parseCLI([]string{"--seed-state", "foreign"}); err == nil {
		t.Fatal("CLI admitted a presentation state outside the closed roster")
	}
	if _, err := parseCLI([]string{"--unknown"}); err == nil {
		t.Fatal("CLI admitted an unknown argument")
	}

	t.Setenv(testLaunchAuthority, "1")
	config, err := parseCLI([]string{"--no-open", "--seed-state", string(StateError), "--launch-file", "/tmp/private-launch"})
	if err != nil {
		t.Fatal(err)
	}
	if !config.noOpen || config.state != StateError || config.launchFile != "/tmp/private-launch" {
		t.Fatalf("config = %#v", config)
	}
	if _, err := parseCLI([]string{"--launch-file", "/tmp/first", "--launch-file", "/tmp/second"}); err == nil {
		t.Fatal("CLI admitted more than one launch authority destination")
	}
}

func TestStudioPrivateLaunchFileIsExclusiveOwnerOnlyAndNotARequestTarget(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(root, "launch.json")
	launch := Launch{Origin: "http://127.0.0.1:43127", URL: "http://127.0.0.1:43127/#access_token=private-fragment-token"}
	if err := writePrivateLaunchFile(name, launch); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(name)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("launch file mode = %s", info.Mode())
	}
	body, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Origin string `json:"origin"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Origin != launch.Origin || parsed.URL != launch.URL || strings.Contains(parsed.Origin, "access_token") {
		t.Fatalf("launch payload = %#v", parsed)
	}
	if err := writePrivateLaunchFile(name, launch); err == nil {
		t.Fatal("launch channel overwrote an existing file")
	}
	if err := writePrivateLaunchFile("relative.json", launch); err == nil {
		t.Fatal("launch channel admitted a relative path")
	}
	if err := os.Chmod(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writePrivateLaunchFile(filepath.Join(root, "second.json"), launch); err == nil {
		t.Fatal("launch channel admitted a nonprivate parent")
	}
}

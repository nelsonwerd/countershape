package webassets

import (
	"io/fs"
	"slices"
	"strings"
	"testing"
)

func TestEmbeddedStudioBundleIsClosedAndEvidenceFree(t *testing.T) {
	want := []string{"assets/studio.css", "assets/studio.js", "index.html"}
	var got []string
	var combined strings.Builder
	if err := fs.WalkDir(Assets(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." || entry.IsDir() {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			t.Fatalf("embedded asset %q is a symlink", name)
		}
		body, err := fs.ReadFile(Assets(), name)
		if err != nil || len(body) == 0 {
			t.Fatalf("embedded asset %q could not be reopened", name)
		}
		got = append(got, name)
		combined.Write(body)
		return nil
	}); err != nil {
		t.Fatalf("walk embedded bundle: %v", err)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("embedded bundle roster = %v, want %v", got, want)
	}

	static := combined.String()
	for _, forbidden := range []string{"argv-first", "config-first", "env-first", "outer deterministic CLI precedence fixture", "candidate:"} {
		if strings.Contains(static, forbidden) {
			t.Fatalf("static bootstrap contains package evidence %q", forbidden)
		}
	}
}

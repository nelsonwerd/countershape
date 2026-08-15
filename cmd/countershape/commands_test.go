package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageCommandsExposeHelpVersionAndExport(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"version"}} {
		var out, err bytes.Buffer
		if code := run(args, &bytes.Buffer{}, &out, &err); code != 0 || out.Len() == 0 || err.Len() != 0 {
			t.Fatalf("%v failed", args)
		}
	}
	root := t.TempDir()
	path := filepath.Join(root, "report.html")
	var out, err bytes.Buffer
	if code := run([]string{"export", "--output", path, "--acknowledge-confidentiality-not-established"}, &bytes.Buffer{}, &out, &err); code != 0 || !strings.Contains(out.String(), WarningForTest()) {
		t.Fatalf("export failed: %s", err.String())
	}
}

func WarningForTest() string { return "CONFIDENTIALITY NOT ESTABLISHED" }

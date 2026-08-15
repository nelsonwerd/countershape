package export_test

import (
	"bytes"
	"testing"

	"github.com/nelsonwerd/countershape/internal/report"
	"github.com/nelsonwerd/countershape/internal/server"
)

func TestPackageIssuedExportHasTheClosedDefaultAndRawProfiles(t *testing.T) {
	snapshot, err := server.BuildSeedExportSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	minimized, err := report.Render(snapshot, false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := report.Render(snapshot, true)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(minimized, raw) || bytes.Contains(minimized, []byte("candidate:")) || !bytes.Contains(raw, []byte("candidate:")) {
		t.Fatal("default/raw export boundary collapsed")
	}
}

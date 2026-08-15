package assets

import (
	"io/fs"
	"testing"
)

func TestProductionAssetsAreExactAndEmbedded(t *testing.T) {
	value, err := Production()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range exactRoster {
		body, err := fs.ReadFile(value, name)
		if err != nil || len(body) == 0 {
			t.Fatalf("asset %s unavailable", name)
		}
	}
}

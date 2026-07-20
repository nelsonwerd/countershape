//go:build darwin && arm64 && cgo

package hostepoch

import (
	"context"
	"testing"
)

func TestC3HostEpochDarwinLiveMeasurement(t *testing.T) {
	epoch, err := Measure(context.Background())
	if err != nil || !epoch.Valid() {
		t.Fatalf("live Darwin measurement failed: %v", err)
	}
	fresh, err := epoch.Revalidate(context.Background())
	if err != nil || !fresh.Valid() || fresh.Digest() != epoch.Digest() {
		t.Fatalf("live Darwin revalidation failed: %v", err)
	}
}

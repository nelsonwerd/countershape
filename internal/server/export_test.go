package server

import (
	"bytes"
	"testing"
)

func TestExportSnapshotIsDeterministicDefensiveAndExact(t *testing.T) {
	first, err := BuildSeedExportSnapshot()
	if err != nil || !first.Valid() {
		t.Fatal(err)
	}
	second, err := BuildSeedExportSnapshot()
	if err != nil || !bytes.Equal(first.BenchBytes(), second.BenchBytes()) || !bytes.Equal(first.DecisionBytes(), second.DecisionBytes()) {
		t.Fatal("seed export changed")
	}
	body := first.BenchBytes()
	body[0] ^= 0xff
	if !first.Valid() {
		t.Fatal("accessor leaked mutable bytes")
	}
	forged := first
	forged.bench = append([]byte(nil), forged.bench...)
	forged.bench[0] ^= 0xff
	if forged.Valid() {
		t.Fatal("forged snapshot validated")
	}
}

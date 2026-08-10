package httpstudy

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/nelsonwerd/countershape/internal/domain"
)

func TestEvidencePublicationIsExactRepeatableAndFreshSeparated(t *testing.T) {
	firstRoot := privateTestRoot(t, "evidence-one")
	secondRoot := privateTestRoot(t, "evidence-two")
	first, err := publishEvidenceInput(testPublicationInput(firstRoot, 1))
	if err != nil {
		t.Fatal(err)
	}
	second, err := publishEvidenceInput(testPublicationInput(secondRoot, 2))
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Manifest()) != 111 || len(second.Manifest()) != 111 || first.ManifestSHA256() == second.ManifestSHA256() {
		t.Fatalf("publication roster/digest = %d/%d %s/%s", len(first.Manifest()), len(second.Manifest()), first.ManifestSHA256(), second.ManifestSHA256())
	}
	for field, digest := range first.DeterministicSHA256() {
		if second.DeterministicSHA256()[field] != digest {
			t.Fatalf("deterministic %s changed across ordinals", field)
		}
	}
	for field, digest := range first.FreshSHA256() {
		if second.FreshSHA256()[field] == digest {
			t.Fatalf("fresh %s repeated across ordinals", field)
		}
	}
	if _, err := publishEvidenceInput(testPublicationInput(firstRoot, 1)); err == nil {
		t.Fatal("nonempty evidence root was reused")
	}
	for _, entry := range first.Manifest() {
		info, err := os.Lstat(filepath.Join(firstRoot, filepath.FromSlash(entry.Path)))
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Mode()&os.ModeSymlink != 0 || info.Size() != int64(entry.Bytes) {
			t.Fatalf("manifest entry %s differs: %+v %v", entry.Path, info, err)
		}
	}
}

func TestEvidencePublicationRejectsPayloadAliasesAndInvalidJSONValues(t *testing.T) {
	alias := testPublicationInput(privateTestRoot(t, "alias"), 1)
	alias.Deterministic.WorldPlan = alias.Deterministic.SourceSpec
	if _, err := publishEvidenceInput(alias); err == nil || !strings.Contains(err.Error(), "PAYLOAD_ALIAS") {
		t.Fatalf("payload alias error = %v", err)
	}

	invalid := testPublicationInput(privateTestRoot(t, "invalid"), 1)
	invalid.Fresh.Captures = EvidencePayload{"not_finite": math.NaN()}
	if _, err := publishEvidenceInput(invalid); err == nil {
		t.Fatal("non-finite JSON payload was admitted")
	}

	mismatchedRun := testPublicationInput(privateTestRoot(t, "mismatched-run"), 1)
	mismatchedRun.Fresh.Confirmation["physical_run_authority"] = "sha256:" + strings.Repeat("f", 64)
	if _, err := publishEvidenceInput(mismatchedRun); err == nil || !strings.Contains(err.Error(), "PHYSICAL_RUN_AUTHORITY") {
		t.Fatalf("physical-run mismatch error = %v", err)
	}
}

func TestEvidencePublicationConcurrentDistinctRoots(t *testing.T) {
	inputs := []evidencePublicationInput{
		testPublicationInput(privateTestRoot(t, "concurrent-one"), 1),
		testPublicationInput(privateTestRoot(t, "concurrent-two"), 2),
		testPublicationInput(privateTestRoot(t, "concurrent-three"), 3),
	}
	publications := make([]EvidencePublication, len(inputs))
	errorsByIndex := make([]error, len(inputs))
	start := make(chan struct{})
	var wait sync.WaitGroup
	for index := range inputs {
		index := index
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			publications[index], errorsByIndex[index] = publishEvidenceInput(inputs[index])
		}()
	}
	close(start)
	wait.Wait()

	seenManifests := make(map[string]struct{}, len(publications))
	for index, publication := range publications {
		if errorsByIndex[index] != nil || len(publication.Manifest()) != 111 {
			t.Fatalf("concurrent publication %d: manifest=%d err=%v", index, len(publication.Manifest()), errorsByIndex[index])
		}
		if _, duplicate := seenManifests[publication.ManifestSHA256()]; duplicate {
			t.Fatalf("concurrent publication %d repeated manifest %s", index, publication.ManifestSHA256())
		}
		seenManifests[publication.ManifestSHA256()] = struct{}{}
	}
}

func TestRunConfigurationRefusesAmbientOrAliasedAuthority(t *testing.T) {
	if _, err := Run(nil, Config{}); err == nil || !strings.Contains(err.Error(), "HTTP_STUDY_CONFIG_REFUSED") {
		t.Fatalf("zero config error = %v", err)
	}
}

func testPublicationInput(root string, ordinal int) evidencePublicationInput {
	physicalRunAuthority, err := domain.ParseDigest("sha256:" + hashStrings([]string{"test-physical-run", fmt.Sprintf("%d", ordinal)}))
	if err != nil {
		panic(err)
	}
	deterministic := func(name string) EvidencePayload {
		return EvidencePayload{"authority": "deterministic-" + name, "digest": strings.Repeat(string('a'+rune(len(name)%20)), 64)}
	}
	fresh := func(name string) EvidencePayload {
		return EvidencePayload{
			"authority": "fresh-" + name, "physical": fmt.Sprintf("%s-run-%d-authority", name, ordinal),
			"physical_run_authority": physicalRunAuthority.String(),
		}
	}
	return evidencePublicationInput{
		Root: root, Domain: evidenceDomainHTTP, Ordinal: ordinal,
		PhysicalRunAuthority: physicalRunAuthority, PhaseTrials: testPhaseTrials(),
		Deterministic: DeterministicEvidencePayloads{
			SourceSpec: deterministic("source"), WorldPlan: deterministic("plan"), Ruling: deterministic("ruling"),
			DecisionRecord: deterministic("decision"), ContractBundle: deterministic("contract"),
		},
		Fresh: FreshEvidencePayloads{
			WorldInstance: fresh("world"), Attempts: fresh("attempts"), Measurements: fresh("measurements"), Captures: fresh("captures"),
			Confirmation: fresh("confirmation"), TargetProjection: fresh("target"),
			ProcessObservation: fresh("process"), StandaloneResult: fresh("result"),
		},
	}
}

func testPhaseTrials() []phaseTrialReceipt {
	result := make([]phaseTrialReceipt, 0, 98)
	for _, phase := range []struct {
		name  string
		count int
	}{{"search", 80}, {"confirm", 8}, {"contract", 10}} {
		for trial := 1; trial <= phase.count; trial++ {
			raw := hashStrings([]string{"test-phase-authority", phase.name, fmt.Sprintf("%d", trial)})
			authority, err := domain.ParseDigest("sha256:" + raw)
			if err != nil {
				panic(err)
			}
			result = append(result, phaseTrialReceipt{phase: phase.name, trial: trial, authority: authority})
		}
	}
	return result
}

func privateTestRoot(t *testing.T, name string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || !filepath.IsAbs(resolved) || filepath.Clean(resolved) != resolved {
		t.Fatalf("private root %q did not resolve canonically: %q %v", root, resolved, err)
	}
	return resolved
}

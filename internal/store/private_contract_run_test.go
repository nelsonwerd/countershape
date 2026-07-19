package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func c2TargetAndClaim(t *testing.T) (*ObjectStore, string, targetStorageRecord, startClaimRecord) {
	t.Helper()
	store, root, target, bootRaw := c2TargetOnlyFixture(t)
	boot, _ := domainDigest(bootRaw)
	_, claim, _, err := acquireInterlockAndStartClaim(context.Background(), store, target, boot, nil)
	if err != nil {
		t.Fatal(err)
	}
	return store, root, target, claim
}

func c2Blob(index int, reference privateEvidenceReference) privateBlobInput {
	return privateBlobInput{
		body:       []byte(fmt.Sprintf("private-blob-%02d", index)),
		references: []privateEvidenceReference{reference},
	}
}

func c2PrivateInputs(count int) []privateBlobInput {
	inputs := make([]privateBlobInput, count)
	for index := range inputs {
		kindIndex := index % len(privateEvidenceKindOrder)
		reference := privateEvidenceReference{
			kind: privateEvidenceKindOrder[kindIndex], digest: c2Digest("89abcdefabcdefa"[kindIndex]),
		}
		inputs[index] = c2Blob(index, reference)
	}
	return inputs
}

func c2MutatePrivateManifest(t *testing.T, body []byte, mutate func(map[string]any)) []byte {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	mutate(value)
	wire, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	exact, err := canon.Canonicalize(wire)
	if err != nil {
		t.Fatal(err)
	}
	return exact
}

func TestC2PrivateManifestEnforcesCountSizeAndRosterBounds(t *testing.T) {
	store, root, target, claim := c2TargetAndClaim(t)
	if record, _, err := createPrivateManifest(context.Background(), store, target, claim, nil, nil); err != nil ||
		record.blobCount != 0 || record.aggregateBytes != 0 {
		t.Fatalf("zero-blob boundary failed: %#v %v", record, err)
	}
	reference := privateEvidenceReference{kind: privateEvidenceKindOrder[0], digest: c2Digest('8')}
	maximum := c2PrivateInputs(maxPrivateBlobs)
	if record, _, err := createPrivateManifest(context.Background(), store, target, claim, maximum, nil); err != nil ||
		record.blobCount != maxPrivateBlobs || len(record.entries) != maxPrivateBlobs {
		t.Fatalf("16-blob boundary failed: count=%d err=%v", record.blobCount, err)
	} else {
		if _, err := parsePrivateManifest(record.canonical); err != nil {
			t.Fatalf("successful 16-blob manifest did not parse: %v", err)
		}
		restarted, err := OpenObjectStore(root)
		if err != nil {
			t.Fatal(err)
		}
		bundle := c2Bundle(t)
		modelTarget := c2Target(t, bundle, c2Digest('3'), '4')
		attempt := issueC2AttemptFixture(restarted, modelTarget.AttemptArtifactDigest())
		restartedTarget, err := openTargetByAttempt(context.Background(), restarted, attempt)
		if err != nil {
			t.Fatal(err)
		}
		restartedClaim, err := openStartClaim(context.Background(), restarted, restartedTarget)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := openPrivateManifest(context.Background(), restarted, restartedTarget, restartedClaim, record.digest); err != nil {
			t.Fatalf("successful 16-blob manifest did not restart-reopen: %v", err)
		}
	}
	tooMany := append(append([]privateBlobInput(nil), maximum...), c2Blob(maxPrivateBlobs, reference))
	if _, _, err := createPrivateManifest(context.Background(), store, target, claim, tooMany, nil); err == nil {
		t.Fatal("17 private blobs were accepted")
	}
	duplicateWithinBlob := privateBlobInput{
		body: []byte("duplicate-ref"), references: []privateEvidenceReference{reference, reference},
	}
	if _, _, err := createPrivateManifest(context.Background(), store, target, claim, []privateBlobInput{duplicateWithinBlob}, nil); err == nil {
		t.Fatal("duplicate reference within one blob was accepted")
	}
	conflictingKind := c2PrivateInputs(2)
	conflictingKind[1].references[0] = privateEvidenceReference{
		kind: conflictingKind[0].references[0].kind, digest: c2Digest('f'),
	}
	manifestDirectory := filepath.Join(store.contractRuns, privateManifestDirectory)
	packDirectory := filepath.Join(store.contractRuns, privatePackDirectory)
	manifestBefore, _ := os.ReadDir(manifestDirectory)
	packBefore, _ := os.ReadDir(packDirectory)
	if _, _, err := createPrivateManifest(context.Background(), store, target, claim, conflictingKind, nil); err == nil {
		t.Fatal("one evidence kind mapped to two digests")
	}
	manifestAfter, _ := os.ReadDir(manifestDirectory)
	packAfter, _ := os.ReadDir(packDirectory)
	if len(manifestBefore) != len(manifestAfter) || len(packBefore) != len(packAfter) {
		t.Fatal("invalid logical roster published private artifacts")
	}
	invalidReference := privateEvidenceReference{kind: "PRIVATE_EVIDENCE_MANIFEST", digest: c2Digest('9')}
	if _, _, err := createPrivateManifest(context.Background(), store, target, claim, []privateBlobInput{c2Blob(0, invalidReference)}, nil); err == nil {
		t.Fatal("manifest self-reference entered the private blob roster")
	}

	largeStore, _, largeTarget, largeClaim := c2TargetAndClaim(t)
	maximumBytes := bytes.Repeat([]byte{'x'}, maxPrivateEvidenceBytes)
	largeBlob := privateBlobInput{body: maximumBytes, references: []privateEvidenceReference{reference}}
	if record, _, err := createPrivateManifest(context.Background(), largeStore, largeTarget, largeClaim, []privateBlobInput{largeBlob}, nil); err != nil ||
		record.aggregateBytes != maxPrivateEvidenceBytes {
		t.Fatalf("64 MiB boundary failed: bytes=%d err=%v", record.aggregateBytes, err)
	}
	tooLarge := append(append([]byte(nil), maximumBytes...), 'y')
	if _, _, err := createPrivateManifest(context.Background(), largeStore, largeTarget, largeClaim, []privateBlobInput{{
		body: tooLarge, references: []privateEvidenceReference{reference},
	}}, nil); err == nil {
		t.Fatal("64 MiB + 1 byte was accepted")
	}

	for count := 0; count <= maxPrivateBlobs; count++ {
		t.Run(fmt.Sprintf("create-parse-closure-%02d", count), func(t *testing.T) {
			candidateStore, candidateRoot, candidateTarget, candidateClaim := c2TargetAndClaim(t)
			record, _, err := createPrivateManifest(
				context.Background(), candidateStore, candidateTarget, candidateClaim, c2PrivateInputs(count), nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := parsePrivateManifest(record.canonical); err != nil {
				t.Fatalf("successful create did not parse: %v", err)
			}
			restarted, err := OpenObjectStore(candidateRoot)
			if err != nil {
				t.Fatal(err)
			}
			bundle := c2Bundle(t)
			modelTarget := c2Target(t, bundle, c2Digest('3'), '4')
			attempt := issueC2AttemptFixture(restarted, modelTarget.AttemptArtifactDigest())
			restartedTarget, err := openTargetByAttempt(context.Background(), restarted, attempt)
			if err != nil {
				t.Fatal(err)
			}
			restartedClaim, err := openStartClaim(context.Background(), restarted, restartedTarget)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := openPrivateManifest(
				context.Background(), restarted, restartedTarget, restartedClaim, record.digest,
			); err != nil {
				t.Fatalf("successful create did not restart-reopen: %v", err)
			}
		})
	}
}

func TestC2MissingPrivateEvidenceBeforeFinalizationRefuses(t *testing.T) {
	fixture := c2CompleteFixture(t)
	if err := os.Remove(fixture.manifest.packPath); err != nil {
		t.Fatal(err)
	}
	if err := syncDirectory(filepath.Dir(fixture.manifest.packPath)); err != nil {
		t.Fatal(err)
	}
	runObject := c2SemanticObject(t, contractRunKind, fixture.run.Digest(), fixture.run.CanonicalBytes())
	if _, _, err := persistFinalizedRunRecord(context.Background(), fixture.store, runStorageInput{
		target: fixture.targetRec, claim: fixture.claim, object: runObject, manifest: fixture.manifest,
	}, nil); err == nil {
		t.Fatal("missing private pack passed finalized-run persistence")
	}
}

func TestC2PostFinalizationPurgeChangesAvailabilityOnly(t *testing.T) {
	for _, phase := range []contractFaultPhase{"", faultAfterPurgeIntentSync, faultAfterPackUnlink, faultAfterPurgeSync} {
		name := "complete"
		if phase != "" {
			name = string(phase)
		}
		t.Run(name, func(t *testing.T) {
			fixture := c2CompleteFixture(t)
			runBefore := fixture.runRec.record.object.CanonicalBytes()
			executionBefore := fixture.execRec.record.object.CanonicalBytes()
			fault := contractFault(nil)
			if phase != "" {
				fault = func(actual contractFaultPhase) error {
					if actual == phase {
						return errors.New("injected purge interruption")
					}
					return nil
				}
			}
			availability, err := purgePrivateEvidence(context.Background(), fixture.store, fixture.runRec, fixture.manifest, fault)
			if availability.state != privateStatePurged {
				t.Fatalf("purge state=%q err=%v", availability.state, err)
			}
			reopened, reopenErr := privateAvailability(context.Background(), fixture.store, fixture.runRec, fixture.manifest)
			if reopenErr != nil || reopened.state != privateStatePurged {
				t.Fatalf("durable purge availability=%q err=%v", reopened.state, reopenErr)
			}
			if !bytes.Equal(runBefore, fixture.runRec.record.object.CanonicalBytes()) ||
				!bytes.Equal(executionBefore, fixture.execRec.record.object.CanonicalBytes()) {
				t.Fatal("purge mutated finalized-run or classification bytes")
			}
		})
	}

	for _, phase := range []contractFaultPhase{
		faultBeforeTemporary,
		faultAfterTempSync,
		faultBeforeLink,
		faultAfterLink,
		faultAfterParentSync,
		faultBeforeReopen,
	} {
		t.Run("create-once-"+string(phase), func(t *testing.T) {
			fixture := c2CompleteFixture(t)
			fault := func(actual contractFaultPhase) error {
				if actual == phase {
					return errors.New("injected create-once purge interruption")
				}
				return nil
			}
			if _, err := purgePrivateEvidence(
				context.Background(), fixture.store, fixture.runRec, fixture.manifest, fault,
			); err == nil {
				t.Fatal("create-once purge boundary fault reported success")
			} else if phase == faultAfterLink || phase == faultAfterParentSync || phase == faultBeforeReopen {
				if mutationEffect(err) != contractAmbiguous {
					t.Fatalf("durability-uncertain purge effect=%q", mutationEffect(err))
				}
			} else if mutationEffect(err) != contractKnownNoEffect {
				t.Fatalf("pre-publication purge effect=%q", mutationEffect(err))
			}
			observed, err := privateAvailability(
				context.Background(), fixture.store, fixture.runRec, fixture.manifest,
			)
			visible := phase == faultAfterLink || phase == faultAfterParentSync || phase == faultBeforeReopen
			if err != nil || (visible && observed.state != privateStatePurged) || (!visible && observed.state != privateStateRetained) {
				t.Fatalf("faulted purge classification=%q visible=%t err=%v", observed.state, visible, err)
			}
			if _, err := os.Lstat(fixture.manifest.packPath); err != nil {
				t.Fatalf("faulted first purge removed pack before durable convergence: %v", err)
			}
			availability, err := purgePrivateEvidence(
				context.Background(), fixture.store, fixture.runRec, fixture.manifest, nil,
			)
			if err != nil || availability.state != privateStatePurged {
				t.Fatalf("purge retry did not converge: state=%q err=%v", availability.state, err)
			}
			if _, err := os.Lstat(fixture.manifest.packPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("converged purge retained pack: %v", err)
			}
			reopened, err := privateAvailability(
				context.Background(), fixture.store, fixture.runRec, fixture.manifest,
			)
			if err != nil || reopened.state != privateStatePurged {
				t.Fatalf("restarted purge classification=%q err=%v", reopened.state, err)
			}
		})
	}

	t.Run("cross-manifest-refuses-without-tombstone", func(t *testing.T) {
		fixture := c2CompleteFixture(t)
		alternateBlobs := make([]privateBlobInput, len(fixture.manifest.entries))
		for index, entry := range fixture.manifest.entries {
			alternateBlobs[index] = privateBlobInput{
				body:       bytes.Repeat([]byte{byte(index + 1)}, int(entry.count)),
				references: append([]privateEvidenceReference(nil), entry.references...),
			}
		}
		alternate, _, err := createPrivateManifest(
			context.Background(), fixture.store, fixture.targetRec, fixture.claim, alternateBlobs, nil,
		)
		if err != nil || alternate.digest == fixture.manifest.digest {
			t.Fatalf("alternate manifest construction failed: %v", err)
		}
		packBefore, err := os.Lstat(alternate.packPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := privateAvailability(context.Background(), fixture.store, fixture.runRec, alternate); err == nil {
			t.Fatal("run availability accepted a different manifest with the same parents and logical roster")
		}
		if _, err := purgePrivateEvidence(context.Background(), fixture.store, fixture.runRec, alternate, nil); err == nil {
			t.Fatal("run purged a different manifest with the same parents and logical roster")
		}
		if _, err := os.Lstat(alternate.purgePath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("wrong-manifest purge wrote a tombstone: %v", err)
		}
		packAfter, err := os.Lstat(alternate.packPath)
		if err != nil || !os.SameFile(packBefore, packAfter) {
			t.Fatal("wrong-manifest purge changed the alternate pack")
		}
		availability, err := privateAvailability(
			context.Background(), fixture.store, fixture.runRec, fixture.manifest,
		)
		if err != nil || availability.state != privateStateRetained {
			t.Fatalf("wrong-manifest refusal changed the exact run manifest: %#v %v", availability, err)
		}
	})
}

func TestC2UnexpectedPrivateLossReportsMissingWithoutCanonicalMutation(t *testing.T) {
	fixture := c2CompleteFixture(t)
	runBefore := fixture.runRec.record.object.CanonicalBytes()
	executionBefore := fixture.execRec.record.object.CanonicalBytes()
	if err := os.Remove(fixture.manifest.packPath); err != nil {
		t.Fatal(err)
	}
	if err := syncDirectory(filepath.Dir(fixture.manifest.packPath)); err != nil {
		t.Fatal(err)
	}
	availability, err := privateAvailability(context.Background(), fixture.store, fixture.runRec, fixture.manifest)
	if err != nil || availability.state != privateStateMissingUnexpected {
		t.Fatalf("unexpected loss state=%q err=%v", availability.state, err)
	}
	if !bytes.Equal(runBefore, fixture.runRec.record.object.CanonicalBytes()) ||
		!bytes.Equal(executionBefore, fixture.execRec.record.object.CanonicalBytes()) {
		t.Fatal("unexpected private loss mutated canonical history")
	}
}

func TestC2PrivateEvidenceFaultMatrixReopensAcrossRestart(t *testing.T) {
	for _, test := range []struct {
		phase         contractFaultPhase
		effect        contractEffect
		packCount     int
		manifestCount int
	}{
		{faultBeforePackLink, contractKnownNoEffect, 0, 0},
		{faultAfterPackSync, contractAmbiguous, 1, 0},
		{faultBeforeManifestLink, contractAmbiguous, 1, 0},
		{faultAfterLink, contractAmbiguous, 1, 1},
		{faultAfterManifestSync, contractAmbiguous, 1, 1},
	} {
		t.Run(string(test.phase), func(t *testing.T) {
			store, root, target, claim := c2TargetAndClaim(t)
			reference := privateEvidenceReference{kind: privateEvidenceKindOrder[0], digest: c2Digest('8')}
			blobs := []privateBlobInput{c2Blob(0, reference)}
			fault := func(actual contractFaultPhase) error {
				if actual == test.phase {
					return errors.New("injected private persistence fault")
				}
				return nil
			}
			record, effect, err := createPrivateManifest(context.Background(), store, target, claim, blobs, fault)
			if err == nil || record.validFor(store) || effect != test.effect || mutationEffect(err) != test.effect {
				t.Fatal("faulted manifest creation returned a record")
			}
			packs, packErr := os.ReadDir(filepath.Join(store.contractRuns, privatePackDirectory))
			manifests, manifestErr := os.ReadDir(filepath.Join(store.contractRuns, privateManifestDirectory))
			if packErr != nil || manifestErr != nil || len(packs) != test.packCount || len(manifests) != test.manifestCount {
				t.Fatalf("fault artifact counts packs=%d manifests=%d: %v %v", len(packs), len(manifests), packErr, manifestErr)
			}
			if test.manifestCount == 1 {
				manifestDigest, err := domain.ParseDigest("sha256:" + manifests[0].Name())
				if err != nil {
					t.Fatal(err)
				}
				restarted, err := OpenObjectStore(root)
				if err != nil {
					t.Fatal(err)
				}
				bundle := c2Bundle(t)
				modelTarget := c2Target(t, bundle, c2Digest('3'), '4')
				attempt := issueC2AttemptFixture(restarted, modelTarget.AttemptArtifactDigest())
				restartedTarget, err := openTargetByAttempt(context.Background(), restarted, attempt)
				if err != nil {
					t.Fatal(err)
				}
				restartedClaim, err := openStartClaim(context.Background(), restarted, restartedTarget)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := openPrivateManifest(
					context.Background(), restarted, restartedTarget, restartedClaim, manifestDigest,
				); err != nil {
					t.Fatalf("visible manifest did not converge on direct reopen: %v", err)
				}
			}
			record, retryEffect, err := createPrivateManifest(context.Background(), store, target, claim, blobs, nil)
			if err != nil {
				t.Fatalf("exact manifest retry failed: %v", err)
			}
			if retryEffect != contractExactConverged {
				t.Fatalf("exact manifest retry effect=%s", retryEffect)
			}
			restarted, err := OpenObjectStore(root)
			if err != nil {
				t.Fatal(err)
			}
			bundle := c2Bundle(t)
			modelTarget := c2Target(t, bundle, c2Digest('3'), '4')
			attempt := issueC2AttemptFixture(restarted, modelTarget.AttemptArtifactDigest())
			restartedTarget, err := openTargetByAttempt(context.Background(), restarted, attempt)
			if err != nil {
				t.Fatal(err)
			}
			restartedClaim, err := openStartClaim(context.Background(), restarted, restartedTarget)
			if err != nil {
				t.Fatal(err)
			}
			reopened, err := openPrivateManifest(context.Background(), restarted, restartedTarget, restartedClaim, record.digest)
			if err != nil || validatePrivateManifestForFinalization(context.Background(), restarted, reopened) != nil {
				t.Fatalf("restart manifest reopen failed: %v", err)
			}
		})
	}

	t.Run("manifest-parser-and-pack-bounds", func(t *testing.T) {
		store, _, target, claim := c2TargetAndClaim(t)
		reference := privateEvidenceReference{kind: privateEvidenceKindOrder[0], digest: c2Digest('8')}
		record, _, err := createPrivateManifest(
			context.Background(), store, target, claim, []privateBlobInput{c2Blob(0, reference)}, nil,
		)
		if err != nil {
			t.Fatal(err)
		}
		entry := func(value map[string]any) map[string]any {
			return value["entries"].([]any)[0].(map[string]any)
		}
		parserMutations := map[string]func(map[string]any){
			"wrong-range": func(value map[string]any) {
				entry(value)["pack_offset"] = int64(len(privatePackHeader) + 1)
			},
			"overflow-count": func(value map[string]any) {
				entry(value)["byte_count"] = int64(9007199254740991)
				value["aggregate_byte_count"] = int64(1)
				value["pack_byte_count"] = int64(len(privatePackHeader) + 1)
			},
			"negative-offset": func(value map[string]any) {
				entry(value)["pack_offset"] = int64(-1)
			},
			"wrong-role": func(value map[string]any) {
				entry(value)["evidence_refs"].([]any)[0].(map[string]any)["kind"] = "PRIVATE_EVIDENCE_MANIFEST"
			},
			"extra-logical-reference": func(value map[string]any) {
				refs := entry(value)["evidence_refs"].([]any)
				entry(value)["evidence_refs"] = append(refs, map[string]any{
					"kind": privateEvidenceKindOrder[0], "digest": c2Digest('9').String(),
				})
				value["logical_reference_count"] = int64(2)
			},
		}
		for name, mutate := range parserMutations {
			t.Run(name, func(t *testing.T) {
				defer func() {
					if recovered := recover(); recovered != nil {
						t.Fatalf("hostile manifest panicked: %v", recovered)
					}
				}()
				body := c2MutatePrivateManifest(t, record.canonical, mutate)
				if _, err := parsePrivateManifest(body); err == nil {
					t.Fatal("hostile manifest parsed")
				}
			})
		}
		for name, mutate := range map[string]func(map[string]any){
			"wrong-blob-hash": func(value map[string]any) {
				entry(value)["blob_digest"] = c2Digest('a').String()
			},
			"wrong-pack-hash": func(value map[string]any) {
				value["pack_digest"] = c2Digest('b').String()
			},
		} {
			t.Run(name, func(t *testing.T) {
				body := c2MutatePrivateManifest(t, record.canonical, mutate)
				parsed, err := parsePrivateManifest(body)
				if err != nil {
					t.Fatal(err)
				}
				parsed.packPath = record.packPath
				if err := validatePrivatePack(record.packPath, parsed); err == nil {
					t.Fatal("manifest hash mutation validated against the retained pack")
				}
			})
		}
	})

	t.Run("dynamic-private-directory-substitution-refuses", func(t *testing.T) {
		for _, selected := range []string{"manifest", "pack", "purge"} {
			t.Run(selected, func(t *testing.T) {
				fixture := c2CompleteFixture(t)
				var path string
				switch selected {
				case "manifest":
					path = filepath.Dir(fixture.manifest.manifestPath)
				case "pack":
					path = filepath.Dir(fixture.manifest.packPath)
				case "purge":
					path = filepath.Dir(fixture.manifest.purgePath)
				}
				c2SubstituteDirectory(t, path)
				if err := validatePrivateManifestForFinalization(
					context.Background(), fixture.store, fixture.manifest,
				); err == nil {
					t.Fatal("private manifest remained valid through directory substitution")
				}
				if _, err := privateAvailability(
					context.Background(), fixture.store, fixture.runRec, fixture.manifest,
				); err == nil {
					t.Fatal("private availability classified through directory substitution")
				}
				if _, err := purgePrivateEvidence(
					context.Background(), fixture.store, fixture.runRec, fixture.manifest, nil,
				); err == nil {
					t.Fatal("private purge wrote through directory substitution")
				}
			})
		}
	})

	t.Run("private-case-alias-preflight-is-known-no-effect", func(t *testing.T) {
		for _, selected := range []string{"manifest", "pack", "purge"} {
			t.Run(selected, func(t *testing.T) {
				store, _, target, claim := c2TargetAndClaim(t)
				blobs := c2PrivateInputs(1)
				record, _, err := createPrivateManifest(context.Background(), store, target, claim, blobs, nil)
				if err != nil {
					t.Fatal(err)
				}
				switch selected {
				case "manifest":
					if err := os.Remove(record.packPath); err != nil {
						t.Fatal(err)
					}
					c2CaseAliasPath(t, record.manifestPath)
				case "pack":
					if err := os.Remove(record.manifestPath); err != nil {
						t.Fatal(err)
					}
					c2CaseAliasPath(t, record.packPath)
				case "purge":
					if err := errors.Join(os.Remove(record.manifestPath), os.Remove(record.packPath)); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(record.purgePath, []byte("case-alias preflight"), 0o600); err != nil {
						t.Fatal(err)
					}
					c2CaseAliasPath(t, record.purgePath)
				}
				if _, effect, err := createPrivateManifest(
					context.Background(), store, target, claim, blobs, nil,
				); err == nil || effect != contractKnownNoEffect {
					t.Fatalf("%s alias preflight effect=%q err=%v", selected, effect, err)
				}
				if selected != "pack" {
					if _, err := os.Lstat(record.packPath); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("%s alias refusal created a pack: %v", selected, err)
					}
				}
				if selected != "manifest" {
					if _, err := os.Lstat(record.manifestPath); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("%s alias refusal created a manifest: %v", selected, err)
					}
				}
			})
		}
	})

	t.Run("dynamic-private-real-directory-replacement-refuses", func(t *testing.T) {
		for _, selected := range []string{"manifest", "pack", "purge"} {
			t.Run(selected, func(t *testing.T) {
				fixture := c2CompleteFixture(t)
				var path string
				switch selected {
				case "manifest":
					path = filepath.Dir(fixture.manifest.manifestPath)
				case "pack":
					path = filepath.Dir(fixture.manifest.packPath)
				case "purge":
					path = filepath.Dir(fixture.manifest.purgePath)
				}
				restore := c2ReplaceDirectoryWithExactClone(t, path)
				if err := validatePrivateManifestForFinalization(
					context.Background(), fixture.store, fixture.manifest,
				); err == nil {
					t.Fatal("private manifest remained valid through real directory replacement")
				}
				if _, err := privateAvailability(
					context.Background(), fixture.store, fixture.runRec, fixture.manifest,
				); err == nil {
					t.Fatal("private availability classified through real directory replacement")
				}
				if _, err := purgePrivateEvidence(
					context.Background(), fixture.store, fixture.runRec, fixture.manifest, nil,
				); err == nil {
					t.Fatal("private purge wrote through real directory replacement")
				}
				restore()
				if err := validatePrivateManifestForFinalization(
					context.Background(), fixture.store, fixture.manifest,
				); err != nil {
					t.Fatalf("private manifest did not recover after retained identity restoration: %v", err)
				}
			})
		}
	})

	for _, test := range []struct {
		name string
		path func(c2StorageFixture) string
	}{
		{"private-manifest-leaf-case-alias-refuses", func(fixture c2StorageFixture) string {
			return fixture.manifest.manifestPath
		}},
		{"private-pack-leaf-case-alias-refuses", func(fixture c2StorageFixture) string {
			return fixture.manifest.packPath
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := c2CompleteFixture(t)
			restore := c2CaseAliasPath(t, test.path(fixture))
			if err := validatePrivateManifestForFinalization(
				context.Background(), fixture.store, fixture.manifest,
			); err == nil {
				t.Fatal("private finalization accepted a case-only evidence leaf alias")
			}
			if _, err := privateAvailability(
				context.Background(), fixture.store, fixture.runRec, fixture.manifest,
			); err == nil {
				t.Fatal("private availability accepted a case-only evidence leaf alias")
			}
			if _, err := purgePrivateEvidence(
				context.Background(), fixture.store, fixture.runRec, fixture.manifest, nil,
			); err == nil {
				t.Fatal("private purge accepted a case-only evidence leaf alias")
			}
			restore()
			if err := validatePrivateManifestForFinalization(
				context.Background(), fixture.store, fixture.manifest,
			); err != nil {
				t.Fatalf("private evidence did not recover after canonical spelling restoration: %v", err)
			}
		})
	}
}

func TestC2PurgeCannotDeleteObjectsHeadsOrRetentionFact(t *testing.T) {
	fixture := c2CompleteFixture(t)
	objectPaths := make([]string, 0, 3)
	objectInfos := make([]os.FileInfo, 0, 3)
	for _, digest := range []domain.Digest{fixture.target.Digest(), fixture.run.Digest(), fixture.execution.Digest()} {
		path, _, err := fixture.store.existingObjectPath(digest)
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		objectPaths = append(objectPaths, path)
		objectInfos = append(objectInfos, info)
	}
	studiesBefore, err := os.Lstat(fixture.store.studies)
	if err != nil {
		t.Fatal(err)
	}
	manifestBefore := append([]byte(nil), fixture.manifest.canonical...)
	if availability, err := purgePrivateEvidence(context.Background(), fixture.store, fixture.runRec, fixture.manifest, nil); err != nil ||
		availability.state != privateStatePurged {
		t.Fatalf("purge failed: %#v %v", availability, err)
	}
	for index, path := range objectPaths {
		after, err := os.Lstat(path)
		if err != nil || !os.SameFile(objectInfos[index], after) {
			t.Fatalf("purge changed semantic object %s: %v", path, err)
		}
	}
	studiesAfter, err := os.Lstat(fixture.store.studies)
	if err != nil || !os.SameFile(studiesBefore, studiesAfter) {
		t.Fatal("purge changed the study-head namespace")
	}
	manifestAfter, err := os.ReadFile(fixture.manifest.manifestPath)
	if err != nil || !bytes.Equal(manifestBefore, manifestAfter) {
		t.Fatal("purge changed the retained-at-finalization manifest fact")
	}
}

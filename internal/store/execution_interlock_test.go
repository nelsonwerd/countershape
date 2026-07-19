package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/nelsonwerd/countershape/internal/domain"
)

const c2InterlockChildRoot = "COUNTERSHAPE_C2_INTERLOCK_CHILD_ROOT"

func c2TargetOnlyFixture(t *testing.T) (*ObjectStore, string, targetStorageRecord, string) {
	t.Helper()
	store, root := newObjectStoreForTest(t)
	record, boot := c2TargetInStore(t, store, c2Digest('3'), '4')
	return store, root, record, boot.String()
}

func c2TargetInStore(
	t *testing.T,
	store *ObjectStore,
	attemptDigest domain.Digest,
	nonce byte,
) (targetStorageRecord, domain.Digest) {
	t.Helper()
	bundle := c2Bundle(t)
	target := c2Target(t, bundle, attemptDigest, nonce)
	attempt := issueC2AttemptFixture(store, target.AttemptArtifactDigest())
	object := c2SemanticObject(t, contractTargetKind, target.Digest(), target.CanonicalBytes())
	record, _, err := persistTargetRecord(context.Background(), store, targetStorageInput{attempt: attempt, object: object}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return record, target.Input().BootSession.IdentityDigest
}

func issueC2TerminalClosureFixture(t *testing.T, fixture c2StorageFixture) terminalClosureRecord {
	t.Helper()
	if err := validatePrivateManifestForFinalization(context.Background(), fixture.store, fixture.manifest); err != nil {
		t.Fatal(err)
	}
	if err := validatePrivateManifestRoster(fixture.runRec.record.object, fixture.manifest); err != nil {
		t.Fatal(err)
	}
	return terminalClosureRecord{
		run: fixture.runRec, manifest: fixture.manifest, seal: &terminalClosureSeal{marker: 1},
	}
}

func issueC2ChangedBootResetFixture(boot domain.Digest) changedBootResetAuthorization {
	return changedBootResetAuthorization{currentBoot: boot, seal: &changedBootResetSeal{marker: 1}}
}

func TestC2InterlockClaimMultiProcessRaceHasOneStorageWinnerAndNoPermit(t *testing.T) {
	if root := os.Getenv(c2InterlockChildRoot); root != "" {
		store, err := OpenObjectStore(root)
		if err != nil {
			t.Fatal(err)
		}
		bundle := c2Bundle(t)
		target := c2Target(t, bundle, c2Digest('3'), '4')
		attempt := issueC2AttemptFixture(store, target.AttemptArtifactDigest())
		record, err := openTargetByAttempt(context.Background(), store, attempt)
		if err != nil {
			t.Fatal(err)
		}
		boot := target.Input().BootSession.IdentityDigest
		_, _, winner, acquireErr := acquireInterlockAndStartClaim(context.Background(), store, record, boot, nil)
		if acquireErr == nil {
			if !winner.validFor(store) {
				t.Fatal("child received an invalid winner")
			}
			fmt.Println("C2_CHILD_WINNER")
			return
		}
		fmt.Println("C2_CHILD_LOSER")
		return
	}

	_, root, _, _ := c2TargetOnlyFixture(t)
	type result struct {
		output string
		err    error
	}
	results := make([]result, 2)
	var wait sync.WaitGroup
	for index := range results {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			command := exec.Command(os.Args[0], "-test.run=^TestC2InterlockClaimMultiProcessRaceHasOneStorageWinnerAndNoPermit$")
			command.Env = append(os.Environ(), c2InterlockChildRoot+"="+root)
			output, err := command.CombinedOutput()
			results[index] = result{output: string(output), err: err}
		}(index)
	}
	wait.Wait()
	winners := 0
	losers := 0
	for _, result := range results {
		if result.err != nil {
			t.Fatalf("child process failed: %v\n%s", result.err, result.output)
		}
		winners += strings.Count(result.output, "C2_CHILD_WINNER")
		losers += strings.Count(result.output, "C2_CHILD_LOSER")
	}
	if winners != 1 || losers != 1 {
		t.Fatalf("race results winners=%d losers=%d: %#v", winners, losers, results)
	}
	if reflect.TypeOf(startClaimWinner{}).NumMethod() != 0 {
		t.Fatal("C2 winner unexpectedly exposes a method surface")
	}
}

func TestC2InterlockMustBeAcquiredBeforeTargetKeyedStartClaim(t *testing.T) {
	store, _, target, bootRaw := c2TargetOnlyFixture(t)
	claimDirectory := filepath.Join(store.contractOps, startClaimDirectory)
	if _, err := os.Lstat(claimDirectory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("claim directory existed before interlock acquisition: %v", err)
	}
	boot, _ := domainDigest(bootRaw)
	if _, _, winner, err := acquireInterlockAndStartClaim(
		context.Background(), store, target, c2Digest('a'), nil,
	); err == nil || winner.validFor(store) {
		t.Fatal("caller boot identity differing from the serialized target was admitted")
	}
	if _, present, err := readExecutionInterlockState(filepath.Join(store.contractOps, interlockFilename)); err != nil || present {
		t.Fatalf("boot mismatch mutated interlock state: present=%t err=%v", present, err)
	}
	lease, claim, winner, err := acquireInterlockAndStartClaim(context.Background(), store, target, boot, nil)
	if err != nil || !lease.validFor(store) || !claim.validFor(store) || !winner.validFor(store) {
		t.Fatalf("ordered acquisition failed: %v", err)
	}
	state, present, err := readExecutionInterlockState(filepath.Join(store.contractOps, interlockFilename))
	if err != nil || !present || state.state != interlockStateHeld || state.generation != claim.generation {
		t.Fatalf("claim was not fenced by exact held generation: %#v %v", state, err)
	}
	if err := consumeStartClaimWinner(context.Background(), store, &winner); err != nil {
		t.Fatalf("fresh winner did not consume exactly once: %v", err)
	}
	if err := consumeStartClaimWinner(context.Background(), store, &winner); err == nil {
		t.Fatal("winner consumed twice")
	}

	t.Run("known-no-claim-reconciliation-reopens-admission", func(t *testing.T) {
		candidateStore, root, candidate, bootRaw := c2TargetOnlyFixture(t)
		candidateBoot, _ := domainDigest(bootRaw)
		fault := func(phase contractFaultPhase) error {
			if phase == faultBeforeClaimLink {
				return errors.New("claim refused before durable link")
			}
			return nil
		}
		if _, _, lost, err := acquireInterlockAndStartClaim(
			context.Background(), candidateStore, candidate, candidateBoot, fault,
		); err == nil || lost.validFor(candidateStore) || mutationEffect(err) != contractKnownNoEffect {
			t.Fatal("known-no-claim path did not reconcile without winner authority")
		}
		state, present, err := readExecutionInterlockState(
			filepath.Join(candidateStore.contractOps, interlockFilename),
		)
		if err != nil || !present || state.state != interlockStateClear || !clearReceiptExact(candidateStore, state) {
			t.Fatalf("known-no-claim clear did not receipt exactly: %#v %v", state, err)
		}
		restarted, err := OpenObjectStore(root)
		if err != nil {
			t.Fatal(err)
		}
		restartedTarget, restartedBoot := c2TargetInStore(t, restarted, c2Digest('3'), '4')
		if _, _, admitted, err := acquireInterlockAndStartClaim(
			context.Background(), restarted, restartedTarget, restartedBoot, nil,
		); err != nil || !admitted.validFor(restarted) {
			t.Fatalf("receipt-backed known-no-claim restart did not readmit its target: %v", err)
		}
	})

	t.Run("case-alias-claim-preflight-does-not-create-interlock", func(t *testing.T) {
		subject, _, subjectTarget, subjectBootRaw := c2TargetOnlyFixture(t)
		subject.instance.mu.Lock()
		claimDirectory, err := subject.ensureContractDirectoryLocked(subject.contractOps, startClaimDirectory)
		subject.instance.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		claimPath := claimPathFor(claimDirectory, subjectTarget.record.object.Digest())
		if err := os.WriteFile(claimPath, []byte("case-alias preflight"), 0o600); err != nil {
			t.Fatal(err)
		}
		c2CaseAliasPath(t, claimPath)
		subjectBoot, _ := domainDigest(subjectBootRaw)
		if _, _, winner, err := acquireInterlockAndStartClaim(
			context.Background(), subject, subjectTarget, subjectBoot, nil,
		); err == nil || winner.validFor(subject) {
			t.Fatal("case-aliased claim path admitted a winner")
		}
		if _, err := os.Lstat(filepath.Join(subject.contractOps, interlockFilename)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("claim alias refusal mutated the interlock: %v", err)
		}
	})
}

func TestC2InterlockRestartReopensIntentWithoutAuthority(t *testing.T) {
	store, root, target, bootRaw := c2TargetOnlyFixture(t)
	boot, _ := domainDigest(bootRaw)
	_, claim, winner, err := acquireInterlockAndStartClaim(context.Background(), store, target, boot, nil)
	if err != nil || !winner.validFor(store) {
		t.Fatal(err)
	}
	restarted, err := OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	bundle := c2Bundle(t)
	modelTarget := c2Target(t, bundle, c2Digest('3'), '4')
	attempt := issueC2AttemptFixture(restarted, modelTarget.AttemptArtifactDigest())
	reopenedTarget, err := openTargetByAttempt(context.Background(), restarted, attempt)
	if err != nil {
		t.Fatal(err)
	}
	reopenedClaim, err := openStartClaim(context.Background(), restarted, reopenedTarget)
	if err != nil || reopenedClaim.digest != claim.digest {
		t.Fatalf("restart intent reopen failed: %v", err)
	}
	if (startClaimWinner{claim: reopenedClaim}).validFor(restarted) {
		t.Fatal("reopened intent recreated winner authority")
	}
	if _, _, replay, err := acquireInterlockAndStartClaim(context.Background(), restarted, reopenedTarget, boot, nil); err == nil || replay.validFor(restarted) {
		t.Fatal("restart reacquired a consumed target")
	}

	t.Run("canonical-boot-substitution-refuses", func(t *testing.T) {
		subject, subjectRoot, subjectTarget, subjectBootRaw := c2TargetOnlyFixture(t)
		subjectBoot, _ := domainDigest(subjectBootRaw)
		_, subjectClaim, subjectWinner, err := acquireInterlockAndStartClaim(
			context.Background(), subject, subjectTarget, subjectBoot, nil,
		)
		if err != nil || !subjectWinner.validFor(subject) {
			t.Fatalf("fixture acquisition failed: %v", err)
		}
		wrongBoot := c2Digest('f')
		if wrongBoot == subjectBoot {
			t.Fatal("hostile boot fixture did not differ")
		}
		subject.instance.mu.Lock()
		substituted, err := expectedStartClaim(subject, subjectTarget, wrongBoot, subjectClaim.generation)
		subject.instance.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		substituted.path = subjectClaim.path
		if err := os.WriteFile(subjectClaim.path, substituted.canonical, 0o600); err != nil {
			t.Fatal(err)
		}
		if !substituted.validFor(subject) {
			t.Fatal("canonical hostile claim fixture was not independently well formed")
		}
		if _, _, err := createPrivateManifest(
			context.Background(), subject, subjectTarget, substituted, nil, nil,
		); err == nil {
			t.Fatal("private persistence accepted a claim with a substituted boot identity")
		}

		subjectRestarted, err := OpenObjectStore(subjectRoot)
		if err != nil {
			t.Fatal(err)
		}
		subjectReopenedTarget, err := openTargetByAttempt(
			context.Background(), subjectRestarted,
			issueC2AttemptFixture(subjectRestarted, subjectTarget.record.relation.attemptDigest),
		)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := openStartClaim(context.Background(), subjectRestarted, subjectReopenedTarget); err == nil {
			t.Fatal("restart reopened a canonical claim with the wrong target boot identity")
		}
	})

	t.Run("after-link-ambiguity-reopens-inert-claim", func(t *testing.T) {
		candidateStore, candidateRoot, candidate, candidateBootRaw := c2TargetOnlyFixture(t)
		candidateBoot, _ := domainDigest(candidateBootRaw)
		fault := func(phase contractFaultPhase) error {
			if phase == faultAfterClaimLink {
				return errors.New("lost claim result before parent sync")
			}
			return nil
		}
		if _, _, candidateWinner, err := acquireInterlockAndStartClaim(
			context.Background(), candidateStore, candidate, candidateBoot, fault,
		); err == nil || candidateWinner.validFor(candidateStore) {
			t.Fatal("claim after-link ambiguity returned winner authority")
		}
		candidateRestarted, err := OpenObjectStore(candidateRoot)
		if err != nil {
			t.Fatal(err)
		}
		candidateTarget, err := openTargetByAttempt(
			context.Background(), candidateRestarted,
			issueC2AttemptFixture(candidateRestarted, candidate.record.relation.attemptDigest),
		)
		if err != nil {
			t.Fatal(err)
		}
		candidateClaim, err := openStartClaim(context.Background(), candidateRestarted, candidateTarget)
		if err != nil || !candidateClaim.validFor(candidateRestarted) {
			t.Fatalf("visible claim did not converge on reopen: %v", err)
		}
		if (startClaimWinner{claim: candidateClaim}).validFor(candidateRestarted) {
			t.Fatal("converged restart claim recreated winner authority")
		}
	})
}

func TestC2InterlockSameBootAmbiguityRemainsHeld(t *testing.T) {
	store, _, target, bootRaw := c2TargetOnlyFixture(t)
	boot, _ := domainDigest(bootRaw)
	fault := func(phase contractFaultPhase) error {
		if phase == faultAfterClaimLink {
			return errors.New("lost claim result")
		}
		return nil
	}
	if _, _, winner, err := acquireInterlockAndStartClaim(context.Background(), store, target, boot, fault); err == nil || winner.validFor(store) {
		t.Fatal("ambiguous claim returned a winner")
	}
	state, present, err := readExecutionInterlockState(filepath.Join(store.contractOps, interlockFilename))
	if err != nil || !present || state.state != interlockStateHeld {
		t.Fatalf("ambiguous claim did not remain held: %#v %v", state, err)
	}
	if _, _, winner, err := acquireInterlockAndStartClaim(context.Background(), store, target, boot, nil); err == nil || winner.validFor(store) {
		t.Fatal("same-boot retry bypassed ambiguity")
	}

}

func TestC2InterlockResetRequiresExplicitDifferentBootSession(t *testing.T) {
	store, root, target, bootRaw := c2TargetOnlyFixture(t)
	boot, _ := domainDigest(bootRaw)
	if _, _, _, err := acquireInterlockAndStartClaim(context.Background(), store, target, boot, nil); err != nil {
		t.Fatal(err)
	}
	if err := resetInterlockAfterBootChange(context.Background(), store, changedBootResetAuthorization{}, nil); err == nil {
		t.Fatal("reset without authorization succeeded")
	}
	same := issueC2ChangedBootResetFixture(boot)
	if err := resetInterlockAfterBootChange(context.Background(), store, same, nil); err == nil {
		t.Fatal("same-boot reset succeeded")
	}
	different := issueC2ChangedBootResetFixture(c2Digest('a'))
	if err := resetInterlockAfterBootChange(context.Background(), store, different, nil); err != nil {
		t.Fatalf("explicit different-boot reset failed: %v", err)
	}
	state, present, err := readExecutionInterlockState(filepath.Join(store.contractOps, interlockFilename))
	if err != nil || !present || state.state != interlockStateClear || state.bootDigest != different.currentBoot {
		t.Fatalf("changed-boot reset state = %#v, %v", state, err)
	}
	if !clearReceiptExact(store, state) {
		t.Fatal("changed-boot reset did not persist an exact clear receipt")
	}
	if _, err := os.Lstat(claimPathFor(filepath.Join(store.contractOps, startClaimDirectory), target.record.object.Digest())); err != nil {
		t.Fatal("changed-boot reset removed the durable target tombstone")
	}
	restarted, err := OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	next, nextBoot := c2TargetInStore(t, restarted, c2Digest('8'), '9')
	if _, _, winner, err := acquireInterlockAndStartClaim(
		context.Background(), restarted, next, nextBoot, nil,
	); err != nil || !winner.validFor(restarted) {
		t.Fatalf("receipt-backed changed-boot reset did not admit a different target: %v", err)
	}
}

func TestC2InterlockReleaseRequiresTestOnlyDurableTerminalClosure(t *testing.T) {
	store, _, target, bootRaw := c2TargetOnlyFixture(t)
	boot, _ := domainDigest(bootRaw)
	_, claim, _, err := acquireInterlockAndStartClaim(context.Background(), store, target, boot, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := releaseInterlockAfterFinalizedRun(context.Background(), store, terminalClosureRecord{}, claim, nil); err == nil {
		t.Fatal("interlock released without a durable finalized-run record")
	}

	fixture := c2CompleteFixture(t)
	closure := issueC2TerminalClosureFixture(t, fixture)
	if err := releaseInterlockAfterFinalizedRun(context.Background(), fixture.store, closure, fixture.claim, nil); err != nil {
		t.Fatalf("exact durable finalized run did not release: %v", err)
	}
	state, present, err := readExecutionInterlockState(filepath.Join(fixture.store.contractOps, interlockFilename))
	if err != nil || !present || state.state != interlockStateClear {
		t.Fatalf("release state = %#v, %v", state, err)
	}
	if !clearReceiptExact(fixture.store, state) {
		t.Fatal("finalized-run release did not persist an exact clear receipt")
	}
	if fixture.winner.validFor(fixture.store) {
		t.Fatal("winner remained live after finalized-run release")
	}
	restarted, err := OpenObjectStore(fixture.root)
	if err != nil {
		t.Fatal(err)
	}
	next, nextBoot := c2TargetInStore(t, restarted, c2Digest('8'), '9')
	if _, _, nextWinner, err := acquireInterlockAndStartClaim(
		context.Background(), restarted, next, nextBoot, nil,
	); err != nil || !nextWinner.validFor(restarted) {
		t.Fatalf("receipt-backed finalized-run release did not admit a different target: %v", err)
	}

	t.Run("ambiguous-clear-cannot-admit-another-target", func(t *testing.T) {
		ambiguous := c2CompleteFixture(t)
		closure := issueC2TerminalClosureFixture(t, ambiguous)
		fault := func(phase contractFaultPhase) error {
			if phase == faultAfterInterlockSync {
				return errors.New("lost clear result")
			}
			return nil
		}
		if err := releaseInterlockAfterFinalizedRun(
			context.Background(), ambiguous.store, closure, ambiguous.claim, fault,
		); err == nil {
			t.Fatal("ambiguous clear reported success")
		}
		state, present, err := readExecutionInterlockState(
			filepath.Join(ambiguous.store.contractOps, interlockFilename),
		)
		if err != nil || !present || state.state != interlockStateClear {
			t.Fatalf("ambiguous clear state = %#v, %v", state, err)
		}
		if clearReceiptExact(ambiguous.store, state) {
			t.Fatal("faulted clear unexpectedly produced an admission receipt")
		}

		bundle := c2Bundle(t)
		next := c2Target(t, bundle, c2Digest('8'), '9')
		attempt := issueC2AttemptFixture(ambiguous.store, next.AttemptArtifactDigest())
		nextObject := c2SemanticObject(t, contractTargetKind, next.Digest(), next.CanonicalBytes())
		nextRecord, _, err := persistTargetRecord(
			context.Background(), ambiguous.store,
			targetStorageInput{attempt: attempt, object: nextObject}, nil,
		)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, winner, err := acquireInterlockAndStartClaim(
			context.Background(), ambiguous.store, nextRecord,
			next.Input().BootSession.IdentityDigest, nil,
		); err == nil || winner.validFor(ambiguous.store) {
			t.Fatal("receiptless clear admitted another target")
		}
	})

	for _, test := range []struct {
		phase          contractFaultPhase
		receiptVisible bool
	}{
		{faultBeforeClearReceipt, false},
		{faultAfterClearReceiptTemp, false},
		{faultBeforeClearReceiptLink, false},
		{faultAfterClearReceiptLink, true},
		{faultAfterClearReceiptSync, true},
		{faultBeforeClearReceiptOpen, true},
	} {
		t.Run(string(test.phase), func(t *testing.T) {
			faulted := c2CompleteFixture(t)
			closure := issueC2TerminalClosureFixture(t, faulted)
			fault := func(actual contractFaultPhase) error {
				if actual == test.phase {
					return errors.New("lost clear-receipt result")
				}
				return nil
			}
			if err := releaseInterlockAfterFinalizedRun(
				context.Background(), faulted.store, closure, faulted.claim, fault,
			); err == nil || mutationEffect(err) != contractAmbiguous {
				t.Fatal("clear-receipt boundary fault did not report ambiguity")
			}
			clear, present, err := readExecutionInterlockState(
				filepath.Join(faulted.store.contractOps, interlockFilename),
			)
			if err != nil || !present || clear.state != interlockStateClear {
				t.Fatalf("clear-receipt boundary classification = %#v %v", clear, err)
			}
			hex, err := strictDigestHex(clear.digest)
			if err != nil {
				t.Fatal(err)
			}
			_, visibleErr := readExactPrivateFile(
				filepath.Join(faulted.store.contractOps, clearReceiptDirectory, hex), -1,
			)
			if (visibleErr == nil) != test.receiptVisible {
				t.Fatalf("clear receipt visibility=%t err=%v", visibleErr == nil, visibleErr)
			}
			reopened, err := OpenObjectStore(faulted.root)
			if err != nil {
				t.Fatal(err)
			}
			next, boot := c2TargetInStore(t, reopened, c2Digest('8'), '9')
			_, _, winner, acquireErr := acquireInterlockAndStartClaim(
				context.Background(), reopened, next, boot, nil,
			)
			if test.receiptVisible && (acquireErr != nil || !winner.validFor(reopened)) {
				t.Fatalf("exact observed clear receipt did not reopen admission: %v", acquireErr)
			}
			if !test.receiptVisible && (acquireErr == nil || winner.validFor(reopened)) {
				t.Fatal("receiptless clear reopened admission")
			}
		})
	}
}

func TestC2InterlockRejectsCorruptCrossStoreAndAlternateOwnerState(t *testing.T) {
	fixture := c2CompleteFixture(t)
	closure := issueC2TerminalClosureFixture(t, fixture)
	foreign, _ := newObjectStoreForTest(t)
	if err := releaseInterlockAfterFinalizedRun(context.Background(), foreign, closure, fixture.claim, nil); err == nil {
		t.Fatal("foreign store released an interlock")
	}
	statePath := filepath.Join(fixture.store.contractOps, interlockFilename)
	body, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	body[len(body)/2] ^= 1
	if err := os.WriteFile(statePath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := releaseInterlockAfterFinalizedRun(context.Background(), fixture.store, closure, fixture.claim, nil); err == nil {
		t.Fatal("corrupt interlock state released")
	}

	for _, mutation := range []string{"missing", "corrupt", "wrong-cause", "wrong-previous"} {
		t.Run("clear-receipt-"+mutation, func(t *testing.T) {
			subject := c2CompleteFixture(t)
			held := subject.winner.lease.state
			closure := issueC2TerminalClosureFixture(t, subject)
			if err := releaseInterlockAfterFinalizedRun(
				context.Background(), subject.store, closure, subject.claim, nil,
			); err != nil {
				t.Fatal(err)
			}
			clear, present, err := readExecutionInterlockState(
				filepath.Join(subject.store.contractOps, interlockFilename),
			)
			if err != nil || !present || !clearReceiptExact(subject.store, clear) {
				t.Fatalf("release did not establish the mutation precondition: %v", err)
			}
			hex, _ := strictDigestHex(clear.digest)
			receiptPath := filepath.Join(subject.store.contractOps, clearReceiptDirectory, hex)
			switch mutation {
			case "missing":
				err = os.Remove(receiptPath)
			case "corrupt":
				body, readErr := os.ReadFile(receiptPath)
				if readErr != nil {
					t.Fatal(readErr)
				}
				body[len(body)/2] ^= 1
				err = os.WriteFile(receiptPath, body, 0o600)
			case "wrong-cause":
				body, buildErr := clearReceiptBytes(clear, held, "CHANGED_BOOT", "")
				if buildErr != nil {
					t.Fatal(buildErr)
				}
				err = os.WriteFile(receiptPath, body, 0o600)
			case "wrong-previous":
				alternate := held
				alternate.targetDigest = c2Digest('a')
				body, buildErr := clearReceiptBytes(clear, alternate, "FINALIZED_RUN", subject.run.Digest())
				if buildErr != nil {
					t.Fatal(buildErr)
				}
				err = os.WriteFile(receiptPath, body, 0o600)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := syncDirectory(filepath.Dir(receiptPath)); err != nil {
				t.Fatal(err)
			}
			restarted, err := OpenObjectStore(subject.root)
			if err != nil {
				t.Fatal(err)
			}
			next, boot := c2TargetInStore(t, restarted, c2Digest('8'), '9')
			if _, _, winner, err := acquireInterlockAndStartClaim(
				context.Background(), restarted, next, boot, nil,
			); err == nil || winner.validFor(restarted) {
				t.Fatal("invalid clear receipt reopened admission")
			}
		})
	}

	t.Run("start-claim-parent-substitution-invalidates-winner", func(t *testing.T) {
		subject := c2CompleteFixture(t)
		c2SubstituteDirectory(t, filepath.Dir(subject.claim.path))
		if subject.claim.validFor(subject.store) || subject.winner.validFor(subject.store) {
			t.Fatal("claim or winner survived start-claim directory substitution")
		}
		if err := consumeStartClaimWinner(context.Background(), subject.store, &subject.winner); err == nil {
			t.Fatal("winner consumed through a substituted start-claim directory")
		}
	})

	t.Run("start-claim-real-parent-replacement-preserves-unconsumed-winner", func(t *testing.T) {
		subject := c2CompleteFixture(t)
		restore := c2ReplaceDirectoryWithExactClone(t, filepath.Dir(subject.claim.path))
		if subject.claim.validFor(subject.store) || subject.winner.validFor(subject.store) {
			t.Fatal("claim or winner survived a real same-mode parent with exact hard-linked contents")
		}
		if err := consumeStartClaimWinner(context.Background(), subject.store, &subject.winner); err == nil {
			t.Fatal("winner consumed through a replacement start-claim directory")
		}
		restore()
		if !subject.claim.validFor(subject.store) || !subject.winner.validFor(subject.store) {
			t.Fatal("winner did not recover when the retained claim directory was restored")
		}
		if err := consumeStartClaimWinner(context.Background(), subject.store, &subject.winner); err != nil {
			t.Fatalf("restored winner did not consume exactly once: %v", err)
		}
		if err := consumeStartClaimWinner(context.Background(), subject.store, &subject.winner); err == nil {
			t.Fatal("restored winner consumed more than once")
		}
	})

	for _, test := range []struct {
		name string
		path func(c2StorageFixture) string
	}{
		{"start-claim-directory-case-alias-preserves-winner", func(subject c2StorageFixture) string {
			return filepath.Dir(subject.claim.path)
		}},
		{"start-claim-leaf-case-alias-preserves-winner", func(subject c2StorageFixture) string {
			return subject.claim.path
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			subject := c2CompleteFixture(t)
			restore := c2CaseAliasPath(t, test.path(subject))
			if subject.claim.validFor(subject.store) || subject.winner.validFor(subject.store) {
				t.Fatal("claim or winner accepted a case-only path alias")
			}
			if err := consumeStartClaimWinner(context.Background(), subject.store, &subject.winner); err == nil {
				t.Fatal("winner consumed through a case-only path alias")
			}
			restore()
			if err := consumeStartClaimWinner(context.Background(), subject.store, &subject.winner); err != nil {
				t.Fatalf("case-alias refusal burned the restored winner: %v", err)
			}
			if err := consumeStartClaimWinner(context.Background(), subject.store, &subject.winner); err == nil {
				t.Fatal("restored winner consumed more than once")
			}
		})
	}

	t.Run("fixed-ops-real-directory-replacement-refuses-consume", func(t *testing.T) {
		subject := c2CompleteFixture(t)
		restore := c2ReplaceDirectoryWithExactClone(t, subject.store.contractOps)
		if err := consumeStartClaimWinner(context.Background(), subject.store, &subject.winner); err == nil {
			t.Fatal("winner consumed through a replacement fixed ops root")
		}
		restore()
		if err := consumeStartClaimWinner(context.Background(), subject.store, &subject.winner); err != nil {
			t.Fatalf("fixed-root refusal burned the restored winner: %v", err)
		}
	})

	t.Run("fixed-ops-case-alias-refuses-consume", func(t *testing.T) {
		subject := c2CompleteFixture(t)
		restore := c2CaseAliasPath(t, subject.store.contractOps)
		if err := consumeStartClaimWinner(context.Background(), subject.store, &subject.winner); err == nil {
			t.Fatal("winner consumed through a case-only fixed ops root")
		}
		restore()
		if err := consumeStartClaimWinner(context.Background(), subject.store, &subject.winner); err != nil {
			t.Fatalf("fixed-root alias refusal burned the restored winner: %v", err)
		}
	})

	t.Run("fixed-ops-substitution-refuses-consume-and-reset", func(t *testing.T) {
		for _, operation := range []string{"consume", "reset"} {
			t.Run(operation, func(t *testing.T) {
				subject := c2CompleteFixture(t)
				c2SubstituteDirectory(t, subject.store.contractOps)
				switch operation {
				case "consume":
					if err := consumeStartClaimWinner(
						context.Background(), subject.store, &subject.winner,
					); err == nil {
						t.Fatal("winner consumed through a substituted fixed ops root")
					}
				case "reset":
					authorization := issueC2ChangedBootResetFixture(c2Digest('a'))
					if err := resetInterlockAfterBootChange(
						context.Background(), subject.store, authorization, nil,
					); err == nil {
						t.Fatal("boot reset wrote through a substituted fixed ops root")
					}
				}
			})
		}
	})
}

func domainDigest(raw string) (domain.Digest, error) {
	return domain.ParseDigest(raw)
}

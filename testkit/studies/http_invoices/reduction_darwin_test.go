//go:build darwin && cgo

package http_invoices

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/choice/promotion"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/confirmation"
	"github.com/nelsonwerd/countershape/internal/contractmaterialize"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	nodeemit "github.com/nelsonwerd/countershape/internal/emit/node"
	"github.com/nelsonwerd/countershape/internal/observe"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
	reducer "github.com/nelsonwerd/countershape/internal/reduce"
	grade "github.com/nelsonwerd/countershape/internal/reduction"
	"github.com/nelsonwerd/countershape/internal/store"
	"github.com/nelsonwerd/countershape/testkit/httpfixture"
)

const (
	httpA21StudyLabel    = "physical HTTP choicepoint promotion"
	httpA21RestartEnv    = "COUNTERSHAPE_A21_HTTP_RESTART_REQUEST"
	httpA21RestartSchema = "countershape/a2.1/restart/v1"
	httpA21RequestLimit  = 8 << 10
	httpA21ResultLimit   = 2 << 20
)

type httpA21RestartRequest struct {
	Schema     string `json:"schema"`
	StoreRoot  string `json:"store_root"`
	SourcePath string `json:"source_path"`
	ResultPath string `json:"result_path"`
}

type httpA21RestartResult struct {
	Schema                  string   `json:"schema"`
	CompilationDigest       string   `json:"compilation_digest"`
	DecisionRecordDigest    string   `json:"decision_record_digest"`
	ChoicepointDigest       string   `json:"choicepoint_digest"`
	SourceDigest            string   `json:"source_digest"`
	SourceProfileDigest     string   `json:"source_profile_digest"`
	Action                  string   `json:"action"`
	SelectedFields          []string `json:"selected_fields"`
	AllowedTupleCanonical64 []string `json:"allowed_tuple_canonical_base64"`
}

type httpA21PreparedDiagnostic struct {
	Valid               bool
	Digest              string
	DecisionRecord      string
	Choicepoint         string
	Source              string
	SourceProfile       string
	Action              string
	SelectedCount       int
	SelectedPreview     []string
	AllowedTupleCount   int
	AllowedTupleSHA256s []string
}

func httpA21ResolvedStoreTempDir(t testing.TB) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		t.Fatalf("resolved temporary directory is not clean and absolute: %q", root)
	}
	return root
}

func httpA21DescribePrepared(prepared nodeemit.PreparedCompilation) httpA21PreparedDiagnostic {
	selected := prepared.SelectedFields()
	selectedLimit := len(selected)
	if selectedLimit > 8 {
		selectedLimit = 8
	}
	tuples := prepared.AllowedTupleCanonicalBytes()
	tupleLimit := len(tuples)
	if tupleLimit > 8 {
		tupleLimit = 8
	}
	tupleDigests := make([]string, tupleLimit)
	for index, tuple := range tuples[:tupleLimit] {
		digest := sha256.Sum256(tuple)
		tupleDigests[index] = fmt.Sprintf("sha256:%x", digest)
	}
	return httpA21PreparedDiagnostic{
		Valid: prepared.Valid(), Digest: prepared.Digest().String(),
		DecisionRecord: prepared.DecisionRecordDigest().String(), Choicepoint: prepared.ChoicepointDigest().String(),
		Source: prepared.SourceDigest().String(), SourceProfile: prepared.SourceProfileDigest().String(), Action: prepared.Action(),
		SelectedCount: len(selected), SelectedPreview: append([]string(nil), selected[:selectedLimit]...),
		AllowedTupleCount: len(tuples), AllowedTupleSHA256s: tupleDigests,
	}
}

func httpA21TrialControlSummaries(result StudyResult) []string {
	summaries := []string{}
	for _, trial := range result.Trials {
		process := trial.Result.Process()
		primary, hasPrimary := process.PrimaryControl()
		if !hasPrimary && trial.Projected && trial.ProjectionRejection == nil {
			continue
		}
		stderr := process.Stderr()
		stderrDigest := sha256.Sum256([]byte(stderr))
		summaries = append(summaries, fmt.Sprintf(
			"%s#%d repetition=%d projected=%t projection-rejected=%t primary=%s/%t diagnostic=%s exit=%d signal=%s stderr-bytes=%d stderr-sha256=%x",
			trial.Role, trial.Slot.Ordinal(), trial.Slot.Repetition(), trial.Projected,
			trial.ProjectionRejection != nil, primary, hasPrimary, process.DiagnosticCode(),
			process.ExitCode(), process.ExitSignal(), len(stderr), stderrDigest,
		))
	}
	return summaries
}

type httpA21BoundedOutput struct {
	data      []byte
	truncated bool
}

func (w *httpA21BoundedOutput) Write(input []byte) (int, error) {
	const limit = 64 << 10
	if remaining := limit - len(w.data); remaining > 0 {
		if len(input) < remaining {
			remaining = len(input)
		}
		w.data = append(w.data, input[:remaining]...)
	}
	if len(w.data) == limit {
		w.truncated = true
	}
	return len(input), nil
}

func (w httpA21BoundedOutput) String() string {
	if w.truncated {
		return string(w.data) + "\n[child output truncated]"
	}
	return string(w.data)
}

func httpA21WriteExclusive(path string, body []byte, limit int) error {
	if len(body) == 0 || len(body) > limit {
		return fmt.Errorf("body size %d is outside 1..%d", len(body), limit)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	n, writeErr := file.Write(body)
	if writeErr == nil && n != len(body) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func httpA21ReadPrivateRegular(path string, limit int) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() <= 0 || info.Size() > int64(limit) {
		return nil, fmt.Errorf("%s is not a bounded private regular file", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(body) == 0 || len(body) > limit || int64(len(body)) != info.Size() {
		return nil, fmt.Errorf("%s changed size while being read", path)
	}
	return body, nil
}

func httpA21DecodeStrict(body []byte, output any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("trailing JSON value: %v", err)
	}
	return nil
}

func httpA21CleanAbsolute(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path
}

func httpA21StoreAuthorityRefused(err error) bool {
	var typed *store.Error
	return errors.As(err, &typed) && typed.Code == "OBJECT_AUTHORITY_REFUSED"
}

func httpA21StoreInventory(t *testing.T, root string) []string {
	t.Helper()
	entries := make([]string, 0, 32)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("store inventory has no Darwin stat identity for %s", relative)
		}
		identity := fmt.Sprintf("%d\x00%d\x00%d", stat.Dev, stat.Ino, info.ModTime().UnixNano())
		if entry.IsDir() {
			entries = append(entries, fmt.Sprintf("D\x00%s\x00%#o\x00%s", relative, info.Mode().Perm(), identity))
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("store inventory contains non-regular entry %s (%s)", relative, entry.Type())
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(body)
		entries = append(entries, fmt.Sprintf("F\x00%s\x00%#o\x00%d\x00%x\x00%s", relative, info.Mode().Perm(), len(body), digest, identity))
		return nil
	})
	if err != nil {
		t.Fatalf("inventory HTTP ruling store: %v", err)
	}
	sort.Strings(entries)
	return entries
}

func exerciseHTTPP07BBPublication(
	t *testing.T,
	storeRoot string,
	correctStore *store.ObjectStore,
	wrongStore *store.ObjectStore,
	studyID store.StudyID,
	prepared nodeemit.PreparedCompilation,
	alternateStoreRoot string,
	alternateStore *store.ObjectStore,
	alternateStudyID store.StudyID,
	alternatePrepared nodeemit.PreparedCompilation,
) {
	t.Helper()
	bundle, err := nodeemit.CompilePrepared(prepared)
	if err != nil || !bundle.Valid() || bundle.Bundle().DecisionAction() != string(choice.ActionCustomExpectation) {
		t.Fatalf("compile real child-bind HTTP terminal bundle: valid %t, %v", bundle.Valid(), err)
	}
	alternateBundle, err := nodeemit.CompilePrepared(alternatePrepared)
	if err != nil || !alternateBundle.Valid() ||
		alternateBundle.Bundle().DecisionAction() != string(choice.ActionAllowObserved) ||
		alternateBundle.BundleDigest() == bundle.BundleDigest() ||
		bytes.Equal(alternateBundle.Bundle().CanonicalBytes(), bundle.Bundle().CanonicalBytes()) {
		t.Fatalf("compile distinct observed HTTP terminal bundle: valid %t, %v", alternateBundle.Valid(), err)
	}
	headBefore, err := correctStore.OpenHead(context.Background(), studyID)
	if err != nil || headBefore.Stage() != store.StageRuling {
		t.Fatalf("open exact HTTP ruling before terminal publication: %v", err)
	}
	inventoryBefore := httpA21StoreInventory(t, storeRoot)
	refused, err := nodeemit.PublishPrepared(context.Background(), wrongStore, bundle)
	if !httpA21StoreAuthorityRefused(err) || refused.Residue.Valid() || refused.Disposition != "" ||
		!reflect.DeepEqual(httpA21StoreInventory(t, storeRoot), inventoryBefore) {
		t.Fatalf("HTTP terminal authority crossed store instance: %#v, %v", refused, err)
	}

	type outcome struct {
		result nodeemit.PublicationResult
		err    error
	}
	ready := make(chan struct{}, 2)
	start := make(chan struct{})
	finished := make(chan outcome, 2)
	raceContext, cancelRace := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelRace()
	for range 2 {
		go func() {
			ready <- struct{}{}
			<-start
			result, publishErr := nodeemit.PublishPrepared(raceContext, correctStore, bundle)
			finished <- outcome{result: result, err: publishErr}
		}()
	}
	<-ready
	<-ready
	close(start)
	outcomes := make([]outcome, 0, 2)
	for len(outcomes) < 2 {
		select {
		case result := <-finished:
			outcomes = append(outcomes, result)
		case <-raceContext.Done():
			t.Fatalf("HTTP terminal publication race did not finish: %v", raceContext.Err())
		}
	}
	created := 0
	already := 0
	for index, outcome := range outcomes {
		if outcome.err != nil || !outcome.result.Residue.Valid() ||
			outcome.result.Residue.BundleDigest() != bundle.BundleDigest() {
			t.Fatalf("HTTP terminal publication result %d is invalid: %#v, %v", index, outcome.result, outcome.err)
		}
		switch outcome.result.Disposition {
		case nodeemit.PublicationCreated:
			created++
		case nodeemit.PublicationAlreadyCurrent:
			already++
		default:
			t.Fatalf("unknown HTTP terminal publication disposition %q", outcome.result.Disposition)
		}
	}
	if created != 1 || already != 1 || outcomes[0].result.Residue.HeadDigest() != outcomes[1].result.Residue.HeadDigest() {
		t.Fatalf("HTTP terminal race did not converge: created=%d already=%d", created, already)
	}
	terminal, err := correctStore.OpenHead(context.Background(), studyID)
	if err != nil || terminal.Stage() != store.StageResidue || terminal.Revision() != 8 ||
		terminal.CurrentDigest() != bundle.BundleDigest() {
		t.Fatalf("HTTP terminal head differs after publication: %#v, %v", terminal, err)
	}
	inventoryAfter := httpA21StoreInventory(t, storeRoot)
	for _, entry := range inventoryAfter {
		if strings.Contains(entry, ".object-") || strings.Contains(entry, ".head-") {
			t.Fatalf("HTTP terminal publication left a temporary store entry: %q", entry)
		}
	}
	replay, err := nodeemit.PublishPrepared(context.Background(), correctStore, bundle)
	if err != nil || replay.Disposition != nodeemit.PublicationAlreadyCurrent || !replay.Residue.Valid() ||
		replay.Residue.HeadDigest() != terminal.HeadDigest() ||
		!reflect.DeepEqual(httpA21StoreInventory(t, storeRoot), inventoryAfter) {
		t.Fatalf("HTTP terminal replay changed durable state: %#v, %v", replay, err)
	}
	wrongReplay, err := nodeemit.PublishPrepared(context.Background(), wrongStore, bundle)
	if !httpA21StoreAuthorityRefused(err) || wrongReplay.Residue.Valid() || wrongReplay.Disposition != "" ||
		!reflect.DeepEqual(httpA21StoreInventory(t, storeRoot), inventoryAfter) {
		t.Fatalf("HTTP terminal replay crossed retained store authority: %#v, %v", wrongReplay, err)
	}
	restartedStore, err := store.OpenObjectStore(storeRoot)
	if err != nil {
		t.Fatal(err)
	}
	restartedResidue, err := nodeemit.OpenResidue(context.Background(), restartedStore, studyID)
	if err != nil || !restartedResidue.Valid() || restartedResidue.BundleDigest() != bundle.BundleDigest() ||
		restartedResidue.HeadDigest() != terminal.HeadDigest() {
		t.Fatalf("HTTP terminal residue did not reconstruct after restart: %#v, %v", restartedResidue, err)
	}

	alternatePublished, err := nodeemit.PublishPrepared(context.Background(), alternateStore, alternateBundle)
	if err != nil || !alternatePublished.Residue.Valid() ||
		alternatePublished.Disposition != nodeemit.PublicationCreated ||
		alternatePublished.Residue.BundleDigest() != alternateBundle.BundleDigest() {
		t.Fatalf("publish distinct observed HTTP residue: %#v, %v", alternatePublished, err)
	}
	alternateTerminal, err := alternateStore.OpenHead(context.Background(), alternateStudyID)
	if err != nil || alternateTerminal.Stage() != store.StageResidue || alternateTerminal.Revision() != 8 ||
		alternateTerminal.CurrentDigest() != alternateBundle.BundleDigest() {
		t.Fatalf("distinct observed HTTP terminal head differs: %#v, %v", alternateTerminal, err)
	}
	alternateInventoryAfter := httpA21StoreInventory(t, alternateStoreRoot)
	alternateRestartedStore, err := store.OpenObjectStore(alternateStoreRoot)
	if err != nil {
		t.Fatal(err)
	}
	alternateRestartedResidue, err := nodeemit.OpenResidue(
		context.Background(), alternateRestartedStore, alternateStudyID,
	)
	if err != nil || !alternateRestartedResidue.Valid() ||
		alternateRestartedResidue.BundleDigest() != alternateBundle.BundleDigest() ||
		alternateRestartedResidue.HeadDigest() != alternateTerminal.HeadDigest() {
		t.Fatalf("distinct observed HTTP residue did not reconstruct after restart: %#v, %v", alternateRestartedResidue, err)
	}

	outputParent := httpA21ResolvedStoreTempDir(t)
	destination := filepath.Join(outputParent, "http-contract")
	first, err := contractmaterialize.Materialize(context.Background(), restartedStore, restartedResidue, destination)
	if err != nil || !first.Valid() || first.Disposition() != contractmaterialize.Created ||
		first.State() != contractmaterialize.StateCreated || first.Destination() != destination ||
		first.BundleDigest() != restartedResidue.BundleDigest() ||
		first.ResidueHeadDigest() != restartedResidue.HeadDigest() {
		t.Fatalf("create exact HTTP contract output: %#v, %v", first, err)
	}
	assertHTTPP07BBOutput(t, destination, bundle)
	if !reflect.DeepEqual(httpA21StoreInventory(t, storeRoot), inventoryAfter) {
		t.Fatal("HTTP materialization changed the terminal object store")
	}
	outputBefore := httpA21StoreInventory(t, destination)
	second, err := contractmaterialize.Materialize(context.Background(), restartedStore, restartedResidue, destination)
	if err != nil || !second.Valid() || second.Disposition() != contractmaterialize.AlreadyExact ||
		second.State() != contractmaterialize.StateAlreadyExact ||
		!reflect.DeepEqual(httpA21StoreInventory(t, destination), outputBefore) {
		t.Fatalf("idempotent HTTP contract reopen rewrote output: %#v, %v", second, err)
	}
	assertHTTPP07BBOutput(t, destination, bundle)
	if !reflect.DeepEqual(httpA21StoreInventory(t, storeRoot), inventoryAfter) {
		t.Fatal("HTTP exact-existing retry changed the terminal object store")
	}
	for _, file := range bundle.Bundle().Files() {
		body, readErr := os.ReadFile(filepath.Join(destination, file.Path()))
		if readErr != nil || !bytes.Equal(body, file.Content()) {
			t.Fatalf("HTTP materialized file %s differs: %v", file.Path(), readErr)
		}
	}

	concurrentDestination := filepath.Join(outputParent, "http-contract-concurrent")
	type materializationOutcome struct {
		receipt contractmaterialize.MaterializedContract
		err     error
	}
	materializationStart := make(chan struct{})
	materializationDone := make(chan materializationOutcome, 2)
	for range 2 {
		go func() {
			<-materializationStart
			receipt, materializeErr := contractmaterialize.Materialize(
				context.Background(), restartedStore, restartedResidue, concurrentDestination,
			)
			materializationDone <- materializationOutcome{receipt: receipt, err: materializeErr}
		}()
	}
	close(materializationStart)
	materializedCreated := 0
	materializedAlready := 0
	for range 2 {
		outcome := <-materializationDone
		if outcome.err != nil || !outcome.receipt.Valid() {
			t.Fatalf("concurrent HTTP materialization failed: %#v, %v", outcome.receipt, outcome.err)
		}
		switch outcome.receipt.Disposition() {
		case contractmaterialize.Created:
			materializedCreated++
		case contractmaterialize.AlreadyExact:
			materializedAlready++
		default:
			t.Fatalf("concurrent HTTP materialization disposition = %q", outcome.receipt.Disposition())
		}
	}
	if materializedCreated != 1 || materializedAlready != 1 {
		t.Fatalf("concurrent HTTP materialization did not converge: created=%d already=%d", materializedCreated, materializedAlready)
	}
	assertHTTPP07BBOutput(t, concurrentDestination, bundle)
	for _, entry := range httpA21StoreInventory(t, outputParent) {
		if strings.Contains(entry, ".countershape-contract-stage-") {
			t.Fatalf("concurrent HTTP materialization left a private stage: %q", entry)
		}
	}
	if !reflect.DeepEqual(httpA21StoreInventory(t, storeRoot), inventoryAfter) {
		t.Fatal("concurrent HTTP materialization changed the terminal object store")
	}

	distinctDestination := filepath.Join(outputParent, "http-contract-distinct-content-race")
	distinctStart := make(chan struct{})
	distinctDone := make(chan materializationOutcome, 2)
	distinctContext, cancelDistinct := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelDistinct()
	for _, candidate := range []struct {
		objectStore *store.ObjectStore
		residue     nodeemit.Residue
	}{
		{objectStore: restartedStore, residue: restartedResidue},
		{objectStore: alternateRestartedStore, residue: alternateRestartedResidue},
	} {
		candidate := candidate
		go func() {
			<-distinctStart
			receipt, materializeErr := contractmaterialize.Materialize(
				distinctContext, candidate.objectStore, candidate.residue, distinctDestination,
			)
			distinctDone <- materializationOutcome{receipt: receipt, err: materializeErr}
		}()
	}
	close(distinctStart)
	distinctCreated := 0
	distinctRefused := 0
	createdDigest := domain.Digest("")
	for range 2 {
		select {
		case outcome := <-distinctDone:
			if outcome.err == nil && outcome.receipt.Valid() &&
				outcome.receipt.Disposition() == contractmaterialize.Created {
				distinctCreated++
				createdDigest = outcome.receipt.BundleDigest()
				continue
			}
			if outcome.receipt == (contractmaterialize.MaterializedContract{}) &&
				contractmaterialize.IsCode(outcome.err, contractmaterialize.CodeExportIncomplete) &&
				!contractmaterialize.IsCode(outcome.err, contractmaterialize.CodeExportAmbiguous) {
				distinctRefused++
				continue
			}
			t.Fatalf("distinct-content materialization outcome = %#v, %v", outcome.receipt, outcome.err)
		case <-distinctContext.Done():
			t.Fatalf("distinct-content materialization race did not finish: %v", distinctContext.Err())
		}
	}
	if distinctCreated != 1 || distinctRefused != 1 {
		t.Fatalf("distinct-content materialization race = created %d, refused %d", distinctCreated, distinctRefused)
	}
	switch createdDigest {
	case bundle.BundleDigest():
		assertHTTPP07BBOutput(t, distinctDestination, bundle)
	case alternateBundle.BundleDigest():
		assertHTTPP07BBOutput(t, distinctDestination, alternateBundle)
	default:
		t.Fatalf("distinct-content race created unknown bundle %s", createdDigest)
	}
	for _, entry := range httpA21StoreInventory(t, outputParent) {
		if strings.Contains(entry, ".countershape-contract-stage-") {
			t.Fatalf("distinct-content materialization left a private stage: %q", entry)
		}
	}
	if !reflect.DeepEqual(httpA21StoreInventory(t, storeRoot), inventoryAfter) ||
		!reflect.DeepEqual(httpA21StoreInventory(t, alternateStoreRoot), alternateInventoryAfter) {
		t.Fatal("distinct-content materialization changed a terminal object store")
	}
}

func assertHTTPP07BBOutput(t *testing.T, destination string, prepared nodeemit.PreparedBundle) {
	t.Helper()
	bundle := prepared.Bundle()
	root, err := os.Lstat(destination)
	if err != nil || !root.IsDir() || root.Mode()&os.ModeSymlink != 0 || root.Mode().Perm() != 0o700 ||
		root.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		t.Fatalf("HTTP contract root facts differ: %v, %v", root, err)
	}
	entries, err := os.ReadDir(destination)
	if err != nil || len(entries) != len(bundle.Files()) {
		t.Fatalf("HTTP contract roster count differs: %d, %v", len(entries), err)
	}
	wantNames := make([]string, len(bundle.Files()))
	for index, file := range bundle.Files() {
		wantNames[index] = file.Path()
	}
	sort.Strings(wantNames)
	actualNames := make([]string, len(entries))
	for index, entry := range entries {
		actualNames[index] = entry.Name()
	}
	sort.Strings(actualNames)
	if !slices.Equal(actualNames, wantNames) {
		t.Fatalf("HTTP contract roster = %v, want %v", actualNames, wantNames)
	}
	for _, file := range bundle.Files() {
		path := filepath.Join(destination, file.Path())
		info, statErr := os.Lstat(path)
		if statErr != nil {
			t.Fatalf("HTTP contract member %s stat failed: %v", file.Path(), statErr)
		}
		stat, statOK := info.Sys().(*syscall.Stat_t)
		body, readErr := os.ReadFile(path)
		digest := sha256.Sum256(body)
		if readErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
			info.Mode().Perm() != 0o644 || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 ||
			!statOK || stat.Nlink != 1 || int64(file.ByteCount()) != info.Size() ||
			file.ByteSHA256().String() != fmt.Sprintf("sha256:%x", digest) || !bytes.Equal(body, file.Content()) {
			t.Fatalf("HTTP contract member %s facts differ: %v", file.Path(), readErr)
		}
	}
}

func assertHTTPCompilationFreshProcessRestart(
	t *testing.T,
	storeRoot string,
	source contractsource.PortableSource,
	prepared nodeemit.PreparedCompilation,
) {
	t.Helper()
	protocolRoot := t.TempDir()
	if err := os.Chmod(protocolRoot, 0o700); err != nil {
		t.Fatalf("make HTTP restart protocol directory private: %v", err)
	}
	sourcePath := filepath.Join(protocolRoot, "portable-source.json")
	requestPath := filepath.Join(protocolRoot, "request.json")
	resultPath := filepath.Join(protocolRoot, "result.json")
	childCWD := filepath.Join(protocolRoot, "child-cwd")
	if err := os.Mkdir(childCWD, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := httpA21WriteExclusive(sourcePath, source.CanonicalBytes(), contractsource.MaxSourceCanonicalBytes); err != nil {
		t.Fatalf("write private HTTP restart source: %v", err)
	}
	requestBody, err := json.Marshal(httpA21RestartRequest{
		Schema: httpA21RestartSchema, StoreRoot: storeRoot, SourcePath: sourcePath, ResultPath: resultPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := httpA21WriteExclusive(requestPath, requestBody, httpA21RequestLimit); err != nil {
		t.Fatalf("write private HTTP restart request: %v", err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(
		ctx,
		executable,
		"-test.run=^TestHTTPCompilationFreshProcessRestartHelper$",
		"-test.count=1",
		"-test.timeout=90s",
	)
	command.Dir = childCWD
	command.Env = []string{httpA21RestartEnv + "=" + requestPath}
	var output httpA21BoundedOutput
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			t.Fatalf("fresh-process HTTP compilation restart timed out: %v; %s", ctx.Err(), output.String())
		}
		t.Fatalf("fresh-process HTTP compilation restart failed: %v; %s", err, output.String())
	}
	resultBody, err := httpA21ReadPrivateRegular(resultPath, httpA21ResultLimit)
	if err != nil {
		t.Fatalf("read private HTTP restart result: %v", err)
	}
	var result httpA21RestartResult
	if err := httpA21DecodeStrict(resultBody, &result); err != nil {
		t.Fatalf("decode strict HTTP restart result: %v", err)
	}
	if result.Schema != httpA21RestartSchema ||
		result.CompilationDigest != prepared.Digest().String() ||
		result.DecisionRecordDigest != prepared.DecisionRecordDigest().String() ||
		result.ChoicepointDigest != prepared.ChoicepointDigest().String() ||
		result.SourceDigest != prepared.SourceDigest().String() ||
		result.SourceProfileDigest != prepared.SourceProfileDigest().String() ||
		result.Action != prepared.Action() || !slices.Equal(result.SelectedFields, prepared.SelectedFields()) {
		t.Fatalf("fresh-process HTTP compilation summaries changed: %#v", result)
	}
	wantTuples := prepared.AllowedTupleCanonicalBytes()
	if len(result.AllowedTupleCanonical64) != len(wantTuples) {
		t.Fatalf("fresh-process HTTP tuple count = %d, want %d", len(result.AllowedTupleCanonical64), len(wantTuples))
	}
	for index, encoded := range result.AllowedTupleCanonical64 {
		decoded, err := base64.StdEncoding.Strict().DecodeString(encoded)
		if err != nil || !bytes.Equal(decoded, wantTuples[index]) {
			t.Fatalf("fresh-process HTTP tuple %d changed: %v", index, err)
		}
	}
}

func TestHTTPCompilationFreshProcessRestartHelper(t *testing.T) {
	requestPath := os.Getenv(httpA21RestartEnv)
	if requestPath == "" {
		return
	}
	if !httpA21CleanAbsolute(requestPath) {
		t.Fatal("HTTP restart request path is not clean and absolute")
	}
	requestRoot := filepath.Dir(requestPath)
	rootInfo, err := os.Lstat(requestRoot)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode().Perm()&0o077 != 0 {
		t.Fatalf("HTTP restart request directory is not private: %v", err)
	}
	requestBody, err := httpA21ReadPrivateRegular(requestPath, httpA21RequestLimit)
	if err != nil {
		t.Fatal(err)
	}
	var request httpA21RestartRequest
	if err := httpA21DecodeStrict(requestBody, &request); err != nil {
		t.Fatal(err)
	}
	if request.Schema != httpA21RestartSchema || !httpA21CleanAbsolute(request.StoreRoot) ||
		!httpA21CleanAbsolute(request.SourcePath) || !httpA21CleanAbsolute(request.ResultPath) ||
		filepath.Dir(request.SourcePath) != requestRoot || filepath.Dir(request.ResultPath) != requestRoot ||
		request.SourcePath == request.ResultPath || request.SourcePath == requestPath || request.ResultPath == requestPath {
		t.Fatal("HTTP restart request is outside the closed path protocol")
	}
	if _, err := os.Lstat(request.ResultPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("HTTP restart result path already exists or is inaccessible: %v", err)
	}
	sourceBody, err := httpA21ReadPrivateRegular(request.SourcePath, contractsource.MaxSourceCanonicalBytes)
	if err != nil {
		t.Fatal(err)
	}
	source, err := contractsource.Parse(sourceBody)
	if err != nil {
		t.Fatalf("strict HTTP restart source parse: %v", err)
	}
	studyID, err := store.NewStudyID(httpA21StudyLabel)
	if err != nil {
		t.Fatal(err)
	}
	objectStore, err := store.OpenObjectStore(request.StoreRoot)
	if err != nil {
		t.Fatal(err)
	}
	ruling, err := promotion.OpenRuling(context.Background(), objectStore, studyID)
	if err != nil {
		t.Fatal(err)
	}
	preparation, err := promotion.PreparePortableRuling(context.Background(), objectStore, ruling)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := nodeemit.PrepareCompilation(context.Background(), objectStore, preparation, source)
	if err != nil || !prepared.Valid() {
		t.Fatalf("fresh-process HTTP preparation is invalid: %#v, %v", httpA21DescribePrepared(prepared), err)
	}
	tuples := prepared.AllowedTupleCanonicalBytes()
	encodedTuples := make([]string, len(tuples))
	for index, tuple := range tuples {
		encodedTuples[index] = base64.StdEncoding.EncodeToString(tuple)
	}
	resultBody, err := json.Marshal(httpA21RestartResult{
		Schema: httpA21RestartSchema, CompilationDigest: prepared.Digest().String(),
		DecisionRecordDigest: prepared.DecisionRecordDigest().String(),
		ChoicepointDigest:    prepared.ChoicepointDigest().String(), SourceDigest: prepared.SourceDigest().String(),
		SourceProfileDigest: prepared.SourceProfileDigest().String(), Action: prepared.Action(),
		SelectedFields: prepared.SelectedFields(), AllowedTupleCanonical64: encodedTuples,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := httpA21WriteExclusive(request.ResultPath, resultBody, httpA21ResultLimit); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPPhysicalTenantSeedNeighborChangesExactLabeledMapWithStableRoster(t *testing.T) {
	referenceStimulus, err := newInvoiceStimulus()
	if err != nil {
		t.Fatal(err)
	}
	tenantlessStimulus, err := newInvoiceStimulusWithSeed(httpfixture.TenantlessSeedJSON())
	if err != nil {
		t.Fatal(err)
	}
	referenceWire, err := counterhttp.EncodeRequest(referenceStimulus, 43210)
	if err != nil {
		t.Fatal(err)
	}
	tenantlessWire, err := counterhttp.EncodeRequest(tenantlessStimulus, 43210)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(referenceWire.Bytes(), tenantlessWire.Bytes()) {
		t.Fatal("tenant-seed neighbor changed request bytes instead of only fixture authority")
	}

	baselineConfig := referenceConfig(t)
	baselineConfig.Repetitions = 1
	baselineConfig.MaxTotalTrials = 4
	baselineConfig.StimulusOverride = &referenceStimulus
	reference, err := Run(context.Background(), baselineConfig)
	if err != nil {
		t.Fatal(err)
	}
	tenantlessConfig := referenceConfig(t)
	tenantlessConfig.Repetitions = 1
	tenantlessConfig.MaxTotalTrials = 4
	tenantlessConfig.Purpose = domain.AttemptReduction
	tenantlessConfig.StimulusOverride = &tenantlessStimulus
	tenantless, err := Run(context.Background(), tenantlessConfig)
	if err != nil {
		t.Fatal(err)
	}
	for name, result := range map[string]StudyResult{"reference": reference, "tenantless": tenantless} {
		if !result.HasOutcomeMap || len(result.OutcomeMap.Entries()) != 4 ||
			len(result.OutcomeMap.Exclusions()) != 0 || !result.OutcomeMap.Divergence() {
			t.Fatalf("%s stable-roster map entries=%d exclusions=%d divergence=%t",
				name, len(result.OutcomeMap.Entries()), len(result.OutcomeMap.Exclusions()), result.OutcomeMap.Divergence())
		}
	}
	assessment := compare.AssessPreservation(reference.OutcomeMap, tenantless.OutcomeMap)
	if !assessment.Valid() || assessment.Relation() != compare.PreservationDifferent ||
		assessment.BaselinePreservationDigest().String() == assessment.ObservedPreservationDigest().String() {
		t.Fatalf("tenant seed shape trap was not exact labeled-map CHANGES: relation=%s reason=%s",
			assessment.Relation(), assessment.ReasonCode())
	}
}

func TestHTTPPhysicalReducerRemovesIrrelevantSeedWithFreshEvidence(t *testing.T) {
	base, err := newInvoiceStimulus()
	if err != nil {
		t.Fatal(err)
	}
	noise, err := counterhttp.NewSeedFile("z-noise.txt", []byte("not-consumed\n"), counterhttp.SeedMode0644)
	if err != nil {
		t.Fatal(err)
	}
	noisySeeds := append(base.Seeds(), noise)
	noisy, err := counterhttp.NewHTTPStimulus(counterhttp.HTTPStimulusConfig{
		Method: base.Method(), Path: base.Path(), Query: base.Query(), Headers: base.Headers(),
		Body: base.Body(), Seeds: noisySeeds,
	})
	if err != nil {
		t.Fatal(err)
	}
	baselineConfig := httpPhysicalReductionConfig(t)
	baselineConfig.StimulusOverride = &noisy
	baselineCheckpoint, err := Run(context.Background(), baselineConfig)
	if err != nil {
		t.Fatal(err)
	}
	baselineResult, err := Run(context.Background(), baselineConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !baselineResult.HasOutcomeMap || baselineResult.OutcomeMap.Phase() != domain.AttemptDiscovery ||
		len(baselineResult.OutcomeMap.Entries()) != 3 || len(baselineResult.OutcomeMap.Exclusions()) != 1 {
		t.Fatalf(
			"physical HTTP baseline did not retain the stable A/B/C map plus excluded D: has-map=%t phase=%s entries=%d exclusions=%d batches=%v trial-controls=%v",
			baselineResult.HasOutcomeMap, baselineResult.OutcomeMap.Phase(), len(baselineResult.OutcomeMap.Entries()),
			len(baselineResult.OutcomeMap.Exclusions()), studyBatchSummaries(baselineResult),
			httpA21TrialControlSummaries(baselineResult),
		)
	}
	if !baselineCheckpoint.HasOutcomeMap || baselineCheckpoint.Plan.Digest() != baselineResult.Plan.Digest() ||
		compare.AssessPreservation(baselineCheckpoint.OutcomeMap, baselineResult.OutcomeMap).Relation() != compare.PreservationEqual ||
		bytes.Equal(baselineCheckpoint.OutcomeMap.CanonicalBytes(), baselineResult.OutcomeMap.CanonicalBytes()) {
		t.Fatal("physical HTTP baseline checkpoint did not retain fresh equivalent evidence")
	}
	baseline, err := compare.RequireDivergence(baselineResult.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := counterhttp.NewHTTPReductionPolicy(counterhttp.HTTPReductionPolicyConfig{
		Anchor: noisy, PinnedSeedPaths: []string{httpfixture.SeedFilename},
		EnabledRules: []counterhttp.HTTPReducerID{counterhttp.HTTPSeedRemove},
	})
	if err != nil {
		t.Fatal(err)
	}
	budget, err := reducer.NewBudgetFromWorldPlan(baselineResult.Plan)
	if err != nil {
		t.Fatal(err)
	}
	if budget.ProposalLimit() != 2 || budget.CandidateTrialLimit() != 12 || budget.WallLimit() != 3*time.Minute {
		t.Fatalf("compiled HTTP reduction budget = %d/%d/%s", budget.ProposalLimit(), budget.CandidateTrialLimit(), budget.WallLimit())
	}
	var evaluatorErr error
	var minimizedStudy StudyResult
	run, err := reducer.Run(context.Background(), reducer.RunInput[counterhttp.HTTPStimulus]{
		Original: noisy,
		Reference: func(stimulus counterhttp.HTTPStimulus) (domain.Digest, reducer.Measure, bool) {
			measure, measureErr := counterhttp.MeasureHTTPStimulus(stimulus, policy)
			return stimulus.Digest(), measure, measureErr == nil
		},
		Enumerate: func(_ context.Context, stimulus counterhttp.HTTPStimulus) ([]reducer.TypedProposal[counterhttp.HTTPStimulus], error) {
			neighbors, enumerateErr := counterhttp.EnumerateHTTPNeighbors(stimulus, policy)
			if enumerateErr != nil {
				return nil, enumerateErr
			}
			proposals := make([]reducer.TypedProposal[counterhttp.HTTPStimulus], len(neighbors))
			for index, neighbor := range neighbors {
				proposals[index] = reducer.TypedProposal[counterhttp.HTTPStimulus]{
					Stimulus: neighbor.Stimulus(), Neighbor: neighbor.Neighbor(),
				}
			}
			return proposals, nil
		},
		Evaluate: func(ctx context.Context, stimulus counterhttp.HTTPStimulus, _ reducer.Neighbor, purpose domain.AttemptPurpose, allowance reducer.EvaluationAllowance) (reducer.EvaluationObservation, error) {
			config := httpPhysicalReductionConfig(t)
			requiredTrials := uint64(config.MaxTotalTrials)
			if allowance.RemainingCandidateTrials < requiredTrials || !time.Now().Before(allowance.WallDeadline) {
				return reducer.EvaluationObservation{}, context.DeadlineExceeded
			}
			config.Purpose = purpose
			config.StimulusOverride = &stimulus
			evaluationContext, cancel := context.WithDeadline(ctx, allowance.WallDeadline)
			defer cancel()
			result, runErr := Run(evaluationContext, config)
			if runErr != nil {
				evaluatorErr = runErr
				return reducer.EvaluationObservation{}, runErr
			}
			if !result.HasOutcomeMap {
				evaluatorErr = context.Canceled
				return reducer.EvaluationObservation{}, evaluatorErr
			}
			outcome := result.OutcomeMap
			if compare.AssessPreservation(baseline.OutcomeMap(), outcome).Relation() == compare.PreservationEqual {
				minimizedStudy = result
			}
			return reducer.EvaluationObservation{OutcomeMap: &outcome, CandidateTrials: uint64(len(result.Trials))}, nil
		},
		Baseline: baseline, ReducerSet: policy.ReducerSet(), Budget: budget,
	})
	if err != nil {
		t.Fatal(err)
	}
	if evaluatorErr != nil {
		t.Fatalf("physical HTTP reducer evaluator failed: %v", evaluatorErr)
	}
	if !run.Valid() || run.DraftGrade() != reducer.GradeBestKnown || !run.HasAcceptedReduction() ||
		run.Transcript().FinalSweepState() != reducer.FinalSweepComplete || len(run.Transcript().Entries()) != 1 ||
		len(run.Transcript().AcceptedPath()) != 1 {
		t.Fatalf("unexpected physical HTTP reduction: grade=%s accepted=%t sweep=%s entries=%d limits=%v",
			run.DraftGrade(), run.HasAcceptedReduction(), run.Transcript().FinalSweepState(),
			len(run.Transcript().Entries()), run.Transcript().Limitations())
	}
	evaluation := run.Transcript().Entries()[0].Evaluation()
	if evaluation.Purpose() != domain.AttemptReduction || evaluation.Decision() != reducer.Preserves ||
		!evaluation.LogicalNonReuseWithBaseline() || len(evaluation.ObservedAttemptDigests()) != 12 {
		t.Fatalf("physical HTTP preserving evidence is incomplete: purpose=%s decision=%s attempts=%d",
			evaluation.Purpose(), evaluation.Decision(), len(evaluation.ObservedAttemptDigests()))
	}
	draft, present, err := run.CompletedSweepDraft()
	if err != nil || !present || !draft.Valid() || len(draft.NeighborDigests()) != 0 ||
		draft.CurrentStimulusDigest() != run.MinimizedStimulusDigest() {
		t.Fatalf("physical HTTP empty-neighbor completed sweep was not retained: present=%t err=%v", present, err)
	}
	sweepStore, err := store.OpenReductionSweepStore(filepath.Join(httpA21ResolvedStoreTempDir(t), "sweep-store"))
	if err != nil {
		t.Fatal(err)
	}
	authority, err := sweepStore.Publish(context.Background(), draft)
	if err != nil {
		t.Fatal(err)
	}
	finalized, err := grade.Finalize(context.Background(), run, &grade.SweepCompletion{
		Store: sweepStore, Draft: draft, Authority: authority,
	})
	if err != nil || !finalized.Valid() || finalized.Grade().Status() != grade.StatusOneMinimalUnder {
		t.Fatalf("physical HTTP durable grade was not quantified: status=%s err=%v",
			finalized.Grade().Status(), err)
	}
	if !minimizedStudy.HasOutcomeMap || minimizedStudy.Stimulus.Digest() != run.MinimizedStimulusDigest() {
		t.Fatal("physical HTTP reducer did not retain the exact minimized preserving study")
	}
	reducedBaseline, err := compare.RequireDivergence(minimizedStudy.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	confirmationConfig := httpPhysicalReductionConfig(t)
	confirmationConfig.Purpose = domain.AttemptConfirmation
	confirmationConfig.StimulusOverride = &minimizedStudy.Stimulus
	confirmationConfig.Confirmation = &ConfirmationInput{
		ReducedBaseline: reducedBaseline, ReductionRun: run, ReductionResult: finalized,
	}
	confirmedStudy, err := Run(context.Background(), confirmationConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !confirmedStudy.HasConfirmation || !confirmedStudy.Confirmation.Valid() || !confirmedStudy.HasOutcomeMap ||
		confirmedStudy.OutcomeMap.Phase() != domain.AttemptConfirmation ||
		confirmedStudy.OutcomeMap.ScheduleStartOffset() != 1 || len(confirmedStudy.OutcomeMap.Exclusions()) != 1 ||
		confirmedStudy.OutcomeMap.Exclusions()[0].Classification != observe.Unstable ||
		len(confirmedStudy.Confirmation.Draft().PhysicalFacts()) != 12 ||
		confirmedStudy.StartSpec.Authority() != counterhttp.HTTPPortableStartAuthorityV1 ||
		confirmedStudy.Readiness.Protocol() != counterhttp.PortableReadinessProtocolV1 {
		t.Fatal("physical HTTP confirmation did not reproduce the complete A/B/C plus unstable-D disposition")
	}
	confirmationDraft := confirmedStudy.Confirmation.Draft()
	if parsed, parseErr := confirmation.ParseRecord(confirmationDraft.CanonicalBytes()); parseErr != nil || parsed.Digest() != confirmationDraft.Digest() {
		t.Fatalf("physical HTTP confirmation did not round trip strictly: %v", parseErr)
	}
	confirmationStore, err := store.OpenObjectStore(filepath.Join(httpA21ResolvedStoreTempDir(t), "confirmation-store"))
	if err != nil {
		t.Fatal(err)
	}
	object, err := store.NewSemanticObject("FreshConfirmation", confirmationDraft.Digest(), confirmationDraft.CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	confirmationAuthority, err := confirmationStore.Publish(context.Background(), object)
	if err != nil {
		t.Fatal(err)
	}
	if err := confirmationStore.Validate(context.Background(), object, confirmationAuthority); err != nil {
		t.Fatal(err)
	}
	originalArtifact, err := choice.NewCanonicalArtifact(
		"HTTPStimulus", baselineResult.Stimulus.Digest(), baselineResult.Stimulus.CanonicalBytes(),
	)
	if err != nil {
		t.Fatal(err)
	}
	minimizedArtifact, err := choice.NewCanonicalArtifact(
		"HTTPStimulus", confirmedStudy.Stimulus.Digest(), confirmedStudy.Stimulus.CanonicalBytes(),
	)
	if err != nil {
		t.Fatal(err)
	}
	reveals := make([]choice.CandidateReveal, len(confirmedStudy.CandidateBindings))
	for index, binding := range confirmedStudy.CandidateBindings {
		reveals[index] = choice.CandidateReveal{
			CandidateExecutionKey: binding.Key(), DisplayRef: string(confirmedStudy.CandidateRoles[binding.Key()]),
			ProducerMetadata: "local deterministic HTTP fixture",
		}
	}
	choicepoint, err := choice.NewChoicepointRecord(choice.ChoicepointInput{
		Scenario: "Which exact invoice response behavior should become the accepted contract?",
		Plan:     confirmedStudy.Plan, Envelope: confirmedStudy.Envelope,
		CandidateBindings: confirmedStudy.CandidateBindings, OriginalStimulus: originalArtifact,
		MinimizedStimulus: minimizedArtifact, Confirmation: confirmationDraft.Record(), CandidateReveals: reveals,
		EvidenceReceipts: []domain.ReceiptReference{},
	})
	if err != nil || !choicepoint.Valid() {
		t.Fatalf("physical HTTP Choicepoint construction failed: %v", err)
	}
	parsedChoicepoint, err := choice.ParseChoicepointRecord(choicepoint.CanonicalBytes())
	if err != nil || parsedChoicepoint.Digest() != choicepoint.Digest() {
		t.Fatalf("physical HTTP Choicepoint did not round trip strictly: %v", err)
	}
	blind, err := choice.NewBlindView(parsedChoicepoint)
	if err != nil || len(blind.DTO().Cards()) != confirmedStudy.OutcomeMap.DistinctProjectionCount() {
		t.Fatalf("physical HTTP blind DTO did not group exact eligible outcomes: %v", err)
	}
	httpFields := []string{
		string(counterhttp.HTTPFieldStatus), string(counterhttp.HTTPFieldContentType),
		string(counterhttp.HTTPFieldBodyKind), string(counterhttp.HTTPFieldBodyMetadata),
	}
	blindDTO := blind.DTO()
	if blindDTO.ProjectionMode() != "ADAPTER_BOUND_PORTABLE_FIELDS_V1" ||
		!slices.Equal(blindDTO.SelectableFields(), httpFields) ||
		!slices.Equal(blindDTO.DifferingFields(), []string{
			string(counterhttp.HTTPFieldStatus), string(counterhttp.HTTPFieldBodyKind),
			string(counterhttp.HTTPFieldBodyMetadata),
		}) {
		t.Fatalf("physical HTTP Choicepoint lacks exact portable profile order or separation: selectable=%#v differing=%#v mode=%q",
			blindDTO.SelectableFields(), blindDTO.DifferingFields(), blindDTO.ProjectionMode())
	}
	for _, card := range blindDTO.Cards() {
		if len(card.Fields) != len(httpFields) {
			t.Fatalf("HTTP blind card has %d fields, want %d", len(card.Fields), len(httpFields))
		}
		for index, field := range card.Fields {
			if field.FieldID != httpFields[index] {
				t.Fatalf("HTTP blind card field %d = %q, want %q", index, field.FieldID, httpFields[index])
			}
		}
		contentType := card.Fields[1]
		contentTypeBytes, decodeErr := base64.StdEncoding.Strict().DecodeString(contentType.CanonicalJSONBase64)
		if decodeErr != nil || base64.StdEncoding.EncodeToString(contentTypeBytes) != contentType.CanonicalJSONBase64 ||
			contentType.Tag != string(choice.ValueOrderedStringList) || contentType.Text != "" || contentType.Boolean ||
			!bytes.Equal(contentTypeBytes, []byte(`["application/json"]`)) {
			t.Fatalf("HTTP blind card lost exact ordered content-type evidence: %#v", contentType)
		}
	}
	excluded := confirmedStudy.OutcomeMap.Exclusions()[0]
	var excludedBinding domain.CandidateExecutionBinding
	var excludedReveal choice.CandidateReveal
	for index, binding := range confirmedStudy.CandidateBindings {
		if binding.Key() == excluded.CandidateKey {
			excludedBinding = binding
			excludedReveal = reveals[index]
			break
		}
	}
	if !excludedBinding.Valid() {
		t.Fatal("physical HTTP excluded candidate lacks its exact binding")
	}
	blindBytes := blind.DTO().CanonicalBytes()
	for _, forbidden := range []string{
		excluded.CandidateKey.String(), excludedBinding.Identity().TreeIdentityDigest.String(),
		excludedReveal.DisplayRef, excludedReveal.ProducerMetadata,
	} {
		if forbidden != "" && bytes.Contains(blindBytes, []byte(forbidden)) {
			t.Fatalf("physical HTTP blind DTO leaked excluded-candidate provenance %q", forbidden)
		}
	}
	decisionSession, err := choice.NewSession(parsedChoicepoint)
	if err != nil {
		t.Fatal(err)
	}
	_, reveal, err := decisionSession.Reveal()
	if err != nil {
		t.Fatalf("physical HTTP early reveal failed: %v", err)
	}
	revealedExclusions := reveal.Exclusions()
	if len(revealedExclusions) != 1 ||
		revealedExclusions[0].Candidate.CandidateExecutionKey != excluded.CandidateKey.String() ||
		revealedExclusions[0].Candidate.TreeIdentityDigest != excludedBinding.Identity().TreeIdentityDigest.String() ||
		revealedExclusions[0].Candidate.DisplayRef != excludedReveal.DisplayRef ||
		revealedExclusions[0].Candidate.ProducerMetadata != excludedReveal.ProducerMetadata ||
		revealedExclusions[0].Classification != string(observe.Unstable) {
		t.Fatalf("physical HTTP reveal did not restore exactly one unstable exclusion with provenance: %#v", revealedExclusions)
	}
	choiceObject, err := store.NewSemanticObject("Choicepoint", choicepoint.Digest(), choicepoint.CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	choiceAuthority, err := confirmationStore.Publish(context.Background(), choiceObject)
	if err != nil || confirmationStore.Validate(context.Background(), choiceObject, choiceAuthority) != nil {
		t.Fatalf("physical HTTP Choicepoint did not persist immutably: %v", err)
	}

	promotionRoot := filepath.Join(httpA21ResolvedStoreTempDir(t), "promotion-store")
	promotionStore, err := store.OpenObjectStore(promotionRoot)
	if err != nil {
		t.Fatal(err)
	}
	studyID, err := store.NewStudyID(httpA21StudyLabel)
	if err != nil {
		t.Fatal(err)
	}
	head, err := promotionStore.CreateStudy(context.Background(), studyID, confirmedStudy.Plan)
	if err != nil {
		t.Fatal(err)
	}
	head, err = promotionStore.AdvanceBaseline(context.Background(), head, baselineCheckpoint.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	head, err = promotionStore.AdvanceDivergence(context.Background(), head, baseline)
	if err != nil {
		t.Fatal(err)
	}
	head, err = promotionStore.AdvanceReduction(context.Background(), head, run)
	if err != nil {
		t.Fatal(err)
	}
	storedConfirmation, err := promotion.PersistConfirmation(context.Background(), promotionStore, head, confirmationDraft)
	if err != nil {
		t.Fatal(err)
	}
	ready, err := promotion.Promote(context.Background(), promotionStore, storedConfirmation, promotion.ChoicepointRequest{
		Scenario: "Which exact invoice response behavior should become the accepted contract?",
		Plan:     confirmedStudy.Plan, Envelope: confirmedStudy.Envelope,
		CandidateBindings: confirmedStudy.CandidateBindings, OriginalStimulus: originalArtifact,
		MinimizedStimulus: minimizedArtifact, CandidateReveals: reveals,
		EvidenceReceipts: []domain.ReceiptReference{},
	})
	if err != nil || ready.Record().Digest() != choicepoint.Digest() {
		t.Fatalf("physical HTTP store-bound Choicepoint promotion failed: %v", err)
	}
	restartedStore, err := store.OpenObjectStore(promotionRoot)
	if err != nil {
		t.Fatal(err)
	}
	reopenedReady, err := promotion.OpenReady(context.Background(), restartedStore, studyID)
	if err != nil || reopenedReady.Record().Digest() != ready.Record().Digest() {
		t.Fatalf("physical HTTP CHOICEPOINT_READY did not survive restart: %v", err)
	}
	readyRecord := reopenedReady.Record()

	smuggledContentType, err := choice.OrderedStringListValue([]string{"application/json"})
	if err != nil {
		t.Fatal(err)
	}
	if _, smuggleErr := choice.NewSelectedTuple(
		readyRecord,
		[]string{string(counterhttp.HTTPFieldStatus)},
		[]choice.FieldValue{{FieldID: string(counterhttp.HTTPFieldContentType), Value: smuggledContentType}},
	); !choice.IsRefusal(smuggleErr, choice.CodeIncompleteTuple) {
		t.Fatalf("same-cardinality unselected HTTP field smuggled into custom expectation: %v", smuggleErr)
	}

	customContentTypes := []string{"application/problem+json", "", "application/problem+json"}
	customContentType, err := choice.OrderedStringListValue(customContentTypes)
	if err != nil {
		t.Fatal(err)
	}
	contentTypeExpectation, err := choice.NewSelectedTuple(
		readyRecord,
		[]string{string(counterhttp.HTTPFieldContentType)},
		[]choice.FieldValue{{FieldID: string(counterhttp.HTTPFieldContentType), Value: customContentType}},
	)
	if err != nil {
		t.Fatal(err)
	}
	contentTypeInput := choice.RulingDraftInput{
		Action: choice.ActionCustomExpectation, SelectedFields: []string{string(counterhttp.HTTPFieldContentType)},
		AllowedAliases: []string{}, CustomExpectation: &contentTypeExpectation,
		CustomReviewer: "physical-http-content-type-reviewer", CustomReviewEvidence: confirmationDraft.Digest(),
	}
	contentTypeSession, err := choice.NewSession(readyRecord)
	if err != nil {
		t.Fatal(err)
	}
	contentTypeSession, _, err = contentTypeSession.Reveal()
	if err != nil {
		t.Fatal(err)
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness, choice.SurfaceMinimizedWitness, choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations, choice.SurfaceNonassertedFields, choice.SurfaceProvenance,
	} {
		contentTypeSession, err = contentTypeSession.Visit(surface)
		if err != nil {
			t.Fatal(err)
		}
	}
	contentTypeSession, err = contentTypeSession.Revise(contentTypeInput, "")
	if err != nil {
		t.Fatal(err)
	}
	_, contentTypeDecision, err := contentTypeSession.Finalize(
		"local-test-operator", "Preserve exact ordered content-type members.", []domain.ReceiptReference{},
	)
	if err != nil || !contentTypeDecision.EarlyReveal() ||
		!slices.Equal(contentTypeDecision.SelectedFields(), []string{string(counterhttp.HTTPFieldContentType)}) ||
		!slices.Equal(contentTypeDecision.NonassertedFields(), []string{
			string(counterhttp.HTTPFieldStatus), string(counterhttp.HTTPFieldBodyKind),
			string(counterhttp.HTTPFieldBodyMetadata),
		}) {
		t.Fatalf("portable HTTP ordered-list DecisionRecord lost its selected-only partition: %v", err)
	}
	contentTypeCompiled, ok := contentTypeDecision.CompilableRuling()
	if !ok {
		t.Fatal("portable HTTP ordered-list DecisionRecord was not compilable")
	}
	contentTypeAllowed := contentTypeCompiled.AllowedTuples()
	if len(contentTypeAllowed) != 1 || len(contentTypeAllowed[0].Fields) != 1 ||
		contentTypeAllowed[0].Fields[0].FieldID != string(counterhttp.HTTPFieldContentType) {
		t.Fatalf("portable HTTP ordered-list DecisionRecord changed tuple shape: %#v", contentTypeAllowed)
	}
	ordered, ok := contentTypeAllowed[0].Fields[0].Value.OrderedStrings()
	if !ok || !slices.Equal(ordered, customContentTypes) {
		t.Fatalf("portable HTTP ordered-list DecisionRecord changed order or duplicates: %#v", ordered)
	}
	parsedContentType, err := choice.ParseDecisionRecord(contentTypeDecision.CanonicalBytes(), readyRecord)
	if err != nil || parsedContentType.Digest() != contentTypeDecision.Digest() {
		t.Fatalf("portable HTTP ordered-list DecisionRecord did not round trip strictly: %v", err)
	}

	status401, err := choice.IntegerValue("401")
	if err != nil {
		t.Fatal(err)
	}
	custom401, err := choice.NewSelectedTuple(
		readyRecord,
		[]string{string(counterhttp.HTTPFieldStatus)},
		[]choice.FieldValue{{FieldID: string(counterhttp.HTTPFieldStatus), Value: status401}},
	)
	if err != nil {
		t.Fatal(err)
	}
	customFields := custom401.Fields()
	if len(customFields) != 1 || customFields[0].FieldID != string(counterhttp.HTTPFieldStatus) ||
		customFields[0].Value.Tag() != choice.ValueInteger || customFields[0].Value.Text() != "401" {
		t.Fatalf("custom HTTP expectation retained unselected context: %#v", customFields)
	}
	customInput := choice.RulingDraftInput{
		Action: choice.ActionCustomExpectation, SelectedFields: []string{string(counterhttp.HTTPFieldStatus)},
		AllowedAliases: []string{}, CustomExpectation: &custom401,
		CustomReviewer: "physical-http-contract-reviewer", CustomReviewEvidence: confirmationDraft.Digest(),
	}
	customSession, err := choice.NewSession(readyRecord)
	if err != nil {
		t.Fatal(err)
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness, choice.SurfaceMinimizedWitness, choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations, choice.SurfaceNonassertedFields,
	} {
		customSession, err = customSession.Visit(surface)
		if err != nil {
			t.Fatal(err)
		}
	}
	customSession, err = customSession.Propose(customInput)
	if err != nil {
		t.Fatal(err)
	}
	customSession, _, err = customSession.Reveal()
	if err != nil {
		t.Fatal(err)
	}
	customSession, err = customSession.Visit(choice.SurfaceProvenance)
	if err != nil {
		t.Fatal(err)
	}
	customSession, err = customSession.Revise(customInput, "")
	if err != nil {
		t.Fatal(err)
	}
	_, customDecision, err := customSession.Finalize(
		"local-test-operator", "Accept only the separately reviewed HTTP 401 status.", []domain.ReceiptReference{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if customDecision.Action() != choice.ActionCustomExpectation ||
		!slices.Equal(customDecision.SelectedFields(), httpFields[:1]) ||
		!slices.Equal(customDecision.NonassertedFields(), httpFields[1:]) ||
		len(customDecision.ConfirmedAllowedOutcomeIDs()) != 0 ||
		len(customDecision.ConfirmedDisallowedOutcomeIDs()) != len(confirmedStudy.OutcomeMap.Entries()) {
		t.Fatalf("custom HTTP DecisionRecord lost its selected-only partition")
	}
	compiled, compilable := customDecision.CompilableRuling()
	if !compilable {
		t.Fatal("custom HTTP 401 decision was not compilable")
	}
	allowed := compiled.AllowedTuples()
	if len(allowed) != 1 || len(allowed[0].Fields) != 1 ||
		allowed[0].Fields[0].FieldID != string(counterhttp.HTTPFieldStatus) ||
		allowed[0].Fields[0].Value.Text() != "401" {
		t.Fatalf("compiled HTTP 401 predicate smuggled profile context: %#v", allowed)
	}
	parsedDecision, err := choice.ParseDecisionRecord(customDecision.CanonicalBytes(), readyRecord)
	if err != nil || parsedDecision.Digest() != customDecision.Digest() {
		t.Fatalf("custom HTTP DecisionRecord did not round trip strictly: %v", err)
	}
	durableRuling, err := promotion.Finalize(context.Background(), restartedStore, reopenedReady, customDecision)
	if err != nil || durableRuling.Record().Digest() != customDecision.Digest() {
		t.Fatalf("physical HTTP DecisionRecord promotion failed: %v", err)
	}
	rulingRestart, err := store.OpenObjectStore(promotionRoot)
	if err != nil {
		t.Fatal(err)
	}
	reopenedRuling, err := promotion.OpenRuling(context.Background(), rulingRestart, studyID)
	if err != nil || reopenedRuling.Record().Digest() != customDecision.Digest() {
		t.Fatalf("physical HTTP RULING did not survive restart: %v", err)
	}
	preparation, err := promotion.PreparePortableRuling(context.Background(), rulingRestart, reopenedRuling)
	if err != nil || !preparation.Valid() || !preparation.ProfileDigest().Valid() ||
		preparation.DecisionDigest() != parsedDecision.Digest() ||
		!slices.Equal(preparation.SelectedFields(), httpFields[:1]) ||
		promotion.ValidatePortableRulingPreparation(context.Background(), rulingRestart, preparation) != nil {
		t.Fatalf("current portable HTTP ruling preparation failed: %#v, %v", preparation, err)
	}
	resolvedProfile, err := projectiontranslate.Resolve(confirmedStudy.ProjectionDefinition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	projectionAuthority, err := httpmodel.ResolveHTTPProjectionAuthority(
		confirmedStudy.ProjectionDefinition.Digest(), confirmedStudy.ProjectionDefinition.CanonicalBytes(),
		confirmedStudy.ProjectionDefinition.Binding(),
	)
	if err != nil {
		t.Fatal(err)
	}
	portableSource, err := contractsource.NewHTTPSource(contractsource.HTTPInput{
		Plan: confirmedStudy.Plan, Stimulus: confirmedStudy.Stimulus, Start: confirmedStudy.StartSpec,
		Capture: confirmedStudy.CapturePolicy, Readiness: confirmedStudy.Readiness,
		Profile: resolvedProfile.Profile(), Projection: projectionAuthority,
	})
	if err != nil {
		t.Fatalf("exact minimized child-bind HTTP source construction failed: %v", err)
	}
	const httpSourceProfileCanonical = `{"adapter_domain":"HTTP","launch_profile":"NODE_REPO_SCRIPT_V1","runtime_family":"NODE","scope":"DECLARED_SOURCE_PROFILE_NOT_EXECUTION_EVIDENCE","semantic_profile":"countershape-node-core-exact/v1","start_profile":"NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1","subject_entrypoint":"fixture/server_child_bind.mjs"}`
	httpSourceProfileDigest, err := canon.DigestBytes("ContractSourceProfile", []byte(httpSourceProfileCanonical))
	if err != nil {
		t.Fatal(err)
	}
	wantCustomTuple := []byte(`{"fields":[{"field_id":"http.status","value":{"canonical":"401","tag":"INTEGER"}}]}`)
	parsedConfirmation, err := confirmation.ParseRecord(confirmationDraft.CanonicalBytes())
	if err != nil || len(parsedConfirmation.ExecutionBindingDigests()) != 12 {
		t.Fatalf("child-bind confirmation binding roster = %d, %v", len(parsedConfirmation.ExecutionBindingDigests()), err)
	}
	for index, binding := range parsedConfirmation.ExecutionBindingDigests() {
		if binding != portableSource.ExecutionBindingDigest() {
			t.Fatalf("child-bind confirmation binding %d differs from PortableSource", index)
		}
	}
	headBeforeCompilation, err := rulingRestart.OpenHead(context.Background(), studyID)
	if err != nil {
		t.Fatal(err)
	}
	inventoryBeforeCompilation := httpA21StoreInventory(t, promotionRoot)
	prepared, err := nodeemit.PrepareCompilation(context.Background(), rulingRestart, preparation, portableSource)
	if err != nil || !prepared.Valid() || !prepared.Digest().Valid() || prepared.Action() != string(choice.ActionCustomExpectation) ||
		prepared.DecisionRecordDigest() != customDecision.Digest() || prepared.SourceDigest() != portableSource.Digest() ||
		prepared.ChoicepointDigest() != choicepoint.Digest() ||
		prepared.SourceProfileDigest().String() != httpSourceProfileDigest.String() ||
		!slices.Equal(prepared.SelectedFields(), httpFields[:1]) ||
		!reflect.DeepEqual(prepared.AllowedTupleCanonicalBytes(), [][]byte{wantCustomTuple}) {
		t.Fatalf("current HTTP custom compilation preparation = %#v, %v", httpA21DescribePrepared(prepared), err)
	}
	headAfterCompilation, err := rulingRestart.OpenHead(context.Background(), studyID)
	if err != nil || headAfterCompilation.HeadDigest() != headBeforeCompilation.HeadDigest() ||
		headAfterCompilation.Stage() != store.StageRuling || headAfterCompilation.CurrentDigest() != customDecision.Digest() ||
		!reflect.DeepEqual(httpA21StoreInventory(t, promotionRoot), inventoryBeforeCompilation) {
		t.Fatalf("successful HTTP compilation preparation changed durable store state: %v", err)
	}
	preparedTuple := prepared.AllowedTupleCanonicalBytes()[0]
	preparedTuple[0] ^= 0xff
	if !prepared.Valid() {
		t.Fatal("HTTP PreparedCompilation tuple getter was not defensive")
	}
	noisySource, err := contractsource.NewHTTPSource(contractsource.HTTPInput{
		Plan: baselineResult.Plan, Stimulus: baselineResult.Stimulus, Start: baselineResult.StartSpec,
		Capture: baselineResult.CapturePolicy, Readiness: baselineResult.Readiness,
		Profile: resolvedProfile.Profile(), Projection: projectionAuthority,
	})
	if err != nil {
		t.Fatal(err)
	}
	parsedNoisySource, err := contractsource.Parse(noisySource.CanonicalBytes())
	if err != nil || !parsedNoisySource.Valid() || parsedNoisySource.Digest() != noisySource.Digest() ||
		!bytes.Equal(parsedNoisySource.CanonicalBytes(), noisySource.CanonicalBytes()) {
		t.Fatalf("noisy HTTP source is not an independently valid exact authority: %v", err)
	}
	if noisySource.Adapter() != portableSource.Adapter() || noisySource.Plan().Digest() != portableSource.Plan().Digest() ||
		!bytes.Equal(noisySource.Plan().CanonicalBytes(), portableSource.Plan().CanonicalBytes()) ||
		noisySource.ProjectionBinding().Digest() != portableSource.ProjectionBinding().Digest() ||
		!bytes.Equal(noisySource.ProjectionBinding().CanonicalBytes(), portableSource.ProjectionBinding().CanonicalBytes()) ||
		noisySource.Profile().Digest() != portableSource.Profile().Digest() ||
		!bytes.Equal(noisySource.Profile().CanonicalBytes(), portableSource.Profile().CanonicalBytes()) ||
		noisySource.StimulusDigest() == portableSource.StimulusDigest() ||
		bytes.Equal(noisySource.StimulusCanonicalBytes(), portableSource.StimulusCanonicalBytes()) ||
		noisySource.ExecutionBindingDigest() == portableSource.ExecutionBindingDigest() ||
		bytes.Equal(noisySource.ExecutionBindingCanonicalBytes(), portableSource.ExecutionBindingCanonicalBytes()) {
		t.Fatal("HTTP cross-study source matrix did not isolate stimulus/execution authority under one valid shape")
	}
	if refused, prepareErr := nodeemit.PrepareCompilation(
		context.Background(), rulingRestart, preparation, noisySource,
	); !nodeemit.IsCode(prepareErr, nodeemit.CodeSourceRulingMismatch) || refused.Valid() ||
		refused.Digest().Valid() || refused.DecisionRecordDigest().Valid() || refused.ChoicepointDigest().Valid() ||
		refused.SourceDigest().Valid() || refused.SourceProfileDigest().Valid() || refused.Action() != "" ||
		len(refused.SelectedFields()) != 0 || len(refused.AllowedTupleCanonicalBytes()) != 0 {
		t.Fatalf("HTTP noisy/original source cross-pair = valid %t, err %v", refused.Valid(), prepareErr)
	}
	reopenedStore, err := store.OpenObjectStore(promotionRoot)
	if err != nil {
		t.Fatal(err)
	}
	if refused, prepareErr := nodeemit.PrepareCompilation(
		context.Background(), reopenedStore, preparation, portableSource,
	); !httpA21StoreAuthorityRefused(prepareErr) || refused.Valid() || refused.Digest().Valid() ||
		refused.DecisionRecordDigest().Valid() || refused.ChoicepointDigest().Valid() || refused.SourceDigest().Valid() ||
		refused.SourceProfileDigest().Valid() || refused.Action() != "" || len(refused.SelectedFields()) != 0 ||
		len(refused.AllowedTupleCanonicalBytes()) != 0 {
		t.Fatalf("HTTP wrong-store preparation = valid %t, err %v", refused.Valid(), prepareErr)
	}
	reissuedRuling, err := promotion.OpenRuling(context.Background(), reopenedStore, studyID)
	if err != nil {
		t.Fatal(err)
	}
	reissued, err := promotion.PreparePortableRuling(context.Background(), reopenedStore, reissuedRuling)
	if err != nil {
		t.Fatal(err)
	}
	reparsedSource, err := contractsource.Parse(portableSource.CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	restartedPrepared, err := nodeemit.PrepareCompilation(
		context.Background(), reopenedStore, reissued, reparsedSource,
	)
	if err != nil || restartedPrepared.Digest() != prepared.Digest() ||
		!reflect.DeepEqual(restartedPrepared.AllowedTupleCanonicalBytes(), prepared.AllowedTupleCanonicalBytes()) {
		t.Fatalf("HTTP restart/reopen changed sanitized input: %#v, %v", httpA21DescribePrepared(restartedPrepared), err)
	}
	assertHTTPCompilationFreshProcessRestart(t, promotionRoot, portableSource, prepared)
	if predecessorErr := promotion.ValidatePortableRulingPreparation(
		context.Background(), rulingRestart, reissued,
	); !httpA21StoreAuthorityRefused(predecessorErr) {
		t.Fatalf("HTTP reissued preparation crossed predecessor authority: %v", predecessorErr)
	}
	if refused, prepareErr := nodeemit.PrepareCompilation(
		context.Background(), rulingRestart, reissued, portableSource,
	); !httpA21StoreAuthorityRefused(prepareErr) || refused.Valid() || refused.Digest().Valid() ||
		refused.DecisionRecordDigest().Valid() || refused.ChoicepointDigest().Valid() || refused.SourceDigest().Valid() ||
		refused.SourceProfileDigest().Valid() || refused.Action() != "" || len(refused.SelectedFields()) != 0 ||
		len(refused.AllowedTupleCanonicalBytes()) != 0 {
		t.Fatalf("HTTP reissued preparation crossed predecessor compilation authority: %#v, %v", httpA21DescribePrepared(refused), prepareErr)
	}
	currentAfterRestartMatrix, err := reopenedStore.OpenHead(context.Background(), studyID)
	if err != nil || currentAfterRestartMatrix.HeadDigest() != headBeforeCompilation.HeadDigest() ||
		currentAfterRestartMatrix.Stage() != store.StageRuling || currentAfterRestartMatrix.CurrentDigest() != customDecision.Digest() ||
		!reflect.DeepEqual(httpA21StoreInventory(t, promotionRoot), inventoryBeforeCompilation) {
		t.Fatalf("HTTP A2.1 restart/refusal matrix changed durable store state: %v", err)
	}

	// A second durable lineage proves the same child-bind confirmation can
	// authorize an observed HTTP predicate without reusing the custom ruling.
	allowRoot := filepath.Join(httpA21ResolvedStoreTempDir(t), "allow-observed-store")
	allowStore, err := store.OpenObjectStore(allowRoot)
	if err != nil {
		t.Fatal(err)
	}
	allowStudyID, err := store.NewStudyID("physical HTTP observed compilation authority")
	if err != nil {
		t.Fatal(err)
	}
	allowHead, err := allowStore.CreateStudy(context.Background(), allowStudyID, confirmedStudy.Plan)
	if err != nil {
		t.Fatal(err)
	}
	allowHead, err = allowStore.AdvanceBaseline(context.Background(), allowHead, baselineCheckpoint.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	allowHead, err = allowStore.AdvanceDivergence(context.Background(), allowHead, baseline)
	if err != nil {
		t.Fatal(err)
	}
	allowHead, err = allowStore.AdvanceReduction(context.Background(), allowHead, run)
	if err != nil {
		t.Fatal(err)
	}
	allowConfirmation, err := promotion.PersistConfirmation(context.Background(), allowStore, allowHead, confirmationDraft)
	if err != nil {
		t.Fatal(err)
	}
	allowReady, err := promotion.Promote(context.Background(), allowStore, allowConfirmation, promotion.ChoicepointRequest{
		Scenario: "Which exact invoice response behavior should become the accepted contract?",
		Plan:     confirmedStudy.Plan, Envelope: confirmedStudy.Envelope,
		CandidateBindings: confirmedStudy.CandidateBindings, OriginalStimulus: originalArtifact,
		MinimizedStimulus: minimizedArtifact, CandidateReveals: reveals,
		EvidenceReceipts: []domain.ReceiptReference{},
	})
	if err != nil {
		t.Fatal(err)
	}
	allowSession, err := choice.NewSession(allowReady.Record())
	if err != nil {
		t.Fatal(err)
	}
	allowCards := allowSession.BlindDTO().Cards()
	if len(allowCards) == 0 {
		t.Fatal("HTTP allow-observed Choicepoint has no cards")
	}
	allowInput := choice.RulingDraftInput{
		Action: choice.ActionAllowObserved, SelectedFields: []string{
			string(counterhttp.HTTPFieldStatus), string(counterhttp.HTTPFieldBodyKind),
		},
		AllowedAliases: []string{allowCards[0].Alias},
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness, choice.SurfaceMinimizedWitness, choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations, choice.SurfaceNonassertedFields,
	} {
		allowSession, err = allowSession.Visit(surface)
		if err != nil {
			t.Fatal(err)
		}
	}
	allowSession, err = allowSession.Propose(allowInput)
	if err != nil {
		t.Fatal(err)
	}
	allowSession, _, err = allowSession.Reveal()
	if err != nil {
		t.Fatal(err)
	}
	allowSession, err = allowSession.Visit(choice.SurfaceProvenance)
	if err != nil {
		t.Fatal(err)
	}
	allowSession, err = allowSession.Revise(allowInput, "")
	if err != nil {
		t.Fatal(err)
	}
	_, allowDecision, err := allowSession.Finalize(
		"local-test-operator", "Accept one exact correlated HTTP status/body-kind tuple.", []domain.ReceiptReference{},
	)
	if err != nil {
		t.Fatal(err)
	}
	allowRuling, err := promotion.Finalize(context.Background(), allowStore, allowReady, allowDecision)
	if err != nil {
		t.Fatal(err)
	}
	allowPreparation, err := promotion.PreparePortableRuling(context.Background(), allowStore, allowRuling)
	if err != nil {
		t.Fatal(err)
	}
	allowPrepared, err := nodeemit.PrepareCompilation(
		context.Background(), allowStore, allowPreparation, portableSource,
	)
	selectedCard := allowCards[0]
	if len(selectedCard.Fields) != len(httpFields) || selectedCard.Fields[0].Tag != string(choice.ValueInteger) ||
		selectedCard.Fields[2].Tag != string(choice.ValueString) {
		t.Fatalf("selected HTTP observed card lacks exact status/body-kind values: %#v", selectedCard)
	}
	wantAllowTuple := []byte(fmt.Sprintf(
		`{"fields":[{"field_id":"http.status","value":{"canonical":%q,"tag":"INTEGER"}},{"field_id":"http.body.kind","value":{"tag":"STRING","value":%q}}]}`,
		selectedCard.Fields[0].Text, selectedCard.Fields[2].Text,
	))
	if err != nil || !allowPrepared.Valid() || !allowPrepared.Digest().Valid() ||
		allowPrepared.DecisionRecordDigest() != allowDecision.Digest() ||
		allowPrepared.ChoicepointDigest() != allowReady.Record().Digest() ||
		allowPrepared.SourceDigest() != portableSource.Digest() ||
		allowPrepared.SourceProfileDigest().String() != httpSourceProfileDigest.String() ||
		allowPrepared.Action() != string(choice.ActionAllowObserved) ||
		!slices.Equal(allowPrepared.SelectedFields(), []string{
			string(counterhttp.HTTPFieldStatus), string(counterhttp.HTTPFieldBodyKind),
		}) || !reflect.DeepEqual(allowPrepared.AllowedTupleCanonicalBytes(), [][]byte{wantAllowTuple}) {
		t.Fatalf("child-bind HTTP allow-observed compilation preparation = %#v, %v", httpA21DescribePrepared(allowPrepared), err)
	}
	exerciseHTTPP07BBPublication(
		t, promotionRoot, reopenedStore, rulingRestart, studyID, restartedPrepared,
		allowRoot, allowStore, allowStudyID, allowPrepared,
	)
}

func httpPhysicalReductionConfig(t *testing.T) Config {
	t.Helper()
	config := referenceConfig(t)
	config.PortableStart = true
	config.ReductionProposalLimit = 2
	// The plan reserves 24 trials for discovery+confirmation and 12 for U5.
	config.ReductionTotalCandidateTrials = 36
	config.ReductionWallMS = (3 * time.Minute).Milliseconds()
	return config
}

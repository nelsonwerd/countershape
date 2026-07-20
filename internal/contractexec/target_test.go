//go:build darwin && arm64 && cgo

package contractexec

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"go/ast"
	"go/build"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"

	countercli "github.com/nelsonwerd/countershape/internal/adapters/cli"
	climodel "github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/choice/promotion"
	"github.com/nelsonwerd/countershape/internal/contractexec/model"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	node "github.com/nelsonwerd/countershape/internal/emit/node"
	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
	"github.com/nelsonwerd/countershape/internal/store"
	"github.com/nelsonwerd/countershape/testkit/clifixture"
	"github.com/nelsonwerd/countershape/testkit/gitrepo"
)

const c3SystemGit = "/usr/bin/git"
const c3SystemNode = "/opt/homebrew/bin/node"

type c3HeadWire struct {
	SchemaVersion        string `json:"schema_version"`
	Kind                 string `json:"kind"`
	StudyID              string `json:"study_id"`
	Revision             int64  `json:"revision"`
	Stage                string `json:"stage"`
	CurrentKind          string `json:"current_kind"`
	CurrentDigest        string `json:"current_digest"`
	LineageRootDigest    string `json:"lineage_root_digest"`
	PreviousHeadDigest   string `json:"previous_head_digest"`
	PreviousObjectDigest string `json:"previous_object_digest"`
}

type c3ChoicepointRevealWire struct {
	CandidateExecutionKey string `json:"candidate_execution_key"`
	DisplayRef            string `json:"display_ref"`
	ProducerMetadata      string `json:"producer_metadata"`
}

type c3ChoicepointWire struct {
	ComparisonEnvelopeBase64 string                    `json:"comparison_envelope_base64"`
	CandidateBindingsBase64  []string                  `json:"candidate_bindings_base64"`
	CandidateReveals         []c3ChoicepointRevealWire `json:"candidate_reveals"`
	OriginalStimulusKind     string                    `json:"original_stimulus_kind"`
	OriginalStimulusBase64   string                    `json:"original_stimulus_base64"`
	OriginalStimulusDigest   string                    `json:"original_stimulus_digest"`
	EvidenceReceipts         []domain.ReceiptWire      `json:"evidence_receipts"`
}

type c3ResidueFixture struct {
	root        string
	objectStore *store.ObjectStore
	study       store.StudyID
	residue     node.Residue
}

type c3GitFixture struct {
	fixture    gitrepo.Repository
	repository gitobj.Repository
	ref        string
}

func c3RepoRoot(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("source location unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func c3Example(t testing.TB, name string, canonical bool) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(c3RepoRoot(t), "spec", "examples", "v1", name))
	if err != nil {
		t.Fatal(err)
	}
	if canonical {
		body, err = canon.Canonicalize(body)
		if err != nil {
			t.Fatal(err)
		}
	} else if len(body) > 0 && body[len(body)-1] == '\n' {
		body = append([]byte(nil), body[:len(body)-1]...)
	}
	return body
}

func c3PublishObject(t testing.TB, objectStore *store.ObjectStore, kind string, digest domain.Digest, exact []byte) {
	t.Helper()
	object, err := store.NewSemanticObject(kind, digest, exact)
	if err != nil {
		t.Fatal(err)
	}
	authority, err := objectStore.Publish(context.Background(), object)
	if err != nil || objectStore.Validate(context.Background(), object, authority) != nil {
		t.Fatalf("publish %s: %v", kind, err)
	}
}

func c3SyncDirectory(t testing.TB, path string) {
	t.Helper()
	handle, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := handle.Sync(); err != nil {
		_ = handle.Close()
		t.Fatal(err)
	}
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}
}

func c3SourceForChoicepoint(t testing.TB, choicepoint choice.ChoicepointRecord) contractsource.PortableSource {
	t.Helper()
	projection, err := countercli.NewCLIProjectionDefinition(countercli.CLIProjectionDefinitionConfig{
		Fields: []countercli.CLIFieldID{
			countercli.CLIFieldStdoutJSONMode,
			countercli.CLIFieldStdoutJSONSource,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	capture, err := countercli.NewCLICapturePolicy(countercli.CLICapturePolicyConfig{
		StdoutBytes: 64 << 10,
		StderrBytes: 64 << 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	environment, err := countercli.PresentEnvironment("APP_MODE", "env")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := countercli.NewFixtureFile(
		"config.json", clifixture.ConfigJSON("config"), countercli.FixtureMode0644,
	)
	if err != nil {
		t.Fatal(err)
	}
	stimulus, err := countercli.NewCLIStimulus(countercli.CLIStimulusConfig{
		Executable: "node",
		BaseArgv:   []string{clifixture.Entrypoint},
		Argv:       []string{"--mode", "argv"},
		Stdin:      countercli.AbsentStdin(),
		Environment: []countercli.CLIEnvironmentBinding{
			environment,
		},
		Fixtures:  []countercli.CLIFixtureFile{fixture},
		CWDPolicy: countercli.CWDMaterializedRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	plan := choicepoint.WorldPlan()
	if plan.CapturePolicyDigest() != capture.Digest() ||
		plan.ProjectionDefinitionDigest() != projection.Binding().Digest() ||
		plan.ProjectionDefinitionBinding().Digest() != projection.Binding().Digest() {
		t.Fatalf(
			"sealed choicepoint authorities differ: capture=%s/%s projection=%s/%s binding=%s/%s",
			plan.CapturePolicyDigest(), capture.Digest(),
			plan.ProjectionDefinitionDigest(), projection.Binding().Digest(),
			plan.ProjectionDefinitionBinding().Digest(), projection.Binding().Digest(),
		)
	}
	resolved, err := projectiontranslate.Resolve(projection.Binding())
	if err != nil {
		t.Fatal(err)
	}
	projectionAuthority, err := climodel.ResolveCLIProjectionAuthority(
		projection.Digest(), projection.CanonicalBytes(), projection.Binding(),
	)
	if err != nil {
		t.Fatal(err)
	}
	source, err := contractsource.NewCLISource(contractsource.CLIInput{
		Plan: plan, Stimulus: stimulus, Capture: capture,
		Profile: resolved.Profile(), Projection: projectionAuthority,
	})
	if err != nil {
		t.Fatal(err)
	}
	minimized := choicepoint.MinimizedStimulus()
	if source.Plan().Digest() != plan.Digest() || source.StimulusDigest() != minimized.Digest() ||
		source.Profile().Digest() != choicepoint.PortableProfile().Digest() {
		t.Fatalf(
			"public CLI source reconstruction differs: plan=%s/%s stimulus=%s/%s profile=%s/%s",
			source.Plan().Digest(), plan.Digest(), source.StimulusDigest(), minimized.Digest(),
			source.Profile().Digest(), choicepoint.PortableProfile().Digest(),
		)
	}
	return source
}

func c3DecodeBase64(t testing.TB, encoded string) []byte {
	t.Helper()
	body, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || base64.StdEncoding.EncodeToString(body) != encoded {
		t.Fatalf("decode exact choicepoint base64: %v", err)
	}
	return body
}

func c3PortableChoicepoint(t testing.TB, legacy choice.ChoicepointRecord) choice.ChoicepointRecord {
	t.Helper()
	var wire c3ChoicepointWire
	if err := json.Unmarshal(legacy.CanonicalBytes(), &wire); err != nil {
		t.Fatal(err)
	}
	envelope, err := domain.ParseComparisonEnvelope(c3DecodeBase64(t, wire.ComparisonEnvelopeBase64))
	if err != nil {
		t.Fatal(err)
	}
	bindings := make([]domain.CandidateExecutionBinding, len(wire.CandidateBindingsBase64))
	for index, encoded := range wire.CandidateBindingsBase64 {
		bindings[index], err = domain.ParseCandidateExecutionBinding(c3DecodeBase64(t, encoded))
		if err != nil {
			t.Fatal(err)
		}
	}
	reveals := make([]choice.CandidateReveal, len(wire.CandidateReveals))
	for index, reveal := range wire.CandidateReveals {
		key, parseErr := domain.ParseCandidateExecutionKey(reveal.CandidateExecutionKey)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		reveals[index] = choice.CandidateReveal{
			CandidateExecutionKey: key,
			DisplayRef:            reveal.DisplayRef,
			ProducerMetadata:      reveal.ProducerMetadata,
		}
	}
	originalDigest, err := domain.ParseDigest(wire.OriginalStimulusDigest)
	if err != nil {
		t.Fatal(err)
	}
	original, err := choice.NewCanonicalArtifact(
		wire.OriginalStimulusKind, originalDigest, c3DecodeBase64(t, wire.OriginalStimulusBase64),
	)
	if err != nil {
		t.Fatal(err)
	}
	receipts := make([]domain.ReceiptReference, len(wire.EvidenceReceipts))
	for index, receipt := range wire.EvidenceReceipts {
		receipts[index], err = domain.ReceiptFromWire(receipt)
		if err != nil {
			t.Fatal(err)
		}
	}
	portable, err := choice.NewChoicepointRecord(choice.ChoicepointInput{
		Scenario: legacy.Scenario(), Plan: legacy.WorldPlan(), Envelope: envelope,
		CandidateBindings: bindings, OriginalStimulus: original,
		MinimizedStimulus: legacy.MinimizedStimulus(), Confirmation: legacy.ConfirmationRecord(),
		CandidateReveals: reveals, EvidenceReceipts: receipts,
	})
	if err != nil || !portable.Valid() || !portable.PortableProfile().Valid() ||
		portable.Digest() == legacy.Digest() {
		t.Fatalf("reconstruct portable choicepoint from exact historical evidence: %v", err)
	}
	return portable
}

func c3DecisionForChoicepoint(t testing.TB, choicepoint choice.ChoicepointRecord) choice.DecisionRecord {
	t.Helper()
	session, err := choice.NewSession(choicepoint)
	if err != nil {
		t.Fatal(err)
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness,
		choice.SurfaceMinimizedWitness,
		choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations,
		choice.SurfaceNonassertedFields,
	} {
		session, err = session.Visit(surface)
		if err != nil {
			t.Fatal(err)
		}
	}
	cards := session.BlindDTO().Cards()
	selected := session.BlindDTO().SelectableFields()
	if len(cards) == 0 || len(selected) == 0 {
		t.Fatal("portable choicepoint has no observed card or selectable field")
	}
	draft := choice.RulingDraftInput{
		Action:         choice.ActionAllowObserved,
		SelectedFields: selected,
		AllowedAliases: []string{cards[0].Alias},
	}
	session, err = session.Propose(draft)
	if err != nil {
		t.Fatal(err)
	}
	session, _, err = session.Reveal()
	if err != nil {
		t.Fatal(err)
	}
	session, err = session.Visit(choice.SurfaceProvenance)
	if err != nil {
		t.Fatal(err)
	}
	session, err = session.Revise(draft, "")
	if err != nil {
		t.Fatal(err)
	}
	_, decision, err := session.Finalize(
		"countershape-c3-test", "exact public portable ruling fixture", []domain.ReceiptReference{},
	)
	if err != nil || !decision.Valid() {
		t.Fatalf("finalize portable decision: %v", err)
	}
	return decision
}

func c3NewResidue(t testing.TB, label string) c3ResidueFixture {
	t.Helper()
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, "store")
	objectStore, err := store.OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	return c3NewResidueInStore(t, root, objectStore, label)
}

func c3NewResidueInStore(t testing.TB, root string, objectStore *store.ObjectStore, label string) c3ResidueFixture {
	t.Helper()
	choiceBytes := c3Example(t, "choicepoint.valid.json", false)
	choicepoint, err := choice.ParseChoicepointRecord(choiceBytes)
	if err != nil {
		t.Fatalf("parse choicepoint example: %v", err)
	}
	choicepoint = c3PortableChoicepoint(t, choicepoint)
	choiceBytes = choicepoint.CanonicalBytes()
	decision := c3DecisionForChoicepoint(t, choicepoint)
	decisionBytes := decision.CanonicalBytes()
	source := c3SourceForChoicepoint(t, choicepoint)
	if source.Plan().Digest() != choicepoint.WorldPlan().Digest() {
		t.Fatal("CLI source and sealed choicepoint do not form one ruling predecessor graph")
	}
	confirmation := choicepoint.ConfirmationRecord()
	for _, item := range []struct {
		kind   string
		digest domain.Digest
		exact  []byte
	}{
		{"WorldPlan", source.Plan().Digest(), source.Plan().CanonicalBytes()},
		{"FreshConfirmation", confirmation.Digest(), confirmation.CanonicalBytes()},
		{"Choicepoint", choicepoint.Digest(), choiceBytes},
		{"DecisionRecord", decision.Digest(), decisionBytes},
	} {
		c3PublishObject(t, objectStore, item.kind, item.digest, item.exact)
	}
	study, err := store.NewStudyID("C3 official target " + label)
	if err != nil {
		t.Fatal(err)
	}
	previous := domain.MustDigest("sha256:" + strings.Repeat("a", 64))
	headBytes, err := canon.CanonicalizeTyped(c3HeadWire{
		SchemaVersion: domain.SchemaVersion, Kind: "StudyHead", StudyID: study.String(), Revision: 7,
		Stage: string(store.StageRuling), CurrentKind: "DecisionRecord", CurrentDigest: decision.Digest().String(),
		LineageRootDigest: source.Plan().Digest().String(), PreviousHeadDigest: previous.String(),
		PreviousObjectDigest: choicepoint.Digest().String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	studyPath := filepath.Join(root, "studies", strings.TrimPrefix(study.Digest().String(), "sha256:"))
	if err := os.Mkdir(studyPath, 0o700); err != nil {
		t.Fatal(err)
	}
	lock, err := os.OpenFile(filepath.Join(studyPath, ".head.lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.Sync(); err != nil {
		_ = lock.Close()
		t.Fatal(err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	headPath := filepath.Join(studyPath, "head.json")
	head, err := os.OpenFile(headPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := head.Write(headBytes); err != nil {
		_ = head.Close()
		t.Fatal(err)
	}
	if err := head.Sync(); err != nil {
		_ = head.Close()
		t.Fatal(err)
	}
	if err := head.Close(); err != nil {
		t.Fatal(err)
	}
	c3SyncDirectory(t, studyPath)
	c3SyncDirectory(t, filepath.Dir(studyPath))
	ctx := context.Background()
	ruling, err := promotion.OpenRuling(ctx, objectStore, study)
	if err != nil {
		t.Fatalf("open exact ruling: %v", err)
	}
	preparation, err := promotion.PreparePortableRuling(ctx, objectStore, ruling)
	if err != nil {
		t.Fatalf("prepare portable ruling: %v", err)
	}
	prepared, err := node.PrepareCompilation(ctx, objectStore, preparation, source)
	if err != nil {
		t.Fatalf("prepare exact compiler input: %v", err)
	}
	compiled, err := node.CompilePrepared(prepared)
	if err != nil {
		t.Fatalf("compile exact contract bundle: %v", err)
	}
	publication, err := node.PublishPrepared(ctx, objectStore, compiled)
	if err != nil || !publication.Residue.Valid() {
		t.Fatalf("publish exact terminal residue: %v", err)
	}
	return c3ResidueFixture{root: root, objectStore: objectStore, study: study, residue: publication.Residue}
}

func c3NewGitWithFiles(t testing.TB, files []gitrepo.File, label string) c3GitFixture {
	t.Helper()
	fixture, err := gitrepo.Init(context.Background(), c3SystemGit, t.TempDir(), gitrepo.SHA1)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := fixture.CommitFiles(context.Background(), files, "", label)
	if err != nil {
		t.Fatal(err)
	}
	ref := "refs/heads/target"
	if err := fixture.UpdateRef(context.Background(), ref, commit); err != nil {
		t.Fatal(err)
	}
	repository, err := gitobj.OpenRepository(context.Background(), gitobj.OpenConfig{
		GitExecutable: c3SystemGit, Repository: fixture.Root, ScratchRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	return c3GitFixture{fixture: fixture, repository: repository, ref: ref}
}

func c3NewGit(t testing.TB) c3GitFixture {
	t.Helper()
	return c3NewGitWithFiles(t, []gitrepo.File{
		{Path: clifixture.Entrypoint, Mode: "100755", Content: clifixture.Program()},
		{Path: "bin/helper", Mode: "100755", Content: []byte("#!/bin/sh\nexit 0\n")},
	}, "C3 runnable target")
}

func c3Publish(t testing.TB, residue c3ResidueFixture, git c3GitFixture) OfficialTarget {
	return c3PublishWithNode(t, residue, git, c3SystemNode)
}

func c3PublishWithNode(t testing.TB, residue c3ResidueFixture, git c3GitFixture, executable string) OfficialTarget {
	t.Helper()
	if _, err := os.Lstat(executable); err != nil {
		t.Skipf("exact Node unavailable: %v", err)
	}
	target, err := PublishOfficialTarget(context.Background(), PublishOfficialTargetRequest{
		Store: residue.objectStore, Residue: residue.residue, Repository: git.repository,
		DisplayRef: git.ref, NodeExecutable: executable,
	})
	if err != nil || !target.Valid() {
		t.Fatalf("publish OfficialTarget: %v", err)
	}
	return target
}

func c3MutableNode(t testing.TB) string {
	t.Helper()
	realNode, err := filepath.EvalSymlinks(c3SystemNode)
	if err != nil {
		t.Skipf("exact Node unavailable: %v", err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "bin")
	lib := filepath.Join(root, "lib")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(lib, 0o700); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(realNode)
	if err != nil {
		t.Fatal(err)
	}
	destinationPath := filepath.Join(bin, "node")
	destination, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		_ = source.Close()
		t.Fatal(err)
	}
	_, copyErr := io.Copy(destination, source)
	sourceCloseErr := source.Close()
	syncErr := destination.Sync()
	destinationCloseErr := destination.Close()
	if copyErr != nil || sourceCloseErr != nil || syncErr != nil || destinationCloseErr != nil {
		t.Fatalf("copy exact Node executable: copy=%v source-close=%v sync=%v destination-close=%v", copyErr, sourceCloseErr, syncErr, destinationCloseErr)
	}
	linkedLibraries, err := filepath.Glob(filepath.Join(filepath.Dir(filepath.Dir(realNode)), "lib", "libnode.*.dylib"))
	if err != nil || len(linkedLibraries) != 1 {
		t.Fatalf("exact Node dynamic-library fixture differs: paths=%v err=%v", linkedLibraries, err)
	}
	if err := os.Symlink(linkedLibraries[0], filepath.Join(lib, filepath.Base(linkedLibraries[0]))); err != nil {
		t.Fatal(err)
	}
	c3SyncDirectory(t, bin)
	c3SyncDirectory(t, lib)
	c3SyncDirectory(t, root)
	return destinationPath
}

func c3HeadSnapshot(t testing.TB, fixture c3ResidueFixture) (os.FileInfo, []byte) {
	t.Helper()
	path := filepath.Join(fixture.root, "studies", strings.TrimPrefix(fixture.study.Digest().String(), "sha256:"), "head.json")
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return info, body
}

func c3DirectoryEntryCount(t testing.TB, path string) int {
	t.Helper()
	entries, err := os.ReadDir(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

type c3ExactTreeFact struct {
	Path          string
	Mode          os.FileMode
	Size          int64
	Device        uint64
	Inode         uint64
	Links         uint64
	UID           uint64
	GID           uint64
	ContentSHA256 [sha256.Size]byte
}

func c3ExactTreeSnapshot(t testing.TB, root string) []c3ExactTreeFact {
	t.Helper()
	const maxEntries = 20_000
	const maxRegularBytes = 64 << 20
	var facts []c3ExactTreeFact
	var regularBytes int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if len(facts) >= maxEntries {
			return errors.New("exact store-tree snapshot entry bound exceeded")
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return errors.New("exact store-tree snapshot encountered a non-regular entry")
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return errors.New("exact store-tree snapshot lacks Darwin stat identity")
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
			return errors.New("exact store-tree snapshot path escaped its root")
		}
		fact := c3ExactTreeFact{
			Path: filepath.ToSlash(relativePath), Mode: info.Mode(), Device: uint64(stat.Dev), Inode: uint64(stat.Ino),
			Links: uint64(stat.Nlink), UID: uint64(stat.Uid), GID: uint64(stat.Gid),
		}
		if info.Mode().IsRegular() {
			fact.Size = info.Size()
			regularBytes += info.Size()
			if regularBytes > maxRegularBytes {
				return errors.New("exact store-tree snapshot byte bound exceeded")
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			after, err := os.Lstat(path)
			if err != nil || !os.SameFile(info, after) || after.Mode() != info.Mode() || after.Size() != info.Size() {
				return errors.New("exact store-tree snapshot observed a changing regular file")
			}
			fact.ContentSHA256 = sha256.Sum256(body)
		}
		facts = append(facts, fact)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(facts, func(left, right int) bool { return facts[left].Path < facts[right].Path })
	return facts
}

func TestC3PublishOfficialTargetJoinsExactLivePrerequisiteGraph(t *testing.T) {
	residue := c3NewResidue(t, "exact")
	git := c3NewGit(t)
	beforeInfo, beforeBytes := c3HeadSnapshot(t, residue)
	target := c3Publish(t, residue, git)
	afterInfo, afterBytes := c3HeadSnapshot(t, residue)
	if !os.SameFile(beforeInfo, afterInfo) || !bytes.Equal(beforeBytes, afterBytes) {
		t.Fatal("OfficialTarget publication mutated terminal study head inode or bytes")
	}
	if !target.Digest().Valid() || !target.Model().Valid() || !target.TargetRecord().Valid() ||
		!target.ContractBundle().Valid() || target.CandidateRoot() == "" || target.Roots().CandidateParent() == "" {
		t.Fatal("OfficialTarget defensive getters are incomplete")
	}
	modelInput := target.Model().Input()
	plan := residue.residue.Bundle().PortableSource().Plan()
	if modelInput.ContractBundleDigest != residue.residue.BundleDigest() ||
		modelInput.Tree.MaterializationPolicyDigest != plan.MaterializationPolicyDigest() ||
		target.ContractBundle().SourceProfile().SubjectEntrypoint() != clifixture.Entrypoint ||
		modelInput.Attempt.ArtifactDigest != target.TargetRecord().AttemptDigest() {
		t.Fatal("OfficialTarget model does not join its live prerequisites")
	}
	alternateDigest := domain.MustDigest("sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	for name, mutate := range map[string]func(*model.TreeBinding){
		"object format":          func(binding *model.TreeBinding) { binding.ObjectFormat = "sha256" },
		"commit object":          func(binding *model.TreeBinding) { binding.CommitOID = strings.Repeat("c", 40) },
		"tree object":            func(binding *model.TreeBinding) { binding.TreeOID = strings.Repeat("d", 40) },
		"tree identity":          func(binding *model.TreeBinding) { binding.TreeIdentityDigest = alternateDigest },
		"portable tree":          func(binding *model.TreeBinding) { binding.PortableTreeDigest = alternateDigest },
		"materialization policy": func(binding *model.TreeBinding) { binding.MaterializationPolicyDigest = alternateDigest },
		"materialization manifest": func(binding *model.TreeBinding) {
			binding.MaterializationManifestDigest = alternateDigest
		},
	} {
		t.Run("tree binding "+name, func(t *testing.T) {
			changed := modelInput.Tree
			mutate(&changed)
			if treeBindingMatches(
				changed, target.state.repository, target.state.source, target.state.receipt,
				target.state.attempt.Roots().CandidateParent(),
			) {
				t.Fatalf("changed %s retained exact tree authority", name)
			}
		})
	}
	if treeBindingMatches(
		modelInput.Tree, target.state.repository, target.state.source, target.state.receipt,
		target.state.attempt.Roots().AttemptRoot(),
	) {
		t.Fatal("wrong candidate parent retained exact tree authority")
	}
	equivalentRepository := c3NewGit(t).repository
	if treeBindingMatches(
		modelInput.Tree, equivalentRepository, target.state.source, target.state.receipt,
		target.state.attempt.Roots().CandidateParent(),
	) {
		t.Fatal("different live repository handle reused another repository's materialization receipt")
	}
}

func TestC3OpenOfficialTargetRebuildsFullAuthorityAcrossRestart(t *testing.T) {
	residue := c3NewResidue(t, "restart")
	git := c3NewGit(t)
	target := c3Publish(t, residue, git)
	targetDigest := target.Digest()
	retainedCopy := target
	if err := target.Close(); err != nil || retainedCopy.Valid() {
		t.Fatalf("original target capability did not close before restart: %v", err)
	}
	if err := git.repository.Close(); err != nil {
		t.Fatalf("original repository capability did not close before restart: %v", err)
	}
	target = OfficialTarget{}
	git.repository = gitobj.Repository{}
	restartedStore, err := store.OpenObjectStore(residue.root)
	if err != nil {
		t.Fatal(err)
	}
	restartedResidue, err := node.OpenResidue(context.Background(), restartedStore, residue.study)
	if err != nil {
		t.Fatal(err)
	}
	restartedRepository, err := gitobj.OpenRepository(context.Background(), gitobj.OpenConfig{
		GitExecutable: c3SystemGit, Repository: git.fixture.Root, ScratchRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenOfficialTarget(context.Background(), OpenOfficialTargetRequest{
		Store: restartedStore, Residue: restartedResidue, Repository: restartedRepository,
		TargetDigest: targetDigest,
	})
	if err != nil || !reopened.Valid() || reopened.Digest() != targetDigest {
		t.Fatalf("full restart reopen failed: %v", err)
	}
	if err := reopened.Close(); err != nil || reopened.Valid() {
		t.Fatalf("restarted in-memory capability did not close: %v", err)
	}
	if err := restartedRepository.Close(); err != nil {
		t.Fatalf("restarted repository did not close: %v", err)
	}
	secondStore, err := store.OpenObjectStore(residue.root)
	if err != nil {
		t.Fatal(err)
	}
	secondResidue, err := node.OpenResidue(context.Background(), secondStore, residue.study)
	if err != nil {
		t.Fatal(err)
	}
	secondRepository, err := gitobj.OpenRepository(context.Background(), gitobj.OpenConfig{
		GitExecutable: c3SystemGit, Repository: git.fixture.Root, ScratchRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer secondRepository.Close()
	durable, err := OpenOfficialTarget(context.Background(), OpenOfficialTargetRequest{
		Store: secondStore, Residue: secondResidue, Repository: secondRepository, TargetDigest: targetDigest,
	})
	if err != nil || !durable.Valid() || durable.Digest() != targetDigest {
		t.Fatalf("durable locator did not rebuild authority after capability closure: %v", err)
	}
}

func TestC3OpenOfficialTargetLiveParentMatrix(t *testing.T) {
	primary := c3NewResidue(t, "open-live-parent-primary")
	primaryGit := c3NewGit(t)
	primaryTarget := c3Publish(t, primary, primaryGit)
	secondStudy := c3NewResidueInStore(t, primary.root, primary.objectStore, "open-live-parent-second-study")
	secondGit := c3NewGit(t)
	secondTarget := c3Publish(t, secondStudy, secondGit)
	foreign := c3NewResidue(t, "open-live-parent-foreign-store")

	snapshot := func(t *testing.T) map[string][]c3ExactTreeFact {
		t.Helper()
		return map[string][]c3ExactTreeFact{
			"primary-store": c3ExactTreeSnapshot(t, primary.root),
			"foreign-store": c3ExactTreeSnapshot(t, foreign.root),
		}
	}
	assertNoMutation := func(t *testing.T, before map[string][]c3ExactTreeFact) {
		t.Helper()
		after := snapshot(t)
		if !reflect.DeepEqual(after, before) {
			t.Fatalf("target reopen changed an exact store tree: before=%v after=%v", before, after)
		}
	}

	t.Run("independent exact-content object database is portable", func(t *testing.T) {
		equivalent := c3NewGit(t)
		originalPinned, err := primaryGit.repository.Pin(context.Background(), primaryTarget.Model().Input().Tree.CommitOID)
		if err != nil {
			t.Fatal(err)
		}
		equivalentPinned, err := equivalent.repository.Pin(context.Background(), primaryTarget.Model().Input().Tree.CommitOID)
		if err != nil {
			t.Fatal(err)
		}
		if originalPinned.IdentityDigest() != equivalentPinned.IdentityDigest() ||
			originalPinned.Provenance().CommitOID != equivalentPinned.Provenance().CommitOID ||
			originalPinned.Provenance().TreeOID != equivalentPinned.Provenance().TreeOID ||
			primaryGit.repository.Fingerprint() == equivalent.repository.Fingerprint() {
			t.Fatal("exact-content repository fixture did not preserve portable identity while changing local provenance")
		}
		before := snapshot(t)
		opened, err := OpenOfficialTarget(context.Background(), OpenOfficialTargetRequest{
			Store: primary.objectStore, Residue: primary.residue, Repository: equivalent.repository,
			TargetDigest: primaryTarget.Digest(),
		})
		assertNoMutation(t, before)
		if err != nil || !opened.Valid() || opened.Digest() != primaryTarget.Digest() ||
			opened.state.receipt.RepositoryFingerprint != equivalent.repository.Fingerprint() {
			t.Fatalf("portable exact-content repository did not rebuild current local authority: %v", err)
		}
		if err := opened.Close(); err != nil {
			t.Fatal(err)
		}
	})

	sameRootStore, err := store.OpenObjectStore(primary.root)
	if err != nil {
		t.Fatal(err)
	}
	unrelated := c3NewGitWithFiles(t, []gitrepo.File{
		{Path: clifixture.Entrypoint, Mode: "100755", Content: []byte("console.log('unrelated')\n")},
	}, "C3 unrelated target")
	closed := c3NewGit(t)
	if err := closed.repository.Close(); err != nil {
		t.Fatal(err)
	}
	absent := domain.MustDigest("sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	cases := []struct {
		name    string
		request OpenOfficialTargetRequest
		code    string
	}{
		{"same-root foreign store handle with retained residue", OpenOfficialTargetRequest{
			Store: sameRootStore, Residue: primary.residue, Repository: primaryGit.repository, TargetDigest: primaryTarget.Digest(),
		}, CodeOfficialTargetChanged},
		{"unrelated store with retained residue", OpenOfficialTargetRequest{
			Store: foreign.objectStore, Residue: primary.residue, Repository: primaryGit.repository, TargetDigest: primaryTarget.Digest(),
		}, CodeOfficialTargetChanged},
		{"foreign-store residue with original store", OpenOfficialTargetRequest{
			Store: primary.objectStore, Residue: foreign.residue, Repository: primaryGit.repository, TargetDigest: primaryTarget.Digest(),
		}, CodeOfficialTargetChanged},
		{"second study residue with original target", OpenOfficialTargetRequest{
			Store: primary.objectStore, Residue: secondStudy.residue, Repository: primaryGit.repository, TargetDigest: primaryTarget.Digest(),
		}, CodeInvalidOfficialTarget},
		{"original residue with second study target", OpenOfficialTargetRequest{
			Store: primary.objectStore, Residue: primary.residue, Repository: secondGit.repository, TargetDigest: secondTarget.Digest(),
		}, CodeInvalidOfficialTarget},
		{"unrelated repository", OpenOfficialTargetRequest{
			Store: primary.objectStore, Residue: primary.residue, Repository: unrelated.repository, TargetDigest: primaryTarget.Digest(),
		}, CodeOfficialTargetChanged},
		{"closed repository", OpenOfficialTargetRequest{
			Store: primary.objectStore, Residue: primary.residue, Repository: closed.repository, TargetDigest: primaryTarget.Digest(),
		}, CodeInvalidOfficialTarget},
		{"zero target locator", OpenOfficialTargetRequest{
			Store: primary.objectStore, Residue: primary.residue, Repository: primaryGit.repository,
		}, CodeInvalidOfficialTarget},
		{"valid absent target locator", OpenOfficialTargetRequest{
			Store: primary.objectStore, Residue: primary.residue, Repository: primaryGit.repository, TargetDigest: absent,
		}, CodeInvalidOfficialTarget},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			before := snapshot(t)
			opened, openErr := OpenOfficialTarget(context.Background(), test.request)
			assertNoMutation(t, before)
			if !IsCode(openErr, test.code) || opened.Valid() || opened.Digest().Valid() || opened.Model().Valid() ||
				opened.TargetRecord().Valid() || opened.ContractBundle().Valid() || opened.CandidateRoot() != "" ||
				opened.Roots().AttemptRoot() != "" {
				t.Fatalf("foreign live parent issued authority: code=%s err=%v", test.code, openErr)
			}
			if !primaryTarget.Valid() || !secondTarget.Valid() {
				t.Fatal("refused reopen revoked an existing target capability")
			}
		})
	}
}

func TestC3OfficialTargetMovingRefCannotRetargetReopen(t *testing.T) {
	residue := c3NewResidue(t, "moving-ref")
	git := c3NewGit(t)
	target := c3Publish(t, residue, git)
	commit, err := git.fixture.CommitFiles(context.Background(), []gitrepo.File{{Path: "other.mjs", Mode: "100644", Content: []byte("other\n")}}, "", "moved")
	if err != nil {
		t.Fatal(err)
	}
	if err := git.fixture.UpdateRef(context.Background(), git.ref, commit); err != nil {
		t.Fatal(err)
	}
	reopened, err := ReopenOfficialTarget(context.Background(), target)
	if err != nil || !reopened.Valid() || reopened.Digest() != target.Digest() {
		t.Fatalf("moving ref changed pinned target reopen: %v", err)
	}
	t.Run("runtime alias is not retained authority", func(t *testing.T) {
		realNode, err := filepath.EvalSymlinks(c3SystemNode)
		if err != nil {
			t.Skipf("exact Node unavailable: %v", err)
		}
		aliasRoot, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		alias := filepath.Join(aliasRoot, "node-alias")
		if err := os.Symlink(realNode, alias); err != nil {
			t.Fatal(err)
		}
		aliasResidue := c3NewResidue(t, "runtime-alias")
		aliasGit := c3NewGit(t)
		aliasTarget := c3PublishWithNode(t, aliasResidue, aliasGit, alias)
		if aliasTarget.Model().Input().Runtime.AdmittedExecutablePath != realNode {
			t.Fatal("runtime alias, rather than its canonical executable, entered target identity")
		}
		if err := os.Remove(alias); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("/usr/bin/false", alias); err != nil {
			t.Fatal(err)
		}
		fresh, err := ReopenOfficialTarget(context.Background(), aliasTarget)
		if err != nil || !fresh.Valid() || fresh.Digest() != aliasTarget.Digest() {
			t.Fatalf("irrelevant alias retargeting changed canonical runtime authority: %v", err)
		}
	})
}

func TestC3OfficialTargetMaterializationMutationRefusesReopen(t *testing.T) {
	for name, mutate := range map[string]func(*testing.T, OfficialTarget){
		"entrypoint bytes": func(t *testing.T, target OfficialTarget) {
			if err := os.WriteFile(filepath.Join(target.CandidateRoot(), clifixture.Entrypoint), []byte("changed\n"), 0o755); err != nil {
				t.Fatal(err)
			}
		},
		"extra member": func(t *testing.T, target OfficialTarget) {
			if err := os.WriteFile(filepath.Join(target.CandidateRoot(), "extra.mjs"), []byte("extra\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			residue := c3NewResidue(t, "mutation-"+strings.ReplaceAll(name, " ", "-"))
			git := c3NewGit(t)
			target := c3Publish(t, residue, git)
			mutate(t, target)
			if !target.Valid() {
				t.Fatal("structural Valid performed physical materialization freshness")
			}
			if reopened, err := ReopenOfficialTarget(context.Background(), target); err == nil || reopened.Valid() {
				t.Fatalf("mutated target reopened authority: %v", err)
			}
		})
	}
	t.Run("runtime executable bytes", func(t *testing.T) {
		residue := c3NewResidue(t, "runtime-byte-mutation")
		git := c3NewGit(t)
		executable := c3MutableNode(t)
		target := c3PublishWithNode(t, residue, git, executable)
		handle, err := os.OpenFile(executable, os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			t.Fatal(err)
		}
		_, writeErr := handle.Write([]byte{0})
		syncErr := handle.Sync()
		closeErr := handle.Close()
		if writeErr != nil || syncErr != nil || closeErr != nil {
			t.Fatalf("mutate exact Node bytes: write=%v sync=%v close=%v", writeErr, syncErr, closeErr)
		}
		if !target.Valid() {
			t.Fatal("structural Valid unexpectedly claimed or performed physical freshness")
		}
		if reopened, err := ReopenOfficialTarget(context.Background(), target); !IsCode(err, CodeOfficialTargetChanged) || reopened.Valid() {
			t.Fatalf("changed runtime bytes reopened target authority: %v", err)
		}
	})
}

func TestC3OfficialTargetLinkedRelationshipMutationRefusesReopen(t *testing.T) {
	linkResidue := c3NewResidue(t, "target-link-mutation")
	linkGit := c3NewGit(t)
	linkTarget := c3Publish(t, linkResidue, linkGit)
	linkDirectory := filepath.Join(linkResidue.root, "contract-execution", "links", "target-by-attempt")
	links, err := os.ReadDir(linkDirectory)
	if err != nil || len(links) != 1 {
		t.Fatalf("exact target-link fixture differs: count=%d err=%v", len(links), err)
	}
	linkPath := filepath.Join(linkDirectory, links[0].Name())
	if err := os.WriteFile(linkPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !linkTarget.Valid() {
		t.Fatal("structural Valid performed durable relationship freshness")
	}
	if linkTarget.TargetRecord().Valid() {
		t.Fatal("the independent durable target record check accepted a mutated relationship")
	}
	if reopened, err := ReopenOfficialTarget(context.Background(), linkTarget); err == nil || reopened.Valid() {
		t.Fatalf("mutated typed target link reopened authority: %v", err)
	}

	t.Run("structural getters never repair missing durable directories", func(t *testing.T) {
		residue := c3NewResidue(t, "pure-structural-getters")
		git := c3NewGit(t)
		target := c3Publish(t, residue, git)
		directory := filepath.Join(residue.root, "contract-execution", "links", "target-by-attempt")
		if err := os.RemoveAll(directory); err != nil {
			t.Fatal(err)
		}
		before := c3ExactTreeSnapshot(t, residue.root)
		if !target.Valid() || !target.Digest().Valid() || !target.Model().Valid() ||
			reflect.ValueOf(target.TargetRecord()).IsZero() || target.Roots().AttemptRoot() == "" ||
			target.CandidateRoot() == "" || !target.ContractBundle().Valid() {
			t.Fatal("structural getters lost their sealed in-memory target facts")
		}
		after := c3ExactTreeSnapshot(t, residue.root)
		if !reflect.DeepEqual(after, before) {
			t.Fatalf("structural validity or getters repaired durable storage: before=%v after=%v", before, after)
		}
		if reopened, err := ReopenOfficialTarget(context.Background(), target); err == nil || reopened.Valid() {
			t.Fatalf("missing durable relationship directory reopened authority: %v", err)
		}
	})

	t.Run("stored boot session differs from live epoch", func(t *testing.T) {
		bootResidue := c3NewResidue(t, "stored-boot-mismatch")
		bootGit := c3NewGit(t)
		base := c3Publish(t, bootResidue, bootGit)
		input := base.Model().Input()
		freshResidue, head, err := reopenResidueHead(context.Background(), bootResidue.objectStore, bootResidue.residue)
		if err != nil {
			t.Fatal(err)
		}
		policy, err := policyForResidue(freshResidue)
		if err != nil {
			t.Fatal(err)
		}
		pinned, err := bootGit.repository.Pin(context.Background(), input.Tree.CommitOID)
		if err != nil || pinned.IdentityDigest() != input.Tree.TreeIdentityDigest {
			t.Fatalf("pin exact boot-mismatch tree: %v", err)
		}
		attempt, err := bootResidue.objectStore.AllocateConformanceAttempt(context.Background(), store.ConformanceAttemptInput{
			ContractBundleDigest: input.ContractBundleDigest, ResidueHeadDigest: head.HeadDigest(),
			TreeIdentityDigest: pinned.IdentityDigest(), MaterializationPolicyDigest: policy.Digest(),
		})
		if err != nil {
			t.Fatal(err)
		}
		source, err := gitobj.Inspect(context.Background(), pinned, policy, attempt.Roots().CandidateParent())
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := gitobj.MaterializeSingleTarget(context.Background(), source, attempt.Roots().CandidateParent())
		if err != nil {
			t.Fatal(err)
		}
		receipt, err = gitobj.ReopenSingleTarget(context.Background(), source, attempt.Roots().CandidateParent())
		if err != nil || receipt.ManifestDigest != input.Tree.MaterializationManifestDigest {
			t.Fatalf("reopen boot-mismatch materialization: %v", err)
		}
		input.Attempt = model.AttemptBinding{ArtifactDigest: attempt.Digest(), InstanceNonce: attempt.InstanceNonce()}
		input.BootSession.IdentityDigest = domain.MustDigest("sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
		alternate, err := model.NewContractExecutionTarget(input)
		if err != nil || alternate.Digest() == base.Digest() {
			t.Fatalf("build stored boot-mismatch target: %v", err)
		}
		record, err := bootResidue.objectStore.PersistContractTargetRecord(context.Background(), attempt, alternate)
		if err != nil || !record.Valid() || record.Digest() != alternate.Digest() {
			t.Fatalf("persist stored boot-mismatch target: %v", err)
		}
		beforeInfo, beforeBytes := c3HeadSnapshot(t, bootResidue)
		opened, err := OpenOfficialTarget(context.Background(), OpenOfficialTargetRequest{
			Store: bootResidue.objectStore, Residue: bootResidue.residue, Repository: bootGit.repository,
			TargetDigest: alternate.Digest(),
		})
		afterInfo, afterBytes := c3HeadSnapshot(t, bootResidue)
		if !IsCode(err, CodeOfficialTargetChanged) || opened.Valid() || !os.SameFile(beforeInfo, afterInfo) ||
			!bytes.Equal(beforeBytes, afterBytes) {
			t.Fatalf("stored boot mismatch reopened authority or changed head: %v", err)
		}
	})
}

func c3OfficialReceiver(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return value.Sel.Name
	case *ast.ParenExpr:
		return c3OfficialReceiver(value.X)
	case *ast.StarExpr:
		return c3OfficialReceiver(value.X)
	case *ast.IndexExpr:
		return c3OfficialReceiver(value.X)
	case *ast.IndexListExpr:
		return c3OfficialReceiver(value.X)
	default:
		return ""
	}
}

func c3OfficialUnparen(expression ast.Expr) ast.Expr {
	for {
		parenthesized, ok := expression.(*ast.ParenExpr)
		if !ok {
			return expression
		}
		expression = parenthesized.X
	}
}

const c3ContractPackagePath = "github.com/nelsonwerd/countershape/internal/contractexec"
const c3ModulePath = "github.com/nelsonwerd/countershape"

type c3OfficialTypeContext struct {
	packagePath string
	imports     map[string]string
	dotImports  map[string]bool
}

type c3OfficialTypeDeclaration struct {
	expression           ast.Expr
	context              c3OfficialTypeContext
	typeParams           *ast.FieldList
	alias                bool
	typeParameter        bool
	unknownTypeParameter bool
	officialTarget       bool
	bindings             map[string]c3OfficialTypeDeclaration
}

type c3OfficialFunctionDeclaration struct {
	function *ast.FuncType
	context  c3OfficialTypeContext
	receiver *ast.FieldList
}

type c3OfficialNamedTypes struct {
	context      c3OfficialTypeContext
	declarations map[string]c3OfficialTypeDeclaration
	functions    map[string]c3OfficialFunctionDeclaration
}

func c3OfficialContext(packagePath string, file *ast.File) c3OfficialTypeContext {
	return c3OfficialContextWithPackageNames(packagePath, file, nil)
}

func c3OfficialContextWithPackageNames(
	packagePath string,
	file *ast.File,
	packageNames map[string]string,
) c3OfficialTypeContext {
	imports := make(map[string]string)
	dotImports := make(map[string]bool)
	for _, specification := range file.Imports {
		path, err := strconv.Unquote(specification.Path.Value)
		if err != nil {
			continue
		}
		alias := filepath.Base(path)
		if specification.Name != nil {
			alias = specification.Name.Name
		} else if declaredName := packageNames[path]; declaredName != "" {
			alias = declaredName
		}
		if alias == "." {
			dotImports[path] = true
			continue
		}
		imports[alias] = path
	}
	return c3OfficialTypeContext{packagePath: packagePath, imports: imports, dotImports: dotImports}
}

func c3OfficialTypeKey(packagePath, name string) string { return packagePath + "." + name }

func c3OfficialAddTypes(declarations map[string]c3OfficialTypeDeclaration, context c3OfficialTypeContext, file *ast.File) {
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, specification := range general.Specs {
			typeSpec, ok := specification.(*ast.TypeSpec)
			if ok {
				declarations[c3OfficialTypeKey(context.packagePath, typeSpec.Name.Name)] = c3OfficialTypeDeclaration{
					expression: typeSpec.Type, context: context, typeParams: typeSpec.TypeParams,
					alias:          typeSpec.Assign.IsValid(),
					officialTarget: context.packagePath == c3ContractPackagePath && typeSpec.Name.Name == "OfficialTarget",
				}
			}
		}
	}
}

func c3OfficialAddFunctions(
	functions map[string]c3OfficialFunctionDeclaration,
	context c3OfficialTypeContext,
	file *ast.File,
) {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		name := function.Name.Name
		if function.Recv != nil && len(function.Recv.List) == 1 {
			receiver := c3OfficialReceiver(function.Recv.List[0].Type)
			if receiver == "" {
				continue
			}
			name = receiver + "." + name
		}
		functions[c3OfficialTypeKey(context.packagePath, name)] = c3OfficialFunctionDeclaration{
			function: function.Type, context: context,
			receiver: function.Recv,
		}
	}
}

func c3OfficialTypeIndex(files ...*ast.File) c3OfficialNamedTypes {
	declarations := make(map[string]c3OfficialTypeDeclaration)
	functions := make(map[string]c3OfficialFunctionDeclaration)
	var context c3OfficialTypeContext
	for _, file := range files {
		context = c3OfficialContext(c3ContractPackagePath, file)
		c3OfficialAddTypes(declarations, context, file)
		c3OfficialAddFunctions(functions, context, file)
	}
	return c3OfficialNamedTypes{context: context, declarations: declarations, functions: functions}
}

func c3OfficialResolveNamed(expression ast.Expr, named c3OfficialNamedTypes) (c3OfficialTypeDeclaration, string, bool) {
	var key string
	switch value := expression.(type) {
	case *ast.Ident:
		key = c3OfficialTypeKey(named.context.packagePath, value.Name)
		if declaration, ok := named.declarations[key]; ok {
			return declaration, key, true
		}
		var matched c3OfficialTypeDeclaration
		var matchedKey string
		for packagePath := range named.context.dotImports {
			candidateKey := c3OfficialTypeKey(packagePath, value.Name)
			if declaration, ok := named.declarations[candidateKey]; ok {
				if matchedKey != "" {
					return c3OfficialTypeDeclaration{}, "", false
				}
				matched, matchedKey = declaration, candidateKey
			}
		}
		if matchedKey != "" {
			return matched, matchedKey, true
		}
	case *ast.SelectorExpr:
		qualifier, ok := value.X.(*ast.Ident)
		if !ok {
			return c3OfficialTypeDeclaration{}, "", false
		}
		packagePath, ok := named.context.imports[qualifier.Name]
		if !ok {
			return c3OfficialTypeDeclaration{}, "", false
		}
		key = c3OfficialTypeKey(packagePath, value.Sel.Name)
	default:
		return c3OfficialTypeDeclaration{}, "", false
	}
	declaration, ok := named.declarations[key]
	return declaration, key, ok
}

func c3OfficialResolveFunction(
	expression ast.Expr,
	named c3OfficialNamedTypes,
) (c3OfficialFunctionDeclaration, string, bool) {
	expression = c3OfficialUnparen(expression)
	var key string
	switch value := expression.(type) {
	case *ast.Ident:
		key = c3OfficialTypeKey(named.context.packagePath, value.Name)
		if declaration, ok := named.functions[key]; ok {
			return declaration, key, true
		}
		var matched c3OfficialFunctionDeclaration
		var matchedKey string
		for packagePath := range named.context.dotImports {
			candidateKey := c3OfficialTypeKey(packagePath, value.Name)
			if declaration, ok := named.functions[candidateKey]; ok {
				if matchedKey != "" {
					return c3OfficialFunctionDeclaration{}, "", false
				}
				matched, matchedKey = declaration, candidateKey
			}
		}
		if matchedKey != "" {
			return matched, matchedKey, true
		}
	case *ast.SelectorExpr:
		qualifier, ok := value.X.(*ast.Ident)
		if !ok {
			return c3OfficialFunctionDeclaration{}, "", false
		}
		packagePath, ok := named.context.imports[qualifier.Name]
		if !ok {
			return c3OfficialFunctionDeclaration{}, "", false
		}
		key = c3OfficialTypeKey(packagePath, value.Sel.Name)
	default:
		return c3OfficialFunctionDeclaration{}, "", false
	}
	declaration, ok := named.functions[key]
	return declaration, key, ok
}

func c3OfficialDirectType(expression ast.Expr, named c3OfficialNamedTypes) bool {
	switch value := expression.(type) {
	case *ast.Ident:
		if value.Name != "OfficialTarget" {
			return false
		}
		if declaration, ok := named.declarations[c3OfficialTypeKey(named.context.packagePath, value.Name)]; ok {
			return declaration.officialTarget
		}
		return named.context.packagePath == c3ContractPackagePath || named.context.dotImports[c3ContractPackagePath]
	case *ast.SelectorExpr:
		qualifier, ok := value.X.(*ast.Ident)
		return ok && value.Sel.Name == "OfficialTarget" && named.context.imports[qualifier.Name] == c3ContractPackagePath
	case *ast.StarExpr:
		return c3OfficialDirectType(value.X, named)
	case *ast.ParenExpr:
		return c3OfficialDirectType(value.X, named)
	default:
		return false
	}
}

func c3OfficialWithTypeParams(named c3OfficialNamedTypes, parameters *ast.FieldList) c3OfficialNamedTypes {
	if parameters == nil || len(parameters.List) == 0 {
		return named
	}
	declarations := make(map[string]c3OfficialTypeDeclaration, len(named.declarations)+len(parameters.List))
	for key, declaration := range named.declarations {
		declarations[key] = declaration
	}
	for _, parameter := range parameters.List {
		for _, identifier := range parameter.Names {
			declarations[c3OfficialTypeKey(named.context.packagePath, identifier.Name)] = c3OfficialTypeDeclaration{
				expression: parameter.Type, context: named.context, typeParameter: true,
			}
		}
	}
	return c3OfficialNamedTypes{context: named.context, declarations: declarations, functions: named.functions}
}

type c3OfficialTypeParameter struct {
	name       string
	constraint ast.Expr
}

func c3OfficialTypeParameters(parameters *ast.FieldList) []c3OfficialTypeParameter {
	if parameters == nil {
		return nil
	}
	var flattened []c3OfficialTypeParameter
	for _, field := range parameters.List {
		for _, identifier := range field.Names {
			flattened = append(flattened, c3OfficialTypeParameter{name: identifier.Name, constraint: field.Type})
		}
	}
	return flattened
}

func c3OfficialCloneDeclarations(
	declarations map[string]c3OfficialTypeDeclaration,
	extra int,
) map[string]c3OfficialTypeDeclaration {
	clone := make(map[string]c3OfficialTypeDeclaration, len(declarations)+extra)
	for key, declaration := range declarations {
		clone[key] = declaration
	}
	return clone
}

func c3OfficialDeclarationTypes(
	declaration c3OfficialTypeDeclaration,
	named c3OfficialNamedTypes,
) c3OfficialNamedTypes {
	declarations := named.declarations
	if len(declaration.bindings) > 0 {
		declarations = c3OfficialCloneDeclarations(named.declarations, len(declaration.bindings))
		for key, binding := range declaration.bindings {
			declarations[key] = binding
		}
	}
	return c3OfficialNamedTypes{
		context: declaration.context, declarations: declarations, functions: named.functions,
	}
}

func c3OfficialDeclarationVisitKey(key string, declaration c3OfficialTypeDeclaration) string {
	return key + "@" + strconv.FormatInt(int64(declaration.expression.Pos()), 10) + ":" +
		strconv.FormatInt(int64(declaration.expression.End()), 10)
}

func c3OfficialIdentity(kind string, parts ...string) string {
	var identity strings.Builder
	identity.WriteString(kind)
	for _, part := range parts {
		identity.WriteByte('[')
		identity.WriteString(strconv.Itoa(len(part)))
		identity.WriteByte(':')
		identity.WriteString(part)
		identity.WriteByte(']')
	}
	return identity.String()
}

func c3OfficialFieldListIdentity(
	fields *ast.FieldList,
	named c3OfficialNamedTypes,
	visiting map[string]bool,
) string {
	if fields == nil {
		return "nil"
	}
	parts := make([]string, 0, len(fields.List))
	for _, field := range fields.List {
		names := make([]string, 0, len(field.Names))
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
		parts = append(parts, c3OfficialIdentity(
			"field", strings.Join(names, ","), c3OfficialTypeIdentity(field.Type, named, visiting),
		))
	}
	return c3OfficialIdentity("fields", parts...)
}

func c3OfficialTypeIdentity(
	expression ast.Expr,
	named c3OfficialNamedTypes,
	visiting map[string]bool,
) string {
	if expression == nil {
		return "nil"
	}
	if c3OfficialDirectType(expression, named) {
		return "official-target"
	}
	switch value := expression.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		declaration, key, ok := c3OfficialResolveNamed(value, named)
		if ok {
			if declaration.unknownTypeParameter {
				return c3OfficialIdentity("unknown-parameter", key)
			}
			if !declaration.typeParameter {
				return c3OfficialIdentity("named", key)
			}
			visitKey := c3OfficialDeclarationVisitKey(key, declaration)
			if visiting[visitKey] {
				return c3OfficialIdentity("cycle", visitKey)
			}
			visiting[visitKey] = true
			identity := c3OfficialIdentity(
				"parameter", key,
				c3OfficialTypeIdentity(declaration.expression, c3OfficialDeclarationTypes(declaration, named), visiting),
			)
			delete(visiting, visitKey)
			return identity
		}
	case *ast.ParenExpr:
		return c3OfficialTypeIdentity(value.X, named, visiting)
	case *ast.StarExpr:
		return c3OfficialIdentity("pointer", c3OfficialTypeIdentity(value.X, named, visiting))
	case *ast.UnaryExpr:
		return c3OfficialIdentity("unary:"+value.Op.String(), c3OfficialTypeIdentity(value.X, named, visiting))
	case *ast.BinaryExpr:
		return c3OfficialIdentity(
			"binary:"+value.Op.String(),
			c3OfficialTypeIdentity(value.X, named, visiting),
			c3OfficialTypeIdentity(value.Y, named, visiting),
		)
	case *ast.ArrayType:
		length := "slice"
		if value.Len != nil {
			var rendered strings.Builder
			_ = printer.Fprint(&rendered, token.NewFileSet(), value.Len)
			length = rendered.String()
		}
		return c3OfficialIdentity("array", length, c3OfficialTypeIdentity(value.Elt, named, visiting))
	case *ast.MapType:
		return c3OfficialIdentity(
			"map", c3OfficialTypeIdentity(value.Key, named, visiting), c3OfficialTypeIdentity(value.Value, named, visiting),
		)
	case *ast.ChanType:
		return c3OfficialIdentity(
			"chan:"+strconv.Itoa(int(value.Dir)), c3OfficialTypeIdentity(value.Value, named, visiting),
		)
	case *ast.Ellipsis:
		return c3OfficialIdentity("ellipsis", c3OfficialTypeIdentity(value.Elt, named, visiting))
	case *ast.IndexExpr:
		return c3OfficialIdentity(
			"index", c3OfficialTypeIdentity(value.X, named, visiting),
			c3OfficialTypeIdentity(value.Index, named, visiting),
		)
	case *ast.IndexListExpr:
		parts := []string{c3OfficialTypeIdentity(value.X, named, visiting)}
		for _, argument := range value.Indices {
			parts = append(parts, c3OfficialTypeIdentity(argument, named, visiting))
		}
		return c3OfficialIdentity("index-list", parts...)
	case *ast.StructType:
		return c3OfficialIdentity("struct", c3OfficialFieldListIdentity(value.Fields, named, visiting))
	case *ast.InterfaceType:
		return c3OfficialIdentity("interface", c3OfficialFieldListIdentity(value.Methods, named, visiting))
	case *ast.FuncType:
		return c3OfficialIdentity(
			"func",
			c3OfficialFieldListIdentity(value.TypeParams, named, visiting),
			c3OfficialFieldListIdentity(value.Params, named, visiting),
			c3OfficialFieldListIdentity(value.Results, named, visiting),
		)
	}
	var rendered strings.Builder
	_ = printer.Fprint(&rendered, token.NewFileSet(), expression)
	return c3OfficialIdentity("syntax", named.context.packagePath, rendered.String())
}

func c3OfficialBindTypeArguments(
	context c3OfficialTypeContext,
	parameters *ast.FieldList,
	arguments []ast.Expr,
	named c3OfficialNamedTypes,
	allowPartial bool,
) (c3OfficialNamedTypes, bool) {
	flattened := c3OfficialTypeParameters(parameters)
	if len(flattened) == 0 || len(arguments) > len(flattened) || (!allowPartial && len(flattened) != len(arguments)) {
		return c3OfficialNamedTypes{}, false
	}
	declarations := c3OfficialCloneDeclarations(named.declarations, len(flattened))
	for index, parameter := range flattened {
		if index >= len(arguments) {
			declarations[c3OfficialTypeKey(context.packagePath, parameter.name)] = c3OfficialTypeDeclaration{
				expression: parameter.constraint, context: context, typeParameter: true, unknownTypeParameter: true,
			}
			continue
		}
		binding := c3OfficialTypeDeclaration{
			expression: arguments[index], context: named.context, typeParameter: true,
		}
		if actual, _, resolved := c3OfficialResolveNamed(arguments[index], named); resolved && actual.typeParameter {
			binding = actual
		}
		declarations[c3OfficialTypeKey(context.packagePath, parameter.name)] = binding
	}
	return c3OfficialNamedTypes{
		context: context, declarations: declarations, functions: named.functions,
	}, true
}

func c3OfficialInstantiate(
	expression ast.Expr,
	named c3OfficialNamedTypes,
) (c3OfficialTypeDeclaration, c3OfficialNamedTypes, string, bool) {
	var base ast.Expr
	var arguments []ast.Expr
	switch value := expression.(type) {
	case *ast.IndexExpr:
		base, arguments = value.X, []ast.Expr{value.Index}
	case *ast.IndexListExpr:
		base, arguments = value.X, value.Indices
	default:
		return c3OfficialTypeDeclaration{}, c3OfficialNamedTypes{}, "", false
	}
	declaration, key, ok := c3OfficialResolveNamed(base, named)
	if !ok {
		return c3OfficialTypeDeclaration{}, c3OfficialNamedTypes{}, "", false
	}
	instantiated, ok := c3OfficialBindTypeArguments(declaration.context, declaration.typeParams, arguments, named, false)
	if !ok {
		return c3OfficialTypeDeclaration{}, c3OfficialNamedTypes{}, "", false
	}
	argumentIdentities := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		argumentIdentities = append(argumentIdentities, c3OfficialTypeIdentity(argument, named, make(map[string]bool)))
	}
	return declaration, instantiated, c3OfficialIdentity("instantiate:"+key, argumentIdentities...), true
}

func c3OfficialInstantiateFunction(
	expression ast.Expr,
	named c3OfficialNamedTypes,
) (c3OfficialFunctionDeclaration, c3OfficialNamedTypes, bool) {
	expression = c3OfficialUnparen(expression)
	var base ast.Expr
	var arguments []ast.Expr
	switch value := expression.(type) {
	case *ast.IndexExpr:
		base, arguments = value.X, []ast.Expr{value.Index}
	case *ast.IndexListExpr:
		base, arguments = value.X, value.Indices
	default:
		base = expression
	}
	declaration, _, ok := c3OfficialResolveFunction(base, named)
	if _, _, shadowed := c3OfficialResolveNamed(base, named); shadowed {
		return c3OfficialFunctionDeclaration{}, c3OfficialNamedTypes{}, false
	}
	if !ok {
		return c3OfficialFunctionDeclaration{}, c3OfficialNamedTypes{}, false
	}
	if len(c3OfficialTypeParameters(declaration.function.TypeParams)) == 0 {
		if len(arguments) != 0 {
			return c3OfficialFunctionDeclaration{}, c3OfficialNamedTypes{}, false
		}
		return declaration, c3OfficialNamedTypes{
			context: declaration.context, declarations: named.declarations, functions: named.functions,
		}, true
	}
	scoped, ok := c3OfficialBindTypeArguments(
		declaration.context, declaration.function.TypeParams, arguments, named, true,
	)
	return declaration, scoped, ok
}

func c3OfficialReceiverInstance(
	expression ast.Expr,
	named c3OfficialNamedTypes,
) (c3OfficialTypeDeclaration, string, []ast.Expr, c3OfficialNamedTypes, bool) {
	return c3OfficialReceiverInstanceSeen(expression, named, make(map[string]bool))
}

func c3OfficialReceiverInstanceSeen(
	expression ast.Expr,
	named c3OfficialNamedTypes,
	seenAliases map[string]bool,
) (c3OfficialTypeDeclaration, string, []ast.Expr, c3OfficialNamedTypes, bool) {
	expression = c3OfficialUnparen(expression)
	for {
		pointer, ok := expression.(*ast.StarExpr)
		if !ok {
			break
		}
		expression = c3OfficialUnparen(pointer.X)
	}
	var base ast.Expr
	var arguments []ast.Expr
	switch value := expression.(type) {
	case *ast.IndexExpr:
		base, arguments = value.X, []ast.Expr{value.Index}
	case *ast.IndexListExpr:
		base, arguments = value.X, value.Indices
	default:
		base = expression
	}
	declaration, key, ok := c3OfficialResolveNamed(base, named)
	if !ok || declaration.typeParameter {
		return c3OfficialTypeDeclaration{}, "", nil, c3OfficialNamedTypes{}, false
	}
	var scoped c3OfficialNamedTypes
	if len(c3OfficialTypeParameters(declaration.typeParams)) == 0 {
		if len(arguments) != 0 {
			return c3OfficialTypeDeclaration{}, "", nil, c3OfficialNamedTypes{}, false
		}
		scoped = c3OfficialNamedTypes{
			context: declaration.context, declarations: named.declarations, functions: named.functions,
		}
	} else {
		var bound bool
		scoped, bound = c3OfficialBindTypeArguments(declaration.context, declaration.typeParams, arguments, named, false)
		if !bound {
			return c3OfficialTypeDeclaration{}, "", nil, c3OfficialNamedTypes{}, false
		}
	}
	if !declaration.alias {
		return declaration, key, arguments, scoped, true
	}
	aliasKey := c3OfficialIdentity(
		"receiver-alias", key, c3OfficialTypeIdentity(expression, named, make(map[string]bool)),
	)
	if seenAliases[aliasKey] {
		return c3OfficialTypeDeclaration{}, "", nil, c3OfficialNamedTypes{}, false
	}
	seenAliases[aliasKey] = true
	return c3OfficialReceiverInstanceSeen(declaration.expression, scoped, seenAliases)
}

func c3OfficialConcreteMethodTypes(
	method c3OfficialFunctionDeclaration,
	receiverType c3OfficialTypeDeclaration,
	arguments []ast.Expr,
	scoped c3OfficialNamedTypes,
) c3OfficialNamedTypes {
	parameters := c3OfficialTypeParameters(receiverType.typeParams)
	if len(parameters) == 0 || method.receiver == nil || len(method.receiver.List) != 1 {
		return scoped
	}
	receiverExpression := c3OfficialUnparen(method.receiver.List[0].Type)
	if pointer, ok := receiverExpression.(*ast.StarExpr); ok {
		receiverExpression = c3OfficialUnparen(pointer.X)
	}
	var receiverArguments []ast.Expr
	switch value := receiverExpression.(type) {
	case *ast.IndexExpr:
		receiverArguments = []ast.Expr{value.Index}
	case *ast.IndexListExpr:
		receiverArguments = value.Indices
	}
	if len(receiverArguments) != len(parameters) || len(arguments) != len(parameters) {
		return scoped
	}
	declarations := c3OfficialCloneDeclarations(scoped.declarations, len(receiverArguments))
	for index, argument := range receiverArguments {
		identifier, ok := argument.(*ast.Ident)
		if !ok {
			continue
		}
		binding, ok := scoped.declarations[c3OfficialTypeKey(receiverType.context.packagePath, parameters[index].name)]
		if !ok {
			continue
		}
		declarations[c3OfficialTypeKey(method.context.packagePath, identifier.Name)] = binding
	}
	return c3OfficialNamedTypes{context: method.context, declarations: declarations, functions: scoped.functions}
}

type c3OfficialMethodCandidate struct {
	expression ast.Expr
	named      c3OfficialNamedTypes
	seen       map[string]bool
}

type c3OfficialMethodMatch struct {
	method c3OfficialFunctionDeclaration
	scoped c3OfficialNamedTypes
}

func c3OfficialResolveMethod(
	selector *ast.SelectorExpr,
	named c3OfficialNamedTypes,
	values map[string]c3OfficialValueDeclaration,
	visiting map[string]bool,
) (c3OfficialFunctionDeclaration, c3OfficialNamedTypes, bool) {
	receiverExpression, receiverNamed, ok := c3OfficialExpressionType(selector.X, named, values, visiting)
	if !ok {
		receiverExpression, receiverNamed = selector.X, named
	}
	frontier := []c3OfficialMethodCandidate{{
		expression: receiverExpression, named: receiverNamed, seen: make(map[string]bool),
	}}
	for len(frontier) > 0 {
		var matches []c3OfficialMethodMatch
		var next []c3OfficialMethodCandidate
		fieldMatches := 0
		for _, candidate := range frontier {
			identity := c3OfficialTypeIdentity(candidate.expression, candidate.named, make(map[string]bool))
			if candidate.seen[identity] {
				continue
			}
			seen := make(map[string]bool, len(candidate.seen)+1)
			for key := range candidate.seen {
				seen[key] = true
			}
			seen[identity] = true
			receiverType, receiverKey, arguments, scoped, resolved := c3OfficialReceiverInstance(
				candidate.expression, candidate.named,
			)
			if resolved {
				if method, found := scoped.functions[receiverKey+"."+selector.Sel.Name]; found && method.receiver != nil {
					matches = append(matches, c3OfficialMethodMatch{
						method: method,
						scoped: c3OfficialConcreteMethodTypes(method, receiverType, arguments, scoped),
					})
				}
			}
			underlying, current, crossedOfficialTarget := c3OfficialUnderlying(candidate.expression, candidate.named)
			if crossedOfficialTarget {
				continue
			}
			structure, ok := underlying.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range structure.Fields.List {
				for _, fieldName := range field.Names {
					if fieldName.Name == selector.Sel.Name {
						fieldMatches++
					}
				}
				if len(field.Names) == 0 {
					if c3OfficialEmbeddedFieldName(field.Type) == selector.Sel.Name {
						fieldMatches++
					}
					next = append(next, c3OfficialMethodCandidate{expression: field.Type, named: current, seen: seen})
				}
			}
		}
		if fieldMatches > 0 {
			// A field at this depth blocks every deeper promoted method. If a same-depth
			// method is also present the selector is ambiguous Go; prefer any issuing
			// method so parser-only hostile fixtures still fail conservatively.
			for _, match := range matches {
				if c3OfficialFunctionProducesTarget(match.method.function, match.scoped) {
					return match.method, match.scoped, true
				}
			}
			if len(matches) > 0 {
				return matches[0].method, matches[0].scoped, true
			}
			return c3OfficialFunctionDeclaration{}, c3OfficialNamedTypes{}, false
		}
		if len(matches) == 1 {
			return matches[0].method, matches[0].scoped, true
		}
		if len(matches) > 1 {
			// Equal-depth promotion is not selectable Go; remain conservative if an ambiguous method could issue.
			for _, match := range matches {
				if c3OfficialFunctionProducesTarget(match.method.function, match.scoped) {
					return match.method, match.scoped, true
				}
			}
			return matches[0].method, matches[0].scoped, true
		}
		frontier = next
	}
	return c3OfficialFunctionDeclaration{}, c3OfficialNamedTypes{}, false
}

func c3OfficialMethodProducesTarget(
	selector *ast.SelectorExpr,
	named c3OfficialNamedTypes,
	values map[string]c3OfficialValueDeclaration,
	visiting map[string]bool,
) (bool, bool) {
	method, scoped, ok := c3OfficialResolveMethod(selector, named, values, visiting)
	if !ok {
		return false, false
	}
	return c3OfficialFunctionProducesTarget(method.function, scoped), true
}

func c3OfficialMethodFactoryExpression(
	expression ast.Expr,
	named c3OfficialNamedTypes,
	values map[string]c3OfficialValueDeclaration,
) bool {
	selector, ok := c3OfficialUnparen(expression).(*ast.SelectorExpr)
	if !ok {
		return false
	}
	result, recognized := c3OfficialMethodProducesTarget(selector, named, values, make(map[string]bool))
	return recognized && result
}

func c3OfficialFunctionProducesTarget(function *ast.FuncType, named c3OfficialNamedTypes) bool {
	if function.Results == nil {
		return false
	}
	for _, result := range function.Results.List {
		if c3OfficialResultType(result.Type, named, make(map[string]bool)) {
			return true
		}
	}
	return false
}

func c3OfficialGenericCallResult(expression ast.Expr, named c3OfficialNamedTypes) (bool, bool) {
	declaration, scoped, ok := c3OfficialInstantiateFunction(expression, named)
	if !ok {
		return false, false
	}
	if declaration.function.Results == nil {
		return false, true
	}
	for _, result := range declaration.function.Results.List {
		if c3OfficialResultType(result.Type, scoped, make(map[string]bool)) {
			return true, true
		}
	}
	return false, true
}

func c3OfficialCallResultType(
	expression ast.Expr,
	named c3OfficialNamedTypes,
	values map[string]c3OfficialValueDeclaration,
	visiting map[string]bool,
) bool {
	if result, recognized := c3OfficialGenericCallResult(expression, named); recognized {
		return result
	}
	if selector, ok := c3OfficialUnparen(expression).(*ast.SelectorExpr); ok {
		if result, recognized := c3OfficialMethodProducesTarget(selector, named, values, visiting); recognized {
			return result
		}
	}
	// Calls outside the indexed module graph stay conservative: an unknown generic could return its argument type.
	return c3OfficialResultType(expression, named, make(map[string]bool))
}

func c3OfficialWithReceiverTypeParams(named c3OfficialNamedTypes, receiver *ast.FieldList) c3OfficialNamedTypes {
	if receiver == nil || len(receiver.List) != 1 {
		return named
	}
	expression := receiver.List[0].Type
	unwrapping := true
	for unwrapping {
		switch value := expression.(type) {
		case *ast.ParenExpr:
			expression = value.X
		case *ast.StarExpr:
			expression = value.X
		default:
			unwrapping = false
		}
	}
	var base ast.Expr
	var arguments []ast.Expr
	switch value := expression.(type) {
	case *ast.IndexExpr:
		base, arguments = value.X, []ast.Expr{value.Index}
	case *ast.IndexListExpr:
		base, arguments = value.X, value.Indices
	default:
		return named
	}
	declaration, _, ok := c3OfficialResolveNamed(base, named)
	if !ok || declaration.typeParams == nil {
		return named
	}
	parameters := c3OfficialTypeParameters(declaration.typeParams)
	if len(arguments) != len(parameters) {
		return named
	}
	declarations := c3OfficialCloneDeclarations(named.declarations, len(arguments))
	bindings := make(map[string]c3OfficialTypeDeclaration, len(parameters))
	for _, parameter := range parameters {
		bindings[c3OfficialTypeKey(declaration.context.packagePath, parameter.name)] = c3OfficialTypeDeclaration{
			expression: parameter.constraint, context: declaration.context, typeParameter: true,
		}
	}
	for index, argument := range arguments {
		identifier, ok := argument.(*ast.Ident)
		if !ok {
			continue
		}
		binding := bindings[c3OfficialTypeKey(declaration.context.packagePath, parameters[index].name)]
		binding.bindings = bindings
		declarations[c3OfficialTypeKey(named.context.packagePath, identifier.Name)] = binding
	}
	return c3OfficialNamedTypes{context: named.context, declarations: declarations, functions: named.functions}
}

func c3OfficialResultType(expression ast.Expr, named c3OfficialNamedTypes, visiting map[string]bool) bool {
	if c3OfficialDirectType(expression, named) {
		return true
	}
	switch value := expression.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		declaration, key, ok := c3OfficialResolveNamed(value, named)
		if !ok {
			return false
		}
		if declaration.unknownTypeParameter {
			return true
		}
		key = c3OfficialDeclarationVisitKey(key, declaration)
		if visiting[key] {
			return false
		}
		visiting[key] = true
		result := c3OfficialResultType(declaration.expression, c3OfficialDeclarationTypes(declaration, named), visiting)
		delete(visiting, key)
		return result
	case *ast.ParenExpr:
		return c3OfficialResultType(value.X, named, visiting)
	case *ast.StarExpr:
		return c3OfficialResultType(value.X, named, visiting)
	case *ast.UnaryExpr:
		return c3OfficialResultType(value.X, named, visiting)
	case *ast.BinaryExpr:
		return c3OfficialResultType(value.X, named, visiting) || c3OfficialResultType(value.Y, named, visiting)
	case *ast.ArrayType:
		return c3OfficialResultType(value.Elt, named, visiting)
	case *ast.MapType:
		return c3OfficialResultType(value.Key, named, visiting) || c3OfficialResultType(value.Value, named, visiting)
	case *ast.ChanType:
		return c3OfficialResultType(value.Value, named, visiting)
	case *ast.Ellipsis:
		return c3OfficialResultType(value.Elt, named, visiting)
	case *ast.IndexExpr, *ast.IndexListExpr:
		if declaration, instantiated, key, ok := c3OfficialInstantiate(value, named); ok {
			if visiting[key] {
				return false
			}
			visiting[key] = true
			result := c3OfficialResultType(declaration.expression, instantiated, visiting)
			delete(visiting, key)
			return result
		}
		switch indexed := value.(type) {
		case *ast.IndexExpr:
			return c3OfficialResultType(indexed.X, named, visiting) ||
				c3OfficialResultType(indexed.Index, named, visiting)
		case *ast.IndexListExpr:
			if c3OfficialResultType(indexed.X, named, visiting) {
				return true
			}
			for _, index := range indexed.Indices {
				if c3OfficialResultType(index, named, visiting) {
					return true
				}
			}
		}
	case *ast.StructType:
		for _, field := range value.Fields.List {
			if c3OfficialResultType(field.Type, named, visiting) {
				return true
			}
		}
	case *ast.InterfaceType:
		for _, method := range value.Methods.List {
			if c3OfficialResultType(method.Type, named, visiting) {
				return true
			}
		}
	case *ast.FuncType:
		named = c3OfficialWithTypeParams(named, value.TypeParams)
		if value.Results == nil {
			return false
		}
		for _, result := range value.Results.List {
			if c3OfficialResultType(result.Type, named, visiting) {
				return true
			}
		}
	}
	return false
}

func c3OfficialFactoryType(expression ast.Expr, named c3OfficialNamedTypes, visiting map[string]bool) bool {
	if declaration, scoped, ok := c3OfficialInstantiateFunction(expression, named); ok {
		return c3OfficialFunctionProducesTarget(declaration.function, scoped)
	}
	switch value := expression.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		declaration, key, ok := c3OfficialResolveNamed(value, named)
		if !ok {
			return false
		}
		key = c3OfficialDeclarationVisitKey(key, declaration)
		if visiting[key] {
			return false
		}
		visiting[key] = true
		result := c3OfficialFactoryType(declaration.expression, c3OfficialDeclarationTypes(declaration, named), visiting)
		delete(visiting, key)
		return result
	case *ast.ParenExpr:
		return c3OfficialFactoryType(value.X, named, visiting)
	case *ast.StarExpr:
		return c3OfficialFactoryType(value.X, named, visiting)
	case *ast.UnaryExpr:
		return c3OfficialFactoryType(value.X, named, visiting)
	case *ast.BinaryExpr:
		return c3OfficialFactoryType(value.X, named, visiting) || c3OfficialFactoryType(value.Y, named, visiting)
	case *ast.ArrayType:
		return c3OfficialFactoryType(value.Elt, named, visiting)
	case *ast.MapType:
		return c3OfficialFactoryType(value.Key, named, visiting) || c3OfficialFactoryType(value.Value, named, visiting)
	case *ast.ChanType:
		return c3OfficialFactoryType(value.Value, named, visiting)
	case *ast.Ellipsis:
		return c3OfficialFactoryType(value.Elt, named, visiting)
	case *ast.IndexExpr, *ast.IndexListExpr:
		if declaration, instantiated, key, ok := c3OfficialInstantiate(value, named); ok {
			if visiting[key] {
				return false
			}
			visiting[key] = true
			result := c3OfficialFactoryType(declaration.expression, instantiated, visiting)
			delete(visiting, key)
			return result
		}
		switch indexed := value.(type) {
		case *ast.IndexExpr:
			return c3OfficialFactoryType(indexed.X, named, visiting) ||
				c3OfficialResultType(indexed.Index, named, make(map[string]bool))
		case *ast.IndexListExpr:
			if c3OfficialFactoryType(indexed.X, named, visiting) {
				return true
			}
			for _, index := range indexed.Indices {
				if c3OfficialResultType(index, named, make(map[string]bool)) {
					return true
				}
			}
		}
	case *ast.StructType:
		for _, field := range value.Fields.List {
			if c3OfficialFactoryType(field.Type, named, visiting) {
				return true
			}
		}
	case *ast.InterfaceType:
		for _, method := range value.Methods.List {
			if c3OfficialFactoryType(method.Type, named, visiting) {
				return true
			}
		}
	case *ast.FuncType:
		named = c3OfficialWithTypeParams(named, value.TypeParams)
		if value.Results == nil {
			return false
		}
		for _, result := range value.Results.List {
			if c3OfficialResultType(result.Type, named, make(map[string]bool)) {
				return true
			}
		}
	}
	return false
}

func c3OfficialDeclaredType(expression ast.Expr, named c3OfficialNamedTypes) bool {
	resolved, current, crossedOfficialTarget := c3OfficialUnderlying(expression, named)
	if crossedOfficialTarget {
		return true
	}
	switch value := resolved.(type) {
	case *ast.StructType:
		// C4 may retain an OfficialTarget as an input field. Output-producing callbacks are still issuers.
		return c3OfficialFactoryType(value, current, make(map[string]bool))
	case *ast.FuncType:
		return c3OfficialResultType(value, current, make(map[string]bool))
	default:
		return c3OfficialResultType(expression, named, make(map[string]bool))
	}
}

func c3OfficialUnderlying(
	expression ast.Expr,
	named c3OfficialNamedTypes,
) (ast.Expr, c3OfficialNamedTypes, bool) {
	resolved := expression
	current := named
	seen := make(map[string]bool)
	for {
		if c3OfficialDirectType(resolved, current) {
			return resolved, current, true
		}
		switch value := resolved.(type) {
		case *ast.ParenExpr:
			resolved = value.X
			continue
		case *ast.StarExpr:
			resolved = value.X
			continue
		}
		if declaration, instantiated, key, ok := c3OfficialInstantiate(resolved, current); ok {
			if seen[key] {
				return resolved, current, false
			}
			seen[key] = true
			resolved = declaration.expression
			current = instantiated
			continue
		}
		declaration, key, ok := c3OfficialResolveNamed(resolved, current)
		if !ok {
			return resolved, current, false
		}
		key = c3OfficialDeclarationVisitKey(key, declaration)
		if seen[key] {
			return resolved, current, false
		}
		seen[key] = true
		resolved = declaration.expression
		current = c3OfficialDeclarationTypes(declaration, current)
	}
}

func c3OfficialNonHolderComposite(expression ast.Expr, named c3OfficialNamedTypes) bool {
	resolved, current, crossedOfficialTarget := c3OfficialUnderlying(expression, named)
	if crossedOfficialTarget {
		return true
	}
	if _, ok := resolved.(*ast.StructType); ok {
		return c3OfficialFactoryType(resolved, current, make(map[string]bool))
	}
	return c3OfficialResultType(expression, named, make(map[string]bool))
}

type c3OfficialValueDeclaration struct {
	explicitType ast.Expr
	initializer  ast.Expr
	context      c3OfficialTypeContext
	tupleIndex   int
	tupleWidth   int
}

func c3OfficialAddValues(
	declarations map[string]c3OfficialValueDeclaration,
	context c3OfficialTypeContext,
	candidate *ast.File,
) {
	for _, declaration := range candidate.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, specification := range general.Specs {
			values, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for index, identifier := range values.Names {
				var initializer ast.Expr
				tupleIndex, tupleWidth := 0, 1
				if len(values.Values) == 1 {
					initializer = values.Values[0]
					if len(values.Names) > 1 {
						tupleIndex, tupleWidth = index, len(values.Names)
					}
				} else if index < len(values.Values) {
					initializer = values.Values[index]
				}
				declarations[identifier.Name] = c3OfficialValueDeclaration{
					explicitType: values.Type, initializer: initializer, context: context,
					tupleIndex: tupleIndex, tupleWidth: tupleWidth,
				}
			}
		}
	}
	return
}

func c3OfficialPackageValues(candidate *ast.File, named c3OfficialNamedTypes) map[string]c3OfficialValueDeclaration {
	declarations := make(map[string]c3OfficialValueDeclaration)
	c3OfficialAddValues(declarations, named.context, candidate)
	return declarations
}

func c3OfficialFunctionResultTypes(function *ast.FuncType) []ast.Expr {
	if function.Results == nil {
		return nil
	}
	var results []ast.Expr
	for _, field := range function.Results.List {
		width := len(field.Names)
		if width == 0 {
			width = 1
		}
		for index := 0; index < width; index++ {
			results = append(results, field.Type)
		}
	}
	return results
}

func c3OfficialCallResultAt(
	expression ast.Expr,
	index int,
	named c3OfficialNamedTypes,
	values map[string]c3OfficialValueDeclaration,
	visiting map[string]bool,
) (ast.Expr, c3OfficialNamedTypes, bool) {
	declaration, scoped, ok := c3OfficialInstantiateFunction(expression, named)
	if !ok {
		selector, selected := c3OfficialUnparen(expression).(*ast.SelectorExpr)
		if !selected {
			return nil, c3OfficialNamedTypes{}, false
		}
		declaration, scoped, ok = c3OfficialResolveMethod(selector, named, values, visiting)
		if !ok {
			return nil, c3OfficialNamedTypes{}, false
		}
	}
	results := c3OfficialFunctionResultTypes(declaration.function)
	if index < 0 || index >= len(results) {
		return nil, c3OfficialNamedTypes{}, false
	}
	return results[index], scoped, true
}

func c3OfficialExpressionType(
	expression ast.Expr,
	named c3OfficialNamedTypes,
	values map[string]c3OfficialValueDeclaration,
	visiting map[string]bool,
) (ast.Expr, c3OfficialNamedTypes, bool) {
	switch value := expression.(type) {
	case *ast.ParenExpr:
		return c3OfficialExpressionType(value.X, named, values, visiting)
	case *ast.UnaryExpr:
		return c3OfficialExpressionType(value.X, named, values, visiting)
	case *ast.StarExpr:
		return c3OfficialExpressionType(value.X, named, values, visiting)
	case *ast.CompositeLit:
		return value.Type, named, value.Type != nil
	case *ast.TypeAssertExpr:
		return value.Type, named, value.Type != nil
	case *ast.CallExpr:
		if identifier, ok := c3OfficialUnparen(value.Fun).(*ast.Ident); ok && len(value.Args) > 0 &&
			(identifier.Name == "make" || identifier.Name == "new") {
			return value.Args[0], named, true
		}
		if c3OfficialDirectType(value.Fun, named) {
			return value.Fun, named, true
		}
		if _, _, _, ok := c3OfficialInstantiate(value.Fun, named); ok {
			return value.Fun, named, true
		}
		if _, _, ok := c3OfficialResolveNamed(value.Fun, named); ok {
			return value.Fun, named, true
		}
	case *ast.SelectorExpr:
		return c3OfficialSelectedFieldType(value, named, values, visiting)
	case *ast.IndexExpr:
		return c3OfficialIndexedElementType(value, named, values, visiting)
	case *ast.Ident:
		declaration, ok := values[value.Name]
		key := "value-type:" + named.context.packagePath + "." + value.Name
		if !ok || visiting[key] {
			return nil, c3OfficialNamedTypes{}, false
		}
		visiting[key] = true
		declarationTypes := c3OfficialNamedTypes{
			context: declaration.context, declarations: named.declarations, functions: named.functions,
		}
		if declaration.explicitType != nil {
			delete(visiting, key)
			return declaration.explicitType, declarationTypes, true
		}
		if call, ok := declaration.initializer.(*ast.CallExpr); ok && declaration.tupleWidth > 1 {
			if result, resultTypes, found := c3OfficialCallResultAt(
				call.Fun, declaration.tupleIndex, declarationTypes, values, visiting,
			); found {
				delete(visiting, key)
				return result, resultTypes, true
			}
		}
		if declaration.initializer != nil {
			resolved, current, found := c3OfficialExpressionType(
				declaration.initializer, declarationTypes, values, visiting,
			)
			delete(visiting, key)
			return resolved, current, found
		}
		delete(visiting, key)
	}
	return nil, c3OfficialNamedTypes{}, false
}

func c3OfficialEmbeddedFieldName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return value.Sel.Name
	case *ast.ParenExpr:
		return c3OfficialEmbeddedFieldName(value.X)
	case *ast.StarExpr:
		return c3OfficialEmbeddedFieldName(value.X)
	case *ast.IndexExpr:
		return c3OfficialEmbeddedFieldName(value.X)
	case *ast.IndexListExpr:
		return c3OfficialEmbeddedFieldName(value.X)
	default:
		return ""
	}
}

func c3OfficialIndexedElementType(
	index *ast.IndexExpr,
	named c3OfficialNamedTypes,
	values map[string]c3OfficialValueDeclaration,
	visiting map[string]bool,
) (ast.Expr, c3OfficialNamedTypes, bool) {
	base, baseTypes, ok := c3OfficialExpressionType(index.X, named, values, visiting)
	if !ok {
		return nil, c3OfficialNamedTypes{}, false
	}
	resolved, current, crossedOfficialTarget := c3OfficialUnderlying(base, baseTypes)
	if crossedOfficialTarget {
		return nil, c3OfficialNamedTypes{}, false
	}
	switch value := resolved.(type) {
	case *ast.ArrayType:
		return value.Elt, current, true
	case *ast.MapType:
		return value.Value, current, true
	default:
		return nil, c3OfficialNamedTypes{}, false
	}
}

type c3OfficialFieldCandidate struct {
	expression ast.Expr
	named      c3OfficialNamedTypes
	seen       map[string]bool
}

type c3OfficialFieldMatch struct {
	expression ast.Expr
	named      c3OfficialNamedTypes
}

func c3OfficialFieldType(
	expression ast.Expr,
	name string,
	named c3OfficialNamedTypes,
) (ast.Expr, c3OfficialNamedTypes, bool) {
	frontier := []c3OfficialFieldCandidate{{expression: expression, named: named, seen: make(map[string]bool)}}
	for len(frontier) > 0 {
		var matches []c3OfficialFieldMatch
		var next []c3OfficialFieldCandidate
		for _, candidate := range frontier {
			identity := c3OfficialTypeIdentity(candidate.expression, candidate.named, make(map[string]bool))
			if candidate.seen[identity] {
				continue
			}
			seen := make(map[string]bool, len(candidate.seen)+1)
			for key := range candidate.seen {
				seen[key] = true
			}
			seen[identity] = true
			resolved, current, crossedOfficialTarget := c3OfficialUnderlying(candidate.expression, candidate.named)
			if crossedOfficialTarget {
				continue
			}
			structure, ok := resolved.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range structure.Fields.List {
				matched := false
				for _, fieldName := range field.Names {
					if fieldName.Name == name {
						matches = append(matches, c3OfficialFieldMatch{expression: field.Type, named: current})
						matched = true
					}
				}
				if len(field.Names) == 0 {
					if c3OfficialEmbeddedFieldName(field.Type) == name {
						matches = append(matches, c3OfficialFieldMatch{expression: field.Type, named: current})
						matched = true
					}
					if !matched {
						next = append(next, c3OfficialFieldCandidate{expression: field.Type, named: current, seen: seen})
					}
				}
			}
		}
		if len(matches) == 1 {
			return matches[0].expression, matches[0].named, true
		}
		if len(matches) > 1 {
			// Equal-depth promotion is not selectable Go; remain conservative if any ambiguous branch bears authority.
			for _, match := range matches {
				if c3OfficialResultType(match.expression, match.named, make(map[string]bool)) {
					return match.expression, match.named, true
				}
			}
			return matches[0].expression, matches[0].named, true
		}
		frontier = next
	}
	return nil, c3OfficialNamedTypes{}, false
}

func c3OfficialSelectedFieldType(
	selector *ast.SelectorExpr,
	named c3OfficialNamedTypes,
	values map[string]c3OfficialValueDeclaration,
	visiting map[string]bool,
) (ast.Expr, c3OfficialNamedTypes, bool) {
	base, baseTypes, ok := c3OfficialExpressionType(selector.X, named, values, visiting)
	if !ok {
		return nil, c3OfficialNamedTypes{}, false
	}
	return c3OfficialFieldType(base, selector.Sel.Name, baseTypes)
}

func c3OfficialTargetBearingExpression(
	expression ast.Expr,
	named c3OfficialNamedTypes,
	values map[string]c3OfficialValueDeclaration,
	visiting map[string]bool,
) bool {
	switch value := expression.(type) {
	case *ast.CompositeLit:
		return value.Type != nil && c3OfficialResultType(value.Type, named, make(map[string]bool))
	case *ast.ParenExpr:
		return c3OfficialTargetBearingExpression(value.X, named, values, visiting)
	case *ast.UnaryExpr:
		return c3OfficialTargetBearingExpression(value.X, named, values, visiting)
	case *ast.TypeAssertExpr:
		return value.Type != nil && c3OfficialResultType(value.Type, named, make(map[string]bool))
	case *ast.CallExpr:
		if c3OfficialCallResultType(value.Fun, named, values, visiting) {
			return true
		}
		if identifier, ok := c3OfficialUnparen(value.Fun).(*ast.Ident); ok && len(value.Args) > 0 {
			switch identifier.Name {
			case "make", "new":
				return c3OfficialResultType(value.Args[0], named, make(map[string]bool))
			case "append":
				return c3OfficialTargetBearingExpression(value.Args[0], named, values, visiting)
			}
		}
	case *ast.SelectorExpr:
		if result, recognized := c3OfficialMethodProducesTarget(value, named, values, visiting); recognized {
			return result
		}
		fieldType, fieldTypes, ok := c3OfficialSelectedFieldType(value, named, values, visiting)
		return ok && c3OfficialResultType(fieldType, fieldTypes, make(map[string]bool))
	case *ast.IndexExpr:
		elementType, elementTypes, ok := c3OfficialIndexedElementType(value, named, values, visiting)
		if ok && c3OfficialResultType(elementType, elementTypes, make(map[string]bool)) {
			return true
		}
		return c3OfficialFactoryType(value, named, make(map[string]bool))
	case *ast.IndexListExpr:
		return c3OfficialFactoryType(value, named, make(map[string]bool))
	case *ast.Ident:
		declaration, ok := values[value.Name]
		if !ok || visiting[value.Name] {
			return false
		}
		visiting[value.Name] = true
		defer delete(visiting, value.Name)
		declarationTypes := c3OfficialNamedTypes{
			context: declaration.context, declarations: named.declarations, functions: named.functions,
		}
		if declaration.explicitType != nil && c3OfficialResultType(declaration.explicitType, declarationTypes, make(map[string]bool)) {
			return true
		}
		if call, ok := declaration.initializer.(*ast.CallExpr); ok && declaration.tupleWidth > 1 {
			if result, resultTypes, found := c3OfficialCallResultAt(
				call.Fun, declaration.tupleIndex, declarationTypes, values, visiting,
			); found {
				return c3OfficialResultType(result, resultTypes, make(map[string]bool))
			}
		}
		return declaration.initializer != nil &&
			c3OfficialTargetBearingExpression(declaration.initializer, declarationTypes, values, visiting)
	}
	return false
}

func c3OfficialIssuedExpression(expression ast.Expr, named c3OfficialNamedTypes) bool {
	return c3OfficialTargetBearingExpression(expression, named, nil, make(map[string]bool))
}

func c3OfficialFunctionTypeViolation(function *ast.FuncType, named c3OfficialNamedTypes) string {
	if function.Params != nil {
		for _, parameter := range function.Params.List {
			if c3OfficialFactoryType(parameter.Type, named, make(map[string]bool)) {
				return "function factory input"
			}
		}
	}
	if function.Results != nil {
		for _, result := range function.Results.List {
			if c3OfficialResultType(result.Type, named, make(map[string]bool)) {
				return "function issuance"
			}
		}
	}
	return ""
}

func c3OfficialWithLocalType(named c3OfficialNamedTypes, typeSpec *ast.TypeSpec) c3OfficialNamedTypes {
	declarations := c3OfficialCloneDeclarations(named.declarations, 1)
	declarations[c3OfficialTypeKey(named.context.packagePath, typeSpec.Name.Name)] = c3OfficialTypeDeclaration{
		expression: typeSpec.Type, context: named.context, typeParams: typeSpec.TypeParams,
		alias: typeSpec.Assign.IsValid(),
	}
	return c3OfficialNamedTypes{context: named.context, declarations: declarations, functions: named.functions}
}

func c3OfficialLocalTypeViolation(typeSpec *ast.TypeSpec, named c3OfficialNamedTypes) string {
	scoped := c3OfficialWithTypeParams(named, typeSpec.TypeParams)
	if c3OfficialResultType(typeSpec.Type, scoped, make(map[string]bool)) {
		return "function-local target-bearing type declaration"
	}
	if c3OfficialDeclaredType(typeSpec.Type, scoped) {
		return "function-local issuer type declaration"
	}
	return ""
}

func c3OfficialBodyViolation(
	body *ast.BlockStmt,
	named c3OfficialNamedTypes,
	packageValues map[string]c3OfficialValueDeclaration,
) string {
	if body == nil {
		return ""
	}
	for _, statement := range body.List {
		if declarationStatement, ok := statement.(*ast.DeclStmt); ok {
			if general, ok := declarationStatement.Decl.(*ast.GenDecl); ok && general.Tok == token.TYPE {
				for _, specification := range general.Specs {
					typeSpec, ok := specification.(*ast.TypeSpec)
					if !ok {
						continue
					}
					named = c3OfficialWithLocalType(named, typeSpec)
					if violation := c3OfficialLocalTypeViolation(typeSpec, named); violation != "" {
						return violation
					}
				}
				continue
			}
		}
		var violation string
		ast.Inspect(statement, func(node ast.Node) bool {
			if violation != "" {
				return false
			}
			switch value := node.(type) {
			case *ast.BlockStmt:
				violation = c3OfficialBodyViolation(value, named, packageValues)
				return false
			case *ast.CaseClause:
				violation = c3OfficialBodyViolation(&ast.BlockStmt{List: value.Body}, named, packageValues)
				return false
			case *ast.CommClause:
				violation = c3OfficialBodyViolation(&ast.BlockStmt{List: value.Body}, named, packageValues)
				return false
			case *ast.CompositeLit:
				if value.Type != nil && c3OfficialNonHolderComposite(value.Type, named) {
					violation = "composite issuance"
				}
			case *ast.FuncLit:
				scoped := c3OfficialWithTypeParams(named, value.Type.TypeParams)
				if violation = c3OfficialFunctionTypeViolation(value.Type, scoped); violation == "" {
					violation = c3OfficialBodyViolation(value.Body, scoped, packageValues)
				}
				return false
			case *ast.ValueSpec:
				if value.Type != nil && c3OfficialFactoryType(value.Type, named, make(map[string]bool)) {
					violation = "function-value issuance"
				}
				for _, initializer := range value.Values {
					if c3OfficialFactoryType(initializer, named, make(map[string]bool)) ||
						c3OfficialMethodFactoryExpression(initializer, named, packageValues) {
						violation = "inferred function-value issuance"
						break
					}
				}
			case *ast.ReturnStmt:
				for _, result := range value.Results {
					if c3OfficialTargetBearingExpression(result, named, packageValues, make(map[string]bool)) {
						violation = "constructed return issuance"
						break
					}
				}
			}
			return true
		})
		if violation != "" {
			return violation
		}
	}
	return ""
}

func c3OfficialIssuerViolation(
	candidate *ast.File,
	named c3OfficialNamedTypes,
	suppliedValues ...map[string]c3OfficialValueDeclaration,
) string {
	topLevelTypes := make(map[*ast.TypeSpec]bool)
	for _, declaration := range candidate.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, specification := range general.Specs {
			if typeSpec, ok := specification.(*ast.TypeSpec); ok {
				topLevelTypes[typeSpec] = true
			}
		}
	}
	packageValues := c3OfficialPackageValues(candidate, named)
	if len(suppliedValues) > 0 {
		packageValues = suppliedValues[0]
	}
	for _, declaration := range candidate.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, specification := range general.Specs {
			values, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, identifier := range values.Names {
				if !ast.IsExported(identifier.Name) {
					continue
				}
				if values.Type != nil && c3OfficialResultType(values.Type, named, make(map[string]bool)) {
					return "exported variable issuance"
				}
				valueDeclaration := packageValues[identifier.Name]
				if valueDeclaration.initializer != nil && c3OfficialTargetBearingExpression(
					identifier, named, packageValues, make(map[string]bool),
				) {
					return "exported variable issuance"
				}
			}
		}
	}
	var violation string
	ast.Inspect(candidate, func(node ast.Node) bool {
		if violation != "" {
			return false
		}
		switch value := node.(type) {
		case *ast.TypeSpec:
			scoped := c3OfficialWithTypeParams(named, value.TypeParams)
			if !topLevelTypes[value] && c3OfficialResultType(value.Type, scoped, make(map[string]bool)) {
				violation = "function-local target-bearing type declaration"
			} else if c3OfficialDeclaredType(value.Type, scoped) {
				violation = "type alias or declaration"
			}
		case *ast.CompositeLit:
			if value.Type != nil && c3OfficialNonHolderComposite(value.Type, named) {
				violation = "composite issuance"
			}
		case *ast.FuncDecl:
			scoped := c3OfficialWithReceiverTypeParams(named, value.Recv)
			scoped = c3OfficialWithTypeParams(scoped, value.Type.TypeParams)
			if violation = c3OfficialFunctionTypeViolation(value.Type, scoped); violation == "" {
				violation = c3OfficialBodyViolation(value.Body, scoped, packageValues)
			}
			return false
		case *ast.FuncLit:
			scoped := c3OfficialWithTypeParams(named, value.Type.TypeParams)
			if violation = c3OfficialFunctionTypeViolation(value.Type, scoped); violation == "" {
				violation = c3OfficialBodyViolation(value.Body, scoped, packageValues)
			}
			return false
		case *ast.ValueSpec:
			if value.Type != nil && c3OfficialFactoryType(value.Type, named, make(map[string]bool)) {
				violation = "function-value issuance"
			}
			for _, initializer := range value.Values {
				if c3OfficialFactoryType(initializer, named, make(map[string]bool)) ||
					c3OfficialMethodFactoryExpression(initializer, named, packageValues) {
					violation = "inferred function-value issuance"
					break
				}
			}
		case *ast.ReturnStmt:
			for _, result := range value.Results {
				if c3OfficialTargetBearingExpression(result, named, packageValues, make(map[string]bool)) {
					violation = "constructed return issuance"
					break
				}
			}
		}
		return true
	})
	return violation
}

func c3OfficialExportedSurface(t *testing.T, parsed *ast.File) []string {
	t.Helper()
	var surface []string
	for _, declaration := range parsed.Decls {
		switch value := declaration.(type) {
		case *ast.FuncDecl:
			if !ast.IsExported(value.Name.Name) {
				continue
			}
			label := value.Name.Name
			if value.Recv != nil {
				receiver := c3OfficialReceiver(value.Recv.List[0].Type)
				if !ast.IsExported(receiver) {
					continue
				}
				label = receiver + "." + label
			}
			surface = append(surface, label)
		case *ast.GenDecl:
			for _, specification := range value.Specs {
				switch item := specification.(type) {
				case *ast.TypeSpec:
					if !ast.IsExported(item.Name.Name) {
						continue
					}
					surface = append(surface, item.Name.Name)
					structure, ok := item.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, field := range structure.Fields.List {
						if len(field.Names) == 0 {
							var rendered strings.Builder
							if err := printer.Fprint(&rendered, token.NewFileSet(), field.Type); err != nil {
								t.Fatal(err)
							}
							surface = append(surface, item.Name.Name+".embed("+rendered.String()+")")
							continue
						}
						for _, name := range field.Names {
							if ast.IsExported(name.Name) {
								surface = append(surface, item.Name.Name+"."+name.Name)
							}
						}
					}
				case *ast.ValueSpec:
					for _, identifier := range item.Names {
						if ast.IsExported(identifier.Name) {
							surface = append(surface, identifier.Name)
						}
					}
				}
			}
		}
	}
	return surface
}

func c3ConcurrencyFixtureInventory(t *testing.T, root string) {
	t.Helper()
	packageDirectories := []string{
		"internal/contractexec", "internal/gitobj", "internal/hostepoch", "internal/noderuntime", "internal/store",
	}
	got := make(map[string]int)
	var parallelTests []string
	var productionGoStatements []string
	for _, relativeDirectory := range packageDirectories {
		directory := filepath.Join(root, filepath.FromSlash(relativeDirectory))
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				continue
			}
			path := filepath.Join(directory, entry.Name())
			fileSet := token.NewFileSet()
			parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
			if err != nil {
				t.Fatalf("parse C3 concurrency inventory %s: %v", path, err)
			}
			if !strings.HasSuffix(entry.Name(), "_test.go") {
				ast.Inspect(parsed, func(node ast.Node) bool {
					statement, ok := node.(*ast.GoStmt)
					if ok {
						position := fileSet.Position(statement.Go)
						productionGoStatements = append(productionGoStatements,
							filepath.ToSlash(filepath.Join(relativeDirectory, entry.Name()))+":"+strconv.Itoa(position.Line))
					}
					return true
				})
				continue
			}
			for _, declaration := range parsed.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Body == nil || !strings.HasPrefix(function.Name.Name, "TestC3") {
					continue
				}
				key := relativeDirectory + "." + function.Name.Name
				goStatements := 0
				ast.Inspect(function.Body, func(node ast.Node) bool {
					switch value := node.(type) {
					case *ast.GoStmt:
						goStatements++
					case *ast.CallExpr:
						selector, selected := value.Fun.(*ast.SelectorExpr)
						if selected && selector.Sel.Name == "Parallel" && len(value.Args) == 0 {
							parallelTests = append(parallelTests, key)
						}
					}
					return true
				})
				if goStatements > 0 {
					got[key] = goStatements
				}
			}
		}
	}
	want := map[string]int{
		"internal/contractexec.TestC3OfficialTargetClosedCapabilityAndDefensiveGetters":   2,
		"internal/hostepoch.TestC3HostEpochConcurrentMeasurementsNeverCache":              1,
		"internal/noderuntime.TestC3NodeRuntimeCopiedCapabilitiesSerializeRevalidation":   1,
		"internal/store.TestC3ConformanceAttemptConcurrentValidationAndReopenAreRaceFree": 3,
	}
	sort.Strings(parallelTests)
	sort.Strings(productionGoStatements)
	if !reflect.DeepEqual(got, want) || len(parallelTests) != 0 || len(productionGoStatements) != 0 {
		t.Fatalf("C3 focused race inventory changed: got=%v want=%v t.Parallel=%v production-go=%v",
			got, want, parallelTests, productionGoStatements)
	}
}

func TestC3OfficialTargetPublicSurfaceAndSoleIssuer(t *testing.T) {
	root := c3RepoRoot(t)
	c3ConcurrencyFixtureInventory(t, root)
	targetPath := filepath.Join(root, "internal", "contractexec", "target.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), targetPath, nil, parser.SkipObjectResolution)
	if err != nil || parsed.Name.Name != "contractexec" {
		t.Fatalf("official-target issuer did not parse exactly: %v", err)
	}
	got := c3OfficialExportedSurface(t, parsed)
	sort.Strings(got)
	want := []string{
		"CodeInvalidOfficialTarget", "CodeOfficialTargetChanged", "CodeOfficialTargetClosed",
		"Error", "Error.Cause", "Error.Code", "Error.Detail", "Error.Error", "Error.Unwrap", "IsCode",
		"OpenOfficialTarget", "OpenOfficialTargetRequest", "OpenOfficialTargetRequest.Repository",
		"OpenOfficialTargetRequest.Residue", "OpenOfficialTargetRequest.Store",
		"OpenOfficialTargetRequest.TargetDigest", "OfficialTarget", "OfficialTarget.CandidateRoot", "OfficialTarget.Close",
		"OfficialTarget.ContractBundle", "OfficialTarget.Digest", "OfficialTarget.Model", "OfficialTarget.Roots",
		"OfficialTarget.TargetRecord", "OfficialTarget.Valid", "PublishOfficialTarget", "PublishOfficialTargetRequest",
		"PublishOfficialTargetRequest.DisplayRef",
		"PublishOfficialTargetRequest.NodeExecutable", "PublishOfficialTargetRequest.Repository",
		"PublishOfficialTargetRequest.Residue", "PublishOfficialTargetRequest.Store", "ReopenOfficialTarget",
	}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("OfficialTarget public surface changed: got %v want %v", got, want)
	}
	embedded, err := parser.ParseFile(token.NewFileSet(), "embedded.go", "package contractexec\nimport \"io\"\ntype Request struct{ io.Reader }\n", parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	embeddedSurface := c3OfficialExportedSurface(t, embedded)
	sort.Strings(embeddedSurface)
	if !reflect.DeepEqual(embeddedSurface, []string{"Request", "Request.embed(io.Reader)"}) {
		t.Fatalf("anonymous exported field escaped the frozen surface inventory: %v", embeddedSurface)
	}
	for name, source := range map[string]string{
		"slice result":        "package contractexec\nfunc leak() []OfficialTarget { return nil }\n",
		"map result":          "package contractexec\nfunc leak() map[string]*OfficialTarget { return nil }\n",
		"channel result":      "package contractexec\nfunc leak() <-chan OfficialTarget { return nil }\n",
		"generic result":      "package contractexec\ntype Box[T any] struct{ Value T }\nfunc leak() Box[OfficialTarget] { return Box[OfficialTarget]{} }\n",
		"generic pair result": "package contractexec\ntype Pair[A, B any] struct{ Left A; Right B }\nfunc leak() Pair[string, OfficialTarget] { return Pair[string, OfficialTarget]{} }\n",
		"nested repeated generic result": "package contractexec\ntype Pair[A, B any] struct{ Left A; Right B }\n" +
			"func leak() Pair[Pair[string, string], Pair[string, OfficialTarget]] { panic(0) }\n",
		"nested rebound generic result": "package contractexec\ntype Pair[A, B any] struct{ Left A; Right B }\n" +
			"type Wrap[T any] Pair[T, T]\nfunc leak() Wrap[Wrap[OfficialTarget]] { panic(0) }\n",
		"struct result":          "package contractexec\nfunc leak() struct{ Target OfficialTarget } { panic(0) }\n",
		"interface result":       "package contractexec\ntype Factory interface{ Open() OfficialTarget }\n",
		"nested func result":     "package contractexec\nfunc leak() func() OfficialTarget { return nil }\n",
		"function literal":       "package contractexec\nvar leak = func() OfficialTarget { return OfficialTarget{} }\n",
		"function value":         "package contractexec\nvar leak func() OfficialTarget\n",
		"collection alias":       "package contractexec\ntype Batch []OfficialTarget\n",
		"wrapped composite":      "package contractexec\nvar leak = []OfficialTarget{}\n",
		"named wrapped result":   "package contractexec\ntype Result struct{ Target OfficialTarget }\nfunc leak() Result { return Result{} }\n",
		"constructed any result": "package contractexec\ntype Result struct{ Target OfficialTarget }\nfunc leak() any { return Result{} }\n",
		"struct-held factory":    "package contractexec\ntype FactoryHolder struct{ Open func() OfficialTarget }\n",
		"nested named factory":   "package contractexec\ntype Maker func() OfficialTarget\ntype FactoryHolder struct{ Open Maker }\n",
		"exported direct holder": "package contractexec\nvar Leak OfficialTarget\n",
		"exported inferred holder": "package contractexec\ntype RunRequest struct{ Target OfficialTarget }\n" +
			"var Leak = RunRequest{}\n",
		"exported erased holder": "package contractexec\ntype RunRequest struct{ Target OfficialTarget }\n" +
			"var Leak any = RunRequest{}\n",
		"exported converted holder": "package contractexec\nvar Leak = []OfficialTarget(nil)\n",
		"exported made holder":      "package contractexec\nvar Leak = make([]OfficialTarget, 0)\n",
		"exported new target":       "package contractexec\nvar Leak = new(OfficialTarget)\n",
		"parenthesized made holder": "package contractexec\nvar Leak = (make)([]OfficialTarget, 0)\n",
		"parenthesized new target":  "package contractexec\nvar Leak = (new)(OfficialTarget)\n",
		"parenthesized conversion":  "package contractexec\nvar Leak = ([]OfficialTarget)(nil)\n",
		"parenthesized append": "package contractexec\nvar hidden []OfficialTarget\n" +
			"var Leak = (append)(hidden, OfficialTarget{})\n",
		"exported identifier holder": "package contractexec\ntype RunRequest struct{ Target OfficialTarget }\n" +
			"var hidden = RunRequest{}\nvar Leak = hidden\n",
		"exported selected holder": "package contractexec\ntype RunRequest struct{ Target OfficialTarget }\n" +
			"var hidden RunRequest\nvar Leak = hidden.Target\n",
		"exported nested selected holder": "package contractexec\ntype RunRequest struct{ Target OfficialTarget }\n" +
			"type Envelope struct{ Request RunRequest }\nvar hidden Envelope\nvar Leak = hidden.Request.Target\n",
		"exported promoted selected holder": "package contractexec\ntype RunRequest struct{ Target OfficialTarget }\n" +
			"type Envelope struct{ RunRequest }\nvar hidden Envelope\nvar Leak = hidden.Target\n",
		"exported pointer selected holder": "package contractexec\ntype RunRequest struct{ Target OfficialTarget }\n" +
			"var hidden *RunRequest\nvar Leak = (*hidden).Target\n",
		"exported indexed holder":          "package contractexec\nvar hidden []OfficialTarget\nvar Leak = hidden[0]\n",
		"constrained generic result":       "package contractexec\nfunc leak[T interface{ OfficialTarget }](target T) T { return target }\n",
		"union-constrained generic result": "package contractexec\nfunc leak[T interface{ ~string | OfficialTarget }](target T) T { return target }\n",
		"constrained generic method result": "package contractexec\ntype Wrapper[T interface{ OfficialTarget }] struct{}\n" +
			"func (Wrapper[T]) leak(target T) T { return target }\n",
		"dependent receiver constraint result": "package contractexec\ntype Wrapper[A interface{ OfficialTarget }, B interface{ ~[]A }] struct{}\n" +
			"func (Wrapper[X, Y]) leak() Y { panic(0) }\n",
		"swapped receiver constraint result": "package contractexec\ntype Wrapper[A interface{ OfficialTarget }, B interface{ ~[]A }] struct{}\n" +
			"func (Wrapper[B, A]) leak() A { panic(0) }\n",
		"lexical generic construction": "package contractexec\nfunc leak[T interface{ OfficialTarget }](target T) any {\n" +
			"return []T{target}\n}\n",
		"lexical generic local wrapper": "package contractexec\nfunc leak[T interface{ OfficialTarget }](target T) any {\n" +
			"type Result struct{ Target T }; return Result{Target: target}\n}\n",
		"function-local wrapper": "package contractexec\nfunc leak(target OfficialTarget) any {\n" +
			"type Result struct{ Target OfficialTarget }; return Result{Target: target}\n}\n",
		"anonymous held factory": "package contractexec\nvar leak = struct{ Open func() OfficialTarget }{}\n",
		"instantiated held factory": "package contractexec\ntype Factory[T any] struct{ Open func() T }\n" +
			"var held Factory[OfficialTarget]\n",
		"instantiated factory input": "package contractexec\ntype Factory[T any] struct{ Open func() T }\n" +
			"func use(Factory[OfficialTarget]) {}\n",
		"instantiated generic producer call": "package contractexec\nfunc identity[T any](value T) T { return value }\n" +
			"var hidden OfficialTarget\nvar Leak any = identity[OfficialTarget](hidden)\n",
		"partially inferred generic producer call": "package contractexec\nfunc choose[A, T any](a A, target T) T { return target }\n" +
			"var hidden OfficialTarget\nvar Leak any = choose[string](\"\", hidden)\n",
		"fully inferred generic producer call": "package contractexec\nfunc identity[T any](value T) T { return value }\n" +
			"var hidden OfficialTarget\nvar Leak any = identity(hidden)\n",
		"instantiated generic producer value": "package contractexec\nfunc identity[T any](value T) T { return value }\n" +
			"var held = identity[OfficialTarget]\n",
		"instantiated generic method call": "package contractexec\ntype Box[T any] struct{ Value T }\n" +
			"func (box Box[T]) Get() T { return box.Value }\nvar hidden Box[OfficialTarget]\nvar Leak = hidden.Get()\n",
		"instantiated generic method value": "package contractexec\ntype Box[T any] struct{ Value T }\n" +
			"func (box Box[T]) Get() T { return box.Value }\nvar hidden Box[OfficialTarget]\nvar held = hidden.Get\n",
		"instantiated generic method expression": "package contractexec\ntype Box[T any] struct{ Value T }\n" +
			"func (box Box[T]) Get() T { return box.Value }\nvar held = Box[OfficialTarget].Get\n",
		"promoted generic method call": "package contractexec\ntype Box[T any] struct{ Value T }\n" +
			"func (box Box[T]) Get() T { return box.Value }\ntype Outer struct{ Box[OfficialTarget] }\n" +
			"var hidden Outer\nvar Leak = hidden.Get()\n",
		"promoted generic method value": "package contractexec\ntype Box[T any] struct{ Value T }\n" +
			"func (box Box[T]) Get() T { return box.Value }\ntype Outer struct{ Box[OfficialTarget] }\n" +
			"var hidden Outer\nvar held = hidden.Get\n",
		"promoted generic method expression": "package contractexec\ntype Box[T any] struct{ Value T }\n" +
			"func (box Box[T]) Get() T { return box.Value }\ntype Outer struct{ Box[OfficialTarget] }\n" +
			"var held = Outer.Get\n",
		"aliased generic method call": "package contractexec\ntype Box[T any] struct{ Value T }\n" +
			"func (box Box[T]) Get() T { return box.Value }\ntype TargetBox = Box[OfficialTarget]\n" +
			"var hidden TargetBox\nvar Leak = hidden.Get()\n",
		"function-local generic held factory": "package contractexec\nfunc inspect() {\n" +
			"type Factory[T any] struct{ Open func() T }; var held Factory[OfficialTarget]; _ = held\n}\n",
		"pre-shadow local construction": "package contractexec\nfunc leak() any {\n" +
			"return OfficialTarget{}; type OfficialTarget struct{}\n}\n",
	} {
		t.Run("refuses "+name, func(t *testing.T) {
			fixture, parseErr := parser.ParseFile(token.NewFileSet(), name+".go", source, parser.SkipObjectResolution)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			if violation := c3OfficialIssuerViolation(fixture, c3OfficialTypeIndex(fixture)); violation == "" {
				t.Fatal("wrapped OfficialTarget issuer escaped the AST policy")
			}
		})
	}
	t.Run("refuses cross-package named wrapper", func(t *testing.T) {
		holder, parseErr := parser.ParseFile(token.NewFileSet(), "holder.go", `package holder
import contractexec "github.com/nelsonwerd/countershape/internal/contractexec"
type Result struct { Target contractexec.OfficialTarget }
`, parser.SkipObjectResolution)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		facade, parseErr := parser.ParseFile(token.NewFileSet(), "facade.go", `package facade
import "github.com/nelsonwerd/countershape/internal/holderapi"
func Publish() holder.Result { return holder.Result{} }
`, parser.SkipObjectResolution)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		declarations := make(map[string]c3OfficialTypeDeclaration)
		holderPath := c3ModulePath + "/internal/holderapi"
		holderContext := c3OfficialContext(holderPath, holder)
		facadeContext := c3OfficialContextWithPackageNames(
			c3ModulePath+"/internal/facade", facade, map[string]string{holderPath: "holder"},
		)
		c3OfficialAddTypes(declarations, holderContext, holder)
		c3OfficialAddTypes(declarations, facadeContext, facade)
		if violation := c3OfficialIssuerViolation(facade, c3OfficialNamedTypes{
			context: facadeContext, declarations: declarations,
		}); violation == "" {
			t.Fatal("cross-package OfficialTarget wrapper escaped the module type graph")
		}
	})
	t.Run("refuses cross-file exported identifier", func(t *testing.T) {
		holder, parseErr := parser.ParseFile(token.NewFileSet(), "holder.go", `package contractexec
type RunRequest struct { Target OfficialTarget }
var hidden = RunRequest{}
`, parser.SkipObjectResolution)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		exported, parseErr := parser.ParseFile(token.NewFileSet(), "exported.go", `package contractexec
var Leak = hidden
`, parser.SkipObjectResolution)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		named := c3OfficialTypeIndex(holder, exported)
		values := make(map[string]c3OfficialValueDeclaration)
		c3OfficialAddValues(values, named.context, holder)
		c3OfficialAddValues(values, named.context, exported)
		if violation := c3OfficialIssuerViolation(exported, named, values); violation == "" {
			t.Fatal("cross-file target-bearing exported identifier escaped the package value graph")
		}
	})
	t.Run("refuses target through multiple dot imports", func(t *testing.T) {
		fixture, parseErr := parser.ParseFile(token.NewFileSet(), "dot.go", `package facade
import (
	. "github.com/nelsonwerd/countershape/internal/contractexec"
	. "github.com/nelsonwerd/countershape/internal/other"
)
func leak() OfficialTarget { panic(0) }
`, parser.SkipObjectResolution)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		context := c3OfficialContext(c3ModulePath+"/internal/facade", fixture)
		if violation := c3OfficialIssuerViolation(fixture, c3OfficialNamedTypes{
			context: context, declarations: make(map[string]c3OfficialTypeDeclaration),
		}); violation == "" {
			t.Fatal("OfficialTarget escaped through the non-last dot import")
		}
	})
	t.Run("tracks module function tuple results by position", func(t *testing.T) {
		issuer, parseErr := parser.ParseFile(token.NewFileSet(), "issuer.go", `package contractexec
func publish() (OfficialTarget, error) { panic(0) }
`, parser.SkipObjectResolution)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		functions := make(map[string]c3OfficialFunctionDeclaration)
		c3OfficialAddFunctions(functions, c3OfficialContext(c3ContractPackagePath, issuer), issuer)
		for name, source := range map[string]string{
			"target result":               "package contractexec\nvar Leak, _ = publish()\n",
			"parenthesized target result": "package contractexec\nvar Leak, _ = (publish)()\n",
			"error result":                "package contractexec\nvar _, Err = publish()\n",
		} {
			t.Run(name, func(t *testing.T) {
				fixture, err := parser.ParseFile(token.NewFileSet(), name+".go", source, parser.SkipObjectResolution)
				if err != nil {
					t.Fatal(err)
				}
				context := c3OfficialContext(c3ContractPackagePath, fixture)
				violation := c3OfficialIssuerViolation(fixture, c3OfficialNamedTypes{
					context: context, declarations: make(map[string]c3OfficialTypeDeclaration), functions: functions,
				})
				if name != "error result" && violation == "" {
					t.Fatal("target-bearing tuple result escaped positional tracking")
				}
				if name == "error result" && violation != "" {
					t.Fatalf("safe tuple result was classified as an issuer: %s", violation)
				}
			})
		}
	})
	t.Run("tracks generic method tuple results by position", func(t *testing.T) {
		for name, exported := range map[string]string{
			"target result": "var Leak, _ = hidden.Pair()\n",
			"error result":  "var _, Err = hidden.Pair()\n",
		} {
			t.Run(name, func(t *testing.T) {
				fixture, parseErr := parser.ParseFile(token.NewFileSet(), name+".go", `package contractexec
type Box[T any] struct{}
func (Box[T]) Pair() (T, error) { panic(0) }
var hidden Box[OfficialTarget]
`+exported, parser.SkipObjectResolution)
				if parseErr != nil {
					t.Fatal(parseErr)
				}
				violation := c3OfficialIssuerViolation(fixture, c3OfficialTypeIndex(fixture))
				if name == "target result" && violation == "" {
					t.Fatal("target-bearing method tuple result escaped positional tracking")
				}
				if name == "error result" && violation != "" {
					t.Fatalf("safe method tuple result was classified as an issuer: %s", violation)
				}
			})
		}
	})
	for name, source := range map[string]string{
		"parameter":                "package contractexec\nfunc consume(OfficialTarget) {}\n",
		"request holder":           "package contractexec\ntype RunRequest struct{ Target OfficialTarget }\n",
		"consumer type":            "package contractexec\ntype Consumer func(OfficialTarget) error\n",
		"callback consumer holder": "package contractexec\ntype RunRequest struct{ Target OfficialTarget; Consume func(OfficialTarget) error }\n",
		"request composite":        "package contractexec\ntype RunRequest struct{ Target OfficialTarget }\nvar request = RunRequest{}\n",
		"request holder derivative": "package contractexec\ntype Request1 struct{ Target OfficialTarget }\n" +
			"type Request2 Request1\nvar request Request2\n",
		"generic callback consumer": "package contractexec\ntype Consumer[T any] struct{ Consume func(T) error }\n" +
			"var consumer = Consumer[OfficialTarget]{}\nfunc use(Consumer[OfficialTarget]) {}\n",
		"generic function consumer call": "package contractexec\nfunc consume[T any](T) error { return nil }\n" +
			"var hidden OfficialTarget\nvar Result = consume[OfficialTarget](hidden)\n",
		"generic function consumer value": "package contractexec\nfunc consume[T any](T) error { return nil }\n" +
			"var Consumer = consume[OfficialTarget]\n",
		"generic method consumer call": "package contractexec\ntype Box[T any] struct{}\n" +
			"func (box Box[T]) Consume(T) error { return nil }\nvar hidden Box[OfficialTarget]\n" +
			"var target OfficialTarget\nvar Result = hidden.Consume(target)\n",
		"promoted generic method consumer": "package contractexec\ntype Box[T any] struct{}\n" +
			"func (box Box[T]) Consume(T) error { return nil }\ntype Outer struct{ Box[OfficialTarget] }\n" +
			"var hidden Outer\nvar target OfficialTarget\nvar Result = hidden.Consume(target)\n",
		"shallower field blocks promoted method": "package contractexec\ntype Box[T any] struct{}\n" +
			"func (Box[T]) Get() T { var zero T; return zero }\ntype Outer struct { Box[OfficialTarget]; Get string }\n" +
			"var hidden Outer\nvar Safe = hidden.Get\n",
		"shadowed capability type parameter": "package contractexec\n" +
			"func echo[OfficialTarget any](value OfficialTarget) OfficialTarget { return value }\n",
		"ordered local capability shadow": "package contractexec\nfunc inspect() {\n" +
			"type OfficialTarget struct{}; _ = OfficialTarget{}\n}\n",
		"shallow safe promoted field": "package contractexec\ntype TargetDeep struct{ Target OfficialTarget }\n" +
			"type A struct{ TargetDeep }; type B struct{ Target string }; type Root struct{ A; B }\n" +
			"var hidden Root\nvar Leak = hidden.Target\n",
		"ordinary result": "package contractexec\nfunc inspect(OfficialTarget) error { return nil }\n",
	} {
		t.Run("allows "+name, func(t *testing.T) {
			fixture, parseErr := parser.ParseFile(token.NewFileSet(), name+".go", source, parser.SkipObjectResolution)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			if violation := c3OfficialIssuerViolation(fixture, c3OfficialTypeIndex(fixture)); violation != "" {
				t.Fatalf("legitimate OfficialTarget consumer was classified as an issuer: %s", violation)
			}
		})
	}
	type parsedProductionFile struct {
		path        string
		file        *ast.File
		packagePath string
		context     c3OfficialTypeContext
	}
	internalRoot := filepath.Join(root, "internal")
	var productionFiles []parsedProductionFile
	buildContext := build.Default
	buildContext.GOOS = runtime.GOOS
	buildContext.GOARCH = runtime.GOARCH
	buildContext.CgoEnabled = true
	err = filepath.WalkDir(internalRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		matched, matchErr := buildContext.MatchFile(filepath.Dir(path), entry.Name())
		if matchErr != nil {
			return matchErr
		}
		if !matched {
			return nil
		}
		candidate, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return parseErr
		}
		relativeDirectory, relativeErr := filepath.Rel(root, filepath.Dir(path))
		if relativeErr != nil || relativeDirectory == "." || strings.HasPrefix(relativeDirectory, ".."+string(filepath.Separator)) {
			return errors.New("production file escaped module root: " + path)
		}
		packagePath := c3ModulePath + "/" + filepath.ToSlash(relativeDirectory)
		productionFiles = append(productionFiles, parsedProductionFile{
			path: path, file: candidate, packagePath: packagePath,
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	packageNames := make(map[string]string)
	for _, candidate := range productionFiles {
		declaredName := candidate.file.Name.Name
		if prior := packageNames[candidate.packagePath]; prior != "" && prior != declaredName {
			t.Fatalf("one import path declared multiple package names: %s: %s / %s", candidate.packagePath, prior, declaredName)
		}
		packageNames[candidate.packagePath] = declaredName
	}
	for index := range productionFiles {
		candidate := &productionFiles[index]
		candidate.context = c3OfficialContextWithPackageNames(candidate.packagePath, candidate.file, packageNames)
	}
	declarations := make(map[string]c3OfficialTypeDeclaration)
	functions := make(map[string]c3OfficialFunctionDeclaration)
	packageValues := make(map[string]map[string]c3OfficialValueDeclaration)
	for _, candidate := range productionFiles {
		c3OfficialAddTypes(declarations, candidate.context, candidate.file)
		c3OfficialAddFunctions(functions, candidate.context, candidate.file)
		values := packageValues[candidate.packagePath]
		if values == nil {
			values = make(map[string]c3OfficialValueDeclaration)
			packageValues[candidate.packagePath] = values
		}
		c3OfficialAddValues(values, candidate.context, candidate.file)
	}
	for _, candidate := range productionFiles {
		if candidate.path == targetPath {
			continue
		}
		named := c3OfficialNamedTypes{
			context: candidate.context, declarations: declarations, functions: functions,
		}
		if violation := c3OfficialIssuerViolation(candidate.file, named, packageValues[candidate.packagePath]); violation != "" {
			t.Fatal(errors.New("OfficialTarget " + violation + " escaped sole issuer into " + candidate.path))
		}
	}
}

func TestC3OfficialTargetAttemptsAreFreshAndDistinct(t *testing.T) {
	residue := c3NewResidue(t, "fresh")
	git := c3NewGit(t)
	first := c3Publish(t, residue, git)
	second := c3Publish(t, residue, git)
	if first.Digest() == second.Digest() || first.TargetRecord().AttemptDigest() == second.TargetRecord().AttemptDigest() ||
		first.Roots().AttemptRoot() == second.Roots().AttemptRoot() {
		t.Fatal("two high-level publications reused target or attempt identity")
	}
}

func TestC3OfficialTargetClosedCapabilityAndDefensiveGetters(t *testing.T) {
	residue := c3NewResidue(t, "close")
	git := c3NewGit(t)
	target := c3Publish(t, residue, git)
	copyOfTarget := target
	openerEntered := make(chan struct{})
	releaseOpener := make(chan struct{})
	openerSentinel := errors.New("controlled reopen completion")
	reopenDone := make(chan error, 1)
	go func() {
		_, err := reopenOfficialTarget(context.Background(), copyOfTarget, func(context.Context, OpenOfficialTargetRequest) (OfficialTarget, error) {
			if copyOfTarget.state.gate.TryLock() {
				copyOfTarget.state.gate.Unlock()
				return OfficialTarget{}, errors.New("reopen callback ran without the shared read gate")
			}
			close(openerEntered)
			<-releaseOpener
			return OfficialTarget{}, openerSentinel
		})
		reopenDone <- err
	}()
	<-openerEntered
	closeDone := make(chan error, 1)
	go func() { closeDone <- target.Close() }()
	close(releaseOpener)
	if err := <-reopenDone; !errors.Is(err, openerSentinel) {
		t.Fatalf("linearized reopen did not preserve its operation result: %v", err)
	}
	if err := <-closeDone; err != nil {
		t.Fatalf("close did not complete after the in-flight reopen: %v", err)
	}
	if target.Valid() || copyOfTarget.Valid() || target.Digest().Valid() || copyOfTarget.Model().Valid() ||
		copyOfTarget.TargetRecord().Valid() || copyOfTarget.ContractBundle().Valid() || copyOfTarget.CandidateRoot() != "" {
		t.Fatal("closing one capability copy did not revoke all copies and getters")
	}
	if reopened, err := ReopenOfficialTarget(context.Background(), copyOfTarget); !IsCode(err, CodeOfficialTargetClosed) || reopened.Valid() {
		t.Fatalf("closed capability reopened: %v", err)
	}
	if err := target.Close(); !IsCode(err, CodeOfficialTargetClosed) {
		t.Fatalf("second close did not refuse: %v", err)
	}
	if (OfficialTarget{}).Valid() || (OfficialTarget{state: &officialTargetState{seal: issuedOfficialTargetSeal}}).Valid() {
		t.Fatal("zero or forged OfficialTarget was valid")
	}
}

func TestC3OfficialTargetInvalidPreSpawnInputsReturnNoAuthority(t *testing.T) {
	residue := c3NewResidue(t, "faults")
	git := c3NewGit(t)
	missing := c3NewGitWithFiles(t, []gitrepo.File{{Path: "subject.mjs", Mode: "100644", Content: []byte("missing\n")}}, "missing entrypoint")
	wrongCase := c3NewGitWithFiles(t, []gitrepo.File{{Path: "Fixture.mjs", Mode: "100644", Content: []byte("wrong case\n")}}, "case-mismatched entrypoint")
	wrongPath := c3NewGitWithFiles(t, []gitrepo.File{{Path: "nested/fixture.mjs", Mode: "100644", Content: []byte("wrong path\n")}}, "path-mismatched entrypoint")
	requests := map[string]PublishOfficialTargetRequest{
		"zero residue":        {Store: residue.objectStore, Repository: git.repository, DisplayRef: git.ref, NodeExecutable: c3SystemNode},
		"moving name absent":  {Store: residue.objectStore, Residue: residue.residue, Repository: git.repository, DisplayRef: "refs/heads/absent", NodeExecutable: c3SystemNode},
		"relative runtime":    {Store: residue.objectStore, Residue: residue.residue, Repository: git.repository, DisplayRef: git.ref, NodeExecutable: "node"},
		"missing entrypoint":  {Store: residue.objectStore, Residue: residue.residue, Repository: missing.repository, DisplayRef: missing.ref, NodeExecutable: c3SystemNode},
		"wrong case":          {Store: residue.objectStore, Residue: residue.residue, Repository: wrongCase.repository, DisplayRef: wrongCase.ref, NodeExecutable: c3SystemNode},
		"wrong relative path": {Store: residue.objectStore, Residue: residue.residue, Repository: wrongPath.repository, DisplayRef: wrongPath.ref, NodeExecutable: c3SystemNode},
	}
	for name, request := range requests {
		t.Run(name, func(t *testing.T) {
			beforeInfo, beforeBytes := c3HeadSnapshot(t, residue)
			result, err := PublishOfficialTarget(context.Background(), request)
			afterInfo, afterBytes := c3HeadSnapshot(t, residue)
			if err == nil || result.Valid() || !os.SameFile(beforeInfo, afterInfo) || !bytes.Equal(beforeBytes, afterBytes) {
				t.Fatalf("invalid input issued authority or changed head: %v", err)
			}
		})
	}

	phases := []officialTargetFaultPhase{
		officialFaultAfterInputsValidated,
		officialFaultAfterInitialResidueReopen,
		officialFaultAfterPolicyDerivation,
		officialFaultAfterTreePin,
		officialFaultAfterAttemptAllocation,
		officialFaultAfterTreeInspection,
		officialFaultAfterEntrypointJoin,
		officialFaultAfterMaterialization,
		officialFaultAfterMaterializationReopen,
		officialFaultAfterRuntimeAdmission,
		officialFaultAfterRuntimeRevalidation,
		officialFaultAfterEpochMeasurement,
		officialFaultAfterTerminalResidueJoin,
		officialFaultAfterTargetBuild,
		officialFaultAfterTargetRecordPersistence,
		officialFaultAfterProvisionalValidation,
		officialFaultAfterFinalResidueJoin,
		officialFaultAfterFinalPolicyJoin,
		officialFaultAfterFinalMaterializationJoin,
		officialFaultAfterFinalRuntimeJoin,
		officialFaultAfterFinalEpochJoin,
		officialFaultAfterFinalAttemptJoin,
		officialFaultAfterFinalTargetRecordJoin,
		officialFaultBeforeAuthoritySeal,
	}
	phaseResidue := c3NewResidue(t, "phase-faults")
	phaseGit := c3NewGit(t)
	for stopIndex, stopPhase := range phases {
		t.Run("phase fault "+string(stopPhase), func(t *testing.T) {
			beforeInfo, beforeBytes := c3HeadSnapshot(t, phaseResidue)
			attemptDirectory := filepath.Join(
				phaseResidue.root, "private-captures", "contract-runs", "conformance-attempts",
			)
			targetDirectory := filepath.Join(
				phaseResidue.root, "contract-execution", "links", "target-by-attempt",
			)
			attemptsBefore := c3DirectoryEntryCount(t, attemptDirectory)
			targetsBefore := c3DirectoryEntryCount(t, targetDirectory)
			sentinel := errors.New("injected official-target phase failure")
			visited := make([]officialTargetFaultPhase, 0, stopIndex+1)
			result, err := publishOfficialTarget(context.Background(), PublishOfficialTargetRequest{
				Store: phaseResidue.objectStore, Residue: phaseResidue.residue, Repository: phaseGit.repository,
				DisplayRef: phaseGit.ref, NodeExecutable: c3SystemNode,
			}, func(phase officialTargetFaultPhase) error {
				visited = append(visited, phase)
				if phase == stopPhase {
					return sentinel
				}
				return nil
			})
			afterInfo, afterBytes := c3HeadSnapshot(t, phaseResidue)
			if !errors.Is(err, sentinel) || !IsCode(err, CodeInvalidOfficialTarget) || result.Valid() ||
				!reflect.DeepEqual(visited, phases[:stopIndex+1]) || !os.SameFile(beforeInfo, afterInfo) ||
				!bytes.Equal(beforeBytes, afterBytes) {
				t.Fatalf("phase fault issued authority, skipped a checkpoint, or changed head: phase=%s visited=%v err=%v", stopPhase, visited, err)
			}
			wantAttempts := 0
			if stopIndex >= 4 {
				wantAttempts = 1
			}
			attemptsAfter := c3DirectoryEntryCount(t, attemptDirectory)
			if attemptsAfter-attemptsBefore != wantAttempts {
				t.Fatalf("phase fault left an unclassified attempt delta: phase=%s before=%d after=%d want-delta=%d", stopPhase, attemptsBefore, attemptsAfter, wantAttempts)
			}
			wantTargets := 0
			if stopIndex >= 14 {
				wantTargets = 1
			}
			targetsAfter := c3DirectoryEntryCount(t, targetDirectory)
			if targetsAfter-targetsBefore != wantTargets {
				t.Fatalf("phase fault left an unclassified target relationship delta: phase=%s before=%d after=%d want-delta=%d", stopPhase, targetsBefore, targetsAfter, wantTargets)
			}
			if operations := c3DirectoryEntryCount(t, filepath.Join(phaseResidue.root, "contract-execution", "operations")); operations != 0 {
				t.Fatalf("pre-spawn phase fault created operational interlock state: phase=%s count=%d", stopPhase, operations)
			}
		})
	}
}

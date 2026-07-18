//go:build darwin

package node

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/emit/node/compiler"
	"github.com/nelsonwerd/countershape/internal/emit/node/internal/compilation"
	internalpublication "github.com/nelsonwerd/countershape/internal/emit/node/internal/publication"
	"github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
	"github.com/nelsonwerd/countershape/internal/store"
	contractfixtures "github.com/nelsonwerd/countershape/testkit/contracts"
)

type directResidueFixture struct {
	root        string
	objectStore *store.ObjectStore
	study       store.StudyID
	ruling      store.HeadToken
	bundle      model.ContractBundle
	authority   internalpublication.Authority
}

type directResidueHeadWire struct {
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

func TestObjectStoreAdvanceResidueDirectClosure(t *testing.T) {
	exact := newDirectResidueFixture(t, "exact")
	advanced, err := exact.objectStore.AdvanceResidue(context.Background(), exact.ruling, exact.authority)
	if err != nil || advanced.Stage() != store.StageResidue || advanced.Revision() != 8 ||
		advanced.CurrentKind() != "ContractBundle" || advanced.CurrentDigest() != exact.bundle.Digest() ||
		advanced.LineageRootDigest() != exact.ruling.LineageRootDigest() {
		t.Fatalf("exact direct residue transition = %#v, %v", advanced, err)
	}
	previousHead, hasPreviousHead := advanced.PreviousHeadDigest()
	previousObject, hasPreviousObject := advanced.PreviousObjectDigest()
	if !hasPreviousHead || previousHead != exact.ruling.HeadDigest() || !hasPreviousObject ||
		previousObject != exact.bundle.DecisionRecordDigest() {
		t.Fatalf("terminal predecessor identity = %s/%t, %s/%t", previousHead, hasPreviousHead, previousObject, hasPreviousObject)
	}
	object, objectAuthority, err := exact.objectStore.Read(
		context.Background(), "ContractBundle", exact.bundle.Digest(),
	)
	if err != nil || exact.objectStore.Validate(context.Background(), object, objectAuthority) != nil ||
		!bytes.Equal(object.CanonicalBytes(), exact.bundle.CanonicalBytes()) {
		t.Fatalf("terminal bundle object did not reopen exactly: %v", err)
	}
	assertDirectNoTemporaryEntries(t, exact.root)

	for _, test := range []struct {
		name     string
		wantCode string
		call     func(directResidueFixture) (store.HeadToken, error)
	}{
		{
			name: "zero-authority", wantCode: "ILLEGAL_LINEAGE",
			call: func(fixture directResidueFixture) (store.HeadToken, error) {
				return fixture.objectStore.AdvanceResidue(
					context.Background(), fixture.ruling, internalpublication.Authority{},
				)
			},
		},
		{
			name: "cancelled", wantCode: "HEAD_UPDATE_FAILED",
			call: func(fixture directResidueFixture) (store.HeadToken, error) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return fixture.objectStore.AdvanceResidue(ctx, fixture.ruling, fixture.authority)
			},
		},
		{
			name: "wrong-expected-head", wantCode: "ILLEGAL_LINEAGE",
			call: func(fixture directResidueFixture) (store.HeadToken, error) {
				wrong, issueErr := internalpublication.Issue(
					fixture.bundle, fixture.study.String(), directResidueDigest("e"),
					fixture.bundle.DecisionRecordDigest(), fixture.ruling.LineageRootDigest(),
				)
				if issueErr != nil || !wrong.Valid() {
					t.Fatalf("issue valid mismatched authority: %v", issueErr)
				}
				return fixture.objectStore.AdvanceResidue(context.Background(), fixture.ruling, wrong)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newDirectResidueFixture(t, test.name)
			before := directResidueTreeSnapshot(t, fixture.root)
			result, err := test.call(fixture)
			if result.HeadDigest().Valid() || !directResidueStoreCode(err, test.wantCode) {
				t.Fatalf("direct residue refusal = %#v, %v; want %s", result, err, test.wantCode)
			}
			after := directResidueTreeSnapshot(t, fixture.root)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("direct residue refusal changed store: before=%v after=%v", before, after)
			}
			if _, statErr := os.Lstat(directResidueObjectPath(fixture.root, fixture.bundle.Digest())); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("direct residue refusal left bundle object: %v", statErr)
			}
		})
	}

	t.Run("foreign-store-token", func(t *testing.T) {
		left := newDirectResidueFixture(t, "foreign-left")
		right := newDirectResidueFixture(t, "foreign-right")
		before := directResidueTreeSnapshot(t, right.root)
		result, err := right.objectStore.AdvanceResidue(context.Background(), left.ruling, left.authority)
		if result.HeadDigest().Valid() || !directResidueStoreCode(err, "ILLEGAL_LINEAGE") ||
			!reflect.DeepEqual(before, directResidueTreeSnapshot(t, right.root)) {
			t.Fatalf("foreign direct residue transition = %#v, %v", result, err)
		}
	})

	t.Run("stale-replay", func(t *testing.T) {
		fixture := newDirectResidueFixture(t, "stale-replay")
		if _, err := fixture.objectStore.AdvanceResidue(context.Background(), fixture.ruling, fixture.authority); err != nil {
			t.Fatal(err)
		}
		before := directResidueTreeSnapshot(t, fixture.root)
		result, err := fixture.objectStore.AdvanceResidue(context.Background(), fixture.ruling, fixture.authority)
		if result.HeadDigest().Valid() || !directResidueStoreCode(err, "CAS_CONFLICT") ||
			!reflect.DeepEqual(before, directResidueTreeSnapshot(t, fixture.root)) {
			t.Fatalf("stale direct residue replay = %#v, %v", result, err)
		}
		assertDirectNoTemporaryEntries(t, fixture.root)
	})
}

func TestObjectStoreConfirmResiduePublicationDirectClosure(t *testing.T) {
	fixture := newDirectResidueFixture(t, "confirm-exact")
	terminal, err := fixture.objectStore.AdvanceResidue(context.Background(), fixture.ruling, fixture.authority)
	if err != nil {
		t.Fatal(err)
	}
	before := directResidueTreeSnapshot(t, fixture.root)
	confirmed, err := fixture.objectStore.ConfirmResiduePublication(
		context.Background(), terminal, fixture.authority,
	)
	if err != nil || !directResidueSameHead(confirmed, terminal) ||
		!reflect.DeepEqual(before, directResidueTreeSnapshot(t, fixture.root)) {
		t.Fatalf("exact direct residue confirmation = %#v, %v", confirmed, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelled, err := fixture.objectStore.ConfirmResiduePublication(ctx, terminal, fixture.authority)
	if cancelled.HeadDigest().Valid() || !directResidueStoreCode(err, "HEAD_UPDATE_AMBIGUOUS") ||
		!reflect.DeepEqual(before, directResidueTreeSnapshot(t, fixture.root)) {
		t.Fatalf("cancelled direct residue confirmation = %#v, %v", cancelled, err)
	}
	invalid, err := fixture.objectStore.ConfirmResiduePublication(
		context.Background(), terminal, internalpublication.Authority{},
	)
	if invalid.HeadDigest().Valid() || !directResidueStoreCode(err, "ILLEGAL_LINEAGE") ||
		!reflect.DeepEqual(before, directResidueTreeSnapshot(t, fixture.root)) {
		t.Fatalf("invalid direct residue confirmation = %#v, %v", invalid, err)
	}

	for _, test := range []struct {
		name   string
		damage func(testing.TB, directResidueFixture, store.HeadToken)
	}{
		{
			name: "missing-bundle-object",
			damage: func(t testing.TB, fixture directResidueFixture, terminal store.HeadToken) {
				t.Helper()
				if err := os.Remove(directResidueObjectPath(fixture.root, terminal.CurrentDigest())); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "corrupt-head",
			damage: func(t testing.TB, fixture directResidueFixture, _ store.HeadToken) {
				t.Helper()
				if err := os.WriteFile(directResidueHeadPath(fixture.root, fixture.study), []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "replaced-studies-namespace",
			damage: func(t testing.TB, fixture directResidueFixture, _ store.HeadToken) {
				t.Helper()
				studies := filepath.Join(fixture.root, "studies")
				if err := os.Rename(studies, filepath.Join(fixture.root, "studies-retained")); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(studies, 0o700); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newDirectResidueFixture(t, "confirm-"+test.name)
			terminal, err := fixture.objectStore.AdvanceResidue(
				context.Background(), fixture.ruling, fixture.authority,
			)
			if err != nil {
				t.Fatal(err)
			}
			test.damage(t, fixture, terminal)
			before := directResidueTreeSnapshot(t, fixture.root)
			confirmed, err := fixture.objectStore.ConfirmResiduePublication(
				context.Background(), terminal, fixture.authority,
			)
			if confirmed.HeadDigest().Valid() || !directResidueStoreCode(err, "HEAD_UPDATE_AMBIGUOUS") {
				t.Fatalf("damaged direct residue confirmation = %#v, %v", confirmed, err)
			}
			after := directResidueTreeSnapshot(t, fixture.root)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("damaged confirmation performed further mutation: before=%v after=%v", before, after)
			}
		})
	}
}

func newDirectResidueFixture(t testing.TB, label string) directResidueFixture {
	t.Helper()
	rootParent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(rootParent, "store")
	objectStore, err := store.OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	source, err := contractfixtures.CLISource()
	if err != nil {
		t.Fatal(err)
	}
	choicepointBytes := directResidueExampleBytes(t, "choicepoint.valid.json")
	choicepoint, err := choice.ParseChoicepointRecord(choicepointBytes)
	if err != nil {
		t.Fatal(err)
	}
	decisionBytes := directResidueExampleBytes(t, "decision-record.valid.json")
	decision, err := choice.ParseDecisionRecord(decisionBytes, choicepoint)
	if err != nil {
		t.Fatal(err)
	}
	bundle := directResidueBundle(t, source, decision.Digest(), choicepoint.Digest())
	for _, object := range []struct {
		kind    string
		digest  domain.Digest
		content []byte
	}{
		{kind: "WorldPlan", digest: source.Plan().Digest(), content: source.Plan().CanonicalBytes()},
		{kind: "Choicepoint", digest: choicepoint.Digest(), content: choicepointBytes},
		{kind: "DecisionRecord", digest: decision.Digest(), content: decisionBytes},
	} {
		semantic, objectErr := store.NewSemanticObject(object.kind, object.digest, object.content)
		if objectErr != nil {
			t.Fatalf("create direct %s object: %v", object.kind, objectErr)
		}
		authority, publishErr := objectStore.Publish(context.Background(), semantic)
		if publishErr != nil || objectStore.Validate(context.Background(), semantic, authority) != nil {
			t.Fatalf("publish direct %s object: %v", object.kind, publishErr)
		}
	}
	study, err := store.NewStudyID("direct residue store closure " + label)
	if err != nil {
		t.Fatal(err)
	}
	headBody, err := canon.CanonicalizeTyped(directResidueHeadWire{
		SchemaVersion: domain.SchemaVersion, Kind: "StudyHead", StudyID: study.String(),
		Revision: 7, Stage: string(store.StageRuling), CurrentKind: "DecisionRecord",
		CurrentDigest: decision.Digest().String(), LineageRootDigest: source.Plan().Digest().String(),
		PreviousHeadDigest: directResidueDigest("a").String(), PreviousObjectDigest: choicepoint.Digest().String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	studyPath := filepath.Dir(directResidueHeadPath(root, study))
	if err := os.Mkdir(studyPath, 0o700); err != nil {
		t.Fatal(err)
	}
	lock, err := os.OpenFile(filepath.Join(studyPath, ".head.lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(directResidueHeadPath(root, study), headBody, 0o600); err != nil {
		t.Fatal(err)
	}
	ruling, err := objectStore.OpenHead(context.Background(), study)
	if err != nil || ruling.Stage() != store.StageRuling || ruling.Revision() != 7 {
		t.Fatalf("open direct ruling snapshot: %#v, %v", ruling, err)
	}
	authority, err := internalpublication.Issue(
		bundle, study.String(), ruling.HeadDigest(), decision.Digest(), ruling.LineageRootDigest(),
	)
	if err != nil || !authority.Valid() {
		t.Fatalf("issue direct residue authority: %#v, %v", authority, err)
	}
	return directResidueFixture{
		root: root, objectStore: objectStore, study: study, ruling: ruling, bundle: bundle, authority: authority,
	}
}

func directResidueBundle(
	t testing.TB,
	source contractsource.PortableSource,
	decision domain.Digest,
	choicepoint domain.Digest,
) model.ContractBundle {
	t.Helper()
	profile, err := model.NewSourceProfile(source)
	if err != nil {
		t.Fatal(err)
	}
	value, err := portablevalue.Bytes([]byte("ok\n"))
	if err != nil {
		t.Fatal(err)
	}
	exact, err := model.NewExactValue(value)
	if err != nil {
		t.Fatal(err)
	}
	field, err := model.NewExactField("cli.stdout.bytes", exact)
	if err != nil {
		t.Fatal(err)
	}
	tuple, err := model.NewExactTuple([]model.ExactField{field})
	if err != nil {
		t.Fatal(err)
	}
	predicate, err := model.NewPredicate(
		source.StimulusDigest(), source.Profile(), []string{"cli.stdout.bytes"}, []model.ExactTuple{tuple},
	)
	if err != nil {
		t.Fatal(err)
	}
	input, err := compilation.New(
		decision, choicepoint, compilation.ActionAllowObserved, source, profile, predicate,
	)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := compiler.Compile(input)
	if err != nil || !bundle.Valid() {
		t.Fatalf("compile direct residue bundle: %v", err)
	}
	return bundle
}

func directResidueExampleBytes(t testing.TB, name string) []byte {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("direct residue fixture path is unavailable")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, "spec", "examples", "v1", name))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 2 || raw[len(raw)-1] != '\n' || bytes.Contains(raw[:len(raw)-1], []byte{'\n'}) {
		t.Fatalf("%s is not one canonical line plus a packaging newline", name)
	}
	return append([]byte(nil), raw[:len(raw)-1]...)
}

func directResidueDigest(character string) domain.Digest {
	digest, _ := domain.ParseDigest("sha256:" + strings.Repeat(character, 64))
	return digest
}

func directResidueHeadPath(root string, study store.StudyID) string {
	return filepath.Join(root, "studies", strings.TrimPrefix(study.String(), "study:"), "head.json")
}

func directResidueObjectPath(root string, digest domain.Digest) string {
	hex := strings.TrimPrefix(digest.String(), "sha256:")
	return filepath.Join(root, "objects", "sha256", hex[:2], hex)
}

func directResidueStoreCode(err error, code string) bool {
	var refusal *store.Error
	return errors.As(err, &refusal) && refusal.Code == code
}

func directResidueSameHead(left, right store.HeadToken) bool {
	leftPreviousHead, leftHasPreviousHead := left.PreviousHeadDigest()
	rightPreviousHead, rightHasPreviousHead := right.PreviousHeadDigest()
	leftPreviousObject, leftHasPreviousObject := left.PreviousObjectDigest()
	rightPreviousObject, rightHasPreviousObject := right.PreviousObjectDigest()
	return left.StudyID().String() == right.StudyID().String() && left.Revision() == right.Revision() &&
		left.Stage() == right.Stage() && left.CurrentKind() == right.CurrentKind() &&
		left.CurrentDigest() == right.CurrentDigest() && left.LineageRootDigest() == right.LineageRootDigest() &&
		left.HeadDigest() == right.HeadDigest() && leftHasPreviousHead == rightHasPreviousHead &&
		leftPreviousHead == rightPreviousHead && leftHasPreviousObject == rightHasPreviousObject &&
		leftPreviousObject == rightPreviousObject
}

func directResidueTreeSnapshot(t testing.TB, root string) []string {
	t.Helper()
	entries := make([]string, 0)
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
		if entry.IsDir() {
			entries = append(entries, fmt.Sprintf("D\x00%s\x00%#o", relative, info.Mode().Perm()))
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(body)
		entries = append(entries, fmt.Sprintf(
			"F\x00%s\x00%#o\x00%d\x00%x", relative, info.Mode().Perm(), len(body), digest,
		))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(entries)
	return entries
}

func assertDirectNoTemporaryEntries(t testing.TB, root string) {
	t.Helper()
	for _, entry := range directResidueTreeSnapshot(t, root) {
		if strings.Contains(entry, ".object-") || strings.Contains(entry, ".head-") {
			t.Fatalf("direct residue store left a temporary entry: %q", entry)
		}
	}
}

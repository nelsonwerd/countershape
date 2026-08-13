package clistudy

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	countercli "github.com/nelsonwerd/countershape/internal/adapters/cli"
	climodel "github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/domain"
	emitmodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
)

func TestCLIStudyProductionAuthorityEdgesStayBounded(t *testing.T) {
	production := []string{"choice.go", "contract.go", "inspect.go", "reduction.go", "study.go"}
	goFiles, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	actualProduction := make([]string, 0, len(production))
	for _, name := range goFiles {
		if !strings.HasSuffix(name, "_test.go") {
			actualProduction = append(actualProduction, name)
		}
	}
	slices.Sort(actualProduction)
	if !slices.Equal(actualProduction, production) {
		t.Fatalf("CLI study production roster = %v, want exact governed five-file roster %v", actualProduction, production)
	}
	requiredStudyEdges := []string{
		"github.com/nelsonwerd/countershape/internal/adapters/cli",
		"github.com/nelsonwerd/countershape/internal/reference/app",
		"github.com/nelsonwerd/countershape/testkit/reference",
	}
	for _, name := range production {
		exact, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read governed production file %s: %v", name, err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), name, exact, 0)
		if err != nil {
			t.Fatalf("parse governed production file %s: %v", name, err)
		}
		imports := make([]string, 0, len(parsed.Imports))
		for _, imported := range parsed.Imports {
			imports = append(imports, strings.Trim(imported.Path.Value, `"`))
		}
		for _, imported := range imports {
			if imported == "github.com/nelsonwerd/countershape/internal/contractexec/"+"model" ||
				strings.Contains(imported, "/contractexec/"+"http") ||
				strings.Contains(imported, "/adapters/"+"http") ||
				strings.Contains(imported, "/reference/"+"httpstudy") ||
				(strings.Contains(imported, "/internal/reference/") && imported != "github.com/nelsonwerd/countershape/internal/reference/app") ||
				(strings.Contains(imported, "/testkit/") && imported != "github.com/nelsonwerd/countershape/testkit/reference") ||
				imported == "os/exec" || imported == "syscall" || imported == "unsafe" ||
				imported == "net" || strings.HasPrefix(imported, "net/") {
				t.Fatalf("governed CLI production file %s imports forbidden authority edge %q", name, imported)
			}
		}
		for _, forbidden := range []string{
			"Contract" + "Execution",
			"Finalized" + "ContractRun",
			"H" + "TTP",
			"go:" + "linkname",
		} {
			if strings.Contains(string(exact), forbidden) {
				t.Fatalf("governed CLI production file %s contains protected token %q", name, forbidden)
			}
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			forbidden := []string{"Contract" + "Execution", "Finalized" + "ContractRun", "H" + "TTP", "go:" + "linkname"}
			switch typed := node.(type) {
			case *ast.Ident:
				for _, protected := range forbidden {
					if strings.Contains(typed.Name, protected) {
						t.Fatalf("governed CLI production file %s contains protected identifier %q", name, typed.Name)
					}
				}
			case *ast.BinaryExpr:
				if assembled, ok := staticStringConcatenation(typed); ok {
					for _, protected := range forbidden {
						if strings.Contains(assembled, protected) {
							t.Fatalf("governed CLI production file %s assembles protected token %q", name, protected)
						}
					}
				}
			}
			return true
		})
		if name == "study.go" {
			for _, required := range requiredStudyEdges {
				if !slices.Contains(imports, required) {
					t.Fatalf("study.go lacks required typed edge %q: %v", required, imports)
				}
			}
		}
	}
}

func TestCLIStudyOfficialTrialDelegatesFreshnessToTheSealedRunner(t *testing.T) {
	exact, err := os.ReadFile("contract.go")
	if err != nil {
		t.Fatalf("read contract production source: %v", err)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), "contract.go", exact, 0)
	if err != nil {
		t.Fatalf("parse contract production source: %v", err)
	}
	var body *ast.BlockStmt
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == "executeOfficialTrial" {
			body = function.Body
			break
		}
	}
	if body == nil {
		t.Fatal("executeOfficialTrial production function is absent")
	}
	calls := make(map[string]int)
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		calls[selector.Sel.Name]++
		if selector.Sel.Name == "ExecuteCLI" || selector.Sel.Name == "ResumeCLIClassification" {
			if len(call.Args) != 2 {
				t.Fatalf("%s argument count = %d, want 2", selector.Sel.Name, len(call.Args))
			}
			retained, ok := call.Args[1].(*ast.Ident)
			if !ok || retained.Name != "published" {
				t.Fatalf("%s does not consume the sealed published target", selector.Sel.Name)
			}
		}
		return true
	})
	for name, want := range map[string]int{
		"PublishOfficialTarget":   0,
		"ExecuteCLI":              1,
		"OpenFinalizedRunRecord":  1,
		"ResumeCLIClassification": 1,
		"ReopenOfficialTarget":    0,
	} {
		if calls[name] != want {
			t.Fatalf("executeOfficialTrial %s call count = %d, want %d", name, calls[name], want)
		}
	}
}

func TestCLIStudyOfficialPublicationIsBoundedBeforeSequentialExecution(t *testing.T) {
	exact, err := os.ReadFile("contract.go")
	if err != nil {
		t.Fatalf("read contract production source: %v", err)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), "contract.go", exact, 0)
	if err != nil {
		t.Fatalf("parse contract production source: %v", err)
	}
	var body *ast.BlockStmt
	publicationOwners := make(map[string]int)
	legacyPublicationHelper := false
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if function.Name.Name == "executeOfficialTrials" {
			body = function.Body
		}
		legacyPublicationHelper = legacyPublicationHelper || function.Name.Name == "publishOfficialTargetRoster"
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if ok && selector.Sel.Name == "PublishOfficialTarget" {
				publicationOwners[function.Name.Name]++
			}
			return true
		})
	}
	if body == nil {
		t.Fatal("executeOfficialTrials production function is absent")
	}
	if legacyPublicationHelper {
		t.Fatal("target-returning publishOfficialTargetRoster helper remains")
	}
	if len(publicationOwners) != 1 || publicationOwners["executeOfficialTrials"] != 1 {
		t.Fatalf("official target publication owner topology = %v, want executeOfficialTrials:1", publicationOwners)
	}
	calls := make(map[string]int)
	var boundedPublication, directPublication, sequentialExecution *ast.CallExpr
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := ""
		switch function := call.Fun.(type) {
		case *ast.Ident:
			name = function.Name
		case *ast.SelectorExpr:
			name = function.Sel.Name
		}
		calls[name]++
		switch name {
		case "runOfficialRepositoryTasks":
			boundedPublication = call
		case "PublishOfficialTarget":
			directPublication = call
		case "executeOfficialTrial":
			sequentialExecution = call
		}
		return true
	})
	for name, want := range map[string]int{
		"runOfficialRepositoryTasks": 1,
		"PublishOfficialTarget":      1,
		"executeOfficialTrial":       1,
		"ExecuteCLI":                 0,
	} {
		if calls[name] != want {
			t.Fatalf("executeOfficialTrials %s call count = %d, want %d", name, calls[name], want)
		}
	}
	if !(boundedPublication.Pos() < directPublication.Pos() && directPublication.End() < boundedPublication.End()) {
		t.Fatal("official target publication is not enclosed by the bounded repository task")
	}
	if boundedPublication.End() >= sequentialExecution.Pos() {
		t.Fatal("official target publication barrier does not precede sequential execution")
	}
	sequentialRange := false
	ast.Inspect(body, func(node ast.Node) bool {
		loop, ok := node.(*ast.RangeStmt)
		if !ok {
			return true
		}
		roster, ok := loop.X.(*ast.Ident)
		if !ok || roster.Name != "published" {
			return true
		}
		ast.Inspect(loop.Body, func(child ast.Node) bool {
			if child == sequentialExecution {
				sequentialRange = true
			}
			return true
		})
		return true
	})
	if !sequentialRange {
		t.Fatal("executeOfficialTrial is not inside the sequential published-target roster loop")
	}
}

func staticStringConcatenation(expression ast.Expr) (string, bool) {
	switch typed := expression.(type) {
	case *ast.BasicLit:
		if typed.Kind != token.STRING {
			return "", false
		}
		value, err := strconv.Unquote(typed.Value)
		return value, err == nil
	case *ast.BinaryExpr:
		if typed.Op != token.ADD {
			return "", false
		}
		left, leftOK := staticStringConcatenation(typed.X)
		right, rightOK := staticStringConcatenation(typed.Y)
		return left + right, leftOK && rightOK
	default:
		return "", false
	}
}

func TestCLITraversalOverlayIsTypedConstructorRefusal(t *testing.T) {
	for _, path := range []string{
		"../escape.json",
		"nested/../../escape.json",
		"/absolute.json",
		`backslash\escape.json`,
		".git/config",
	} {
		fixture, err := countercli.NewFixtureFile(path, []byte("untrusted"), countercli.FixtureMode0644)
		var refusal *climodel.Refusal
		if !errors.As(err, &refusal) || refusal.Code != countercli.CodeFixturePath ||
			fixture.Path() != "" || len(fixture.Contents()) != 0 {
			t.Fatalf("overlay %q = fixture(%q,%d) refusal=%+v err=%v", path, fixture.Path(), len(fixture.Contents()), refusal, err)
		}
	}

	root := t.TempDir()
	before, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := countercli.NewFixtureFile(filepath.Join("..", filepath.Base(root), "escape"), []byte("x"), countercli.FixtureMode0644); err == nil {
		t.Fatal("traversal overlay unexpectedly constructed")
	}
	after, err := os.ReadDir(root)
	if err != nil || len(before) != 0 || len(after) != 0 {
		t.Fatalf("pure constructor refusal changed a filesystem root: before=%v after=%v err=%v", before, after, err)
	}
}

type choiceAuditProofTestFixture struct {
	record   choice.ChoicepointRecord
	decision choice.DecisionRecord
	bundle   emitmodel.ContractBundle
	audit    choiceAudit
}

func TestChoiceAuditProofBindsEveryRetainedInput(t *testing.T) {
	baseline := checkedChoiceAuditProofTestFixture(t)
	rebuilt, err := choiceAuditProof(baseline.record, baseline.decision, baseline.bundle, baseline.audit)
	if err != nil || rebuilt != baseline.audit.validationProof {
		t.Fatalf("rebuild valid choice audit proof: rebuilt=%s retained=%s err=%v",
			rebuilt, baseline.audit.validationProof, err)
	}
	if err := validateChoiceAuditProof(baseline.audit, baseline.record, baseline.decision, baseline.bundle); err != nil {
		t.Fatalf("validate unmodified choice audit proof: %v", err)
	}

	mutations := []struct {
		name   string
		mutate func(*choiceAuditProofTestFixture)
	}{
		{name: "audit decision digest", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.decisionDigest = choiceAuditProofTestDigest('b')
		}},
		{name: "early reveal scalar", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.earlyReveal = !value.audit.earlyReveal
		}},
		{name: "session state scalar", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.finalState = choice.SessionRevealed
		}},
		{name: "ambiguity scalar", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.ambiguityCode = choice.CodeInvalidSessionState
		}},
		{name: "noncompilable scalar", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.noncompilableCode += "-changed"
		}},
		{name: "tuple roster", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.allowedTupleBytes = append(value.audit.allowedTupleBytes, []byte("third-tuple"))
		}},
		{name: "nested tuple byte", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.allowedTupleBytes[0][0] ^= 0x01
		}},
		{name: "audit destination before", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.destinationBefore[0] += "-changed"
		}},
		{name: "audit destination after", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.destinationAfter[0] += "-changed"
		}},
		{name: "action", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.actions[0].action = choice.ActionCustomExpectation
		}},
		{name: "action decision", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.actions[0].decision = choice.DecisionRecord{}
		}},
		{name: "action refusal code", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.actions[0].preparationRefusalCode += "-changed"
		}},
		{name: "action error", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.actions[0].preparationErr = errors.New("changed typed refusal")
		}},
		{name: "action path", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.actions[0].destinationPath += "-changed"
		}},
		{name: "action destination before", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.actions[0].destinationBefore[0] += "-changed"
		}},
		{name: "action destination after", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.actions[0].destinationAfter[0] += "-changed"
		}},
		{name: "action reviewer", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.actions[0].customReviewer += "-changed"
		}},
		{name: "action evidence", mutate: func(value *choiceAuditProofTestFixture) {
			value.audit.actions[0].customReviewEvidence = choiceAuditProofTestDigest('c')
		}},
		{name: "choicepoint record", mutate: func(value *choiceAuditProofTestFixture) {
			value.record = choice.ChoicepointRecord{}
		}},
		{name: "decision record", mutate: func(value *choiceAuditProofTestFixture) {
			value.decision = choice.DecisionRecord{}
		}},
		{name: "contract bundle", mutate: func(value *choiceAuditProofTestFixture) {
			value.bundle = emitmodel.ContractBundle{}
		}},
	}
	for _, test := range mutations {
		t.Run(test.name, func(t *testing.T) {
			mutated := cloneChoiceAuditProofTestFixture(baseline)
			test.mutate(&mutated)
			if err := validateChoiceAuditProof(mutated.audit, mutated.record, mutated.decision, mutated.bundle); err == nil {
				t.Fatal("retained construction proof accepted a mutation")
			}
		})
	}

	if err := validateChoiceAuditProof(baseline.audit, baseline.record, baseline.decision, baseline.bundle); err != nil {
		t.Fatalf("mutation cases aliased the valid baseline: %v", err)
	}
}

func TestChoiceAuditConstructionProofDoesNotReplacePublicStaticReplay(t *testing.T) {
	fixture := checkedChoiceAuditProofTestFixture(t)
	if err := validateChoiceAuditProof(fixture.audit, fixture.record, fixture.decision, fixture.bundle); err != nil {
		t.Fatalf("construction proof fixture is not internally bound: %v", err)
	}
	if err := validateChoiceAuditStatic(fixture.audit, fixture.record, fixture.decision, fixture.bundle); err == nil {
		t.Fatal("construction proof alone unexpectedly satisfied the full static protocol replay")
	}

	exact, err := os.ReadFile("reduction.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range [][]byte{
		[]byte("return validateCompletedCLIStudy(completed) == nil"),
		[]byte("return validateCompletedCLIStudyWithChoiceProof(completed, false)"),
		[]byte("else if err := validateChoiceAuditStatic("),
	} {
		if !bytes.Contains(exact, required) {
			t.Fatalf("public CompletedCLIStudy validation no longer retains the full static choice replay edge %q", required)
		}
	}
}

func checkedChoiceAuditProofTestFixture(t testing.TB) choiceAuditProofTestFixture {
	t.Helper()
	record, err := choice.ParseChoicepointRecord(checkedCLIStudyCanonicalExample(t, "choicepoint.valid.json"))
	if err != nil || !record.Valid() {
		t.Fatalf("parse checked Choicepoint example: valid=%t err=%v", record.Valid(), err)
	}
	decision, err := choice.ParseDecisionRecord(
		checkedCLIStudyCanonicalExample(t, "decision-record.valid.json"), record,
	)
	if err != nil || !decision.Valid() {
		t.Fatalf("parse checked DecisionRecord example: valid=%t err=%v", decision.Valid(), err)
	}
	bundleBytes := checkedCLIStudyCanonicalExample(t, "contract-bundle.valid.json")
	bundleDigest, err := canon.DigestBytes(emitmodel.BundleDigestDomain, bundleBytes)
	if err != nil {
		t.Fatal(err)
	}
	expectedBundleDigest, err := domain.ParseDigest(bundleDigest.String())
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := emitmodel.ParseContractBundle(bundleBytes, expectedBundleDigest)
	if err != nil || !bundle.Valid() {
		t.Fatalf("parse checked ContractBundle example: valid=%t err=%v", bundle.Valid(), err)
	}
	audit := choiceAudit{
		decisionDigest:    decision.Digest(),
		earlyReveal:       decision.EarlyReveal(),
		blindState:        choice.SessionBlindOpen,
		provisionalState:  choice.SessionProvisionalRecorded,
		revealedState:     choice.SessionRevealed,
		finalState:        choice.SessionFinalized,
		ambiguityCode:     choice.CodeAmbiguousScope,
		allowedTupleBytes: [][]byte{[]byte("first-tuple"), []byte("second-tuple")},
		noncompilableCode: string(choice.CodeNoncompilablePredicate),
		destinationBefore: []string{"audit-before"},
		destinationAfter:  []string{"audit-after"},
		actions: []choiceActionAudit{{
			action:                 choice.ActionAllowObserved,
			decision:               decision,
			preparationRefusalCode: "",
			preparationErr:         nil,
			destinationPath:        filepath.Join(string(filepath.Separator), "tmp", "choice-proof"),
			destinationBefore:      []string{"action-before"},
			destinationAfter:       []string{"action-after"},
			customReviewer:         "checked-reviewer",
			customReviewEvidence:   choiceAuditProofTestDigest('a'),
		}},
	}
	audit.validationProof, err = choiceAuditProof(record, decision, bundle, audit)
	if err != nil || !audit.validationProof.Valid() {
		t.Fatalf("construct checked choice audit proof: valid=%t err=%v", audit.validationProof.Valid(), err)
	}
	return choiceAuditProofTestFixture{record: record, decision: decision, bundle: bundle, audit: audit}
}

func cloneChoiceAuditProofTestFixture(value choiceAuditProofTestFixture) choiceAuditProofTestFixture {
	cloned := value
	cloned.audit.allowedTupleBytes = make([][]byte, len(value.audit.allowedTupleBytes))
	for index := range value.audit.allowedTupleBytes {
		cloned.audit.allowedTupleBytes[index] = append([]byte(nil), value.audit.allowedTupleBytes[index]...)
	}
	cloned.audit.destinationBefore = append([]string(nil), value.audit.destinationBefore...)
	cloned.audit.destinationAfter = append([]string(nil), value.audit.destinationAfter...)
	cloned.audit.actions = append([]choiceActionAudit(nil), value.audit.actions...)
	for index := range value.audit.actions {
		cloned.audit.actions[index].destinationBefore = append([]string(nil), value.audit.actions[index].destinationBefore...)
		cloned.audit.actions[index].destinationAfter = append([]string(nil), value.audit.actions[index].destinationAfter...)
	}
	return cloned
}

func checkedCLIStudyCanonicalExample(t testing.TB, name string) []byte {
	t.Helper()
	pretty, err := os.ReadFile(filepath.Join("..", "..", "..", "spec", "examples", "v1", name))
	if err != nil {
		t.Fatal(err)
	}
	exact, err := canon.Canonicalize(pretty)
	if err != nil {
		t.Fatal(err)
	}
	return exact
}

func choiceAuditProofTestDigest(character byte) domain.Digest {
	return domain.MustDigest("sha256:" + strings.Repeat(string(character), 64))
}

func assertCLIChoiceClosure(t testing.TB, completed CompletedCLIStudy) {
	t.Helper()
	if !completed.choicepoint.Valid() || !completed.decision.Valid() || !completed.durableRuling.Record().Valid() ||
		!completed.portableSource.Valid() || !completed.bundle.Valid() || !completed.residue.Valid() {
		t.Fatal("completed CLI study retained an invalid semantic authority")
	}
	if completed.decision.ChoicepointDigest() != completed.choicepoint.Digest() ||
		completed.durableRuling.Record().Digest() != completed.decision.Digest() ||
		completed.bundle.DecisionRecordDigest() != completed.decision.Digest() ||
		completed.bundle.ChoicepointDigest() != completed.choicepoint.Digest() ||
		completed.bundle.PortableSource().Digest() != completed.portableSource.Digest() ||
		completed.residue.BundleDigest() != completed.bundle.Digest() ||
		completed.residue.DecisionRecordDigest() != completed.decision.Digest() ||
		completed.residue.ChoicepointDigest() != completed.choicepoint.Digest() {
		t.Fatal("source, ruling, decision, bundle, and residue do not form one exact digest-linked graph")
	}
	if !bytes.Equal(completed.durableRuling.Record().CanonicalBytes(), completed.decision.CanonicalBytes()) ||
		!bytes.Equal(completed.bundle.PortableSource().CanonicalBytes(), completed.portableSource.CanonicalBytes()) ||
		completed.residue.Bundle().Digest() != completed.bundle.Digest() ||
		!bytes.Equal(completed.residue.Bundle().CanonicalBytes(), completed.bundle.CanonicalBytes()) {
		t.Fatal("source, ruling, decision, bundle, and residue digest joins conceal unequal canonical authorities")
	}
	view, cliView := completed.portableSource.CLIView()
	if !cliView || !view.Valid() || view.SourceDigest() != completed.portableSource.Digest() ||
		view.Stimulus().Digest() != completed.confirmed.Stimulus.Digest() ||
		completed.portableSource.Plan().Digest() != completed.confirmed.Plan.Digest() ||
		completed.portableSource.ProjectionBinding().Digest() != completed.confirmed.Plan.ProjectionDefinitionBinding().Digest() {
		t.Fatal("portable CLI source does not rejoin the confirmed minimized witness")
	}
	predicate := completed.bundle.Predicate()
	wantSelectedFields := []string{
		string(countercli.CLIFieldExitCode),
		string(countercli.CLIFieldStdoutJSONMode),
		string(countercli.CLIFieldStdoutJSONSource),
	}
	if !predicate.ValidFor(completed.portableSource.Profile(), completed.portableSource.StimulusDigest()) ||
		!slices.Equal(predicate.SelectedFields(), wantSelectedFields) ||
		!slices.Equal(completed.decision.SelectedFields(), wantSelectedFields) {
		t.Fatal("bundle predicate does not rejoin the exact source profile and selected decision fields")
	}
	compiledDecision, compilableDecision := completed.decision.CompilableRuling()
	if !compilableDecision || len(compiledDecision.DisallowedTuples()) != 1 ||
		!exactCLIChoiceTuple(compiledDecision.DisallowedTuples()[0], "env") {
		t.Fatal("operative decision did not retain sole disallowed exit0/env/env tuple")
	}

	session, err := choice.NewSession(completed.choicepoint)
	if err != nil || session.State() != choice.SessionBlindOpen {
		t.Fatalf("open blind decision session: state=%s err=%v", session.State(), err)
	}
	blind := session.BlindDTO()
	if !blind.Digest().Valid() || len(blind.Cards()) < 3 || len(blind.SelectableFields()) < 3 {
		t.Fatalf("blind view is incomplete: cards=%d fields=%v", len(blind.Cards()), blind.SelectableFields())
	}
	for _, binding := range completed.confirmed.CandidateBindings {
		for _, forbidden := range []string{binding.Key().String(), binding.Identity().TreeIdentityDigest.String()} {
			if forbidden != "" && bytes.Contains(blind.CanonicalBytes(), []byte(forbidden)) {
				t.Fatalf("blind view leaked candidate authority %q", forbidden)
			}
		}
	}
	for _, role := range completed.confirmed.CandidateRoles {
		if bytes.Contains(blind.CanonicalBytes(), []byte(string(role))) {
			t.Fatalf("blind view leaked reveal-only role %q", role)
		}
	}
	if _, err := session.Visit(choice.SurfaceProvenance); !choice.IsRefusal(err, choice.CodeInvalidSessionState) {
		t.Fatalf("provenance was visible before reveal: %v", err)
	}
	ambiguousAliases := make([]string, 0, 2)
	for _, card := range blind.Cards() {
		mode, source := "", ""
		for _, field := range card.Fields {
			switch field.FieldID {
			case string(countercli.CLIFieldStdoutJSONMode):
				mode = field.Text
			case string(countercli.CLIFieldStdoutJSONSource):
				source = field.Text
			}
		}
		if mode == source && (mode == "config" || mode == "argv") {
			ambiguousAliases = append(ambiguousAliases, card.Alias)
		}
	}
	if len(ambiguousAliases) != 2 {
		t.Fatalf("blind ambiguity control could not identify exact config+argv aliases: %v", ambiguousAliases)
	}
	ambiguous := choice.RulingDraftInput{
		Action:         choice.ActionAllowObserved,
		SelectedFields: []string{string(countercli.CLIFieldExitCode)}, AllowedAliases: ambiguousAliases,
	}
	if refused, err := session.Propose(ambiguous); !choice.IsRefusal(err, choice.CodeRequiredSurfaceNotVisited) || refused.State() != "" {
		t.Fatalf("blind proposal bypassed review surfaces: %v", err)
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
			t.Fatalf("visit blind surface %s: %v", surface, err)
		}
	}
	if refused, err := session.Propose(ambiguous); !choice.IsRefusal(err, choice.CodeAmbiguousScope) || refused.State() != "" {
		t.Fatalf("exit-only config+argv scope collapse was not refused without a downstream draft: %v", err)
	}
	if completed.choiceAudit.decisionDigest != completed.decision.Digest() ||
		completed.choiceAudit.earlyReveal || completed.decision.EarlyReveal() || completed.decision.ChangedAfterReveal() ||
		completed.choiceAudit.blindState != choice.SessionBlindOpen ||
		completed.choiceAudit.provisionalState != choice.SessionProvisionalRecorded ||
		completed.choiceAudit.revealedState != choice.SessionRevealed ||
		completed.choiceAudit.finalState != choice.SessionFinalized ||
		completed.choiceAudit.ambiguityCode != choice.CodeAmbiguousScope {
		t.Fatalf("retained choice audit differs from blind-first decision facts: %+v", completed.choiceAudit)
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness,
		choice.SurfaceMinimizedWitness,
		choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations,
		choice.SurfaceNonassertedFields,
		choice.SurfaceProvenance,
	} {
		if !slices.Contains(completed.decision.VisitedSurfaces(), surface) {
			t.Fatalf("completed decision omitted required review surface %s", surface)
		}
	}

	allowed := predicate.AllowedTuples()
	if len(allowed) != 2 || len(completed.choiceAudit.allowedTupleBytes) != len(allowed) {
		t.Fatalf("allow-many tuple roster = predicate:%d audit:%d", len(allowed), len(completed.choiceAudit.allowedTupleBytes))
	}
	allowedByMode := make(map[string][]emitmodel.ExactField, 2)
	for index, tuple := range allowed {
		if !tuple.Valid() || !bytes.Equal(tuple.CanonicalBytes(), completed.choiceAudit.allowedTupleBytes[index]) || len(tuple.Fields()) != 3 {
			t.Fatalf("allowed tuple %d lost complete canonical correlation", index)
		}
		fields := tuple.Fields()
		exit, exitInteger := fields[0].Value().PortableValue().IntegerText()
		mode, modeString := fields[1].Value().PortableValue().StringText()
		source, sourceString := fields[2].Value().PortableValue().StringText()
		if fields[0].FieldID() != string(countercli.CLIFieldExitCode) ||
			fields[1].FieldID() != string(countercli.CLIFieldStdoutJSONMode) ||
			fields[2].FieldID() != string(countercli.CLIFieldStdoutJSONSource) ||
			!exitInteger || exit != "0" || !modeString || !sourceString || mode != source ||
			(mode != "argv" && mode != "config") {
			t.Fatalf("allowed tuple %d is not one complete exit0 config/argv pair: %q/%q/%q", index, exit, mode, source)
		}
		allowedByMode[mode] = fields
	}
	configFields, hasConfig := allowedByMode["config"]
	argvFields, hasArgv := allowedByMode["argv"]
	if !hasConfig || !hasArgv || len(allowedByMode) != 2 {
		t.Fatalf("allowed tuple roster lacks exact config and argv pairs: %v", allowedByMode)
	}
	for name, fields := range map[string][]emitmodel.ExactField{
		"config-mode/argv-source": {configFields[0], configFields[1], argvFields[2]},
		"argv-mode/config-source": {argvFields[0], argvFields[1], configFields[2]},
	} {
		cross, err := emitmodel.NewExactTuple(fields)
		if err != nil {
			t.Fatalf("construct complete %s cross-pair control: %v", name, err)
		}
		if matched, err := predicate.Matches(cross); err != nil || matched {
			t.Fatalf("allow-many predicate admitted %s Cartesian expansion: matched=%t err=%v", name, matched, err)
		}
	}
	envPortable, err := portablevalue.String("env")
	if err != nil {
		t.Fatalf("construct env hostile value: %v", err)
	}
	envExact, err := emitmodel.NewExactValue(envPortable)
	if err != nil {
		t.Fatalf("construct exact env hostile value: %v", err)
	}
	envMode, err := emitmodel.NewExactField(string(countercli.CLIFieldStdoutJSONMode), envExact)
	if err != nil {
		t.Fatalf("construct exact env hostile field: %v", err)
	}
	envSource, err := emitmodel.NewExactField(string(countercli.CLIFieldStdoutJSONSource), envExact)
	if err != nil {
		t.Fatalf("construct exact env hostile source: %v", err)
	}
	for name, fields := range map[string][]emitmodel.ExactField{
		"sole disallowed exit0/env/env": {configFields[0], envMode, envSource},
		"unwitnessed exit0/env/argv":    {configFields[0], envMode, argvFields[2]},
	} {
		hostile, err := emitmodel.NewExactTuple(fields)
		if err != nil {
			t.Fatalf("construct %s tuple: %v", name, err)
		}
		if matched, err := predicate.Matches(hostile); err != nil || matched {
			t.Fatalf("allow-many predicate admitted %s tuple: matched=%t err=%v", name, matched, err)
		}
	}
	wantActions := []choice.Action{
		choice.ActionAllowObserved,
		choice.ActionCustomExpectation,
		choice.ActionRejectAll,
		choice.ActionDefer,
		choice.ActionRefine,
	}
	if len(completed.choiceAudit.actions) != len(wantActions) {
		t.Fatalf("choice action audit count = %d, want exact five", len(completed.choiceAudit.actions))
	}
	for index, want := range wantActions {
		fact := completed.choiceAudit.actions[index]
		if fact.action != want || !fact.decision.Valid() || fact.decision.Action() != want ||
			fact.decision.ChoicepointDigest() != completed.choicepoint.Digest() {
			t.Fatalf("choice action audit %d = %s with invalid or cross-choicepoint decision", index, fact.action)
		}
		parsed, err := choice.ParseDecisionRecord(fact.decision.CanonicalBytes(), completed.choicepoint)
		if err != nil || parsed.Digest() != fact.decision.Digest() {
			t.Fatalf("choice action audit %s does not strictly recover: %v", fact.action, err)
		}
		switch want {
		case choice.ActionAllowObserved:
			if fact.decision.Digest() != completed.decision.Digest() || fact.decision.EarlyReveal() {
				t.Fatal("ALLOW_OBSERVED audit is not the exact blind-first final decision")
			}
			if _, err := choice.InspectPortableRuling(fact.decision); err != nil {
				t.Fatalf("ALLOW_OBSERVED failed direct portable inspection: %v", err)
			}
		case choice.ActionCustomExpectation:
			compiled, compilable := fact.decision.CompilableRuling()
			if !compilable {
				t.Fatal("CUSTOM_EXPECTATION audit was relabeled noncompilable")
			}
			customTuples := compiled.AllowedTuples()
			if len(customTuples) != 1 || len(customTuples[0].Fields) != 1 ||
				customTuples[0].Fields[0].FieldID != string(countercli.CLIFieldStdoutJSONMode) ||
				customTuples[0].Fields[0].Value.Tag() != choice.ValueString ||
				customTuples[0].Fields[0].Value.Text() != "reviewed-custom" {
				t.Fatalf("CUSTOM_EXPECTATION audit changed the exact reviewed custom value: %+v", customTuples)
			}
			review, reviewed := compiled.CustomReview()
			if !reviewed || fact.customReviewer == "" ||
				review.Reviewer() != fact.customReviewer || !fact.customReviewEvidence.Valid() ||
				review.EvidenceDigest() != fact.customReviewEvidence ||
				fact.customReviewEvidence != completed.confirmed.Confirmation.Draft().Digest() {
				t.Fatal("CUSTOM_EXPECTATION audit lacks its distinct exact reviewer evidence")
			}
			if _, err := choice.InspectPortableRuling(fact.decision); err != nil {
				t.Fatalf("CUSTOM_EXPECTATION failed direct portable inspection: %v", err)
			}
		default:
			_, actualRefusal := choice.InspectPortableRuling(fact.decision)
			currentDestination, destinationErr := snapshotChoiceDestination(fact.destinationPath)
			if _, compilable := fact.decision.CompilableRuling(); compilable ||
				!choice.IsRefusal(actualRefusal, choice.CodeNoncompilablePredicate) ||
				fact.preparationRefusalCode != string(choice.CodeNoncompilablePredicate) ||
				!choice.IsRefusal(fact.preparationErr, choice.CodeNoncompilablePredicate) ||
				actualRefusal.Error() != fact.preparationErr.Error() ||
				destinationErr != nil || !filepath.IsAbs(fact.destinationPath) ||
				len(fact.destinationBefore) != 0 || len(fact.destinationAfter) != 0 ||
				!slices.Equal(fact.destinationBefore, fact.destinationAfter) ||
				!slices.Equal(fact.destinationAfter, currentDestination) {
				t.Fatalf("%s did not remain a typed noncompilable zero-file action: %+v", want, fact)
			}
		}
	}
	if completed.choiceAudit.noncompilableCode != string(choice.CodeNoncompilablePredicate) ||
		len(completed.choiceAudit.destinationBefore) != 0 || len(completed.choiceAudit.destinationAfter) != 0 ||
		!slices.Equal(completed.choiceAudit.destinationBefore, completed.choiceAudit.destinationAfter) {
		t.Fatalf("noncompilable control created executable residue: %+v", completed.choiceAudit)
	}
}

func exactCLIChoiceTuple(tuple choice.CompleteTuple, mode string) bool {
	fields := tuple.Fields
	return len(fields) == 3 &&
		fields[0].FieldID == string(countercli.CLIFieldExitCode) &&
		fields[0].Value.Tag() == choice.ValueInteger && fields[0].Value.Text() == "0" &&
		fields[1].FieldID == string(countercli.CLIFieldStdoutJSONMode) &&
		fields[1].Value.Tag() == choice.ValueString && fields[1].Value.Text() == mode &&
		fields[2].FieldID == string(countercli.CLIFieldStdoutJSONSource) &&
		fields[2].Value.Tag() == choice.ValueString && fields[2].Value.Text() == mode
}

func assertCLISemanticControls(t testing.TB, completed CompletedCLIStudy, ambientHome string) {
	t.Helper()
	wantKinds := []string{
		"CONTROL_STDIN_ABSENT",
		"CONTROL_STDIN_PRESENT_EMPTY",
		"CONTROL_APP_MODE_ABSENT",
		"CONTROL_APP_MODE_PRESENT_EMPTY",
		"CONTROL_ORDERED_ARGV_MODE_THEN_STDERR",
		"CONTROL_ORDERED_ARGV_STDERR_THEN_MODE",
		"CONTROL_SPARSE_IRRELEVANT_ENV_ALPHA",
		"CONTROL_SPARSE_IRRELEVANT_ENV_OMEGA",
		"CONTROL_DEFAULT_EXIT_ZERO_HOME_ISOLATION",
		"CONTROL_UNSELECTED_STDERR_ALPHA",
		"CONTROL_UNSELECTED_STDERR_BETA",
		"CONTROL_EXIT_SEVEN_SUBSTRATE",
		"CONTROL_SIGNAL_SELECTED_EXIT_MISSING",
		"CONTROL_TIMEOUT",
		"CONTROL_MALFORMED_SELECTED_JSON",
		"CONTROL_OUTPUT_LIMIT",
	}
	if len(completed.semanticControls) != len(wantKinds) {
		t.Fatalf("semantic control count = %d, want %d", len(completed.semanticControls), len(wantKinds))
	}
	byKind := make(map[string]physicalControl, len(wantKinds))
	seenAttempts := make(map[string]struct{}, len(wantKinds))
	for index, control := range completed.semanticControls {
		if control.kind != wantKinds[index] || control.role == "" || !control.result.World().Digest().Valid() {
			t.Fatalf("semantic control %d = %q/%q, want %q with one direct physical world", index, control.kind, control.role, wantKinds[index])
		}
		attempt := control.result.FinalizedAttempt().ArtifactDigest().String()
		world := control.result.World()
		finalized := control.result.FinalizedAttempt()
		if attempt == "" || !control.plan.Digest().Valid() || !control.stimulus.Valid() || !control.binding.Valid() ||
			!control.definition.Valid() || control.binding.PlanDigest() != control.plan.Digest() ||
			control.binding.StimulusDigest() != control.stimulus.Digest() ||
			control.binding.CapturePolicyDigest() != control.plan.CapturePolicyDigest() ||
			control.binding.ProjectionDefinitionBinding().Digest() != control.definition.Binding().Digest() ||
			world.PlanDigest() != control.plan.Digest() || world.StimulusDigest() != control.stimulus.Digest() ||
			world.EnvelopeDigest() != control.plan.ComparisonEnvelopeDigest() ||
			world.CapturePolicyDigest() != control.binding.CapturePolicyDigest() ||
			world.ProjectionDefinitionDigest() != control.definition.Binding().Digest() ||
			world.AttemptArtifactDigest() != finalized.ArtifactDigest() || world.CandidateKey() != control.observation.CandidateKey() ||
			!control.measurements.Digest().Valid() || control.measurements.SubjectDigest() != world.Digest() ||
			!control.observation.Valid() || control.observation.WorldDigest() != world.Digest() ||
			control.observation.PlanDigest() != control.plan.Digest() ||
			control.observation.StimulusDigest() != control.stimulus.Digest() ||
			control.observation.ExecutionPayloadDigest() != control.binding.ExecutionPayloadDigest() ||
			control.observation.AttemptDigest() != finalized.ArtifactDigest() ||
			control.observation.CapturePolicyDigest() != control.binding.CapturePolicyDigest() ||
			control.observation.ProjectionDefinitionDigest() != control.definition.Binding().Digest() {
			t.Fatalf("semantic control %s lacks an admitted physical attempt", control.kind)
		}
		if control.projected {
			derivation := control.projection.Derivation()
			replayed, replayErr := control.definition.Project(control.observation)
			if replayErr != nil || control.projectionRejection != nil || !derivation.Valid() ||
				derivation.ObservationDigest() != control.observation.Digest() ||
				derivation.DefinitionDigest() != control.definition.Binding().Digest() ||
				derivation.AdapterDefinitionDigest() != control.definition.Digest() ||
				replayed.DerivationDigest() != control.projection.DerivationDigest() ||
				!bytes.Equal(replayed.CanonicalBytes(), control.projection.CanonicalBytes()) ||
				!bytes.Equal(replayed.ProjectionBytes(), control.projection.ProjectionBytes()) {
				t.Fatalf("semantic control %s projection does not replay from its exact capture: %v", control.kind, replayErr)
			}
		} else if control.projectionRejection != nil {
			_, replayErr := control.definition.Project(control.observation)
			var replayed *countercli.ProjectionRejection
			if !errors.As(replayErr, &replayed) || !control.projectionRejection.Valid() || !replayed.Valid() ||
				control.projectionRejection.ObservationDigest() != control.observation.Digest() ||
				control.projectionRejection.DefinitionDigest() != control.definition.Binding().Digest() ||
				replayed.EvidenceDigest() != control.projectionRejection.EvidenceDigest() ||
				!bytes.Equal(replayed.CanonicalBytes(), control.projectionRejection.CanonicalBytes()) {
				t.Fatalf("semantic control %s rejection does not replay from its exact capture: %v", control.kind, replayErr)
			}
		} else if primary, controlled := finalized.PrimaryControl(); !controlled ||
			(primary != domain.ControlTimeout && primary != domain.ControlOutputLimit) {
			t.Fatalf("semantic control %s lacks exact projection or lifecycle authority", control.kind)
		}
		if _, duplicate := seenAttempts[attempt]; duplicate {
			t.Fatalf("semantic control %s reused physical attempt %s", control.kind, attempt)
		}
		seenAttempts[attempt] = struct{}{}
		byKind[control.kind] = control
	}
	if len(byKind) != len(wantKinds) || len(seenAttempts) != len(wantKinds) {
		t.Fatal("semantic control labels or physical attempts are not one-to-one")
	}
	if completed.overlayAudit.refusalCode != countercli.CodeFixturePath ||
		completed.overlayAudit.attemptsBefore != 0 || completed.overlayAudit.attemptsAfter != 0 ||
		completed.overlayAudit.worldsBefore != 0 || completed.overlayAudit.worldsAfter != 0 {
		t.Fatalf("invalid overlay did not remain a pre-spawn constructor refusal: %+v", completed.overlayAudit)
	}

	absentStdin := byKind["CONTROL_STDIN_ABSENT"]
	emptyStdin := byKind["CONTROL_STDIN_PRESENT_EMPTY"]
	if absentStdin.stimulus.Stdin().Present() || !emptyStdin.stimulus.Stdin().Present() ||
		len(emptyStdin.stimulus.Stdin().Bytes()) != 0 || absentStdin.stimulus.Digest() == emptyStdin.stimulus.Digest() ||
		absentStdin.result.Process().StdinPresence() == emptyStdin.result.Process().StdinPresence() {
		t.Fatal("absent and present-empty stdin collapsed in stimulus identity or physical receipt")
	}

	absentMode, absentPresent := cliEnvironment(byKind["CONTROL_APP_MODE_ABSENT"].stimulus, "APP_MODE")
	emptyMode, emptyPresent := cliEnvironment(byKind["CONTROL_APP_MODE_PRESENT_EMPTY"].stimulus, "APP_MODE")
	if !absentPresent || !emptyPresent || absentMode.Present() || !emptyMode.Present() || emptyMode.Value() != "" ||
		byKind["CONTROL_APP_MODE_ABSENT"].stimulus.Digest() == byKind["CONTROL_APP_MODE_PRESENT_EMPTY"].stimulus.Digest() {
		t.Fatal("absent and present-empty APP_MODE collapsed in the retained stimulus")
	}

	orderedFirst := byKind["CONTROL_ORDERED_ARGV_MODE_THEN_STDERR"]
	orderedSecond := byKind["CONTROL_ORDERED_ARGV_STDERR_THEN_MODE"]
	if !slices.Equal(orderedFirst.stimulus.Argv(), []string{"--mode", "argv", "--stderr", "ordered"}) ||
		!slices.Equal(orderedSecond.stimulus.Argv(), []string{"--stderr", "ordered", "--mode", "argv"}) ||
		orderedFirst.stimulus.Digest() == orderedSecond.stimulus.Digest() ||
		!bytes.Equal(orderedFirst.projection.ProjectionBytes(), orderedSecond.projection.ProjectionBytes()) {
		t.Fatal("ordered argv pair lost order identity or stable selected projection")
	}

	sparseAlpha := byKind["CONTROL_SPARSE_IRRELEVANT_ENV_ALPHA"]
	sparseOmega := byKind["CONTROL_SPARSE_IRRELEVANT_ENV_OMEGA"]
	alpha, hasAlpha := cliEnvironment(sparseAlpha.stimulus, "A_UNUSED")
	omega, hasOmega := cliEnvironment(sparseOmega.stimulus, "Z_UNUSED")
	if !hasAlpha || !hasOmega || !alpha.Present() || !omega.Present() || alpha.Value() != "alpha" || omega.Value() != "omega" ||
		sparseAlpha.stimulus.Digest() == sparseOmega.stimulus.Digest() ||
		!bytes.Equal(sparseAlpha.projection.ProjectionBytes(), sparseOmega.projection.ProjectionBytes()) {
		t.Fatal("sparse irrelevant environment pair lost distinct inputs or stable selected projection")
	}

	home := byKind["CONTROL_DEFAULT_EXIT_ZERO_HOME_ISOLATION"]
	if home.result.Roots().Home() == ambientHome || !filepath.IsAbs(home.result.Roots().Home()) {
		t.Fatalf("ambient HOME crossed into the physical world: ambient=%q world=%q", ambientHome, home.result.Roots().Home())
	}
	for _, binding := range home.result.Process().Environment() {
		if binding == "HOME="+ambientHome || strings.Contains(binding, "ambient-must-not-cross") {
			t.Fatalf("ambient HOME sentinel crossed into child environment: %q", binding)
		}
	}
	selectedCompletion, selectedPresent := home.observation.Completion()
	selectedCode, selectedExited := selectedCompletion.ExitCode()
	selectedFields := home.projection.Fields()
	if !selectedPresent || !selectedExited || selectedCode != 0 || selectedCompletion.Kind() != countercli.CompletionExited ||
		!home.observation.ProjectionEligible() || !home.projected || home.result.FinalizedAttempt().HasControls() ||
		len(selectedFields) != 1 || selectedFields[0].ID() != countercli.CLIFieldExitCode ||
		selectedFields[0].Value().Tag() != countercli.ExactInteger {
		t.Fatal("enrolled default-exit-zero control lost its clean selected tuple or HOME isolation")
	}

	stderrAlpha := byKind["CONTROL_UNSELECTED_STDERR_ALPHA"]
	stderrBeta := byKind["CONTROL_UNSELECTED_STDERR_BETA"]
	if bytes.Equal(stderrAlpha.observation.Stderr().Bytes(), stderrBeta.observation.Stderr().Bytes()) ||
		!bytes.Equal(stderrAlpha.projection.ProjectionBytes(), stderrBeta.projection.ProjectionBytes()) {
		t.Fatal("unselected stderr mutation changed the selected projection or failed to mutate stderr")
	}

	substrate := byKind["CONTROL_EXIT_SEVEN_SUBSTRATE"]
	substrateCompletion, substratePresent := substrate.observation.Completion()
	substrateCode, exited := substrateCompletion.ExitCode()
	if !substratePresent || !exited || substrateCompletion.Kind() != countercli.CompletionExited || substrateCode != 7 ||
		!substrate.observation.ProjectionEligible() || !substrate.projected || substrate.result.FinalizedAttempt().HasControls() {
		t.Fatal("fixture exit-7 substrate was relabeled as an ineligible or enrolled official result")
	}

	signal := byKind["CONTROL_SIGNAL_SELECTED_EXIT_MISSING"]
	signalCompletion, signalPresent := signal.observation.Completion()
	signalName, signaled := signalCompletion.Signal()
	_, signalHasExit := signalCompletion.ExitCode()
	signalFields := signal.projection.Fields()
	if !signalPresent || !signaled || signalName == "" || signalHasExit ||
		signalCompletion.Kind() != countercli.CompletionSignaled || !signal.observation.ProjectionEligible() ||
		!signal.projected || signal.result.FinalizedAttempt().HasControls() ||
		len(signalFields) != 1 || signalFields[0].ID() != countercli.CLIFieldExitCode ||
		signalFields[0].Value().Tag() != countercli.ExactMissing ||
		bytes.Equal(signal.projection.ProjectionBytes(), home.projection.ProjectionBytes()) {
		t.Fatal("clean signal did not retain eligible SIGNALED/MISSING evidence contradicting the selected exit-zero tuple")
	}

	for _, kind := range []string{"CONTROL_TIMEOUT", "CONTROL_OUTPUT_LIMIT"} {
		control := byKind[kind]
		if !control.result.Process().PhysicalExecutionEntered() || !control.result.FinalizedAttempt().HasControls() || control.projected {
			t.Fatalf("%s was relabeled as an official or projected execution", kind)
		}
	}
	malformed := byKind["CONTROL_MALFORMED_SELECTED_JSON"]
	if malformed.result.FinalizedAttempt().HasControls() || malformed.projected || malformed.projectionRejection == nil {
		t.Fatal("malformed selected JSON did not remain a post-capture projection refusal")
	}
}

func cliEnvironment(stimulus countercli.CLIStimulus, name string) (countercli.CLIEnvironmentBinding, bool) {
	for _, binding := range stimulus.Environment() {
		if binding.Name() == name {
			return binding, true
		}
	}
	return countercli.CLIEnvironmentBinding{}, false
}

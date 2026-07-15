//go:build darwin && cgo

package gitobj_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/testkit/gitrepo"
)

const systemGit = "/usr/bin/git"

func fixtureRepository(t *testing.T, format gitrepo.ObjectFormat) gitrepo.Repository {
	t.Helper()
	repository, err := gitrepo.Init(context.Background(), systemGit, t.TempDir(), format)
	if errors.Is(err, gitrepo.ErrUnsupportedObjectFormat) {
		t.Skipf("native Git does not support %s object repositories: %v", format, err)
	}
	if err != nil {
		t.Fatal(err)
	}
	return repository
}

func openRepository(t *testing.T, fixture gitrepo.Repository, executable string) gitobj.Repository {
	t.Helper()
	repository, err := gitobj.OpenRepository(context.Background(), gitobj.OpenConfig{
		GitExecutable: executable,
		Repository:    fixture.Root,
		ScratchRoot:   t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := repository.Close(); err != nil {
			t.Errorf("close repository: %v", err)
		}
	})
	return repository
}

func commitRef(t *testing.T, fixture gitrepo.Repository, ref string, files []gitrepo.File, parent, message string) string {
	t.Helper()
	commit, err := fixture.CommitFiles(context.Background(), files, parent, message)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.UpdateRef(context.Background(), ref, commit); err != nil {
		t.Fatal(err)
	}
	return commit
}

func pinRef(t *testing.T, repository gitobj.Repository, ref string) gitobj.PinnedTree {
	t.Helper()
	pinned, err := repository.Pin(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	return pinned
}

func policy(t *testing.T, entries int, total, single int64) gitobj.Policy {
	t.Helper()
	value, err := gitobj.NewPolicy(entries, total, single)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func targetParent(t *testing.T) string {
	t.Helper()
	parent := filepath.Join(t.TempDir(), "target")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func inspect(t *testing.T, pinned gitobj.PinnedTree, materializationPolicy gitobj.Policy, target string) gitobj.InspectedTree {
	t.Helper()
	value, err := gitobj.Inspect(context.Background(), pinned, materializationPolicy, target)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func refusalCode(t *testing.T, err error, want gitobj.RefusalCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("wanted refusal %s, got success", want)
	}
	got, ok := gitobj.RefusalCodeOf(err)
	if !ok || got != want {
		t.Fatalf("wanted refusal %s, got %v", want, err)
	}
}

func fixedDigest(character string) domain.Digest {
	return domain.MustDigest("sha256:" + strings.Repeat(character, 64))
}

func projectionBinding(t *testing.T) domain.ProjectionDefinitionBinding {
	t.Helper()
	binding, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain:        domain.AdapterCLI,
		ImplementationDigest: fixedDigest("7"),
		ConfigurationDigest:  fixedDigest("8"),
		AcceptedChannels:     []string{"exit", "stderr", "stdout"},
		Operations: []domain.ProjectionOperationBinding{
			{Name: "decode", RuleDigest: fixedDigest("9")},
			{Name: "select", RuleDigest: fixedDigest("a")},
		},
		Comparator:          domain.ProjectionComparatorExact,
		FieldRegistryDigest: fixedDigest("b"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func bindCandidates(t *testing.T, declaration gitobj.CandidateSetDeclaration, materializationPolicy gitobj.Policy) []gitobj.BoundCandidate {
	t.Helper()
	plan, err := domain.NewWorldPlan(domain.WorldPlanConfig{
		CandidateSetDigest:          declaration.Digest(),
		MaterializationPolicyDigest: materializationPolicy.Digest(),
		ComparisonEnvelopeDigest:    fixedDigest("3"),
		Adapter: domain.Adapter{
			Domain: domain.AdapterCLI, AdapterVersion: "cli/v1", RunnerDigest: fixedDigest("4"),
		},
		ExecutionShape:       domain.OneCLIInvocation,
		StartArgv:            []string{"node", "fixture.mjs"},
		SetupArgv:            []string{},
		Environment:          []domain.EnvironmentEntry{{Name: "LANG", Value: "C"}, {Name: "NODE_NO_WARNINGS", Value: "1"}},
		SecretSlots:          []domain.SecretSlot{},
		FixtureRecipeDigest:  fixedDigest("5"),
		Readiness:            domain.Readiness{Kind: domain.ReadinessNone},
		CapturePolicyDigest:  fixedDigest("6"),
		ProjectionDefinition: projectionBinding(t),
		RepeatSchedule:       domain.RepeatSchedule{DiscoveryRepeats: 2, ConfirmationRepeats: 2},
		RequiredTools:        []domain.RequiredTool{{Name: "node", VersionConstraint: "executed-major-only"}},
		Budgets: domain.Budgets{
			CandidateCount:            declaration.CandidateCount(),
			MaterializedEntryCount:    64,
			MaterializedBytesPerWorld: 1 << 20,
			SingleBlobBytes:           1 << 18,
			ReadinessMS:               0,
			ProbeMS:                   1000,
			TeardownMS:                1000,
			StdoutBytes:               1 << 16,
			StderrBytes:               1 << 16,
			HTTPBodyBytes:             1 << 16,
			ProposedShrinkStimuli:     4,
			TotalCandidateTrials:      16,
			ShrinkWallMS:              10000,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	bound, err := declaration.Bind(plan)
	if err != nil {
		t.Fatal(err)
	}
	return bound
}

func twoCandidateDeclaration(t *testing.T, fixture gitrepo.Repository, firstFiles, secondFiles []gitrepo.File) (gitobj.CandidateSetDeclaration, gitobj.Policy) {
	t.Helper()
	commitRef(t, fixture, "refs/heads/first", firstFiles, "", "first candidate")
	commitRef(t, fixture, "refs/heads/second", secondFiles, "", "second candidate")
	repository := openRepository(t, fixture, systemGit)
	first := pinRef(t, repository, "refs/heads/first")
	second := pinRef(t, repository, "refs/heads/second")
	selected, err := gitobj.SelectTrees(first, second)
	if err != nil {
		t.Fatal(err)
	}
	materializationPolicy := policy(t, 64, 1<<20, 1<<18)
	declaration, err := gitobj.InspectSelected(context.Background(), selected, materializationPolicy, targetParent(t))
	if err != nil {
		t.Fatal(err)
	}
	return declaration, materializationPolicy
}

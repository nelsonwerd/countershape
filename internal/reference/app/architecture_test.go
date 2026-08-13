package app

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const referenceModulePrefix = "github.com/nelsonwerd/countershape/"

func TestGenericTruthPackagesDoNotDependOnReferenceDomains(t *testing.T) {
	root := referenceRepositoryRoot(t)
	for _, relative := range []string{
		"internal/canon",
		"internal/domain",
		"internal/observe",
		"internal/compare",
		"internal/reduce",
		"internal/reduction",
		"internal/choice",
	} {
		for _, source := range productionGoSources(t, filepath.Join(root, relative)) {
			for _, imported := range parsedImports(t, source) {
				local := strings.TrimPrefix(imported, referenceModulePrefix)
				if local != imported && (strings.HasPrefix(local, "internal/reference/") || strings.HasPrefix(local, "internal/adapters/")) {
					t.Fatalf("generic truth package %s imports domain/reference edge %s", source, imported)
				}
			}
		}
	}
}

func TestReferenceDomainEdgesHaveOneCrossDomainOwner(t *testing.T) {
	root := referenceRepositoryRoot(t)
	checks := []struct {
		root      string
		forbidden func(string) bool
	}{
		{
			root: "internal/reference/app",
			forbidden: func(local string) bool {
				return strings.HasPrefix(local, "internal/reference/httpstudy") ||
					strings.HasPrefix(local, "internal/reference/clistudy") ||
					strings.HasPrefix(local, "internal/reference/reproduce") ||
					strings.HasPrefix(local, "internal/adapters/") ||
					strings.HasPrefix(local, "testkit/reference")
			},
		},
		{
			root: "internal/reference/httpstudy",
			forbidden: func(local string) bool {
				return strings.HasPrefix(local, "internal/reference/clistudy") ||
					strings.HasPrefix(local, "internal/reference/reproduce") ||
					strings.HasPrefix(local, "internal/adapters/cli")
			},
		},
		{
			root: "internal/reference/clistudy",
			forbidden: func(local string) bool {
				return strings.HasPrefix(local, "internal/reference/httpstudy") ||
					strings.HasPrefix(local, "internal/reference/reproduce") ||
					strings.HasPrefix(local, "internal/adapters/http")
			},
		},
	}
	for _, check := range checks {
		for _, source := range productionGoSources(t, filepath.Join(root, check.root)) {
			for _, imported := range parsedImports(t, source) {
				local := strings.TrimPrefix(imported, referenceModulePrefix)
				if local != imported && check.forbidden(local) {
					t.Fatalf("forbidden reference edge %s -> %s", source, imported)
				}
			}
		}
	}

	reproductionSources := productionGoSources(t, filepath.Join(root, "internal/reference/reproduce"))
	if len(reproductionSources) != 1 || filepath.Base(reproductionSources[0]) != "run_darwin.go" {
		t.Fatalf("cross-domain owner inventory = %v, want only run_darwin.go", reproductionSources)
	}
	want := []string{
		referenceModulePrefix + "internal/reference/app",
		referenceModulePrefix + "internal/reference/clistudy",
		referenceModulePrefix + "internal/reference/httpstudy",
	}
	got := localImports(t, reproductionSources[0])
	if !architectureStringsEqual(got, want) {
		t.Fatalf("cross-domain owner imports = %v, want %v", got, want)
	}
}

func referenceRepositoryRoot(t testing.TB) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root {
		t.Fatalf("repository root is not canonical: root=%q resolved=%q err=%v", root, resolved, err)
	}
	return root
}

func productionGoSources(t testing.TB, root string) []string {
	t.Helper()
	var sources []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return &fs.PathError{Op: "walk", Path: path, Err: fs.ErrInvalid}
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			sources = append(sources, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(sources)
	return sources
}

func parsedImports(t testing.TB, source string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), source, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", source, err)
	}
	imports := make([]string, 0, len(parsed.Imports))
	for _, spec := range parsed.Imports {
		value, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatalf("decode import %s in %s: %v", spec.Path.Value, source, err)
		}
		imports = append(imports, value)
	}
	sort.Strings(imports)
	return imports
}

func localImports(t testing.TB, source string) []string {
	t.Helper()
	var result []string
	for _, imported := range parsedImports(t, source) {
		if strings.HasPrefix(imported, referenceModulePrefix) {
			result = append(result, imported)
		}
	}
	return result
}

func architectureStringsEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

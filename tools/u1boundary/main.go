// Command u1boundary proves the deliberately small, pure-Go U1 package
// boundary from source. It is intentionally stdlib-only and fail-closed.
package main

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

type errorCode string

const (
	codeRootType           errorCode = "BOUNDARY_ROOT_TYPE"
	codeModuleType         errorCode = "BOUNDARY_MODULE_TYPE"
	codeModulePolicy       errorCode = "BOUNDARY_MODULE_POLICY"
	codeModuleTopology     errorCode = "BOUNDARY_MODULE_TOPOLOGY"
	codeInternalType       errorCode = "BOUNDARY_INTERNAL_TYPE"
	codePackageMissing     errorCode = "BOUNDARY_PACKAGE_MISSING"
	codePackageExtra       errorCode = "BOUNDARY_PACKAGE_EXTRA"
	codePackageEntryType   errorCode = "BOUNDARY_PACKAGE_ENTRY_TYPE"
	codeNonGoArtifact      errorCode = "BOUNDARY_NON_GO_ARTIFACT"
	codeFilenameConstraint errorCode = "BOUNDARY_FILENAME_CONSTRAINT"
	codeSourceMalformed    errorCode = "BOUNDARY_SOURCE_MALFORMED"
	codeBuildConstraint    errorCode = "BOUNDARY_BUILD_CONSTRAINT"
	codePackageName        errorCode = "BOUNDARY_PACKAGE_NAME"
	codeImportLiteral      errorCode = "BOUNDARY_IMPORT_LITERAL"
	codeImportAlias        errorCode = "BOUNDARY_IMPORT_ALIAS"
	codeImportNotAllowed   errorCode = "BOUNDARY_IMPORT_NOT_ALLOWED"
	codeProcessSurface     errorCode = "BOUNDARY_PROCESS_SURFACE"
)

type boundaryError struct {
	Code   errorCode
	Path   string
	Detail string
}

func (e *boundaryError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s: %s: %s", e.Code, e.Path, e.Detail)
}

func refuse(code errorCode, path, detail string) *boundaryError {
	return &boundaryError{Code: code, Path: path, Detail: detail}
}

const (
	expectedModule = "github.com/nelsonwerd/countershape"
	expectedGo     = "1.24.0"
)

var packageOrder = []string{
	"canon",
	"choice",
	"compare",
	"domain",
	"observe",
	"reduce",
	"spec",
}

var allowedProductionImports = map[string]map[string]bool{
	"canon": importSet(
		"bytes", "crypto/sha256", "encoding", "encoding/hex", "encoding/json",
		"fmt", "reflect", "sort", "strconv", "strings", "unicode",
		"unicode/utf16", "unicode/utf8",
	),
	"choice": importSet(
		"bytes", "errors", "fmt", "sort", "strconv", "strings", "unicode/utf8",
		expectedModule+"/internal/canon",
		expectedModule+"/internal/compare",
		expectedModule+"/internal/domain",
	),
	"compare": importSet(
		"sort", "strconv", "strings",
		expectedModule+"/internal/canon",
		expectedModule+"/internal/domain",
		expectedModule+"/internal/observe",
	),
	"domain": importSet(
		"bytes", "fmt", "path/filepath", "regexp", "sort", "strings", "unicode", "unicode/utf8",
		expectedModule+"/internal/canon",
	),
	"observe": importSet(
		"fmt", "sort",
		expectedModule+"/internal/canon",
		expectedModule+"/internal/domain",
	),
	"reduce": importSet(
		"sort",
		expectedModule+"/internal/compare",
		expectedModule+"/internal/domain",
	),
	"spec": importSet(
		"fmt", "sort",
		expectedModule+"/internal/canon",
		expectedModule+"/internal/domain",
	),
}

var allowedTestImports = importSet(
	"bytes", "encoding/json", "errors", "fmt", "math/rand", "os", "reflect",
	"strconv", "strings", "sync", "testing", "testing/quick", "unicode/utf8",
	expectedModule+"/internal/canon",
	expectedModule+"/internal/choice",
	expectedModule+"/internal/compare",
	expectedModule+"/internal/domain",
	expectedModule+"/internal/observe",
	expectedModule+"/internal/reduce",
	expectedModule+"/internal/spec",
)

var implicitGOOS = stringSet(
	"aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos",
	"ios", "js", "linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1",
	"windows", "zos",
)

var implicitGOARCH = stringSet(
	"386", "amd64", "amd64p32", "arm", "armbe", "arm64", "arm64be", "loong64",
	"mips", "mipsle", "mips64", "mips64le", "mips64p32", "mips64p32le", "ppc",
	"ppc64", "ppc64le", "riscv", "riscv64", "s390", "s390x", "sparc",
	"sparc64", "wasm",
)

func importSet(values ...string) map[string]bool { return stringSet(values...) }

func stringSet(values ...string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

type result struct {
	Packages int
	GoFiles  int
}

func inspect(root string) (result, error) {
	root = filepath.Clean(root)
	if err := requireDirectory(root, ".", codeRootType); err != nil {
		return result{}, err
	}
	if err := inspectRepositoryTopology(root); err != nil {
		return result{}, err
	}
	if err := inspectModule(root); err != nil {
		return result{}, err
	}

	internal := filepath.Join(root, "internal")
	if err := requireDirectory(internal, "internal", codeInternalType); err != nil {
		return result{}, err
	}
	entries, err := os.ReadDir(internal)
	if err != nil {
		return result{}, refuse(codeInternalType, "internal", "cannot enumerate internal boundary")
	}

	found := make(map[string]bool, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		rel := filepath.ToSlash(filepath.Join("internal", name))
		info, statErr := os.Lstat(filepath.Join(internal, name))
		if statErr != nil {
			return result{}, refuse(codePackageEntryType, rel, "cannot inspect package entry")
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return result{}, refuse(codePackageEntryType, rel, "package entry is not a real directory")
		}
		if _, allowed := allowedProductionImports[name]; !allowed {
			return result{}, refuse(codePackageExtra, rel, "package is outside the exact U1 package set")
		}
		found[name] = true
	}
	for _, name := range packageOrder {
		if !found[name] {
			return result{}, refuse(codePackageMissing, filepath.ToSlash(filepath.Join("internal", name)), "required U1 package is absent")
		}
	}

	goFiles := 0
	for _, name := range packageOrder {
		count, inspectErr := inspectPackage(root, name)
		if inspectErr != nil {
			return result{}, inspectErr
		}
		goFiles += count
	}
	return result{Packages: len(packageOrder), GoFiles: goFiles}, nil
}

func requireDirectory(path, label string, code errorCode) error {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return refuse(code, label, "expected a real directory")
	}
	return nil
}

func inspectRepositoryTopology(root string) error {
	toolFiles := map[string]bool{
		"tools/u1boundary/main.go":      false,
		"tools/u1boundary/main_test.go": false,
	}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return refuse(codeModuleTopology, relative(root, path), "cannot enumerate repository topology")
		}
		if path == root {
			return nil
		}
		rel := relative(root, path)
		if entry.IsDir() && (strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_")) {
			return filepath.SkipDir
		}
		if entry.IsDir() && rel == "internal" {
			// The stricter package scanner below owns every artifact under internal.
			return filepath.SkipDir
		}
		info, err := os.Lstat(path)
		if err != nil {
			return refuse(codeModuleTopology, rel, "cannot inspect repository artifact")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return refuse(codeModuleTopology, rel, "repository topology contains a symbolic link")
		}
		if entry.IsDir() {
			if rel == "vendor" {
				return refuse(codeModuleTopology, rel, "vendored source is outside the dependency-free U1 module")
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return refuse(codeModuleTopology, rel, "repository topology contains a nonregular artifact")
		}
		if strings.HasPrefix(rel, "tools/u1boundary/") {
			if _, reviewed := toolFiles[rel]; !reviewed {
				return refuse(codeModuleTopology, rel, "analyzer directory contains an unreviewed compiler input")
			}
			toolFiles[rel] = true
			return nil
		}
		base := filepath.Base(path)
		if base == "go.mod" || base == "go.sum" || base == "go.work" || base == "go.work.sum" {
			if rel != "go.mod" {
				return refuse(codeModuleTopology, rel, "unexpected module or workspace control file")
			}
		}
		if filepath.Ext(base) != ".go" {
			return nil
		}
		return refuse(codeModuleTopology, rel, "Go source is outside the exact U1 module topology")
	})
	if err != nil {
		return err
	}
	for path, found := range toolFiles {
		if !found {
			return refuse(codeModuleTopology, path, "required boundary-analyzer source is absent")
		}
	}
	return nil
}

func inspectModule(root string) error {
	path := filepath.Join(root, "go.mod")
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return refuse(codeModuleType, "go.mod", "expected a regular, non-symlink module file")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return refuse(codeModuleType, "go.mod", "cannot read module file")
	}
	lines := make([]string, 0, 2)
	for _, rawLine := range strings.Split(string(raw), "\n") {
		line := strings.TrimSpace(rawLine)
		if line != "" {
			lines = append(lines, line)
		}
	}
	wantModule := "module " + expectedModule
	wantGo := "go " + expectedGo
	if len(lines) != 2 || lines[0] != wantModule || lines[1] != wantGo {
		return refuse(codeModulePolicy, "go.mod", "module must contain only the pinned module and Go version directives")
	}
	return nil
}

func inspectPackage(root, packageName string) (int, error) {
	directory := filepath.Join(root, "internal", packageName)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return 0, refuse(codePackageEntryType, filepath.ToSlash(filepath.Join("internal", packageName)), "cannot enumerate package")
	}
	if len(entries) == 0 {
		return 0, refuse(codePackageEntryType, filepath.ToSlash(filepath.Join("internal", packageName)), "package contains no Go source")
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		rel := relative(root, path)
		info, statErr := os.Lstat(path)
		if statErr != nil {
			return 0, refuse(codePackageEntryType, rel, "cannot inspect package artifact")
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return 0, refuse(codePackageEntryType, rel, "nested directories, symlinks, and nonregular artifacts are forbidden")
		}
		if filepath.Ext(entry.Name()) != ".go" {
			return 0, refuse(codeNonGoArtifact, rel, "only Go source is admitted at the U1 boundary")
		}
		if strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") {
			return 0, refuse(codeFilenameConstraint, rel, "Go ignores source files with a leading dot or underscore")
		}
		if hasImplicitBuildSuffix(entry.Name()) {
			return 0, refuse(codeFilenameConstraint, rel, "implicit GOOS or GOARCH filename constraints are forbidden")
		}
		files = append(files, path)
	}
	sort.Strings(files)
	for _, path := range files {
		if err := inspectGoFile(root, packageName, path); err != nil {
			return 0, err
		}
	}
	return len(files), nil
}

func relative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func hasImplicitBuildSuffix(filename string) bool {
	// Match go/build's goodOSArchFile spelling rules: the filename identity ends
	// at the first dot, and implicit tags apply only after a nonempty prefix plus
	// underscore. This deliberately catches names such as x_linux.generated.go.
	base, _, _ := strings.Cut(filename, ".")
	firstUnderscore := strings.Index(base, "_")
	if firstUnderscore < 0 {
		return false
	}
	parts := strings.Split(base[firstUnderscore:], "_")
	if len(parts) > 0 && parts[len(parts)-1] == "test" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) < 2 {
		return false
	}
	last := parts[len(parts)-1]
	if implicitGOOS[last] || implicitGOARCH[last] {
		return true
	}
	return len(parts) >= 3 && implicitGOOS[parts[len(parts)-2]] && implicitGOARCH[last]
}

func inspectGoFile(root, packageName, path string) error {
	rel := relative(root, path)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return refuse(codeSourceMalformed, rel, "Go parser rejected source")
	}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			text := strings.TrimSpace(comment.Text)
			if strings.HasPrefix(text, "//go:build") || strings.HasPrefix(text, "// +build") {
				return refuse(codeBuildConstraint, rel, "explicit build constraints are forbidden")
			}
		}
	}
	if file.Name == nil || file.Name.Name != packageName {
		return refuse(codePackageName, rel, "source package must exactly match its U1 directory")
	}

	isTest := strings.HasSuffix(path, "_test.go")
	osAliases := make(map[string]bool)
	for _, imported := range file.Imports {
		literal := imported.Path.Value
		pathValue, unquoteErr := strconv.Unquote(literal)
		if unquoteErr != nil || pathValue == "" || strings.IndexByte(pathValue, 0) >= 0 {
			return refuse(codeImportLiteral, rel, "import path is not a valid Go string")
		}
		if imported.Name != nil && (imported.Name.Name == "." || imported.Name.Name == "_") {
			return refuse(codeImportAlias, rel, "dot and blank imports are outside the proof surface")
		}
		if !importAllowed(packageName, pathValue, isTest) {
			return refuse(codeImportNotAllowed, rel, "import is outside the positive package policy: "+pathValue)
		}
		if pathValue == "os" {
			alias := "os"
			if imported.Name != nil {
				alias = imported.Name.Name
			}
			osAliases[alias] = true
		}
	}

	var surfaceErr error
	ast.Inspect(file, func(node ast.Node) bool {
		if surfaceErr != nil {
			return false
		}
		selector, ok := node.(*ast.SelectorExpr)
		if !ok || selector.Sel == nil || (selector.Sel.Name != "StartProcess" && selector.Sel.Name != "Exit") {
			return true
		}
		qualifier, ok := selector.X.(*ast.Ident)
		if ok && osAliases[qualifier.Name] {
			surfaceErr = refuse(codeProcessSurface, rel, "os.StartProcess and os.Exit are forbidden even where test-only os access is admitted")
			return false
		}
		return true
	})
	return surfaceErr
}

func importAllowed(packageName, imported string, testFile bool) bool {
	if allowedProductionImports[packageName][imported] {
		return true
	}
	return testFile && allowedTestImports[imported]
}

func fixture(root string) error {
	if err := os.MkdirAll(filepath.Join(root, "internal"), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+expectedModule+"\n\ngo "+expectedGo+"\n"), 0o600); err != nil {
		return err
	}
	toolDirectory := filepath.Join(root, "tools", "u1boundary")
	if err := os.MkdirAll(toolDirectory, 0o700); err != nil {
		return err
	}
	for _, filename := range []string{"main.go", "main_test.go"} {
		if err := os.WriteFile(filepath.Join(toolDirectory, filename), []byte("package main\n"), 0o600); err != nil {
			return err
		}
	}
	for _, name := range packageOrder {
		directory := filepath.Join(root, "internal", name)
		if err := os.Mkdir(directory, 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(directory, "value.go"), []byte("package "+name+"\n"), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func expectCode(root string, want errorCode) error {
	_, err := inspect(root)
	var boundary *boundaryError
	if !errors.As(err, &boundary) {
		return fmt.Errorf("wanted %s, got non-boundary error %v", want, err)
	}
	if boundary.Code != want {
		return fmt.Errorf("wanted %s, got %s (%v)", want, boundary.Code, boundary)
	}
	return nil
}

func runCase(mutate func(string) error, want errorCode) error {
	root, err := os.MkdirTemp("", "countershape-u1-boundary-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	if err := fixture(root); err != nil {
		return err
	}
	if mutate != nil {
		if err := mutate(root); err != nil {
			return err
		}
	}
	if want == "" {
		_, err := inspect(root)
		return err
	}
	return expectCode(root, want)
}

func selfTest() error {
	cases := []struct {
		name   string
		mutate func(string) error
		want   errorCode
	}{
		{name: "clean"},
		{name: "missing package", want: codePackageMissing, mutate: func(root string) error {
			return os.RemoveAll(filepath.Join(root, "internal", "reduce"))
		}},
		{name: "extra package", want: codePackageExtra, mutate: func(root string) error {
			return os.Mkdir(filepath.Join(root, "internal", "server"), 0o700)
		}},
		{name: "malformed source", want: codeSourceMalformed, mutate: writeSource("canon", "broken.go", "package canon\nfunc {")},
		{name: "aliased forbidden import", want: codeImportNotAllowed, mutate: writeSource("canon", "broken.go", "package canon\nimport command \"os/exec\"\n")},
		{name: "raw forbidden import", want: codeImportNotAllowed, mutate: writeSource("canon", "broken.go", "package canon\nimport `os/exec`\n")},
		{name: "escaped forbidden import", want: codeImportNotAllowed, mutate: writeSource("canon", "broken.go", "package canon\nimport \"os\\x2fexec\"\n")},
		{name: "process surface in test", want: codeProcessSurface, mutate: writeSource("spec", "process_test.go", "package spec\nimport system \"os\"\nvar _ = system.StartProcess\n")},
		{name: "test exit bypass", want: codeProcessSurface, mutate: writeSource("spec", "exit_test.go", "package spec\nimport system \"os\"\nfunc bypass() { system.Exit(0) }\n")},
		{name: "test file import is inspected", want: codeImportNotAllowed, mutate: writeSource("spec", "edge_test.go", "package spec\nimport \"net/http\"\n")},
		{name: "build tag", want: codeBuildConstraint, mutate: writeSource("canon", "tagged.go", "//go:build darwin\n\npackage canon\n")},
		{name: "implicit platform suffix", want: codeFilenameConstraint, mutate: writeSource("canon", "escape_linux.go", "package canon\n")},
		{name: "non Go artifact", want: codeNonGoArtifact, mutate: func(root string) error {
			return os.WriteFile(filepath.Join(root, "internal", "canon", "payload.s"), []byte("TEXT x(SB),$0-0\n"), 0o600)
		}},
		{name: "nested directory", want: codePackageEntryType, mutate: func(root string) error {
			return os.Mkdir(filepath.Join(root, "internal", "canon", "nested"), 0o700)
		}},
		{name: "symlink", want: codePackageEntryType, mutate: func(root string) error {
			return os.Symlink(filepath.Join(root, "go.mod"), filepath.Join(root, "internal", "canon", "linked.go"))
		}},
		{name: "fifo", want: codePackageEntryType, mutate: func(root string) error {
			return syscall.Mkfifo(filepath.Join(root, "internal", "canon", "pipe.go"), 0o600)
		}},
		{name: "wrong package", want: codePackageName, mutate: writeSource("canon", "broken.go", "package server\n")},
		{name: "module dependency", want: codeModulePolicy, mutate: func(root string) error {
			return os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+expectedModule+"\n\ngo "+expectedGo+"\n\nrequire example.invalid/edge v1.0.0\n"), 0o600)
		}},
		{name: "root Go source", want: codeModuleTopology, mutate: func(root string) error {
			return os.WriteFile(filepath.Join(root, "rogue.go"), []byte("package rogue\n"), 0o600)
		}},
		{name: "command package", want: codeModuleTopology, mutate: func(root string) error {
			directory := filepath.Join(root, "cmd", "rogue")
			if err := os.MkdirAll(directory, 0o700); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(directory, "main.go"), []byte("package main\n"), 0o600)
		}},
		{name: "workspace control", want: codeModuleTopology, mutate: func(root string) error {
			return os.WriteFile(filepath.Join(root, "go.work"), []byte("go "+expectedGo+"\n"), 0o600)
		}},
		{name: "extra analyzer source", want: codeModuleTopology, mutate: func(root string) error {
			return os.WriteFile(filepath.Join(root, "tools", "u1boundary", "extra.go"), []byte("package main\n"), 0o600)
		}},
		{name: "analyzer assembly input", want: codeModuleTopology, mutate: func(root string) error {
			return os.WriteFile(filepath.Join(root, "tools", "u1boundary", "escape.s"), []byte("TEXT ·escape(SB),$0-0\nRET\n"), 0o600)
		}},
		{name: "leading underscore source", want: codeFilenameConstraint, mutate: writeSource("canon", "_ignored.go", "package canon\n")},
		{name: "leading dot test", want: codeFilenameConstraint, mutate: writeSource("canon", ".ignored_test.go", "package canon\n")},
	}
	for _, test := range cases {
		if err := runCase(test.mutate, test.want); err != nil {
			return fmt.Errorf("self-test %q: %w", test.name, err)
		}
	}
	return nil
}

func writeSource(packageName, filename, source string) func(string) error {
	return func(root string) error {
		return os.WriteFile(filepath.Join(root, "internal", packageName, filename), []byte(source), 0o600)
	}
}

func main() {
	root := flag.String("root", ".", "repository root")
	runSelfTest := flag.Bool("self-test", false, "run exact fail-closed scanner self-tests")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "BOUNDARY_ARGUMENTS: unexpected positional arguments")
		os.Exit(2)
	}
	if *runSelfTest {
		if err := selfTest(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("U1 boundary analyzer self-test passed")
		return
	}
	checked, err := inspect(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("U1 boundary clean: %d packages, %d Go files\n", checked.Packages, checked.GoFiles)
}

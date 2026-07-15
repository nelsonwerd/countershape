// Command u2boundary proves the deliberately impure U2 source boundary. It is
// stdlib-only, parses every admitted production and test file, and fails closed
// when topology, imports, process authority, Git vocabulary, pre-plan identity,
// production materialization authority, or the mutation contract drifts.
package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type errorCode string

const (
	codeRootType             errorCode = "U2_BOUNDARY_ROOT_TYPE"
	codeModulePolicy         errorCode = "U2_BOUNDARY_MODULE_POLICY"
	codeTopology             errorCode = "U2_BOUNDARY_TOPOLOGY"
	codeSourceMalformed      errorCode = "U2_BOUNDARY_SOURCE_MALFORMED"
	codeBuildConstraint      errorCode = "U2_BOUNDARY_BUILD_CONSTRAINT"
	codePackageName          errorCode = "U2_BOUNDARY_PACKAGE_NAME"
	codeImportLiteral        errorCode = "U2_BOUNDARY_IMPORT_LITERAL"
	codeImportAlias          errorCode = "U2_BOUNDARY_IMPORT_ALIAS"
	codeAuthorityImport      errorCode = "U2_BOUNDARY_AUTHORITY_IMPORT"
	codeImportNotAllowed     errorCode = "U2_BOUNDARY_IMPORT_NOT_ALLOWED"
	codeExecutionSurface     errorCode = "U2_BOUNDARY_EXECUTION_SURFACE"
	codeEnvironmentSurface   errorCode = "U2_BOUNDARY_ENVIRONMENT_SURFACE"
	codeGitSurface           errorCode = "U2_BOUNDARY_GIT_SURFACE"
	codeCgoBoundary          errorCode = "U2_BOUNDARY_CGO_PREAMBLE"
	codeTestkitBoundary      errorCode = "U2_BOUNDARY_TESTKIT"
	codeDigestBoundary       errorCode = "U2_BOUNDARY_DIGEST_BOUNDARY"
	codeMaterializerBoundary errorCode = "U2_BOUNDARY_MATERIALIZER_BOUNDARY"
	codeAnchorContract       errorCode = "U2_BOUNDARY_ANCHOR_CONTRACT"
)

type boundaryError struct {
	Code   errorCode
	Path   string
	Detail string
}

func (e *boundaryError) Error() string {
	return fmt.Sprintf("%s: %s: %s", e.Code, e.Path, e.Detail)
}

func refuse(code errorCode, path, detail string) *boundaryError {
	return &boundaryError{Code: code, Path: path, Detail: detail}
}

const (
	expectedModule = "github.com/nelsonwerd/countershape"
	expectedGo     = "1.24.0"
)

var packageFiles = map[string][]string{
	"gitobj": {
		"errors.go", "filesystem.go", "filesystem_darwin.go", "filesystem_unsupported.go",
		"fuzz_test.go", "git.go", "helpers_external_test.go", "identity_external_test.go",
		"inspect.go", "inspect_test.go", "materialize.go", "materialize_external_test.go",
		"mutation_contract_test.go", "publish_darwin.go", "publish_darwin_test.go",
		"publish_unsupported.go", "refusal_external_test.go", "types.go", "validate.go",
	},
	"world": {
		"allocate.go", "api.go", "capture.go", "errors.go", "execute_darwin_test.go",
		"finalize.go", "process.go", "process_darwin.go", "process_darwin_test.go",
		"process_mutation_darwin_test.go", "process_unsupported.go", "receipt.go", "tools.go",
	},
}

var expectedBuildTags = map[string]string{
	"internal/gitobj/filesystem_darwin.go":           "darwin",
	"internal/gitobj/filesystem_unsupported.go":      "!darwin",
	"internal/gitobj/helpers_external_test.go":       "darwin && cgo",
	"internal/gitobj/identity_external_test.go":      "darwin && cgo",
	"internal/gitobj/materialize_external_test.go":   "darwin && cgo",
	"internal/gitobj/mutation_contract_test.go":      "darwin && cgo",
	"internal/gitobj/publish_darwin.go":              "darwin && cgo",
	"internal/gitobj/publish_darwin_test.go":         "darwin && cgo",
	"internal/gitobj/publish_unsupported.go":         "!darwin || !cgo",
	"internal/gitobj/refusal_external_test.go":       "darwin && cgo",
	"internal/world/execute_darwin_test.go":          "darwin && cgo",
	"internal/world/process_darwin.go":               "darwin",
	"internal/world/process_darwin_test.go":          "darwin && cgo",
	"internal/world/process_mutation_darwin_test.go": "darwin && cgo",
	"internal/world/process_unsupported.go":          "!darwin",
}

var externalGitobjTests = stringSet(
	"helpers_external_test.go", "identity_external_test.go", "materialize_external_test.go",
	"refusal_external_test.go",
)

var forbiddenAuthorityImports = stringSet(
	expectedModule+"/internal/observe",
	expectedModule+"/internal/compare",
	expectedModule+"/internal/reduce",
	expectedModule+"/internal/choice",
	expectedModule+"/internal/spec",
)

var productionImports = map[string]map[string]bool{
	"gitobj": stringSet(
		"bytes", "context", "crypto/sha1", "crypto/sha256", "encoding/hex", "errors", "fmt",
		"hash", "io", "math", "os", "os/exec", "path", "path/filepath", "sort", "strconv",
		"strings", "sync", "sync/atomic", "syscall", "unicode", "unicode/utf8", "unsafe", "C",
		expectedModule+"/internal/canon", expectedModule+"/internal/domain",
	),
	"world": stringSet(
		"bytes", "context", "errors", "fmt", "io", "math", "os", "os/exec", "path/filepath",
		"regexp", "sort", "strconv", "strings", "sync", "syscall", "time", "unicode", "unicode/utf8",
		expectedModule+"/internal/canon", expectedModule+"/internal/domain", expectedModule+"/internal/gitobj",
	),
}

var testImports = map[string]map[string]bool{
	"gitobj": stringSet(
		"bytes", "context", "encoding/hex", "errors", "io", "os", "path/filepath", "strings", "testing",
		expectedModule+"/internal/domain", expectedModule+"/internal/gitobj", expectedModule+"/testkit/gitrepo",
	),
	"world": stringSet(
		"context", "encoding/json", "errors", "os", "os/exec", "path/filepath", "runtime", "strconv",
		"strings", "sync", "syscall", "testing", "time", expectedModule+"/internal/domain",
		expectedModule+"/internal/gitobj", expectedModule+"/testkit/gitrepo",
	),
}

var requiredAnchors = map[string]string{
	"MUTANT_U2_ENABLE_REPLACEMENT_REFS":          "internal/gitobj/git.go",
	"MUTANT_U2_ALLOW_LAZY_FETCH":                 "internal/gitobj/git.go",
	"MUTANT_U2_ALLOW_ALTERNATE_OBJECTS":          "internal/gitobj/git.go",
	"MUTANT_U2_COLLAPSE_EXECUTABLE_MODE":         "internal/gitobj/inspect.go",
	"MUTANT_U2_ACCEPT_SYMLINK_MODE":              "internal/gitobj/inspect.go",
	"MUTANT_U2_VALIDATE_PATH_AFTER_WRITE":        "internal/gitobj/inspect.go",
	"MUTANT_U2_ACCEPT_LFS_POINTER":               "internal/gitobj/inspect.go",
	"MUTANT_U2_LEXICAL_ONLY_PATH_COLLISION":      "internal/gitobj/inspect.go",
	"MUTANT_U2_SKIP_BLOB_REHASH":                 "internal/gitobj/inspect.go",
	"MUTANT_U2_ALLOW_DESTINATION_OVERWRITE":      "internal/gitobj/publish_darwin.go",
	"MUTANT_U2_ACCEPT_SHA1_WIDTH_FOR_SHA256":     "internal/gitobj/types.go",
	"MUTANT_U2_ALLOW_DUPLICATE_PORTABLE_TREE":    "internal/gitobj/types.go",
	"MUTANT_U2_POLLUTE_PREPLAN_DIGEST":           "internal/gitobj/types.go",
	"MUTANT_U2_INHERIT_AMBIENT_ENVIRONMENT":      "internal/world/allocate.go",
	"MUTANT_U2_CREATE_MARKER_AFTER_SPAWN":        "internal/world/api.go",
	"MUTANT_U2_CLAIM_PROCESS_ESCAPE_CONTAINMENT": "internal/world/process.go",
	"MUTANT_U2_SPAWN_THROUGH_SHELL":              "internal/world/process_darwin.go",
	"MUTANT_U2_SIGNAL_DIRECT_PID":                "internal/world/process_darwin.go",
	"MUTANT_U2_REMOVE_KILL_ESCALATION":           "internal/world/process_darwin.go",
	"MUTANT_U2_REMOVE_FINAL_GROUP_PROBE":         "internal/world/process_darwin.go",
	"MUTANT_U2_MAP_OUTPUT_LIMIT_TO_SUCCESS":      "internal/world/process_darwin.go",
	"MUTANT_U2_REPORT_DRAIN_TIMEOUT_COMPLETE":    "internal/world/process_darwin.go",
	"MUTANT_U2_LATE_CONTROL_OVERRIDES_OUTPUT":    "internal/world/process_darwin.go",
	"MUTANT_U2_SHARE_OUTPUT_CAP":                 "internal/world/process_darwin.go",
	"MUTANT_U2_RESOLVE_TOOL_THROUGH_PATH":        "internal/world/tools.go",
}

var anchorPattern = regexp.MustCompile(`MUTANT_U2_[A-Z0-9_]+`)

var environmentAuthoritySelectors = stringSet(
	"Clearenv", "Environ", "ExpandEnv", "Getenv", "LookupEnv", "Setenv", "Unsetenv",
	"UserCacheDir", "UserConfigDir", "UserHomeDir",
)

var processAuthoritySelectors = stringSet(
	"Args", "Executable", "Exit", "FindProcess", "Getpid", "Getppid", "Interrupt", "Kill",
	"ProcAttr", "Process", "ProcessState", "Signal", "StartProcess", "Stderr", "Stdin", "Stdout",
)

var expectedExecSelectors = map[string]int{
	"internal/gitobj/git.go\x00Cmd":                    1,
	"internal/gitobj/git.go\x00CommandContext":         1,
	"internal/world/process_darwin.go\x00Cmd":          3,
	"internal/world/process_darwin.go\x00ExitError":    1,
	"internal/world/process_darwin_test.go\x00Command": 1,
}

const publicationCgoPreamble = `/*
#include <errno.h>
#include <stdlib.h>
#include <sys/stdio.h>

static int countershape_rename_exclusive(const char *from, const char *to, int exclusive) {
	unsigned int flags = RENAME_NOFOLLOW_ANY;
	if (exclusive) flags |= RENAME_EXCL;
	if (renamex_np(from, to, flags) == 0) return 0;
	return errno;
}
*/`

type helperSpec struct {
	packageName string
	digest      string
	imports     map[string]bool
}

var exactTestkitHelpers = map[string]helperSpec{
	"testkit/gitrepo/gitrepo.go": {
		packageName: "gitrepo",
		digest:      "4a8ea9c5ceca7ae895cbf7249b845d441bc4bdaccf1a34adc67ffb12feb8684e",
		imports: stringSet(
			"bytes", "context", "encoding/hex", "errors", "fmt", "os", "os/exec", "path/filepath", "sort", "strings",
		),
	},
	"testkit/processfixture/main.go": {
		packageName: "main",
		digest:      "e8b6c9afef82717f94ed7b7701254f242c4968f3f1ad1d1654df33834d0fca00",
		imports: stringSet(
			"encoding/json", "errors", "fmt", "os", "os/exec", "os/signal", "path/filepath", "sort", "strconv", "strings", "syscall", "time",
		),
	},
}

type sourceFile struct {
	rel         string
	packageName string
	isTest      bool
	raw         []byte
	file        *ast.File
	aliases     map[string]string
}

type result struct {
	Packages int
	GoFiles  int
	Anchors  int
}

func stringSet(values ...string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func inspect(root string) (result, error) {
	root = filepath.Clean(root)
	if info, err := os.Lstat(root); err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return result{}, refuse(codeRootType, ".", "repository root must be a real directory")
	}
	if err := inspectModule(root); err != nil {
		return result{}, err
	}
	if err := inspectAnalyzerTopology(root); err != nil {
		return result{}, err
	}
	helperCount, err := inspectTestkitHelpers(root)
	if err != nil {
		return result{}, err
	}

	var sources []*sourceFile
	packages := []string{"gitobj", "world"}
	for _, packageName := range packages {
		found, err := inspectPackage(root, packageName)
		if err != nil {
			return result{}, err
		}
		sources = append(sources, found...)
	}
	if err := inspectAnchors(sources); err != nil {
		return result{}, err
	}
	if err := inspectExecutionSurfaces(sources); err != nil {
		return result{}, err
	}
	if err := inspectPublicationCgoPreamble(sources); err != nil {
		return result{}, err
	}
	if err := inspectGitVocabulary(sources); err != nil {
		return result{}, err
	}
	if err := inspectSelectedTreeBoundary(sources); err != nil {
		return result{}, err
	}
	if err := inspectProductionExecute(sources); err != nil {
		return result{}, err
	}
	return result{Packages: len(packages), GoFiles: len(sources) + helperCount, Anchors: len(requiredAnchors)}, nil
}

func inspectModule(root string) error {
	path := filepath.Join(root, "go.mod")
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return refuse(codeModulePolicy, "go.mod", "module control must be a regular non-symlink file")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return refuse(codeModulePolicy, "go.mod", "cannot read module control")
	}
	var lines []string
	for _, rawLine := range strings.Split(string(raw), "\n") {
		if line := strings.TrimSpace(rawLine); line != "" {
			lines = append(lines, line)
		}
	}
	want := []string{"module " + expectedModule, "go " + expectedGo}
	if strings.Join(lines, "\x00") != strings.Join(want, "\x00") {
		return refuse(codeModulePolicy, "go.mod", "module must contain only the pinned module and Go directives")
	}
	return nil
}

func inspectAnalyzerTopology(root string) error {
	directory := filepath.Join(root, "tools", "u2boundary")
	info, err := os.Lstat(directory)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return refuse(codeTopology, "tools/u2boundary", "analyzer directory must be real")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 || entries[0].Name() != "main.go" {
		return refuse(codeTopology, "tools/u2boundary", "exactly main.go is admitted as analyzer compiler input")
	}
	mainInfo, err := os.Lstat(filepath.Join(directory, "main.go"))
	if err != nil || mainInfo.Mode()&os.ModeSymlink != 0 || !mainInfo.Mode().IsRegular() {
		return refuse(codeTopology, "tools/u2boundary/main.go", "analyzer entrypoint must be a regular non-symlink file")
	}
	return nil
}

func inspectTestkitHelpers(root string) (int, error) {
	testkitRoot := filepath.Join(root, "testkit")
	info, err := os.Lstat(testkitRoot)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return 0, refuse(codeTestkitBoundary, "testkit", "testkit root must be a real directory")
	}
	rootEntries, err := os.ReadDir(testkitRoot)
	if err != nil || len(rootEntries) != 2 || rootEntries[0].Name() != "gitrepo" || rootEntries[1].Name() != "processfixture" {
		return 0, refuse(codeTestkitBoundary, "testkit", "exactly the two reviewed helper directories are admitted")
	}

	paths := make([]string, 0, len(exactTestkitHelpers))
	for rel := range exactTestkitHelpers {
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	for _, rel := range paths {
		spec := exactTestkitHelpers[rel]
		directory := filepath.Dir(filepath.Join(root, filepath.FromSlash(rel)))
		directoryInfo, statErr := os.Lstat(directory)
		if statErr != nil || directoryInfo.Mode()&os.ModeSymlink != 0 || !directoryInfo.IsDir() {
			return 0, refuse(codeTestkitBoundary, filepath.ToSlash(filepath.Dir(rel)), "helper directory must be real")
		}
		entries, readErr := os.ReadDir(directory)
		if readErr != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(rel) {
			return 0, refuse(codeTestkitBoundary, filepath.ToSlash(filepath.Dir(rel)), "helper directory differs from its exact one-file manifest")
		}
		path := filepath.Join(root, filepath.FromSlash(rel))
		fileInfo, statErr := os.Lstat(path)
		if statErr != nil || fileInfo.Mode()&os.ModeSymlink != 0 || !fileInfo.Mode().IsRegular() {
			return 0, refuse(codeTestkitBoundary, rel, "helper must be a regular non-symlink file")
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return 0, refuse(codeTestkitBoundary, rel, "cannot read helper source")
		}
		digest := sha256.Sum256(raw)
		if fmt.Sprintf("%x", digest) != spec.digest {
			return 0, refuse(codeTestkitBoundary, rel, "helper bytes differ from the reviewed SHA-256-frozen source")
		}
		parsed, parseErr := parser.ParseFile(token.NewFileSet(), path, raw, parser.AllErrors|parser.ParseComments)
		if parseErr != nil || parsed.Name == nil || parsed.Name.Name != spec.packageName {
			return 0, refuse(codeTestkitBoundary, rel, "helper is not the exact parsed package")
		}
		imports := map[string]int{}
		for _, imported := range parsed.Imports {
			pathValue, unquoteErr := strconv.Unquote(imported.Path.Value)
			if unquoteErr != nil || imported.Name != nil || !spec.imports[pathValue] {
				return 0, refuse(codeTestkitBoundary, rel, "helper import surface differs from the exact positive policy")
			}
			imports[pathValue]++
		}
		if len(imports) != len(spec.imports) {
			return 0, refuse(codeTestkitBoundary, rel, "helper import multiset differs from the reviewed source")
		}
		for pathValue := range spec.imports {
			if imports[pathValue] != 1 {
				return 0, refuse(codeTestkitBoundary, rel, "helper import is absent or duplicated: "+pathValue)
			}
		}
	}
	return len(paths), nil
}

func inspectPackage(root, packageName string) ([]*sourceFile, error) {
	directory := filepath.Join(root, "internal", packageName)
	info, err := os.Lstat(directory)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, refuse(codeTopology, filepath.ToSlash(filepath.Join("internal", packageName)), "package boundary must be a real directory")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, refuse(codeTopology, filepath.ToSlash(filepath.Join("internal", packageName)), "cannot enumerate package boundary")
	}
	expected := stringSet(packageFiles[packageName]...)
	if len(entries) != len(expected) {
		return nil, refuse(codeTopology, filepath.ToSlash(filepath.Join("internal", packageName)), "package file count differs from the reviewed manifest")
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	var sources []*sourceFile
	for _, name := range names {
		rel := filepath.ToSlash(filepath.Join("internal", packageName, name))
		path := filepath.Join(directory, name)
		fileInfo, statErr := os.Lstat(path)
		if statErr != nil || fileInfo.Mode()&os.ModeSymlink != 0 || !fileInfo.Mode().IsRegular() {
			return nil, refuse(codeTopology, rel, "source artifact must be a regular non-symlink file")
		}
		if !expected[name] {
			return nil, refuse(codeTopology, rel, "source file is outside the reviewed manifest")
		}
		source, parseErr := parseSource(path, rel, packageName)
		if parseErr != nil {
			return nil, parseErr
		}
		sources = append(sources, source)
	}
	return sources, nil
}

func parseSource(path, rel, packageName string) (*sourceFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, refuse(codeSourceMalformed, rel, "cannot read source")
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, raw, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return nil, refuse(codeSourceMalformed, rel, "Go parser rejected source")
	}
	isTest := strings.HasSuffix(rel, "_test.go")
	wantPackage := packageName
	if packageName == "gitobj" && externalGitobjTests[filepath.Base(rel)] {
		wantPackage = "gitobj_test"
	}
	if file.Name == nil || file.Name.Name != wantPackage {
		return nil, refuse(codePackageName, rel, "source package differs from the reviewed package boundary")
	}
	if err := inspectBuildTags(rel, raw, file); err != nil {
		return nil, err
	}
	aliases := map[string]string{}
	for _, imported := range file.Imports {
		pathValue, unquoteErr := strconv.Unquote(imported.Path.Value)
		if unquoteErr != nil || pathValue == "" || strings.IndexByte(pathValue, 0) >= 0 {
			return nil, refuse(codeImportLiteral, rel, "import path is not a valid Go string")
		}
		if imported.Name != nil && (imported.Name.Name == "." || imported.Name.Name == "_") {
			return nil, refuse(codeImportAlias, rel, "dot and blank imports are outside the proof surface")
		}
		if forbiddenAuthorityImports[pathValue] {
			return nil, refuse(codeAuthorityImport, rel, "U2 cannot import semantic U1 authority: "+pathValue)
		}
		if !importAllowed(packageName, pathValue, isTest) || !dangerousImportAtReviewedFile(rel, pathValue) {
			return nil, refuse(codeImportNotAllowed, rel, "import is outside the positive U2 policy: "+pathValue)
		}
		alias := filepath.Base(pathValue)
		if pathValue == "C" {
			alias = "C"
		}
		if imported.Name != nil {
			alias = imported.Name.Name
		}
		aliases[alias] = pathValue
	}
	return &sourceFile{rel: rel, packageName: packageName, isTest: isTest, raw: raw, file: file, aliases: aliases}, nil
}

func inspectBuildTags(rel string, raw []byte, file *ast.File) error {
	var tags []string
	for _, group := range file.Comments {
		for _, comment := range group.List {
			text := strings.TrimSpace(comment.Text)
			if strings.HasPrefix(text, "//go:build") {
				tags = append(tags, strings.TrimSpace(strings.TrimPrefix(text, "//go:build")))
			}
			if strings.HasPrefix(text, "// +build") {
				return refuse(codeBuildConstraint, rel, "legacy build constraints are not admitted")
			}
		}
	}
	want := expectedBuildTags[rel]
	if want == "" && len(tags) != 0 || want != "" && (len(tags) != 1 || tags[0] != want) {
		return refuse(codeBuildConstraint, rel, "build constraint differs from the exact reviewed platform split")
	}
	if want != "" && !bytes.HasPrefix(raw, []byte("//go:build "+want+"\n\n")) {
		return refuse(codeBuildConstraint, rel, "platform constraint is not the exact effective file prefix")
	}
	return nil
}

func importAllowed(packageName, pathValue string, isTest bool) bool {
	if productionImports[packageName][pathValue] {
		return true
	}
	return isTest && testImports[packageName][pathValue]
}

func dangerousImportAtReviewedFile(rel, pathValue string) bool {
	switch pathValue {
	case "os/exec":
		return rel == "internal/gitobj/git.go" || rel == "internal/world/process_darwin.go" || rel == "internal/world/process_darwin_test.go"
	case "runtime":
		return rel == "internal/world/process_darwin_test.go"
	case "syscall":
		return rel == "internal/gitobj/filesystem_darwin.go" || rel == "internal/gitobj/publish_darwin.go" ||
			rel == "internal/world/process_darwin.go" || rel == "internal/world/process_darwin_test.go" ||
			rel == "internal/world/process_mutation_darwin_test.go"
	case "unsafe", "C":
		return rel == "internal/gitobj/publish_darwin.go"
	default:
		return true
	}
}

func inspectAnchors(sources []*sourceFile) error {
	counts := map[string]int{}
	for _, source := range sources {
		for _, anchor := range anchorPattern.FindAllString(string(source.raw), -1) {
			expectedFile, present := requiredAnchors[anchor]
			if !present {
				return refuse(codeAnchorContract, source.rel, "unknown U2 mutation anchor: "+anchor)
			}
			if expectedFile != source.rel {
				return refuse(codeAnchorContract, source.rel, "mutation anchor moved from its reviewed authority edge: "+anchor)
			}
			counts[anchor]++
		}
	}
	if len(requiredAnchors) != 25 || len(counts) != len(requiredAnchors) {
		return refuse(codeAnchorContract, "internal", "U2 requires exactly 25 distinct mutation anchors")
	}
	for anchor := range requiredAnchors {
		if counts[anchor] != 1 {
			return refuse(codeAnchorContract, requiredAnchors[anchor], fmt.Sprintf("anchor %s occurs %d times, want exactly one", anchor, counts[anchor]))
		}
	}
	return nil
}

func inspectExecutionSurfaces(sources []*sourceFile) error {
	gitExecCalls := 0
	fixtureExecCalls := 0
	worldCommandLiterals := 0
	shellLiterals := 0
	environCalls := 0
	getenvCalls := 0
	execSelectors := map[string]int{}

	for _, source := range sources {
		if err := walkWithAncestors(source.file, func(node ast.Node, ancestors []ast.Node) error {
			switch typed := node.(type) {
			case *ast.CallExpr:
				importPath, selector, qualified := importedSelectorCall(typed, source.aliases)
				if qualified && importPath == "os/exec" {
					switch source.rel {
					case "internal/gitobj/git.go":
						if selector != "CommandContext" || !exactGitExecCall(typed) {
							return refuse(codeExecutionSurface, source.rel, "Git may spawn only its exact admitted CommandContext capability")
						}
						gitExecCalls++
					case "internal/world/process_darwin_test.go":
						if selector != "Command" || !exactFixtureBuildCall(typed) {
							return refuse(codeExecutionSurface, source.rel, "test process authority differs from the exact fixture build")
						}
						fixtureExecCalls++
					default:
						return refuse(codeExecutionSurface, source.rel, "arbitrary os/exec call is outside the U2 boundary")
					}
				}
				if qualified && importPath == "os" {
					switch selector {
					case "Environ":
						if source.rel != "internal/world/allocate.go" || len(typed.Args) != 0 ||
							!insideCondition(ancestors, "inheritAmbientEnvironment", false) {
							return refuse(codeEnvironmentSurface, source.rel, "os.Environ is allowed only under the exact mutation seam")
						}
						environCalls++
					case "Getenv":
						if source.rel != "internal/world/tools.go" || len(typed.Args) != 1 ||
							stringLiteral(typed.Args[0]) != "PATH" || !insideCondition(ancestors, "resolveToolsThroughAmbientPATH", false) {
							return refuse(codeEnvironmentSurface, source.rel, "ambient PATH is allowed only under the exact mutation seam")
						}
						getenvCalls++
					case "Exit", "StartProcess":
						return refuse(codeExecutionSurface, source.rel, "os process escape surface is forbidden")
					case "LookupEnv", "ExpandEnv", "Setenv", "Unsetenv", "Clearenv", "UserHomeDir":
						return refuse(codeEnvironmentSurface, source.rel, "ambient environment access is outside the two exact mutation seams")
					}
				}
				if qualified && importPath == "syscall" && (selector == "Exec" || selector == "ForkExec") {
					return refuse(codeExecutionSurface, source.rel, "syscall process replacement/spawn is forbidden")
				}
			case *ast.CompositeLit:
				if importedSelectorType(typed.Type, source.aliases, "os/exec", "Cmd") {
					if source.rel != "internal/world/process_darwin.go" {
						return refuse(codeExecutionSurface, source.rel, "exec.Cmd construction is outside the direct Darwin owner")
					}
					pathValue := compositeField(typed, "Path")
					switch {
					case stringLiteral(pathValue) == "/bin/sh":
						if !insideCondition(ancestors, "directExecOnly", true) {
							return refuse(codeExecutionSurface, source.rel, "shell path escaped the exact negative mutation seam")
						}
						worldCommandLiterals++
					case expressionChain(pathValue) == "request.tool.absolutePath":
						worldCommandLiterals++
					default:
						return refuse(codeExecutionSurface, source.rel, "exec.Cmd Path is not an admitted measured capability")
					}
				}
			case *ast.BasicLit:
				if !source.isTest && typed.Kind == token.STRING && stringLiteral(typed) == "/bin/sh" {
					if source.rel != "internal/world/process_darwin.go" || !insideCondition(ancestors, "directExecOnly", true) {
						return refuse(codeExecutionSurface, source.rel, "shell literal exists outside the exact mutation seam")
					}
					shellLiterals++
				}
			case *ast.SelectorExpr:
				identifier, ok := typed.X.(*ast.Ident)
				if !ok || typed.Sel == nil {
					break
				}
				importPath := source.aliases[identifier.Name]
				if importPath == "os/exec" {
					if err := inspectExecSelector(source, typed, ancestors); err != nil {
						return err
					}
					execSelectors[source.rel+"\x00"+typed.Sel.Name]++
				}
				if importPath == "os" && environmentAuthoritySelectors[typed.Sel.Name] {
					_, direct := directSelectorCall(typed, ancestors)
					if !direct || typed.Sel.Name != "Environ" && typed.Sel.Name != "Getenv" {
						return refuse(codeEnvironmentSurface, source.rel, "os environment authority must remain an exact direct mutation-seam call: "+typed.Sel.Name)
					}
				}
				if importPath == "os" && processAuthoritySelectors[typed.Sel.Name] {
					if typed.Sel.Name != "ProcessState" || source.rel != "internal/world/process_darwin.go" {
						return refuse(codeExecutionSurface, source.rel, "os process authority is outside the exact direct process owner: "+typed.Sel.Name)
					}
				}
				if importPath == "runtime" && !stringSet("Caller", "GOROOT")[typed.Sel.Name] {
					return refuse(codeExecutionSurface, source.rel, "runtime selector is outside the exact fixture-build introspection")
				}
				if importPath == "syscall" && !allowedSyscallSelector(source.rel, typed.Sel.Name) {
					return refuse(codeExecutionSurface, source.rel, "syscall selector is outside the exact process/filesystem vocabulary: "+typed.Sel.Name)
				}
				if importPath == "unsafe" && typed.Sel.Name != "Pointer" {
					return refuse(codeExecutionSurface, source.rel, "unsafe authority exceeds the reviewed C pointer conversion")
				}
				if importPath == "C" && !stringSet("CString", "free", "int", "countershape_rename_exclusive")[typed.Sel.Name] {
					return refuse(codeExecutionSurface, source.rel, "C authority exceeds exclusive Darwin publication")
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	if gitExecCalls != 1 || fixtureExecCalls != 1 || worldCommandLiterals != 2 || shellLiterals != 1 || environCalls != 1 || getenvCalls != 1 ||
		!sameCounts(execSelectors, expectedExecSelectors) {
		return refuse(codeExecutionSurface, "internal", fmt.Sprintf(
			"authority surface counts changed: git_exec=%d fixture_exec=%d world_cmd=%d shell=%d environ=%d getenv=%d exec_selectors=%d",
			gitExecCalls, fixtureExecCalls, worldCommandLiterals, shellLiterals, environCalls, getenvCalls, len(execSelectors),
		))
	}
	return nil
}

func inspectExecSelector(source *sourceFile, selector *ast.SelectorExpr, ancestors []ast.Node) error {
	_, direct := directSelectorCall(selector, ancestors)
	switch selector.Sel.Name {
	case "CommandContext":
		if source.rel != "internal/gitobj/git.go" || !direct {
			return refuse(codeExecutionSurface, source.rel, "exec.CommandContext must remain the exact direct closed-Git invocation")
		}
	case "Command":
		if source.rel != "internal/world/process_darwin_test.go" || !direct {
			return refuse(codeExecutionSurface, source.rel, "exec.Command must remain the exact direct fixture-build invocation")
		}
	case "Cmd":
		if source.rel != "internal/gitobj/git.go" && source.rel != "internal/world/process_darwin.go" {
			return refuse(codeExecutionSurface, source.rel, "exec.Cmd type authority is outside its exact owners")
		}
	case "ExitError":
		if source.rel != "internal/world/process_darwin.go" {
			return refuse(codeExecutionSurface, source.rel, "exec.ExitError is outside the direct Darwin owner")
		}
	default:
		return refuse(codeExecutionSurface, source.rel, "os/exec selector is outside the exact reviewed vocabulary: "+selector.Sel.Name)
	}
	return nil
}

func inspectPublicationCgoPreamble(sources []*sourceFile) error {
	source := sourceByPath(sources, "internal/gitobj/publish_darwin.go")
	if source == nil {
		return refuse(codeCgoBoundary, "internal/gitobj/publish_darwin.go", "exclusive publication source is absent")
	}
	wantPrefix := "//go:build darwin && cgo\n\npackage gitobj\n\n" + publicationCgoPreamble + "\nimport \"C\"\n\n"
	if !bytes.HasPrefix(source.raw, []byte(wantPrefix)) {
		return refuse(codeCgoBoundary, source.rel, "cgo includes and exclusive rename implementation differ from the exact reviewed preamble")
	}
	cImports := 0
	for _, imported := range source.file.Imports {
		pathValue, err := strconv.Unquote(imported.Path.Value)
		if err == nil && pathValue == "C" {
			if imported.Name != nil {
				return refuse(codeCgoBoundary, source.rel, "cgo import aliases are outside the exact preamble boundary")
			}
			cImports++
		}
	}
	if cImports != 1 || bytes.Count(source.raw, []byte(publicationCgoPreamble)) != 1 {
		return refuse(codeCgoBoundary, source.rel, "exactly one parsed C import and one frozen preamble are required")
	}
	return nil
}

func allowedSyscallSelector(rel, selector string) bool {
	allowed := map[string]map[string]bool{
		"internal/gitobj/filesystem_darwin.go": stringSet("Stat_t"),
		"internal/gitobj/publish_darwin.go":    stringSet("Errno"),
		"internal/world/process_darwin.go": stringSet(
			"EPERM", "ESRCH", "Getpgid", "Kill", "SIGKILL", "SIGTERM", "Signal", "SysProcAttr", "WaitStatus",
		),
		"internal/world/process_darwin_test.go": stringSet(
			"EPERM", "ESRCH", "Kill", "SIGKILL", "SIGTERM", "Signal",
		),
		"internal/world/process_mutation_darwin_test.go": stringSet("ESRCH", "Kill", "SIGKILL"),
	}
	return allowed[rel][selector]
}

func exactGitExecCall(call *ast.CallExpr) bool {
	return call.Ellipsis.IsValid() && len(call.Args) == 3 && expressionChain(call.Args[0]) == "ctx" &&
		expressionChain(call.Args[1]) == "g.executable" && expressionChain(call.Args[2]) == "closedArgs"
}

func exactFixtureBuildCall(call *ast.CallExpr) bool {
	if call.Ellipsis.IsValid() || len(call.Args) != 6 || expressionChain(call.Args[0]) != "goExecutable" ||
		expressionChain(call.Args[4]) != "output" {
		return false
	}
	want := []string{"build", "-trimpath", "-o", "./testkit/processfixture"}
	got := []string{stringLiteral(call.Args[1]), stringLiteral(call.Args[2]), stringLiteral(call.Args[3]), stringLiteral(call.Args[5])}
	return strings.Join(got, "\x00") == strings.Join(want, "\x00")
}

func inspectGitVocabulary(sources []*sourceFile) error {
	source := sourceByPath(sources, "internal/gitobj/git.go")
	if source == nil {
		return refuse(codeGitSurface, "internal/gitobj/git.go", "Git authority source is absent")
	}
	want := map[string]int{
		"runner.run\x00config\x00--local\x00--null\x00--name-only\x00--list\x00--no-includes": 1,
		"runner.run\x00version": 1,
		"runner.run\x00rev-parse\x00--path-format=absolute\x00--git-common-dir":      1,
		"runner.run\x00rev-parse\x00--path-format=absolute\x00--git-path\x00objects": 1,
		"runner.run\x00rev-parse\x00--show-object-format=storage":                    1,
		"r.state.run\x00rev-parse\x00--verify\x00--end-of-options":                   1,
		"state.run\x00cat-file\x00-s":                                                1,
		"state.runner.command\x00cat-file":                                           1,
	}
	got := map[string]int{}
	dynamicWrappers := map[string]int{}
	if err := walkWithAncestors(source.file, func(node ast.Node, ancestors []ast.Node) error {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok || selector.Sel == nil || selector.Sel.Name != "run" && selector.Sel.Name != "command" {
			return nil
		}
		call, direct := directSelectorCall(selector, ancestors)
		if !direct {
			return refuse(codeGitSurface, source.rel, "Git command authority escaped a direct invocation through a function value or alias")
		}
		receiver := expressionChain(selector.X)
		if call.Ellipsis.IsValid() {
			key := receiver + "." + selector.Sel.Name
			validForward := key == "g.command" && len(call.Args) == 3 && expressionChain(call.Args[2]) == "args" ||
				key == "state.runner.run" && len(call.Args) == 4 && expressionChain(call.Args[3]) == "args"
			if !validForward {
				return refuse(codeGitSurface, source.rel, "dynamic Git argv escaped its two exact private forwarding edges")
			}
			dynamicWrappers[key]++
			return nil
		}
		sequence := firstStringSequence(call.Args)
		if len(sequence) == 0 {
			return refuse(codeGitSurface, source.rel, "Git authority call has no statically closed subcommand")
		}
		key := receiver + "." + selector.Sel.Name + "\x00" + strings.Join(sequence, "\x00")
		if _, admitted := want[key]; !admitted {
			return refuse(codeGitSurface, source.rel, "Git receiver/subcommand tuple is outside the exact plumbing vocabulary: "+receiver+"."+selector.Sel.Name+" "+strings.Join(sequence, " "))
		}
		got[key]++
		return nil
	}); err != nil {
		return err
	}
	if !sameCounts(got, want) || dynamicWrappers["g.command"] != 1 || dynamicWrappers["state.runner.run"] != 1 || len(dynamicWrappers) != 2 {
		return refuse(codeGitSurface, source.rel, "Git plumbing call multiset differs from the reviewed closed vocabulary")
	}
	for _, other := range sources {
		if other.isTest || other.rel == source.rel || other.packageName != "gitobj" {
			continue
		}
		var escaped bool
		ast.Inspect(other.file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if ok && selector.Sel != nil && (selector.Sel.Name == "run" || selector.Sel.Name == "command") {
				escaped = true
				return false
			}
			return true
		})
		if escaped {
			return refuse(codeGitSurface, other.rel, "Git command authority moved outside git.go")
		}
	}
	return nil
}

func sameCounts(got, want map[string]int) bool {
	if len(got) != len(want) {
		return false
	}
	for key, count := range want {
		if got[key] != count {
			return false
		}
	}
	return true
}

func firstStringSequence(arguments []ast.Expr) []string {
	start := -1
	for index, argument := range arguments {
		if _, ok := argument.(*ast.BasicLit); ok && stringLiteral(argument) != "" {
			start = index
			break
		}
	}
	if start < 0 {
		return nil
	}
	var result []string
	for _, argument := range arguments[start:] {
		value, ok := argument.(*ast.BasicLit)
		if !ok || value.Kind != token.STRING {
			break
		}
		decoded, err := strconv.Unquote(value.Value)
		if err != nil {
			return nil
		}
		result = append(result, decoded)
	}
	return result
}

func inspectSelectedTreeBoundary(sources []*sourceFile) error {
	source := sourceByPath(sources, "internal/gitobj/types.go")
	if source == nil {
		return refuse(codeDigestBoundary, "internal/gitobj/types.go", "pre-plan identity source is absent")
	}
	selectedType := findType(source.file, "SelectedTreeSet")
	structure, ok := selectedType.(*ast.StructType)
	if !ok || strings.Join(structFieldNames(structure), "\x00") != "digest\x00pinned" {
		return refuse(codeDigestBoundary, source.rel, "SelectedTreeSet may contain only digest and pinned authority")
	}

	constructors := findFunctions(source.file, "SelectTrees", "")
	if len(constructors) != 1 {
		return refuse(codeDigestBoundary, source.rel, "exactly one SelectTrees constructor is required")
	}
	constructor := constructors[0]
	if constructor.Type.Params == nil || len(constructor.Type.Params.List) != 1 ||
		len(constructor.Type.Params.List[0].Names) != 1 || constructor.Type.Params.List[0].Names[0].Name != "pinned" {
		return refuse(codeDigestBoundary, source.rel, "SelectTrees accepts only the variadic pinned-tree authority")
	}
	ellipsis, ok := constructor.Type.Params.List[0].Type.(*ast.Ellipsis)
	if !ok || expressionChain(ellipsis.Elt) != "PinnedTree" {
		return refuse(codeDigestBoundary, source.rel, "SelectTrees input is not variadic PinnedTree")
	}
	forbiddenFragments := []string{
		"worldplan", "candidatekey", "repositoryfingerprint", "fingerprint", "policy",
		"inspectedtree", "inspection", "portabletreedigest",
	}
	var forbiddenIdentifier string
	var forbiddenCall string
	ast.Inspect(constructor.Body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.Ident:
			lower := strings.ToLower(typed.Name)
			for _, fragment := range forbiddenFragments {
				if strings.Contains(lower, fragment) {
					forbiddenIdentifier = typed.Name
					return false
				}
			}
		case *ast.CallExpr:
			name := expressionChain(typed.Fun)
			// []PinnedTree(nil) is the exact zero-length slice conversion used by
			// the independently checked append-copy expression below; it is a type
			// conversion, not a helper capable of importing post-selection state.
			admitted := stringSet("len", "append", "make", "string", "refuse", "sort.Slice", "digestIdentity", "candidate.Valid", "[]PinnedTree")[name] ||
				name == "candidate.identity.String" || strings.HasPrefix(name, "copyPinned[") && strings.HasSuffix(name, "].identity.String")
			if !admitted {
				forbiddenCall = name
				return false
			}
		}
		return forbiddenIdentifier == "" && forbiddenCall == ""
	})
	if forbiddenIdentifier != "" || forbiddenCall != "" {
		return refuse(codeDigestBoundary, source.rel, "SelectTrees references post-selection authority/helper: "+forbiddenIdentifier+forbiddenCall)
	}

	identityFields := []string{}
	identityValuesValid := false
	pinnedMutation := false
	copyPinnedCopies := 0
	identitySorts := 0
	membersMake := 0
	membersAppend := 0
	digestCalls := 0
	populatedReturns := 0
	ast.Inspect(constructor.Body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.AssignStmt:
			for _, left := range typed.Lhs {
				if nestedAssignmentRootedIn(left, "pinned", "copyPinned", "candidate") {
					pinnedMutation = true
				}
			}
			if !exactPinnedCopyAssignment(typed) {
				for _, right := range typed.Rhs {
					if carriesPinnedContainer(right) {
						pinnedMutation = true
					}
				}
			}
			if len(typed.Lhs) == 1 && expressionChain(typed.Lhs[0]) == "copyPinned" && len(typed.Rhs) == 1 {
				call, ok := typed.Rhs[0].(*ast.CallExpr)
				if ok && expressionChain(call.Fun) == "append" && call.Ellipsis.IsValid() && len(call.Args) == 2 &&
					expressionChain(call.Args[0]) == "[]PinnedTree(nil)" && expressionChain(call.Args[1]) == "pinned" {
					copyPinnedCopies++
				}
			}
			if len(typed.Lhs) == 1 && expressionChain(typed.Lhs[0]) == "identity" && len(typed.Rhs) == 1 {
				if composite, ok := typed.Rhs[0].(*ast.CompositeLit); ok {
					if identityStruct, ok := composite.Type.(*ast.StructType); ok {
						identityFields = structFieldNames(identityStruct)
						identityValuesValid = len(composite.Elts) == 4 && expressionChain(composite.Elts[0]) == "domain.SchemaVersion" &&
							stringLiteral(composite.Elts[1]) == "SelectedTreeSet" && expressionChain(composite.Elts[2]) == "len(members)" &&
							expressionChain(composite.Elts[3]) == "members"
					}
				}
			}
			if len(typed.Lhs) == 1 && expressionChain(typed.Lhs[0]) == "members" && len(typed.Rhs) == 1 {
				call, ok := typed.Rhs[0].(*ast.CallExpr)
				if ok && expressionChain(call.Fun) == "make" {
					membersMake++
				}
				if ok && expressionChain(call.Fun) == "append" && len(call.Args) == 2 && expressionChain(call.Args[0]) == "members" &&
					expressionChain(call.Args[1]) == "candidate.identity.String()" {
					membersAppend++
				}
			}
		case *ast.CallExpr:
			if expressionChain(typed.Fun) == "append" && len(typed.Args) > 0 && expressionRootedIn(typed.Args[0], "pinned", "copyPinned") {
				pinnedMutation = true
			}
			if expressionChain(typed.Fun) == "sort.Slice" && exactIdentitySort(typed) {
				identitySorts++
			}
			if expressionChain(typed.Fun) == "digestIdentity" && len(typed.Args) == 2 &&
				stringLiteral(typed.Args[0]) == "SelectedTreeSet" && expressionChain(typed.Args[1]) == "identity" {
				digestCalls++
			}
		case *ast.CompositeLit:
			if expressionChain(typed.Type) == "SelectedTreeSet" && len(typed.Elts) > 0 {
				fields := compositeFieldNames(typed)
				if strings.Join(fields, "\x00") != "digest\x00pinned" || expressionChain(compositeField(typed, "digest")) != "digest" ||
					expressionChain(compositeField(typed, "pinned")) != "copyPinned" {
					populatedReturns = -100
				} else {
					populatedReturns++
				}
			}
		case *ast.IncDecStmt:
			if nestedAssignmentRootedIn(typed.X, "pinned", "copyPinned", "candidate") {
				pinnedMutation = true
			}
		case *ast.UnaryExpr:
			if typed.Op == token.AND && expressionRootedIn(typed.X, "pinned", "copyPinned", "candidate") {
				pinnedMutation = true
			}
		}
		return true
	})
	if strings.Join(identityFields, "\x00") != "SchemaVersion\x00Kind\x00Count\x00Members" || !identityValuesValid ||
		pinnedMutation || copyPinnedCopies != 1 || identitySorts != 1 || membersMake != 1 || membersAppend != 1 || digestCalls != 1 || populatedReturns != 1 {
		return refuse(codeDigestBoundary, source.rel, "SelectedTreeSet digest input differs from count plus sorted pinned identities")
	}

	selectedDigests := findFunctions(source.file, "Digest", "SelectedTreeSet")
	if len(selectedDigests) != 1 || !exactReturn(selectedDigests[0], "s.digest") {
		return refuse(codeDigestBoundary, source.rel, "SelectedTreeSet.Digest must return only its sealed digest")
	}
	declarationDigests := findFunctions(source.file, "Digest", "CandidateSetDeclaration")
	if len(declarationDigests) != 1 || !exactReturn(declarationDigests[0], "s.selected.digest") {
		return refuse(codeDigestBoundary, source.rel, "CandidateSetDeclaration.Digest must preserve the pre-plan selected digest")
	}

	declarationConstructors := findFunctions(source.file, "NewCandidateSet", "")
	if len(declarationConstructors) != 1 {
		return refuse(codeDigestBoundary, source.rel, "exactly one inspected candidate-set constructor is required")
	}
	populatedDeclarations := 0
	assignmentPollution := false
	ast.Inspect(declarationConstructors[0].Body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.AssignStmt:
			for _, left := range typed.Lhs {
				chain := strings.ToLower(expressionChain(left))
				if chain == "selected" || strings.Contains(chain, "selected") && strings.HasSuffix(chain, ".digest") {
					assignmentPollution = true
				}
			}
		case *ast.UnaryExpr:
			if typed.Op == token.AND && expressionChain(typed.X) == "selected" {
				assignmentPollution = true
			}
		case *ast.CompositeLit:
			if expressionChain(typed.Type) == "CandidateSetDeclaration" && len(typed.Elts) > 0 {
				if strings.Join(compositeFieldNames(typed), "\x00") != "selected\x00policy\x00candidates" ||
					expressionChain(compositeField(typed, "selected")) != "selected" {
					assignmentPollution = true
				}
				populatedDeclarations++
			}
		}
		return true
	})
	if assignmentPollution || populatedDeclarations != 1 {
		return refuse(codeDigestBoundary, source.rel, "inspection/policy authority can alter the pre-plan selected digest")
	}
	return nil
}

func exactIdentitySort(call *ast.CallExpr) bool {
	if call == nil || len(call.Args) != 2 || expressionChain(call.Args[0]) != "copyPinned" {
		return false
	}
	function, ok := call.Args[1].(*ast.FuncLit)
	if !ok || function.Body == nil || len(function.Body.List) != 1 {
		return false
	}
	returned, ok := function.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return false
	}
	comparison, ok := returned.Results[0].(*ast.BinaryExpr)
	return ok && comparison.Op == token.LSS && expressionChain(comparison.X) == "copyPinned[a].identity.String()" &&
		expressionChain(comparison.Y) == "copyPinned[b].identity.String()"
}

func exactPinnedCopyAssignment(statement *ast.AssignStmt) bool {
	if statement == nil || len(statement.Lhs) != 1 || expressionChain(statement.Lhs[0]) != "copyPinned" || len(statement.Rhs) != 1 {
		return false
	}
	call, ok := statement.Rhs[0].(*ast.CallExpr)
	return ok && expressionChain(call.Fun) == "append" && call.Ellipsis.IsValid() && len(call.Args) == 2 &&
		expressionChain(call.Args[0]) == "[]PinnedTree(nil)" && expressionChain(call.Args[1]) == "pinned"
}

func nestedAssignmentRootedIn(expression ast.Expr, names ...string) bool {
	if _, identifier := expression.(*ast.Ident); identifier {
		return false
	}
	return expressionRootedIn(expression, names...)
}

func expressionRootedIn(expression ast.Expr, names ...string) bool {
	allowed := stringSet(names...)
	var root func(ast.Expr) string
	root = func(candidate ast.Expr) string {
		switch typed := candidate.(type) {
		case *ast.Ident:
			return typed.Name
		case *ast.SelectorExpr:
			return root(typed.X)
		case *ast.IndexExpr:
			return root(typed.X)
		case *ast.IndexListExpr:
			return root(typed.X)
		case *ast.SliceExpr:
			return root(typed.X)
		case *ast.ParenExpr:
			return root(typed.X)
		case *ast.StarExpr:
			return root(typed.X)
		case *ast.UnaryExpr:
			return root(typed.X)
		default:
			return ""
		}
	}
	return allowed[root(expression)]
}

func carriesPinnedContainer(expression ast.Expr) bool {
	switch typed := expression.(type) {
	case *ast.Ident:
		return typed.Name == "pinned" || typed.Name == "copyPinned"
	case *ast.ParenExpr:
		return carriesPinnedContainer(typed.X)
	case *ast.SliceExpr:
		return expressionRootedIn(typed, "pinned", "copyPinned")
	case *ast.UnaryExpr:
		return typed.Op == token.AND && expressionRootedIn(typed.X, "pinned", "copyPinned")
	default:
		return false
	}
}

func inspectProductionExecute(sources []*sourceFile) error {
	source := sourceByPath(sources, "internal/world/api.go")
	if source == nil {
		return refuse(codeMaterializerBoundary, "internal/world/api.go", "production Execute source is absent")
	}
	functions := findFunctions(source.file, "Execute", "")
	if len(functions) != 1 || functions[0].Body == nil || len(functions[0].Body.List) != 1 {
		return refuse(codeMaterializerBoundary, source.rel, "production must expose exactly one single-return Execute")
	}
	returned, ok := functions[0].Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return refuse(codeMaterializerBoundary, source.rel, "Execute must directly return its private implementation")
	}
	call, ok := returned.Results[0].(*ast.CallExpr)
	if !ok || expressionChain(call.Fun) != "executeWithMaterializer" || len(call.Args) != 3 ||
		expressionChain(call.Args[0]) != "ctx" || expressionChain(call.Args[1]) != "request" {
		return refuse(codeMaterializerBoundary, source.rel, "Execute is not a direct private materializer call")
	}
	materializer, ok := call.Args[2].(*ast.CompositeLit)
	if !ok || expressionChain(materializer.Type) != "gitobj.DefaultMaterializer" || len(materializer.Elts) != 0 {
		return refuse(codeMaterializerBoundary, source.rel, "production Execute is not hard-wired to DefaultMaterializer")
	}
	productionCalls := 0
	for _, candidate := range sources {
		if candidate.packageName != "world" || candidate.isTest {
			continue
		}
		ast.Inspect(candidate.file, func(node ast.Node) bool {
			if candidateCall, ok := node.(*ast.CallExpr); ok && expressionChain(candidateCall.Fun) == "executeWithMaterializer" {
				productionCalls++
			}
			return true
		})
	}
	if productionCalls != 1 {
		return refuse(codeMaterializerBoundary, source.rel, "private materializer injection has another production caller")
	}
	implementations := findFunctions(source.file, "executeWithMaterializer", "")
	if len(implementations) != 1 || implementations[0].Type.Params == nil || len(implementations[0].Type.Params.List) != 3 ||
		len(implementations[0].Type.Params.List[2].Names) != 1 || implementations[0].Type.Params.List[2].Names[0].Name != "source" ||
		expressionChain(implementations[0].Type.Params.List[2].Type) != "materializer" {
		return refuse(codeMaterializerBoundary, source.rel, "private implementation must take exactly one private materializer as its third capability")
	}
	interfaceType, ok := findType(source.file, "materializer").(*ast.InterfaceType)
	if !ok || interfaceType.Methods == nil || len(interfaceType.Methods.List) != 1 ||
		len(interfaceType.Methods.List[0].Names) != 1 || interfaceType.Methods.List[0].Names[0].Name != "Materialize" {
		return refuse(codeMaterializerBoundary, source.rel, "private materializer interface differs from its one-method boundary")
	}
	materializerIdentifiers := 0
	materializeCalls := 0
	for _, candidate := range sources {
		if candidate.packageName != "world" || candidate.isTest {
			continue
		}
		if err := walkWithAncestors(candidate.file, func(node ast.Node, ancestors []ast.Node) error {
			switch typed := node.(type) {
			case *ast.Ident:
				if typed.Name == "materializer" {
					materializerIdentifiers++
				}
			case *ast.SelectorExpr:
				if typed.Sel == nil {
					return nil
				}
				if identifier, imported := typed.X.(*ast.Ident); imported && candidate.aliases[identifier.Name] == expectedModule+"/internal/gitobj" && typed.Sel.Name == "Materializer" {
					return refuse(codeMaterializerBoundary, candidate.rel, "exported gitobj.Materializer cannot enter the world execution boundary")
				}
				if typed.Sel.Name != "Materialize" {
					return nil
				}
				invocation, direct := directSelectorCall(typed, ancestors)
				if candidate.rel != source.rel || !direct || expressionChain(typed.X) != "source" || enclosingFunctionName(ancestors) != "executeWithMaterializer" ||
					len(invocation.Args) != 3 || expressionChain(invocation.Args[0]) != "ctx" || expressionChain(invocation.Args[1]) != "request.Candidate" ||
					expressionChain(invocation.Args[2]) != "allocated.roots.candidateParent" {
					return refuse(codeMaterializerBoundary, candidate.rel, "materialization authority escaped the one private direct invocation")
				}
				materializeCalls++
			}
			return nil
		}); err != nil {
			return err
		}
	}
	if materializerIdentifiers != 2 || materializeCalls != 1 {
		return refuse(codeMaterializerBoundary, source.rel, "private materializer type/use or direct invocation count changed")
	}
	return nil
}

func sourceByPath(sources []*sourceFile, rel string) *sourceFile {
	for _, source := range sources {
		if source.rel == rel {
			return source
		}
	}
	return nil
}

func findFunctions(file *ast.File, name, receiver string) []*ast.FuncDecl {
	var result []*ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name == nil || function.Name.Name != name || receiverName(function) != receiver {
			continue
		}
		result = append(result, function)
	}
	return result
}

func receiverName(function *ast.FuncDecl) string {
	if function.Recv == nil || len(function.Recv.List) != 1 {
		return ""
	}
	typeExpression := function.Recv.List[0].Type
	if pointer, ok := typeExpression.(*ast.StarExpr); ok {
		typeExpression = pointer.X
	}
	return expressionChain(typeExpression)
}

func findType(file *ast.File, name string) ast.Expr {
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, specification := range general.Specs {
			typed, ok := specification.(*ast.TypeSpec)
			if ok && typed.Name != nil && typed.Name.Name == name {
				return typed.Type
			}
		}
	}
	return nil
}

func structFieldNames(structure *ast.StructType) []string {
	var names []string
	if structure == nil || structure.Fields == nil {
		return names
	}
	for _, field := range structure.Fields.List {
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
	}
	return names
}

func exactReturn(function *ast.FuncDecl, want string) bool {
	if function == nil || function.Body == nil || len(function.Body.List) != 1 {
		return false
	}
	returned, ok := function.Body.List[0].(*ast.ReturnStmt)
	return ok && len(returned.Results) == 1 && expressionChain(returned.Results[0]) == want
}

func compositeField(composite *ast.CompositeLit, name string) ast.Expr {
	if composite == nil {
		return nil
	}
	for _, element := range composite.Elts {
		keyed, ok := element.(*ast.KeyValueExpr)
		if ok && expressionChain(keyed.Key) == name {
			return keyed.Value
		}
	}
	return nil
}

func compositeFieldNames(composite *ast.CompositeLit) []string {
	var names []string
	for _, element := range composite.Elts {
		keyed, ok := element.(*ast.KeyValueExpr)
		if !ok {
			return nil
		}
		names = append(names, expressionChain(keyed.Key))
	}
	return names
}

func importedSelectorCall(call *ast.CallExpr, aliases map[string]string) (string, string, bool) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel == nil {
		return "", "", false
	}
	identifier, ok := selector.X.(*ast.Ident)
	if !ok {
		return "", "", false
	}
	pathValue, present := aliases[identifier.Name]
	return pathValue, selector.Sel.Name, present
}

func importedSelectorType(expression ast.Expr, aliases map[string]string, importPath, selectorName string) bool {
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok || selector.Sel == nil || selector.Sel.Name != selectorName {
		return false
	}
	identifier, ok := selector.X.(*ast.Ident)
	return ok && aliases[identifier.Name] == importPath
}

func directSelectorCall(selector *ast.SelectorExpr, ancestors []ast.Node) (*ast.CallExpr, bool) {
	if len(ancestors) == 0 {
		return nil, false
	}
	call, ok := ancestors[len(ancestors)-1].(*ast.CallExpr)
	return call, ok && call.Fun == selector
}

func enclosingFunctionName(ancestors []ast.Node) string {
	for index := len(ancestors) - 1; index >= 0; index-- {
		if function, ok := ancestors[index].(*ast.FuncDecl); ok && function.Name != nil {
			return function.Name.Name
		}
	}
	return ""
}

func expressionChain(expression ast.Expr) string {
	switch typed := expression.(type) {
	case nil:
		return ""
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		left := expressionChain(typed.X)
		if left == "" || typed.Sel == nil {
			return ""
		}
		return left + "." + typed.Sel.Name
	case *ast.ParenExpr:
		return expressionChain(typed.X)
	}
	var buffer bytes.Buffer
	if err := printer.Fprint(&buffer, token.NewFileSet(), expression); err != nil {
		return ""
	}
	return buffer.String()
}

func stringLiteral(expression ast.Expr) string {
	literal, ok := expression.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return ""
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		return ""
	}
	return value
}

func insideCondition(ancestors []ast.Node, name string, negated bool) bool {
	for index := len(ancestors) - 1; index >= 0; index-- {
		statement, ok := ancestors[index].(*ast.IfStmt)
		if !ok {
			continue
		}
		if negated {
			unary, ok := statement.Cond.(*ast.UnaryExpr)
			if ok && unary.Op == token.NOT && expressionChain(unary.X) == name {
				return true
			}
			continue
		}
		if conditionRequiresTrue(statement.Cond, name) {
			return true
		}
	}
	return false
}

func conditionRequiresTrue(expression ast.Expr, name string) bool {
	switch typed := expression.(type) {
	case *ast.Ident:
		return typed.Name == name
	case *ast.ParenExpr:
		return conditionRequiresTrue(typed.X, name)
	case *ast.BinaryExpr:
		// Only conjunction is safe: if either side requires the closed-false
		// mutation seam to be true, the guarded authority cannot execute in the
		// production profile. An OR expression would not provide that guarantee.
		return typed.Op == token.LAND &&
			(conditionRequiresTrue(typed.X, name) || conditionRequiresTrue(typed.Y, name))
	default:
		return false
	}
}

func walkWithAncestors(root ast.Node, visit func(ast.Node, []ast.Node) error) error {
	var stack []ast.Node
	var visitErr error
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			return false
		}
		if visitErr != nil {
			return false
		}
		visitErr = visit(node, stack)
		if visitErr != nil {
			return false
		}
		stack = append(stack, node)
		return true
	})
	return visitErr
}

func fixture(root, templateRoot string) error {
	if err := os.MkdirAll(filepath.Join(root, "tools", "u2boundary"), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+expectedModule+"\n\ngo "+expectedGo+"\n"), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "tools", "u2boundary", "main.go"), []byte("package main\n"), 0o600); err != nil {
		return err
	}
	for _, packageName := range []string{"gitobj", "world"} {
		directory := filepath.Join(root, "internal", packageName)
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return err
		}
		for _, filename := range packageFiles[packageName] {
			rel := filepath.ToSlash(filepath.Join("internal", packageName, filename))
			source := fixtureSource(rel, packageName, filename)
			if tag := expectedBuildTags[rel]; tag != "" {
				source = "//go:build " + tag + "\n\n" + source
			}
			for anchor, expectedFile := range requiredAnchors {
				if expectedFile == rel {
					source += "\n// " + anchor + "\n"
				}
			}
			if err := os.WriteFile(filepath.Join(directory, filename), []byte(source), 0o600); err != nil {
				return err
			}
		}
	}
	for rel := range exactTestkitHelpers {
		raw, err := os.ReadFile(filepath.Join(templateRoot, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func fixtureSource(rel, packageName, filename string) string {
	if rel == "internal/gitobj/publish_darwin.go" {
		return "package gitobj\n\n" + publicationCgoPreamble + `
import "C"

import (
	"syscall"
	"unsafe"
)

const exclusivePublicationRequired = true

func renameExclusive(from, to string) error {
	fromCString := C.CString(from)
	toCString := C.CString(to)
	defer C.free(unsafe.Pointer(fromCString))
	defer C.free(unsafe.Pointer(toCString))
	exclusive := C.int(0)
	if exclusivePublicationRequired { exclusive = 1 }
	if errorNumber := C.countershape_rename_exclusive(fromCString, toCString, exclusive); errorNumber != 0 {
		return syscall.Errno(errorNumber)
	}
	return nil
}
`
	}
	if rel == "internal/gitobj/git.go" {
		return `package gitobj

import (
	"context"
	"os/exec"
)

type closedGit struct{ executable string }
type repositoryState struct{ runner closedGit }
type Repository struct{ state *repositoryState }

func (g closedGit) command(ctx context.Context, inRepository bool, args ...string) *exec.Cmd {
	closedArgs := args
	_ = inRepository
	return exec.CommandContext(ctx, g.executable, closedArgs...)
}

func (g closedGit) run(ctx context.Context, inRepository bool, outputLimit int, args ...string) {
	_ = g.command(ctx, inRepository, args...)
	_ = outputLimit
}

func (state *repositoryState) run(ctx context.Context, outputLimit int, args ...string) {
	state.runner.run(ctx, true, outputLimit, args...)
}

func closedVocabulary(ctx context.Context, runner closedGit, state *repositoryState, r Repository, displayRef, objectType, oid string) {
	runner.run(ctx, true, 1, "config", "--local", "--null", "--name-only", "--list", "--no-includes")
	runner.run(ctx, false, 1, "version")
	runner.run(ctx, true, 1, "rev-parse", "--path-format=absolute", "--git-common-dir")
	runner.run(ctx, true, 1, "rev-parse", "--path-format=absolute", "--git-path", "objects")
	runner.run(ctx, true, 1, "rev-parse", "--show-object-format=storage")
	r.state.run(ctx, 1, "rev-parse", "--verify", "--end-of-options", displayRef+"^{commit}")
	state.run(ctx, 1, "cat-file", "-s", oid)
	_ = state.runner.command(ctx, true, "cat-file", objectType, oid)
}
`
	}
	if rel == "internal/gitobj/types.go" {
		return `package gitobj

import (
	"sort"
	"github.com/nelsonwerd/countershape/internal/domain"
)

type PinnedTree struct{ identity domain.Digest }
type Policy struct{}
type InspectedTree struct{}
type SelectedTreeSet struct {
	digest domain.Digest
	pinned []PinnedTree
}

func SelectTrees(pinned ...PinnedTree) (SelectedTreeSet, error) {
	copyPinned := append([]PinnedTree(nil), pinned...)
	sort.Slice(copyPinned, func(a, b int) bool { return copyPinned[a].identity.String() < copyPinned[b].identity.String() })
	members := make([]string, 0, len(copyPinned))
	for _, candidate := range copyPinned {
		members = append(members, candidate.identity.String())
	}
	identity := struct {
		SchemaVersion string
		Kind string
		Count int
		Members []string
	}{domain.SchemaVersion, "SelectedTreeSet", len(members), members}
	digest, err := digestIdentity("SelectedTreeSet", identity)
	if err != nil { return SelectedTreeSet{}, err }
	return SelectedTreeSet{digest: digest, pinned: copyPinned}, nil
}

func (s SelectedTreeSet) Digest() domain.Digest { return s.digest }

type CandidateSetDeclaration struct {
	selected SelectedTreeSet
	policy Policy
	candidates []InspectedTree
}

func NewCandidateSet(selected SelectedTreeSet, policy Policy, candidates ...InspectedTree) (CandidateSetDeclaration, error) {
	copyCandidates := append([]InspectedTree(nil), candidates...)
	return CandidateSetDeclaration{selected: selected, policy: policy, candidates: copyCandidates}, nil
}

func (s CandidateSetDeclaration) Digest() domain.Digest { return s.selected.digest }
`
	}
	if rel == "internal/world/allocate.go" {
		return `package world

import "os"

const inheritAmbientEnvironment = false

func buildEnvironment() {
	if inheritAmbientEnvironment {
		for range os.Environ() {}
	}
}
`
	}
	if rel == "internal/world/api.go" {
		return `package world

import (
	"context"
	"github.com/nelsonwerd/countershape/internal/gitobj"
)

type materializer interface {
	Materialize(context.Context, gitobj.BoundCandidate, string) (gitobj.MaterializationReceipt, error)
}

type Request struct{ Candidate gitobj.BoundCandidate }
type Result struct{}

func Execute(ctx context.Context, request Request) (Result, error) {
	return executeWithMaterializer(ctx, request, gitobj.DefaultMaterializer{})
}

func executeWithMaterializer(ctx context.Context, request Request, source materializer) (Result, error) {
	_, err := source.Materialize(ctx, request.Candidate, allocated.roots.candidateParent)
	return Result{}, err
}
`
	}
	if rel == "internal/world/process_darwin.go" {
		return `package world

import "os/exec"

const directExecOnly = true

type fixtureRequest struct{ tool struct{ absolutePath string } }

var fixtureExitError *exec.ExitError

func newDirectCommand(request fixtureRequest) *exec.Cmd {
	if !directExecOnly {
		return &exec.Cmd{Path: "/bin/sh"}
	}
	return &exec.Cmd{Path: request.tool.absolutePath}
}
`
	}
	if rel == "internal/world/process_darwin_test.go" {
		return `package world

import "os/exec"

func buildFixture() {
	command := exec.Command(goExecutable, "build", "-trimpath", "-o", output, "./testkit/processfixture")
	_ = command
}
`
	}
	if rel == "internal/world/tools.go" {
		return `package world

import "os"

const resolveToolsThroughAmbientPATH = false

func resolveTool() {
	candidate := "fixture"
	if resolveToolsThroughAmbientPATH && candidate != "" {
		_ = os.Getenv("PATH")
	}
}
`
	}
	packageDeclaration := packageName
	if packageName == "gitobj" && externalGitobjTests[filename] {
		packageDeclaration = "gitobj_test"
	}
	return "package " + packageDeclaration + "\n"
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

func runCase(templateRoot string, mutate func(string) error, want errorCode) error {
	root, err := os.MkdirTemp("", "countershape-u2-boundary-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	if err := fixture(root, templateRoot); err != nil {
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

func selfTest(templateRoot string) error {
	cases := []struct {
		name   string
		mutate func(string) error
		want   errorCode
	}{
		{name: "clean"},
		{name: "forbidden semantic import", want: codeAuthorityImport, mutate: func(root string) error {
			return os.WriteFile(filepath.Join(root, "internal", "gitobj", "errors.go"), []byte(
				"package gitobj\nimport \""+expectedModule+"/internal/observe\"\n"), 0o600)
		}},
		{name: "network import", want: codeImportNotAllowed, mutate: func(root string) error {
			return os.WriteFile(filepath.Join(root, "internal", "world", "errors.go"), []byte("package world\nimport \"net/http\"\n"), 0o600)
		}},
		{name: "runtime import in production", want: codeImportNotAllowed, mutate: func(root string) error {
			return os.WriteFile(filepath.Join(root, "internal", "world", "errors.go"), []byte("package world\nimport \"runtime\"\n"), 0o600)
		}},
		{name: "unreviewed Git subcommand", want: codeGitSurface, mutate: replaceFixture(
			"internal/gitobj/git.go", `runner.run(ctx, false, 1, "version")`, `runner.run(ctx, false, 1, "fetch")`,
		)},
		{name: "pre-plan digest pollution", want: codeDigestBoundary, mutate: replaceFixture(
			"internal/gitobj/types.go", "return s.selected.digest", "return s.policy.digest",
		)},
		{name: "pre-plan member pollution", want: codeDigestBoundary, mutate: replaceFixture(
			"internal/gitobj/types.go", "candidate.identity.String()", "foreign(candidate)",
		)},
		{name: "selected pinned identity assignment", want: codeDigestBoundary, mutate: replaceFixture(
			"internal/gitobj/types.go", "copyPinned := append([]PinnedTree(nil), pinned...)",
			"copyPinned := append([]PinnedTree(nil), pinned...)\n\tcopyPinned[0].identity = \"corrupt\"",
		)},
		{name: "ambient environment outside seam", want: codeEnvironmentSurface, mutate: replaceFixture(
			"internal/world/allocate.go", "if inheritAmbientEnvironment {", "if true {",
		)},
		{name: "ambient PATH outside conjoined seam", want: codeEnvironmentSurface, mutate: replaceFixture(
			"internal/world/tools.go", "if resolveToolsThroughAmbientPATH &&", "if true &&",
		)},
		{name: "ambient PATH function alias", want: codeEnvironmentSurface, mutate: appendFixture(
			"internal/world/tools.go", "\nfunc hostileEnvironmentAlias() { if resolveToolsThroughAmbientPATH { getenv := os.Getenv; _ = getenv(\"PATH\") } }\n",
		)},
		{name: "shell path outside seam", want: codeExecutionSurface, mutate: func(root string) error {
			path := filepath.Join(root, "internal", "world", "errors.go")
			return os.WriteFile(path, []byte("package world\nvar hostileShell = \"/bin/sh\"\n"), 0o600)
		}},
		{name: "arbitrary process constructor", want: codeExecutionSurface, mutate: appendFixture(
			"internal/world/process_darwin.go", "\nfunc hostileProcess() { _ = exec.Command(\"/bin/echo\") }\n",
		)},
		{name: "process constructor function alias", want: codeExecutionSurface, mutate: appendFixture(
			"internal/world/process_darwin_test.go", "\nfunc hostileExecAlias() { spawn := exec.Command; _ = spawn(\"/bin/echo\") }\n",
		)},
		{name: "os process authority function alias", want: codeExecutionSurface, mutate: func(root string) error {
			path := filepath.Join(root, "internal", "world", "errors.go")
			return os.WriteFile(path, []byte("package world\nimport \"os\"\nvar hostileFindProcess = os.FindProcess\n"), 0o600)
		}},
		{name: "Git runner function alias", want: codeGitSurface, mutate: appendFixture(
			"internal/gitobj/git.go", "\nfunc hostileGitAlias(ctx context.Context, runner closedGit) { escaped := runner.run; escaped(ctx, true, 1, \"fetch\") }\n",
		)},
		{name: "Git command function alias", want: codeGitSurface, mutate: appendFixture(
			"internal/gitobj/git.go", "\nfunc hostileGitCommandAlias(ctx context.Context, state *repositoryState) { escaped := state.runner.command; _ = escaped(ctx, true, \"fetch\") }\n",
		)},
		{name: "Git decoy receiver preserves vocabulary", want: codeGitSurface, mutate: replaceFixture(
			"internal/gitobj/git.go",
			`runner.run(ctx, false, 1, "version")`,
			`decoy := struct { run func(context.Context, bool, int, ...string) }{run: func(context.Context, bool, int, ...string) {}}
	decoy.run(ctx, false, 1, "version")`,
		)},
		{name: "syscall network escape", want: codeExecutionSurface, mutate: func(root string) error {
			path := filepath.Join(root, "internal", "world", "process_darwin.go")
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			changed := strings.Replace(string(raw), `import "os/exec"`, "import (\n\t\"os/exec\"\n\t\"syscall\"\n)", 1)
			changed += "\nvar hostileSocket = syscall.Socket\n"
			return os.WriteFile(path, []byte(changed), 0o600)
		}},
		{name: "cgo preamble system escape", want: codeCgoBoundary, mutate: replaceFixture(
			"internal/gitobj/publish_darwin.go", "if (exclusive) flags |= RENAME_EXCL;",
			"if (exclusive) flags |= RENAME_EXCL;\n\t(void)system(\"/bin/false\");",
		)},
		{name: "production materializer substitution", want: codeMaterializerBoundary, mutate: replaceFixture(
			"internal/world/api.go", "gitobj.DefaultMaterializer{}", "callerMaterializer{}",
		)},
		{name: "alternate exported caller materializer", want: codeMaterializerBoundary, mutate: appendFixture(
			"internal/world/api.go", "\nfunc ExecuteCallerMaterializer(ctx context.Context, request Request, source materializer) (Result, error) { _, err := source.Materialize(ctx, request.Candidate, \"elsewhere\"); return Result{}, err }\n",
		)},
		{name: "Git testkit helper authority mutation", want: codeTestkitBoundary, mutate: appendFixture(
			"testkit/gitrepo/gitrepo.go", "\nfunc hostileHelperAlias(ctx context.Context) { spawn := exec.CommandContext; _ = spawn(ctx, \"/usr/bin/false\") }\n",
		)},
		{name: "process testkit helper authority mutation", want: codeTestkitBoundary, mutate: appendFixture(
			"testkit/processfixture/main.go", "\nvar hostileHelperProcess = os.FindProcess\n",
		)},
		{name: "ineffective moved build constraint", want: codeBuildConstraint, mutate: replaceFixture(
			"internal/world/process_unsupported.go", "//go:build !darwin\n\npackage world\n", "package world\n\n//go:build !darwin\n",
		)},
		{name: "missing mutation anchor", want: codeAnchorContract, mutate: replaceFixture(
			"internal/gitobj/git.go", "MUTANT_U2_ENABLE_REPLACEMENT_REFS", "MUTANT_REMOVED",
		)},
		{name: "duplicate mutation anchor", want: codeAnchorContract, mutate: func(root string) error {
			path := filepath.Join(root, "internal", "gitobj", "git.go")
			handle, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
			if err != nil {
				return err
			}
			_, writeErr := handle.WriteString("\n// MUTANT_U2_ENABLE_REPLACEMENT_REFS\n")
			return errors.Join(writeErr, handle.Close())
		}},
		{name: "symlink source", want: codeTopology, mutate: func(root string) error {
			path := filepath.Join(root, "internal", "world", "errors.go")
			if err := os.Remove(path); err != nil {
				return err
			}
			return os.Symlink(filepath.Join(root, "go.mod"), path)
		}},
		{name: "unlisted source", want: codeTopology, mutate: func(root string) error {
			return os.WriteFile(filepath.Join(root, "internal", "gitobj", "unlisted.go"), []byte("package gitobj\n"), 0o600)
		}},
	}
	for _, test := range cases {
		if err := runCase(templateRoot, test.mutate, test.want); err != nil {
			return fmt.Errorf("self-test %q: %w", test.name, err)
		}
	}
	return nil
}

func replaceFixture(rel, before, after string) func(string) error {
	return func(root string) error {
		path := filepath.Join(root, filepath.FromSlash(rel))
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Count(string(raw), before) != 1 {
			return fmt.Errorf("fixture replacement %q is not unique in %s", before, rel)
		}
		return os.WriteFile(path, []byte(strings.Replace(string(raw), before, after, 1)), 0o600)
	}
}

func appendFixture(rel, suffix string) func(string) error {
	return func(root string) error {
		path := filepath.Join(root, filepath.FromSlash(rel))
		handle, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		_, writeErr := handle.WriteString(suffix)
		return errors.Join(writeErr, handle.Close())
	}
}

func main() {
	root := flag.String("root", ".", "repository root supplied only by the fixed public wrapper")
	runSelfTest := flag.Bool("self-test", false, "exercise hostile fail-closed fixtures")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "U2_BOUNDARY_ARGUMENTS: unexpected positional arguments")
		os.Exit(2)
	}
	if *runSelfTest {
		if err := selfTest(*root); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("U2 boundary analyzer self-test passed")
		return
	}
	checked, err := inspect(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("U2 boundary clean: %d packages, %d Go files, %d exact mutation anchors\n", checked.Packages, checked.GoFiles, checked.Anchors)
}

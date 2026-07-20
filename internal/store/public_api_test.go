package store_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	pathpkg "path"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/store"
)

var c2ProductionSurfaceFiles = [...]string{
	"execution_interlock.go", "nonhead_contract.go", "object_store.go", "private_contract_run.go",
}

func c2ReceiverName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return c2ReceiverName(value.X)
	case *ast.IndexExpr:
		return c2ReceiverName(value.X)
	case *ast.IndexListExpr:
		return c2ReceiverName(value.X)
	case *ast.SelectorExpr:
		return value.Sel.Name
	case *ast.ParenExpr:
		return c2ReceiverName(value.X)
	default:
		return ""
	}
}

func c2FileExportedSurface(t *testing.T, parsed *ast.File) []string {
	t.Helper()
	declarations := make([]string, 0)
	for _, declaration := range parsed.Decls {
		switch value := declaration.(type) {
		case *ast.FuncDecl:
			if !ast.IsExported(value.Name.Name) {
				continue
			}
			label := value.Name.Name
			if value.Recv != nil && len(value.Recv.List) == 1 {
				receiver := c2ReceiverName(value.Recv.List[0].Type)
				if receiver == "" {
					t.Fatalf("exported method %s has an unrecognized receiver", value.Name.Name)
				}
				if !ast.IsExported(receiver) {
					continue
				}
				label = receiver + "." + label
			}
			declarations = append(declarations, label)
		case *ast.GenDecl:
			for _, specification := range value.Specs {
				switch item := specification.(type) {
				case *ast.TypeSpec:
					if ast.IsExported(item.Name.Name) {
						declarations = append(declarations, item.Name.Name)
					}
					structure, ok := item.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, field := range structure.Fields.List {
						if len(field.Names) == 0 {
							name := c2ReceiverName(field.Type)
							if name == "" {
								t.Fatalf("struct %s has an unrecognized embedded field", item.Name.Name)
							}
							if ast.IsExported(name) {
								declarations = append(declarations, item.Name.Name+"."+name)
							}
							continue
						}
						for _, name := range field.Names {
							if ast.IsExported(name.Name) {
								declarations = append(declarations, item.Name.Name+"."+name.Name)
							}
						}
					}
				case *ast.ValueSpec:
					for _, identifier := range item.Names {
						if ast.IsExported(identifier.Name) {
							declarations = append(declarations, identifier.Name)
						}
					}
				}
			}
		}
	}
	return declarations
}

func c2ForbiddenProcessSurface(t *testing.T, parsed *ast.File) []string {
	t.Helper()
	forbidden := make([]string, 0)
	osAlias := ""
	for _, specification := range parsed.Imports {
		importPath, err := strconv.Unquote(specification.Path.Value)
		if err != nil {
			t.Fatalf("production import path did not unquote: %v", err)
		}
		if importPath == "os/exec" {
			forbidden = append(forbidden, "import:"+importPath)
		}
		if importPath != "os" {
			continue
		}
		alias := pathpkg.Base(importPath)
		if specification.Name != nil {
			alias = specification.Name.Name
			forbidden = append(forbidden, "aliased-import:os:"+alias)
		}
		osAlias = alias
	}
	if osAlias != "" {
		ast.Inspect(parsed, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			identifier, ok := selector.X.(*ast.Ident)
			if ok && identifier.Name == osAlias && (selector.Sel.Name == "StartProcess" || selector.Sel.Name == "FindProcess") {
				forbidden = append(forbidden, "os."+selector.Sel.Name)
			}
			return true
		})
	}
	return forbidden
}

func c2ExportedProductionSurface(t *testing.T) []string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("public-surface source location is unavailable")
	}
	directory := filepath.Dir(source)
	files := token.NewFileSet()
	declarations := make([]string, 0)
	for _, name := range c2ProductionSurfaceFiles {
		path := filepath.Join(directory, name)
		if info, err := os.Lstat(path); err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("public-surface source %s is not an exact regular file: %v", name, err)
		}
		parsed, err := parser.ParseFile(files, path, nil, parser.SkipObjectResolution)
		if err != nil || parsed.Name.Name != "store" {
			t.Fatalf("public-surface source %s did not parse as package store: %v", name, err)
		}
		declarations = append(declarations, c2FileExportedSurface(t, parsed)...)
		if forbidden := c2ForbiddenProcessSurface(t, parsed); len(forbidden) != 0 {
			t.Fatalf("production source %s exposes process capability: %v", name, forbidden)
		}
	}
	sort.Strings(declarations)
	return declarations
}

func objectStorePublicMethods() []string {
	typeOfStore := reflect.TypeOf((*store.ObjectStore)(nil))
	methods := make([]string, typeOfStore.NumMethod())
	for index := range methods {
		methods[index] = typeOfStore.Method(index).Name
	}
	sort.Strings(methods)
	return methods
}

func TestC2StoreExportsNoOfficialIssuerOrRunPermit(t *testing.T) {
	wantPackageSurface := []string{
		"ConformanceAttemptInput", "ConformanceAttemptInput.ContractBundleDigest",
		"ConformanceAttemptInput.MaterializationPolicyDigest", "ConformanceAttemptInput.ResidueHeadDigest",
		"ConformanceAttemptInput.TreeIdentityDigest", "ConformanceAttemptRecord",
		"ConformanceAttemptRecord.ContractBundleDigest", "ConformanceAttemptRecord.Digest",
		"ConformanceAttemptRecord.InstanceNonce", "ConformanceAttemptRecord.MaterializationPolicyDigest",
		"ConformanceAttemptRecord.ResidueHeadDigest", "ConformanceAttemptRecord.Roots",
		"ConformanceAttemptRecord.TreeIdentityDigest", "ConformanceAttemptRecord.Valid",
		"ConformanceAttemptRoots", "ConformanceAttemptRoots.AttemptRoot", "ConformanceAttemptRoots.CandidateParent",
		"ConformanceAttemptRoots.EvidenceRoot", "ConformanceAttemptRoots.FixtureRoot", "ConformanceAttemptRoots.HomeRoot",
		"ConformanceAttemptRoots.MarkerPath", "ConformanceAttemptRoots.StateRoot", "ConformanceAttemptRoots.TemporaryRoot",
		"ConformanceAttemptRoots.XDGCacheRoot", "ConformanceAttemptRoots.XDGConfigRoot",
		"ConformanceAttemptRoots.XDGDataRoot", "ConformanceAttemptRoots.XDGStateRoot",
		"ContractTargetRecord", "ContractTargetRecord.AttemptDigest", "ContractTargetRecord.Digest", "ContractTargetRecord.Valid",
		"Error", "Error.Cause", "Error.Code", "Error.Detail", "Error.Error", "Error.Unwrap",
		"NewSemanticObject", "ObjectAuthority", "ObjectStore",
		"ObjectStore.AllocateConformanceAttempt", "ObjectStore.Open", "ObjectStore.OpenConformanceAttempt",
		"ObjectStore.OpenContractTargetRecord", "ObjectStore.PersistContractTargetRecord",
		"ObjectStore.Publish", "ObjectStore.Read", "ObjectStore.Validate",
		"ObjectStore.ValidateExternalPublicationPath", "OpenObjectStore", "SemanticObject",
		"SemanticObject.CanonicalBytes", "SemanticObject.Digest", "SemanticObject.Kind", "SemanticObject.Valid",
	}
	if actual := c2ExportedProductionSurface(t); !reflect.DeepEqual(actual, wantPackageSurface) {
		t.Fatalf("C2 changed the compiler-parsed package surface: got %v, want %v", actual, wantPackageSurface)
	}
	t.Run("grouped exported struct field is visible", func(t *testing.T) {
		parsed, err := parser.ParseFile(token.NewFileSet(), "grouped.go", `package store
type ObjectStore struct { contractOps, ContractOps string }
`, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		actual := c2FileExportedSurface(t, parsed)
		if !reflect.DeepEqual(actual, []string{"ObjectStore", "ObjectStore.ContractOps"}) {
			t.Fatalf("grouped exported field was not compiler-enumerated: %v", actual)
		}
	})
	t.Run("aliased process import is visible", func(t *testing.T) {
		for _, source := range []string{
			`package store
import launcher "os/exec"
var _ = launcher.ErrNotFound
	`,
			`package store
import ("os"; launcher "os/exec")
var _, _ = os.ErrNotExist, launcher.ErrNotFound
	`,
		} {
			parsed, err := parser.ParseFile(token.NewFileSet(), "imports.go", source, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			if actual := c2ForbiddenProcessSurface(t, parsed); !reflect.DeepEqual(actual, []string{"import:os/exec"}) {
				t.Fatalf("aliased process import escaped the compiler guard: %v", actual)
			}
		}
	})
	want := []string{
		"AdvanceBaseline", "AdvanceChoicepoint", "AdvanceConfirmation", "AdvanceDivergence",
		"AdvanceReduction", "AdvanceResidue", "AdvanceRuling", "AllocateConformanceAttempt",
		"ConfirmResiduePublication", "CreateStudy", "Open", "OpenConformanceAttempt",
		"OpenContractTargetRecord", "OpenHead", "PersistContractTargetRecord", "Publish", "Read", "Validate",
		"ValidateExternalPublicationPath",
	}
	if actual := objectStorePublicMethods(); !reflect.DeepEqual(actual, want) {
		t.Fatalf("C2 changed the exported ObjectStore surface: got %v, want %v", actual, want)
	}
}

func TestC3StoreBridgeExportsOnlyInertAttemptAndTargetRecords(t *testing.T) {
	for _, value := range []any{
		store.ConformanceAttemptRecord{}, store.ConformanceAttemptRoots{}, store.ContractTargetRecord{},
	} {
		typeOfValue := reflect.TypeOf(value)
		for index := 0; index < typeOfValue.NumField(); index++ {
			if typeOfValue.Field(index).IsExported() {
				t.Fatalf("%s exposes authority-bearing field %s", typeOfValue, typeOfValue.Field(index).Name)
			}
		}
	}
	for _, forbidden := range []string{"official", "permit", "process", "spawn", "interlock", "latest", "list", "status"} {
		for _, method := range objectStorePublicMethods() {
			if strings.Contains(strings.ToLower(method), forbidden) {
				t.Fatalf("store bridge exposes forbidden %s method %q", forbidden, method)
			}
		}
	}
}

func TestC2StoreExportsNoListLatestTraversalStatusOrHeadMutationSurface(t *testing.T) {
	for _, method := range objectStorePublicMethods() {
		lower := strings.ToLower(method)
		for _, forbidden := range []string{"list", "latest", "travers", "status", "advancehead", "replacehead", "forcehead"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("C2 exported forbidden discovery or generic-head method %q", method)
			}
		}
	}
}

func TestRawHeadAdvanceCannotMintFreshConfirmationAuthority(t *testing.T) {
	objectStore := reflect.TypeOf((*store.ObjectStore)(nil))
	if _, exposed := objectStore.MethodByName("AdvanceHead"); exposed {
		t.Fatal("generic raw head advancement is exported")
	}
	method, present := objectStore.MethodByName("AdvanceConfirmation")
	if !present || method.Type.NumIn() != 4 ||
		method.Type.In(3).PkgPath() != "github.com/nelsonwerd/countershape/internal/confirmation/internal/publication" {
		t.Fatalf("confirmation transition does not require its opaque internal publication authority: %v", method.Type)
	}
}

func TestRawHeadAdvanceCannotPublishRefineRuling(t *testing.T) {
	objectStore := reflect.TypeOf((*store.ObjectStore)(nil))
	if _, exposed := objectStore.MethodByName("AdvanceHead"); exposed {
		t.Fatal("generic raw head advancement is exported")
	}
	method, present := objectStore.MethodByName("AdvanceRuling")
	if !present || method.Type.NumIn() != 4 ||
		method.Type.In(3).PkgPath() != "github.com/nelsonwerd/countershape/internal/choice/promotion/internal/publication" {
		t.Fatalf("ruling transition does not require its promotion-issued authority: %v", method.Type)
	}
}

func TestStudyHeadExportedTransitionsAreTypedAndClosed(t *testing.T) {
	objectStore := reflect.TypeOf((*store.ObjectStore)(nil))
	expected := map[string]string{
		"CreateStudy":         "github.com/nelsonwerd/countershape/internal/domain",
		"AdvanceBaseline":     "github.com/nelsonwerd/countershape/internal/compare",
		"AdvanceDivergence":   "github.com/nelsonwerd/countershape/internal/compare",
		"AdvanceReduction":    "github.com/nelsonwerd/countershape/internal/reduce",
		"AdvanceConfirmation": "github.com/nelsonwerd/countershape/internal/confirmation/internal/publication",
		"AdvanceChoicepoint":  "github.com/nelsonwerd/countershape/internal/choice/promotion/internal/publication",
		"AdvanceRuling":       "github.com/nelsonwerd/countershape/internal/choice/promotion/internal/publication",
		"AdvanceResidue":      "github.com/nelsonwerd/countershape/internal/emit/node/internal/publication",
	}
	for name, authorityPackage := range expected {
		method, present := objectStore.MethodByName(name)
		if !present || method.Type.NumIn() != 4 || method.Type.In(3).PkgPath() != authorityPackage {
			t.Fatalf("%s input authority = %v, want package %s", name, method.Type, authorityPackage)
		}
	}
	for _, forbidden := range []string{"AdvanceHead", "ReplaceHead", "ForceHead"} {
		if _, exposed := objectStore.MethodByName(forbidden); exposed {
			t.Fatalf("unsafe head mutation surface %s is exported", forbidden)
		}
	}
}

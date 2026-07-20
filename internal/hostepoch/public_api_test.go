package hostepoch_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func receiverName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return receiverName(value.X)
	default:
		return ""
	}
}

func hostEpochExportedSurface(t *testing.T, parsed *ast.File) []string {
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
				receiver := receiverName(value.Recv.List[0].Type)
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
						for _, fieldName := range field.Names {
							if ast.IsExported(fieldName.Name) {
								surface = append(surface, item.Name.Name+"."+fieldName.Name)
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

func TestC3HostEpochPublicSurfaceIsClosed(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("source location unavailable")
	}
	directory := filepath.Dir(file)
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	var production []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			production = append(production, entry.Name())
		}
	}
	sort.Strings(production)
	wantProduction := []string{"epoch.go", "source_darwin_cgo.go", "source_unsupported.go"}
	if !reflect.DeepEqual(production, wantProduction) {
		t.Fatalf("hostepoch production topology changed: got %v want %v", production, wantProduction)
	}
	var got []string
	for _, name := range production {
		path := filepath.Join(directory, name)
		info, statErr := os.Lstat(path)
		if statErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("hostepoch production source %s is not exact: %v", name, statErr)
		}
		parsed, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if parseErr != nil || parsed.Name.Name != "hostepoch" {
			t.Fatalf("hostepoch production source %s did not parse exactly: %v", name, parseErr)
		}
		got = append(got, hostEpochExportedSurface(t, parsed)...)
	}
	sort.Strings(got)
	want := []string{"CodeInvalidMeasurement", "CodeUnstableMeasurement", "CodeUnsupportedPlatform", "Epoch", "Epoch.Digest", "Epoch.Revalidate", "Epoch.Valid", "Error", "Error.Cause", "Error.Code", "Error.Detail", "Error.Error", "Error.Unwrap", "IsCode", "Measure"}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("hostepoch surface changed: got %v want %v", got, want)
	}
}

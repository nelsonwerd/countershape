package noderuntime_test

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func c3RuntimeReceiver(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return c3RuntimeReceiver(value.X)
	default:
		return ""
	}
}

func c3RuntimeExportedSurface(t *testing.T, parsed *ast.File) []string {
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
				receiver := c3RuntimeReceiver(value.Recv.List[0].Type)
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

func TestC3NodeRuntimePublicSurfaceAndSoleSpawnEdge(t *testing.T) {
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
	wantProduction := []string{"identity_darwin.go", "probe_darwin.go", "runtime.go", "unsupported.go"}
	if !reflect.DeepEqual(production, wantProduction) {
		t.Fatalf("noderuntime production topology changed: got %v want %v", production, wantProduction)
	}
	var got []string
	spawnImports := 0
	for _, name := range production {
		path := filepath.Join(directory, name)
		info, statErr := os.Lstat(path)
		if statErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("noderuntime production source %s is not exact: %v", name, statErr)
		}
		parsed, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if parseErr != nil || parsed.Name.Name != "noderuntime" {
			t.Fatalf("noderuntime production source %s did not parse exactly: %v", name, parseErr)
		}
		for _, specification := range parsed.Imports {
			importPath, unquoteErr := strconv.Unquote(specification.Path.Value)
			if unquoteErr != nil {
				t.Fatal(unquoteErr)
			}
			if importPath == "os/exec" {
				spawnImports++
				if name != "probe_darwin.go" {
					t.Fatalf("os/exec escaped probe owner into %s", name)
				}
			}
		}
		got = append(got, c3RuntimeExportedSurface(t, parsed)...)
	}
	if spawnImports != 1 {
		t.Fatalf("production os/exec import count=%d want 1", spawnImports)
	}
	sort.Strings(got)
	want := []string{"Admit", "CodeInvalidRuntime", "CodeProbeFailed", "CodeRuntimeChanged", "CodeUnsupportedPlatform", "Error", "Error.Cause", "Error.Code", "Error.Detail", "Error.Error", "Error.Unwrap", "IsCode", "Runtime", "Runtime.Architecture", "Runtime.ExecutableByteCount", "Runtime.ExecutableBytesDigest", "Runtime.ExecutableMode", "Runtime.Major", "Runtime.Path", "Runtime.Platform", "Runtime.ProbeProgramDigest", "Runtime.Revalidate", "Runtime.Valid", "Runtime.Version"}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("noderuntime surface changed: got %v want %v", got, want)
	}
	for _, name := range production {
		body, readErr := os.ReadFile(filepath.Join(directory, name))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if name != "probe_darwin.go" && (bytes.Contains(body, []byte("exec.Command")) || bytes.Contains(body, []byte("LookPath"))) {
			t.Fatalf("process lookup/spawn text escaped into %s", name)
		}
	}
}

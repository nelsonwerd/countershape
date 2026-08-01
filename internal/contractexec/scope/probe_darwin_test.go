//go:build darwin

package scope

import (
	"bytes"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/testkit/httpfixture"
)

func probeDiagnosticErrorChain(err error) string {
	if err == nil {
		return "<nil>"
	}
	parts := make([]string, 0, 8)
	var visit func(error, int)
	visit = func(current error, depth int) {
		if current == nil || depth >= 16 || len(parts) >= 64 {
			return
		}
		parts = append(parts, current.Error())
		switch wrapped := current.(type) {
		case interface{ Unwrap() []error }:
			for _, child := range wrapped.Unwrap() {
				visit(child, depth+1)
			}
		case interface{ Unwrap() error }:
			visit(wrapped.Unwrap(), depth+1)
		}
	}
	visit(err, 0)
	return strings.Join(parts, " <- ")
}

func TestC5InventoryBindsReferenceFixtureAndRejectsSymlink(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	materializeC5ReferenceInventory(t, root)
	inventory, err := Snapshot(root)
	if err != nil || !inventory.Valid() || !inventory.ReferenceHTTPFixture() ||
		inventory.EntryCount() != 3 {
		t.Fatalf("reference inventory valid=%t enrolled=%t entries=%d err=%v",
			inventory.Valid(), inventory.ReferenceHTTPFixture(), inventory.EntryCount(), err)
	}
	if err := os.Symlink("candidate-role.json", filepath.Join(root, "role-link")); err != nil {
		t.Fatal(err)
	}
	if hostile, snapshotErr := Snapshot(root); snapshotErr == nil || hostile.Valid() {
		t.Fatalf("symlink-bearing inventory was admitted: valid=%t err=%v", hostile.Valid(), snapshotErr)
	} else if diagnostic, ok := DiagnosticOf(snapshotErr); !ok ||
		(diagnostic.Code() != CodeInventoryChanged && diagnostic.Code() != CodeInventorySpecial) {
		t.Fatalf("symlink refusal lost its bounded diagnostic: %v", snapshotErr)
	}

	specialModes := []struct {
		name string
		mode os.FileMode
	}{
		{name: "setuid", mode: os.ModeSetuid},
		{name: "setgid", mode: os.ModeSetgid},
		{name: "sticky", mode: os.ModeSticky},
	}
	locations := []struct {
		name     string
		relative string
		code     string
	}{
		{name: "root", code: CodeInventoryIdentity},
		{name: "nested-directory", relative: "fixture", code: CodeInventorySpecial},
		{
			name:     "nested-regular-file",
			relative: "fixture/server_child_bind.mjs",
			code:     CodeInventorySpecial,
		},
	}
	for _, special := range specialModes {
		for _, location := range locations {
			t.Run(special.name+"/"+location.name, func(t *testing.T) {
				hostileRoot, evalErr := filepath.EvalSymlinks(t.TempDir())
				if evalErr != nil {
					t.Fatal(evalErr)
				}
				materializeC5ReferenceInventory(t, hostileRoot)
				hostilePath := hostileRoot
				if location.relative != "" {
					hostilePath = filepath.Join(hostileRoot, filepath.FromSlash(location.relative))
				}
				before, statErr := os.Lstat(hostilePath)
				if statErr != nil {
					t.Fatal(statErr)
				}
				if chmodErr := os.Chmod(hostilePath, before.Mode().Perm()|special.mode); chmodErr != nil {
					t.Fatal(chmodErr)
				}
				after, statErr := os.Lstat(hostilePath)
				if statErr != nil {
					t.Fatalf(
						"host filesystem did not retain %s on %s: err=%v",
						special.name, location.name, statErr,
					)
				}
				if after.Mode()&special.mode == 0 {
					t.Fatalf(
						"host filesystem did not retain %s on %s: mode=%v",
						special.name, location.name, after.Mode(),
					)
				}
				hostile, snapshotErr := Snapshot(hostileRoot)
				diagnostic, diagnosed := DiagnosticOf(snapshotErr)
				if snapshotErr == nil || hostile.Valid() || !diagnosed ||
					diagnostic.Code() != location.code {
					t.Fatalf(
						"%s on %s was not refused exactly: valid=%t code=%s want=%s err=%v",
						special.name, location.name, hostile.Valid(), diagnostic.Code(),
						location.code, snapshotErr,
					)
				}
			})
		}
	}
}

func materializeC5ReferenceInventory(t testing.TB, root string) {
	t.Helper()
	files, err := httpfixture.PortableCandidateFiles(httpfixture.Forbidden)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		path := filepath.Join(root, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		if file.Mode == "100755" {
			mode = 0o755
		}
		if err := os.WriteFile(path, file.Content, mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() ||
			info.Mode().Perm() != mode {
			t.Fatalf(
				"reference inventory file mode=%v want=%v err=%v",
				info,
				mode,
				err,
			)
		}
	}
}

func TestC5ProbeMeasuresImportAndServiceCanariesAndCleansIdentity(t *testing.T) {
	privateRoot, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	probe, err := NewProbe(privateRoot)
	if err != nil {
		t.Fatal(err)
	}
	environment := probe.Environment()
	moduleRoot := filepath.Dir(environment[ImportModuleEnvironment])
	socketAlias := filepath.Dir(environment[ImportSocketEnvironment])
	socketRoot := filepath.Dir(socketAlias)
	if filepath.Dir(moduleRoot) != privateRoot {
		t.Fatalf("probe payload root %q escaped attempt-private root %q", moduleRoot, privateRoot)
	}
	resolvedAlias, err := filepath.EvalSymlinks(socketAlias)
	if err != nil || resolvedAlias != moduleRoot {
		t.Fatalf("short socket alias resolved to %q, want %q: %v", resolvedAlias, moduleRoot, err)
	}
	for _, name := range []string{ImportSocketEnvironment, ServiceSocketEnvironment} {
		if len([]byte(environment[name])) > maxDarwinUnixSocketPathBytes {
			t.Fatalf("%s path exceeds Darwin sockaddr_un capacity: %q", name, environment[name])
		}
	}
	for _, name := range []string{
		ImportModuleEnvironment,
		ImportSocketEnvironment,
		ServiceSocketEnvironment,
	} {
		if environment[name] == "" {
			t.Fatalf("probe environment omitted %s", name)
		}
	}
	for _, name := range []string{ImportSocketEnvironment, ServiceSocketEnvironment} {
		connection, dialErr := net.Dial("unix", environment[name])
		if dialErr != nil {
			t.Fatalf("connect %s: %v", name, dialErr)
		}
		if closeErr := connection.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	measurements, err := probe.Finish(nil)
	if err != nil || measurements.Ambiguous || measurements.ImportAccepts != 1 ||
		measurements.ServiceAccepts != 1 || len(measurements.ModuleSHA256) != 64 {
		t.Fatalf("probe measurements = %#v, err=%v", measurements, err)
	}
	again, err := probe.Finish(nil)
	if err != nil || again != measurements {
		t.Fatalf("probe finish was not convergent: first=%#v second=%#v err=%v",
			measurements, again, err)
	}
	for _, path := range environment {
		if _, statErr := os.Lstat(path); !os.IsNotExist(statErr) {
			t.Fatalf("probe residue survived at %s: %v", path, statErr)
		}
	}
	for _, path := range []string{moduleRoot, socketAlias, socketRoot} {
		if _, statErr := os.Lstat(path); !os.IsNotExist(statErr) {
			t.Fatalf("probe container survived at %s: %v", path, statErr)
		}
	}
	if info, statErr := os.Lstat(privateRoot); statErr != nil || !info.IsDir() {
		t.Fatalf("attempt-private parent did not survive probe cleanup: %v", statErr)
	}
}

func TestC5ProbeSameSizeModuleRewriteIsIntegrityAmbiguity(t *testing.T) {
	privateRoot := privateProbeRoot(t)
	probe, err := NewProbe(privateRoot)
	if err != nil {
		t.Fatalf("new probe: err=%v chain=%s", err, probeDiagnosticErrorChain(err))
	}
	environment := probe.Environment()
	module := environment[ImportModuleEnvironment]
	before, err := os.Lstat(module)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(module)
	if err != nil || len(body) == 0 {
		t.Fatalf("read retained module: bytes=%d err=%v", len(body), err)
	}
	rewritten := append([]byte(nil), body...)
	rewritten[0] ^= 0x01
	file, err := os.OpenFile(module, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		t.Fatal(err)
	}
	written, writeErr := file.Write(rewritten)
	syncErr := file.Sync()
	closeErr := file.Close()
	after, statErr := os.Lstat(module)
	if writeErr != nil || syncErr != nil || closeErr != nil || statErr != nil ||
		written != len(rewritten) || after.Size() != before.Size() ||
		!os.SameFile(before, after) {
		t.Fatalf(
			"same-identity rewrite did not close exactly: written=%d size=%d/%d err=%v",
			written, before.Size(), after.Size(),
			errors.Join(writeErr, syncErr, closeErr, statErr),
		)
	}
	measurements, finishErr := probe.Finish(nil)
	diagnostic, diagnosed := DiagnosticOf(finishErr)
	if finishErr == nil || !measurements.Ambiguous || !diagnosed ||
		diagnostic.Code() != CodeProbeIntegrity ||
		!bytes.Equal(rewritten, mustReadFile(t, module)) {
		t.Fatalf(
			"same-size rewrite did not fail closed: measurements=%#v diagnostic=%s err=%v",
			measurements, diagnostic.Code(), finishErr,
		)
	}
	socketRoot := filepath.Dir(filepath.Dir(environment[ImportSocketEnvironment]))
	if _, err := os.Lstat(socketRoot); !os.IsNotExist(err) {
		t.Fatalf("independently exact short alias shell survived ambiguity: %v", err)
	}
}

func TestC5ProbeForeignResidueIsBoundedAndPreserved(t *testing.T) {
	privateRoot := privateProbeRoot(t)
	probe, err := NewProbe(privateRoot)
	if err != nil {
		t.Fatalf("new probe: err=%v chain=%s", err, probeDiagnosticErrorChain(err))
	}
	environment := probe.Environment()
	probeRoot := filepath.Dir(environment[ImportModuleEnvironment])
	foreignRoot := filepath.Join(probeRoot, "foreign")
	if err := os.Mkdir(foreignRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(foreignRoot, strings.Repeat("x", 64))
	want := []byte("do-not-traverse-or-delete")
	if err := os.WriteFile(sentinel, want, 0o600); err != nil {
		t.Fatal(err)
	}
	measurements, finishErr := probe.Finish(nil)
	diagnostic, diagnosed := DiagnosticOf(finishErr)
	if finishErr == nil || !measurements.Ambiguous || !diagnosed ||
		diagnostic.Code() != CodeProbeIntegrity ||
		!bytes.Equal(mustReadFile(t, sentinel), want) {
		t.Fatalf(
			"foreign residue did not fail closed: measurements=%#v diagnostic=%s err=%v",
			measurements, diagnostic.Code(), finishErr,
		)
	}
	for _, path := range []string{
		environment[ImportModuleEnvironment],
		filepath.Join(probeRoot, filepath.Base(environment[ImportSocketEnvironment])),
		filepath.Join(probeRoot, filepath.Base(environment[ServiceSocketEnvironment])),
	} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("retained payload leaf was deleted after foreign roster: %s: %v", path, err)
		}
	}
}

func privateProbeRoot(t testing.TB) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func mustReadFile(t testing.TB, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

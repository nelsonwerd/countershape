//go:build darwin && arm64 && cgo

package noderuntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/nelsonwerd/countershape/internal/domain"
)

func c3RuntimeFixture(t testing.TB) (string, string, runtimeSnapshot, probeResult) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(root, "node")
	if err := os.WriteFile(executable, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	probeParent := filepath.Join(root, "probe")
	if err := os.Mkdir(probeParent, 0o700); err != nil {
		t.Fatal(err)
	}
	digest := domain.MustDigest("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	snapshot := runtimeSnapshot{
		identity: executableIdentity{device: 1, inode: 2, links: 1, uid: 3, gid: 4, mode: 0o100755, size: 7},
		path:     executable, bytesDigest: digest, mode: "100755", byteCount: 7,
	}
	probe := probeResult{
		path: executable, version: "v25.2.1", major: 25, platform: "darwin", architecture: "arm64",
		programDigest: nodeProbeDigest(),
	}
	return executable, probeParent, snapshot, probe
}

func c3CanonicalTempDir(t testing.TB) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestC3NodeRuntimeMeasureProbeMeasureAdmission(t *testing.T) {
	executable, parent, snapshot, probe := c3RuntimeFixture(t)
	measureCalls, probeCalls := 0, 0
	runtime, err := admitWithOperations(context.Background(), executable, parent, runtimeOperations{
		measure: func(path string) (runtimeSnapshot, error) { measureCalls++; return snapshot, nil },
		probe:   func(ctx context.Context, path, root string) (probeResult, error) { probeCalls++; return probe, nil },
	}, nil)
	if err != nil || !runtime.Valid() || measureCalls != 2 || probeCalls != 1 {
		t.Fatalf("admission failed: measures=%d probes=%d err=%v", measureCalls, probeCalls, err)
	}
	if runtime.Path() != executable || runtime.Version() != "v25.2.1" || runtime.Major() != 25 ||
		runtime.Platform() != "darwin" || runtime.Architecture() != "arm64" ||
		runtime.ExecutableBytesDigest() != snapshot.bytesDigest || runtime.ExecutableMode() != "100755" ||
		runtime.ExecutableByteCount() != 7 || runtime.ProbeProgramDigest() != nodeProbeDigest() {
		t.Fatal("admitted runtime getters differ from the exact tuple")
	}
}

func TestC3NodeRuntimeFaultMatrixRefusesAuthority(t *testing.T) {
	executable, parent, snapshot, probe := c3RuntimeFixture(t)
	tests := map[string]runtimeOperations{
		"first measure": {measure: func(string) (runtimeSnapshot, error) { return runtimeSnapshot{}, errors.New("first") }, probe: func(context.Context, string, string) (probeResult, error) { return probe, nil }},
		"probe":         {measure: func(string) (runtimeSnapshot, error) { return snapshot, nil }, probe: func(context.Context, string, string) (probeResult, error) { return probeResult{}, errors.New("probe") }},
		"second measure": {measure: func() func(string) (runtimeSnapshot, error) {
			calls := 0
			return func(string) (runtimeSnapshot, error) {
				calls++
				if calls == 2 {
					return runtimeSnapshot{}, errors.New("second")
				}
				return snapshot, nil
			}
		}(), probe: func(context.Context, string, string) (probeResult, error) { return probe, nil }},
		"identity change": {measure: func() func(string) (runtimeSnapshot, error) {
			calls := 0
			return func(string) (runtimeSnapshot, error) {
				calls++
				value := snapshot
				if calls == 2 {
					value.identity.inode++
				}
				return value, nil
			}
		}(), probe: func(context.Context, string, string) (probeResult, error) { return probe, nil }},
		"probe path": {measure: func(string) (runtimeSnapshot, error) { return snapshot, nil }, probe: func(context.Context, string, string) (probeResult, error) {
			value := probe
			value.path += ".other"
			return value, nil
		}},
		"version over bound": {measure: func(string) (runtimeSnapshot, error) { return snapshot, nil }, probe: func(context.Context, string, string) (probeResult, error) {
			value := probe
			value.version = "v1." + strings.Repeat("0", 126)
			value.major = 1
			return value, nil
		}},
	}
	for name, operations := range tests {
		t.Run(name, func(t *testing.T) {
			if value, err := admitWithOperations(context.Background(), executable, parent, operations, nil); err == nil || value.Valid() {
				t.Fatalf("fault issued runtime authority: %#v %v", value, err)
			}
		})
	}
}

func TestC3NodeRuntimeRevalidationIsFreshAndExact(t *testing.T) {
	executable, parent, snapshot, probe := c3RuntimeFixture(t)
	operations := runtimeOperations{
		measure: func(string) (runtimeSnapshot, error) { return snapshot, nil },
		probe:   func(context.Context, string, string) (probeResult, error) { return probe, nil },
	}
	retained, err := admitWithOperations(context.Background(), executable, parent, operations, nil)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := revalidateWithOperations(context.Background(), retained, operations)
	if err != nil || !fresh.Valid() || fresh.state == retained.state || fresh.state.gate != retained.state.gate {
		t.Fatalf("revalidation did not return fresh joined authority: %v", err)
	}
	alternateDigest := domain.MustDigest("sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	drifts := map[string]struct {
		snapshot func(*runtimeSnapshot)
		probe    func(*probeResult)
	}{
		"device":             {snapshot: func(value *runtimeSnapshot) { value.identity.device++ }},
		"inode":              {snapshot: func(value *runtimeSnapshot) { value.identity.inode++ }},
		"links":              {snapshot: func(value *runtimeSnapshot) { value.identity.links++ }},
		"owner":              {snapshot: func(value *runtimeSnapshot) { value.identity.uid++ }},
		"group":              {snapshot: func(value *runtimeSnapshot) { value.identity.gid++ }},
		"modification time":  {snapshot: func(value *runtimeSnapshot) { value.identity.mtimeNsec++ }},
		"change time":        {snapshot: func(value *runtimeSnapshot) { value.identity.ctimeNsec++ }},
		"birth time":         {snapshot: func(value *runtimeSnapshot) { value.identity.birthNsec++ }},
		"executable bytes":   {snapshot: func(value *runtimeSnapshot) { value.bytesDigest = alternateDigest }},
		"executable mode":    {snapshot: func(value *runtimeSnapshot) { value.identity.mode = 0o100555; value.mode = "100555" }},
		"executable count":   {snapshot: func(value *runtimeSnapshot) { value.identity.size++; value.byteCount++ }},
		"snapshot path":      {snapshot: func(value *runtimeSnapshot) { value.path += ".other" }},
		"probe version":      {probe: func(value *probeResult) { value.version = "v25.3.0" }},
		"probe major":        {probe: func(value *probeResult) { value.major++ }},
		"probe platform":     {probe: func(value *probeResult) { value.platform = "linux" }},
		"probe architecture": {probe: func(value *probeResult) { value.architecture = "x64" }},
		"probe path":         {probe: func(value *probeResult) { value.path += ".other" }},
		"probe program":      {probe: func(value *probeResult) { value.programDigest = alternateDigest }},
	}
	for name, mutate := range drifts {
		t.Run(name, func(t *testing.T) {
			changedSnapshot, changedProbe := snapshot, probe
			if mutate.snapshot != nil {
				mutate.snapshot(&changedSnapshot)
			}
			if mutate.probe != nil {
				mutate.probe(&changedProbe)
			}
			changed := runtimeOperations{
				measure: func(string) (runtimeSnapshot, error) { return changedSnapshot, nil },
				probe:   func(context.Context, string, string) (probeResult, error) { return changedProbe, nil },
			}
			if value, err := revalidateWithOperations(context.Background(), retained, changed); err == nil || value.Valid() {
				t.Fatalf("changed %s reopened authority: %v", name, err)
			}
		})
	}
	retiredParent := parent + ".retired"
	if err := os.Rename(parent, retiredParent); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	if value, err := revalidateWithOperations(context.Background(), retained, operations); !IsCode(err, CodeRuntimeChanged) || value.Valid() {
		t.Fatalf("replaced private probe parent reopened authority: %#v %v", value, err)
	}
}

func TestC3NodeRuntimeCopiedCapabilitiesSerializeRevalidation(t *testing.T) {
	executable, parent, snapshot, probe := c3RuntimeFixture(t)
	operations := runtimeOperations{
		measure: func(string) (runtimeSnapshot, error) { return snapshot, nil },
		probe:   func(context.Context, string, string) (probeResult, error) { return probe, nil },
	}
	retained, err := admitWithOperations(context.Background(), executable, parent, operations, nil)
	if err != nil {
		t.Fatal(err)
	}
	var mutex sync.Mutex
	active, peak := 0, 0
	var firstProbe sync.Once
	probeEntered := make(chan struct{})
	releaseProbe := make(chan struct{})
	serialized := runtimeOperations{
		measure: operations.measure,
		probe: func(context.Context, string, string) (probeResult, error) {
			mutex.Lock()
			active++
			current := active
			if active > peak {
				peak = active
			}
			mutex.Unlock()
			first := false
			firstProbe.Do(func() {
				first = true
				close(probeEntered)
			})
			if first {
				<-releaseProbe
			}
			mutex.Lock()
			active--
			mutex.Unlock()
			if current < 1 {
				return probeResult{}, errors.New("probe activity counter underflowed")
			}
			return probe, nil
		},
	}
	launch := make(chan struct{})
	var ready sync.WaitGroup
	var attempted atomic.Int32
	var wait sync.WaitGroup
	for range 16 {
		ready.Add(1)
		wait.Add(1)
		go func(copy Runtime) {
			defer wait.Done()
			ready.Done()
			<-launch
			attempted.Add(1)
			if _, err := revalidateWithOperations(context.Background(), copy, serialized); err != nil {
				t.Errorf("revalidation failed: %v", err)
			}
		}(retained)
	}
	ready.Wait()
	close(launch)
	<-probeEntered
	for attempted.Load() != 16 {
		runtime.Gosched()
	}
	close(releaseProbe)
	wait.Wait()
	if peak != 1 {
		t.Fatalf("copied capabilities revalidated concurrently: peak=%d", peak)
	}
}

func TestC3NodeRuntimeRejectsAmbientOrForgedInputs(t *testing.T) {
	executable, parent, snapshot, probe := c3RuntimeFixture(t)
	for _, path := range []string{"", "node", "/tmp/../tmp/node", "/tmp/node\x00tail", "/tmp/node\ntail", string([]byte{'/', 't', 'm', 'p', '/', 0xff})} {
		if _, err := Admit(context.Background(), path, parent); err == nil {
			t.Fatalf("invalid path %q was admitted", path)
		}
	}
	newlineTarget := filepath.Join(filepath.Dir(executable), "node\nresolved")
	if err := os.WriteFile(newlineTarget, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(filepath.Dir(executable), "clean-node-alias")
	if err := os.Symlink(filepath.Base(newlineTarget), alias); err != nil {
		t.Fatal(err)
	}
	if resolved, err := resolveExplicitExecutable(alias); err == nil || resolved != "" {
		t.Fatalf("clean alias resolved to control-character authority: %q %v", resolved, err)
	}
	closed, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Admit(closed, "/opt/homebrew/bin/node", parent); err == nil {
		t.Fatal("closed context was admitted")
	}
	if (Runtime{}).Valid() || (Runtime{state: &runtimeState{seal: admittedRuntimeSeal}}).Valid() {
		t.Fatal("zero or forged runtime was valid")
	}
	base := runtimeState{
		gate: &sync.Mutex{}, snapshot: snapshot, probe: probe, probeParent: parent,
		parentID: probeParentIdentity{device: 1, inode: 2, owner: 3, permissions: 0o700}, seal: admittedRuntimeSeal,
	}
	if !(&Runtime{state: &base}).Valid() {
		t.Fatal("complete closed runtime fixture was invalid")
	}
	alternateDigest := domain.MustDigest("sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	for name, mutate := range map[string]func(*runtimeState){
		"size count mismatch": func(value *runtimeState) { value.snapshot.identity.size++ },
		"mode text mismatch":  func(value *runtimeState) { value.snapshot.mode = "100555" },
		"non executable":      func(value *runtimeState) { value.snapshot.identity.mode = 0o100644; value.snapshot.mode = "100644" },
		"group writable":      func(value *runtimeState) { value.snapshot.identity.mode = 0o100775; value.snapshot.mode = "100775" },
		"special mode":        func(value *runtimeState) { value.snapshot.identity.mode = 0o104755; value.snapshot.mode = "100755" },
		"non regular":         func(value *runtimeState) { value.snapshot.identity.mode = 0o040755; value.snapshot.mode = "100755" },
		"arbitrary probe":     func(value *runtimeState) { value.probe.programDigest = alternateDigest },
	} {
		t.Run("forged "+name, func(t *testing.T) {
			forged := base
			forged.gate = &sync.Mutex{}
			mutate(&forged)
			if (&Runtime{state: &forged}).Valid() {
				t.Fatalf("forged %s retained authority", name)
			}
		})
	}
}

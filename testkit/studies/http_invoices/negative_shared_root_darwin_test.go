//go:build darwin

package http_invoices

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/testkit/httpfixture"
)

const negativeFixtureLabel = "NEGATIVE_FIXTURE_NON_PRODUCT"

type negativeFixtureResponse struct {
	status        int
	body          []byte
	projection    []byte
	requestID     string
	scratchRoot   string
	invocationRaw []byte
}

func TestNegativeSharedRootContaminationCreatesFalseEquality(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is not installed")
	}
	root := t.TempDir()
	sharedState := filepath.Join(root, "deliberately-shared-state")
	if err := os.Mkdir(sharedState, 0o700); err != nil {
		t.Fatal(err)
	}

	first := runNegativeFixture(t, node, root, sharedState, httpfixture.ConcealNotFound, 0)
	marker := filepath.Join(sharedState, httpfixture.ContaminationFilename)
	if info, err := os.Lstat(marker); err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		t.Fatalf("conceal candidate did not leave the deliberate contamination marker: %+v %v", info, err)
	}
	second := runNegativeFixture(t, node, root, sharedState, httpfixture.MetadataDisclosure, 1)

	if negativeFixtureLabel != "NEGATIVE_FIXTURE_NON_PRODUCT" {
		t.Fatal("negative fixture lost its non-product label")
	}
	if first.status != 404 || second.status != 404 {
		t.Fatalf("shared-root statuses = %d/%d, want false 404/404 equality", first.status, second.status)
	}
	if first.requestID == second.requestID || first.requestID == "" || second.requestID == "" ||
		first.scratchRoot != sharedState || second.scratchRoot != sharedState {
		t.Fatal("negative fixture did not retain distinct invocation IDs and the exact shared root")
	}
	if bytes.Equal(first.body, second.body) {
		t.Fatal("volatile captured bodies unexpectedly became equal before projection")
	}
	if !bytes.Equal(first.projection, second.projection) {
		t.Fatalf("shared-root contamination did not create false projected equality:\n%s\n%s", first.projection, second.projection)
	}
	if !bytes.Contains(first.body, []byte(first.requestID)) || !bytes.Contains(second.body, []byte(second.requestID)) ||
		!bytes.Contains(first.body, []byte(sharedState)) || !bytes.Contains(second.body, []byte(sharedState)) {
		t.Fatal("negative captured evidence lost the volatile request ID or scratch root")
	}
	if len(first.invocationRaw) == 0 || len(second.invocationRaw) == 0 ||
		bytes.Equal(first.invocationRaw, second.invocationRaw) {
		t.Fatal("negative fixture did not retain distinct invocation evidence")
	}
	t.Logf("%s shared_root=%s statuses=404/404 projected_equal=true", negativeFixtureLabel, sharedState)
}

func runNegativeFixture(
	t *testing.T,
	node, parent, sharedState string,
	role httpfixture.CandidateRole,
	index int,
) negativeFixtureResponse {
	t.Helper()
	attemptRoot := filepath.Join(parent, "negative-"+strconv.Itoa(index)+"-"+string(role))
	candidateRoot := filepath.Join(attemptRoot, "candidate")
	fixtureRoot := filepath.Join(attemptRoot, "fixture")
	evidenceRoot := filepath.Join(attemptRoot, "evidence")
	homeRoot := filepath.Join(attemptRoot, "home")
	temporaryRoot := filepath.Join(attemptRoot, "tmp")
	for _, path := range []string{attemptRoot, candidateRoot, fixtureRoot, evidenceRoot, homeRoot, temporaryRoot} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	files, err := httpfixture.CandidateFiles(role)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		path := filepath.Join(candidateRoot, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o600)
		if file.Mode == "100755" {
			mode = 0o700
		}
		if err := os.WriteFile(path, file.Content, mode); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(fixtureRoot, httpfixture.SeedFilename), httpfixture.SeedJSON(), 0o600); err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		_ = listener.Close()
		t.Fatal("loopback listener is not TCP")
	}
	listenerFile, err := tcpListener.File()
	if err != nil {
		_ = listener.Close()
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		_ = listenerFile.Close()
		t.Fatal(err)
	}
	readinessRead, readinessWrite, err := os.Pipe()
	if err != nil {
		_ = listenerFile.Close()
		t.Fatal(err)
	}

	attemptID := fmt.Sprintf("attempt:negative-%d-%s", index, role)
	stimulusDigest := fmt.Sprintf("sha256:%064x", index+1)
	command := exec.Command(node, httpfixture.Entrypoint)
	command.Dir = candidateRoot
	command.Env = []string{
		"HOME=" + homeRoot,
		"TMPDIR=" + temporaryRoot,
		"LANG=C",
		"LC_ALL=C",
		"TZ=UTC",
		"NO_COLOR=1",
		"NODE_NO_WARNINGS=1",
		"COUNTERSHAPE_ATTEMPT_ID=" + attemptID,
		"COUNTERSHAPE_EVIDENCE_ROOT=" + evidenceRoot,
		"COUNTERSHAPE_FIXTURE_ROOT=" + fixtureRoot,
		"COUNTERSHAPE_STATE_ROOT=" + sharedState,
		"COUNTERSHAPE_SCHEDULE_REPETITION=0",
		"COUNTERSHAPE_HTTP_STIMULUS_DIGEST=" + stimulusDigest,
	}
	command.ExtraFiles = []*os.File{listenerFile, readinessWrite}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		_ = listenerFile.Close()
		_ = readinessRead.Close()
		_ = readinessWrite.Close()
		t.Fatal(err)
	}
	_ = listenerFile.Close()
	_ = readinessWrite.Close()
	ready := make(chan struct {
		bytes []byte
		err   error
	}, 1)
	go func() {
		value, readErr := io.ReadAll(readinessRead)
		_ = readinessRead.Close()
		ready <- struct {
			bytes []byte
			err   error
		}{value, readErr}
	}()
	select {
	case receipt := <-ready:
		if receipt.err != nil || !bytes.Equal(receipt.bytes, []byte{0x01}) {
			_ = command.Process.Kill()
			_ = command.Wait()
			t.Fatalf("negative readiness = %x %v, want exact 01+EOF; stderr=%q", receipt.bytes, receipt.err, stderr.Bytes())
		}
	case <-time.After(5 * time.Second):
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatal("negative fixture readiness timed out")
	}

	connection, err := net.DialTimeout("tcp4", address, 3*time.Second)
	if err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("dial negative fixture: %v; stderr=%q", err, stderr.Bytes())
	}
	port := connection.RemoteAddr().(*net.TCPAddr).Port
	stimulus, err := newInvoiceStimulus()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := counterhttp.EncodeRequest(stimulus, port)
	if err != nil {
		t.Fatal(err)
	}
	request := encoded.Bytes()
	if err := connection.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		_ = connection.Close()
		t.Fatal(err)
	}
	if _, err := connection.Write(request); err != nil {
		_ = connection.Close()
		t.Fatal(err)
	}
	responseWire, err := io.ReadAll(connection)
	_ = connection.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("negative fixture exited unsuccessfully: %v stdout=%q stderr=%q", err, stdout.Bytes(), stderr.Bytes())
	}
	invocationRaw, err := os.ReadFile(filepath.Join(evidenceRoot, httpfixture.InvocationReceiptFilename))
	if err != nil {
		t.Fatal(err)
	}
	return parseNegativeResponse(t, responseWire, invocationRaw)
}

func parseNegativeResponse(t *testing.T, wire, invocationRaw []byte) negativeFixtureResponse {
	t.Helper()
	policy, err := counterhttp.NewHTTPCapturePolicy(counterhttp.HTTPCapturePolicyConfig{
		StatusLineBytes: 1024, HeaderBytes: 32 << 10, HeaderCount: 64, BodyBytes: 64 << 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := counterhttp.ParseResponse(wire, policy)
	if err != nil {
		t.Fatalf("production HTTP parser rejected negative fixture response: %v", err)
	}
	body := response.Body()
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	requestID, requestPresent := value["request_id"].(string)
	scratchRoot, rootPresent := value["scratch_root"].(string)
	if !requestPresent || !rootPresent {
		t.Fatal("negative response omitted volatile capture")
	}
	// The runner is deliberately non-product, but the semantic equality it
	// demonstrates is produced by the exact same fixed projection core used by
	// the lineage-gated product Project operation.
	projection, err := counterhttp.ProjectParsedResponse(response, scratchRoot)
	if err != nil {
		t.Fatalf("production projection rejected negative fixture response: %v", err)
	}
	return negativeFixtureResponse{
		status: response.Status(), body: body, projection: projection,
		requestID: requestID, scratchRoot: scratchRoot, invocationRaw: append([]byte(nil), invocationRaw...),
	}
}

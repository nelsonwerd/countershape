//go:build darwin

package http

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func TestC5WaitAndRuntimeRevalidationFailuresRetainDistinctCauses(t *testing.T) {
	result := processResult{}
	classifyWait(&result, waitObservation{err: errors.New("wait failed exactly")})
	recordRuntimeRevalidation(&result, errors.New("runtime changed exactly"))
	if result.waitError != "wait failed exactly" ||
		result.runtimeRevalidationError != "runtime changed exactly" ||
		!result.teardownError || !result.orphanRisk ||
		result.diagnosticCode != "WAIT_FAILED" {
		t.Fatalf(
			"causal facts wait=%q runtime=%q teardown=%t orphan=%t diagnostic=%q",
			result.waitError, result.runtimeRevalidationError,
			result.teardownError, result.orphanRisk, result.diagnosticCode,
		)
	}

	closeOnly := processResult{diagnosticCode: "WAIT_FAILED"}
	closeErr := errors.New("descriptor close failed exactly")
	recordDescriptorCloseFailure(&closeOnly, closeFailureTerminalReaders, closeErr)
	recordDescriptorCloseFailure(&closeOnly, closeFailureStartError, closeErr)
	recordDescriptorCloseFailure(&closeOnly, closeFailureParentWriters, closeErr)
	recordDescriptorCloseFailure(&closeOnly, closeFailureTerminalReaders, closeErr)
	if !closeOnly.teardownError || closeOnly.orphanRisk ||
		closeOnly.diagnosticCode != "WAIT_FAILED" ||
		strings.Join(closeOnly.descriptorCloseDiagnostics(), ",") !=
			"HTTP_START_ERROR_DESCRIPTOR_CLOSE_FAILED,"+
				"HTTP_PARENT_WRITER_CLOSE_FAILED,"+
				"HTTP_TERMINAL_READER_CLOSE_FAILED" {
		t.Fatalf(
			"descriptor-close facts teardown=%t orphan=%t diagnostic=%q close=%v",
			closeOnly.teardownError, closeOnly.orphanRisk, closeOnly.diagnosticCode,
			closeOnly.descriptorCloseDiagnostics(),
		)
	}

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	owned := newOwnedServiceDescriptor(writer)
	underlying := owned.closeFn
	closeCalls := 0
	owned.closeFn = func(file *os.File) error {
		closeCalls++
		return errors.Join(underlying(file), closeErr)
	}
	firstClose := owned.Close()
	secondClose := owned.Close()
	if closeCalls != 1 || !errors.Is(firstClose, closeErr) || !errors.Is(secondClose, closeErr) {
		t.Fatalf(
			"descriptor close-once calls=%d first=%v second=%v",
			closeCalls, firstClose, secondClose,
		)
	}
}

func TestC5AwaitExactReadinessControlMatrix(t *testing.T) {
	contract, err := httpmodel.NewPortableHTTPReadinessContract()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name       string
		payload    []byte
		closeWrite bool
		overflow   bool
		budget     time.Duration
		control    domain.ControlReason
		diagnostic string
		eof        bool
	}{
		{
			name: "partial-eof", payload: []byte("COUNTERSHAPE_READY_V1 43127"),
			closeWrite: true, budget: time.Second,
			control: domain.ControlReadinessError, diagnostic: "HTTP_READINESS_PROTOCOL_REJECTED", eof: true,
		},
		{
			name: "malformed-eof", payload: []byte("COUNTERSHAPE_READY_V1 01\n"),
			closeWrite: true, budget: time.Second,
			control: domain.ControlReadinessError, diagnostic: "HTTP_READINESS_PROTOCOL_REJECTED", eof: true,
		},
		{
			name: "withheld-timeout", budget: 100 * time.Millisecond,
			control: domain.ControlReadinessError, diagnostic: "HTTP_READINESS_TIMEOUT",
		},
		{
			name: "process-output-overflow", overflow: true, budget: time.Second,
			control: domain.ControlOutputLimit, diagnostic: "HTTP_PROCESS_OUTPUT_LIMIT_BEFORE_READINESS",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = reader.Close()
				_ = writer.Close()
			})
			if len(testCase.payload) > 0 {
				if count, writeErr := writer.Write(testCase.payload); writeErr != nil ||
					count != len(testCase.payload) {
					t.Fatalf("write readiness payload count=%d err=%v", count, writeErr)
				}
			}
			if testCase.closeWrite {
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
			}
			overflowC := make(chan struct{}, 2)
			stdout := newCappedCapture(8, overflowC)
			stderr := newCappedCapture(8, overflowC)
			if testCase.overflow {
				stdout.retain([]byte("123456789"))
			}
			waitC := make(chan waitObservation, 1)
			result, waited, control, diagnostic := awaitExactReadiness(
				context.Background(), reader, testCase.budget,
				stdout, stderr, overflowC, waitC, contract,
			)
			if waited != nil || control != testCase.control || diagnostic != testCase.diagnostic ||
				result.eof != testCase.eof || !equalReadinessBytes(result.bytes, testCase.payload) {
				t.Fatalf("readiness control waited=%v control=%s diagnostic=%s bytes=%q eof=%t err=%v",
					waited, control, diagnostic, result.bytes, result.eof, result.err)
			}
		})
	}
	t.Run("early-close-error-retained-without-retry", func(t *testing.T) {
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = writer.Close() })
		owned := newOwnedServiceDescriptor(reader)
		closeErr := errors.New("early readiness close failed exactly")
		closeFn := owned.closeFn
		closeCalls := 0
		owned.closeFn = func(file *os.File) error {
			closeCalls++
			return errors.Join(closeFn(file), closeErr)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		overflowC := make(chan struct{}, 1)
		stdout := newCappedCapture(8, overflowC)
		stderr := newCappedCapture(8, overflowC)
		waitC := make(chan waitObservation, 1)
		_, waited, control, diagnostic := awaitExactReadiness(
			ctx,
			owned,
			time.Second,
			stdout,
			stderr,
			overflowC,
			waitC,
			contract,
		)
		if closeCalls != 1 {
			t.Fatalf("stopReadiness did not perform the first close exactly once: calls=%d", closeCalls)
		}
		cachedCloseErr := owned.Close()
		if closeCalls != 1 || !errors.Is(cachedCloseErr, closeErr) {
			t.Fatalf(
				"terminal readiness close did not reuse the first close result: calls=%d err=%v",
				closeCalls,
				cachedCloseErr,
			)
		}
		process := processResult{diagnosticCode: diagnostic}
		recordDescriptorCloseFailure(
			&process,
			closeFailureTerminalReaders,
			cachedCloseErr,
		)
		if waited != nil || control != domain.ControlCancelled ||
			diagnostic != "HTTP_READINESS_CANCELLED" ||
			closeCalls != 1 || !process.teardownError || process.orphanRisk ||
			process.diagnosticCode != "HTTP_READINESS_CANCELLED" ||
			strings.Join(process.descriptorCloseDiagnostics(), ",") !=
				"HTTP_TERMINAL_READER_CLOSE_FAILED" {
			t.Fatalf(
				"early close waited=%v control=%s diagnostic=%s calls=%d teardown=%t orphan=%t close=%v",
				waited,
				control,
				diagnostic,
				closeCalls,
				process.teardownError,
				process.orphanRisk,
				process.descriptorCloseDiagnostics(),
			)
		}
	})
}

func TestC5ResponseErrorControlMatrix(t *testing.T) {
	standard, err := httpmodel.NewHTTPCapturePolicy(httpmodel.HTTPCapturePolicyConfig{
		StatusLineBytes: 1024, HeaderBytes: 4096, HeaderCount: 32, BodyBytes: 32,
	})
	if err != nil {
		t.Fatal(err)
	}
	statusLimited, err := httpmodel.NewHTTPCapturePolicy(httpmodel.HTTPCapturePolicyConfig{
		StatusLineBytes: 16, HeaderBytes: 4096, HeaderCount: 32, BodyBytes: 32,
	})
	if err != nil {
		t.Fatal(err)
	}
	bodyLimited, err := httpmodel.NewHTTPCapturePolicy(httpmodel.HTTPCapturePolicyConfig{
		StatusLineBytes: 1024, HeaderBytes: 4096, HeaderCount: 32, BodyBytes: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name       string
		raw        []byte
		policy     httpmodel.HTTPCapturePolicy
		control    domain.ControlReason
		diagnostic string
	}{
		{
			name: "bad-status",
			raw:  []byte("HTTP/1.0 200 OK\r\ncontent-length: 0\r\n\r\n"), policy: standard,
			control: domain.ControlProbeTransportError, diagnostic: httpmodel.CodeResponseStatus,
		},
		{
			name: "status-limit",
			raw:  []byte("HTTP/1.1 200 Very Long Reason\r\ncontent-length: 0\r\n\r\n"), policy: statusLimited,
			control: domain.ControlOutputLimit, diagnostic: httpmodel.CodeResponseStatusLimit,
		},
		{
			name: "body-limit",
			raw:  []byte("HTTP/1.1 200 OK\r\ncontent-length: 4\r\n\r\nfour"), policy: bodyLimited,
			control: domain.ControlOutputLimit, diagnostic: httpmodel.CodeResponseBodyLimit,
		},
		{
			name: "length-mismatch",
			raw:  []byte("HTTP/1.1 200 OK\r\ncontent-length: 3\r\n\r\nab"), policy: bodyLimited,
			control: domain.ControlProbeTransportError, diagnostic: httpmodel.CodeResponseLength,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, parseErr := httpmodel.ParseResponse(testCase.raw, testCase.policy)
			if parseErr == nil {
				t.Fatal("response control fixture unexpectedly parsed")
			}
			control, diagnostic := classifyResponseError(parseErr)
			if control != testCase.control || diagnostic != testCase.diagnostic {
				t.Fatalf("response control=%s diagnostic=%s want=%s/%s",
					control, diagnostic, testCase.control, testCase.diagnostic)
			}
		})
	}
}

func TestC5TeardownProcessGroupEscalatesAndCleans(t *testing.T) {
	command := exec.Command(os.Args[0], "-test.run=^TestC5TeardownHelper$")
	command.Env = append(os.Environ(), "COUNTERSHAPE_C5_TEARDOWN_HELPER=1")
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if command.ProcessState == nil {
			_ = command.Process.Kill()
			_, _ = command.Process.Wait()
		}
	})
	ready := make([]byte, len("ready\n"))
	if _, err := io.ReadFull(stdout, ready); err != nil || string(ready) != "ready\n" {
		t.Fatalf("teardown helper readiness=%q err=%v", ready, err)
	}
	result := processResult{
		started: true, pid: command.Process.Pid,
		processGroupID: command.Process.Pid, processGroupOwned: true,
		preTermProbe: preTermNotApplicable,
	}
	teardownProcessGroup(&result, time.Now().Add(2*time.Second), 900*time.Millisecond)
	waitErr := command.Wait()
	var exitError *exec.ExitError
	if !errors.As(waitErr, &exitError) {
		t.Fatalf("teardown helper did not exit by signal: %v", waitErr)
	}
	clean, probeErr := waitForGroupAbsence(result.processGroupID, time.Now().Add(time.Second))
	if result.preTermProbe != preTermPresent || !result.termSent || !result.killSent ||
		result.teardownError || result.orphanRisk || result.diagnosticCode != "" ||
		!clean || probeErr != nil {
		t.Fatalf("teardown escalation preterm=%s term=%t kill=%t error=%t orphan=%t diagnostic=%s clean=%t probe=%v",
			result.preTermProbe, result.termSent, result.killSent, result.teardownError,
			result.orphanRisk, result.diagnosticCode, clean, probeErr)
	}
}

func TestC5TeardownHelper(t *testing.T) {
	if os.Getenv("COUNTERSHAPE_C5_TEARDOWN_HELPER") != "1" {
		return
	}
	signal.Ignore(syscall.SIGTERM)
	if _, err := os.Stdout.WriteString("ready\n"); err != nil {
		os.Exit(97)
	}
	select {}
}

func equalReadinessBytes(left, right []byte) bool {
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

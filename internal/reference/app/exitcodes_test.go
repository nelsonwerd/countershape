package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMachineStudyExitCodesDoNotPromoteIncompleteOrForgedSuccess(t *testing.T) {
	tests := []struct {
		name        string
		handler     StudyHandler
		wantExit    int
		wantFailure string
		wantGreen   bool
	}{
		{
			name: "closed evidence",
			handler: func(request StudyRequest) error {
				return request.Workspace.WriteFileExclusive("probe.json", []byte("{}\n"))
			},
			wantExit: ExitOK, wantGreen: true,
		},
		{
			name: "typed authority refusal",
			handler: func(StudyRequest) error {
				return &InputError{Code: "STUDY_JOIN_REFUSED", Detail: "the supplied domains do not form one closed result"}
			},
			wantExit: ExitRefused, wantFailure: "STUDY_JOIN_REFUSED",
		},
		{
			name: "unexpected internal failure",
			handler: func(StudyRequest) error {
				return errors.New("untrusted internal detail must not escape")
			},
			wantExit: ExitInternal, wantFailure: "INTERNAL_FAILURE",
		},
		{
			name: "artifact then internal failure",
			handler: func(request StudyRequest) error {
				if err := request.Workspace.WriteFileExclusive("plausible.json", []byte("{}\n")); err != nil {
					return err
				}
				return errors.New("untrusted post-publication detail must not escape")
			},
			wantExit: ExitInternal, wantFailure: "INTERNAL_FAILURE",
		},
		{
			name: "panic",
			handler: func(StudyRequest) error {
				panic("untrusted panic detail")
			},
			wantExit: ExitInternal, wantFailure: "INTERNAL_FAILURE",
		},
		{
			name: "artifact then panic",
			handler: func(request StudyRequest) error {
				if err := request.Workspace.WriteFileExclusive("plausible.json", []byte("{}\n")); err != nil {
					return err
				}
				panic("untrusted post-publication panic detail")
			},
			wantExit: ExitInternal, wantFailure: "INTERNAL_FAILURE",
		},
		{
			name:        "missing terminal evidence",
			handler:     func(StudyRequest) error { return nil },
			wantExit:    ExitRefused,
			wantFailure: "STUDY_EVIDENCE_INCOMPLETE",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := runExitCodeMachineStudy(t, context.Background(), map[string]StudyHandler{"http": test.handler}, []string{"study", "http", "--json"})
			if result.code != test.wantExit || result.stderr != "" {
				t.Fatalf("machine result exit=%d stderr=%q stdout=%q", result.code, result.stderr, result.stdout)
			}
			var value map[string]any
			if err := json.Unmarshal([]byte(result.stdout), &value); err != nil {
				t.Fatalf("decode machine result: %v; raw=%q", err, result.stdout)
			}
			if test.wantGreen {
				want := map[string]any{
					"schema_version": StudyDomainResultSchemaVersion,
					"domain":         "http",
					"ordinal":        float64(1),
					"status":         "GREEN",
				}
				if !reflect.DeepEqual(value, want) {
					t.Fatalf("machine success = %#v, want exact four-field result %#v", value, want)
				}
				return
			}
			failure, ok := value["failure"].(map[string]any)
			if !ok || failure["code"] != test.wantFailure || value["status"] != "REFUSED" || value["exit_code"] != float64(test.wantExit) {
				t.Fatalf("machine refusal = %#v, want exit=%d failure=%s", value, test.wantExit, test.wantFailure)
			}
			if next, _ := value["next_action"].(string); !strings.Contains(next, "Leave this attempt unclaimed") {
				t.Fatalf("machine refusal omitted the evidence-safe next action: %#v", value)
			}
			if strings.Contains(result.stdout, "untrusted") {
				t.Fatalf("machine refusal disclosed internal detail: %q", result.stdout)
			}
		})
	}
}

func TestMachineStudyUsageConfigurationAndCancellationNeverInvokeHandler(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		studies     map[string]StudyHandler
		ctx         context.Context
		wantExit    int
		wantFailure string
	}{
		{
			name: "extra machine argument", args: []string{"study", "http", "--json", "extra"},
			studies: map[string]StudyHandler{"http": func(StudyRequest) error { return nil }},
			ctx:     context.Background(), wantExit: ExitUsage, wantFailure: "ARGUMENT_UNKNOWN",
		},
		{
			name: "uninstalled domain", args: []string{"study", "cli", "--json"},
			studies: map[string]StudyHandler{"http": func(StudyRequest) error { return nil }},
			ctx:     context.Background(), wantExit: ExitRefused, wantFailure: "STUDY_HANDLER_UNAVAILABLE",
		},
		{
			name: "invalid installed handler", args: []string{"study", "http", "--json"},
			studies: map[string]StudyHandler{"http": nil},
			ctx:     context.Background(), wantExit: ExitRefused, wantFailure: "STUDY_CONFIGURATION_INVALID",
		},
		{
			name: "canceled before handler", args: []string{"study", "http", "--json"},
			studies: map[string]StudyHandler{"http": func(StudyRequest) error { return nil }},
			ctx:     canceledExitCodeContext(), wantExit: ExitRefused, wantFailure: "STUDY_CONTEXT_CANCELED",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invoked := false
			studies := make(map[string]StudyHandler, len(test.studies))
			for domain, handler := range test.studies {
				if handler == nil {
					studies[domain] = nil
					continue
				}
				original := handler
				studies[domain] = func(request StudyRequest) error {
					invoked = true
					return original(request)
				}
			}
			result := runExitCodeMachineStudy(t, test.ctx, studies, test.args)
			if result.code != test.wantExit || result.stderr != "" || !strings.Contains(result.stdout, `"code":"`+test.wantFailure+`"`) {
				t.Fatalf("machine refusal exit=%d stderr=%q stdout=%q", result.code, result.stderr, result.stdout)
			}
			if invoked {
				t.Fatal("refused machine route invoked a study handler")
			}
		})
	}
}

func TestMachineStudyRejectsEvidenceMutationAfterTerminalCheck(t *testing.T) {
	mutate := make(chan struct{})
	mutated := make(chan error, 1)
	handler := CloseStudy(func(request StudyRequest) error {
		if err := request.Workspace.WriteFileExclusive("probe.json", []byte("{\"value\":\"checked\"}\n")); err != nil {
			return err
		}
		go func(path string) {
			<-mutate
			mutated <- os.WriteFile(path, []byte("{\"value\":\"mutated-after-check\"}\n"), 0o600)
		}(filepath.Join(request.EvidenceRoot, "probe.json"))
		return nil
	}, func(_ StudyRequest, snapshot StudyEvidenceSnapshot) (StudyTerminalCommit, error) {
		files := snapshot.Files()
		if len(snapshot.Directories()) != 0 || len(files) != 1 || files[0].Path() != "probe.json" ||
			!bytes.Equal(files[0].ExactBytes(), []byte("{\"value\":\"checked\"}\n")) {
			return nil, errors.New("terminal check did not receive the expected stable snapshot")
		}
		// The mutation belongs to a goroutine started by the handler but is
		// released only after the semantic check has accepted the first snapshot.
		close(mutate)
		if err := <-mutated; err != nil {
			return nil, err
		}
		return func() error { return nil }, nil
	})

	result := runExitCodeMachineStudy(t, context.Background(), map[string]StudyHandler{"http": handler}, []string{"study", "http", "--json"})
	if result.code != ExitRefused || result.stderr != "" ||
		!strings.Contains(result.stdout, `"code":"STUDY_EVIDENCE_CLOSURE_REFUSED"`) ||
		strings.Contains(result.stdout, `"status":"GREEN"`) {
		t.Fatalf("post-check mutation false-green: %#v", result)
	}
}

func TestEvidenceWorkspaceCloseRejectsBoundSnapshotMutationAndStillCloses(t *testing.T) {
	root := privateExitCodeDirectory(t, "late-write")
	metadata, err := os.Lstat(root)
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := newEvidenceWorkspace(root, metadata)
	if err != nil {
		t.Fatal(err)
	}
	if err := workspace.WriteFileExclusive("probe.json", []byte("{\"value\":\"checked\"}\n")); err != nil {
		t.Fatal(err)
	}
	snapshot, err := workspace.SnapshotEvidence()
	if err != nil {
		t.Fatal(err)
	}
	committed := false
	if err := workspace.bindValidatedSnapshot(snapshot, func() error {
		committed = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "probe.json"), []byte("{\"value\":\"mutated-after-check\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	closeErr := workspace.close()
	var input *InputError
	if !errors.As(closeErr, &input) || input.Code != "STUDY_EVIDENCE_CLOSURE_REFUSED" {
		t.Fatalf("late-write close err=%v, want typed closure refusal", closeErr)
	}
	if committed || workspace.root != nil || !workspace.closed || workspace.validation != nil {
		t.Fatal("failed terminal validation left the workspace capability open")
	}
}

func TestMachineStudyDefersTerminalCommitUntilEveryOuterGatePasses(t *testing.T) {
	t.Run("outer handler failure", func(t *testing.T) {
		committed := false
		inner := terminalCommitProbeHandler(&committed)
		outer := func(request StudyRequest) error {
			if err := inner(request); err != nil {
				return err
			}
			return errors.New("untrusted outer failure after inner terminal check")
		}
		result := runExitCodeMachineStudy(t, context.Background(), map[string]StudyHandler{"http": outer}, []string{"study", "http", "--json"})
		if committed || result.code != ExitInternal || result.stderr != "" ||
			!strings.Contains(result.stdout, `"code":"INTERNAL_FAILURE"`) || strings.Contains(result.stdout, `"status":"GREEN"`) {
			t.Fatalf("outer failure published terminal commit: committed=%t result=%#v", committed, result)
		}
	})

	t.Run("nested terminal binding refusal", func(t *testing.T) {
		innerCommitted := false
		outerCommitted := false
		inner := terminalCommitProbeHandler(&innerCommitted)
		outer := CloseStudy(inner, func(StudyRequest, StudyEvidenceSnapshot) (StudyTerminalCommit, error) {
			return func() error {
				outerCommitted = true
				return nil
			}, nil
		})
		result := runExitCodeMachineStudy(t, context.Background(), map[string]StudyHandler{"http": outer}, []string{"study", "http", "--json"})
		if innerCommitted || outerCommitted || result.code != ExitRefused || result.stderr != "" ||
			!strings.Contains(result.stdout, `"code":"STUDY_EVIDENCE_CLOSURE_REFUSED"`) ||
			strings.Contains(result.stdout, `"status":"GREEN"`) {
			t.Fatalf("nested bind refusal published terminal commit: inner=%t outer=%t result=%#v",
				innerCommitted, outerCommitted, result)
		}
	})

	t.Run("boundary verification failure", func(t *testing.T) {
		committed := false
		inner := terminalCommitProbeHandler(&committed)
		outer := func(request StudyRequest) error {
			if err := inner(request); err != nil {
				return err
			}
			return os.Chmod(request.WorkingDirectory, 0o755)
		}
		result := runExitCodeMachineStudy(t, context.Background(), map[string]StudyHandler{"http": outer}, []string{"study", "http", "--json"})
		if committed || result.code != ExitRefused || result.stderr != "" ||
			!strings.Contains(result.stdout, `"code":"STUDY_WORKSPACE_CHANGED"`) || strings.Contains(result.stdout, `"status":"GREEN"`) {
			t.Fatalf("boundary failure published terminal commit: committed=%t result=%#v", committed, result)
		}
	})

	t.Run("final context failure", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		committed := false
		inner := terminalCommitProbeHandler(&committed)
		outer := func(request StudyRequest) error {
			if err := inner(request); err != nil {
				return err
			}
			cancel()
			return nil
		}
		result := runExitCodeMachineStudy(t, ctx, map[string]StudyHandler{"http": outer}, []string{"study", "http", "--json"})
		if committed || result.code != ExitRefused || result.stderr != "" ||
			!strings.Contains(result.stdout, `"code":"STUDY_CONTEXT_CANCELED"`) || strings.Contains(result.stdout, `"status":"GREEN"`) {
			t.Fatalf("final context failure published terminal commit: committed=%t result=%#v", committed, result)
		}
	})

	t.Run("terminal commit cancels context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		committed := false
		handler := CloseStudy(func(request StudyRequest) error {
			return request.Workspace.WriteFileExclusive("probe.json", []byte("{}\n"))
		}, func(StudyRequest, StudyEvidenceSnapshot) (StudyTerminalCommit, error) {
			return func() error {
				committed = true
				cancel()
				return nil
			}, nil
		})
		result := runExitCodeMachineStudy(t, ctx, map[string]StudyHandler{"http": handler}, []string{"study", "http", "--json"})
		if !committed || result.code != ExitRefused || result.stderr != "" ||
			!strings.Contains(result.stdout, `"code":"STUDY_CONTEXT_CANCELED"`) || strings.Contains(result.stdout, `"status":"GREEN"`) {
			t.Fatalf("commit cancellation false-green: committed=%t result=%#v", committed, result)
		}
	})
}

func terminalCommitProbeHandler(committed *bool) StudyHandler {
	return CloseStudy(func(request StudyRequest) error {
		return request.Workspace.WriteFileExclusive("probe.json", []byte("{}\n"))
	}, func(StudyRequest, StudyEvidenceSnapshot) (StudyTerminalCommit, error) {
		return func() error {
			*committed = true
			return nil
		}, nil
	})
}

func TestCloseStudyRefusesMissingFailingAndPanickingTerminalCommits(t *testing.T) {
	tests := []struct {
		name  string
		check StudyTerminalCheck
	}{
		{
			name: "missing commit",
			check: func(StudyRequest, StudyEvidenceSnapshot) (StudyTerminalCommit, error) {
				return nil, nil
			},
		},
		{
			name: "failing commit",
			check: func(StudyRequest, StudyEvidenceSnapshot) (StudyTerminalCommit, error) {
				return func() error { return errors.New("untrusted commit failure") }, nil
			},
		},
		{
			name: "panicking commit",
			check: func(StudyRequest, StudyEvidenceSnapshot) (StudyTerminalCommit, error) {
				return func() error { panic("untrusted commit panic") }, nil
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := CloseStudy(func(request StudyRequest) error {
				return request.Workspace.WriteFileExclusive("probe.json", []byte("{}\n"))
			}, test.check)
			result := runExitCodeMachineStudy(t, context.Background(), map[string]StudyHandler{"http": handler}, []string{"study", "http", "--json"})
			if result.code != ExitRefused || result.stderr != "" ||
				!strings.Contains(result.stdout, `"code":"STUDY_EVIDENCE_CLOSURE_REFUSED"`) ||
				strings.Contains(result.stdout, "untrusted") || strings.Contains(result.stdout, `"status":"GREEN"`) {
				t.Fatalf("terminal commit false-green: %#v", result)
			}
		})
	}
}

func TestCloseStudyTerminalCheckCannotReuseTheClosedNodeRunner(t *testing.T) {
	checked := false
	handler := CloseStudy(func(request StudyRequest) error {
		return request.Workspace.WriteFileExclusive("probe.json", []byte("{}\n"))
	}, func(request StudyRequest, _ StudyEvidenceSnapshot) (StudyTerminalCommit, error) {
		if _, err := request.RunNodeTest(context.Background(), StudyNodeTestInput{}); err == nil || !strings.Contains(err.Error(), "runner is closed") {
			return nil, errors.New("terminal check retained live Node-runner authority")
		}
		checked = true
		return func() error { return nil }, nil
	})
	result := runExitCodeMachineStudy(t, context.Background(), map[string]StudyHandler{"http": handler}, []string{"study", "http", "--json"})
	if !checked || result.code != ExitOK || result.stderr != "" || !strings.Contains(result.stdout, `"status":"GREEN"`) {
		t.Fatalf("terminal check runner order result=%#v checked=%t", result, checked)
	}
}

func canceledExitCodeContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

type exitCodeRun struct {
	code           int
	stdout, stderr string
}

func runExitCodeMachineStudy(t testing.TB, ctx context.Context, studies map[string]StudyHandler, args []string) exitCodeRun {
	t.Helper()
	root := privateExitCodeDirectory(t, "root")
	working := privateExitCodeSubdirectory(t, root, "working")
	scratch := privateExitCodeSubdirectory(t, root, "scratch")
	evidence := privateExitCodeSubdirectory(t, root, "evidence")
	tools := privateExitCodeSubdirectory(t, root, "tools")
	node := exitCodeExecutable(t, tools, "node")
	git := exitCodeExecutable(t, tools, "git")
	environment := map[string]string{
		"COUNTERSHAPE_STUDY_DOMAIN":  "http",
		"COUNTERSHAPE_STUDY_ORDINAL": "1",
		"COUNTERSHAPE_EVIDENCE_ROOT": evidence,
		"COUNTERSHAPE_NODE":          node,
		"COUNTERSHAPE_GIT":           git,
		"TMPDIR":                     scratch,
	}
	var stdout, stderr bytes.Buffer
	code := Run(args, Runtime{
		Stdout: &stdout, Stderr: &stderr, WorkingDirectory: working, Context: ctx, Studies: studies,
		LookupEnvironment: func(key string) (string, bool) {
			value, present := environment[key]
			return value, present
		},
	})
	return exitCodeRun{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func privateExitCodeDirectory(t testing.TB, pattern string) string {
	t.Helper()
	path, err := os.MkdirTemp("", "countershape-u7d-"+pattern+"-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || !filepath.IsAbs(resolved) || filepath.Clean(resolved) != resolved {
		t.Fatalf("private root cannot be canonicalized: path=%q resolved=%q err=%v", path, resolved, err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(resolved) })
	return resolved
}

func privateExitCodeSubdirectory(t testing.TB, parent, name string) string {
	t.Helper()
	path := filepath.Join(parent, name)
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func exitCodeExecutable(t testing.TB, root, name string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

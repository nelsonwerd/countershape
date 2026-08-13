package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
)

// StudyHandler is the domain-neutral application seam for one frozen study
// route. Domain packages implement the handler; app owns admission, dispatch,
// presentation, and the process-wide success protocol.
type StudyHandler func(StudyRequest) error

// StudyTerminalCommit publishes only inert facts prepared by a successful
// StudyTerminalCheck. App invokes it at most once, and only after the Node
// runner is terminal, the checked evidence bytes have been re-read unchanged,
// and every application gate preceding the commit has succeeded.
type StudyTerminalCommit func() error

// StudyTerminalCheck consumes a defensive byte snapshot only after the
// installed domain handler has completed and its Node runner is terminal. The
// check may validate transport shape and joins, but must defer state mutation
// to its returned commit. App remains the owner of the evidence-root capability
// and the four-field process result.
type StudyTerminalCheck func(StudyRequest, StudyEvidenceSnapshot) (StudyTerminalCommit, error)

type studyTerminalCommitContextKey struct{}

type studyTerminalCommitCoordinator struct {
	mu     sync.Mutex
	staged StudyTerminalCommit
	taken  bool
}

func (coordinator *studyTerminalCommitCoordinator) stage(commit StudyTerminalCommit) error {
	if coordinator == nil || commit == nil {
		return errors.New("terminal evidence commit coordinator is invalid")
	}
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	if coordinator.staged != nil || coordinator.taken {
		return errors.New("terminal evidence commit was already staged")
	}
	coordinator.staged = commit
	return nil
}

func (coordinator *studyTerminalCommitCoordinator) take() StudyTerminalCommit {
	if coordinator == nil {
		return nil
	}
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	if coordinator.taken {
		return nil
	}
	coordinator.taken = true
	commit := coordinator.staged
	coordinator.staged = nil
	return commit
}

func terminalCommitCoordinator(ctx context.Context) *studyTerminalCommitCoordinator {
	if ctx == nil {
		return nil
	}
	coordinator, _ := ctx.Value(studyTerminalCommitContextKey{}).(*studyTerminalCommitCoordinator)
	return coordinator
}

// CloseStudy adds one domain-neutral terminal-evidence gate to a study
// handler. Invalid composition returns nil so completeRuntime rejects it as an
// unavailable implementation before any study execution begins.
func CloseStudy(handler StudyHandler, check StudyTerminalCheck) StudyHandler {
	if handler == nil || check == nil {
		return nil
	}
	return func(request StudyRequest) error {
		coordinator := terminalCommitCoordinator(request.Context)
		if coordinator == nil {
			return &InputError{Code: "STUDY_EVIDENCE_CLOSURE_REFUSED", Detail: "the terminal evidence check is outside the admitted application route"}
		}
		if err := handler(request); err != nil {
			return err
		}
		if request.Context == nil || request.Context.Err() != nil {
			return &InputError{Code: "STUDY_CONTEXT_CANCELED", Detail: "the admitted study context ended before terminal evidence closure"}
		}
		if request.Workspace == nil {
			return &InputError{Code: "STUDY_EVIDENCE_INCOMPLETE", Detail: "the handler returned without an admitted evidence workspace"}
		}
		if err := request.closeNodeTestRunner(); err != nil {
			return &InputError{Code: "STUDY_PROCESS_TERMINALITY_REFUSED", Detail: "the study process authority did not close before terminal evidence validation"}
		}
		snapshot, err := request.Workspace.SnapshotEvidence()
		if err != nil {
			return &InputError{Code: "STUDY_EVIDENCE_INVALID", Detail: "the terminal evidence tree could not be closed as exact private bytes"}
		}
		commit, err := check(request, snapshot)
		if err != nil {
			return err
		}
		if commit == nil {
			return &InputError{Code: "STUDY_EVIDENCE_CLOSURE_REFUSED", Detail: "the terminal evidence check returned no deferred commit"}
		}
		if err := request.Workspace.bindValidatedSnapshot(snapshot, func() error { return coordinator.stage(commit) }); err != nil {
			return &InputError{Code: "STUDY_EVIDENCE_CLOSURE_REFUSED", Detail: "the validated terminal evidence snapshot could not be retained"}
		}
		return nil
	}
}

type Runtime struct {
	Stdin                   io.Reader
	Stdout                  io.Writer
	Stderr                  io.Writer
	WorkingDirectory        string
	Columns                 int
	Context                 context.Context
	Studies                 map[string]StudyHandler
	LookupEnvironment       func(string) (string, bool)
	studyConfigurationError error
}

type invocation struct {
	command     string
	specPath    string
	studyDomain string
	json        bool
}

// Run is the domain-neutral composition shell used by cmd/countershape.
// Validate and preflight remain inert; installed studies are reachable only
// through their exact frozen machine routes.
func Run(args []string, runtime Runtime) int {
	runtime = completeRuntime(runtime)
	parsed, err := parseInvocation(args)
	var response ResponseEnvelope
	if err != nil {
		response = failureEnvelope(commandName(args), err)
	} else {
		switch parsed.command {
		case "help":
			response = helpEnvelope(runtime.Studies)
		case "validate":
			response = validateSourceCommand(parsed.specPath, runtime)
		case "preflight":
			response = preflightSourceCommand(parsed.specPath, runtime)
		case "study":
			if parsed.json {
				return runMachineStudyWithTerminalCommit(parsed.studyDomain, runtime)
			}
			response = humanStudyRefusal(parsed.studyDomain, runtime.Studies)
		default:
			response = failureEnvelope(parsed.command, &InputError{Code: "COMMAND_UNKNOWN", Detail: "the command is not available in this reference boundary"})
			response.ExitCode = ExitUsage
		}
	}

	writer := runtime.Stdout
	if !parsed.json && response.ExitCode != ExitOK {
		writer = runtime.Stderr
	}
	var renderErr error
	if parsed.json {
		renderErr = renderJSON(writer, response)
	} else {
		renderErr = renderHuman(writer, response, runtime.Columns)
	}
	if renderErr != nil {
		return ExitInternal
	}
	return response.ExitCode
}

func runMachineStudyWithTerminalCommit(domain string, runtime Runtime) int {
	coordinator := &studyTerminalCommitCoordinator{}
	buffered := runtime
	var output bytes.Buffer
	buffered.Stdout = &output
	buffered.Context = context.WithValue(runtime.Context, studyTerminalCommitContextKey{}, coordinator)

	code := runMachineStudy(domain, buffered)
	if code != ExitOK {
		_ = coordinator.take()
		return flushMachineStudyOutput(runtime.Stdout, output.Bytes(), code)
	}
	commit := coordinator.take()
	if commit != nil {
		if runtime.Context.Err() != nil {
			return renderTerminalStudyRefusal(runtime.Stdout, &InputError{Code: "STUDY_CONTEXT_CANCELED", Detail: "the admitted study context ended before terminal evidence publication"})
		}
		if err := invokeStudyTerminalCommit(commit); err != nil {
			return renderTerminalStudyRefusal(runtime.Stdout, &InputError{Code: "STUDY_EVIDENCE_CLOSURE_REFUSED", Detail: "the terminal evidence commit was refused"})
		}
	}
	if runtime.Context.Err() != nil {
		return renderTerminalStudyRefusal(runtime.Stdout, &InputError{Code: "STUDY_CONTEXT_CANCELED", Detail: "the admitted study context ended during terminal evidence publication"})
	}
	// The commit describes only terminal artifact-byte facts. A later writer
	// failure still returns ExitInternal and conveys no stdout or process-result
	// authority to the collector.
	return flushMachineStudyOutput(runtime.Stdout, output.Bytes(), ExitOK)
}

func flushMachineStudyOutput(writer io.Writer, exact []byte, code int) int {
	written, err := writer.Write(exact)
	if err != nil || written != len(exact) {
		return ExitInternal
	}
	return code
}

func renderTerminalStudyRefusal(writer io.Writer, err error) int {
	response := failureEnvelope("study", err)
	var input *InputError
	if errors.As(err, &input) {
		response.ExitCode = ExitRefused
	}
	response.NextAction = "Leave this attempt unclaimed, repair the frozen study boundary, and rerun it only through the repository harness."
	if renderErr := renderJSON(writer, response); renderErr != nil {
		return ExitInternal
	}
	return response.ExitCode
}

func completeRuntime(runtime Runtime) Runtime {
	if runtime.Stdin == nil {
		runtime.Stdin = os.Stdin
	}
	if runtime.Stdout == nil {
		runtime.Stdout = io.Discard
	}
	if runtime.Stderr == nil {
		runtime.Stderr = io.Discard
	}
	if runtime.Context == nil {
		runtime.Context = context.Background()
	}
	if runtime.LookupEnvironment == nil {
		runtime.LookupEnvironment = os.LookupEnv
	}
	if runtime.Studies != nil {
		copied := make(map[string]StudyHandler, len(runtime.Studies))
		for domain, handler := range runtime.Studies {
			if !validStudyDomain(domain) || handler == nil {
				runtime.studyConfigurationError = &InputError{Code: "STUDY_CONFIGURATION_INVALID", Detail: "installed study handlers must have one valid domain and a non-nil implementation"}
				continue
			}
			copied[domain] = handler
		}
		runtime.Studies = copied
	}
	return runtime
}

func parseInvocation(args []string) (invocation, error) {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h")) {
		return invocation{command: "help"}, nil
	}
	parsed := invocation{command: args[0]}
	if parsed.command == "study" {
		return parseStudyInvocation(args, parsed)
	}
	if parsed.command != "validate" && parsed.command != "preflight" {
		return parsed, &InputError{Code: "COMMAND_UNKNOWN", Detail: "the command is not available in this reference boundary"}
	}
	for index := 1; index < len(args); index++ {
		switch args[index] {
		case "--json":
			if parsed.json {
				return parsed, &InputError{Code: "ARGUMENT_DUPLICATE", Detail: "--json may appear only once"}
			}
			parsed.json = true
		case "--spec":
			if parsed.specPath != "" {
				return parsed, &InputError{Code: "ARGUMENT_DUPLICATE", Detail: "--spec may appear only once"}
			}
			index++
			if index >= len(args) || args[index] == "" {
				return parsed, &InputError{Code: "ARGUMENT_MISSING", Detail: "--spec requires one path or '-'"}
			}
			if args[index] != "-" && strings.HasPrefix(args[index], "-") {
				return parsed, &InputError{Code: "ARGUMENT_MISSING", Detail: "--spec requires one path or '-'; prefix a dash-leading filename with './'"}
			}
			parsed.specPath = args[index]
		default:
			return parsed, &InputError{Code: "ARGUMENT_UNKNOWN", Detail: "only --spec and --json are accepted for this command"}
		}
	}
	if parsed.specPath == "" {
		return parsed, &InputError{Code: "ARGUMENT_MISSING", Detail: "--spec is required"}
	}
	return parsed, nil
}

func parseStudyInvocation(args []string, parsed invocation) (invocation, error) {
	if len(args) < 2 || !validStudyDomain(args[1]) {
		return parsed, &InputError{Code: "ARGUMENT_MISSING", Detail: "study requires one lowercase domain name"}
	}
	parsed.studyDomain = args[1]
	if len(args) == 2 {
		return parsed, nil
	}
	if args[2] == "--json" {
		parsed.json = true
		if len(args) == 3 {
			return parsed, nil
		}
		if len(args) == 4 && args[3] == "--json" {
			return parsed, &InputError{Code: "ARGUMENT_DUPLICATE", Detail: "--json may appear only once"}
		}
		return parsed, &InputError{Code: "ARGUMENT_UNKNOWN", Detail: "study accepts exactly one domain followed by --json"}
	}
	return parsed, &InputError{Code: "ARGUMENT_UNKNOWN", Detail: "study accepts only the exact machine suffix --json"}
}

func validStudyDomain(value string) bool {
	if len(value) < 1 || len(value) > 32 || value[0] < 'a' || value[0] > 'z' || value[len(value)-1] == '-' {
		return false
	}
	for index := 1; index < len(value); index++ {
		current := value[index]
		if current != '-' && (current < 'a' || current > 'z') && (current < '0' || current > '9') {
			return false
		}
		if current == '-' && value[index-1] == '-' {
			return false
		}
	}
	return true
}

func commandName(args []string) string {
	if len(args) == 0 {
		return "help"
	}
	return args[0]
}

func helpEnvelope(studies map[string]StudyHandler) ResponseEnvelope {
	domains := installedStudyDomains(studies)
	message := "Inert validation and preflight commands are available. No study handlers are installed in this composition."
	commands := []string{
		"validate --spec <path|-> [--json]",
		"preflight --spec <path|-> [--json]",
	}
	if len(domains) == 0 {
		commands = append(commands, "study <domain> --json (frozen harness only)")
	} else {
		message = "Inert validation and preflight commands plus installed frozen study machine routes are available: " + strings.Join(domains, ", ") + "."
		for _, domain := range domains {
			commands = append(commands, "study "+domain+" --json (frozen harness only)")
		}
	}
	response := baseEnvelope("help", "AVAILABLE", message, "", ExitOK)
	response.Help = &HelpView{
		Usage:    "countershape <help|validate|preflight|study>",
		Commands: commands,
	}
	return response
}

func installedStudyDomains(studies map[string]StudyHandler) []string {
	domains := make([]string, 0, len(studies))
	for domain, handler := range studies {
		if validStudyDomain(domain) && handler != nil {
			domains = append(domains, domain)
		}
	}
	sort.Strings(domains)
	return domains
}

func humanStudyRefusal(domain string, studies map[string]StudyHandler) ResponseEnvelope {
	handler, installed := studies[domain]
	if !installed || handler == nil {
		err := &InputError{Code: "STUDY_HANDLER_UNAVAILABLE", Detail: "no installed handler owns the requested study domain"}
		response := failureEnvelope("study", err)
		response.ExitCode = ExitRefused
		response.NextAction = "Run `countershape help` and choose one installed study domain. No study execution occurred."
		return response
	}
	err := &InputError{Code: "STUDY_HARNESS_ONLY", Detail: "a study cannot execute from the human-facing command route"}
	response := failureEnvelope("study", err)
	response.ExitCode = ExitRefused
	warning := TrustWarning
	response.Warning = &warning
	response.NextAction = "Use the repository's frozen study harness for the exact `countershape study " + domain + " --json` machine route; do not invoke that execution route interactively."
	return response
}

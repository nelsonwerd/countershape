package app

import (
	"context"
	"io"
	"os"
	"strings"
)

// StudyHandler is the domain-neutral application seam for one frozen study
// route. Domain packages implement the handler; app owns admission, dispatch,
// presentation, and the process-wide success protocol.
type StudyHandler func(StudyRequest) error

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

// Run is the composition shell used by cmd/countershape. U7A deliberately
// installs no candidate or study handler: validate and preflight are inert.
func Run(args []string, runtime Runtime) int {
	runtime = completeRuntime(runtime)
	parsed, err := parseInvocation(args)
	var response ResponseEnvelope
	if err != nil {
		response = failureEnvelope(commandName(args), err)
	} else {
		switch parsed.command {
		case "help":
			response = helpEnvelope()
		case "validate":
			response = validateSourceCommand(parsed.specPath, runtime)
		case "preflight":
			response = preflightSourceCommand(parsed.specPath, runtime)
		case "study":
			if parsed.json {
				return runMachineStudy(parsed.studyDomain, runtime)
			}
			response = humanStudyRefusal(parsed.studyDomain)
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

func helpEnvelope() ResponseEnvelope {
	response := baseEnvelope("help", "AVAILABLE", "U7A validation and preflight commands plus installed frozen study routes are available.", "", ExitOK)
	response.Help = &HelpView{
		Usage: "countershape <help|validate|preflight|study>",
		Commands: []string{
			"validate --spec <path|-> [--json]",
			"preflight --spec <path|-> [--json]",
			"study <domain> --json (frozen harness only)",
		},
	}
	return response
}

func humanStudyRefusal(domain string) ResponseEnvelope {
	err := &InputError{Code: "STUDY_HARNESS_ONLY", Detail: "a study cannot execute from the human-facing command route"}
	response := failureEnvelope("study", err)
	response.ExitCode = ExitRefused
	warning := TrustWarning
	response.Warning = &warning
	response.NextAction = "Use the repository's frozen study harness for the exact `countershape study " + domain + " --json` machine route; do not invoke that execution route interactively."
	return response
}

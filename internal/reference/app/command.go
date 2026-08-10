package app

import (
	"io"
	"os"
	"strings"
)

type Runtime struct {
	Stdin            io.Reader
	Stdout           io.Writer
	Stderr           io.Writer
	WorkingDirectory string
	Columns          int
}

type invocation struct {
	command  string
	specPath string
	json     bool
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
	return runtime
}

func parseInvocation(args []string) (invocation, error) {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h")) {
		return invocation{command: "help"}, nil
	}
	parsed := invocation{command: args[0]}
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

func commandName(args []string) string {
	if len(args) == 0 {
		return "help"
	}
	return args[0]
}

func helpEnvelope() ResponseEnvelope {
	response := baseEnvelope("help", "AVAILABLE", "U7A inert validation and preflight commands are available.", "", ExitOK)
	response.Help = &HelpView{
		Usage: "countershape <help|validate|preflight>",
		Commands: []string{
			"validate --spec <path|-> [--json]",
			"preflight --spec <path|-> [--json]",
		},
	}
	return response
}

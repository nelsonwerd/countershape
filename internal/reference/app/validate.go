package app

import (
	"errors"
	"strings"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	corespec "github.com/nelsonwerd/countershape/internal/spec"
)

func validateSourceCommand(path string, runtime Runtime) ResponseEnvelope {
	input, err := readSourceInput(path, runtime.Stdin, runtime.WorkingDirectory)
	if err != nil {
		return failureEnvelope("validate", err)
	}
	validated, err := parseValidatedSpec(input)
	if err != nil {
		return failureEnvelope("validate", err)
	}
	summary := validated.Summary()
	response := baseEnvelope(
		"validate",
		"SOURCE_VALID",
		"The strict SourceSpec grammar accepted the input. No candidate code was executed.",
		preflightNextAction(input.argument),
		ExitOK,
	)
	response.Source = &summary
	return response
}

func preflightNextAction(inputArgument string) string {
	if inputArgument == "-" {
		return "Save the same SourceSpec bytes to a regular file, then run `countershape preflight --spec <file>`; stdin bytes are not retained."
	}
	return "Run `countershape preflight --spec " + quoteShellArgument(inputArgument) + "` to inspect execution prerequisites and declared cost ceilings."
}

func quoteShellArgument(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func failureEnvelope(command string, err error) ResponseEnvelope {
	failure, exitCode := failureView(err)
	next := "Fix the refused input, then rerun the same command. No candidate code was executed."
	if exitCode == ExitUsage {
		next = "Run `countershape help` and retry with one documented command shape."
	}
	response := baseEnvelope(command, "REFUSED", "Countershape refused the request before execution.", next, exitCode)
	response.Failure = &failure
	return response
}

func failureView(err error) (FailureView, int) {
	var input *InputError
	if errors.As(err, &input) {
		exitCode := ExitInput
		if strings.HasPrefix(input.Code, "ARGUMENT_") || strings.HasPrefix(input.Code, "COMMAND_") {
			exitCode = ExitUsage
		}
		return FailureView{Code: input.Code, Detail: input.Detail}, exitCode
	}
	var source *corespec.Error
	if errors.As(err, &source) {
		return FailureView{Code: source.Code, Path: source.Path, Detail: source.Detail}, ExitRefused
	}
	var canonical *canon.Error
	if errors.As(err, &canonical) {
		var offset *int
		if canonical.Offset != canon.UnknownOffset {
			value := canonical.Offset
			offset = &value
		}
		return FailureView{Code: string(canonical.Code), Offset: offset, Detail: canonical.Detail}, ExitRefused
	}
	var semantic *domain.Error
	if errors.As(err, &semantic) {
		return FailureView{Code: semantic.Code, Detail: semantic.Detail}, ExitRefused
	}
	var presentation *presentationError
	if errors.As(err, &presentation) {
		return FailureView{Code: "SPEC_PRESENTATION_INVARIANT", Detail: "validated source presentation failed closed"}, ExitInternal
	}
	return FailureView{Code: "INTERNAL_FAILURE", Detail: "an unexpected internal error occurred"}, ExitInternal
}

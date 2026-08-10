package app

func preflightSourceCommand(path string, runtime Runtime) ResponseEnvelope {
	input, err := readSourceInput(path, runtime.Stdin, runtime.WorkingDirectory)
	if err != nil {
		return failureEnvelope("preflight", err)
	}
	validated, err := parseValidatedSpec(input)
	if err != nil {
		return failureEnvelope("preflight", err)
	}
	summary := validated.Summary()
	warning := TrustWarning
	response := baseEnvelope(
		"preflight",
		"SOURCE_VALID_DEPENDENCIES_UNRESOLVED",
		"The source and declared ceilings are valid. This foundation does not resolve projection, candidate, or runtime authority and cannot authorize execution.",
		"Resolve the exact projection binding, committed candidate set, and executable tool identities before starting a new study attempt.",
		ExitOK,
	)
	response.Warning = &warning
	response.Source = &summary
	response.Preflight = &PreflightView{
		ExecutionStarted: false, ExecutionAuthorized: false,
		SourceAuthority: "STRICT_SOURCE_SPEC_VALIDATED", ProjectionAuthority: "UNRESOLVED",
		CandidateAuthority: "UNRESOLVED", RuntimeAuthority: "UNRESOLVED",
		BudgetAuthority: "SOURCE_OVERRIDES_ON_CORE_DEFAULTS", Budgets: summary.Budgets,
	}
	return response
}

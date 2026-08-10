package app

const (
	CLISchemaVersion = "countershape-cli/v1"

	TrustWarning = "Countershape will run trusted repository code with your user permissions and host network access. The temporary directory is for repeatability, not security isolation."

	ExitOK       = 0
	ExitUsage    = 2
	ExitInput    = 3
	ExitRefused  = 4
	ExitInternal = 70
)

type FailureView struct {
	Code   string `json:"code"`
	Path   string `json:"path"`
	Offset *int   `json:"offset"`
	Detail string `json:"detail"`
}

type PreflightView struct {
	ExecutionStarted    bool       `json:"execution_started"`
	ExecutionAuthorized bool       `json:"execution_authorized"`
	SourceAuthority     string     `json:"source_authority"`
	ProjectionAuthority string     `json:"projection_authority"`
	CandidateAuthority  string     `json:"candidate_authority"`
	RuntimeAuthority    string     `json:"runtime_authority"`
	BudgetAuthority     string     `json:"budget_authority"`
	Budgets             BudgetView `json:"budgets"`
}

type HelpView struct {
	Usage    string   `json:"usage"`
	Commands []string `json:"commands"`
}

// ResponseEnvelope is presentation transport only. It carries facts supplied
// by stricter constructors; it does not classify, compare, digest, or otherwise
// manufacture Countershape semantic authority.
type ResponseEnvelope struct {
	SchemaVersion string         `json:"schema_version"`
	Command       string         `json:"command"`
	Status        string         `json:"status"`
	ExitCode      int            `json:"exit_code"`
	Message       string         `json:"message"`
	Warning       *string        `json:"warning"`
	NextAction    string         `json:"next_action"`
	Source        *SpecSummary   `json:"source"`
	Preflight     *PreflightView `json:"preflight"`
	Help          *HelpView      `json:"help"`
	Failure       *FailureView   `json:"failure"`
}

func baseEnvelope(command, status, message, nextAction string, exitCode int) ResponseEnvelope {
	return ResponseEnvelope{
		SchemaVersion: CLISchemaVersion,
		Command:       command,
		Status:        status,
		ExitCode:      exitCode,
		Message:       message,
		NextAction:    nextAction,
	}
}

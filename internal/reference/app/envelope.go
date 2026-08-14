package app

const (
	CLISchemaVersion               = "countershape-cli/v1"
	StudyDomainResultSchemaVersion = "countershape/u7-study-domain-result/v1"

	TrustWarning = "Countershape will run trusted repository code with your user permissions and host network access. The temporary directory is for repeatability, not security isolation."

	ExitOK       = 0
	ExitUsage    = 2
	ExitInput    = 3
	ExitRefused  = 4
	ExitInternal = 70
)

// StudyDomainResult is deliberately separate from ResponseEnvelope. The
// frozen U7 harness admits exactly these four fields and no presentation or
// application-envelope metadata.
type StudyDomainResult struct {
	SchemaVersion string `json:"schema_version"`
	Domain        string `json:"domain"`
	Ordinal       int    `json:"ordinal"`
	Status        string `json:"status"`
}

// EvidenceFile is a caller-supplied raw-byte identity for one terminal study
// artifact. It carries no parsed JSON, semantic classification, or authority.
type EvidenceFile struct {
	Path   string
	Bytes  int64
	SHA256 string
}

// EvidenceManifest is the complete expected private tree beneath one admitted
// evidence workspace. The workspace verifies this transport shape; the owning
// study remains responsible for every semantic meaning of the bytes.
type EvidenceManifest struct {
	Directories []string
	Files       []EvidenceFile
}

// StudyEvidenceEntry is one inert terminal artifact snapshot. Its fields are
// private so consumers cannot relabel or mutate the captured bytes in place;
// every accessor returns a scalar or a defensive copy. The entry carries no
// parsed payload or study-semantic authority.
type StudyEvidenceEntry struct {
	path   string
	mode   string
	bytes  int64
	sha256 string
	exact  []byte
}

func (entry StudyEvidenceEntry) Path() string   { return entry.path }
func (entry StudyEvidenceEntry) Mode() string   { return entry.mode }
func (entry StudyEvidenceEntry) Bytes() int64   { return entry.bytes }
func (entry StudyEvidenceEntry) SHA256() string { return entry.sha256 }

func (entry StudyEvidenceEntry) ExactBytes() []byte {
	return append([]byte(nil), entry.exact...)
}

// StudyEvidenceSnapshot is an exact private-tree byte projection produced by
// EvidenceWorkspace after a successful domain handler returns. It is roster
// agnostic: the consuming reproduction layer, not app, owns any domain-specific
// path, count, wrapper, or semantic checks.
type StudyEvidenceSnapshot struct {
	directories []string
	files       []StudyEvidenceEntry
	totalBytes  int64
}

func (snapshot StudyEvidenceSnapshot) Directories() []string {
	return append([]string(nil), snapshot.directories...)
}

func (snapshot StudyEvidenceSnapshot) Files() []StudyEvidenceEntry {
	result := make([]StudyEvidenceEntry, len(snapshot.files))
	for index, entry := range snapshot.files {
		result[index] = entry
		result[index].exact = append([]byte(nil), entry.exact...)
	}
	return result
}

func (snapshot StudyEvidenceSnapshot) TotalBytes() int64 { return snapshot.totalBytes }

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

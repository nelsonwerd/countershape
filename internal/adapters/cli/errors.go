// Package cli owns Countershape's typed CLI stimulus, capture, and projection
// facts. It deliberately performs no process or filesystem I/O.
package cli

import (
	"fmt"

	"github.com/nelsonwerd/countershape/internal/domain"
)

// Refusal is a construction-time failure. Code is stable machine data; Detail
// is explanatory and never participates in semantic identity.
type Refusal struct {
	Code   string
	Detail string
}

func (r *Refusal) Error() string {
	if r == nil {
		return "<nil>"
	}
	if r.Detail == "" {
		return r.Code
	}
	return fmt.Sprintf("%s: %s", r.Code, r.Detail)
}

func refuse(code, detail string) error {
	return &Refusal{Code: code, Detail: detail}
}

const (
	CodeInvalidStimulus         = "CLI_INVALID_STIMULUS"
	CodeInvalidExecutable       = "CLI_INVALID_EXECUTABLE"
	CodeInvalidArgv             = "CLI_INVALID_ARGV"
	CodeInvalidStdin            = "CLI_INVALID_STDIN"
	CodeInvalidEnvironment      = "CLI_INVALID_ENVIRONMENT"
	CodeInvalidFixture          = "CLI_INVALID_FIXTURE"
	CodeFixtureOrder            = "CLI_FIXTURE_ORDER_INVALID"
	CodeFixturePath             = "CLI_FIXTURE_PATH_INVALID"
	CodeFixtureCollision        = "CLI_FIXTURE_PATH_COLLISION"
	CodeInputLimit              = "CLI_INPUT_LIMIT"
	CodeInvalidCWDPolicy        = "CLI_INVALID_CWD_POLICY"
	CodeInvalidCapturePolicy    = "CLI_INVALID_CAPTURE_POLICY"
	CodeCaptureLineageMismatch  = "CLI_CAPTURE_LINEAGE_MISMATCH"
	CodeCaptureEvidenceInvalid  = "CLI_CAPTURE_EVIDENCE_INVALID"
	CodeInvalidProjection       = "CLI_INVALID_PROJECTION_DEFINITION"
	CodeProjectionControl       = "CLI_PROJECTION_CONTROL_INELIGIBLE"
	CodeProjectionChannel       = "CLI_PROJECTION_CHANNEL_REQUIRED"
	CodeProjectionTruncated     = "CLI_PROJECTION_CHANNEL_TRUNCATED"
	CodeProjectionUTF8          = "CLI_PROJECTION_INVALID_UTF8"
	CodeProjectionJSON          = "CLI_PROJECTION_INVALID_STRICT_JSON"
	CodeProjectionJSONRoot      = "CLI_PROJECTION_JSON_ROOT_REQUIRED"
	CodeProjectionJSONFieldType = "CLI_PROJECTION_JSON_FIELD_TYPE"
	CodeProjectionResourceLimit = "CLI_PROJECTION_RESOURCE_LIMIT"
	CodeProjectionInternal      = "CLI_PROJECTION_INTERNAL_REFUSAL"
)

// ProjectionRejection is the only unsuccessful projection result. It is a
// typed control fact, never an empty or fallback behavior value.
type ProjectionRejection struct {
	Code                    string
	Operation               string
	Channel                 CLIChannel
	Field                   CLIFieldID
	Detail                  string
	digest                  domain.Digest
	canonicalBytes          []byte
	observationDigest       domain.Digest
	adapterDefinitionDigest domain.Digest
	definitionDigest        domain.Digest
	transcript              []CLIProjectionTraceEntry
}

func (r *ProjectionRejection) Error() string {
	if r == nil {
		return "<nil>"
	}
	location := r.Operation
	if r.Field != "" {
		location += "/" + string(r.Field)
	}
	if r.Channel != "" {
		location += "/" + string(r.Channel)
	}
	if r.Detail == "" {
		return fmt.Sprintf("%s: %s", r.Code, location)
	}
	return fmt.Sprintf("%s: %s: %s", r.Code, location, r.Detail)
}

func (r *ProjectionRejection) EvidenceDigest() domain.Digest {
	if r == nil {
		return ""
	}
	return r.digest
}

func (r *ProjectionRejection) CanonicalBytes() []byte {
	if r == nil {
		return nil
	}
	return append([]byte(nil), r.canonicalBytes...)
}

func (r *ProjectionRejection) ObservationDigest() domain.Digest {
	if r == nil {
		return ""
	}
	return r.observationDigest
}

func (r *ProjectionRejection) DefinitionDigest() domain.Digest {
	if r == nil {
		return ""
	}
	return r.definitionDigest
}

func (r *ProjectionRejection) Transcript() []CLIProjectionTraceEntry {
	if r == nil {
		return nil
	}
	return cloneTranscript(r.transcript)
}

package model

import (
	"bytes"
	"slices"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/runnerprofile"
)

const (
	RuntimeFamilyNodeV1         = "NODE"
	DeclaredSourceScopeV1       = "DECLARED_SOURCE_PROFILE_NOT_EXECUTION_EVIDENCE"
	ContractSourceProfileDomain = "ContractSourceProfile"
	httpPortableStartProfileV1  = "NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1"
)

type sourceProfileWire struct {
	RuntimeFamily     string `json:"runtime_family"`
	SemanticProfile   string `json:"semantic_profile"`
	AdapterDomain     string `json:"adapter_domain"`
	LaunchProfile     string `json:"launch_profile"`
	SubjectEntrypoint string `json:"subject_entrypoint"`
	StartProfile      string `json:"start_profile"`
	Scope             string `json:"scope"`
}

// SourceProfile is the seven-member declared emitter profile. It is not the
// portable projection profile and contains no measured host/runtime fact.
type SourceProfile struct {
	adapter    domain.AdapterDomain
	entrypoint string
	start      string
	digest     domain.Digest
	canonical  []byte
}

func NewSourceProfile(source contractsource.PortableSource) (SourceProfile, error) {
	if !source.Valid() {
		return SourceProfile{}, refuse("INVALID_SOURCE_PROFILE", "exact PortableSource is required", nil)
	}
	reparsed, err := contractsource.Parse(source.CanonicalBytes())
	if err != nil || reparsed.Digest() != source.Digest() || !bytes.Equal(reparsed.CanonicalBytes(), source.CanonicalBytes()) {
		return SourceProfile{}, refuse("INVALID_SOURCE_PROFILE", "PortableSource did not reparse exactly", err)
	}
	adapter := reparsed.Adapter()
	start := reparsed.StartProfile()
	if (adapter == domain.AdapterCLI && start != contractsource.CLIStartProfileV1) ||
		(adapter == domain.AdapterHTTP && start != httpPortableStartProfileV1) {
		return SourceProfile{}, refuse("INVALID_SOURCE_PROFILE", "adapter and start profile are not one portable pair", nil)
	}
	runner, err := contractsource.RequiredRunnerDigest(adapter)
	plan := reparsed.Plan()
	if err != nil || plan.Adapter().RunnerDigest != runner ||
		!slices.Equal(plan.StartArgv(), []string{"node", reparsed.Entrypoint()}) {
		return SourceProfile{}, refuse("INVALID_SOURCE_PROFILE", "source plan is not the exact logical Node runner", err)
	}
	wire := sourceProfileWire{
		RuntimeFamily: RuntimeFamilyNodeV1, SemanticProfile: contractsource.SourceProfileV1,
		AdapterDomain: string(adapter), LaunchProfile: contractsource.LaunchProfileV1,
		SubjectEntrypoint: reparsed.Entrypoint(), StartProfile: start, Scope: DeclaredSourceScopeV1,
	}
	digestRaw, canonical, err := canon.DigestTyped(ContractSourceProfileDomain, wire)
	if err != nil {
		return SourceProfile{}, refuse("INVALID_SOURCE_PROFILE", "declared source profile could not be canonicalized", err)
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return SourceProfile{}, refuse("INVALID_SOURCE_PROFILE", "declared source profile digest is invalid", err)
	}
	return SourceProfile{
		adapter: adapter, entrypoint: reparsed.Entrypoint(), start: start,
		digest: digest, canonical: canonical,
	}, nil
}

func (p SourceProfile) Valid() bool {
	if (p.adapter != domain.AdapterCLI && p.adapter != domain.AdapterHTTP) || p.entrypoint == "" ||
		p.start == "" || !p.digest.Valid() || len(p.canonical) == 0 {
		return false
	}
	expectedStart := contractsource.CLIStartProfileV1
	if p.adapter == domain.AdapterHTTP {
		expectedStart = httpPortableStartProfileV1
	}
	if p.start != expectedStart || !runnerprofile.ValidRepositoryNodeEntrypoint(p.entrypoint) {
		return false
	}
	wire := sourceProfileWire{
		RuntimeFamily: RuntimeFamilyNodeV1, SemanticProfile: contractsource.SourceProfileV1,
		AdapterDomain: string(p.adapter), LaunchProfile: contractsource.LaunchProfileV1,
		SubjectEntrypoint: p.entrypoint, StartProfile: p.start, Scope: DeclaredSourceScopeV1,
	}
	digest, canonical, err := canon.DigestTyped(ContractSourceProfileDomain, wire)
	return err == nil && digest.String() == p.digest.String() && bytes.Equal(canonical, p.canonical)
}

func (p SourceProfile) ValidFor(source contractsource.PortableSource) bool {
	rebuilt, err := NewSourceProfile(source)
	return err == nil && p.Valid() && rebuilt.digest == p.digest && bytes.Equal(rebuilt.canonical, p.canonical)
}

func (p SourceProfile) Digest() domain.Digest               { return p.digest }
func (p SourceProfile) CanonicalBytes() []byte              { return append([]byte(nil), p.canonical...) }
func (p SourceProfile) AdapterDomain() domain.AdapterDomain { return p.adapter }
func (p SourceProfile) SubjectEntrypoint() string           { return p.entrypoint }
func (p SourceProfile) StartProfile() string                { return p.start }

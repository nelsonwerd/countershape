package projectiontranslate

import (
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
)

const (
	portableProfileRosterV1EntryCount           = 128
	portableProfileRosterV1Digest               = "sha256:aec21f3f76f9384a09ab3f71cea3d1def86fb8ef725322f35af296aa8e1560b7"
	portableExpectationDomainRosterV1EntryCount = 128
	portableExpectationDomainRosterV1Digest     = "sha256:d0951b6681fc1893691db83accfb5aeaac2a500cba9d4814308518aeff4566c7"
	portableModeSemanticsV1Digest               = "sha256:9b51d42ab58078eac4d652fbaeb925d614803d5926bf63e7c59fba4134f57cae"
)

const (
	portableTupleIdentityRuleV1 = "FIELD_ID_PROFILE_ORDER_PLUS_TAGGED_EXACT_PAYLOAD_BYTES_V1"
	portableSelectionRuleV1     = "SELECTED_ONLY_COMPLETE_TUPLE_RESTRICTION_NO_UNSELECTED_CONTEXT_V1"
	portableProofRuleV1         = "COMPLETE_CONFIRMED_ROSTER_VERIFY_ALL_PROOFS_BEFORE_TRANSLATION_V1"
)

type portableProfileRosterEntryIdentity struct {
	AdapterDomain           string `json:"adapter_domain"`
	ProjectionBindingDigest string `json:"projection_binding_digest"`
	ProjectionBindingBase64 string `json:"projection_binding_base64"`
	ProfileDigest           string `json:"profile_digest"`
	ProfileBase64           string `json:"profile_base64"`
}

type portableProfileRosterIdentity struct {
	SchemaVersion string                               `json:"schema_version"`
	Kind          string                               `json:"kind"`
	Entries       []portableProfileRosterEntryIdentity `json:"entries"`
}

type portableExpectationDomainRosterEntryIdentity struct {
	ProfileDigest     string `json:"profile_digest"`
	ExpectationDigest string `json:"expectation_domain_digest"`
	ExpectationBase64 string `json:"expectation_domain_base64"`
}

type portableExpectationDomainRosterIdentity struct {
	SchemaVersion        string                                         `json:"schema_version"`
	Kind                 string                                         `json:"kind"`
	ChoiceProjectionMode string                                         `json:"choice_projection_mode"`
	Semantics            string                                         `json:"semantics"`
	Entries              []portableExpectationDomainRosterEntryIdentity `json:"entries"`
}

type portableModeSemanticsV1Identity struct {
	SchemaVersion                   string   `json:"schema_version"`
	Kind                            string   `json:"kind"`
	ChoiceProjectionMode            string   `json:"choice_projection_mode"`
	PortableTags                    []string `json:"portable_tags"`
	MaxStringBytes                  int      `json:"max_string_bytes"`
	MaxBytesValueBytes              int      `json:"max_bytes_value_bytes"`
	MaxListMembers                  int      `json:"max_list_members"`
	MaxListMemberBytes              int      `json:"max_list_member_bytes"`
	MaxListAggregateBytes           int      `json:"max_list_aggregate_bytes"`
	MaxListCanonicalBytes           int      `json:"max_list_canonical_bytes"`
	MaxCanonicalJSONBytes           int      `json:"max_canonical_json_bytes"`
	MaxTupleFields                  int      `json:"max_tuple_fields"`
	MaxTupleRetainedBytes           int      `json:"max_tuple_retained_bytes"`
	MaxTupleEncodedBytes            int      `json:"max_tuple_encoded_bytes"`
	CompatibilityFieldOverheadBytes int      `json:"compatibility_field_overhead_bytes"`
	TupleIdentityRule               string   `json:"tuple_identity_rule"`
	SelectionRule                   string   `json:"selection_rule"`
	ProofRule                       string   `json:"proof_rule"`
	ProfileRosterDigest             string   `json:"profile_roster_digest"`
	ExpectationDomainRosterDigest   string   `json:"expectation_domain_roster_digest"`
}

var (
	portableProfileRosterV1Once sync.Once
	portableProfileRosterV1Err  error
)

// verifyPortableProfileRosterV1 freezes the interpretation of the existing
// portable Choicepoint mode. A descriptor, translator-version, binding, or
// profile-wire drift must introduce a new mode; it cannot reinterpret sealed
// ADAPTER_BOUND_PORTABLE_FIELDS_V1 history in place.
func verifyPortableProfileRosterV1() error {
	portableProfileRosterV1Once.Do(func() {
		digest, count, err := derivePortableProfileRosterV1()
		if err != nil {
			portableProfileRosterV1Err = err
			return
		}
		if count != portableProfileRosterV1EntryCount || digest.String() != portableProfileRosterV1Digest {
			portableProfileRosterV1Err = refuse(
				CodeProfileMismatch,
				"",
				fmt.Sprintf("portable profile roster v1 drifted: count=%d digest=%s", count, digest.String()),
			)
			return
		}
		expectationDigest, expectationCount, err := derivePortableExpectationDomainRosterV1()
		if err != nil {
			portableProfileRosterV1Err = err
			return
		}
		if expectationCount != portableExpectationDomainRosterV1EntryCount || expectationDigest.String() != portableExpectationDomainRosterV1Digest {
			portableProfileRosterV1Err = refuse(
				CodeProfileMismatch,
				"",
				fmt.Sprintf("portable expectation-domain roster v1 drifted: count=%d digest=%s", expectationCount, expectationDigest.String()),
			)
			return
		}
		semanticsDigest, err := derivePortableModeSemanticsV1()
		if err != nil {
			portableProfileRosterV1Err = err
			return
		}
		if semanticsDigest.String() != portableModeSemanticsV1Digest {
			portableProfileRosterV1Err = refuse(
				CodeProfileMismatch,
				"",
				fmt.Sprintf("portable mode semantics v1 drifted: digest=%s", semanticsDigest.String()),
			)
		}
	})
	return portableProfileRosterV1Err
}

func derivePortableProfileRosterV1() (domain.Digest, int, error) {
	resolvedProfiles, err := portableResolvedProfilesV1()
	if err != nil {
		return domain.Digest(""), 0, err
	}
	entries := make([]portableProfileRosterEntryIdentity, 0, portableProfileRosterV1EntryCount)
	seenBindings := make(map[string]struct{}, portableProfileRosterV1EntryCount)
	seenProfiles := make(map[string]struct{}, portableProfileRosterV1EntryCount)
	for _, resolved := range resolvedProfiles {
		entry, err := portableProfileRosterEntry(resolved)
		if err != nil {
			return domain.Digest(""), 0, err
		}
		if _, duplicate := seenBindings[entry.ProjectionBindingDigest]; duplicate {
			return domain.Digest(""), 0, refuse(CodeProfileMismatch, "", "CLI portable profile roster v1 repeats a binding")
		}
		if _, duplicate := seenProfiles[entry.ProfileDigest]; duplicate {
			return domain.Digest(""), 0, refuse(CodeProfileMismatch, "", "CLI portable profile roster v1 repeats a profile")
		}
		seenBindings[entry.ProjectionBindingDigest] = struct{}{}
		seenProfiles[entry.ProfileDigest] = struct{}{}
		entries = append(entries, entry)
	}
	if len(entries) != portableProfileRosterV1EntryCount {
		return domain.Digest(""), len(entries), refuse(CodeProfileMismatch, "", "portable profile roster v1 cardinality drifted")
	}

	digestRaw, _, err := canon.DigestTyped("PortableProjectionProfileRosterV1", portableProfileRosterIdentity{
		SchemaVersion: domain.SchemaVersion,
		Kind:          "PortableProjectionProfileRosterV1",
		Entries:       entries,
	})
	if err != nil {
		return domain.Digest(""), 0, refuse(CodeProfileMismatch, "", "portable profile roster v1 identity failed")
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return domain.Digest(""), 0, refuse(CodeProfileMismatch, "", "portable profile roster v1 digest failed")
	}
	return digest, len(entries), nil
}

func portableResolvedProfilesV1() ([]Resolved, error) {
	registry := cli.CLIFieldRegistry()
	if len(registry) != 7 {
		return nil, refuse(CodeProfileMismatch, "", "CLI portable profile roster v1 no longer has seven fields")
	}
	resolvedProfiles := make([]Resolved, 0, portableProfileRosterV1EntryCount)
	for mask := 1; mask < 1<<len(registry); mask++ {
		fields := make([]cli.CLIFieldID, 0, len(registry))
		for index, descriptor := range registry {
			if mask&(1<<index) != 0 {
				fields = append(fields, descriptor.ID())
			}
		}
		definition, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: fields})
		if err != nil {
			return nil, refuse(CodeProfileMismatch, "", "CLI portable profile roster v1 definition failed")
		}
		resolved, err := newCLIResolvedProfile(definition, registry)
		if err != nil {
			return nil, err
		}
		resolvedProfiles = append(resolvedProfiles, resolved)
	}
	httpDefinition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		return nil, refuse(CodeProfileMismatch, "", "HTTP portable profile roster v1 definition failed")
	}
	httpResolved, err := resolveHTTP(httpDefinition.Binding())
	if err != nil {
		return nil, err
	}
	resolvedProfiles = append(resolvedProfiles, httpResolved)
	if len(resolvedProfiles) != portableProfileRosterV1EntryCount {
		return nil, refuse(CodeProfileMismatch, "", "portable profile roster v1 cardinality drifted")
	}
	return resolvedProfiles, nil
}

func derivePortableExpectationDomainRosterV1() (domain.Digest, int, error) {
	resolvedProfiles, err := portableResolvedProfilesV1()
	if err != nil {
		return domain.Digest(""), 0, err
	}
	entries := make([]portableExpectationDomainRosterEntryIdentity, len(resolvedProfiles))
	seen := make(map[string]struct{}, len(resolvedProfiles))
	for index, resolved := range resolvedProfiles {
		expectation, expectationErr := newExpectationDomain(resolved)
		if expectationErr != nil {
			return domain.Digest(""), 0, expectationErr
		}
		key := expectation.Digest().String()
		if _, duplicate := seen[key]; duplicate {
			return domain.Digest(""), 0, refuse(CodeProfileMismatch, "", "portable expectation-domain roster v1 repeats a domain")
		}
		seen[key] = struct{}{}
		entries[index] = portableExpectationDomainRosterEntryIdentity{
			ProfileDigest: resolved.Profile().Digest().String(), ExpectationDigest: key,
			ExpectationBase64: base64.StdEncoding.EncodeToString(expectation.CanonicalBytes()),
		}
	}
	digestRaw, _, err := canon.DigestTyped("PortableExpectationDomainRosterV1", portableExpectationDomainRosterIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "PortableExpectationDomainRosterV1",
		ChoiceProjectionMode: PortableChoiceModeV1, Semantics: portableExpectationSemanticsV1, Entries: entries,
	})
	if err != nil {
		return domain.Digest(""), 0, refuse(CodeProfileMismatch, "", "portable expectation-domain roster v1 identity failed")
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return domain.Digest(""), 0, refuse(CodeProfileMismatch, "", "portable expectation-domain roster v1 digest failed")
	}
	return digest, len(entries), nil
}

func derivePortableModeSemanticsV1() (domain.Digest, error) {
	digestRaw, _, err := canon.DigestTyped("PortableModeSemanticsV1", portableModeSemanticsV1Identity{
		SchemaVersion: domain.SchemaVersion, Kind: "PortableModeSemanticsV1",
		ChoiceProjectionMode: PortableChoiceModeV1,
		PortableTags: []string{
			string(portablevalue.TagMissing), string(portablevalue.TagNull), string(portablevalue.TagBoolean),
			string(portablevalue.TagInteger), string(portablevalue.TagString), string(portablevalue.TagBytes),
			string(portablevalue.TagOrderedStringList), string(portablevalue.TagCanonicalJSON),
		},
		MaxStringBytes: portablevalue.MaxStringBytes, MaxBytesValueBytes: portablevalue.MaxBytesValueBytes,
		MaxListMembers: portablevalue.MaxListMembers, MaxListMemberBytes: portablevalue.MaxListMemberBytes,
		MaxListAggregateBytes: portablevalue.MaxListAggregateBytes, MaxListCanonicalBytes: portablevalue.MaxListCanonicalBytes,
		MaxCanonicalJSONBytes: portablevalue.MaxCanonicalJSONBytes, MaxTupleFields: portablevalue.MaxTupleFields,
		MaxTupleRetainedBytes: portablevalue.MaxTupleRetainedBytes, MaxTupleEncodedBytes: portablevalue.MaxTupleEncodedBytes,
		CompatibilityFieldOverheadBytes: portablevalue.CompatibilityFieldOverheadBytes,
		TupleIdentityRule:               portableTupleIdentityRuleV1, SelectionRule: portableSelectionRuleV1,
		ProofRule: portableProofRuleV1, ProfileRosterDigest: portableProfileRosterV1Digest,
		ExpectationDomainRosterDigest: portableExpectationDomainRosterV1Digest,
	})
	if err != nil {
		return domain.Digest(""), refuse(CodeProfileMismatch, "", "portable mode semantics v1 identity failed")
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return domain.Digest(""), refuse(CodeProfileMismatch, "", "portable mode semantics v1 digest failed")
	}
	return digest, nil
}

func portableProfileRosterEntry(resolved Resolved) (portableProfileRosterEntryIdentity, error) {
	if !resolved.Valid() {
		return portableProfileRosterEntryIdentity{}, refuse(CodeProfileMismatch, "", "portable profile roster v1 contains an invalid resolution")
	}
	profile := resolved.Profile()
	binding := profile.Binding()
	return portableProfileRosterEntryIdentity{
		AdapterDomain:           string(profile.AdapterDomain()),
		ProjectionBindingDigest: binding.Digest().String(),
		ProjectionBindingBase64: base64.StdEncoding.EncodeToString(binding.CanonicalBytes()),
		ProfileDigest:           profile.Digest().String(),
		ProfileBase64:           base64.StdEncoding.EncodeToString(profile.CanonicalBytes()),
	}, nil
}

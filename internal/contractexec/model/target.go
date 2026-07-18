package model

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	terminalResidueProfileV1 = "EXACT_TERMINAL_RESIDUE_HEAD_V1"
	treeAuthorityV1          = "GIT_PIN_INSPECT_MATERIALIZE_V1"
	executionRootScopeV1     = "PRIVATE_PINNED_MATERIALIZATION_ONLY_V1"
	attemptAllocationV1      = "PRIVATE_FRESH_ROOT_V1"
	attemptMarkerOrderingV1  = "DURABLE_BEFORE_SPAWN"
	bootSessionProfileV1     = "DARWIN_KERN_BOOTTIME_V1"
	runtimeAuthorityV1       = "ADMITTED_NODE_PROCESS_EXEC_PATH_V1"
	runtimeChildResolutionV1 = "PROCESS_EXEC_PATH_EQUALS_ADMITTED_RUNTIME_V1"
)

type TerminalResidueBinding struct {
	StudyID           string
	HeadRevision      int64
	HeadDigest        domain.Digest
	LineageRootDigest domain.Digest
}

type TreeBinding struct {
	ObjectFormat                  string
	CommitOID                     string
	TreeOID                       string
	TreeIdentityDigest            domain.Digest
	PortableTreeDigest            domain.Digest
	MaterializationPolicyDigest   domain.Digest
	MaterializationManifestDigest domain.Digest
}

type AttemptBinding struct {
	ArtifactDigest domain.Digest
	InstanceNonce  string
}

type BootSessionBinding struct {
	IdentityDigest domain.Digest
}

type RuntimeBinding struct {
	AdmittedExecutablePath  string
	MeasuredProcessExecPath string
	Version                 string
	Major                   int64
	Platform                string
	Architecture            string
	ExecutableBytesDigest   domain.Digest
	ExecutableMode          string
	ExecutableByteCount     int64
	ProbeProgramDigest      domain.Digest
}

type TargetInput struct {
	ContractBundleDigest domain.Digest
	TerminalResidue      TerminalResidueBinding
	Tree                 TreeBinding
	Attempt              AttemptBinding
	BootSession          BootSessionBinding
	Runtime              RuntimeBinding
}

type ContractExecutionTarget struct {
	input     TargetInput
	digest    domain.Digest
	canonical []byte
}

func NewContractExecutionTarget(input TargetInput) (ContractExecutionTarget, error) {
	if err := validateTargetInput(input); err != nil {
		return ContractExecutionTarget{}, err
	}
	wire := map[string]any{
		"schema_version":         domain.SchemaVersion,
		"kind":                   TargetKind,
		"target_version":         TargetVersionV1,
		"publication_scope":      TargetPublicationScopeV1,
		"contract_bundle_digest": input.ContractBundleDigest.String(),
		"terminal_residue_binding": map[string]any{
			"profile":             terminalResidueProfileV1,
			"study_id":            input.TerminalResidue.StudyID,
			"head_revision":       input.TerminalResidue.HeadRevision,
			"head_digest":         input.TerminalResidue.HeadDigest.String(),
			"lineage_root_digest": input.TerminalResidue.LineageRootDigest.String(),
			"stage":               "RESIDUE",
			"current_kind":        "ContractBundle",
			"current_digest":      input.ContractBundleDigest.String(),
		},
		"tree_binding": map[string]any{
			"authority":                       treeAuthorityV1,
			"object_format":                   input.Tree.ObjectFormat,
			"commit_oid":                      input.Tree.CommitOID,
			"tree_oid":                        input.Tree.TreeOID,
			"tree_identity_digest":            input.Tree.TreeIdentityDigest.String(),
			"portable_tree_digest":            input.Tree.PortableTreeDigest.String(),
			"materialization_policy_digest":   input.Tree.MaterializationPolicyDigest.String(),
			"materialization_manifest_digest": input.Tree.MaterializationManifestDigest.String(),
			"execution_root_scope":            executionRootScopeV1,
		},
		"attempt_binding": map[string]any{
			"purpose":                 "CONFORMANCE",
			"attempt_artifact_digest": input.Attempt.ArtifactDigest.String(),
			"instance_nonce":          input.Attempt.InstanceNonce,
			"allocation_profile":      attemptAllocationV1,
			"marker_ordering":         attemptMarkerOrderingV1,
		},
		"boot_session_binding": map[string]any{
			"profile":         bootSessionProfileV1,
			"identity_digest": input.BootSession.IdentityDigest.String(),
		},
		"runtime_binding": map[string]any{
			"authority":                  runtimeAuthorityV1,
			"name":                       "node",
			"admitted_executable_path":   input.Runtime.AdmittedExecutablePath,
			"measured_process_exec_path": input.Runtime.MeasuredProcessExecPath,
			"version":                    input.Runtime.Version,
			"major":                      input.Runtime.Major,
			"platform":                   input.Runtime.Platform,
			"architecture":               input.Runtime.Architecture,
			"executable_identity": map[string]any{
				"bytes_digest": input.Runtime.ExecutableBytesDigest.String(),
				"mode":         input.Runtime.ExecutableMode,
				"byte_count":   input.Runtime.ExecutableByteCount,
			},
			"probe_program_digest": input.Runtime.ProbeProgramDigest.String(),
			"child_resolution":     runtimeChildResolutionV1,
		},
	}
	exact, digest, err := canonicalObject(TargetKind, wire)
	if err != nil {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "target body could not be built", err)
	}
	return ContractExecutionTarget{input: cloneTargetInput(input), digest: digest, canonical: exact}, nil
}

func validateTargetInput(input TargetInput) error {
	residue := input.TerminalResidue
	tree := input.Tree
	attempt := input.Attempt
	runtime := input.Runtime
	if !input.ContractBundleDigest.Valid() || !studyIDPattern.MatchString(residue.StudyID) ||
		residue.HeadRevision < 1 || !residue.HeadDigest.Valid() || !residue.LineageRootDigest.Valid() {
		return refuse(CodeInvalidTarget, "terminal residue binding is incomplete", nil)
	}
	validOIDs := (tree.ObjectFormat == "sha1" && sha1OIDPattern.MatchString(tree.CommitOID) && sha1OIDPattern.MatchString(tree.TreeOID)) ||
		(tree.ObjectFormat == "sha256" && sha2OIDPattern.MatchString(tree.CommitOID) && sha2OIDPattern.MatchString(tree.TreeOID))
	if !validOIDs || !tree.TreeIdentityDigest.Valid() || !tree.PortableTreeDigest.Valid() ||
		!tree.MaterializationPolicyDigest.Valid() || !tree.MaterializationManifestDigest.Valid() {
		return refuse(CodeInvalidTarget, "tree binding is incomplete or has inconsistent OID width", nil)
	}
	derivedTree, err := pinnedTreeIdentityDigest(tree)
	if err != nil || derivedTree != tree.TreeIdentityDigest {
		return refuse(CodeInvalidTarget, "tree identity digest differs from the exact pinned-tree preimage", err)
	}
	if !attempt.ArtifactDigest.Valid() || !noncePattern.MatchString(attempt.InstanceNonce) || !input.BootSession.IdentityDigest.Valid() {
		return refuse(CodeInvalidTarget, "attempt or boot-session binding is incomplete", nil)
	}
	if !validAbsoluteExecutablePath(runtime.AdmittedExecutablePath) ||
		runtime.MeasuredProcessExecPath != runtime.AdmittedExecutablePath || !versionPattern.MatchString(runtime.Version) ||
		runtime.Major < 1 || runtime.Major > 9007199254740991 || runtime.Platform != "darwin" ||
		runtime.Architecture != "arm64" || !runtime.ExecutableBytesDigest.Valid() ||
		!modePattern.MatchString(runtime.ExecutableMode) || runtime.ExecutableByteCount < 1 ||
		runtime.ExecutableByteCount > 9007199254740991 || !runtime.ProbeProgramDigest.Valid() {
		return refuse(CodeInvalidTarget, "runtime binding is incomplete or outside the exact v1 profile", nil)
	}
	majorText := strings.TrimPrefix(strings.SplitN(runtime.Version, ".", 2)[0], "v")
	major, err := strconv.ParseInt(majorText, 10, 64)
	if err != nil || major != runtime.Major {
		return refuse(CodeInvalidTarget, "runtime major differs from the measured version", err)
	}
	return nil
}

func pinnedTreeIdentityDigest(tree TreeBinding) (domain.Digest, error) {
	digest, _, err := canon.DigestTyped("PinnedTreeIdentity", map[string]any{
		"schema_version": domain.SchemaVersion,
		"kind":           "PinnedTreeIdentity",
		"object_format":  tree.ObjectFormat,
		"commit_oid":     tree.CommitOID,
		"tree_oid":       tree.TreeOID,
	})
	if err != nil {
		return "", err
	}
	return domain.ParseDigest(digest.String())
}

func ParseContractExecutionTarget(exact []byte, expected domain.Digest) (ContractExecutionTarget, error) {
	root, err := parseExactObject(exact, TargetKind, expected, CodeInvalidTarget, CodeTargetDigest)
	if err != nil {
		return ContractExecutionTarget{}, err
	}
	if err := requireRoster(root, "schema_version", "kind", "target_version", "publication_scope",
		"contract_bundle_digest", "terminal_residue_binding", "tree_binding", "attempt_binding",
		"boot_session_binding", "runtime_binding"); err != nil {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "target root roster differs", err)
	}
	schema, _ := textMember(root, "schema_version")
	kind, _ := textMember(root, "kind")
	version, _ := textMember(root, "target_version")
	scope, _ := textMember(root, "publication_scope")
	if schema != domain.SchemaVersion || kind != TargetKind || version != TargetVersionV1 || scope != TargetPublicationScopeV1 {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "target constants differ", nil)
	}
	bundle, err := digestMember(root, "contract_bundle_digest")
	if err != nil {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "bundle reference is invalid", err)
	}
	residueValue, err := member(root, "terminal_residue_binding")
	if err != nil || requireRoster(residueValue, "profile", "study_id", "head_revision", "head_digest", "lineage_root_digest", "stage", "current_kind", "current_digest") != nil {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "terminal residue roster differs", err)
	}
	residueProfile, _ := textMember(residueValue, "profile")
	stage, _ := textMember(residueValue, "stage")
	currentKind, _ := textMember(residueValue, "current_kind")
	currentDigest, currentErr := digestMember(residueValue, "current_digest")
	studyID, _ := textMember(residueValue, "study_id")
	revision, revisionErr := integerMember(residueValue, "head_revision")
	headDigest, headErr := digestMember(residueValue, "head_digest")
	lineage, lineageErr := digestMember(residueValue, "lineage_root_digest")
	if residueProfile != terminalResidueProfileV1 || stage != "RESIDUE" || currentKind != "ContractBundle" ||
		currentErr != nil || revisionErr != nil || headErr != nil || lineageErr != nil || currentDigest != bundle {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "terminal residue constants or joins differ", nil)
	}

	treeValue, err := member(root, "tree_binding")
	if err != nil || requireRoster(treeValue, "authority", "object_format", "commit_oid", "tree_oid", "tree_identity_digest",
		"portable_tree_digest", "materialization_policy_digest", "materialization_manifest_digest", "execution_root_scope") != nil {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "tree roster differs", err)
	}
	treeAuthority, _ := textMember(treeValue, "authority")
	rootScope, _ := textMember(treeValue, "execution_root_scope")
	if treeAuthority != treeAuthorityV1 || rootScope != executionRootScopeV1 {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "tree constants differ", nil)
	}
	tree := TreeBinding{}
	tree.ObjectFormat, _ = textMember(treeValue, "object_format")
	tree.CommitOID, _ = textMember(treeValue, "commit_oid")
	tree.TreeOID, _ = textMember(treeValue, "tree_oid")
	tree.TreeIdentityDigest, _ = digestMember(treeValue, "tree_identity_digest")
	tree.PortableTreeDigest, _ = digestMember(treeValue, "portable_tree_digest")
	tree.MaterializationPolicyDigest, _ = digestMember(treeValue, "materialization_policy_digest")
	tree.MaterializationManifestDigest, _ = digestMember(treeValue, "materialization_manifest_digest")

	attemptValue, err := member(root, "attempt_binding")
	if err != nil || requireRoster(attemptValue, "purpose", "attempt_artifact_digest", "instance_nonce", "allocation_profile", "marker_ordering") != nil {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "attempt roster differs", err)
	}
	purpose, _ := textMember(attemptValue, "purpose")
	allocation, _ := textMember(attemptValue, "allocation_profile")
	marker, _ := textMember(attemptValue, "marker_ordering")
	if purpose != "CONFORMANCE" || allocation != attemptAllocationV1 || marker != attemptMarkerOrderingV1 {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "attempt constants differ", nil)
	}
	attemptDigest, _ := digestMember(attemptValue, "attempt_artifact_digest")
	nonce, _ := textMember(attemptValue, "instance_nonce")

	bootValue, err := member(root, "boot_session_binding")
	if err != nil || requireRoster(bootValue, "profile", "identity_digest") != nil {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "boot-session roster differs", err)
	}
	bootProfile, _ := textMember(bootValue, "profile")
	bootDigest, _ := digestMember(bootValue, "identity_digest")
	if bootProfile != bootSessionProfileV1 {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "boot-session profile differs", nil)
	}

	runtimeValue, err := member(root, "runtime_binding")
	if err != nil || requireRoster(runtimeValue, "authority", "name", "admitted_executable_path", "measured_process_exec_path",
		"version", "major", "platform", "architecture", "executable_identity", "probe_program_digest", "child_resolution") != nil {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "runtime roster differs", err)
	}
	runtimeAuthority, _ := textMember(runtimeValue, "authority")
	runtimeName, _ := textMember(runtimeValue, "name")
	childResolution, _ := textMember(runtimeValue, "child_resolution")
	if runtimeAuthority != runtimeAuthorityV1 || runtimeName != "node" || childResolution != runtimeChildResolutionV1 {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "runtime constants differ", nil)
	}
	executableValue, err := member(runtimeValue, "executable_identity")
	if err != nil || requireRoster(executableValue, "bytes_digest", "mode", "byte_count") != nil {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "executable identity roster differs", err)
	}
	runtime := RuntimeBinding{}
	runtime.AdmittedExecutablePath, _ = textMember(runtimeValue, "admitted_executable_path")
	runtime.MeasuredProcessExecPath, _ = textMember(runtimeValue, "measured_process_exec_path")
	runtime.Version, _ = textMember(runtimeValue, "version")
	runtime.Major, _ = integerMember(runtimeValue, "major")
	runtime.Platform, _ = textMember(runtimeValue, "platform")
	runtime.Architecture, _ = textMember(runtimeValue, "architecture")
	runtime.ExecutableBytesDigest, _ = digestMember(executableValue, "bytes_digest")
	runtime.ExecutableMode, _ = textMember(executableValue, "mode")
	runtime.ExecutableByteCount, _ = integerMember(executableValue, "byte_count")
	runtime.ProbeProgramDigest, _ = digestMember(runtimeValue, "probe_program_digest")

	rebuilt, err := NewContractExecutionTarget(TargetInput{
		ContractBundleDigest: bundle,
		TerminalResidue:      TerminalResidueBinding{StudyID: studyID, HeadRevision: revision, HeadDigest: headDigest, LineageRootDigest: lineage},
		Tree:                 tree,
		Attempt:              AttemptBinding{ArtifactDigest: attemptDigest, InstanceNonce: nonce},
		BootSession:          BootSessionBinding{IdentityDigest: bootDigest},
		Runtime:              runtime,
	})
	if err != nil || rebuilt.digest != expected || !bytes.Equal(rebuilt.canonical, exact) {
		return ContractExecutionTarget{}, refuse(CodeInvalidTarget, "target does not reconstruct exactly", err)
	}
	return rebuilt, nil
}

func cloneTargetInput(input TargetInput) TargetInput { return input }

func (target ContractExecutionTarget) Valid() bool {
	rebuilt, err := ParseContractExecutionTarget(target.canonical, target.digest)
	return err == nil && rebuilt.digest == target.digest && bytes.Equal(rebuilt.canonical, target.canonical)
}

func (target ContractExecutionTarget) Digest() domain.Digest  { return target.digest }
func (target ContractExecutionTarget) CanonicalBytes() []byte { return cloneBytes(target.canonical) }
func (target ContractExecutionTarget) ContractBundleDigest() domain.Digest {
	return target.input.ContractBundleDigest
}
func (target ContractExecutionTarget) AttemptArtifactDigest() domain.Digest {
	return target.input.Attempt.ArtifactDigest
}
func (target ContractExecutionTarget) Input() TargetInput { return cloneTargetInput(target.input) }

func (target ContractExecutionTarget) Equal(other ContractExecutionTarget) bool {
	return target.Valid() && other.Valid() && target.digest == other.digest && bytes.Equal(target.canonical, other.canonical)
}

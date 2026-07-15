package gitobj

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

type ObjectFormat string

const (
	ObjectSHA1   ObjectFormat = "sha1"
	ObjectSHA256 ObjectFormat = "sha256"
)

func (f ObjectFormat) valid() bool { return f == ObjectSHA1 || f == ObjectSHA256 }

func (f ObjectFormat) oidBytes() int {
	if f == ObjectSHA1 {
		return 20
	}
	if f == ObjectSHA256 {
		return 32 // MUTANT_U2_ACCEPT_SHA1_WIDTH_FOR_SHA256
	}
	return 0
}

// OpenConfig requires explicit capabilities. Git and repository lookup never
// consult ambient PATH, HOME, or configuration.
type OpenConfig struct {
	GitExecutable string
	Repository    string
	ScratchRoot   string
}

type repositoryState struct {
	mu                     sync.Mutex
	runner                 closedGit
	fingerprint            domain.Digest
	objectFormat           ObjectFormat
	commonDirectory        string
	objectDirectory        string
	commonFilesystem       filesystemIdentity
	objectFilesystem       filesystemIdentity
	gitExecutableIdentity  filesystemIdentity
	gitExecutableDigest    domain.Digest
	repositoryConfigPath   string
	repositoryConfigDigest domain.Digest
	gitVersion             string
	repositoryDisplayPath  string
	closed                 atomic.Bool
}

// Repository is an opaque admitted local object-database capability. Copying
// the value shares the same internal admission record.
type Repository struct{ state *repositoryState }

func (r Repository) Valid() bool {
	return r.state != nil && !r.state.closed.Load() && r.state.fingerprint.Valid() && r.state.objectFormat.valid()
}

func (r Repository) Fingerprint() domain.Digest {
	if !r.Valid() {
		return ""
	}
	return r.state.fingerprint
}

func (r Repository) ObjectFormat() ObjectFormat {
	if !r.Valid() {
		return ""
	}
	return r.state.objectFormat
}

type Provenance struct {
	DisplayRef            string
	GitVersion            string
	CommitOID             string
	TreeOID               string
	RepositoryFingerprint domain.Digest
}

// PinnedTree is produced only by resolving a display ref once. Provenance is
// exposed separately and is absent from execution identity.
type PinnedTree struct {
	repository *repositoryState
	identity   domain.Digest
	commitOID  string
	treeOID    string
	displayRef string
}

func (p PinnedTree) Valid() bool {
	return p.repository != nil && !p.repository.closed.Load() && p.identity.Valid() && validOID(p.repository.objectFormat, p.commitOID) &&
		validOID(p.repository.objectFormat, p.treeOID)
}

func (p PinnedTree) IdentityDigest() domain.Digest { return p.identity }

func (p PinnedTree) Provenance() Provenance {
	if !p.Valid() {
		return Provenance{}
	}
	return Provenance{
		DisplayRef: p.displayRef, GitVersion: p.repository.gitVersion, CommitOID: p.commitOID, TreeOID: p.treeOID,
		RepositoryFingerprint: p.repository.fingerprint,
	}
}

type Policy struct {
	digest     domain.Digest
	entryLimit int
	totalBytes int64
	singleBlob int64
	profile    string
}

func NewPolicy(entryLimit int, totalBytes, singleBlobBytes int64) (Policy, error) {
	if entryLimit < 1 || entryLimit > 100000 || totalBytes < 1 || totalBytes > 1<<30 ||
		singleBlobBytes < 1 || singleBlobBytes > 1<<27 || singleBlobBytes > totalBytes {
		return Policy{}, refuse(CodeInvalidConfig, "materialization limits are outside the v1 profile", nil)
	}
	identity := struct {
		SchemaVersion   string   `json:"schema_version"`
		Kind            string   `json:"kind"`
		Profile         string   `json:"profile"`
		EntryLimit      int      `json:"entry_limit"`
		TotalBytes      int64    `json:"total_bytes"`
		SingleBlobBytes int64    `json:"single_blob_bytes"`
		Modes           []string `json:"modes"`
		GitFilters      string   `json:"git_filters"`
		GitLFS          string   `json:"git_lfs"`
		GitAlternates   string   `json:"git_alternates"`
	}{
		SchemaVersion: domain.SchemaVersion, Kind: "MaterializationPolicy", Profile: "git-blobs-v1",
		EntryLimit: entryLimit, TotalBytes: totalBytes, SingleBlobBytes: singleBlobBytes,
		Modes: []string{"100644", "100755"}, GitFilters: "FORBIDDEN", GitLFS: "REJECT", GitAlternates: "REJECT",
	}
	digest, err := digestIdentity("MaterializationPolicy", identity)
	if err != nil {
		return Policy{}, err
	}
	return Policy{digest: digest, entryLimit: entryLimit, totalBytes: totalBytes, singleBlob: singleBlobBytes, profile: "git-blobs-v1"}, nil
}

func PolicyFromBudgets(b domain.Budgets) (Policy, error) {
	return NewPolicy(b.MaterializedEntryCount, b.MaterializedBytesPerWorld, b.SingleBlobBytes)
}

func (p Policy) Valid() bool {
	return p.digest.Valid() && p.profile == "git-blobs-v1" && p.entryLimit > 0 && p.totalBytes > 0 && p.singleBlob > 0
}

func (p Policy) Digest() domain.Digest { return p.digest }

type entryRecord struct {
	path           string
	mode           string
	oid            string
	size           int64
	portableDigest domain.Digest
}

type ManifestEntry struct {
	Path               string
	Mode               string
	Bytes              int64
	GitOID             string
	PortableBlobDigest domain.Digest
}

type targetVolume struct {
	identity filesystemIdentity
}

// InspectedTree contains no blob bytes. Materialization must stream and verify
// every object again from the pinned object database.
type InspectedTree struct {
	pinned             PinnedTree
	policy             Policy
	entries            []entryRecord
	portableTreeDigest domain.Digest
	targetVolume       targetVolume
}

func (i InspectedTree) Valid() bool {
	return i.pinned.Valid() && i.policy.Valid() && i.portableTreeDigest.Valid() && len(i.entries) > 0 && i.targetVolume.identity.Valid()
}

func (i InspectedTree) TreeIdentityDigest() domain.Digest { return i.pinned.identity }
func (i InspectedTree) PortableTreeDigest() domain.Digest { return i.portableTreeDigest }
func (i InspectedTree) PolicyDigest() domain.Digest       { return i.policy.digest }
func (i InspectedTree) Provenance() Provenance            { return i.pinned.Provenance() }

func (i InspectedTree) Entries() []ManifestEntry {
	result := make([]ManifestEntry, len(i.entries))
	for index, entry := range i.entries {
		result[index] = ManifestEntry{Path: entry.path, Mode: entry.mode, Bytes: entry.size, GitOID: entry.oid, PortableBlobDigest: entry.portableDigest}
	}
	return result
}

// SelectedTreeSet is the pre-plan candidate authority. It contains only sorted
// portable pinned-tree identities; repository receipts, policy, blob results,
// plan digest, and candidate keys cannot enter this digest.
type SelectedTreeSet struct {
	digest domain.Digest
	pinned []PinnedTree
}

func SelectTrees(pinned ...PinnedTree) (SelectedTreeSet, error) {
	if len(pinned) < 2 || len(pinned) > 4 {
		return SelectedTreeSet{}, refuse(CodeCandidateSetRejected, "candidate count is outside 2..4", nil)
	}
	copyPinned := append([]PinnedTree(nil), pinned...)
	for _, candidate := range copyPinned {
		if !candidate.Valid() {
			return SelectedTreeSet{}, refuse(CodeCandidateSetRejected, "candidate pin is invalid", nil)
		}
	}
	sort.Slice(copyPinned, func(a, b int) bool { return copyPinned[a].identity.String() < copyPinned[b].identity.String() })
	seenIdentity := map[domain.Digest]struct{}{}
	seenExecutableTree := map[string]struct{}{}
	members := make([]string, 0, len(copyPinned))
	for _, candidate := range copyPinned {
		if _, duplicate := seenIdentity[candidate.identity]; duplicate {
			return SelectedTreeSet{}, refuse(CodeCandidateSetRejected, "duplicate pinned-tree identity", nil)
		}
		executableKey := string(candidate.repository.objectFormat) + ":" + candidate.treeOID
		if _, duplicate := seenExecutableTree[executableKey]; duplicate {
			return SelectedTreeSet{}, refuse(CodeCandidateSetRejected, "two selections name the same executable Git tree", nil)
		}
		seenIdentity[candidate.identity] = struct{}{}
		seenExecutableTree[executableKey] = struct{}{}
		members = append(members, candidate.identity.String())
	}
	identity := struct {
		SchemaVersion string   `json:"schema_version"`
		Kind          string   `json:"kind"`
		Count         int      `json:"count"`
		Members       []string `json:"pinned_tree_identity_digests"`
	}{domain.SchemaVersion, "SelectedTreeSet", len(members), members}
	digest, err := digestIdentity("SelectedTreeSet", identity)
	if err != nil {
		return SelectedTreeSet{}, err
	}
	return SelectedTreeSet{digest: digest, pinned: copyPinned}, nil
}

func (s SelectedTreeSet) Valid() bool {
	if !s.digest.Valid() || len(s.pinned) < 2 || len(s.pinned) > 4 {
		return false
	}
	for index, pinned := range s.pinned {
		if !pinned.Valid() || (index > 0 && s.pinned[index-1].identity.String() >= pinned.identity.String()) {
			return false
		}
	}
	return true
}
func (s SelectedTreeSet) Digest() domain.Digest { return s.digest }
func (s SelectedTreeSet) CandidateCount() int   { return len(s.pinned) }

// CandidateSetDeclaration refines the selected set with completed inspection
// under one policy. Its plan-facing digest remains SelectedTreeSet.Digest().
type CandidateSetDeclaration struct {
	selected   SelectedTreeSet
	policy     Policy
	candidates []InspectedTree
}

func NewCandidateSet(selected SelectedTreeSet, policy Policy, candidates ...InspectedTree) (CandidateSetDeclaration, error) {
	if !selected.Valid() || !policy.Valid() || len(candidates) != len(selected.pinned) {
		return CandidateSetDeclaration{}, refuse(CodeCandidateSetRejected, "inspection does not cover the selected tree set", nil)
	}
	copyCandidates := append([]InspectedTree(nil), candidates...)
	sort.Slice(copyCandidates, func(a, b int) bool {
		return copyCandidates[a].pinned.identity.String() < copyCandidates[b].pinned.identity.String()
	})
	seenPortable := map[domain.Digest]struct{}{}
	for index, candidate := range copyCandidates {
		if !candidate.Valid() || candidate.policy.digest != policy.digest || candidate.pinned.identity != selected.pinned[index].identity {
			return CandidateSetDeclaration{}, refuse(CodeCandidateSetRejected, "inspection does not match the selected tree and policy", nil)
		}
		if _, duplicate := seenPortable[candidate.portableTreeDigest]; duplicate { // MUTANT_U2_ALLOW_DUPLICATE_PORTABLE_TREE
			return CandidateSetDeclaration{}, refuse(CodeCandidateSetRejected, "duplicate executable portable tree", nil)
		}
		seenPortable[candidate.portableTreeDigest] = struct{}{}
	}
	return CandidateSetDeclaration{selected: selected, policy: policy, candidates: copyCandidates}, nil
}

func (s CandidateSetDeclaration) Valid() bool {
	if !s.selected.Valid() || !s.policy.Valid() || len(s.candidates) != len(s.selected.pinned) {
		return false
	}
	for index, candidate := range s.candidates {
		if !candidate.Valid() || candidate.policy.digest != s.policy.digest ||
			candidate.pinned.identity != s.selected.pinned[index].identity {
			return false
		}
	}
	return true
}

func (s CandidateSetDeclaration) Digest() domain.Digest { // MUTANT_U2_POLLUTE_PREPLAN_DIGEST
	return s.selected.digest
}
func (s CandidateSetDeclaration) PolicyDigest() domain.Digest { return s.policy.digest }
func (s CandidateSetDeclaration) CandidateCount() int         { return len(s.candidates) }

// BoundCandidate retains both the opaque pure binding and the inspected source
// capability. Callers cannot replace either half with copied digest strings.
type BoundCandidate struct {
	source  InspectedTree
	binding domain.CandidateExecutionBinding
}

func (b BoundCandidate) Valid() bool {
	return b.source.Valid() && b.binding.Valid() && b.binding.Identity().TreeIdentityDigest == b.source.pinned.identity
}

func (b BoundCandidate) Binding() domain.CandidateExecutionBinding { return b.binding }
func (b BoundCandidate) TreeIdentityDigest() domain.Digest         { return b.source.pinned.identity }
func (b BoundCandidate) PortableTreeDigest() domain.Digest         { return b.source.portableTreeDigest }
func (b BoundCandidate) Provenance() Provenance                    { return b.source.pinned.Provenance() }

func (s CandidateSetDeclaration) Bind(plan domain.WorldPlan) ([]BoundCandidate, error) {
	if !s.Valid() || plan.CandidateSetDigest() != s.selected.digest || plan.MaterializationPolicyDigest() != s.policy.digest ||
		plan.Budgets().CandidateCount != len(s.candidates) {
		return nil, refuse(CodePlanBindingMismatch, "world plan does not declare the exact inspected candidate set", nil)
	}
	result := make([]BoundCandidate, 0, len(s.candidates))
	for _, source := range s.candidates {
		binding, err := domain.NewCandidateExecutionBinding(domain.CandidateExecutionIdentity{
			TreeIdentityDigest: source.pinned.identity, MaterializationPolicyDigest: s.policy.digest,
			WorldPlanDigest: plan.Digest(), AdapterDigest: plan.AdapterDigest(), RunnerDigest: plan.Adapter().RunnerDigest,
			ProjectionDefinitionDigest: plan.ProjectionDefinitionDigest(),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, BoundCandidate{source: source, binding: binding})
	}
	return result, nil
}

type MaterializationReceipt struct {
	ManifestDigest        domain.Digest
	PortableTreeDigest    domain.Digest
	PolicyDigest          domain.Digest
	TargetFilesystem      string
	Entries               []ManifestEntry
	PublishedRoot         string
	ManifestPath          string
	ObjectFormat          ObjectFormat
	RepositoryFingerprint domain.Digest
	seal                  domain.Digest
}

func (r MaterializationReceipt) Valid() bool {
	if !r.ManifestDigest.Valid() || !r.PortableTreeDigest.Valid() || !r.PolicyDigest.Valid() ||
		!r.RepositoryFingerprint.Valid() || !r.ObjectFormat.valid() ||
		!strings.HasPrefix(r.TargetFilesystem, "device=") || len(r.TargetFilesystem) == len("device=") ||
		r.PublishedRoot == "" || r.ManifestPath == "" || len(r.Entries) == 0 || !r.seal.Valid() {
		return false
	}
	for index, entry := range r.Entries {
		if entry.Path == "" || (entry.Mode != "100644" && entry.Mode != "100755") || entry.Bytes < 0 ||
			!validOID(r.ObjectFormat, entry.GitOID) || !entry.PortableBlobDigest.Valid() ||
			(index > 0 && r.Entries[index-1].Path >= entry.Path) {
			return false
		}
	}
	seal, err := materializationReceiptSeal(r)
	return err == nil && seal == r.seal
}

type Materializer interface {
	Materialize(context.Context, BoundCandidate, string) (MaterializationReceipt, error)
}

type DefaultMaterializer struct{}

func digestIdentity(kind string, value any) (domain.Digest, error) {
	digest, _, err := canon.DigestTyped(kind, value)
	if err != nil {
		return "", err
	}
	return domain.ParseDigest(digest.String())
}

func validOID(format ObjectFormat, raw string) bool {
	if !format.valid() || len(raw) != format.oidBytes()*2 || strings.ToLower(raw) != raw {
		return false
	}
	decoded, err := hex.DecodeString(raw)
	return err == nil && len(decoded) == format.oidBytes()
}

func entriesEqual(left, right []entryRecord) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func canonicalEntryIdentity(entries []entryRecord) []struct {
	Path   string `json:"path"`
	Mode   string `json:"mode"`
	Bytes  int64  `json:"bytes"`
	Digest string `json:"portable_blob_digest"`
} {
	result := make([]struct {
		Path   string `json:"path"`
		Mode   string `json:"mode"`
		Bytes  int64  `json:"bytes"`
		Digest string `json:"portable_blob_digest"`
	}, len(entries))
	for index, entry := range entries {
		result[index].Path = entry.path
		result[index].Mode = entry.mode
		result[index].Bytes = entry.size
		result[index].Digest = entry.portableDigest.String()
	}
	return result
}

func sameCanonicalBytes(left, right []byte) bool { return bytes.Equal(left, right) }

func formatCount(format ObjectFormat) string {
	return fmt.Sprintf("%s/%d", format, format.oidBytes()*8)
}

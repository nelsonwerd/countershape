package world

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/gitobj"
)

const (
	stateRootEnvironment      = "COUNTERSHAPE_STATE_ROOT"
	evidenceRootEnvironment   = "COUNTERSHAPE_EVIDENCE_ROOT"
	attemptIDEnvironment      = "COUNTERSHAPE_ATTEMPT_ID"
	markerFilename            = "attempt.marker.json"
	inheritAmbientEnvironment = false // MUTANT_U2_INHERIT_AMBIENT_ENVIRONMENT
)

type Roots struct {
	attempt         string
	candidateParent string
	home            string
	temporary       string
	xdgConfig       string
	xdgCache        string
	xdgData         string
	xdgState        string
	state           string
	evidence        string
	marker          string
}

func (r Roots) Attempt() string         { return r.attempt }
func (r Roots) CandidateParent() string { return r.candidateParent }
func (r Roots) Home() string            { return r.home }
func (r Roots) Temporary() string       { return r.temporary }
func (r Roots) XDGConfig() string       { return r.xdgConfig }
func (r Roots) XDGCache() string        { return r.xdgCache }
func (r Roots) XDGData() string         { return r.xdgData }
func (r Roots) XDGState() string        { return r.xdgState }
func (r Roots) State() string           { return r.state }
func (r Roots) Evidence() string        { return r.evidence }
func (r Roots) Marker() string          { return r.marker }

type markerInput struct {
	SchemaVersion         string `json:"schema_version"`
	Kind                  string `json:"kind"`
	WorldPlanDigest       string `json:"world_plan_digest"`
	CandidateExecutionKey string `json:"candidate_execution_key"`
	TreeIdentityDigest    string `json:"tree_identity_digest"`
	StimulusDigest        string `json:"stimulus_digest"`
	Purpose               string `json:"attempt_purpose"`
	InstanceNonce         string `json:"instance_nonce"`
	ScheduleOrdinal       int    `json:"schedule_ordinal"`
	AllocationNonce       string `json:"allocation_nonce"`
	MarkerOrdering        string `json:"marker_ordering"`
}

type allocatedAttempt struct {
	roots         Roots
	markerDigest  domain.Digest
	markerBytes   []byte
	attemptID     string
	world         domain.WorldInstance
	domainAttempt domain.Attempt
}

func allocateAttempt(request Request) (allocatedAttempt, error) {
	binding := request.Candidate.Binding()
	marker := markerInput{
		SchemaVersion: domain.SchemaVersion, Kind: "AttemptMarker",
		WorldPlanDigest: request.Plan.Digest().String(), CandidateExecutionKey: binding.Key().String(),
		TreeIdentityDigest: request.Candidate.TreeIdentityDigest().String(), StimulusDigest: request.StimulusDigest.String(),
		Purpose: string(request.Purpose), InstanceNonce: request.InstanceNonce, ScheduleOrdinal: request.ScheduleOrdinal,
		MarkerOrdering: "DURABLE_BEFORE_SPAWN",
	}
	roots, markerDigest, bytes, attemptID, err := allocateOwnedRoots(request.AllocationRoot, marker)
	if err != nil {
		return allocatedAttempt{}, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(roots.attempt)
		}
	}()
	domainAttempt, err := domain.NewAttempt(attemptID, markerDigest, request.Purpose)
	if err != nil {
		return allocatedAttempt{}, err
	}
	world, err := domain.NewWorldInstance(request.Plan, binding, domain.WorldInstanceConfig{
		StimulusDigest: request.StimulusDigest, AttemptArtifactDigest: markerDigest,
		Purpose: request.Purpose, InstanceNonce: request.InstanceNonce, ScheduleOrdinal: request.ScheduleOrdinal,
	})
	if err != nil {
		return allocatedAttempt{}, err
	}
	cleanup = false
	return allocatedAttempt{
		roots: roots, markerDigest: markerDigest, markerBytes: append([]byte(nil), bytes...),
		attemptID: attemptID, world: world, domainAttempt: domainAttempt,
	}, nil
}

func allocateOwnedRoots(rawBase string, marker markerInput) (Roots, domain.Digest, []byte, string, error) {
	base, err := privateCanonicalDirectory(rawBase)
	if err != nil {
		return Roots{}, "", nil, "", refuse(
			CodeInvalidAllocation, "allocation root must be an existing private canonical directory", err,
		)
	}
	attemptRoot, err := os.MkdirTemp(base, "countershape-attempt-")
	if err != nil {
		return Roots{}, "", nil, "", refuse(CodeInvalidAllocation, "attempt root allocation failed", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(attemptRoot)
		}
	}()
	if err := os.Chmod(attemptRoot, 0o700); err != nil {
		return Roots{}, "", nil, "", refuse(CodeInvalidAllocation, "attempt root mode failed", err)
	}
	roots := Roots{attempt: attemptRoot}
	for name, target := range map[string]*string{
		"candidate-parent": &roots.candidateParent,
		"home":             &roots.home,
		"tmp":              &roots.temporary,
		"xdg-config":       &roots.xdgConfig,
		"xdg-cache":        &roots.xdgCache,
		"xdg-data":         &roots.xdgData,
		"xdg-state":        &roots.xdgState,
		"state":            &roots.state,
		"evidence":         &roots.evidence,
	} {
		path := filepath.Join(attemptRoot, name)
		if err := os.Mkdir(path, 0o700); err != nil {
			return Roots{}, "", nil, "", refuse(CodeInvalidAllocation, "owned root allocation failed: "+name, err)
		}
		if err := os.Chmod(path, 0o700); err != nil {
			return Roots{}, "", nil, "", refuse(CodeInvalidAllocation, "owned root mode failed: "+name, err)
		}
		*target = path
	}
	marker.AllocationNonce = filepath.Base(attemptRoot)
	digest, bytes, err := canon.DigestTyped("AttemptMarker", marker)
	if err != nil {
		return Roots{}, "", nil, "", refuse(CodeMarkerWriteFailed, "attempt marker cannot be canonicalized", err)
	}
	markerDigest, err := domain.ParseDigest(digest.String())
	if err != nil {
		return Roots{}, "", nil, "", err
	}
	attemptID := "attempt:" + digest.String()[len("sha256:"):]
	roots.marker = filepath.Join(roots.evidence, markerFilename)
	if err := writeDurableExclusive(roots.marker, bytes); err != nil {
		return Roots{}, "", nil, "", refuse(CodeMarkerWriteFailed, "attempt marker could not be durably created", err)
	}
	if err := syncDirectory(roots.evidence); err != nil {
		return Roots{}, "", nil, "", refuse(CodeMarkerWriteFailed, "attempt evidence directory sync failed", err)
	}
	cleanup = false
	return roots, markerDigest, append([]byte(nil), bytes...), attemptID, nil
}

func privateCanonicalDirectory(raw string) (string, error) {
	if !filepath.IsAbs(raw) || filepath.Clean(raw) != raw {
		return "", errors.New("path is not clean and absolute")
	}
	resolved, err := filepath.EvalSymlinks(raw)
	if err != nil || resolved != raw {
		return "", errors.New("path is not already symlink-resolved")
	}
	info, err := os.Lstat(raw)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return "", errors.New("directory is absent, non-directory, or accessible to group/other")
	}
	return raw, nil
}

func writeDurableExclusive(path string, data []byte) error {
	handle, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	written := 0
	for written < len(data) {
		count, writeErr := handle.Write(data[written:])
		written += count
		if writeErr != nil {
			_ = handle.Close()
			return writeErr
		}
		if count == 0 {
			_ = handle.Close()
			return io.ErrNoProgress
		}
	}
	if err := handle.Chmod(0o600); err != nil {
		_ = handle.Close()
		return err
	}
	return errors.Join(handle.Sync(), handle.Close())
}

func syncDirectory(path string) error {
	handle, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(handle.Sync(), handle.Close())
}

func buildEnvironment(entries []domain.EnvironmentEntry, roots Roots, attemptID string) []string {
	values := make(map[string]string, len(entries)+9)
	if inheritAmbientEnvironment {
		for _, entry := range os.Environ() {
			if name, value, present := strings.Cut(entry, "="); present {
				values[name] = value
			}
		}
	}
	for _, entry := range entries {
		values[entry.Name] = entry.Value
	}
	values["HOME"] = roots.home
	values["TMPDIR"] = roots.temporary
	values["XDG_CONFIG_HOME"] = roots.xdgConfig
	values["XDG_CACHE_HOME"] = roots.xdgCache
	values["XDG_DATA_HOME"] = roots.xdgData
	values["XDG_STATE_HOME"] = roots.xdgState
	values[stateRootEnvironment] = roots.state
	values[evidenceRootEnvironment] = roots.evidence
	values[attemptIDEnvironment] = attemptID
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	environment := make([]string, len(names))
	for index, name := range names {
		environment[index] = name + "=" + values[name]
	}
	return environment
}

func cloneMaterializationReceipt(input gitobj.MaterializationReceipt) gitobj.MaterializationReceipt {
	result := input
	result.Entries = append([]gitobj.ManifestEntry(nil), input.Entries...)
	return result
}

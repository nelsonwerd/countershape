package gitobj

import (
	"bytes"
	"context"
	"crypto/sha1" // Git SHA-1 compatibility; portable identity is SHA-256.
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	maxGitPathBytes                    = 4096
	maxGitComponentBytes               = 255
	maxTreeDepth                       = 128
	maxTreeObjectReads                 = 8192
	preserveExecutableMode             = true  // MUTANT_U2_COLLAPSE_EXECUTABLE_MODE
	acceptSymlinkMode                  = false // MUTANT_U2_ACCEPT_SYMLINK_MODE
	validatePathsBeforeCandidateBytes  = true  // MUTANT_U2_VALIDATE_PATH_AFTER_WRITE
	rejectLFSPointerContent            = true  // MUTANT_U2_ACCEPT_LFS_POINTER
	requireTargetFilesystemReservation = true  // MUTANT_U2_LEXICAL_ONLY_PATH_COLLISION
)

func gitHasher(format ObjectFormat) (hash.Hash, error) {
	switch format {
	case ObjectSHA1:
		return sha1.New(), nil
	case ObjectSHA256:
		return sha256.New(), nil
	default:
		return nil, refuse(CodeUnsupportedFormat, "unknown object hasher", nil)
	}
}

func writeObjectHeader(writer io.Writer, objectType string, size int64) {
	_, _ = io.WriteString(writer, objectType)
	_, _ = io.WriteString(writer, " ")
	_, _ = io.WriteString(writer, strconv.FormatInt(size, 10))
	_, _ = writer.Write([]byte{0})
}

func requireMatchingObjectOID(expected, computed, objectType string) error {
	if computed != expected { // MUTANT_U2_SKIP_BLOB_REHASH
		return refuse(CodeObjectHashMismatch, objectType+" object bytes do not match the requested OID", nil)
	}
	return nil
}

func (state *repositoryState) readVerifiedObject(ctx context.Context, objectType, oid string, limit int64) ([]byte, error) {
	size, err := state.objectSize(ctx, oid)
	if err != nil {
		return nil, err
	}
	if size < 0 || size > limit || size > int64(int(^uint(0)>>1)) {
		return nil, refuse(CodeBudgetExceeded, objectType+" object exceeds its verification bound", nil)
	}
	buffer := bytes.NewBuffer(make([]byte, 0, int(size)))
	objectHash, err := gitHasher(state.objectFormat)
	if err != nil {
		return nil, err
	}
	writeObjectHeader(objectHash, objectType, size)
	if err := state.streamObject(ctx, objectType, oid, size, io.MultiWriter(buffer, objectHash)); err != nil {
		return nil, err
	}
	computed := hex.EncodeToString(objectHash.Sum(nil))
	if err := requireMatchingObjectOID(oid, computed, objectType); err != nil {
		return nil, err
	}
	return append([]byte(nil), buffer.Bytes()...), nil
}

func (state *repositoryState) verifyObjectOnly(ctx context.Context, objectType, oid string, limit int64) error {
	size, err := state.objectSize(ctx, oid)
	if err != nil {
		return err
	}
	if size < 0 || size > limit {
		return refuse(CodeBudgetExceeded, objectType+" object exceeds its verification bound", nil)
	}
	objectHash, err := gitHasher(state.objectFormat)
	if err != nil {
		return err
	}
	writeObjectHeader(objectHash, objectType, size)
	if err := state.streamObject(ctx, objectType, oid, size, objectHash); err != nil {
		return err
	}
	return requireMatchingObjectOID(oid, hex.EncodeToString(objectHash.Sum(nil)), objectType)
}

func treeOIDFromCommit(format ObjectFormat, commit []byte) (string, error) {
	lineEnd := bytes.IndexByte(commit, '\n')
	if lineEnd < 0 || lineEnd > 5+format.oidBytes()*2 || !bytes.HasPrefix(commit[:lineEnd], []byte("tree ")) {
		return "", refuse(CodeMalformedTree, "verified commit has no canonical leading tree header", nil)
	}
	oid := string(commit[len("tree "):lineEnd])
	if !validOID(format, oid) {
		return "", refuse(CodeMalformedTree, "verified commit tree OID is malformed", nil)
	}
	return oid, nil
}

type rawTreeEntry struct {
	mode string
	name []byte
	oid  string
}

type treeParseBudget struct {
	records         int
	metadata        int
	treeObjects     int
	recordLimit     int
	metadataLimit   int
	treeObjectLimit int
}

func newTreeParseBudget(entryLimit int) *treeParseBudget {
	treeObjectLimit := entryLimit + maxTreeDepth + 1
	if treeObjectLimit > maxTreeObjectReads {
		treeObjectLimit = maxTreeObjectReads
	}
	recordLimit := entryLimit + treeObjectLimit + 256
	metadataLimit := recordLimit * maxGitPathBytes
	if metadataLimit > 32<<20 {
		metadataLimit = 32 << 20
	}
	return &treeParseBudget{
		recordLimit: recordLimit, metadataLimit: metadataLimit, treeObjectLimit: treeObjectLimit,
	}
}

func (budget *treeParseBudget) admitTreeObject() error {
	if budget == nil {
		return refuse(CodeBudgetExceeded, "tree object-read budget is unavailable", nil)
	}
	budget.treeObjects++
	if budget.treeObjects > budget.treeObjectLimit {
		return refuse(CodeBudgetExceeded, "tree object-read budget exceeded", nil)
	}
	return nil
}

func parseRawTreeObject(format ObjectFormat, raw []byte, budget *treeParseBudget) ([]rawTreeEntry, error) {
	if len(raw) == 0 || budget == nil {
		return nil, refuse(CodeMalformedTree, "tree object is empty or has no parse budget", nil)
	}
	entries := make([]rawTreeEntry, 0)
	var priorSortKey []byte
	for offset := 0; offset < len(raw); {
		spaceRelative := bytes.IndexByte(raw[offset:], ' ')
		if spaceRelative < 1 {
			return nil, refuse(CodeMalformedTree, "raw tree mode boundary is malformed", nil)
		}
		space := offset + spaceRelative
		mode := string(raw[offset:space])
		nameStart := space + 1
		nulRelative := bytes.IndexByte(raw[nameStart:], 0)
		if nulRelative < 1 {
			return nil, refuse(CodeMalformedTree, "raw tree name boundary is malformed", nil)
		}
		nul := nameStart + nulRelative
		oidStart := nul + 1
		oidEnd := oidStart + format.oidBytes()
		if oidEnd > len(raw) || format.oidBytes() == 0 {
			return nil, refuse(CodeMalformedTree, "raw tree OID bytes are truncated", nil)
		}
		name := append([]byte(nil), raw[nameStart:nul]...)
		if bytes.IndexByte(name, '/') >= 0 {
			return nil, refuse(CodeMalformedTree, "raw tree component contains slash", nil)
		}
		oid := hex.EncodeToString(raw[oidStart:oidEnd])
		if !validOID(format, oid) {
			return nil, refuse(CodeMalformedTree, "raw tree OID is malformed", nil)
		}
		budget.records++
		budget.metadata += oidEnd - offset
		if budget.records > budget.recordLimit || budget.metadata > budget.metadataLimit {
			return nil, refuse(CodeBudgetExceeded, "tree record or metadata budget exceeded", nil)
		}
		isTree := mode == "40000"
		sortKey := append([]byte(nil), name...)
		if isTree {
			sortKey = append(sortKey, '/')
		}
		if len(priorSortKey) > 0 && bytes.Compare(priorSortKey, sortKey) >= 0 {
			return nil, refuse(CodeMalformedTree, "raw tree entries are duplicate or noncanonical", nil)
		}
		priorSortKey = sortKey
		entries = append(entries, rawTreeEntry{mode: mode, name: name, oid: oid})
		offset = oidEnd
	}
	return entries, nil
}

func readVerifiedTreeEntries(
	ctx context.Context,
	state *repositoryState,
	treeOID string,
	prefix []byte,
	depth int,
	entryLimit int,
	budget *treeParseBudget,
	stack map[string]struct{},
) ([]entryRecord, error) {
	if depth > maxTreeDepth {
		return nil, refuse(CodeBudgetExceeded, "tree nesting depth exceeds the closed profile", nil)
	}
	if _, recursive := stack[treeOID]; recursive {
		return nil, refuse(CodeMalformedTree, "tree object graph is recursive", nil)
	}
	if err := budget.admitTreeObject(); err != nil {
		return nil, err
	}
	stack[treeOID] = struct{}{}
	defer delete(stack, treeOID)
	remainingMetadata := budget.metadataLimit - budget.metadata
	if remainingMetadata < 1 {
		return nil, refuse(CodeBudgetExceeded, "tree metadata budget exhausted", nil)
	}
	raw, err := state.readVerifiedObject(ctx, "tree", treeOID, int64(remainingMetadata))
	if err != nil {
		return nil, err
	}
	entries, err := parseRawTreeObject(state.objectFormat, raw, budget)
	if err != nil {
		return nil, err
	}
	result := make([]entryRecord, 0)
	for _, entry := range entries {
		full := append([]byte(nil), prefix...)
		if len(full) > 0 {
			full = append(full, '/')
		}
		full = append(full, entry.name...)
		if len(full) > maxGitPathBytes {
			return nil, refuse(CodeBudgetExceeded, "materialized path metadata exceeds its bound", nil)
		}
		if validatePathsBeforeCandidateBytes {
			if err := validateRawPath(full); err != nil {
				return nil, err
			}
		}
		switch entry.mode {
		case "40000":
			children, err := readVerifiedTreeEntries(ctx, state, entry.oid, full, depth+1, entryLimit, budget, stack)
			if err != nil {
				return nil, err
			}
			result = append(result, children...)
		case "100644", "100755":
			result = append(result, entryRecord{path: string(full), mode: entry.mode, oid: entry.oid})
		case "120000":
			if !acceptSymlinkMode {
				return nil, refuse(CodeUnsupportedMode, entry.mode+" blob", nil)
			}
			result = append(result, entryRecord{path: string(full), mode: entry.mode, oid: entry.oid})
		default:
			return nil, refuse(CodeUnsupportedMode, entry.mode+" object", nil)
		}
		if len(result) > entryLimit {
			return nil, refuse(CodeBudgetExceeded, "materialized entry count exceeds policy", nil)
		}
	}
	return result, nil
}

func validateRawPath(raw []byte) error {
	if len(raw) < 1 || len(raw) > maxGitPathBytes || !utf8.Valid(raw) || raw[0] == '/' || raw[len(raw)-1] == '/' || bytes.IndexByte(raw, '\\') >= 0 {
		return refuse(CodeUnsafePath, fmt.Sprintf("unsafe path bytes %q", raw), nil)
	}
	value := string(raw)
	if path.IsAbs(value) || path.Clean(value) != value || filepath.VolumeName(value) != "" {
		return refuse(CodeUnsafePath, fmt.Sprintf("noncanonical path %q", raw), nil)
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || component == "." || component == ".." || len(component) > maxGitComponentBytes || strings.EqualFold(component, ".git") {
			return refuse(CodeUnsafePath, fmt.Sprintf("unsafe path component in %q", raw), nil)
		}
		for _, character := range component {
			if unicode.IsControl(character) || character == 0x7f {
				return refuse(CodeUnsafePath, fmt.Sprintf("control text in path %q", raw), nil)
			}
		}
	}
	return nil
}

func validateCrossEntryPaths(entries []entryRecord) error {
	for index, entry := range entries {
		if index > 0 {
			previous := entries[index-1].path
			if entry.path == previous || strings.HasPrefix(entry.path, previous+"/") {
				return refuse(CodePathCollision, "duplicate or file/directory prefix conflict", nil)
			}
		}
	}
	return nil
}

func probeTargetTopology(targetParent string, entries []entryRecord) (targetVolume, error) {
	parent, err := canonicalDirectory(targetParent)
	if err != nil {
		return targetVolume{}, refuse(CodeInvalidConfig, "target parent is unavailable", err)
	}
	parentIdentity, err := filesystemIdentityAt(parent)
	if err != nil {
		return targetVolume{}, refuse(CodeUnsupportedPlatform, "target filesystem cannot be measured", err)
	}
	probe, err := os.MkdirTemp(parent, ".countershape-topology-probe-")
	if err != nil {
		return targetVolume{}, refuse(CodePathCollision, "target topology probe allocation failed", err)
	}
	defer os.RemoveAll(probe)
	if err := os.Chmod(probe, 0o700); err != nil {
		return targetVolume{}, refuse(CodePathCollision, "target topology probe mode failed", err)
	}
	seenDirectories := map[string]struct{}{}
	for _, entry := range entries {
		components := strings.Split(entry.path, "/")
		for depth := 1; depth < len(components); depth++ {
			logical := strings.Join(components[:depth], "/")
			if _, exact := seenDirectories[logical]; exact {
				continue
			}
			if err := os.Mkdir(filepath.Join(probe, filepath.FromSlash(logical)), 0o700); err != nil {
				return targetVolume{}, refuse(CodePathCollision, "distinct directory names alias on the target filesystem", err)
			}
			seenDirectories[logical] = struct{}{}
		}
		file, err := os.OpenFile(filepath.Join(probe, filepath.FromSlash(entry.path)), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return targetVolume{}, refuse(CodePathCollision, "distinct file names alias on the target filesystem", err)
		}
		if err := file.Close(); err != nil {
			return targetVolume{}, refuse(CodePathCollision, "target topology probe close failed", err)
		}
	}
	return targetVolume{identity: parentIdentity}, nil
}

// reserveTargetVolume keeps the policy decision and the filesystem probe in
// one narrow boundary. Passing the probe explicitly makes it possible to
// prove that the closed profile actually reserves names on the target volume,
// without replacing the production probe with mutable package state.
func reserveTargetVolume(
	targetParent string,
	entries []entryRecord,
	topologyProbe func(string, []entryRecord) (targetVolume, error),
) (targetVolume, error) {
	if requireTargetFilesystemReservation {
		if topologyProbe == nil {
			return targetVolume{}, refuse(CodeInvalidConfig, "target topology probe is unavailable", nil)
		}
		return topologyProbe(targetParent, entries)
	}
	parent, err := canonicalDirectory(targetParent)
	if err != nil {
		return targetVolume{}, refuse(CodeInvalidConfig, "target parent is unavailable", err)
	}
	identity, err := filesystemIdentityAt(parent)
	if err != nil {
		return targetVolume{}, err
	}
	return targetVolume{identity: identity}, nil
}

type prefixCapture struct {
	bytes []byte
	limit int
}

func (p *prefixCapture) Write(data []byte) (int, error) {
	original := len(data)
	remaining := p.limit - len(p.bytes)
	if remaining > 0 {
		if remaining > len(data) {
			remaining = len(data)
		}
		p.bytes = append(p.bytes, data[:remaining]...)
	}
	return original, nil
}

func portableDigestFromHash(hasher hash.Hash) (domain.Digest, error) {
	return domain.ParseDigest("sha256:" + hex.EncodeToString(hasher.Sum(nil)))
}

func verifyBlobStream(ctx context.Context, state *repositoryState, entry entryRecord, destination io.Writer) (domain.Digest, error) {
	objectHash, err := gitHasher(state.objectFormat)
	if err != nil {
		return "", err
	}
	writeObjectHeader(objectHash, "blob", entry.size)
	portableHash := sha256.New()
	_, _ = io.WriteString(portableHash, "countershape/portable-blob/v1")
	_, _ = portableHash.Write([]byte{0})
	portableMode := entry.mode
	if !preserveExecutableMode && portableMode == "100755" {
		portableMode = "100644"
	}
	_, _ = io.WriteString(portableHash, portableMode)
	_, _ = portableHash.Write([]byte{0})
	_, _ = io.WriteString(portableHash, strconv.FormatInt(entry.size, 10))
	_, _ = portableHash.Write([]byte{0})
	prefix := &prefixCapture{limit: 128}
	writers := []io.Writer{objectHash, portableHash, prefix}
	if destination != nil {
		writers = append(writers, destination)
	}
	if err := state.streamObject(ctx, "blob", entry.oid, entry.size, io.MultiWriter(writers...)); err != nil {
		return "", err
	}
	computedOID := hex.EncodeToString(objectHash.Sum(nil))
	if err := requireMatchingObjectOID(entry.oid, computedOID, "blob"); err != nil {
		return "", err
	}
	if rejectLFSPointerContent && bytes.HasPrefix(prefix.bytes, []byte("version https://git-lfs.github.com/spec/v1")) {
		return "", refuse(CodeLFSPointer, "Git LFS pointer content is outside the v1 source profile", nil)
	}
	return portableDigestFromHash(portableHash)
}

func Inspect(ctx context.Context, pinned PinnedTree, policy Policy, targetParent string) (InspectedTree, error) {
	if !pinned.Valid() || !policy.Valid() {
		return InspectedTree{}, refuse(CodeInvalidConfig, "pin or policy is invalid", nil)
	}
	state := pinned.repository
	state.mu.Lock()
	defer state.mu.Unlock()
	if err := state.revalidate(); err != nil {
		return InspectedTree{}, err
	}
	entries, err := readVerifiedTreeEntries(
		ctx, state, pinned.treeOID, nil, 0, policy.entryLimit, newTreeParseBudget(policy.entryLimit), map[string]struct{}{},
	)
	if err != nil {
		return InspectedTree{}, err
	}
	if len(entries) == 0 {
		return InspectedTree{}, refuse(CodeMalformedTree, "selected tree contains no supported blobs", nil)
	}
	sort.Slice(entries, func(a, b int) bool { return entries[a].path < entries[b].path })
	if err := validateCrossEntryPaths(entries); err != nil {
		return InspectedTree{}, err
	}
	var total int64
	for index := range entries {
		size, err := state.objectSize(ctx, entries[index].oid)
		if err != nil {
			return InspectedTree{}, err
		}
		if size > policy.singleBlob || total > math.MaxInt64-size || total+size > policy.totalBytes {
			return InspectedTree{}, refuse(CodeBudgetExceeded, "blob or aggregate materialization bytes exceed policy", nil)
		}
		entries[index].size = size
		total += size
	}
	for index := range entries {
		digest, err := verifyBlobStream(ctx, state, entries[index], nil)
		if err != nil {
			return InspectedTree{}, err
		}
		entries[index].portableDigest = digest
	}
	if !validatePathsBeforeCandidateBytes {
		for _, entry := range entries {
			if err := validateRawPath([]byte(entry.path)); err != nil {
				return InspectedTree{}, err
			}
		}
	}
	volume, err := reserveTargetVolume(targetParent, entries, probeTargetTopology)
	if err != nil {
		return InspectedTree{}, err
	}
	portableTreeDigest, err := digestIdentity("PortableTree", struct {
		SchemaVersion string      `json:"schema_version"`
		Kind          string      `json:"kind"`
		Entries       interface{} `json:"entries"`
	}{domain.SchemaVersion, "PortableTree", canonicalEntryIdentity(entries)})
	if err != nil {
		return InspectedTree{}, err
	}
	return InspectedTree{
		pinned: pinned, policy: policy, entries: append([]entryRecord(nil), entries...),
		portableTreeDigest: portableTreeDigest, targetVolume: volume,
	}, nil
}

func InspectSelected(ctx context.Context, selected SelectedTreeSet, policy Policy, targetParent string) (CandidateSetDeclaration, error) {
	if !selected.Valid() || !policy.Valid() {
		return CandidateSetDeclaration{}, refuse(CodeInvalidConfig, "selected set or policy is invalid", nil)
	}
	inspected := make([]InspectedTree, 0, len(selected.pinned))
	for _, pinned := range selected.pinned {
		candidate, err := Inspect(ctx, pinned, policy, targetParent)
		if err != nil {
			return CandidateSetDeclaration{}, err
		}
		inspected = append(inspected, candidate)
	}
	return NewCandidateSet(selected, policy, inspected...)
}

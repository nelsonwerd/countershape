package world

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const httpSeedOverlayAuthority = "U4_PRIVATE_SEED_ROOT_EXCLUSIVE_REOPEN_REHASH_V1"

type HTTPSeedEntryReceipt struct {
	path       string
	mode       string
	bytes      int64
	byteDigest domain.Digest
}

func (r HTTPSeedEntryReceipt) Path() string              { return r.path }
func (r HTTPSeedEntryReceipt) Mode() string              { return r.mode }
func (r HTTPSeedEntryReceipt) Bytes() int64              { return r.bytes }
func (r HTTPSeedEntryReceipt) ByteDigest() domain.Digest { return r.byteDigest }

type HTTPSeedOverlayReceipt struct {
	digest                 domain.Digest
	canonicalBytes         []byte
	executionBindingDigest domain.Digest
	fixtureRecipeDigest    domain.Digest
	root                   string
	entries                []HTTPSeedEntryReceipt
}

type httpSeedOverlayIdentity struct {
	SchemaVersion          string `json:"schema_version"`
	Kind                   string `json:"kind"`
	Authority              string `json:"authority"`
	ExecutionBindingDigest string `json:"http_execution_binding_digest"`
	FixtureRecipeDigest    string `json:"fixture_recipe_digest"`
	Root                   string `json:"root"`
	Entries                []struct {
		Path       string `json:"path"`
		Mode       string `json:"mode"`
		Bytes      int64  `json:"bytes"`
		ByteDigest string `json:"byte_digest"`
	} `json:"entries"`
}

func materializeHTTPSeeds(binding httpmodel.HTTPExecutionBinding, root string) (HTTPSeedOverlayReceipt, error) {
	if !binding.Valid() || !binding.FixtureRecipe().Valid() || binding.FixtureRecipeDigest() != binding.FixtureRecipe().Digest() ||
		!filepath.IsAbs(root) || filepath.Clean(root) != root {
		return HTTPSeedOverlayReceipt{}, refuse(CodeHTTPSeedRejected, "seed authority or root is invalid", nil)
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root {
		return HTTPSeedOverlayReceipt{}, refuse(CodeHTTPSeedRejected, "seed root is not canonical", err)
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode().Perm()&0o077 != 0 {
		return HTTPSeedOverlayReceipt{}, refuse(CodeHTTPSeedRejected, "seed root is not a private directory", err)
	}
	entries := make([]HTTPSeedEntryReceipt, 0, len(binding.Seeds()))
	for _, seed := range binding.Seeds() {
		entry, err := writeAndReopenHTTPSeed(root, seed)
		if err != nil {
			return HTTPSeedOverlayReceipt{}, err
		}
		entries = append(entries, entry)
	}
	if err := syncDirectory(root); err != nil {
		return HTTPSeedOverlayReceipt{}, refuse(CodeHTTPSeedRejected, "seed root sync failed", err)
	}
	identity := httpSeedOverlayIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "HTTPSeedOverlayReceipt", Authority: httpSeedOverlayAuthority,
		ExecutionBindingDigest: binding.Digest().String(), FixtureRecipeDigest: binding.FixtureRecipeDigest().String(), Root: root,
		Entries: make([]struct {
			Path       string `json:"path"`
			Mode       string `json:"mode"`
			Bytes      int64  `json:"bytes"`
			ByteDigest string `json:"byte_digest"`
		}, len(entries)),
	}
	for index, entry := range entries {
		identity.Entries[index].Path, identity.Entries[index].Mode = entry.path, entry.mode
		identity.Entries[index].Bytes, identity.Entries[index].ByteDigest = entry.bytes, entry.byteDigest.String()
	}
	digestRaw, canonicalBytes, err := canon.DigestTyped("HTTPSeedOverlayReceipt", identity)
	if err != nil {
		return HTTPSeedOverlayReceipt{}, err
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return HTTPSeedOverlayReceipt{}, err
	}
	receipt := HTTPSeedOverlayReceipt{
		digest: digest, canonicalBytes: canonicalBytes, executionBindingDigest: binding.Digest(),
		fixtureRecipeDigest: binding.FixtureRecipeDigest(), root: root, entries: entries,
	}
	if !receipt.Valid() {
		return HTTPSeedOverlayReceipt{}, refuse(CodeHTTPSeedRejected, "constructed seed receipt is invalid", nil)
	}
	return receipt, nil
}

func writeAndReopenHTTPSeed(root string, seed httpmodel.HTTPSeedFile) (HTTPSeedEntryReceipt, error) {
	segments := strings.Split(seed.Path(), "/")
	if len(segments) == 0 {
		return HTTPSeedEntryReceipt{}, refuse(CodeHTTPSeedRejected, "seed path is empty", nil)
	}
	parent := root
	for _, segment := range segments[:len(segments)-1] {
		if strings.EqualFold(segment, ".git") {
			return HTTPSeedEntryReceipt{}, refuse(CodeHTTPSeedRejected, "seed path enters reserved metadata", nil)
		}
		next, err := ensureExactFixtureDirectory(parent, segment)
		if err != nil {
			return HTTPSeedEntryReceipt{}, refuse(CodeHTTPSeedRejected, "seed parent creation failed", err)
		}
		parent = next
	}
	leaf := segments[len(segments)-1]
	if strings.EqualFold(leaf, ".git") {
		return HTTPSeedEntryReceipt{}, refuse(CodeHTTPSeedRejected, "seed path uses reserved metadata", nil)
	}
	if present, _, err := exactOrAliasedChild(parent, leaf); err != nil || present {
		return HTTPSeedEntryReceipt{}, refuse(CodeHTTPSeedRejected, "seed destination exists or aliases another name", err)
	}
	if seed.Mode() != httpmodel.SeedMode0644 {
		return HTTPSeedEntryReceipt{}, refuse(CodeHTTPSeedRejected, "seed mode exceeds fixed authority", nil)
	}
	contents := seed.Contents()
	digestRaw, err := canon.DigestBytes("HTTPSeedBytes", contents)
	if err != nil {
		return HTTPSeedEntryReceipt{}, err
	}
	expectedDigest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return HTTPSeedEntryReceipt{}, err
	}
	destination := filepath.Join(parent, leaf)
	if err := writeExclusiveSyncedFile(destination, contents, 0o644); err != nil {
		return HTTPSeedEntryReceipt{}, refuse(CodeHTTPSeedRejected, "seed exclusive write failed", err)
	}
	if err := syncDirectory(parent); err != nil {
		return HTTPSeedEntryReceipt{}, refuse(CodeHTTPSeedRejected, "seed parent sync failed", err)
	}
	actualDigest, actualBytes, err := reopenAndHashHTTPSeed(destination, int64(len(contents)))
	if err != nil {
		return HTTPSeedEntryReceipt{}, err
	}
	// MUTATION_ANCHOR: http-seed-must-be-reopened-and-rehashed-in-fresh-root
	if actualDigest != expectedDigest || !bytes.Equal(actualBytes, contents) {
		return HTTPSeedEntryReceipt{}, refuse(CodeHTTPSeedRejected, "reopened seed differs from immutable binding", nil)
	}
	return HTTPSeedEntryReceipt{path: seed.Path(), mode: string(seed.Mode()), bytes: int64(len(contents)), byteDigest: actualDigest}, nil
}

func reopenAndHashHTTPSeed(path string, expectedBytes int64) (domain.Digest, []byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o644 || info.Size() != expectedBytes {
		return "", nil, refuse(CodeHTTPSeedRejected, "seed did not reopen with exact regular-file facts", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || int64(len(data)) != expectedBytes {
		return "", nil, refuse(CodeHTTPSeedRejected, "seed reopen failed", err)
	}
	digestRaw, err := canon.DigestBytes("HTTPSeedBytes", data)
	if err != nil {
		return "", nil, err
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	return digest, data, err
}

func (r HTTPSeedOverlayReceipt) Digest() domain.Digest { return r.digest }
func (r HTTPSeedOverlayReceipt) CanonicalBytes() []byte {
	return append([]byte(nil), r.canonicalBytes...)
}
func (r HTTPSeedOverlayReceipt) ExecutionBindingDigest() domain.Digest {
	return r.executionBindingDigest
}
func (r HTTPSeedOverlayReceipt) FixtureRecipeDigest() domain.Digest { return r.fixtureRecipeDigest }
func (r HTTPSeedOverlayReceipt) Root() string                       { return r.root }
func (r HTTPSeedOverlayReceipt) Authority() string                  { return httpSeedOverlayAuthority }
func (r HTTPSeedOverlayReceipt) Entries() []HTTPSeedEntryReceipt {
	return append([]HTTPSeedEntryReceipt(nil), r.entries...)
}
func (r HTTPSeedOverlayReceipt) Valid() bool {
	if !r.digest.Valid() || !r.executionBindingDigest.Valid() || !r.fixtureRecipeDigest.Valid() ||
		!filepath.IsAbs(r.root) || filepath.Clean(r.root) != r.root {
		return false
	}
	identity := httpSeedOverlayIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "HTTPSeedOverlayReceipt", Authority: httpSeedOverlayAuthority,
		ExecutionBindingDigest: r.executionBindingDigest.String(), FixtureRecipeDigest: r.fixtureRecipeDigest.String(), Root: r.root,
		Entries: make([]struct {
			Path       string `json:"path"`
			Mode       string `json:"mode"`
			Bytes      int64  `json:"bytes"`
			ByteDigest string `json:"byte_digest"`
		}, len(r.entries)),
	}
	for index, entry := range r.entries {
		if entry.path == "" || entry.mode != string(httpmodel.SeedMode0644) || entry.bytes < 0 || !entry.byteDigest.Valid() ||
			(index > 0 && r.entries[index-1].path >= entry.path) {
			return false
		}
		identity.Entries[index].Path, identity.Entries[index].Mode = entry.path, entry.mode
		identity.Entries[index].Bytes, identity.Entries[index].ByteDigest = entry.bytes, entry.byteDigest.String()
	}
	digestRaw, canonicalBytes, err := canon.DigestTyped("HTTPSeedOverlayReceipt", identity)
	return err == nil && digestRaw.String() == r.digest.String() && bytes.Equal(canonicalBytes, r.canonicalBytes)
}

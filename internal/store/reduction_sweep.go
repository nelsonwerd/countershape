package store

import (
	"bytes"
	"context"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/reduce"
)

const (
	reductionSweepObjectsDirectory = objectDirectory
	reductionSweepTempPrefix       = objectTempPrefix
	codeInvalidReductionSweepStore = "INVALID_REDUCTION_SWEEP_STORE"
	codeInvalidCompletedSweepDraft = "INVALID_COMPLETED_SWEEP_DRAFT"
	codeSweepAuthorityRefused      = "SWEEP_AUTHORITY_REFUSED"
)

// ReductionSweepStore retains U5's typed API while delegating every byte to
// the one shared immutable-object publisher used by later semantic artifacts.
type ReductionSweepStore struct {
	store   *ObjectStore
	objects string // compatibility-visible only to same-package adversarial tests
}

type SweepCompletionAuthority struct {
	storeInstance *objectStoreInstance
	object        ObjectAuthority
	draft         reduce.CompletedSweepDraft
}

func OpenReductionSweepStore(root string) (*ReductionSweepStore, error) {
	objects, err := OpenObjectStore(root)
	if err != nil {
		return nil, err
	}
	return &ReductionSweepStore{store: objects, objects: objects.digestRoot}, nil
}

func (s *ReductionSweepStore) Publish(ctx context.Context, draft reduce.CompletedSweepDraft) (SweepCompletionAuthority, error) {
	object, err := sweepObject(draft)
	if err != nil {
		return SweepCompletionAuthority{}, err
	}
	if s == nil || s.store == nil {
		return SweepCompletionAuthority{}, refuse(codeInvalidReductionSweepStore, "store is zero or uninitialized", nil)
	}
	authority, err := s.store.Publish(ctx, object)
	if err != nil {
		return SweepCompletionAuthority{}, err
	}
	// Re-run the owning parser after generic reopen. The generic store cannot
	// silently become the semantic codec for CompletedSweepDraft.
	parsed, err := reduce.ParseCompletedSweepDraft(authority.object.canonical)
	if err != nil || !sameCompletedSweepDraft(parsed, draft) {
		return SweepCompletionAuthority{}, refuse(codeSweepAuthorityRefused, "published object failed its owning parser", err)
	}
	return SweepCompletionAuthority{storeInstance: s.store.instance, object: authority, draft: parsed}, nil
}

func (s *ReductionSweepStore) Open(ctx context.Context, draft reduce.CompletedSweepDraft) (SweepCompletionAuthority, error) {
	object, err := sweepObject(draft)
	if err != nil {
		return SweepCompletionAuthority{}, err
	}
	if s == nil || s.store == nil {
		return SweepCompletionAuthority{}, refuse(codeInvalidReductionSweepStore, "store is zero or uninitialized", nil)
	}
	authority, err := s.store.Open(ctx, object)
	if err != nil {
		return SweepCompletionAuthority{}, err
	}
	parsed, err := reduce.ParseCompletedSweepDraft(authority.object.canonical)
	if err != nil || !sameCompletedSweepDraft(parsed, draft) {
		return SweepCompletionAuthority{}, refuse(codeSweepAuthorityRefused, "reopened object failed its owning parser", err)
	}
	return SweepCompletionAuthority{storeInstance: s.store.instance, object: authority, draft: parsed}, nil
}

func (s *ReductionSweepStore) Validate(ctx context.Context, draft reduce.CompletedSweepDraft, authority SweepCompletionAuthority) error {
	object, err := sweepObject(draft)
	if err != nil {
		return err
	}
	if s == nil || s.store == nil || authority.storeInstance == nil || authority.storeInstance != s.store.instance ||
		!sameCompletedSweepDraft(authority.draft, draft) {
		return refuse(codeSweepAuthorityRefused, "authority was not issued for this store and exact draft", nil)
	}
	if err := s.store.Validate(ctx, object, authority.object); err != nil {
		return refuse(codeSweepAuthorityRefused, "durable object authority was refused", err)
	}
	parsed, err := reduce.ParseCompletedSweepDraft(object.canonical)
	if err != nil || !sameCompletedSweepDraft(parsed, draft) {
		return refuse(codeSweepAuthorityRefused, "validated object failed its owning parser", err)
	}
	return nil
}

func sweepObject(draft reduce.CompletedSweepDraft) (SemanticObject, error) {
	if err := validateCompletedSweepDraft(draft); err != nil {
		return SemanticObject{}, err
	}
	return NewSemanticObject("CompletedSweepDraft", draft.Digest(), draft.CanonicalBytes())
}

func validateCompletedSweepDraft(draft reduce.CompletedSweepDraft) error {
	body := draft.CanonicalBytes()
	if !draft.Valid() || len(body) == 0 || len(body) > canon.MaxInputBytes {
		return refuse(codeInvalidCompletedSweepDraft, "draft is zero, invalid, or oversized", nil)
	}
	parsed, err := reduce.ParseCompletedSweepDraft(body)
	if err != nil || !sameCompletedSweepDraft(parsed, draft) {
		return refuse(codeInvalidCompletedSweepDraft, "draft does not round-trip through its owning parser", err)
	}
	return nil
}

func sameCompletedSweepDraft(left, right reduce.CompletedSweepDraft) bool {
	return left.Valid() && right.Valid() && left.Digest() == right.Digest() && bytes.Equal(left.CanonicalBytes(), right.CanonicalBytes())
}

// objectPath exists only for same-package filesystem adversarial tests.
func (s *ReductionSweepStore) objectPath(draft reduce.CompletedSweepDraft) string {
	if s == nil || s.store == nil {
		return ""
	}
	object, err := sweepObject(draft)
	if err != nil {
		return ""
	}
	s.store.instance.mu.Lock()
	defer s.store.instance.mu.Unlock()
	path, shard, err := s.store.ensureObjectPath(object.digest)
	if err != nil {
		return ""
	}
	s.objects = shard
	return path
}

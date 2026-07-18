// Package contractmaterialize owns the optional physical edge from one
// freshly reopened terminal residue to its exact six-file contract directory.
// It deliberately accepts no raw or parsed bundle publication authority.
package contractmaterialize

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/nelsonwerd/countershape/internal/domain"
	node "github.com/nelsonwerd/countershape/internal/emit/node"
	"github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/store"
)

const (
	CodeInvalidResidue         = "INVALID_RESIDUE"
	CodeDestinationRefused     = "CONTRACT_DESTINATION_REFUSED"
	CodePublicationUnsupported = "EXPORT_PUBLICATION_UNSUPPORTED"
	CodeExportIncomplete       = "RESIDUE_PERSISTED_EXPORT_INCOMPLETE"
	CodeExportAmbiguous        = "RESIDUE_PERSISTED_EXPORT_AMBIGUOUS"
	StateCreated               = "RESIDUE_PERSISTED_EXPORT_CREATED"
	StateAlreadyExact          = "RESIDUE_PERSISTED_EXPORT_ALREADY_EXACT"

	contractDirectoryMode         = os.FileMode(0o700)
	contractFileMode              = os.FileMode(0o644)
	maximumParentDirectoryEntries = 16_384
	parentDirectoryReadBatch      = 64
	stageAllocationAttempts       = 32
	stageNamePrefix               = ".countershape-contract-stage-"
)

type Error struct {
	Code   string
	Detail string
	Cause  error
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return e.Code + ": " + e.Detail
}

func (e *Error) Unwrap() error { return e.Cause }

func IsCode(err error, code string) bool {
	if err == nil {
		return false
	}
	if typed, ok := err.(*Error); ok && typed.Code == code {
		return true
	}
	if single, ok := err.(interface{ Unwrap() error }); ok && IsCode(single.Unwrap(), code) {
		return true
	}
	if multiple, ok := err.(interface{ Unwrap() []error }); ok {
		for _, child := range multiple.Unwrap() {
			if IsCode(child, code) {
				return true
			}
		}
	}
	return false
}

func refuse(code, detail string, cause error) error {
	return &Error{Code: code, Detail: detail, Cause: cause}
}

type exactObservationUncertain struct{ cause error }

func (e *exactObservationUncertain) Error() string {
	return "exact filesystem observation was uncertain"
}
func (e *exactObservationUncertain) Unwrap() error { return e.cause }

func observationUncertain(err error) error {
	return &exactObservationUncertain{cause: err}
}

func isObservationUncertain(err error) bool {
	var uncertain *exactObservationUncertain
	return errors.As(err, &uncertain)
}

type Disposition string

const (
	Created      Disposition = "CREATED"
	AlreadyExact Disposition = "ALREADY_EXACT"
)

type materializedContractSeal struct{}

var sealedMaterializedContract = &materializedContractSeal{}

// MaterializedContract is an inert live receipt. It is neither a seventh
// bundle member nor semantic, head-transition, target, or execution authority.
type MaterializedContract struct {
	destination  string
	bundleDigest domain.Digest
	headDigest   domain.Digest
	disposition  Disposition
	seal         *materializedContractSeal
}

func (m MaterializedContract) Valid() bool {
	return m.seal == sealedMaterializedContract && m.bundleDigest.Valid() && m.headDigest.Valid() &&
		validDestinationSyntax(m.destination) == nil &&
		(m.disposition == Created || m.disposition == AlreadyExact)
}

func (m MaterializedContract) Destination() string {
	if !m.Valid() {
		return ""
	}
	return m.destination
}

func (m MaterializedContract) BundleDigest() domain.Digest {
	if !m.Valid() {
		return domain.Digest("")
	}
	return m.bundleDigest
}

func (m MaterializedContract) ResidueHeadDigest() domain.Digest {
	if !m.Valid() {
		return domain.Digest("")
	}
	return m.headDigest
}

func (m MaterializedContract) Disposition() Disposition {
	if !m.Valid() {
		return ""
	}
	return m.disposition
}

func (m MaterializedContract) State() string {
	if !m.Valid() {
		return ""
	}
	if m.disposition == Created {
		return StateCreated
	}
	return StateAlreadyExact
}

func (m MaterializedContract) String() string {
	if !m.Valid() {
		return "invalid materialized contract"
	}
	return fmt.Sprintf("%s %s %s", m.disposition, m.bundleDigest, m.destination)
}

// Materialize starts only from a freshly reopened terminal Residue and reopens
// it again immediately before accepting or publishing physical output.
func Materialize(
	ctx context.Context,
	objectStore *store.ObjectStore,
	residue node.Residue,
	destination string,
) (MaterializedContract, error) {
	if ctx == nil || objectStore == nil || !residue.Valid() {
		return MaterializedContract{}, refuse(CodeInvalidResidue, "sealed residue, store, and context are required", nil)
	}
	fresh, err := node.ReopenResidue(ctx, objectStore, residue)
	if err != nil {
		return MaterializedContract{}, refuse(CodeInvalidResidue, "terminal residue did not freshly reopen", err)
	}
	if !nativePublicationSupported() {
		return MaterializedContract{}, incomplete(
			"the local platform/toolchain has no receipted Darwin/arm64+cgo publication edge",
			refuse(CodePublicationUnsupported, "Darwin/arm64 with cgo is required", nil),
		)
	}
	return materializeReopened(ctx, objectStore, fresh, destination, defaultOperations())
}

type materializationSnapshot struct {
	bundle       model.ContractBundle
	bundleDigest domain.Digest
	headDigest   domain.Digest
}

func snapshotFromResidue(residue node.Residue) materializationSnapshot {
	return materializationSnapshot{
		bundle: residue.Bundle(), bundleDigest: residue.BundleDigest(), headDigest: residue.HeadDigest(),
	}
}

func (s materializationSnapshot) valid() bool {
	return s.bundle.Valid() && s.bundleDigest.Valid() && s.headDigest.Valid() && s.bundle.Digest() == s.bundleDigest
}

func sameMaterializationSnapshot(left, right materializationSnapshot) bool {
	return left.valid() && right.valid() && left.bundleDigest == right.bundleDigest &&
		left.headDigest == right.headDigest && bytes.Equal(left.bundle.CanonicalBytes(), right.bundle.CanonicalBytes())
}

type materializationSource struct {
	snapshot     materializationSnapshot
	revalidate   func(context.Context) (materializationSnapshot, error)
	validatePath func(context.Context, string) error
}

func sourceFromResidue(objectStore *store.ObjectStore, residue node.Residue) (materializationSource, error) {
	snapshot := snapshotFromResidue(residue)
	if objectStore == nil || !residue.Valid() || !snapshot.valid() {
		return materializationSource{}, refuse(CodeInvalidResidue, "fresh residue did not expose one exact bundle", nil)
	}
	return materializationSource{
		snapshot: snapshot,
		revalidate: func(ctx context.Context) (materializationSnapshot, error) {
			fresh, err := node.ReopenResidue(ctx, objectStore, residue)
			if err != nil {
				return materializationSnapshot{}, err
			}
			return snapshotFromResidue(fresh), nil
		},
		validatePath: func(ctx context.Context, path string) error {
			return objectStore.ValidateExternalPublicationPath(ctx, path)
		},
	}, nil
}

func (s materializationSource) valid() bool {
	return s.snapshot.valid() && s.revalidate != nil && s.validatePath != nil
}

func (s materializationSource) reopen(ctx context.Context) (materializationSnapshot, error) {
	if !s.valid() {
		return materializationSnapshot{}, refuse(CodeInvalidResidue, "materialization source is incomplete", nil)
	}
	fresh, err := s.revalidate(ctx)
	if err != nil {
		return materializationSnapshot{}, err
	}
	if !sameMaterializationSnapshot(s.snapshot, fresh) {
		return materializationSnapshot{}, refuse(CodeInvalidResidue, "terminal residue changed during materialization", nil)
	}
	return fresh, nil
}

type durableFile interface {
	io.Reader
	io.Writer
	Stat() (os.FileInfo, error)
	Chmod(os.FileMode) error
	Sync() error
	Close() error
}

type durableDirectory interface {
	Stat() (os.FileInfo, error)
	ReadDir(int) ([]os.DirEntry, error)
	Chmod(os.FileMode) error
	Sync() error
	Close() error
	Fd() uintptr
}

type operations struct {
	openDirectoryPath func(string) (durableDirectory, error)
	openDirectoryAt   func(durableDirectory, string) (durableDirectory, error)
	openFileAt        func(durableDirectory, string, int, os.FileMode) (durableFile, error)
	mkdirAt           func(durableDirectory, string, os.FileMode) error
	unlinkAt          func(durableDirectory, string, bool) error
	renameExclusiveAt func(durableDirectory, string, string) error
}

func defaultOperations() operations {
	return operations{
		openDirectoryPath: openDirectoryPathNoFollow,
		openDirectoryAt:   openDirectoryAtNoFollow,
		openFileAt:        openFileAtNoFollow,
		mkdirAt:           mkdirDirectoryAt,
		unlinkAt:          unlinkEntryAt,
		renameExclusiveAt: renameExclusiveDirectoryAt,
	}
}

func (o operations) valid() bool {
	return o.openDirectoryPath != nil && o.openDirectoryAt != nil && o.openFileAt != nil &&
		o.mkdirAt != nil && o.unlinkAt != nil && o.renameExclusiveAt != nil
}

type retainedParent struct {
	path   string
	handle durableDirectory
	info   os.FileInfo
}

type retainedStage struct {
	name   string
	handle durableDirectory
	info   os.FileInfo
}

func materializeReopened(
	ctx context.Context,
	objectStore *store.ObjectStore,
	residue node.Residue,
	destination string,
	ops operations,
) (receipt MaterializedContract, resultErr error) {
	source, err := sourceFromResidue(objectStore, residue)
	if err != nil {
		return MaterializedContract{}, incomplete("fresh residue could not narrow physical publication authority", err)
	}
	return materializeAuthorized(ctx, source, destination, ops)
}

func materializeAuthorized(
	ctx context.Context,
	source materializationSource,
	destination string,
	ops operations,
) (receipt MaterializedContract, resultErr error) {
	if !ops.valid() {
		return MaterializedContract{}, incomplete("materializer operations are incomplete", nil)
	}
	if !source.valid() {
		return MaterializedContract{}, incomplete("materialization source is incomplete", nil)
	}
	if err := contextRefusal(ctx); err != nil {
		return MaterializedContract{}, incomplete("materialization was cancelled before destination inspection", err)
	}
	if err := validDestinationSyntax(destination); err != nil {
		return MaterializedContract{}, incomplete("destination syntax was refused", err)
	}
	parentPath := filepath.Dir(destination)
	if err := source.validatePath(ctx, destination); err != nil {
		return MaterializedContract{}, incomplete("destination overlaps the durable object store", err)
	}
	if err := source.validatePath(ctx, parentPath); err != nil {
		return MaterializedContract{}, incomplete("destination parent overlaps the durable object store", err)
	}
	parent, err := retainParent(parentPath, ops)
	if err != nil {
		return MaterializedContract{}, incomplete("destination parent was refused", err)
	}
	renamed := false
	outputEstablished := false
	defer func() {
		if closeErr := parent.handle.Close(); closeErr != nil {
			receipt = MaterializedContract{}
			var classified error
			if renamed || outputEstablished {
				classified = ambiguous("retained destination-parent close failed after publication", closeErr)
			} else {
				classified = incomplete("retained destination-parent close failed", closeErr)
			}
			if resultErr != nil {
				resultErr = errors.Join(resultErr, classified)
			} else {
				resultErr = classified
			}
		}
	}()
	destinationName := filepath.Base(destination)
	bundle := source.snapshot.bundle

	existing, stateErr := openExactNamedDirectory(parent.handle, destinationName, ops)
	if stateErr != nil {
		return MaterializedContract{}, incomplete("destination name or type was refused", stateErr)
	}
	if existing != nil {
		receipt, resultErr = acceptExisting(ctx, source, bundle, destination, parent, existing, ops)
		outputEstablished = resultErr == nil && receipt.Valid()
		return receipt, resultErr
	}

	stage, err := allocateStage(parent.handle, ops)
	if err != nil {
		return MaterializedContract{}, incomplete("private staging directory allocation failed", err)
	}
	stageOwned := true
	defer func() {
		if stage.handle == nil {
			return
		}
		if stageOwned {
			if cleanupErr := cleanupStage(parent.handle, stage, bundle, ops); cleanupErr != nil {
				receipt = MaterializedContract{}
				resultErr = incomplete(
					"private staging cleanup could not prove durable exact removal",
					errors.Join(resultErr, cleanupErr),
				)
				return
			}
			stage.handle = nil
			return
		}
		if closeErr := stage.handle.Close(); closeErr != nil {
			receipt = MaterializedContract{}
			var classified error
			if renamed {
				classified = ambiguous("published contract-directory handle close failed", closeErr)
			} else {
				classified = incomplete("staging-directory handle close failed", closeErr)
			}
			if resultErr != nil {
				resultErr = errors.Join(resultErr, classified)
			} else {
				resultErr = classified
			}
		}
	}()
	if err := validateParentPathIdentity(parent, ops); err != nil {
		return MaterializedContract{}, incomplete("destination parent changed after staging allocation", err)
	}
	if err := writeExactBundle(ctx, stage.handle, bundle, ops); err != nil {
		return MaterializedContract{}, incomplete("exact bundle staging failed", err)
	}
	if err := verifyExactDirectoryHandle(stage.handle, bundle, ops); err != nil {
		return MaterializedContract{}, incomplete("staged contract verification failed", err)
	}
	if err := stage.handle.Sync(); err != nil {
		return MaterializedContract{}, incomplete("staging directory sync failed", err)
	}
	if err := contextRefusal(ctx); err != nil {
		return MaterializedContract{}, incomplete("materialization was cancelled before terminal revalidation", err)
	}
	second, err := source.reopen(ctx)
	if err != nil {
		return MaterializedContract{}, incomplete("terminal residue changed before output publication", err)
	}
	if err := source.validatePath(ctx, destination); err != nil {
		return MaterializedContract{}, incomplete("destination overlap changed before publication", err)
	}
	if err := validateParentPathIdentity(parent, ops); err != nil {
		return MaterializedContract{}, incomplete("destination parent changed before publication", err)
	}
	existing, stateErr = openExactNamedDirectory(parent.handle, destinationName, ops)
	if stateErr != nil {
		return MaterializedContract{}, incomplete("destination changed before publication", stateErr)
	}
	if existing != nil {
		nextSource := source
		nextSource.snapshot = second
		receipt, resultErr = acceptExisting(ctx, nextSource, bundle, destination, parent, existing, ops)
		outputEstablished = resultErr == nil && receipt.Valid()
		return receipt, resultErr
	}
	if err := contextRefusal(ctx); err != nil {
		return MaterializedContract{}, incomplete("materialization was cancelled at the publication boundary", err)
	}

	renameErr := ops.renameExclusiveAt(parent.handle, stage.name, destinationName)
	if renameErr != nil {
		existing, inspectErr := openExactNamedDirectory(parent.handle, destinationName, ops)
		if inspectErr == nil && existing != nil {
			receipt, resultErr = acceptAlreadyReopened(second, bundle, destination, parent, existing, ops)
			if closeErr := existing.Close(); closeErr != nil {
				receipt = MaterializedContract{}
				classified := ambiguous("collision destination handle close failed", closeErr)
				if resultErr == nil {
					resultErr = classified
				} else {
					resultErr = errors.Join(resultErr, classified)
				}
			}
			outputEstablished = resultErr == nil && receipt.Valid()
			return receipt, resultErr
		}
		stageStillNamed, stageProbeErr := retainedStageStillNamed(parent.handle, stage, ops)
		if stageProbeErr == nil && stageStillNamed {
			return MaterializedContract{}, incomplete(
				"exclusive no-follow directory publication failed before consuming the retained stage",
				errors.Join(renameErr, inspectErr),
			)
		}
		stageOwned = false
		renamed = true
		return MaterializedContract{}, ambiguous(
			"exclusive publication outcome could not be reconciled to an exact destination or retained stage",
			errors.Join(renameErr, inspectErr, stageProbeErr),
		)
	}
	renamed = true
	stageOwned = false

	// The destination may now be durable. Ignore cancellation while syncing and
	// reconciling through the retained parent descriptor.
	if err := stage.handle.Sync(); err != nil {
		return MaterializedContract{}, ambiguous("renamed contract directory sync failed", err)
	}
	if err := parent.handle.Sync(); err != nil {
		return MaterializedContract{}, ambiguous("destination rename succeeded but parent sync failed", err)
	}
	if err := validateParentPathIdentity(parent, ops); err != nil {
		return MaterializedContract{}, ambiguous("destination parent path changed after publication", err)
	}
	final, err := requireNamedDirectory(parent.handle, destinationName, ops)
	if err != nil {
		return MaterializedContract{}, ambiguous("published destination could not be reopened by retained parent", err)
	}
	finalInfo, statErr := final.Stat()
	if statErr != nil || !os.SameFile(stage.info, finalInfo) {
		return MaterializedContract{}, ambiguous(
			"published destination identity differs from the retained stage", errors.Join(statErr, final.Close()),
		)
	}
	if err := verifyExactDirectoryHandle(final, bundle, ops); err != nil {
		return MaterializedContract{}, ambiguous(
			"published destination did not reopen exactly", errors.Join(err, final.Close()),
		)
	}
	if err := final.Close(); err != nil {
		return MaterializedContract{}, ambiguous("published destination reopen handle close failed", err)
	}
	last, err := requireNamedDirectory(parent.handle, destinationName, ops)
	if err != nil {
		return MaterializedContract{}, ambiguous("published destination final name reopen failed", err)
	}
	lastInfo, statErr := last.Stat()
	if statErr != nil || !os.SameFile(stage.info, lastInfo) {
		return MaterializedContract{}, ambiguous(
			"published destination final name identity changed", errors.Join(statErr, last.Close()),
		)
	}
	if err := verifyExactDirectoryHandle(last, bundle, ops); err != nil {
		return MaterializedContract{}, ambiguous(
			"published destination final name verification failed", errors.Join(err, last.Close()),
		)
	}
	if err := last.Close(); err != nil {
		return MaterializedContract{}, ambiguous("published destination final verification handle close failed", err)
	}
	receipt = newReceipt(second, destination, Created)
	if !receipt.Valid() {
		return MaterializedContract{}, ambiguous("published destination produced an invalid receipt", nil)
	}
	return receipt, nil
}

func retainedStageStillNamed(parent durableDirectory, stage retainedStage, ops operations) (bool, error) {
	current, err := openExactNamedDirectory(parent, stage.name, ops)
	if err != nil {
		return false, err
	}
	if current == nil {
		return false, nil
	}
	info, statErr := current.Stat()
	closeErr := current.Close()
	if statErr != nil || closeErr != nil {
		return false, errors.Join(statErr, closeErr)
	}
	return os.SameFile(stage.info, info), nil
}

func acceptExisting(
	ctx context.Context,
	source materializationSource,
	bundle model.ContractBundle,
	destination string,
	parent retainedParent,
	existing durableDirectory,
	ops operations,
) (MaterializedContract, error) {
	receipt, resultErr := acceptExistingOpen(ctx, source, bundle, destination, parent, existing, ops)
	closeErr := existing.Close()
	if closeErr == nil {
		return receipt, resultErr
	}
	if resultErr != nil {
		return MaterializedContract{}, errors.Join(
			resultErr, ambiguous("existing destination handle close failed", closeErr),
		)
	}
	return MaterializedContract{}, ambiguous("existing destination handle close failed", closeErr)
}

func acceptExistingOpen(
	ctx context.Context,
	source materializationSource,
	bundle model.ContractBundle,
	destination string,
	parent retainedParent,
	existing durableDirectory,
	ops operations,
) (MaterializedContract, error) {
	if err := verifyExactDirectoryHandle(existing, bundle, ops); err != nil {
		if isObservationUncertain(err) {
			return MaterializedContract{}, ambiguous("existing destination exactness could not be durably observed", err)
		}
		return MaterializedContract{}, incomplete("existing destination is not the exact contract", err)
	}
	if err := contextRefusal(ctx); err != nil {
		return MaterializedContract{}, incomplete("materialization was cancelled before existing-output revalidation", err)
	}
	second, err := source.reopen(ctx)
	if err != nil {
		return MaterializedContract{}, incomplete("terminal residue changed before accepting existing output", err)
	}
	if err := source.validatePath(ctx, destination); err != nil {
		return MaterializedContract{}, incomplete("existing destination overlap changed", err)
	}
	if err := validateParentPathIdentity(parent, ops); err != nil {
		return MaterializedContract{}, incomplete("destination parent changed before accepting existing output", err)
	}
	return acceptAlreadyReopened(second, bundle, destination, parent, existing, ops)
}

func acceptAlreadyReopened(
	residue materializationSnapshot,
	bundle model.ContractBundle,
	destination string,
	parent retainedParent,
	existing durableDirectory,
	ops operations,
) (MaterializedContract, error) {
	before, err := existing.Stat()
	if err != nil {
		return MaterializedContract{}, ambiguous("existing destination identity is unavailable", err)
	}
	if err := verifyExactDirectoryHandle(existing, bundle, ops); err != nil {
		if isObservationUncertain(err) {
			return MaterializedContract{}, ambiguous("existing destination exactness could not be durably observed", err)
		}
		return MaterializedContract{}, incomplete("existing destination is not exact", err)
	}
	if err := existing.Sync(); err != nil {
		return MaterializedContract{}, ambiguous("exact existing destination directory sync failed", err)
	}
	if err := parent.handle.Sync(); err != nil {
		return MaterializedContract{}, ambiguous("exact existing destination parent sync failed", err)
	}
	if err := validateParentPathIdentity(parent, ops); err != nil {
		return MaterializedContract{}, ambiguous("destination parent path changed during existing-output acceptance", err)
	}
	reopened, err := requireNamedDirectory(parent.handle, filepath.Base(destination), ops)
	if err != nil {
		return MaterializedContract{}, ambiguous("exact existing destination could not be reopened", err)
	}
	receipt, resultErr := acceptReopenedDirectory(residue, bundle, destination, before, reopened, ops)
	closeErr := reopened.Close()
	if resultErr != nil {
		if closeErr != nil {
			resultErr = errors.Join(
				resultErr, ambiguous("exact existing destination reopen handle close failed", closeErr),
			)
		}
		return MaterializedContract{}, resultErr
	}
	if closeErr != nil {
		return MaterializedContract{}, ambiguous("exact existing destination reopen handle close failed", closeErr)
	}
	return receipt, nil
}

func acceptReopenedDirectory(
	residue materializationSnapshot,
	bundle model.ContractBundle,
	destination string,
	before os.FileInfo,
	reopened durableDirectory,
	ops operations,
) (MaterializedContract, error) {
	after, statErr := reopened.Stat()
	if statErr != nil || !os.SameFile(before, after) {
		return MaterializedContract{}, ambiguous("exact existing destination identity changed", statErr)
	}
	if err := verifyExactDirectoryHandle(reopened, bundle, ops); err != nil {
		return MaterializedContract{}, ambiguous("exact existing output changed during acceptance", err)
	}
	receipt := newReceipt(residue, destination, AlreadyExact)
	if !receipt.Valid() {
		return MaterializedContract{}, ambiguous("exact existing destination produced an invalid receipt", nil)
	}
	return receipt, nil
}

func newReceipt(residue materializationSnapshot, destination string, disposition Disposition) MaterializedContract {
	return MaterializedContract{
		destination: destination, bundleDigest: residue.bundleDigest, headDigest: residue.headDigest,
		disposition: disposition, seal: sealedMaterializedContract,
	}
}

func retainParent(path string, ops operations) (retainedParent, error) {
	handle, err := ops.openDirectoryPath(path)
	if err != nil {
		return retainedParent{}, refuse(CodeDestinationRefused, "destination parent cannot be opened without following links", err)
	}
	info, statErr := handle.Stat()
	if statErr != nil || info == nil || !info.IsDir() || hasSpecialMode(info.Mode()) ||
		!trustedPublicationParent(info) {
		return retainedParent{}, refuse(
			CodeDestinationRefused,
			"destination parent is not one caller-owned directory without group or other write authority",
			errors.Join(statErr, handle.Close()),
		)
	}
	parent := retainedParent{path: path, handle: handle, info: info}
	if err := validateParentPathIdentity(parent, ops); err != nil {
		return retainedParent{}, errors.Join(err, handle.Close())
	}
	return parent, nil
}

func validateParentPathIdentity(parent retainedParent, ops operations) error {
	reopened, err := ops.openDirectoryPath(parent.path)
	if err != nil {
		return err
	}
	info, statErr := reopened.Stat()
	closeErr := reopened.Close()
	if statErr != nil || closeErr != nil || info == nil || !info.IsDir() || !os.SameFile(parent.info, info) ||
		hasSpecialMode(info.Mode()) || !trustedPublicationParent(info) {
		return errors.Join(statErr, closeErr, errors.New("destination parent path no longer names the retained directory"))
	}
	return nil
}

func allocateStage(parent durableDirectory, ops operations) (retainedStage, error) {
	for attempt := 0; attempt < stageAllocationAttempts; attempt++ {
		random := make([]byte, 16)
		if _, err := rand.Read(random); err != nil {
			return retainedStage{}, err
		}
		name := stageNamePrefix + hex.EncodeToString(random)
		if err := ops.mkdirAt(parent, name, contractDirectoryMode); err != nil {
			if errors.Is(err, os.ErrExist) {
				continue
			}
			return retainedStage{}, err
		}
		handle, err := requireNamedDirectory(parent, name, ops)
		if err != nil {
			return retainedStage{}, errors.Join(err, errors.New("new staging directory could not be retained for safe cleanup"))
		}
		before, statErr := handle.Stat()
		if statErr != nil || before == nil || !before.IsDir() || !ownedByEffectiveUser(before) {
			_ = handle.Close()
			return retainedStage{}, errors.Join(statErr, errors.New("new staging directory identity is unavailable"))
		}
		stage := retainedStage{name: name, handle: handle, info: before}
		if err := handle.Chmod(contractDirectoryMode); err != nil {
			return retainedStage{}, errors.Join(err, cleanupStage(parent, stage, model.ContractBundle{}, ops))
		}
		info, err := handle.Stat()
		if err != nil || info == nil || !os.SameFile(before, info) || !ownedByEffectiveUser(info) ||
			!exactPhysicalMode(info.Mode(), contractDirectoryMode) {
			return retainedStage{}, errors.Join(
				err, errors.New("staging directory identity or mode differs"),
				cleanupStage(parent, stage, model.ContractBundle{}, ops),
			)
		}
		stage.info = info
		return stage, nil
	}
	return retainedStage{}, errors.New("private staging name allocation exhausted")
}

// openExactNamedDirectory enumerates the retained parent before opening the
// requested name. If the filesystem resolves a case or normalization alias,
// openat can succeed but the exact stored name will be absent; that is refused.
func openExactNamedDirectory(parent durableDirectory, name string, ops operations) (durableDirectory, error) {
	exactName, err := directoryContainsExactName(parent, name, ops)
	if err != nil {
		return nil, err
	}
	handle, openErr := ops.openDirectoryAt(parent, name)
	if !exactName {
		if openErr == nil {
			_ = handle.Close()
			return nil, refuse(CodeDestinationRefused, "destination resolves through a filesystem name alias", nil)
		}
		if errors.Is(openErr, os.ErrNotExist) {
			return nil, nil
		}
		return nil, refuse(CodeDestinationRefused, "destination has a non-directory or aliased entry", openErr)
	}
	if openErr != nil {
		return nil, refuse(CodeDestinationRefused, "exact destination name is not one real directory", openErr)
	}
	return handle, nil
}

func requireNamedDirectory(parent durableDirectory, name string, ops operations) (durableDirectory, error) {
	handle, err := openExactNamedDirectory(parent, name, ops)
	if err != nil {
		return nil, err
	}
	if handle == nil {
		return nil, os.ErrNotExist
	}
	return handle, nil
}

func directoryContainsExactName(parent durableDirectory, name string, ops operations) (bool, error) {
	reader, err := ops.openDirectoryAt(parent, ".")
	if err != nil {
		return false, err
	}
	present, scanErr := scanDirectoryForExactName(reader, name)
	closeErr := reader.Close()
	if scanErr != nil || closeErr != nil {
		return false, errors.Join(scanErr, closeErr)
	}
	return present, nil
}

func scanDirectoryForExactName(reader durableDirectory, name string) (present bool, resultErr error) {
	seen := 0
	for {
		entries, readErr := reader.ReadDir(parentDirectoryReadBatch)
		seen += len(entries)
		if seen > maximumParentDirectoryEntries {
			return false, refuse(CodeDestinationRefused, "destination parent roster exceeds the reviewed bound", nil)
		}
		for _, entry := range entries {
			if entry.Name() == name {
				present = true
				continue
			}
			if strings.EqualFold(entry.Name(), name) {
				return false, refuse(CodeDestinationRefused, "destination has an alternate-case sibling", nil)
			}
		}
		if errors.Is(readErr, io.EOF) {
			return present, nil
		}
		if readErr != nil {
			return false, readErr
		}
		if len(entries) == 0 {
			return false, io.ErrNoProgress
		}
	}
}

func writeExactBundle(ctx context.Context, stage durableDirectory, bundle model.ContractBundle, ops operations) error {
	files := bundle.Files()
	if len(files) != model.ContractFileCount {
		return refuse(CodeDestinationRefused, "bundle file roster is not exact", nil)
	}
	for _, contractFile := range files {
		if err := contextRefusal(ctx); err != nil {
			return err
		}
		if filepath.Base(contractFile.Path()) != contractFile.Path() || contractFile.Mode() != model.ContractFileModeV1 {
			return refuse(CodeDestinationRefused, "bundle file path or mode escaped the fixed policy", nil)
		}
		handle, err := ops.openFileAt(stage, contractFile.Path(), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return refuse(CodeDestinationRefused, "exclusive staged file creation failed", err)
		}
		if err := writeAndCloseExact(handle, contractFile.Content()); err != nil {
			return refuse(CodeDestinationRefused, "staged file write or durability failed", err)
		}
		if err := verifyExactFileAt(stage, contractFile, ops); err != nil {
			return err
		}
	}
	return nil
}

func writeAndCloseExact(handle durableFile, exact []byte) error {
	closed := false
	defer func() {
		if !closed {
			_ = handle.Close()
		}
	}()
	written := 0
	for written < len(exact) {
		count, err := handle.Write(exact[written:])
		if count < 0 || count > len(exact)-written {
			return io.ErrShortWrite
		}
		written += count
		if err != nil {
			return err
		}
		if count == 0 {
			return io.ErrNoProgress
		}
	}
	if err := handle.Chmod(contractFileMode); err != nil {
		return err
	}
	if err := handle.Sync(); err != nil {
		return err
	}
	closeErr := handle.Close()
	closed = true
	if closeErr != nil {
		return closeErr
	}
	return nil
}

func verifyExactDirectoryHandle(directory durableDirectory, bundle model.ContractBundle, ops operations) error {
	info, err := directory.Stat()
	if err != nil {
		return observationUncertain(refuse(CodeDestinationRefused, "contract directory facts could not be observed", err))
	}
	if info == nil || !info.IsDir() || !ownedByEffectiveUser(info) ||
		!exactPhysicalMode(info.Mode(), contractDirectoryMode) {
		return refuse(CodeDestinationRefused, "contract directory type or exact mode differs", nil)
	}
	files := bundle.Files()
	if _, err := readExactRoster(directory, files, ops); err != nil {
		return err
	}
	for _, contractFile := range files {
		if err := verifyExactFileAt(directory, contractFile, ops); err != nil {
			return err
		}
	}
	if _, err := readExactRoster(directory, files, ops); err != nil {
		return refuse(CodeDestinationRefused, "contract directory roster changed during exact member verification", err)
	}
	return nil
}

func readExactRoster(
	directory durableDirectory,
	files []model.ContractFile,
	ops operations,
) (map[string]struct{}, error) {
	reader, err := ops.openDirectoryAt(directory, ".")
	if err != nil {
		return nil, observationUncertain(refuse(
			CodeDestinationRefused, "contract directory cannot be duplicated without following links", err,
		))
	}
	entries, readErr := readBoundedDirectory(reader, model.ContractFileCount)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		return nil, observationUncertain(refuse(
			CodeDestinationRefused, "contract directory roster cannot be read exactly", errors.Join(readErr, closeErr),
		))
	}
	if len(entries) != len(files) {
		return nil, refuse(CodeDestinationRefused, "contract directory has a missing or extra member", nil)
	}
	entryNames := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		entryNames[entry.Name()] = struct{}{}
	}
	for _, contractFile := range files {
		if _, present := entryNames[contractFile.Path()]; !present {
			return nil, refuse(CodeDestinationRefused, "contract directory roster differs from the fixed six names", nil)
		}
	}
	return entryNames, nil
}

func verifyExactFileAt(directory durableDirectory, expected model.ContractFile, ops operations) error {
	first, err := ops.openFileAt(directory, expected.Path(), os.O_RDONLY, 0)
	if err != nil {
		return observationUncertain(refuse(
			CodeDestinationRefused, "contract member cannot be opened without following links", err,
		))
	}
	firstBody, firstInfo, err := readExactFile(first, expected)
	if err != nil {
		return err
	}
	reopened, err := ops.openFileAt(directory, expected.Path(), os.O_RDONLY, 0)
	if err != nil {
		return observationUncertain(refuse(CodeDestinationRefused, "contract member final reopen failed", err))
	}
	secondBody, secondInfo, err := readExactFile(reopened, expected)
	if err != nil {
		return err
	}
	if !os.SameFile(firstInfo, secondInfo) || !bytes.Equal(firstBody, secondBody) {
		return refuse(CodeDestinationRefused, "contract member identity or bytes changed between exact reads", nil)
	}
	return nil
}

func readExactFile(handle durableFile, expected model.ContractFile) ([]byte, os.FileInfo, error) {
	before, statErr := handle.Stat()
	if statErr != nil {
		return nil, nil, observationUncertain(refuse(
			CodeDestinationRefused, "contract member facts could not be observed", errors.Join(statErr, handle.Close()),
		))
	}
	if !exactFileFacts(before, expected) {
		closeErr := handle.Close()
		if closeErr != nil {
			return nil, nil, observationUncertain(refuse(
				CodeDestinationRefused, "nonexact contract member handle could not be closed", closeErr,
			))
		}
		return nil, nil, refuse(
			CodeDestinationRefused, "contract member type, mode, count, or link facts differ",
			nil,
		)
	}
	exact, readErr := io.ReadAll(io.LimitReader(handle, int64(expected.ByteCount())+1))
	after, afterErr := handle.Stat()
	syncErr := handle.Sync()
	closeErr := handle.Close()
	if readErr != nil || afterErr != nil || syncErr != nil || closeErr != nil {
		return nil, nil, observationUncertain(refuse(
			CodeDestinationRefused, "contract member could not be durably observed",
			errors.Join(readErr, afterErr, syncErr, closeErr),
		))
	}
	if !os.SameFile(before, after) || !exactFileFacts(after, expected) || len(exact) != expected.ByteCount() {
		return nil, nil, refuse(
			CodeDestinationRefused, "contract member changed during bounded read",
			nil,
		)
	}
	digest := sha256.Sum256(exact)
	if expected.ByteSHA256().String() != "sha256:"+hex.EncodeToString(digest[:]) || !bytes.Equal(exact, expected.Content()) {
		return nil, nil, refuse(CodeDestinationRefused, "contract member bytes or raw digest differ", nil)
	}
	return exact, after, nil
}

func exactFileFacts(info os.FileInfo, expected model.ContractFile) bool {
	if info == nil || !info.Mode().IsRegular() || !ownedByEffectiveUser(info) ||
		!exactPhysicalMode(info.Mode(), contractFileMode) ||
		info.Size() != int64(expected.ByteCount()) {
		return false
	}
	links, ok := physicalLinkCount(info)
	return ok && links == 1
}

func exactPhysicalMode(actual, expected os.FileMode) bool {
	return actual.Perm() == expected.Perm() && !hasSpecialMode(actual)
}

func hasSpecialMode(mode os.FileMode) bool {
	return mode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0
}

func readBoundedDirectory(handle durableDirectory, maximum int) ([]os.DirEntry, error) {
	entries := make([]os.DirEntry, 0, maximum)
	for {
		remaining := maximum + 1 - len(entries)
		if remaining < 1 {
			remaining = 1
		}
		batch, err := handle.ReadDir(remaining)
		entries = append(entries, batch...)
		if len(entries) > maximum {
			return nil, errors.New("directory roster exceeds exact bound")
		}
		if errors.Is(err, io.EOF) {
			return entries, nil
		}
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			return nil, io.ErrNoProgress
		}
	}
}

func cleanupStage(parent durableDirectory, stage retainedStage, bundle model.ContractBundle, ops operations) (resultErr error) {
	stageClosed := false
	defer func() {
		if !stageClosed {
			resultErr = errors.Join(resultErr, stage.handle.Close())
		}
	}()
	current, err := requireNamedDirectory(parent, stage.name, ops)
	if err != nil {
		return err
	}
	currentInfo, statErr := current.Stat()
	closeErr := current.Close()
	if statErr != nil || closeErr != nil || !os.SameFile(stage.info, currentInfo) {
		return errors.Join(statErr, closeErr, errors.New("staging name no longer binds the retained directory"))
	}
	entries, err := requireSafeCleanupRoster(stage.handle, bundle, ops)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := ops.unlinkAt(stage.handle, entry, false); err != nil {
			return err
		}
	}
	if err := stage.handle.Sync(); err != nil {
		return err
	}
	bound, err := requireNamedDirectory(parent, stage.name, ops)
	if err != nil {
		return err
	}
	boundInfo, statErr := bound.Stat()
	boundCloseErr := bound.Close()
	if statErr != nil || boundCloseErr != nil || !os.SameFile(stage.info, boundInfo) {
		return errors.Join(statErr, boundCloseErr, errors.New("staging name changed before directory removal"))
	}
	if err := ops.unlinkAt(parent, stage.name, true); err != nil {
		return err
	}
	if err := stage.handle.Close(); err != nil {
		stageClosed = true
		return err
	}
	stageClosed = true
	if err := parent.Sync(); err != nil {
		return err
	}
	if handle, err := openExactNamedDirectory(parent, stage.name, ops); err != nil {
		return err
	} else if handle != nil {
		_ = handle.Close()
		return errors.New("staging directory remains after cleanup")
	}
	return nil
}

func requireSafeCleanupRoster(stage durableDirectory, bundle model.ContractBundle, ops operations) ([]string, error) {
	reader, err := ops.openDirectoryAt(stage, ".")
	if err != nil {
		return nil, err
	}
	entries, readErr := readBoundedDirectory(reader, model.ContractFileCount)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr, errors.New("staging cleanup roster is unavailable"))
	}
	want := make(map[string]struct{}, model.ContractFileCount)
	for _, file := range bundle.Files() {
		want[file.Path()] = struct{}{}
	}
	for _, entry := range entries {
		if _, ok := want[entry.Name()]; !ok {
			return nil, errors.New("staging cleanup roster contains an unknown entry")
		}
	}
	names := make([]string, len(entries))
	for index, entry := range entries {
		names[index] = entry.Name()
	}
	return names, nil
}

func validDestinationSyntax(destination string) error {
	if destination == "" || !filepath.IsAbs(destination) || filepath.Clean(destination) != destination ||
		filepath.Dir(destination) == destination || filepath.Base(destination) == "." || filepath.Base(destination) == ".." {
		return refuse(CodeDestinationRefused, "destination must be non-root, clean, and absolute", nil)
	}
	for _, character := range destination {
		if unicode.IsControl(character) {
			return refuse(CodeDestinationRefused, "destination contains control text", nil)
		}
	}
	return nil
}

func contextRefusal(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is nil")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func incomplete(detail string, cause error) error {
	return refuse(CodeExportIncomplete, detail, cause)
}

func ambiguous(detail string, cause error) error {
	return refuse(CodeExportAmbiguous, detail, cause)
}

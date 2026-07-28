#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { readFile, readdir, lstat } from "node:fs/promises";
import { dirname, isAbsolute, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { TextDecoder } from "node:util";

const checkerPath = fileURLToPath(import.meta.url);
const repositoryRoot = resolve(dirname(checkerPath), "..");
const modulePath = "github.com/nelsonwerd/countershape";
const packagePath = `${modulePath}/internal/contractexec/model`;
const contractPackagePath = `${modulePath}/internal/contractexec`;
const gitPackagePath = `${modulePath}/internal/gitobj`;
const hostEpochPackagePath = `${modulePath}/internal/hostepoch`;
const nodeRuntimePackagePath = `${modulePath}/internal/noderuntime`;
const processMechanicsPackagePath = `${modulePath}/internal/processmechanics`;
const contractRunnerPackagePath = `${modulePath}/internal/contractexec/runner`;
const storePackagePath = `${modulePath}/internal/store`;
const contractCLITestPackagePath = `${modulePath}/testkit/contractexec/cli`;
const goExecutable = process.env.COUNTERSHAPE_GO ?? "/opt/homebrew/bin/go";

const expectedC2ProductionFiles = Object.freeze([
	"contract_run_bridge.go", "execution_interlock.go", "head.go", "head_darwin.go", "nonhead_contract.go",
	"object_store.go", "private_contract_run.go", "reduction_sweep.go",
]);
const expectedC2TestFiles = Object.freeze([
	"contract_run_bridge_test.go", "execution_interlock_test.go", "head_test.go", "nonhead_contract_test.go",
	"object_store_test.go", "private_contract_run_test.go", "reduction_sweep_test.go",
]);
const expectedC2XTestFiles = Object.freeze(["public_api_test.go"]);
const expectedC2Imports = Object.freeze({
	"contract_run_bridge.go": Object.freeze([
		"bytes", "context", "errors", `${modulePath}/internal/canon`, `${modulePath}/internal/contractexec/model`,
		`${modulePath}/internal/domain`, `${modulePath}/internal/hostepoch`, "path/filepath", "sync",
	]),
	"execution_interlock.go": Object.freeze([
		"bytes", "context", "errors", "fmt", `${modulePath}/internal/canon`,
		`${modulePath}/internal/domain`, "os", "path/filepath", "sync",
	]),
	"nonhead_contract.go": Object.freeze([
		"bytes", "context", "crypto/rand", "encoding/hex", "errors", "fmt", `${modulePath}/internal/canon`,
		`${modulePath}/internal/contractexec/model`, `${modulePath}/internal/domain`, "io", "os", "path/filepath", "sort", "strings",
	]),
	"object_store.go": Object.freeze([
		"bytes", "context", "errors", "fmt", `${modulePath}/internal/canon`,
		`${modulePath}/internal/domain`, "io", "os", "path/filepath", "strings", "sync", "unicode/utf8",
	]),
	"private_contract_run.go": Object.freeze([
		"bytes", "context", "errors", "fmt", `${modulePath}/internal/canon`,
		`${modulePath}/internal/domain`, "io", "os", "path/filepath", "sort", "strings", "unicode/utf8",
	]),
});
const expectedC2PackageImports = Object.freeze([
	"bytes", "context", "crypto/rand", "encoding/hex", "encoding/json", "errors", "fmt", `${modulePath}/internal/canon`,
	`${modulePath}/internal/choice/promotion/authority`, `${modulePath}/internal/compare`,
	`${modulePath}/internal/confirmation/authority`, `${modulePath}/internal/contractexec/model`, `${modulePath}/internal/domain`,
	`${modulePath}/internal/emit/node/authority`, `${modulePath}/internal/hostepoch`, `${modulePath}/internal/reduce`, "io", "os",
	"path/filepath", "sort", "strconv", "strings", "sync", "syscall", "time", "unicode", "unicode/utf8",
]);
const c2NonheadTests = Object.freeze([
	"TestC2FixturePublishesAndReopensExactNonheadObjects",
	"TestC2ExactKeyMappingsConvergeOnlyForTypedParents",
	"TestC2MappingsRejectWrongKindCrossStoreAndAlternateLinks",
	"TestC2CopiedOrParsedBodiesCannotEnterProductionMechanics",
	"TestC2NonheadFaultMatrixSeparatesNoEffectAndUnknownEffect",
	"TestC2NonheadPublicationNeverMutatesStudyHead",
]);
const c2InterlockTests = Object.freeze([
	"TestC2InterlockClaimMultiProcessRaceHasOneStorageWinnerAndNoPermit",
	"TestC2InterlockMustBeAcquiredBeforeTargetKeyedStartClaim",
	"TestC2InterlockRestartReopensIntentWithoutAuthority",
	"TestC2InterlockSameBootAmbiguityRemainsHeld",
	"TestC2InterlockResetRequiresExplicitDifferentBootSession",
	"TestC2InterlockReleaseRequiresTestOnlyDurableTerminalClosure",
	"TestC2InterlockRejectsCorruptCrossStoreAndAlternateOwnerState",
]);
const c2PrivateTests = Object.freeze([
	"TestC2PrivateManifestEnforcesCountSizeAndRosterBounds",
	"TestC2MissingPrivateEvidenceBeforeFinalizationRefuses",
	"TestC2PostFinalizationPurgeChangesAvailabilityOnly",
	"TestC2UnexpectedPrivateLossReportsMissingWithoutCanonicalMutation",
	"TestC2PrivateEvidenceFaultMatrixReopensAcrossRestart",
	"TestC2PurgeCannotDeleteObjectsHeadsOrRetentionFact",
]);
const c2PublicTests = Object.freeze([
	"TestC2StoreExportsNoOfficialIssuerOrRunPermit",
	"TestC2StoreExportsNoListLatestTraversalStatusOrHeadMutationSurface",
]);
const c3StoreTests = Object.freeze([
	"TestC3ConformanceAttemptConcurrentValidationAndReopenAreRaceFree",
	"TestC3ConformanceAttemptIsFreshDurableAndRestartReopenable",
	"TestC3ConformanceAttemptMarkerMutationRefusesReopen",
	"TestC3ConformanceAttemptRejectsCrossStoreAndRootReplacement",
	"TestC3ContractTargetBridgeConvergesExactAndRejectsReuse",
	"TestC3StoreBridgeExportsOnlyInertAttemptAndTargetRecords",
]);
const c4ProcessMechanicsTests = Object.freeze([
	"TestStdoutAndStderrHaveIndependentExactCaps",
	"TestStdoutAndStderrLimitsAreIndependentMutationGuard",
	"TestSimultaneousChannelOverflowRetainsIndependentFacts",
	"TestPreTermRetryNeverUsesPostDeadlineProbeAsSignalAuthority",
	"TestPreTermRetryWaitOvershootDoesNotConsumeAnotherProbe",
	"TestPreparedProcessStartsOnceAndClosesWithParentObservedFacts",
	"TestCopiedPreparedHandleCannotMultiplyStartAuthority",
	"TestCopiedRunningHandleSharesOneTerminalClosure",
	"TestPresentEmptyStdinRemainsPhysicallyDistinctFromAbsentStdin",
	"TestExecutionBudgetStartsAtPhysicalStartNotClose",
	"TestPostPermitCancellationStillProducesAChildObservation",
	"TestSpawnObservationPersistenceAbortIsClosedAndTerminal",
]);
const c4AdmissionTests = Object.freeze([
	"TestConcurrentAdmissionProducesExactlyOneStart",
	"TestRunPermitConsumptionIsSingleUseAndAdjacentToStart",
	"TestStartErrorClosesDurableRunAndClassificationWithoutChild",
	"TestPreparedCLIEnvironmentUsesFreshAttemptEvidenceRootAndShortCanaries",
	"TestParentSentinelEnvironmentIsOmittedFromRealChild",
	"TestSameTargetRetryRefusesAfterTerminalClosure",
	"TestCallerCancellationAfterAdmissionStillClosesTerminalFacts",
]);
const c4CLITests = Object.freeze([
	"TestCLIContractExecutionClosesStandaloneScope",
	"TestCLIContractExecutionForbiddenPositiveControls",
	"TestCLIContractExecutionChildBindingEvidenceStates",
	"TestCLIContractExecutionTargetMutationBlocksFinalization",
]);
const c4FinalizedRunTests = Object.freeze([
	"TestC4ContractRunBridgePersistsClosesClassifiesAndReopens",
	"TestC4FinalizedReleaseConvergesReceiptWithoutRewritingClear",
	"TestC4FinalizedClearReceiptWithoutRunLinkCannotReadmit",
	"TestC4FinalizedRunRefusesMissingSpawnObservationBeforePublication",
	"TestC4TerminalClosureRequiresDurableSpawnObservationButClassificationDoesNot",
]);
const c4ClassificationTests = Object.freeze([
	"TestC4ClassificationRecoveryUsesHistoricalRunReleaseLink",
	"TestC4ClassificationRecoveryDoesNotRequireRetainedPrivatePack",
]);
const c4AuthorityRaceTests = Object.freeze([
	"TestC4ContractRunBridgeRefusesSkippedAndMismatchedEdges",
	"TestC4SpawnObservationPersistsEveryClosedStartErrorExactly",
	"TestC4PrivateManifestDerivesRefsGroupsBodiesAndCannotFork",
	"TestC4StoreRunBridgeExportsOnlyOpaqueTypedAuthority",
]);
const expectedC2TestSymbols = Object.freeze([
	...c2NonheadTests, ...c2InterlockTests, ...c2PrivateTests, ...c2PublicTests,
].sort());
const expectedC2StructFields = Object.freeze({
	attemptStorageRecord: Object.freeze(["storeInstance", "digest", "seal", "state"]),
	conformanceAttemptRoots: Object.freeze([
		"attempt", "candidateParent", "fixture", "home", "temporary", "xdgConfig", "xdgCache", "xdgData", "xdgState",
		"state", "evidence", "marker",
	]),
	conformanceAttemptState: Object.freeze([
		"input", "nonce", "canonical", "object", "authority", "roots", "dirs", "marker", "seal",
	]),
	ConformanceAttemptInput: Object.freeze([
		"ContractBundleDigest", "ResidueHeadDigest", "TreeIdentityDigest", "MaterializationPolicyDigest",
	]),
	ConformanceAttemptRecord: Object.freeze(["store", "record"]),
	ContractTargetRecord: Object.freeze(["store", "record"]),
	targetStorageInput: Object.freeze(["attempt", "object"]),
	runStorageInput: Object.freeze(["target", "claim", "object", "manifest"]),
	executionStorageInput: Object.freeze(["run", "object"]),
	contractStorageRecord: Object.freeze([
		"storeInstance", "object", "authority", "witnessPath", "relationPath", "relation", "relationBytes",
	]),
	interlockLease: Object.freeze(["storeInstance", "state", "seal"]),
	startClaimRecord: Object.freeze([
		"storeInstance", "targetDigest", "attemptDigest", "bootDigest", "generation", "digest", "canonical", "path",
	]),
	startClaimWinner: Object.freeze(["lease", "claim", "seal"]),
	terminalClosureRecord: Object.freeze(["run", "manifest", "seal"]),
	changedBootResetAuthorization: Object.freeze(["currentBoot", "seal"]),
	privateManifestRecord: Object.freeze([
		"storeInstance", "targetDigest", "attemptDigest", "startClaimDigest", "digest", "canonical", "packDigest",
		"packBytes", "blobCount", "aggregateBytes", "entries", "manifestPath", "packPath", "purgePath", "seal",
	]),
});
const c2ProductionPaths = Object.freeze([
	"internal/store/execution_interlock.go",
	"internal/store/nonhead_contract.go",
	"internal/store/private_contract_run.go",
]);
const c2ImportPaths = Object.freeze([
	"internal/store/contract_run_bridge.go",
	...c2ProductionPaths,
	"internal/store/object_store.go",
]);
const c2ReviewedPaths = Object.freeze([
	"internal/store/contract_run_bridge.go",
	...c2ProductionPaths,
	"internal/store/object_store.go",
	"internal/store/contract_run_bridge_test.go",
	"internal/store/execution_interlock_test.go",
	"internal/store/nonhead_contract_test.go",
	"internal/store/object_store_test.go",
	"internal/store/private_contract_run_test.go",
	"internal/store/public_api_test.go",
]);
const expectedC2ObjectStoreSurface = Object.freeze([
	"Error", "Error.Error", "Error.Unwrap", "NewSemanticObject", "ObjectAuthority", "ObjectStore",
	"ObjectStore.Open", "ObjectStore.Publish", "ObjectStore.Read", "ObjectStore.Validate",
	"ObjectStore.ValidateExternalPublicationPath", "OpenObjectStore", "SemanticObject", "SemanticObject.CanonicalBytes",
	"SemanticObject.Digest", "SemanticObject.Kind", "SemanticObject.Valid",
].sort());
const expectedC2ObjectStoreFields = Object.freeze([
	"root", "objects", "digestRoot", "studies", "privateCaptures", "contractRoot", "contractLinks", "contractOps",
	"contractRuns", "rootInfo", "objectsInfo", "digestRootInfo", "studiesInfo", "privateInfo", "contractInfo",
	"contractLinkInfo", "contractOpsInfo", "contractRunInfo", "shardInfos", "studyInfos", "contractInfos", "instance",
]);
const expectedC2TestFilesByProfile = Object.freeze({
	"internal/store/nonhead_contract_test.go": c2NonheadTests,
	"internal/store/execution_interlock_test.go": c2InterlockTests,
	"internal/store/private_contract_run_test.go": c2PrivateTests,
	"internal/store/public_api_test.go": c2PublicTests,
});
const expectedC3StoreFilesByProfile = Object.freeze({
	"internal/store/nonhead_contract_test.go": Object.freeze(c3StoreTests.filter((name) => name !== "TestC3StoreBridgeExportsOnlyInertAttemptAndTargetRecords")),
	"internal/store/public_api_test.go": Object.freeze(["TestC3StoreBridgeExportsOnlyInertAttemptAndTargetRecords"]),
});
const expectedC3StoreSurface = Object.freeze([
	"ConformanceAttemptInput", "ConformanceAttemptRecord", "ConformanceAttemptRecord.ContractBundleDigest",
	"ConformanceAttemptRecord.Digest", "ConformanceAttemptRecord.InstanceNonce",
	"ConformanceAttemptRecord.MaterializationPolicyDigest", "ConformanceAttemptRecord.ResidueHeadDigest",
	"ConformanceAttemptRecord.Roots", "ConformanceAttemptRecord.TreeIdentityDigest", "ConformanceAttemptRecord.Valid",
	"ConformanceAttemptRoots", "ConformanceAttemptRoots.AttemptRoot", "ConformanceAttemptRoots.CandidateParent",
	"ConformanceAttemptRoots.EvidenceRoot", "ConformanceAttemptRoots.FixtureRoot", "ConformanceAttemptRoots.HomeRoot",
	"ConformanceAttemptRoots.MarkerPath", "ConformanceAttemptRoots.StateRoot", "ConformanceAttemptRoots.TemporaryRoot",
	"ConformanceAttemptRoots.XDGCacheRoot", "ConformanceAttemptRoots.XDGConfigRoot",
	"ConformanceAttemptRoots.XDGDataRoot", "ConformanceAttemptRoots.XDGStateRoot", "ContractTargetRecord",
	"ContractTargetRecord.AttemptDigest", "ContractTargetRecord.Digest", "ContractTargetRecord.Valid",
	"ObjectStore.AllocateConformanceAttempt", "ObjectStore.OpenConformanceAttempt",
	"ObjectStore.OpenContractTargetRecord", "ObjectStore.PersistContractTargetRecord",
].sort());

const c3OfficialTargetTests = Object.freeze([
	"TestC3OfficialTargetAttemptsAreFreshAndDistinct",
	"TestC3OfficialTargetClosedCapabilityAndDefensiveGetters",
	"TestC3OfficialTargetInvalidPreSpawnInputsReturnNoAuthority",
	"TestC3OfficialTargetLinkedRelationshipMutationRefusesReopen",
	"TestC3OfficialTargetMaterializationMutationRefusesReopen",
	"TestC3OfficialTargetMovingRefCannotRetargetReopen",
	"TestC3OfficialTargetPublicSurfaceAndSoleIssuer",
	"TestC3OpenOfficialTargetRebuildsFullAuthorityAcrossRestart",
	"TestC3OpenOfficialTargetLiveParentMatrix",
	"TestC3PublishOfficialTargetJoinsExactLivePrerequisiteGraph",
]);
const c3SingleTargetTests = Object.freeze([
	"TestC3SingleTargetAmbiguityRequiresExplicitReopen",
	"TestC3SingleTargetCannotPublishTwiceIntoOnePrivateParent",
	"TestC3SingleTargetExcludesDirtyWorktreeBytes",
	"TestC3SingleTargetMaterializesDirectlyFromInspectedAuthority",
	"TestC3SingleTargetMovingRefCannotChangePinnedBytes",
	"TestC3SingleTargetRefusesUnsupportedTreeFormsBeforePublication",
	"TestC3SingleTargetReopenRefusesEveryPublishedMutation",
]);
const c3HostEpochTests = Object.freeze([
	"TestC3HostEpochCanonicalMeasurement",
	"TestC3HostEpochConcurrentMeasurementsNeverCache",
	"TestC3HostEpochContextAndForgedAuthorityRefuse",
	"TestC3HostEpochDarwinLiveMeasurement",
	"TestC3HostEpochFaultAndInstabilityRefuse",
	"TestC3HostEpochMalformedSamplesRefuse",
	"TestC3HostEpochPublicSurfaceIsClosed",
	"TestC3HostEpochRevalidationRequiresSameFreshMeasurement",
]);
const c3NodeRuntimeTests = Object.freeze([
	"TestC3NodeRuntimeCopiedCapabilitiesSerializeRevalidation",
	"TestC3NodeRuntimeDarwinLiveAdmissionAndRevalidation",
	"TestC3NodeRuntimeDarwinRejectsNonNodeExecutable",
	"TestC3NodeRuntimeDarwinRejectsNonPrivateProbeParent",
	"TestC3NodeRuntimeFaultMatrixRefusesAuthority",
	"TestC3NodeRuntimeMeasureProbeMeasureAdmission",
	"TestC3NodeRuntimeProbeParserIsClosed",
	"TestC3NodeRuntimeProbeProgramDigestIsExact",
	"TestC3NodeRuntimeProbeWritersDrainAfterBounds",
	"TestC3NodeRuntimePublicSurfaceAndSoleSpawnEdge",
	"TestC3NodeRuntimeRejectsAmbientOrForgedInputs",
	"TestC3NodeRuntimeRevalidationIsFreshAndExact",
]);
const c3ReviewedPaths = Object.freeze([...new Set([
	...c2ReviewedPaths,
	"internal/contractexec/target.go",
	"internal/contractexec/target_test.go",
	"internal/gitobj/single_target.go",
	"internal/gitobj/single_target_test.go",
	"internal/hostepoch/epoch.go",
	"internal/hostepoch/epoch_darwin_test.go",
	"internal/hostepoch/epoch_test.go",
	"internal/hostepoch/public_api_test.go",
	"internal/hostepoch/source_darwin_cgo.go",
	"internal/hostepoch/source_unsupported.go",
	"internal/noderuntime/identity_darwin.go",
	"internal/noderuntime/probe_darwin.go",
	"internal/noderuntime/probe_darwin_test.go",
	"internal/noderuntime/public_api_test.go",
	"internal/noderuntime/runtime.go",
	"internal/noderuntime/runtime_darwin_test.go",
	"internal/noderuntime/runtime_test.go",
	"internal/noderuntime/unsupported.go",
	"spec/verification/p07b-c-c3-predecessors.json",
])]);
const expectedC3BuildTags = Object.freeze({
	"internal/contractexec/target.go": "",
	"internal/contractexec/target_test.go": "darwin && arm64 && cgo",
	"internal/gitobj/single_target.go": "",
	"internal/gitobj/single_target_test.go": "darwin && cgo",
	"internal/hostepoch/epoch.go": "",
	"internal/hostepoch/epoch_darwin_test.go": "darwin && arm64 && cgo",
	"internal/hostepoch/epoch_test.go": "",
	"internal/hostepoch/public_api_test.go": "",
	"internal/hostepoch/source_darwin_cgo.go": "darwin && arm64 && cgo",
	"internal/hostepoch/source_unsupported.go": "!darwin || !arm64 || !cgo",
	"internal/noderuntime/identity_darwin.go": "darwin && arm64 && cgo",
	"internal/noderuntime/probe_darwin.go": "darwin && arm64 && cgo",
	"internal/noderuntime/probe_darwin_test.go": "darwin && arm64 && cgo",
	"internal/noderuntime/public_api_test.go": "",
	"internal/noderuntime/runtime.go": "",
	"internal/noderuntime/runtime_darwin_test.go": "darwin && arm64 && cgo",
	"internal/noderuntime/runtime_test.go": "darwin && arm64 && cgo",
	"internal/noderuntime/unsupported.go": "!darwin || !arm64 || !cgo",
});
const expectedC3SourceImports = Object.freeze({
	"internal/contractexec/target.go": Object.freeze([
		"context", "errors", "fmt", `${modulePath}/internal/contractexec/model`, `${modulePath}/internal/domain`,
		`${modulePath}/internal/emit/node`, `${modulePath}/internal/emit/node/model`, `${modulePath}/internal/gitobj`,
		`${modulePath}/internal/hostepoch`, `${modulePath}/internal/noderuntime`, `${modulePath}/internal/store`,
		"path/filepath", "strings", "sync",
	].sort()),
	"internal/gitobj/single_target.go": Object.freeze(["context", "errors", "io", "os", "path/filepath", "sort", "strings"]),
	"internal/hostepoch/epoch.go": Object.freeze([
		"bytes", "context", "errors", "fmt", `${modulePath}/internal/canon`, `${modulePath}/internal/domain`,
	].sort()),
	"internal/hostepoch/source_darwin_cgo.go": Object.freeze(["C", "errors", "unsafe"]),
	"internal/hostepoch/source_unsupported.go": Object.freeze([]),
	"internal/noderuntime/identity_darwin.go": Object.freeze([
		"crypto/sha256", "encoding/hex", "errors", "fmt", `${modulePath}/internal/domain`, "io", "os", "path/filepath",
		"syscall", "unsafe",
	].sort()),
	"internal/noderuntime/probe_darwin.go": Object.freeze([
		"bytes", "context", "errors", `${modulePath}/internal/canon`, `${modulePath}/internal/domain`, "os", "os/exec",
		"path/filepath", "syscall", "time",
	].sort()),
	"internal/noderuntime/runtime.go": Object.freeze([
		"context", "errors", "fmt", `${modulePath}/internal/domain`, "path/filepath", "regexp", "strconv", "strings", "sync",
		"unicode/utf8",
	].sort()),
	"internal/noderuntime/unsupported.go": Object.freeze([
		"context", `${modulePath}/internal/domain`,
	].sort()),
});
const expectedC3Surfaces = Object.freeze({
	"internal/contractexec/target.go": Object.freeze([
		"CodeInvalidOfficialTarget", "CodeOfficialTargetChanged", "CodeOfficialTargetClosed", "Error", "Error.Error",
		"Error.Unwrap", "IsCode", "OpenOfficialTarget", "OpenOfficialTargetRequest", "OfficialTarget",
		"OfficialTarget.CandidateRoot", "OfficialTarget.Close", "OfficialTarget.ContractBundle", "OfficialTarget.Digest",
		"OfficialTarget.Model", "OfficialTarget.Roots", "OfficialTarget.TargetRecord", "OfficialTarget.Valid",
		"PublishOfficialTarget", "PublishOfficialTargetRequest", "ReopenOfficialTarget",
	].sort()),
	"internal/gitobj/single_target.go": Object.freeze(["MaterializeSingleTarget", "ReopenSingleTarget"]),
	"internal/hostepoch/epoch.go": Object.freeze([
		"CodeInvalidMeasurement", "CodeUnstableMeasurement", "CodeUnsupportedPlatform", "Epoch", "Epoch.Digest",
		"Epoch.Revalidate", "Epoch.Valid", "Error", "Error.Error", "Error.Unwrap", "IsCode", "Measure",
	].sort()),
	"internal/noderuntime/runtime.go": Object.freeze([
		"Admit", "CodeInvalidRuntime", "CodeProbeFailed", "CodeRuntimeChanged", "CodeUnsupportedPlatform", "Error",
		"Error.Error", "Error.Unwrap", "IsCode", "Runtime", "Runtime.Architecture", "Runtime.ExecutableByteCount",
		"Runtime.ExecutableBytesDigest", "Runtime.ExecutableMode", "Runtime.Major", "Runtime.Path", "Runtime.Platform",
		"Runtime.ProbeProgramDigest", "Runtime.Revalidate", "Runtime.Valid", "Runtime.Version",
	].sort()),
});
const expectedC3Packages = Object.freeze({
	[contractPackagePath]: Object.freeze({
		name: "contractexec", go: ["target.go"], cgo: [], test: ["target_test.go"], xtest: [], ignored: [],
		imports: expectedC3SourceImports["internal/contractexec/target.go"],
	}),
	[gitPackagePath]: Object.freeze({
		name: "gitobj",
		go: [
			"errors.go", "filesystem.go", "filesystem_darwin.go", "git.go", "inspect.go", "materialize.go",
			"single_target.go", "types.go", "validate.go",
		],
		cgo: ["publish_darwin.go"],
		test: ["fuzz_test.go", "inspect_test.go", "mutation_contract_test.go", "publish_darwin_test.go", "single_target_test.go"],
		xtest: ["helpers_external_test.go", "identity_external_test.go", "materialize_external_test.go", "refusal_external_test.go"],
		ignored: ["filesystem_unsupported.go", "publish_unsupported.go"],
		imports: [
			"C", "bytes", "context", "crypto/sha1", "crypto/sha256", "encoding/hex", "errors", "fmt",
			`${modulePath}/internal/canon`, `${modulePath}/internal/domain`, "hash", "io", "math", "os", "os/exec", "path",
			"path/filepath", "sort", "strconv", "strings", "sync", "sync/atomic", "syscall", "unicode", "unicode/utf8", "unsafe",
		].sort(),
	}),
	[hostEpochPackagePath]: Object.freeze({
		name: "hostepoch", go: ["epoch.go"], cgo: ["source_darwin_cgo.go"],
		test: ["epoch_darwin_test.go", "epoch_test.go"], xtest: ["public_api_test.go"], ignored: ["source_unsupported.go"],
		imports: ["C", "bytes", "context", "errors", "fmt", `${modulePath}/internal/canon`, `${modulePath}/internal/domain`, "unsafe"].sort(),
	}),
	[nodeRuntimePackagePath]: Object.freeze({
		name: "noderuntime", go: ["identity_darwin.go", "probe_darwin.go", "runtime.go"], cgo: [],
		test: ["probe_darwin_test.go", "runtime_darwin_test.go", "runtime_test.go"], xtest: ["public_api_test.go"],
		ignored: ["unsupported.go"],
		imports: [
			"bytes", "context", "crypto/sha256", "encoding/hex", "errors", "fmt", `${modulePath}/internal/canon`,
			`${modulePath}/internal/domain`, "io", "os", "os/exec", "path/filepath", "regexp", "strconv", "strings", "sync",
			"syscall", "time", "unicode/utf8", "unsafe",
		].sort(),
	}),
});
const c4ProductionPaths = Object.freeze([
	"internal/processmechanics/capture.go",
	"internal/processmechanics/process.go",
	"internal/processmechanics/process_darwin.go",
	"internal/processmechanics/process_unsupported.go",
	"internal/contractexec/runner/cli.go",
	"internal/contractexec/runner/evidence.go",
	"internal/contractexec/runner/runner.go",
	"internal/contractexec/runner/runner_darwin.go",
	"internal/contractexec/runner/runner_unsupported.go",
	"internal/contractexec/runner/scope_darwin.go",
	"internal/store/contract_run_bridge.go",
	"internal/store/execution_interlock.go",
	"internal/world/capture.go",
	"internal/world/process.go",
	"internal/world/process_darwin.go",
	"internal/world/process_unsupported.go",
]);
const c4TestPaths = Object.freeze([
	"internal/processmechanics/capture_darwin_test.go",
	"internal/processmechanics/process_darwin_test.go",
	"internal/contractexec/runner/runner_darwin_test.go",
	"internal/store/contract_run_bridge_test.go",
	"internal/store/execution_interlock_test.go",
	"internal/store/public_api_test.go",
	"internal/world/process_darwin_test.go",
	"internal/world/process_mutation_darwin_test.go",
	"testkit/contractexec/cli/fixture_darwin.go",
	"testkit/contractexec/cli/runner_darwin_test.go",
]);
const c4ReviewedPaths = Object.freeze([...new Set([...c3ReviewedPaths, ...c4ProductionPaths, ...c4TestPaths])]);
const c4ClaimStatusPath = "docs/status/P07B-C-C4-CLI-PROFILE.md";
const c4FinalRunbookPath = "tools/check-p07b-c-plan.mjs";
const c4BoundarySnapshotPaths = Object.freeze([
	...new Set([
		...c4ReviewedPaths,
		c4ClaimStatusPath,
		c4FinalRunbookPath,
		"tools/check-p07b-c-unit-scope.mjs",
		"spec/verification/p07b-c-unit-paths.json",
		"docs/HANDOFF_MODE_C.md",
	]),
]);
const expectedC4BuildTags = Object.freeze({
	"internal/processmechanics/capture.go": "",
	"internal/processmechanics/process.go": "",
	"internal/processmechanics/process_darwin.go": "darwin",
	"internal/processmechanics/process_unsupported.go": "!darwin",
	"internal/processmechanics/capture_darwin_test.go": "darwin && cgo",
	"internal/processmechanics/process_darwin_test.go": "darwin && cgo",
	"internal/contractexec/runner/cli.go": "darwin",
	"internal/contractexec/runner/evidence.go": "darwin",
	"internal/contractexec/runner/runner.go": "",
	"internal/contractexec/runner/runner_darwin.go": "darwin",
	"internal/contractexec/runner/runner_unsupported.go": "!darwin",
	"internal/contractexec/runner/scope_darwin.go": "darwin",
	"internal/contractexec/runner/runner_darwin_test.go": "darwin && arm64 && cgo",
	"internal/store/contract_run_bridge.go": "",
	"internal/store/contract_run_bridge_test.go": "darwin && cgo",
	"internal/store/execution_interlock.go": "",
	"internal/store/execution_interlock_test.go": "",
	"internal/store/public_api_test.go": "",
	"internal/world/capture.go": "",
	"internal/world/process.go": "",
	"internal/world/process_darwin.go": "darwin",
	"internal/world/process_unsupported.go": "!darwin",
	"internal/world/process_darwin_test.go": "darwin && cgo",
	"internal/world/process_mutation_darwin_test.go": "darwin && cgo",
	"testkit/contractexec/cli/fixture_darwin.go": "darwin && arm64 && cgo",
	"testkit/contractexec/cli/runner_darwin_test.go": "darwin && arm64 && cgo",
});
const expectedC4Packages = Object.freeze({
	[processMechanicsPackagePath]: Object.freeze({
		name: "processmechanics", go: ["capture.go", "process.go", "process_darwin.go"], cgo: [],
		test: ["capture_darwin_test.go", "process_darwin_test.go"], xtest: [], ignored: ["process_unsupported.go"],
		imports: [
			"bytes", "context", "crypto/sha256", "encoding/binary", "encoding/hex", "errors", "hash", "io", "math",
			"os", "os/exec", "path/filepath", "sync", "sync/atomic", "syscall", "time",
		],
	}),
	[contractRunnerPackagePath]: Object.freeze({
		name: "runner", go: ["cli.go", "evidence.go", "runner.go", "runner_darwin.go", "scope_darwin.go"], cgo: [],
		test: ["runner_darwin_test.go"], xtest: [], ignored: ["runner_unsupported.go"],
		imports: [
			"bytes", "context", "crypto/rand", "crypto/sha256", "encoding/binary", "encoding/hex", "errors", "fmt",
			`${modulePath}/internal/adapters/cli`, `${modulePath}/internal/adapters/cli/model`, `${modulePath}/internal/canon`,
			contractPackagePath, packagePath, `${modulePath}/internal/contractsource`, `${modulePath}/internal/domain`,
			`${modulePath}/internal/emit/node/model`, hostEpochPackagePath, processMechanicsPackagePath,
			`${modulePath}/internal/projectiontranslate`, storePackagePath,
			"io", "math", "net", "os", "path/filepath", "sort", "strings", "sync", "sync/atomic", "syscall", "time",
		].sort(),
	}),
	[contractCLITestPackagePath]: Object.freeze({
		name: "cli", go: ["fixture_darwin.go"], cgo: [], test: [], xtest: ["runner_darwin_test.go"], ignored: [],
		imports: [
			"context", "encoding/base64", "encoding/json", `${modulePath}/internal/adapters/cli`,
			`${modulePath}/internal/adapters/cli/model`, `${modulePath}/internal/canon`, `${modulePath}/internal/choice`,
			`${modulePath}/internal/choice/promotion`, contractPackagePath, packagePath,
			`${modulePath}/internal/contractsource`, `${modulePath}/internal/domain`, `${modulePath}/internal/emit/node`,
			gitPackagePath, hostEpochPackagePath, `${modulePath}/internal/projectiontranslate`, storePackagePath,
			`${modulePath}/testkit/clifixture`, `${modulePath}/testkit/gitrepo`,
			"os", "path/filepath", "runtime", "strings", "sync", "testing",
		].sort(),
	}),
});
const expectedC4MechanicsSurface = Object.freeze([
	"AbsentStdin", "BindingDigest", "BindingDigest.String", "BindingDigest.Valid", "ControlCancelled",
	"ControlOutputLimit", "ControlProbeTransportError", "ControlStartError", "ControlTimeout", "EscapeExclusion",
	"Invocation", "Limits", "NewInvocation", "PreTermProbeAbsent", "PreTermProbeNotApplicable", "PreTermProbePresent",
	"PreTermProbeUncertain", "Prepare", "Prepared", "Prepared.BindingDigest", "Prepared.Start", "PresentStdin",
	"ProcessGroupReuseExclusion", "Result", "Running", "Running.AbortReadinessTransition",
	"Running.AbortSpawnObservationPersistence", "Running.Close", "SpawnObservation", "StartError", "StartError.Code",
	"StartError.Error", "StartError.Result", "StartError.Unwrap", "Stdin",
].sort());
const expectedC4RunnerSurface = Object.freeze([
	"CodeAdmissionRefused", "CodeEvidenceClosureFailed", "CodeFixtureRejected", "CodeInvalidRequest",
	"CodeRecoveryRefused", "CodeSpawnClosureFailed", "CodeTargetChanged", "CodeUnsupportedProfile", "Error",
	"Error.Error", "Error.Unwrap", "ExecuteCLI", "IsCode", "ResumeCLIClassification",
].sort());
const expectedC4StoreBridgeSurface = Object.freeze([
	"AcquireContractRunOwner", "ContractExecutionRecord", "ContractExecutionRecord.Digest",
	"ContractExecutionRecord.Model", "ContractExecutionRecord.Valid", "ContractRunOwner",
	"ContractRunOwner.ConsumeForStart", "ContractRunOwner.PersistFinalizedRun",
	"ContractRunOwner.PersistPrivateRunManifest", "ContractRunOwner.PersistSpawnObservation",
	"ContractRunOwner.StartClaimDigest", "FinalizedRunRecord", "FinalizedRunRecord.Digest", "FinalizedRunRecord.Model",
	"FinalizedRunRecord.Valid", "OpenFinalizedRunRecord", "OpenTerminalClosure", "PersistContractExecutionRecord",
	"PrivateRunManifest", "PrivateRunManifest.EvidenceRef", "PrivateRunManifest.Summary", "PrivateRunManifest.Valid",
	"TerminalClosure", "TerminalClosure.FinalizedRun", "TerminalClosure.Release",
].sort());
const expectedC4TestsByFile = Object.freeze({
	"internal/processmechanics/capture_darwin_test.go": Object.freeze(c4ProcessMechanicsTests.slice(0, 3)),
	"internal/processmechanics/process_darwin_test.go": Object.freeze(c4ProcessMechanicsTests.slice(3)),
	"internal/contractexec/runner/runner_darwin_test.go": c4AdmissionTests,
	"testkit/contractexec/cli/runner_darwin_test.go": c4CLITests,
	"internal/store/contract_run_bridge_test.go": Object.freeze([
		"TestC4ContractRunBridgePersistsClosesClassifiesAndReopens",
		"TestC4ContractRunBridgeRefusesSkippedAndMismatchedEdges",
		"TestC4SpawnObservationPersistsEveryClosedStartErrorExactly",
		"TestC4PrivateManifestDerivesRefsGroupsBodiesAndCannotFork",
		"TestC4FinalizedRunRefusesMissingSpawnObservationBeforePublication",
		"TestC4TerminalClosureRequiresDurableSpawnObservationButClassificationDoesNot",
		"TestC4ClassificationRecoveryUsesHistoricalRunReleaseLink",
		"TestC4ClassificationRecoveryDoesNotRequireRetainedPrivatePack",
	]),
	"internal/store/execution_interlock_test.go": Object.freeze([
		"TestC4FinalizedReleaseConvergesReceiptWithoutRewritingClear",
		"TestC4FinalizedClearReceiptWithoutRunLinkCannotReadmit",
	]),
	"internal/store/public_api_test.go": Object.freeze(["TestC4StoreRunBridgeExportsOnlyOpaqueTypedAuthority"]),
});
const c4ProfileNames = Object.freeze([
	"c4-processmechanics-parity", "c4-admission-permit", "c4-cli-closure",
	"c4-finalized-run-release", "c4-classification-recovery", "c4-authority-race",
]);
const expectedC4ProfileTests = Object.freeze({
	"c4-processmechanics-parity": c4ProcessMechanicsTests,
	"c4-admission-permit": c4AdmissionTests,
	"c4-cli-closure": c4CLITests,
	"c4-finalized-run-release": c4FinalizedRunTests,
	"c4-classification-recovery": c4ClassificationTests,
	"c4-authority-race": c4AuthorityRaceTests,
});
function predecessorClaims(rows) {
	return rows.map(([label, type], index) => ({ index, label, type, grade: "TREE-EXACT" }));
}

function sealedPredecessorRecord({
	unit, commit, tree, subject, parent, scopeDigest, noteBlob, noteBodySHA256, claims,
}) {
	return Object.freeze({
		unit,
		commit,
		tree,
		subject,
		parent,
		scope_digest: scopeDigest,
		note_ref: "refs/notes/didrun",
		note_type: "blob",
		note_blob: noteBlob,
		note_body_sha256: noteBodySHA256,
		claim_count: claims.length,
		event_coverage: { complete: claims.length, total_events: claims.length },
		secrets_override: true,
		claims: predecessorClaims(claims),
	});
}

const expectedC3SPredecessor = sealedPredecessorRecord({
	unit: "C3S",
	commit: "47a65b45a50c0f39fc39e1336fc7744a97a72b8f",
	tree: "fce8593ad1fe7af1474b4e113f03e82a9a052f79",
	subject: "fix: make U6 C3 fixture phase-stable",
	parent: "63038644ba347d7b934a5557a490af95bb4428a4",
	scopeDigest: "sha256:1d5f793e70caea0ad7379e92bbf781da50846276c28a2aca2d3e7e2ed8c17a26",
	noteBlob: "7dff66800e79dc5b1a149ffe16df4dbf9178836a",
	noteBodySHA256: "b65f71e25dd53509f8df876a28fb4027559464f77861a34d1aa0c5854bfdbae6",
	claims: [
		["P07B-C C3S cumulative-selftest plan coherence", "tests-pass"],
		["P07B-C C3S cumulative-selftest plan defensive self-test", "tests-pass"],
		["P07B-C C3S historical U6 architecture compatibility", "tests-pass"],
		["P07B-C C3S U6 phase-stable fixture defensive self-test", "tests-pass"],
		["P07B-C C3S unit-scope defensive self-test", "tests-pass"],
		["P07B-C C3S cumulative verifier self-test", "tests-pass"],
		["P07B-C C3S cumulative verification", "tests-pass"],
		["P07B-C C3S exact eight-path staged scope and diff integrity", "command-succeeded"],
		["P07B-C C3S scoped staged credential-pattern scan", "command-succeeded"],
		["P07B-C C3S sealed-C3F predecessor and preceding didrun chain integrity", "command-succeeded"],
	],
});

const expectedC3FPredecessor = sealedPredecessorRecord({
	unit: "C3F",
	commit: "63038644ba347d7b934a5557a490af95bb4428a4",
	tree: "3b4c39cfa907c2907d01dcd21399cafe52075872",
	subject: "fix: admit exact C3 target codec surface through B",
	parent: "efa918928246c7793f4ad7201c003020e3a42193",
	scopeDigest: "sha256:f64293895a1b824faafddf236757d381c813b59c40286225bb1c352d5c812c1a",
	noteBlob: "098958ba7a9b2d9d237cee901a34279424b68128",
	noteBodySHA256: "a23982f84573505fc966378dbcc4f825daf11da8a08b2bfd24f7f71de588d315",
	claims: [
		["P07B-C C3F B future-surface plan coherence", "tests-pass"],
		["P07B-C C3F B future-surface plan defensive self-test", "tests-pass"],
		["P07B-C C3F historical B architecture compatibility", "tests-pass"],
		["P07B-C C3F B conditional future-surface defensive self-test", "tests-pass"],
		["P07B-C C3F unit-scope defensive self-test", "tests-pass"],
		["P07B-C C3F cumulative verifier self-test", "tests-pass"],
		["P07B-C C3F cumulative verification", "tests-pass"],
		["P07B-C C3F exact ten-path staged scope and diff integrity", "command-succeeded"],
		["P07B-C C3F scoped staged credential-pattern scan", "command-succeeded"],
		["P07B-C C3F sealed-C3L predecessor and preceding didrun chain integrity", "command-succeeded"],
	],
});

const expectedC3LPredecessor = sealedPredecessorRecord({
	unit: "C3L",
	commit: "efa918928246c7793f4ad7201c003020e3a42193",
	tree: "d9187c6449392c269a385b53c563ba4411112460",
	subject: "fix: admit exact C3 store model edge through U6",
	parent: "2b84d2841971568784d2ac955775b4a99ca7f0f6",
	scopeDigest: "sha256:b3624e1337681e0909e6c074246103e066118a5924e8f790e7d1f42359da9dc0",
	noteBlob: "b21365e156719fbc84e480108a7e8672644f5a71",
	noteBodySHA256: "d4721b344adb95442c5273e596c83ea9cb1b855d26655198a9803771f83e2e9d",
	claims: [
		["P07B-C C3L legacy-lattice plan coherence", "tests-pass"],
		["P07B-C C3L legacy-lattice plan defensive self-test", "tests-pass"],
		["P07B-C C3L historical U6 architecture compatibility", "tests-pass"],
		["P07B-C C3L U6 phase-admission defensive self-test", "tests-pass"],
		["P07B-C C3L cumulative B architecture compatibility", "tests-pass"],
		["P07B-C C3L B defensive self-test", "tests-pass"],
		["P07B-C C3L unit-scope defensive self-test", "tests-pass"],
		["P07B-C C3L cumulative verifier self-test", "tests-pass"],
		["P07B-C C3L cumulative verification", "tests-pass"],
		["P07B-C C3L exact nine-path staged scope and diff integrity", "command-succeeded"],
		["P07B-C C3L scoped staged credential-pattern scan", "command-succeeded"],
		["P07B-C C3L sealed-C3A predecessor and preceding didrun chain integrity", "command-succeeded"],
	],
});

const expectedC3APredecessor = sealedPredecessorRecord({
	unit: "C3A",
	commit: "2b84d2841971568784d2ac955775b4a99ca7f0f6",
	tree: "b8d858033431fe51c262ce725b9d9b051ba9e8c9",
	subject: "fix: align pre-C3 architecture and runtime bounds",
	parent: "13369122ba7d5557eba1949095c1135a41843070",
	scopeDigest: "sha256:cc9bb98ceeb2f281336d3e235377d3cdb8920664f833baea830a84b847621188",
	noteBlob: "3e516946f4b86e536ae9ea0124938bd534573983",
	noteBodySHA256: "07a0d04dc667963ea16ec6bb458b929d3a7659637e705bb642b5267766609a8c",
	claims: [
		["P07B-C C3A cumulative-admission plan coherence", "tests-pass"],
		["P07B-C C3A cumulative-admission plan defensive self-test", "tests-pass"],
		["P07B-C C3A runtime-version exact 128-byte model boundary", "tests-pass"],
		["P07B-C C3A cumulative B architecture compatibility", "tests-pass"],
		["P07B-C C3A B admission and entrypoint defensive self-test", "tests-pass"],
		["P07B-C C3A unit-scope defensive self-test", "tests-pass"],
		["P07B-C C3A cumulative verifier self-test", "tests-pass"],
		["P07B-C C3A cumulative verification", "tests-pass"],
		["P07B-C C3A exact thirteen-path staged scope and diff integrity", "command-succeeded"],
		["P07B-C C3A scoped staged credential-pattern scan", "command-succeeded"],
		["P07B-C C3A preceding didrun chain integrity", "command-succeeded"],
	],
});

const expectedC3Predecessor = Object.freeze({
	schema_version: "p07b-c-c3-predecessors/v3",
	unit: "C3",
	source_parent: expectedC3SPredecessor,
	ancestry: [expectedC3FPredecessor, expectedC3LPredecessor, expectedC3APredecessor],
});
const expectedC3PredecessorsByUnit = Object.freeze({
	C3S: expectedC3SPredecessor,
	C3F: expectedC3FPredecessor,
	C3L: expectedC3LPredecessor,
	C3A: expectedC3APredecessor,
});
const c3Redacted = "«redacted:high-entropy»";
const c3RedactedPair = `${c3Redacted}.${c3Redacted}`;
const c3SealedPredecessorRepositoryRoot = "/Users/drewnelson/Documents/codex-ap-dev-stresstest-creative";
function expectedC3SealedArgvPrefix(unit) {
	const slug = unit.toLowerCase();
	const prefix = [
		"/usr/bin/env", "-i", c3RedactedPair, `PWD=${c3SealedPredecessorRepositoryRoot}`, c3RedactedPair, c3RedactedPair,
		`${c3Redacted}.countershape/p07bc-${slug}-final/gocache`, c3RedactedPair, c3RedactedPair,
		"GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off",
		"GOFLAGS=-mod=readonly -buildvcs=false -p=1", "CGO_ENABLED=1", "GOMAXPROCS=2",
		"LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1", "NODE_OPTIONS=", "NODE_PATH=",
		"PATH=/opt/homebrew/bin:/usr/bin:/bin", "SHELL=/bin/sh", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_NO_LAZY_FETCH=1", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0",
		c3Redacted, c3Redacted, c3Redacted, "COUNTERSHAPE_SH=/bin/sh", c3Redacted, c3Redacted,
		"CC=/usr/bin/clang", "CXX=/usr/bin/clang++",
	];
	if (prefix.length !== 39) throw new Error("P07B-C C3 sealed argv prefix invariant");
	return Object.freeze(prefix);
}
const expectedC3ClaimArgvPrefixesByUnit = Object.freeze({
	C3S: expectedC3SealedArgvPrefix("C3S"),
	C3F: expectedC3SealedArgvPrefix("C3F"),
	C3L: expectedC3SealedArgvPrefix("C3L"),
	C3A: expectedC3SealedArgvPrefix("C3A"),
});
const expectedC3ClaimArgvTailsByUnit = Object.freeze({
	C3S: Object.freeze([
		["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"],
		["/opt/homebrew/bin/node", "tools/check-u6-architecture.mjs"],
		["/opt/homebrew/bin/node", "tools/check-u6-architecture-selftest.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"],
		["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"],
		["/opt/homebrew/bin/node", "tools/verify-current.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3S", "--source-final-gate"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3S", "--credential-scan"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3s-preseal-ledger"],
	].map((tail) => Object.freeze(tail))),
	C3F: Object.freeze([
		["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"],
		["/opt/homebrew/bin/node", "tools/check-p07b-b-architecture.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-b-architecture-selftest.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"],
		["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"],
		["/opt/homebrew/bin/node", "tools/verify-current.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3F", "--source-final-gate"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3F", "--credential-scan"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3f-preseal-ledger"],
	].map((tail) => Object.freeze(tail))),
	C3L: Object.freeze([
		["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"],
		["/opt/homebrew/bin/node", "tools/check-u6-architecture.mjs"],
		["/opt/homebrew/bin/node", "tools/check-u6-architecture-selftest.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-b-architecture.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-b-architecture-selftest.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"],
		["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"],
		["/opt/homebrew/bin/node", "tools/verify-current.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3L", "--source-final-gate"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3L", "--credential-scan"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3l-preseal-ledger"],
	].map((tail) => Object.freeze(tail))),
	C3A: Object.freeze([
		["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--self-test"],
		["/opt/homebrew/bin/go", "test", "-mod=readonly", "-buildvcs=false", "-p=1", "-count=1", "-run", "^TestTargetPrimitiveBoundsAndDerivedRelations$", "./internal/contractexec/model"],
		["/opt/homebrew/bin/node", "tools/check-p07b-b-architecture.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-b-architecture-selftest.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--self-test"],
		["/opt/homebrew/bin/node", "tools/verify-current-selftest.mjs"],
		["/opt/homebrew/bin/node", "tools/verify-current.mjs"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3A", "--source-final-gate"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-unit-scope.mjs", "--unit", "C3A", "--credential-scan"],
		["/opt/homebrew/bin/node", "tools/check-p07b-c-plan.mjs", "--verify-c3a-preseal-ledger"],
	].map((tail) => Object.freeze(tail))),
});

const expectedProductionFiles = Object.freeze([
	"codec.go",
	"doc.go",
	"errors.go",
	"execution.go",
	"run.go",
	"run_parse.go",
	"target.go",
	"tuple_parse.go",
	"types.go",
]);
const expectedTestFiles = Object.freeze([
	"algebra_test.go",
	"codec_test.go",
	"fuzz_test.go",
	"model_test.go",
	"schema_parity_test.go",
]);
const expectedModelEntries = Object.freeze([
	...expectedProductionFiles,
	...expectedTestFiles,
].sort().map((file) => `${file}:file`));
const expectedProductionImports = Object.freeze([
	"bytes",
	"encoding/base64",
	"errors",
	"fmt",
	`${modulePath}/internal/canon`,
	`${modulePath}/internal/domain`,
	`${modulePath}/internal/emit/node/model`,
	`${modulePath}/internal/portablevalue`,
	"regexp",
	"sort",
	"strconv",
	"strings",
	"unicode",
	"unicode/utf8",
]);
const expectedTestImports = Object.freeze([
	"bytes",
	"crypto/sha256",
	"encoding/base64",
	"encoding/json",
	"fmt",
	`${modulePath}/internal/canon`,
	`${modulePath}/internal/domain`,
	`${modulePath}/internal/emit/node/model`,
	`${modulePath}/internal/portablevalue`,
	"io",
	"os",
	"path/filepath",
	"runtime",
	"testing",
]);
const expectedTestSymbols = Object.freeze([
	"FuzzContractExecutionParser",
	"FuzzContractExecutionTargetParser",
	"FuzzFinalizedContractRunParser",
	"TestAllStartErrorCodesRoundTripWithExactClosure",
	"TestC1ExampleProbe",
	"TestC1SchemaRuntimeOverapproximationProbe",
	"TestCanonicalGettersAndInputsAreDefensive",
	"TestCanonicalObjectBodiesHaveNoSelfDigestOrMutableSelectors",
	"TestCanonicalObjectGraphRoundTripsAndClassifiesWithoutRequestedResult",
	"TestCheckedExamplesParseRebuildAndClassifyExactly",
	"TestClassifierProfileIdentityGolden",
	"TestClassifierTruthTableUsesOnlyEligibleCompleteTupleMembership",
	"TestCleanWitnessUsesExactSixteenTypedReferences",
	"TestClosedEnumBoundsAndTypedRoles",
	"TestClosedRunAlgebraExhaustiveCrossProduct",
	"TestParentMismatchAndNoncanonicalBytesRefuse",
	"TestPrimaryAndCleanupControlsRemainIndependent",
	"TestPrivateEvidenceCeilingsAndZeroCorrelation",
	"TestSchemaProjectionOverapproximationsRefuseAtRuntime",
	"TestScopeDerivationViolationOutranksMissing",
	"TestStrictParsersRejectCoherentlyRehashedUnknownAndDerivedMembers",
	"TestTargetPrimitiveBoundsAndDerivedRelations",
]);
const optInTestSymbols = Object.freeze([
	"TestC1ExampleProbe",
	"TestC1SchemaRuntimeOverapproximationProbe",
]);
const goJSONProfiles = Object.freeze({
	"model-suite": Object.freeze({
		packagePath,
		pass: Object.freeze(expectedTestSymbols.filter((name) => !optInTestSymbols.includes(name))),
		skip: optInTestSymbols,
	}),
	"exhaustive-algebra": Object.freeze({
		packagePath,
		pass: Object.freeze([
			"TestClassifierTruthTableUsesOnlyEligibleCompleteTupleMembership",
			"TestClosedRunAlgebraExhaustiveCrossProduct",
		]),
		skip: Object.freeze([]),
	}),
	"target-parser-fuzz": Object.freeze({
		packagePath,
		pass: Object.freeze(["FuzzContractExecutionTargetParser"]),
		skip: Object.freeze([]),
	}),
	"finalized-run-parser-fuzz": Object.freeze({
		packagePath,
		pass: Object.freeze(["FuzzFinalizedContractRunParser"]),
		skip: Object.freeze([]),
	}),
	"execution-parser-fuzz": Object.freeze({
		packagePath,
		pass: Object.freeze(["FuzzContractExecutionParser"]),
		skip: Object.freeze([]),
	}),
	"c2-nonhead-persistence": Object.freeze({
		packagePath: storePackagePath, packageArgument: "./internal/store",
		pass: c2NonheadTests, skip: Object.freeze([]),
	}),
	"c2-interlock": Object.freeze({
		packagePath: storePackagePath, packageArgument: "./internal/store",
		pass: c2InterlockTests, skip: Object.freeze([]),
	}),
	"c2-private-evidence": Object.freeze({
		packagePath: storePackagePath, packageArgument: "./internal/store",
		pass: c2PrivateTests, skip: Object.freeze([]),
	}),
	"c2-public-surface": Object.freeze({
		packagePath: storePackagePath, packageArgument: "./internal/store",
		pass: c2PublicTests, skip: Object.freeze([]),
	}),
	"c3-official-target": Object.freeze({
		packagePath: contractPackagePath, packageArgument: "./internal/contractexec",
		pass: c3OfficialTargetTests, skip: Object.freeze([]),
	}),
	"c3-single-target": Object.freeze({
		packagePath: gitPackagePath, packageArgument: "./internal/gitobj",
		pass: c3SingleTargetTests, skip: Object.freeze([]),
	}),
	"c3-hostepoch": Object.freeze({
		packagePath: hostEpochPackagePath, packageArgument: "./internal/hostepoch",
		pass: c3HostEpochTests, skip: Object.freeze([]),
	}),
	"c3-noderuntime": Object.freeze({
		packagePath: nodeRuntimePackagePath, packageArgument: "./internal/noderuntime",
		pass: c3NodeRuntimeTests, skip: Object.freeze([]),
	}),
	"c3-store-bridge": Object.freeze({
		packagePath: storePackagePath, packageArgument: "./internal/store",
		pass: c3StoreTests, skip: Object.freeze([]),
	}),
	"c4-processmechanics-parity": Object.freeze({
		packagePath: processMechanicsPackagePath, packageArgument: "./internal/processmechanics",
		pass: c4ProcessMechanicsTests, skip: Object.freeze([]), race: true,
	}),
	"c4-admission-permit": Object.freeze({
		packagePath: contractRunnerPackagePath, packageArgument: "./internal/contractexec/runner",
		pass: c4AdmissionTests, skip: Object.freeze([]), race: false,
	}),
	"c4-cli-closure": Object.freeze({
		packagePath: contractCLITestPackagePath, packageArgument: "./testkit/contractexec/cli",
		pass: c4CLITests, skip: Object.freeze([]), race: false,
	}),
	"c4-finalized-run-release": Object.freeze({
		packagePath: storePackagePath, packageArgument: "./internal/store",
		pass: c4FinalizedRunTests, skip: Object.freeze([]), race: false,
	}),
	"c4-classification-recovery": Object.freeze({
		packagePath: storePackagePath, packageArgument: "./internal/store",
		pass: c4ClassificationTests, skip: Object.freeze([]), race: false,
	}),
	"c4-authority-race": Object.freeze({
		packagePath: storePackagePath, packageArgument: "./internal/store",
		pass: c4AuthorityRaceTests, skip: Object.freeze([]), race: true,
	}),
});
const expectedLocalDependencies = Object.freeze([
	`${modulePath}/internal/adapters/cli/model`,
	`${modulePath}/internal/adapters/http/model`,
	`${modulePath}/internal/canon`,
	`${modulePath}/internal/contractexec/model`,
	`${modulePath}/internal/contractsource`,
	`${modulePath}/internal/domain`,
	`${modulePath}/internal/emit/node/model`,
	`${modulePath}/internal/emit/node/program/v1`,
	`${modulePath}/internal/portablevalue`,
	`${modulePath}/internal/projectionprofile`,
	`${modulePath}/internal/runnerprofile`,
]);
const expectedTargetRoot = Object.freeze([
	"schema_version",
	"kind",
	"target_version",
	"publication_scope",
	"contract_bundle_digest",
	"terminal_residue_binding",
	"tree_binding",
	"attempt_binding",
	"boot_session_binding",
	"runtime_binding",
]);
const expectedRunRoot = Object.freeze([
	"schema_version",
	"kind",
	"run_version",
	"publication_scope",
	"contract_execution_target_digest",
	"attempt_artifact_digest",
	"start_claim_ref",
	"closed_run_witness",
]);
const expectedExecutionRoot = Object.freeze([
	"schema_version",
	"kind",
	"execution_version",
	"publication_scope",
	"classifier_profile",
	"contract_execution_target_digest",
	"finalized_contract_run_digest",
	"result",
]);
const expectedObjects = Object.freeze(["ContractExecutionTarget", "FinalizedContractRun", "ContractExecution"]);
const expectedEvidenceKinds = Object.freeze([
	"MATERIALIZATION_REVALIDATION",
	"RUNTIME_REVALIDATION",
	"PROCESS_RESULT",
	"WAIT_RESULT",
	"DRAIN_RESULT",
	"TEARDOWN_RESULT",
	"ORPHAN_CHECK",
	"FINALIZATION_MARKER",
	"CAPTURED_OBSERVATION",
	"PROJECTION_RESULT",
	"TARGET_INVENTORY",
	"CHILD_BINDINGS",
	"IMPORT_RESOLUTION",
	"SERVICE_BINDINGS",
	"NAMED_PARENT_SECRET_SENTINEL_INHERITANCE",
	"PRIVATE_EVIDENCE_MANIFEST",
]);
const expectedStartErrors = Object.freeze([
	"OS_START_ERROR",
	"PRESPAWN_MATERIALIZATION_REVALIDATION_FAILED",
	"PRESPAWN_RUNTIME_REVALIDATION_FAILED",
]);
const expectedPrimaryReasons = Object.freeze([
	"NONE",
	"MATERIALIZATION_ERROR",
	"START_ERROR",
	"READINESS_ERROR",
	"PROBE_TRANSPORT_ERROR",
	"TIMEOUT",
	"CANCELLED",
	"OUTPUT_LIMIT",
	"PROJECTION_REJECTED",
]);
const expectedScopeDomains = Object.freeze([
	"TARGET_INVENTORY",
	"CHILD_BINDINGS",
	"IMPORT_RESOLUTION",
	"SERVICE_BINDINGS",
	"NAMED_PARENT_SECRET_SENTINEL_INHERITANCE",
]);
const expectedScopeStates = Object.freeze(["COMPLETE", "PARTIAL", "VIOLATED"]);
const expectedResults = Object.freeze(["CONFORMS", "CONTRADICTS", "INELIGIBLE_EXECUTION"]);
const forbiddenSchemaMembers = Object.freeze([
	"artifact_digest",
	"terminal_disposition",
	"requested_result",
	"requested_classification",
	"study_head_advanced",
	"choicepoint_freshened",
	"historical_execution_evidence_reused",
	"latest",
	"current",
	"head",
]);

class ArchitectureError extends Error {
	constructor(code, detail) {
		super(`${code}: ${detail}`);
		this.code = code;
	}
}

function sorted(values) { return [...values].sort(); }
function exact(left, right) { return JSON.stringify(left) === JSON.stringify(right); }
function sha256(bytes) { return createHash("sha256").update(bytes).digest("hex"); }
function canonicalJSON(value) {
	if (value === null || typeof value === "boolean" || typeof value === "string") return JSON.stringify(value);
	if (typeof value === "number") {
		if (!Number.isSafeInteger(value) || Object.is(value, -0)) throw new ArchitectureError("P07B_C1_EXAMPLE_CANONICAL", "unsafe number");
		return String(value);
	}
	if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(",")}]`;
	if (!value || typeof value !== "object") throw new ArchitectureError("P07B_C1_EXAMPLE_CANONICAL", typeof value);
	const keys = Object.keys(value).sort((left, right) => Buffer.compare(Buffer.from(left), Buffer.from(right)));
	return `{${keys.map((key) => `${JSON.stringify(key)}:${canonicalJSON(value[key])}`).join(",")}}`;
}
function typedDigest(kind, body) {
	return `sha256:${createHash("sha256").update(`countershape/v1/${kind}\0`).update(canonicalJSON(body)).digest("hex")}`;
}
function slash(value) { return value.split(sep).join("/"); }

function parseJSONStream(source) {
	const values = [];
	let start = -1;
	let depth = 0;
	let quoted = false;
	let escaped = false;
	for (let index = 0; index < source.length; index += 1) {
		const character = source[index];
		if (start < 0) {
			if (/\s/u.test(character)) continue;
			if (character !== "{") throw new ArchitectureError("P07B_C1_GO_LIST_FRAME", `offset ${index}`);
			start = index;
			depth = 1;
			continue;
		}
		if (quoted) {
			if (escaped) escaped = false;
			else if (character === "\\") escaped = true;
			else if (character === "\"") quoted = false;
			continue;
		}
		if (character === "\"") quoted = true;
		else if (character === "{") depth += 1;
		else if (character === "}") {
			depth -= 1;
			if (depth === 0) {
				values.push(JSON.parse(source.slice(start, index + 1)));
				start = -1;
			}
		}
	}
	if (start >= 0 || quoted || depth !== 0) throw new ArchitectureError("P07B_C1_GO_LIST_FRAME", "truncated JSON stream");
	return values;
}

function run(executable, args, code, timeout = 180_000) {
	const result = spawnSync(executable, args, {
		cwd: repositoryRoot,
		encoding: "utf8",
		timeout,
		maxBuffer: 32 * 1024 * 1024,
		env: {
			...process.env,
			COUNTERSHAPE_GO: goExecutable,
			GOMAXPROCS: "2",
			GOFLAGS: "-mod=readonly -buildvcs=false -p=1",
		},
	});
	if (result.error || result.signal || result.status !== 0) {
		throw new ArchitectureError(code, `${result.status ?? result.signal}: ${result.stderr || result.stdout || result.error}`);
	}
	return result.stdout;
}

export function parseC4StatusClaimMap(source) {
	const startToken = "## Intended C4 claim map\n";
	const endToken = "\n## Nonclaims and next gate";
	const start = source.indexOf(startToken);
	const end = start < 0 ? -1 : source.indexOf(endToken, start + startToken.length);
	if (start < 0 || end < 0 || source.indexOf(startToken, start + startToken.length) !== -1 ||
		source.indexOf(endToken, end + endToken.length) !== -1) {
		throw new ArchitectureError("P07B_C4_PROFILE_COMMAND", "C4 status claim-map section framing");
	}
	const lines = source.slice(start + startToken.length, end).trim().split(/\r?\n/u);
	if (lines.length !== 82 || lines[0] !== "| # | Claim | Type | Intended grade |" ||
		lines[1] !== "| ---: | --- | --- | --- |") {
		throw new ArchitectureError("P07B_C4_PROFILE_COMMAND", `C4 status claim-map table shape: ${lines.length}`);
	}
	return lines.slice(2).map((line, index) => {
		const match = /^\| ([1-9][0-9]*) \| `([^`\r\n]+)` \| `(tests-pass|command-succeeded)` \| `UNRECEIPTED` \|$/u.exec(line);
		if (match === null || Number(match[1]) !== index + 1) {
			throw new ArchitectureError("P07B_C4_PROFILE_COMMAND", `C4 status claim-map row ${index + 1}`);
		}
		return Object.freeze({ number: index + 1, label: match[2], type: match[3], grade: "UNRECEIPTED" });
	});
}

function renderC4FinalRunbook() {
	const result = spawnSync(process.execPath, [resolve(repositoryRoot, c4FinalRunbookPath), "--print-final-runbook", "C4"], {
		cwd: repositoryRoot,
		encoding: "utf8",
		timeout: 30_000,
		maxBuffer: 32 * 1024 * 1024,
		env: {
			...process.env,
			COUNTERSHAPE_GO: goExecutable,
			GOMAXPROCS: "2",
			GOFLAGS: "-mod=readonly -buildvcs=false -p=1",
		},
	});
	if (result.error || result.signal || result.status !== 0 || result.stderr !== "") {
		throw new ArchitectureError(
			"P07B_C4_PROFILE_COMMAND",
			`C4 final runbook render failed: ${result.status ?? result.signal}: ${result.stderr || result.stdout || result.error}`,
		);
	}
	return result.stdout;
}

export function parseC4FinalRunbookClaimMap(source) {
	const headings = [...source.matchAll(/^# ([1-9][0-9]*)\. ([^\r\n]+)$/gmu)];
	const claims = [...source.matchAll(/^\/opt\/homebrew\/bin\/didrun claim '(tests-pass|command-succeeded)' --label '([^'\r\n]+)'$/gmu)];
	if (headings.length !== 80 || claims.length !== 80) {
		throw new ArchitectureError("P07B_C4_PROFILE_COMMAND", `C4 final runbook claim-map shape: ${headings.length}/${claims.length}`);
	}
	return headings.map((heading, index) => {
		const number = Number(heading[1]);
		const label = heading[2];
		const claim = claims[index];
		if (number !== index + 1 || claim[2] !== label) {
			throw new ArchitectureError("P07B_C4_PROFILE_COMMAND", `C4 final runbook claim-map row ${index + 1}`);
		}
		return Object.freeze({ number, label, type: claim[1], grade: "UNRECEIPTED" });
	});
}

async function collectC4ClaimMapFacts() {
	const statusSource = await readFile(resolve(repositoryRoot, c4ClaimStatusPath), "utf8");
	return Object.freeze({
		status: parseC4StatusClaimMap(statusSource),
		runbook: parseC4FinalRunbookClaimMap(renderC4FinalRunbook()),
	});
}

const c3GitEnvironment = Object.freeze({
	HOME: "/",
	PATH: "/usr/bin:/bin",
	LANG: "C",
	LC_ALL: "C",
	TZ: "UTC",
	NO_COLOR: "1",
	GIT_CONFIG_NOSYSTEM: "1",
	GIT_CONFIG_GLOBAL: "/dev/null",
	GIT_NO_LAZY_FETCH: "1",
	GIT_OPTIONAL_LOCKS: "0",
	GIT_TERMINAL_PROMPT: "0",
});
const fatalUTF8 = new TextDecoder("utf-8", { fatal: true });

function runC3Git(args, acceptedStatuses = [0]) {
	const result = spawnSync("/usr/bin/git", ["--no-replace-objects", ...args], {
		cwd: repositoryRoot,
		timeout: 30_000,
		maxBuffer: 4 * 1024 * 1024,
		env: c3GitEnvironment,
	});
	if (result.error || result.signal || !acceptedStatuses.includes(result.status) || (result.stderr?.length ?? 0) !== 0) {
		throw new ArchitectureError(
			"P07B_C3_PREDECESSOR_AUTHORITY",
			`git ${args[0]} failed (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`,
		);
	}
	return Object.freeze({
		status: result.status,
		stdout: result.stdout ?? Buffer.alloc(0),
		stderr: result.stderr ?? Buffer.alloc(0),
	});
}

function decodeC3Git(bytes, operation) {
	try {
		return fatalUTF8.decode(bytes);
	} catch (error) {
		throw new ArchitectureError("P07B_C3_PREDECESSOR_AUTHORITY", `${operation} is not UTF-8 (${error.message})`);
	}
}

function decodeC3GitLine(bytes, operation) {
	const text = decodeC3Git(bytes, operation);
	if (text.length === 0 || !text.endsWith("\n") || text.slice(0, -1).includes("\n") || text.includes("\r")) {
		throw new ArchitectureError(
			"P07B_C3_PREDECESSOR_AUTHORITY",
			`${operation} did not emit one exact LF-terminated line`,
		);
	}
	return text.slice(0, -1);
}

function c3GitLine(args) {
	return decodeC3GitLine(runC3Git(args).stdout, `git ${args[0]}`);
}

function requireExactKeys(value, expected, detail) {
	if (!value || typeof value !== "object" || Array.isArray(value) || !exact(Object.keys(value).sort(), [...expected].sort())) {
		throw new ArchitectureError("P07B_C3_PREDECESSOR_AUTHORITY", `${detail} field roster`);
	}
}

function normalizeC3DidrunNote(rawBody, expectedRecord, reopenedTree) {
	let note;
	try {
		note = JSON.parse(decodeC3Git(rawBody, `${expectedRecord.unit} didrun note`));
	} catch (error) {
		if (error instanceof ArchitectureError) throw error;
		throw new ArchitectureError("P07B_C3_PREDECESSOR_AUTHORITY", `${expectedRecord.unit} didrun note JSON (${error.message})`);
	}
	requireExactKeys(note, ["claims", "commit", "coverage", "secrets_override", "tree", "version"], `${expectedRecord.unit} note`);
	if (note.version !== 1 || note.commit !== expectedRecord.commit || note.tree !== reopenedTree || note.secrets_override !== true ||
		!Array.isArray(note.claims) || note.claims.length !== expectedRecord.claim_count) {
		throw new ArchitectureError("P07B_C3_PREDECESSOR_AUTHORITY", `${expectedRecord.unit} note identity or claim count`);
	}
	requireExactKeys(note.coverage, ["by_coverage", "total_events"], `${expectedRecord.unit} note coverage`);
	requireExactKeys(note.coverage.by_coverage, ["complete"], `${expectedRecord.unit} note coverage class`);
	if (note.coverage.by_coverage.complete !== expectedRecord.event_coverage.complete ||
		note.coverage.total_events !== expectedRecord.event_coverage.total_events) {
		throw new ArchitectureError("P07B_C3_PREDECESSOR_AUTHORITY", `${expectedRecord.unit} note coverage counts`);
	}
	const claims = note.claims.map((recorded, index) => {
		requireExactKeys(recorded, ["claim", "delta", "exit_code", "grade", "reason", "supporting_event_index"], `${expectedRecord.unit} recorded claim ${index}`);
		requireExactKeys(recorded.claim, ["argv_preview", "ctype", "declared_at_index", "event_indices", "label", "pathspecs"], `${expectedRecord.unit} claim ${index}`);
		const expectedClaim = expectedRecord.claims[index];
		const expectedArgvTail = expectedC3ClaimArgvTailsByUnit[expectedRecord.unit]?.[index];
		const expectedArgvPrefix = expectedC3ClaimArgvPrefixesByUnit[expectedRecord.unit];
		const expectedArgv = expectedArgvTail === undefined || expectedArgvPrefix === undefined
			? undefined
			: [...expectedArgvPrefix, ...expectedArgvTail];
		if (!Array.isArray(recorded.claim.argv_preview) || recorded.claim.argv_preview.some((argument) => typeof argument !== "string") ||
			expectedArgv === undefined || !exact(recorded.claim.argv_preview, expectedArgv) ||
			recorded.claim.declared_at_index !== index || !exact(recorded.claim.event_indices, [index]) ||
			recorded.claim.label !== expectedClaim.label || recorded.claim.ctype !== expectedClaim.type ||
			!exact(recorded.claim.pathspecs, []) || !exact(recorded.delta, []) || recorded.exit_code !== 0 ||
			recorded.grade !== "tree-exact" || recorded.reason !== "self-stable command ran against the sealed tree" ||
			recorded.supporting_event_index !== index) {
			throw new ArchitectureError("P07B_C3_PREDECESSOR_AUTHORITY", `${expectedRecord.unit} claim ${index} didrun shape`);
		}
		return { index, label: recorded.claim.label, type: recorded.claim.ctype, grade: "TREE-EXACT" };
	});
	return Object.freeze({
		commit: note.commit,
		tree: note.tree,
		secrets_override: note.secrets_override,
		claim_count: claims.length,
		event_coverage: { complete: note.coverage.by_coverage.complete, total_events: note.coverage.total_events },
		claims,
	});
}

export function inspectC3DidrunNote(unit, rawBody) {
	const expectedRecord = expectedC3PredecessorsByUnit[unit];
	if (expectedRecord === undefined || !Buffer.isBuffer(rawBody)) {
		throw new ArchitectureError("P07B_C3_PREDECESSOR_AUTHORITY", "C3 didrun-note fixture unit or body is invalid");
	}
	return normalizeC3DidrunNote(rawBody, expectedRecord, expectedRecord.tree);
}

export function collectC3PredecessorRecordAuthority(unitOrRecord, transform = (_args, result) => result) {
	const expectedRecord = typeof unitOrRecord === "string" ? expectedC3PredecessorsByUnit[unitOrRecord] : unitOrRecord;
	if (expectedRecord === undefined || typeof transform !== "function") {
		throw new ArchitectureError("P07B_C3_PREDECESSOR_AUTHORITY", "C3 predecessor fixture unit or transformer is invalid");
	}
	const read = (args, acceptedStatuses = [0]) => {
		const original = runC3Git(args, acceptedStatuses);
		const result = transform(Object.freeze([...args]), original);
		if (!result || typeof result !== "object" || !exact(Object.keys(result).sort(), ["status", "stderr", "stdout"]) ||
			!Number.isInteger(result.status) || !Buffer.isBuffer(result.stdout) || !Buffer.isBuffer(result.stderr) ||
			result.stderr.length !== 0 || !acceptedStatuses.includes(result.status)) {
			throw new ArchitectureError("P07B_C3_PREDECESSOR_AUTHORITY", `${expectedRecord.unit} hostile Git fixture shape`);
		}
		return result;
	};
	const line = (args) => decodeC3GitLine(read(args).stdout, `git ${args[0]}`);
	const commit = line(["rev-parse", "--verify", `${expectedRecord.commit}^{commit}`]);
	const tree = line(["rev-parse", "--verify", `${expectedRecord.commit}^{tree}`]);
	const parentLine = line(["show", "-s", "--format=%P", expectedRecord.commit]);
	const parents = parentLine === "" ? [] : parentLine.split(" ");
	const subject = line(["show", "-s", "--format=%s", expectedRecord.commit]);
	const noteBlob = line(["notes", "--ref=refs/notes/didrun", "list", expectedRecord.commit]);
	const noteType = line(["cat-file", "-t", noteBlob]);
	const noteBody = read(["cat-file", "blob", noteBlob]).stdout;
	const noteBodySHA256 = createHash("sha256").update(noteBody).digest("hex");
	if (commit !== expectedRecord.commit || parents.length !== 1 || parents.some((parent) => !/^[0-9a-f]{40}$/u.test(parent))) {
		throw new ArchitectureError("P07B_C3_PREDECESSOR_AUTHORITY", `${expectedRecord.unit} commit identity or parent cardinality`);
	}
	const authority = Object.freeze({
		unit: expectedRecord.unit,
		commit,
		tree,
		subject,
		parents,
		note_ref: "refs/notes/didrun",
		note_type: noteType,
		note_blob: noteBlob,
		note_body_sha256: noteBodySHA256,
		note: normalizeC3DidrunNote(noteBody, expectedRecord, tree),
	});
	if (!exact(authority, expectedC3PredecessorRecordAuthority(expectedRecord))) {
		throw new ArchitectureError("P07B_C3_PREDECESSOR_AUTHORITY", `${expectedRecord.unit} raw predecessor authority mismatch`);
	}
	return authority;
}

function expectedC3PredecessorRecordAuthority(record) {
	return Object.freeze({
		unit: record.unit,
		commit: record.commit,
		tree: record.tree,
		subject: record.subject,
		parents: [record.parent],
		note_ref: record.note_ref,
		note_type: record.note_type,
		note_blob: record.note_blob,
		note_body_sha256: record.note_body_sha256,
			note: {
			commit: record.commit,
			tree: record.tree,
			secrets_override: record.secrets_override,
			claim_count: record.claim_count,
			event_coverage: record.event_coverage,
			claims: record.claims,
		},
	});
}

const expectedC3PredecessorAuthority = Object.freeze({
	source_parent: expectedC3PredecessorRecordAuthority(expectedC3SPredecessor),
	ancestry: [
		expectedC3PredecessorRecordAuthority(expectedC3FPredecessor),
		expectedC3PredecessorRecordAuthority(expectedC3LPredecessor),
		expectedC3PredecessorRecordAuthority(expectedC3APredecessor),
	],
	chain: {
		source_parent_is_ancestor_of_head: true,
		exact_parent_chain: true,
	},
});

export function collectC3PredecessorAuthority(transform = (_args, result) => result) {
	if (typeof transform !== "function") {
		throw new ArchitectureError("P07B_C3_PREDECESSOR_AUTHORITY", "C3 predecessor chain transformer is invalid");
	}
	const sourceParent = collectC3PredecessorRecordAuthority(expectedC3SPredecessor, transform);
	const ancestry = [
		collectC3PredecessorRecordAuthority(expectedC3FPredecessor, transform),
		collectC3PredecessorRecordAuthority(expectedC3LPredecessor, transform),
		collectC3PredecessorRecordAuthority(expectedC3APredecessor, transform),
	];
	const ancestorArgs = ["merge-base", "--is-ancestor", expectedC3SPredecessor.commit, "HEAD"];
	const ancestorResult = transform(Object.freeze([...ancestorArgs]), runC3Git(ancestorArgs, [0, 1]));
	if (!ancestorResult || typeof ancestorResult !== "object" ||
		!exact(Object.keys(ancestorResult).sort(), ["status", "stderr", "stdout"]) ||
		!Number.isInteger(ancestorResult.status) || ![0, 1].includes(ancestorResult.status) ||
		!Buffer.isBuffer(ancestorResult.stdout) || ancestorResult.stdout.length !== 0 ||
		!Buffer.isBuffer(ancestorResult.stderr) || ancestorResult.stderr.length !== 0) {
		throw new ArchitectureError("P07B_C3_PREDECESSOR_AUTHORITY", "C3 predecessor ancestry result is malformed");
	}
	const authority = Object.freeze({
		source_parent: sourceParent,
		ancestry,
		chain: {
			source_parent_is_ancestor_of_head: ancestorResult.status === 0,
			exact_parent_chain: sourceParent.parents[0] === ancestry[0].commit &&
				ancestry[0].parents[0] === ancestry[1].commit &&
				ancestry[1].parents[0] === ancestry[2].commit &&
				ancestry[2].parents[0] === expectedC3APredecessor.parent,
		},
	});
	if (!exact(authority, expectedC3PredecessorAuthority)) {
		throw new ArchitectureError("P07B_C3_PREDECESSOR_AUTHORITY", "C3 predecessor chain authority mismatch");
	}
	return authority;
}

function goList(args) {
	if (!isAbsolute(goExecutable)) throw new ArchitectureError("P07B_C1_GO_EXECUTABLE", "COUNTERSHAPE_GO must be absolute");
	return parseJSONStream(run(goExecutable, ["list", "-json", ...args], "P07B_C1_GO_LIST"));
}

export function productionStoreOwnerReferenceCount(source) {
	return count(goCodeOnly(source), /\bAcquireContractRunOwner\b/gu);
}

async function productionStoreOwnerReferenceSites(packages) {
	const paths = [];
	for (const packageValue of packages) {
		const filenames = [...new Set([
			...(packageValue.GoFiles ?? []), ...(packageValue.CgoFiles ?? []), ...(packageValue.IgnoredGoFiles ?? []),
		])].filter((name) => name.endsWith(".go") && !name.endsWith("_test.go"));
		for (const filename of filenames) {
			const absolute = resolve(packageValue.Dir, filename);
			const relativeFile = relative(repositoryRoot, absolute).split(sep).join("/");
			if (relativeFile === ".." || relativeFile.startsWith("../") || isAbsolute(relativeFile)) {
				throw new ArchitectureError("P07B_C4_PRODUCTION_CALL_SITE", `${absolute} escapes the repository root`);
			}
			const source = await readFile(absolute, "utf8");
			const occurrences = productionStoreOwnerReferenceCount(source);
			if (occurrences > 0) paths.push(`${relativeFile}:${occurrences}`);
		}
	}
	return sorted(paths);
}

function goTestSymbols() {
	const output = run(goExecutable, [
		"test", "-mod=readonly", "-buildvcs=false", "-p=1", "-count=1", "-list", ".", "./internal/contractexec/model",
	], "P07B_C1_GO_TEST_LIST");
	const lines = output.trimEnd().split(/\r?\n/u);
	const trailer = lines.pop();
	const trailerParts = trailer?.trim().split(/\s+/u) ?? [];
	if (trailerParts.length !== 3 || trailerParts[0] !== "ok" || trailerParts[1] !== packagePath || !/^[0-9.]+s$/u.test(trailerParts[2])) {
		throw new ArchitectureError("P07B_C1_GO_TEST_LIST_FRAME", trailer ?? "missing trailer");
	}
	if (lines.some((line) => !/^(?:Test|Fuzz)[A-Za-z0-9_]+$/u.test(line))) {
		throw new ArchitectureError("P07B_C1_GO_TEST_LIST_FRAME", JSON.stringify(lines));
	}
	return sorted(lines);
}

function goTestC2Symbols() {
	const output = run(goExecutable, [
		"test", "-mod=readonly", "-buildvcs=false", "-p=1", "-count=1", "-list", "^TestC2", "./internal/store",
	], "P07B_C2_GO_TEST_LIST");
	const lines = output.trimEnd().split(/\r?\n/u);
	const trailer = lines.pop();
	const trailerParts = trailer?.trim().split(/\s+/u) ?? [];
	if (trailerParts.length !== 3 || trailerParts[0] !== "ok" || trailerParts[1] !== storePackagePath || !/^[0-9.]+s$/u.test(trailerParts[2])) {
		throw new ArchitectureError("P07B_C2_GO_TEST_LIST_FRAME", trailer ?? "missing trailer");
	}
	if (lines.some((line) => !/^TestC2[A-Za-z0-9_]+$/u.test(line))) {
		throw new ArchitectureError("P07B_C2_GO_TEST_LIST_FRAME", JSON.stringify(lines));
	}
	return sorted(lines);
}

export function validateGoJSONTranscript(profileName, bytes) {
	const profile = goJSONProfiles[profileName];
	if (!profile) throw new ArchitectureError("P07B_C1_GO_JSON_PROFILE", profileName);
	if (!Buffer.isBuffer(bytes) || bytes.length === 0 || bytes.length > 64 * 1024 * 1024) {
		throw new ArchitectureError("P07B_C1_GO_JSON_SIZE", bytes?.length ?? "not-buffer");
	}
	let source;
	try {
		source = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
	} catch (error) {
		throw new ArchitectureError("P07B_C1_GO_JSON_UTF8", error.message);
	}
	const lines = source.trimEnd().split(/\r?\n/u);
	if (lines.some((line) => line.length === 0 || Buffer.byteLength(line, "utf8") > 1024 * 1024)) {
		throw new ArchitectureError("P07B_C1_GO_JSON_FRAME", "blank or oversized line");
	}
	const events = lines.map((line, index) => {
		try {
			return JSON.parse(line);
		} catch (error) {
			throw new ArchitectureError("P07B_C1_GO_JSON_PARSE", `${index + 1}:${error.message}`);
		}
	});
	if (events.some((event) => !event || typeof event !== "object" || Array.isArray(event) || event.Package !== profile.packagePath)) {
		throw new ArchitectureError("P07B_C1_GO_JSON_PACKAGE", "foreign or malformed event");
	}
	if (events.some((event) => event.Action === "fail")) {
		throw new ArchitectureError("P07B_C1_GO_JSON_FAILURE", "Go reported failure");
	}
	const expected = new Set([...profile.pass, ...profile.skip]);
	const topLevelEvents = events.filter((event) => typeof event.Test === "string" && !event.Test.includes("/"));
	const observed = new Set(topLevelEvents.map((event) => event.Test));
	if (!exact(sorted(observed), sorted(expected))) {
		throw new ArchitectureError("P07B_C1_GO_JSON_TEST_ROSTER", JSON.stringify(sorted(observed)));
	}
	for (const name of expected) {
		const named = topLevelEvents.filter((event) => event.Test === name);
		const runCount = named.filter((event) => event.Action === "run").length;
		const passCount = named.filter((event) => event.Action === "pass").length;
		const skipCount = named.filter((event) => event.Action === "skip").length;
		const wantsPass = profile.pass.includes(name);
		if (runCount !== 1 || passCount !== (wantsPass ? 1 : 0) || skipCount !== (wantsPass ? 0 : 1)) {
			throw new ArchitectureError("P07B_C1_GO_JSON_TEST_RESULT", `${name}:run=${runCount},pass=${passCount},skip=${skipCount}`);
		}
	}
	const packagePasses = events.filter((event) => event.Action === "pass" && !Object.hasOwn(event, "Test"));
	if (packagePasses.length !== 1) {
		throw new ArchitectureError("P07B_C1_GO_JSON_PACKAGE_RESULT", String(packagePasses.length));
	}
	return Object.freeze({ profile: profileName, passed: profile.pass.length, skipped: profile.skip.length });
}

export function goJSONArguments(profileName) {
	const profile = goJSONProfiles[profileName];
	if (!profile?.packageArgument || profile.skip.length !== 0 || profile.pass.length === 0 ||
		profile.pass.some((name) => !/^(?:Test|Fuzz)[A-Za-z0-9_]+$/u.test(name)) ||
		(profileName.startsWith("c4-") && typeof profile.race !== "boolean")) {
		throw new ArchitectureError("P07B_C2_GO_JSON_RUN_PROFILE", profileName);
	}
	const pattern = `^(?:${profile.pass.join("|")})$`;
	return Object.freeze([
		"test", ...(profile.race === true ? ["-race"] : []),
		"-mod=readonly", "-buildvcs=false", "-p=1", "-count=1",
		...(profileName === "c3-official-target" ? ["-timeout=12m"] : []),
		"-json", "-run", pattern, profile.packageArgument,
	]);
}

export const c3ArchitectureProfileTimeoutMS = 15 * 60 * 1000;

export function runGoJSONProfile(profileName) {
	const output = run(
		goExecutable,
		goJSONArguments(profileName),
		profileName.startsWith("c4-") ? "P07B_C4_GO_JSON_RUN" :
			profileName.startsWith("c3-") ? "P07B_C3_GO_JSON_RUN" : "P07B_C2_GO_JSON_RUN",
		(profileName.startsWith("c3-") || profileName.startsWith("c4-")) ? c3ArchitectureProfileTimeoutMS : 180_000,
	);
	return validateGoJSONTranscript(profileName, Buffer.from(output, "utf8"));
}

async function readStandardInput() {
	const chunks = [];
	let size = 0;
	for await (const chunk of process.stdin) {
		const bytes = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
		size += bytes.length;
		if (size > 64 * 1024 * 1024) throw new ArchitectureError("P07B_C1_GO_JSON_SIZE", size);
		chunks.push(bytes);
	}
	return Buffer.concat(chunks);
}

async function readJSON(relativePath) {
	return JSON.parse(await readFile(resolve(repositoryRoot, relativePath), "utf8"));
}

function walk(value, visit, pointer = "") {
	visit(value, pointer);
	if (Array.isArray(value)) value.forEach((entry, index) => walk(entry, visit, `${pointer}/${index}`));
	else if (value && typeof value === "object") {
		for (const [key, entry] of Object.entries(value)) walk(entry, visit, `${pointer}/${key}`);
	}
}

function schemaFacts(targetSchema, runSchema, executionSchema) {
	const unclosedObjects = [];
	const incompleteRequiredObjects = [];
	const forbiddenMembers = [];
	for (const [name, schema] of Object.entries({ target: targetSchema, run: runSchema, execution: executionSchema })) {
		walk(schema, (node, pointer) => {
			if (!node || typeof node !== "object" || Array.isArray(node)) return;
			if (node.type === "object" && node.additionalProperties !== false) unclosedObjects.push(`${name}${pointer}`);
			if (node.type === "object" && node.additionalProperties === false && node.properties && typeof node.properties === "object") {
				const properties = sorted(Object.keys(node.properties));
				const required = Array.isArray(node.required) ? sorted(node.required) : [];
				if (
					new Set(required).size !== required.length
					|| required.some((member) => typeof member !== "string")
					|| !exact(required, properties)
				) incompleteRequiredObjects.push(`${name}${pointer}`);
			}
			if (node.properties && typeof node.properties === "object") {
				for (const member of Object.keys(node.properties)) {
					if (forbiddenSchemaMembers.includes(member)) forbiddenMembers.push(`${name}${pointer}/properties/${member}`);
				}
			}
		});
	}
	return {
		targetRoot: Object.keys(targetSchema.properties ?? {}),
		runRoot: Object.keys(runSchema.properties ?? {}),
		executionRoot: Object.keys(executionSchema.properties ?? {}),
		objectKinds: [
			targetSchema.properties?.kind?.const,
			runSchema.properties?.kind?.const,
			executionSchema.properties?.kind?.const,
		],
		unclosedObjects,
		incompleteRequiredObjects,
		forbiddenMembers,
		evidenceKinds: runSchema.$defs?.EvidenceKind?.enum ?? [],
		startErrors: runSchema.$defs?.SpawnObservation?.oneOf?.[0]?.properties?.error_code?.enum ?? [],
		primaryReasons: runSchema.$defs?.ProcessClosure?.properties?.primary_reason?.enum ?? [],
		cleanupControls: runSchema.$defs?.ProcessClosure?.properties?.cleanup_controls?.items?.enum ?? [],
		scopeDomains: runSchema.$defs?.ScopeCheck?.oneOf?.[0]?.properties?.domain?.enum ?? [],
		scopeStates: runSchema.$defs?.StandaloneScope?.properties?.status?.enum ?? [],
		results: executionSchema.properties?.result?.enum ?? [],
		startClaimKind: runSchema.properties?.start_claim_ref?.properties?.kind?.const,
		constants: {
			targetScope: targetSchema.properties?.publication_scope?.const,
			runScope: runSchema.properties?.publication_scope?.const,
			executionScope: executionSchema.properties?.publication_scope?.const,
			classifierProfile: executionSchema.properties?.classifier_profile?.const,
		},
		limits: {
			pathMax: targetSchema.properties?.runtime_binding?.properties?.admitted_executable_path?.maxLength,
			pidMax: runSchema.$defs?.SpawnObservation?.oneOf?.[1]?.properties?.pid?.maximum,
			processEvidenceMax: runSchema.$defs?.ProcessClosure?.properties?.evidence_refs?.maxItems,
			scopeCheckCount: runSchema.$defs?.StandaloneScope?.properties?.checks?.maxItems,
			privateBlobMax: runSchema.$defs?.PrivateEvidence?.properties?.blob_count?.maximum,
			privateByteMax: runSchema.$defs?.PrivateEvidence?.properties?.aggregate_byte_count?.maximum,
		},
	};
}

function exampleFacts(bundle, target, finalizedRun, execution) {
	const witnessRefs = [];
	walk(finalizedRun.closed_run_witness, (node) => {
		if (
			node
			&& typeof node === "object"
			&& !Array.isArray(node)
			&& Object.keys(node).length === 2
			&& Object.hasOwn(node, "kind")
			&& Object.hasOwn(node, "digest")
		) witnessRefs.push(node.kind);
	});
	const targetDigest = typedDigest("ContractExecutionTarget", target);
	const runDigest = typedDigest("FinalizedContractRun", finalizedRun);
	return {
		witnessReferenceKinds: sorted(witnessRefs),
		witnessReferenceCount: witnessRefs.length,
		scopeOrder: finalizedRun.closed_run_witness?.standalone_scope?.checks?.map((check) => check.domain) ?? [],
		result: execution.result,
		graph: {
			bundleToTarget: target.contract_bundle_digest === typedDigest("ContractBundle", bundle)
				&& target.terminal_residue_binding?.current_digest === target.contract_bundle_digest,
			targetToRun: finalizedRun.contract_execution_target_digest === targetDigest
				&& finalizedRun.attempt_artifact_digest === target.attempt_binding?.attempt_artifact_digest,
			runToExecution: execution.contract_execution_target_digest === targetDigest
				&& execution.finalized_contract_run_digest === runDigest,
		},
	};
}

function count(source, expression) { return source.match(expression)?.length ?? 0; }

function goImports(source) {
	const imports = [];
	for (const block of source.matchAll(/^\s*import\s*\(([^]*?)^\s*\)/gmu)) {
		for (const match of block[1].matchAll(/^\s*(?:[._A-Za-z][A-Za-z0-9_]*\s+)?"([^"]+)"/gmu)) imports.push(match[1]);
	}
	for (const match of source.matchAll(/^\s*import\s+(?:[._A-Za-z][A-Za-z0-9_]*\s+)?"([^"]+)"/gmu)) imports.push(match[1]);
	return sorted(imports);
}

function goBuildTag(source) {
	return /^\/\/go:build\s+([^\r\n]+)$/mu.exec(source)?.[1] ?? "";
}

function packageBuildFacts(value) {
	return {
		name: value?.Name,
		modulePath: value?.Module?.Path,
		moduleMain: value?.Module?.Main === true,
		go: sorted(value?.GoFiles ?? []),
		cgo: sorted(value?.CgoFiles ?? []),
		test: sorted(value?.TestGoFiles ?? []),
		xtest: sorted(value?.XTestGoFiles ?? []),
		ignored: sorted(value?.IgnoredGoFiles ?? []),
		invalid: sorted(value?.InvalidGoFiles ?? []),
		imports: sorted(value?.Imports ?? []),
	};
}

// Preserve code positions while blanking comments and literals. Authority-
// symbol checks must be satisfied by executable Go, never prose or test data.
function goCodeOnly(source) {
	const output = source.split("");
	let state = "code";
	for (let index = 0; index < source.length; index += 1) {
		const character = source[index];
		const next = source[index + 1] ?? "";
		if (state === "code") {
			if (character === "/" && next === "/") {
				output[index] = output[index + 1] = " ";
				index += 1;
				state = "line-comment";
			} else if (character === "/" && next === "*") {
				output[index] = output[index + 1] = " ";
				index += 1;
				state = "block-comment";
			} else if (character === '"' || character === "'" || character === "`") {
				output[index] = " ";
				state = character === '"' ? "string" : character === "'" ? "rune" : "raw-string";
			}
		} else if (state === "line-comment") {
			output[index] = character === "\n" ? "\n" : " ";
			if (character === "\n") state = "code";
		} else if (state === "block-comment") {
			output[index] = character === "\n" ? "\n" : " ";
			if (character === "*" && next === "/") {
				output[index + 1] = " ";
				index += 1;
				state = "code";
			}
		} else if (state === "raw-string") {
			output[index] = character === "\n" ? "\n" : " ";
			if (character === "`") state = "code";
		} else {
			output[index] = character === "\n" ? "\n" : " ";
			if (character === "\\") {
				if (index + 1 < source.length) output[index + 1] = source[index + 1] === "\n" ? "\n" : " ";
				index += 1;
			} else if ((state === "string" && character === '"') || (state === "rune" && character === "'")) {
				state = "code";
			}
		}
	}
	return output.join("");
}

function balancedBody(source, expression) {
	const match = expression.exec(source);
	if (!match) return "";
	const openBrace = source.indexOf("{", match.index);
	if (openBrace < 0) return "";
	let depth = 0;
	for (let index = openBrace; index < source.length; index += 1) {
		if (source[index] === "{") depth += 1;
		else if (source[index] === "}") {
			depth -= 1;
			if (depth === 0) return source.slice(openBrace + 1, index);
		}
	}
	return "";
}

function functionBody(source, name) {
	return balancedBody(source, new RegExp(`^func\\s+(?:\\([^)]*\\)\\s+)?${name}\\s*\\(`, "mu"));
}

function methodBody(source, receiver, name) {
	return balancedBody(source, new RegExp(
		`^func\\s+\\(\\s*[^)]*\\*?${receiver}\\s*\\)\\s+${name}\\s*\\(`, "mu",
	));
}

function structFields(source, name) {
	const body = balancedBody(source, new RegExp(`^type\\s+${name}\\s+struct\\s*`, "mu"));
	return body.split(/\r?\n/u).map((line) => line.trim()).filter(Boolean)
		.flatMap((line) => {
			const names = /^([A-Za-z_][A-Za-z0-9_]*(?:\s*,\s*[A-Za-z_][A-Za-z0-9_]*)*)\s+/u.exec(line)?.[1];
			return names ? names.split(/\s*,\s*/u) : [];
		});
}

function exportedSurface(source) {
	const functions = [...source.matchAll(/^func\s+([A-Z][A-Za-z0-9_]*)\s*(?:\[[^\]]*\]\s*)?\(/gmu)].map((match) => match[1]);
	const types = [...source.matchAll(/^type\s+([A-Z][A-Za-z0-9_]*)\b/gmu)].map((match) => match[1]);
	const values = [...source.matchAll(/^(?:var|const)\s+([A-Z][A-Za-z0-9_]*)\b/gmu)].map((match) => match[1]);
	const grouped = [...source.matchAll(/^(?:type|var|const)\s*\(([^]*?)^\)/gmu)]
		.flatMap((block) => [...block[1].matchAll(/^\s*([A-Z][A-Za-z0-9_]*)\b/gmu)].map((match) => match[1]));
	const methods = [];
	for (const match of source.matchAll(/^func\s+\(([^)]*)\)\s+([A-Z][A-Za-z0-9_]*)\s*\(/gmu)) {
		const receiver = /\*?([A-Za-z_][A-Za-z0-9_]*)\s*$/u.exec(match[1])?.[1];
		if (receiver && /^[A-Z]/u.test(receiver)) methods.push(`${receiver}.${match[2]}`);
	}
	return sorted([...functions, ...types, ...values, ...grouped, ...methods]);
}

function ordered(body, anchors) {
	let cursor = -1;
	for (const anchor of anchors) {
		cursor = body.indexOf(anchor, cursor + 1);
		if (cursor < 0) return false;
	}
	return true;
}

async function readC2Source(relativePath) {
	const absolute = resolve(repositoryRoot, relativePath);
	const fromRoot = relative(repositoryRoot, absolute);
	if (isAbsolute(fromRoot) || fromRoot === ".." || fromRoot.startsWith(`..${sep}`)) {
		throw new ArchitectureError("P07B_C2_PATH_ESCAPE", relativePath);
	}
	const before = await lstat(absolute);
	if (!before.isFile() || before.isSymbolicLink()) throw new ArchitectureError("P07B_C2_NONREGULAR_FILE", relativePath);
	const bytes = await readFile(absolute);
	const after = await lstat(absolute);
	if (!after.isFile() || after.isSymbolicLink() || before.dev !== after.dev || before.ino !== after.ino || before.size !== after.size) {
		throw new ArchitectureError("P07B_C2_FILE_CHANGED", relativePath);
	}
	let source;
	try {
		source = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
	} catch (error) {
		throw new ArchitectureError("P07B_C2_INVALID_UTF8", `${relativePath}:${error.message}`);
	}
	return Object.freeze({ path: relativePath, source, digest: sha256(bytes) });
}

async function topologyFacts() {
	const contractexec = await readdir(resolve(repositoryRoot, "internal/contractexec"), { withFileTypes: true });
	const model = await readdir(resolve(repositoryRoot, "internal/contractexec/model"), { withFileTypes: true });
	const contractexecEntries = [];
	const modelEntries = [];
	for (const entry of contractexec) {
		const stats = await lstat(resolve(repositoryRoot, "internal/contractexec", entry.name));
		contractexecEntries.push(`${entry.name}:${stats.isSymbolicLink() ? "symlink" : entry.isDirectory() ? "directory" : entry.isFile() ? "file" : "other"}`);
	}
	for (const entry of model) {
		const stats = await lstat(resolve(repositoryRoot, "internal/contractexec/model", entry.name));
		modelEntries.push(`${entry.name}:${stats.isSymbolicLink() ? "symlink" : entry.isDirectory() ? "directory" : entry.isFile() ? "file" : "other"}`);
	}
	return { contractexecEntries: sorted(contractexecEntries), modelEntries: sorted(modelEntries) };
}

export async function collectFacts() {
	const [modelPackage] = goList(["./internal/contractexec/model"]);
	const dependencyPackages = goList(["-deps", "./internal/contractexec/model"]);
	const repositoryPackages = goList(["./..."]);
	const [targetSchema, runSchema, executionSchema, bundle, target, finalizedRun, execution, c0] = await Promise.all([
		readJSON("spec/schema/v1/contract-execution-target.schema.json"),
		readJSON("spec/schema/v1/finalized-contract-run.schema.json"),
		readJSON("spec/schema/v1/contract-execution.schema.json"),
		readJSON("spec/examples/v1/contract-bundle.valid.json"),
		readJSON("spec/examples/v1/contract-execution-target.valid.json"),
		readJSON("spec/examples/v1/finalized-contract-run.valid.json"),
		readJSON("spec/examples/v1/contract-execution.valid.json"),
		readJSON("spec/verification/p07b-c-c0-authority.json"),
	]);
	const nonGoBuildFields = [
		"CFiles", "CXXFiles", "MFiles", "HFiles", "FFiles", "SFiles", "SwigFiles", "SwigCXXFiles", "SysoFiles", "EmbedFiles",
	];
	return {
		package: {
			importPath: modelPackage?.ImportPath,
			name: modelPackage?.Name,
			modulePath: modelPackage?.Module?.Path,
			moduleMain: modelPackage?.Module?.Main === true,
			productionFiles: sorted(modelPackage?.GoFiles ?? []),
			testFiles: sorted(modelPackage?.TestGoFiles ?? []),
			xTestFiles: sorted(modelPackage?.XTestGoFiles ?? []),
			productionImports: sorted(modelPackage?.Imports ?? []),
			testImports: sorted(modelPackage?.TestImports ?? []),
			xTestImports: sorted(modelPackage?.XTestImports ?? []),
			testSymbols: goTestSymbols(),
			ignoredGoFiles: sorted(modelPackage?.IgnoredGoFiles ?? []),
			invalidGoFiles: sorted(modelPackage?.InvalidGoFiles ?? []),
			nonGoBuildFiles: sorted(nonGoBuildFields.flatMap((field) => modelPackage?.[field] ?? [])),
		},
		localDependencies: sorted(dependencyPackages
			.map((entry) => entry.ImportPath)
			.filter((entry) => entry === modulePath || entry.startsWith(`${modulePath}/`))),
		externalDependencies: sorted(dependencyPackages
			.filter((entry) => !entry.Standard && entry.ImportPath !== modulePath && !entry.ImportPath.startsWith(`${modulePath}/`))
			.map((entry) => entry.ImportPath)),
		productionImporters: sorted(repositoryPackages
			.filter((entry) => entry.ImportPath !== packagePath && (entry.Imports ?? []).includes(packagePath))
			.map((entry) => entry.ImportPath)),
		topology: await topologyFacts(),
		schema: schemaFacts(targetSchema, runSchema, executionSchema),
		example: exampleFacts(bundle, target, finalizedRun, execution),
		c0: {
			objects: c0.semantic_objects?.map((entry) => entry.name) ?? [],
			scopeDomains: c0.scope_domains ?? [],
			scopeStates: c0.scope_states ?? [],
		},
	};
}

export async function collectC2Facts() {
	const [storePackage] = goList(["./internal/store"]);
	const entries = await Promise.all(c2ReviewedPaths.map(readC2Source));
	const sources = Object.fromEntries(entries.map((entry) => [entry.path, entry.source]));
	const newProduction = c2ProductionPaths.map((path) => sources[path]).join("\n");
	const changedProduction = c2ImportPaths.map((path) => sources[path]).join("\n");
	const changedProductionCode = goCodeOnly(changedProduction);
	const allTests = Object.entries(sources).filter(([path]) => path.endsWith("_test.go"))
		.map(([, source]) => source).join("\n");
	const nonhead = sources["internal/store/nonhead_contract.go"];
	const interlock = sources["internal/store/execution_interlock.go"];
	const privateRun = sources["internal/store/private_contract_run.go"];
	const objectStore = sources["internal/store/object_store.go"];
	const publicAPI = sources["internal/store/public_api_test.go"];
	const directoryEntries = [];
	for (const entry of await readdir(resolve(repositoryRoot, "internal/store"), { withFileTypes: true })) {
		const metadata = await lstat(resolve(repositoryRoot, "internal/store", entry.name));
		directoryEntries.push(`${entry.name}:${metadata.isSymbolicLink() ? "symlink" : entry.isFile() ? "file" : entry.isDirectory() ? "directory" : "other"}`);
	}
	const nonGoBuildFields = [
		"CFiles", "CXXFiles", "MFiles", "HFiles", "FFiles", "SFiles", "SwigFiles", "SwigCXXFiles", "SysoFiles", "EmbedFiles",
	];
	const structSources = {
		attemptStorageRecord: nonhead,
		conformanceAttemptRoots: nonhead,
		conformanceAttemptState: nonhead,
		ConformanceAttemptInput: nonhead,
		ConformanceAttemptRecord: nonhead,
		ContractTargetRecord: nonhead,
		targetStorageInput: nonhead,
		runStorageInput: nonhead,
		executionStorageInput: nonhead,
		contractStorageRecord: nonhead,
		interlockLease: interlock,
		startClaimRecord: interlock,
		startClaimWinner: interlock,
		terminalClosureRecord: interlock,
		changedBootResetAuthorization: interlock,
		privateManifestRecord: privateRun,
	};
	const privateKindsBody = /var\s+privateEvidenceKindOrder\s*=\s*\[\.\.\.\]string\s*\{([^]*?)\n\}/mu.exec(privateRun)?.[1] ?? "";
	const testFiles = {};
	for (const path of new Set([...Object.keys(expectedC2TestFilesByProfile), ...Object.keys(expectedC3StoreFilesByProfile)])) {
		testFiles[path] = sorted([...sources[path].matchAll(/^func\s+(TestC[23][A-Za-z0-9_]+)\s*\(/gmu)].map((match) => match[1]));
	}
	const forbiddenPatterns = [
		["process-start", /\b(?:exec\.Command|os\.StartProcess)\s*\(/u],
		["production-capability", /\b(?:OfficialTarget|RunPermit)\b/u],
		["semantic-head", /\b(?:CreateStudy|OpenHead|AdvanceBaseline|AdvanceDivergence|AdvanceReduction|AdvanceConfirmation|AdvanceChoicepoint|AdvanceRuling|AdvanceResidue|ConfirmResiduePublication)\s*\(/u],
	];
	return structuredClone({
		package: {
			importPath: storePackage?.ImportPath,
			name: storePackage?.Name,
			modulePath: storePackage?.Module?.Path,
			moduleMain: storePackage?.Module?.Main === true,
			productionFiles: sorted(storePackage?.GoFiles ?? []),
			testFiles: sorted(storePackage?.TestGoFiles ?? []),
			xTestFiles: sorted(storePackage?.XTestGoFiles ?? []),
			ignoredGoFiles: sorted(storePackage?.IgnoredGoFiles ?? []),
			invalidGoFiles: sorted(storePackage?.InvalidGoFiles ?? []),
			nonGoBuildFiles: sorted(nonGoBuildFields.flatMap((field) => storePackage?.[field] ?? [])),
			productionImports: sorted(storePackage?.Imports ?? []),
		},
		directoryEntries: sorted(directoryEntries),
		imports: Object.fromEntries(c2ImportPaths.map((path) => [path.split("/").at(-1), goImports(sources[path])])),
		modelImporters: c2ImportPaths.filter((path) => goImports(sources[path]).includes(`${modulePath}/internal/contractexec/model`)),
		newProductionExports: Object.fromEntries(c2ProductionPaths.map((path) => [path, exportedSurface(sources[path])])),
		objectStoreExports: exportedSurface(objectStore),
		compilerParsedSurface: [
			"go/ast", "go/parser", "go/token", "parser.ParseFile", "parser.SkipObjectResolution",
			"ast.IsExported", "*ast.StructType", "field.Names", "c2ReceiverName", "c2FileExportedSurface",
			"c2ForbiddenProcessSurface", "parsed.Imports", "strconv.Unquote", "os/exec",
			"c2ExportedProductionSurface", "wantPackageSurface",
			"execution_interlock.go", "nonhead_contract.go", "object_store.go", "private_contract_run.go",
		].every((anchor) => publicAPI.includes(anchor)),
		structs: Object.fromEntries(Object.entries(structSources).map(([name, source]) => [name, structFields(source, name)])),
		testSymbols: goTestC2Symbols(),
		testFiles,
		forbiddenSurface: forbiddenPatterns.filter(([, expression]) => expression.test(changedProductionCode)).map(([name]) => name),
		namespaces: {
			fields: structFields(objectStore, "ObjectStore"),
			paths: [
				"contractDirectory       = \"contract-execution\"",
				"contractLinkDirectory   = \"links\"",
				"contractOpsDirectory    = \"operations\"",
				"contractRunDirectory    = \"contract-runs\"",
			].every((anchor) => objectStore.includes(anchor)),
			retained: ["s.contractRoot", "s.contractLinks", "s.contractOps", "s.contractRuns", "s.contractInfos"]
				.every((anchor) => functionBody(objectStore, "assertReady").includes(anchor)),
			replacementTest: sources["internal/store/object_store_test.go"].includes("replacement contract-operation directory retained store authority"),
			caseAliasGuard: count(functionBody(objectStore, "assertReady"), /rejectPathCaseAlias\s*\(/gu) === 2,
		},
		relations: {
			constants: [
				"CONFORMANCE_ATTEMPT_TO_TARGET", "TARGET_TO_FINALIZED_RUN", "RUN_PROFILE_TO_EXECUTION",
				"target-by-attempt", "run-by-target", "execution-by-run-profile", "contract-storage-relation/v1",
			].every((anchor) => nonhead.includes(`\"${anchor}\"`)),
			effects: sorted([...nonhead.matchAll(/contractEffect\s*=\s*"([A-Z_]+)"/gmu)].map((match) => match[1])),
			keyRoster: [
				"ContractStorageRelationKey", "parent_kind", "parent_digest", "secondary_parent_kind", "secondary_parent_digest",
				"child_kind", "child_digest", "target_digest", "attempt_digest", "start_claim_digest", "classifier_profile_digest",
			].every((anchor) => functionBody(nonhead, "contractRelationMaterial").includes(anchor)),
			algebraClosed: ["relationAttemptTarget", "relationTargetRun", "relationRunExecution"].every((anchor) =>
				functionBody(nonhead, "contractRelationAlgebraValid").includes(anchor)) &&
				["targetAttemptDigest", "finalizedRunJoins", "executionJoins"].every((anchor) =>
					functionBody(nonhead, "validateContractRelation").includes(anchor)),
			persistenceOrder: ordered(functionBody(nonhead, "persistContractRecord"), [
				"publishLocked", "createExactHardLink", "relationDirectoryLocked", "createExactPrivateFile",
				"reopenContractWitness", "readExactPrivateFile", "store.assertReady",
			]),
			openConvergence: ordered(functionBody(nonhead, "openContractRecordByParent"), [
				"relationDirectoryLocked", "filepath.Dir(relationPath)", "readExactPrivateFile", "parseContractRelation",
				"contractRelationMaterial", "createExactPrivateFile",
				"contractExactConverged", "readObjectReference", "reopenContractWitness",
			]) && ["relationDirectoryLocked", "ensureContractDirectoryLocked", "filepath.Dir(record.relationPath)",
				"filepath.Dir(record.witnessPath)", "store.assertReady"].every((anchor) =>
				methodBody(nonhead, "contractStorageRecord", "validForLocked").includes(anchor)),
			runManifestGate: ordered(functionBody(nonhead, "persistFinalizedRunRecord"), [
				"validatePrivateManifestForFinalization", "validatePrivateManifestRoster", "persistContractRecord",
			]),
			executionProfileDerived: functionBody(nonhead, "persistExecutionRecord").includes("contractClassifierProfileDigest()") &&
				!functionBody(nonhead, "persistExecutionRecord").includes("input.profile"),
			identityGuards: count(functionBody(nonhead, "ensureContractDirectoryLocked"), /rejectCaseAlias\s*\(/gu) === 2 &&
				functionBody(nonhead, "createExactHardLink").includes("rejectPathCaseAlias(destination)") &&
				functionBody(nonhead, "createExactPrivateFile").includes("rejectCaseAlias(directory, name)") &&
				count(functionBody(nonhead, "readExactPrivateFile"), /rejectPathCaseAlias\s*\(path\)/gu) === 2,
			identityReplacementTests: [
				"c2ReplaceDirectoryWithExactClone", "os.Link(from, to)",
				"typed-relation-real-parent-replacement", "typed-publication-real-parent-replacement",
				"typed-relation-directory-case-alias", "typed-relation-leaf-case-alias",
				"typed-publication-leaf-case-alias",
			].every((anchor) => allTests.includes(anchor)),
		},
		interlock: {
			bootJoin: ordered(functionBody(interlock, "acquireInterlockAndStartClaim"), [
				"targetBootDigest", "targetBoot != bootDigest", "openAndLockStudy",
			]),
			acquisitionOrder: ordered(functionBody(interlock, "acquireInterlockAndStartClaim"), [
				"openAndLockStudy", "os.Lstat(claimPath)", "readExecutionInterlockState", "clearReceiptExactLocked",
				"replaceExecutionInterlock", "createExactPrivateFile", "readExactPrivateFile",
				"readExecutionInterlockState", "startClaimWinner{",
			]),
			releaseOrder: ordered(functionBody(interlock, "releaseInterlockAfterFinalizedRun"), [
				"validatePrivateManifestLocked", "validatePrivateManifestRoster", "readExecutionInterlockState",
				"replaceExecutionInterlock", "persistClearReceiptLocked",
			]),
			resetOrder: ordered(functionBody(interlock, "resetInterlockAfterBootChange"), [
				"current.bootDigest == authorization.currentBoot", "replaceExecutionInterlock", "persistClearReceiptLocked",
			]),
			winnerFreshAndConsumed: ordered(functionBody(interlock, "validForStoreLocked"), [
				"store.assertReady", "winner.claim.validForLocked", "readExecutionInterlockState",
			]) && ordered(functionBody(interlock, "consumeStartClaimWinner"), [
				"store.assertReady", "openAndLockStudy", "validForStoreLocked", "winner.seal.consumed = true",
			]),
			startClaimConvergence: ordered(functionBody(interlock, "openStartClaim"), [
				"store.assertReady", "openAndLockStudy", "readExactPrivateFile", "parseStartClaim",
				"createExactPrivateFile", "contractExactConverged",
			]),
			clearReceiptTransition: [
				"previous_target_digest", "previous_attempt_digest", "previous_boot_digest", "previous_generation",
				"previous_revision", "deriveInterlockGeneration", "sameInterlockState(expectedClear, clear)",
			].every((anchor) => interlock.includes(anchor)),
			clearReceiptConvergence: ordered(functionBody(interlock, "clearReceiptExactLocked"), [
				"value.CanonicalChecked", "createExactPrivateFile", "contractExactConverged", "store.assertReady",
			]),
			clearReceiptFaults: [
				"before-clear-receipt-create", "after-clear-receipt-temporary-sync", "before-clear-receipt-link",
				"after-clear-receipt-link", "after-clear-receipt-directory-sync", "before-clear-receipt-reopen",
			].every((anchor) => interlock.includes(`\"${anchor}\"`)),
			identityBoundary: functionBody(interlock, "replaceExecutionInterlock").includes("rejectPathCaseAlias(path)") &&
				functionBody(interlock, "acquireInterlockAndStartClaim").includes("rejectPathCaseAlias(claimPath)") && [
				"start-claim-real-parent-replacement-preserves-unconsumed-winner",
				"start-claim-directory-case-alias-preserves-winner",
				"start-claim-leaf-case-alias-preserves-winner",
				"case-alias-claim-preflight-does-not-create-interlock",
				"fixed-ops-real-directory-replacement-refuses-consume",
				"fixed-ops-case-alias-refuses-consume",
			].every((anchor) => allTests.includes(anchor)),
			productionSeals: {
				attempt: count(newProduction, /&attemptRecordSeal\{marker:\s*1\}/gu),
				terminal: count(newProduction, /&terminalClosureSeal\{marker:\s*1\}/gu),
				reset: count(newProduction, /&changedBootResetSeal\{marker:\s*1\}/gu),
				lease: count(newProduction, /&interlockLeaseSeal\{marker:\s*1\}/gu),
				winner: count(newProduction, /&startClaimWinnerSeal\{marker:\s*1\}/gu),
				manifest: count(newProduction, /&privateManifestSeal\{marker:\s*1\}/gu),
			},
			testSeals: {
				attempt: count(allTests, /&attemptRecordSeal\{marker:\s*1\}/gu),
				terminal: count(allTests, /&terminalClosureSeal\{marker:\s*1\}/gu),
				reset: count(allTests, /&changedBootResetSeal\{marker:\s*1\}/gu),
			},
		},
		privateEvidence: (() => {
			const purgeBody = functionBody(privateRun, "purgePrivateEvidence");
			const freshPurgeBody = balancedBody(purgeBody, /if\s+!purgeDurable\s*/u);
			return {
			limits: /maxPrivateBlobs\s*=\s*16\b/u.test(privateRun) &&
				/maxPrivateEvidenceBytes\s*=\s*64\s*\*\s*1024\s*\*\s*1024\b/u.test(privateRun),
			kinds: [...privateKindsBody.matchAll(/"([A-Z_]+)"/gu)].map((match) => match[1]),
			states: sorted([...privateRun.matchAll(/privateState[A-Za-z]+\s*=\s*"([A-Z_]+)"/gmu)].map((match) => match[1])),
			manifestClosure: ordered(functionBody(privateRun, "openPrivateManifest"), [
				"parsePrivateManifest", "record.targetDigest", "record.attemptDigest", "record.startClaimDigest", "record.validFor",
			]) && ordered(functionBody(privateRun, "reopenPrivateManifestLocked"), [
				"privateRunDirectoriesLocked", "strictDigestHex", "filepath.Join(manifestDirectory, hex)",
				"filepath.Join(packDirectory, hex)", "filepath.Join(purgeDirectory, hex)",
				"readExactPrivateFile", "samePrivateManifestEntries",
			]),
			manifestOpenConvergence: ordered(functionBody(privateRun, "openPrivateManifest"), [
				"readExactPrivateFile", "parsePrivateManifest", "createExactPrivateFile", "contractExactConverged", "record.validFor",
			]),
			arithmeticBounded: functionBody(privateRun, "parsePrivateManifest").includes("countedBytes > maxPrivateEvidenceBytes-count") &&
				functionBody(privateRun, "validatePrivatePack").includes("entry.count > record.packBytes-entry.offset"),
			availabilityJoins: ordered(functionBody(privateRun, "privateAvailability"), [
				"run.validForLocked", "validatePrivateManifestRoster", "reopenPrivateManifestLocked",
				"readExactPrivateFile", "createExactPrivateFile", "validatePrivatePack",
			]),
			purgeOrder: ordered(purgeBody, [
				"run.validForLocked", "validatePrivateManifestRoster", "reopenPrivateManifestLocked",
				"privatePurgeIntent", "createExactPrivateFile", "purgeDurable = true", "os.Remove", "syncDirectory",
			]),
			purgeCreateCount: count(purgeBody, /\bcreateExactPrivateFile\s*\(/gu) === 2,
			freshPurgeOrder: ordered(freshPurgeBody, [
				"faultBeforePurgeIntent", "createExactPrivateFile", "faultAfterPurgeIntentSync", "readExactPrivateFile",
			]),
				missingPackGate: purgeBody
					.includes("else if err := validatePrivatePack(manifest.packPath, manifest); err != nil"),
				noCanonicalMutation: !/\b(?:publishLocked|advanceHead|CreateStudy|OpenHead)\s*\(/u.test(privateRun),
				identityGuards: functionBody(privateRun, "validatePrivateManifestLocked")
					.includes("rejectPathCaseAlias(record.purgePath)") &&
					count(functionBody(privateRun, "openPrivateManifest"), /rejectPathCaseAlias\s*\(/gu) === 3 &&
					count(functionBody(privateRun, "createPrivateManifest"), /rejectPathCaseAlias\s*\(/gu) === 3 &&
					["manifestAliasErr", "packAliasErr", "purgeAliasErr"].every((anchor) =>
						functionBody(privateRun, "reopenPrivateManifestLocked").includes(anchor)) &&
					count(functionBody(privateRun, "validatePrivatePack"), /rejectPathCaseAlias\s*\(path\)/gu) === 2 &&
					ordered(purgeBody, ["rejectPathCaseAlias(manifest.packPath)", "os.Remove(manifest.packPath)"]) &&
					functionBody(privateRun, "createExactPrivatePack").includes("rejectCaseAlias(directory, name)"),
				identityReplacementTests: [
					"dynamic-private-real-directory-replacement-refuses",
					"private-case-alias-preflight-is-known-no-effect",
					"private-manifest-leaf-case-alias-refuses", "private-pack-leaf-case-alias-refuses",
				].every((anchor) => allTests.includes(anchor)),
			};
		})(),
	});
}

export async function collectC3Facts() {
	const packageValues = goList([
		"./internal/contractexec", "./internal/gitobj", "./internal/hostepoch", "./internal/noderuntime",
	]);
	const entries = await Promise.all(c3ReviewedPaths.map(readC2Source));
	const sources = Object.fromEntries(entries.map((entry) => [entry.path, entry.source]));
	const c2Facts = await collectC2Facts();
	const productionPaths = Object.keys(expectedC3SourceImports);
	const target = sources["internal/contractexec/target.go"];
	const targetTest = sources["internal/contractexec/target_test.go"];
	const singleTarget = sources["internal/gitobj/single_target.go"];
	const singleTargetTest = sources["internal/gitobj/single_target_test.go"];
	const epoch = sources["internal/hostepoch/epoch.go"];
	const epochSource = sources["internal/hostepoch/source_darwin_cgo.go"];
	const runtime = sources["internal/noderuntime/runtime.go"];
	const runtimeIdentity = sources["internal/noderuntime/identity_darwin.go"];
	const runtimeProbe = sources["internal/noderuntime/probe_darwin.go"];
	const storeBridge = sources["internal/store/nonhead_contract.go"];
	const c3TestFiles = Object.fromEntries(entries
		.filter((entry) => entry.path.endsWith("_test.go"))
		.map((entry) => [
			entry.path,
			sorted([...entry.source.matchAll(/^func\s+(TestC3[A-Za-z0-9_]+)\s*\(/gmu)].map((match) => match[1])),
		]));
	const packageFacts = Object.fromEntries(packageValues.map((value) => [value.ImportPath, packageBuildFacts(value)]));
	const sourceImports = Object.fromEntries(productionPaths.map((path) => [path, goImports(sources[path])]));
	const sourceSurfaces = Object.fromEntries(productionPaths.map((path) => [path, exportedSurface(sources[path])]));
	const targetCompositionBodies = [
		functionBody(target, "PublishOfficialTarget"), functionBody(target, "publishOfficialTarget"),
		functionBody(target, "OpenOfficialTarget"),
		functionBody(target, "rejoinOfficial"),
	].join("\n");
	const targetPublicationBodies = [
		functionBody(target, "publishOfficialTarget"), functionBody(target, "rejoinOfficial"),
	].join("\n");
	return structuredClone({
		packages: packageFacts,
		buildTags: Object.fromEntries(Object.keys(expectedC3BuildTags).map((path) => [path, goBuildTag(sources[path])])),
		sourceImports,
		sourceSurfaces,
		testFiles: c3TestFiles,
		c2Problems: validateC2Facts(c2Facts),
		storeBridge: {
			modelImporters: c2Facts.modelImporters,
			exports: c2Facts.newProductionExports?.["internal/store/nonhead_contract.go"],
			rootRoster: ordered(storeBridge, [
				'"candidate-parent"', '"evidence"', '"fixture"', '"home"', '"state"', '"tmp"',
				'"xdg-cache"', '"xdg-config"', '"xdg-data"', '"xdg-state"',
			]),
			markerContract: [
				'contractAttemptV1     = "contract-conformance-attempt/v1"',
				'contractAttemptsDir   = "conformance-attempts"',
				'contractAttemptMarker = "attempt.marker.json"',
				'"allocation_profile": "PRIVATE_FRESH_ROOT_V1"',
				'"marker_ordering": "DURABLE_BEFORE_SPAWN"',
			].every((anchor) => storeBridge.includes(anchor)),
			freshNonce: functionBody(storeBridge, "AllocateConformanceAttempt").includes("rand.Read(random[:])") &&
				functionBody(storeBridge, "buildConformanceAttempt").includes("hex.DecodeString(nonce)"),
			exactReopen: ordered(functionBody(storeBridge, "openConformanceAttemptLocked"), [
				"readExactPrivateFile", "parseConformanceAttempt", "NewSemanticObject", "reopenExactObject",
				"inspectAttemptRoots", "validForLocked",
			]),
			targetJoin: ordered(functionBody(storeBridge, "PersistContractTargetRecord"), [
				"attemptJoinsTarget", "NewSemanticObject", "persistTargetRecord", "openTargetByAttempt",
			]) && ordered(functionBody(storeBridge, "OpenContractTargetRecord"), [
				"openTargetByAttempt", "parseContractTargetStorageRecord", "attemptJoinsTarget",
			]) && functionBody(storeBridge, "parseContractTargetStorageRecord").includes("contractmodel.ParseContractExecutionTarget"),
			noIssuerOrProcessEdge: !/\b(?:exec\.Command(?:Context)?|os\.StartProcess|RunPermit|StartClaimWinner)\s*\(/u.test(storeBridge),
		},
		gitTarget: {
			directOpaqueSource: /source\s+InspectedTree/u.test(goCodeOnly(singleTarget)) &&
				!["WorldPlan", "BoundCandidate", "WorldInstance"].some((anchor) => goCodeOnly(singleTarget).includes(anchor)),
			publicationOrder: ordered(functionBody(singleTarget, "materializeSingleTarget"), [
				"requirePrivateEmptyParent", "Inspect", "sameInspectedTree", "requirePrivateEmptyParent",
				"os.MkdirTemp", "reserveTopology", "writeVerifiedBlob", "stagedPortableDigest",
				"buildMaterializationManifest", "syncTreeDirectories", "writeDurableExclusiveFile",
				"publishStagedDirectory", "ReopenSingleTarget",
			]),
			reopenOrder: ordered(functionBody(singleTarget, "ReopenSingleTarget"), [
				"exactPublishedParent", "requireExactParentRoster", "Inspect", "sameInspectedTree",
				"requireExactParentRoster", "buildMaterializationManifest", "validateSingleTargetManifest",
				"validateSingleTargetTopology", "syncTreeDirectories", "syncDirectory", "requireExactParentRoster",
				"materializationReceiptSeal",
			]),
			ambiguousRecovery: count(singleTarget, /CodePublicationAmbiguous/gu) >= 4 &&
				["syncFailure", "rollbackFailure", "rollbackSyncFailure", "ReopenSingleTarget"].every((anchor) => singleTargetTest.includes(anchor)),
			dirtyWorktreeTest: singleTargetTest.includes("TestC3SingleTargetExcludesDirtyWorktreeBytes") &&
				singleTargetTest.includes("untracked-secret.txt"),
		},
		hostEpoch: {
			twoSamples: count(functionBody(epoch, "measureWithSource"), /\bsource\s*\(\)/gu) === 2,
			canonicalUUID: ["len(raw) != len(canonical)", "index == 8 || index == 13 || index == 18 || index == 23",
				"value + ('a' - 'A')", "!nonzero"].every((anchor) => functionBody(epoch, "canonicalUUID").includes(anchor)),
			typedDigest: functionBody(epoch, "measureWithSource").includes('canon.DigestBytes("DarwinBootSessionIdentity", first[:])'),
			fixedDarwinSource: epochSource.includes('sysctlbyname("kern.bootsessionuuid"') &&
				epochSource.includes("int(length) != len(buffer)") && epochSource.includes("buffer[len(buffer)-1] != 0"),
			noFallbackOrProcess: !/(?:kern\.boottime|os\/exec|exec\.Command|StartProcess|time\.Now|os\.Getpid)/u.test(`${epoch}\n${epochSource}`),
		},
		nodeRuntime: {
			spawnOwners: productionPaths.filter((path) => goImports(sources[path]).includes("os/exec")),
			noAmbientPath: !/(?:\bLookPath\b|"PATH=)/u.test(`${runtime}\n${runtimeIdentity}\n${runtimeProbe}`),
			measureProbeMeasure: ordered(functionBody(runtime, "measureProbeMeasure"), [
				"resolvePrivateProbeParent", "operations.measure", "operations.probe", "operations.measure",
				"resolvePrivateProbeParent", "before.equal(after)", "measured.valid(resolved)", "nodeProbeDigest()",
			]),
			fixedProbe: runtimeProbe.includes("exec.CommandContext(probeContext, executable, \"--eval\", nodeProbeProgram)") &&
				runtimeProbe.includes('command.Env = []string{"HOME=" + home, "TMPDIR=" + tmp, "LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1"}') &&
				runtimeProbe.includes("context.WithTimeout(ctx, 5*time.Second)") &&
				runtimeProbe.includes("stdout := &boundedBuffer{limit: 8192}") &&
				runtimeProbe.includes("stderr := &boundedBuffer{limit: 4096}") &&
				runtimeProbe.includes("command.WaitDelay = time.Second"),
			executableIdentity: ["syscall.O_NOFOLLOW", "syscall.Fstat", "descriptorPath", "io.LimitReader",
				"executableIdentityFromStat(after) != identity"].every((anchor) => runtimeIdentity.includes(anchor)),
		},
		officialTarget: {
			publishOrder: ordered(functionBody(target, "publishOfficialTarget"), [
				"reopenResidueHead", "policyForResidue", "Repository.Pin", "AllocateConformanceAttempt", "gitobj.Inspect",
				"sourceProfileJoinsTree", "gitobj.MaterializeSingleTarget", "gitobj.ReopenSingleTarget", "noderuntime.Admit",
				"runtimeAuthority.Revalidate", "hostepoch.Measure", "reopenResidueHead", "buildTarget",
				"PersistContractTargetRecord", "rejoinOfficial",
			]),
			openOrder: ordered(functionBody(target, "OpenOfficialTarget"), [
				"reopenResidueHead", "policyForResidue", "Store.Read", "Store.Validate", "ParseContractExecutionTarget",
				"OpenConformanceAttempt", "OpenContractTargetRecord", "Repository.Pin", "gitobj.Inspect",
				"sourceProfileJoinsTree", "gitobj.ReopenSingleTarget", "treeBindingMatches", "noderuntime.Admit",
				"runtimeBindingMatches", "hostepoch.Measure", "rejoinOfficial",
			]),
			finalRejoin: ordered(functionBody(target, "rejoinOfficial"), [
				"reopenResidueHead", "policyForResidue", "gitobj.ReopenSingleTarget", "runtime.Revalidate",
				"epoch.Revalidate", "OpenConformanceAttempt", "OpenContractTargetRecord", "issuedOfficialTargetSeal",
			]),
			faultCoverage: ordered(targetPublicationBodies, [
				"officialFaultAfterInputsValidated", "officialFaultAfterInitialResidueReopen",
				"officialFaultAfterPolicyDerivation", "officialFaultAfterTreePin", "officialFaultAfterAttemptAllocation",
				"officialFaultAfterTreeInspection", "officialFaultAfterEntrypointJoin", "officialFaultAfterMaterialization",
				"officialFaultAfterMaterializationReopen", "officialFaultAfterRuntimeAdmission",
				"officialFaultAfterRuntimeRevalidation", "officialFaultAfterEpochMeasurement",
				"officialFaultAfterTerminalResidueJoin", "officialFaultAfterTargetBuild",
				"officialFaultAfterTargetRecordPersistence", "officialFaultAfterProvisionalValidation",
				"officialFaultAfterFinalResidueJoin", "officialFaultAfterFinalPolicyJoin",
				"officialFaultAfterFinalMaterializationJoin", "officialFaultAfterFinalRuntimeJoin",
				"officialFaultAfterFinalEpochJoin", "officialFaultAfterFinalAttemptJoin",
				"officialFaultAfterFinalTargetRecordJoin", "officialFaultBeforeAuthoritySeal",
			]),
			treeJoinClosed: [
				"receipt.PublishedRoot", "receipt.ManifestPath", "receipt.PortableTreeDigest", "receipt.PolicyDigest",
				"receipt.ObjectFormat", "receipt.RepositoryFingerprint", "receipt.Entries[index]", "sourceEntries[index]",
			].every((anchor) => functionBody(target, "treeBindingMatches").includes(anchor)),
			preSpawnOnly: !/\b(?:exec\.Command(?:Context)?|os\.StartProcess|acquireInterlock|RunPermit|StartClaim|persistFinalizedRun|persistExecution)\s*\(/u.test(targetCompositionBodies),
			headPreservationTest: ["c3HeadSnapshot", "os.SameFile(beforeInfo, afterInfo)", "bytes.Equal(beforeBytes, afterBytes)"].every((anchor) => targetTest.includes(anchor)),
		},
		predecessor: JSON.parse(sources["spec/verification/p07b-c-c3-predecessors.json"]),
		predecessorAuthority: collectC3PredecessorAuthority(),
	});
}

export async function collectC4Facts() {
	const packageValues = goList([
		"./internal/processmechanics", "./internal/contractexec/runner", "./testkit/contractexec/cli",
	]);
	const repositoryPackages = goList(["./..."]);
	const entries = await Promise.all(c4ReviewedPaths.map(readC2Source));
	const sources = Object.fromEntries(entries.map((entry) => [entry.path, entry.source]));
	const c3Facts = await collectC3Facts();
	const claimMap = await collectC4ClaimMapFacts();
	const mechanics = [
		"internal/processmechanics/capture.go", "internal/processmechanics/process.go",
		"internal/processmechanics/process_darwin.go", "internal/processmechanics/process_unsupported.go",
	].map((path) => sources[path]).join("\n");
	const mechanicsCommon = sources["internal/processmechanics/process.go"];
	const mechanicsDarwin = sources["internal/processmechanics/process_darwin.go"];
	const mechanicsUnsupported = sources["internal/processmechanics/process_unsupported.go"];
	const runnerCLI = sources["internal/contractexec/runner/cli.go"];
	const runnerCommon = sources["internal/contractexec/runner/runner.go"];
	const runnerDarwin = sources["internal/contractexec/runner/runner_darwin.go"];
	const runnerEvidence = sources["internal/contractexec/runner/evidence.go"];
	const runnerScope = sources["internal/contractexec/runner/scope_darwin.go"];
	const runnerDarwinTests = sources["internal/contractexec/runner/runner_darwin_test.go"];
	const cliFixture = sources["testkit/contractexec/cli/fixture_darwin.go"];
	const cliPhysicalTests = sources["testkit/contractexec/cli/runner_darwin_test.go"];
	const cliClosureTestBody = functionBody(cliPhysicalTests, "TestCLIContractExecutionClosesStandaloneScope");
	const cliChildBindingTestBody = functionBody(cliPhysicalTests, "TestCLIContractExecutionChildBindingEvidenceStates");
	const cliTargetMutationTestBody = functionBody(cliPhysicalTests, "TestCLIContractExecutionTargetMutationBlocksFinalization");
	const cliReferenceExecutionBody = functionBody(cliPhysicalTests, "assertReferenceContractExecution");
	const cliConformingExecutionBody = functionBody(cliPhysicalTests, "assertConformingContractExecution");
	const cliForbiddenTestBody = functionBody(cliPhysicalTests, "TestCLIContractExecutionForbiddenPositiveControls");
	const cliPhysicalSchedulerBody = functionBody(cliPhysicalTests, "runC4PhysicalTest");
	const storeNonhead = sources["internal/store/nonhead_contract.go"];
	const storeNonheadTests = sources["internal/store/nonhead_contract_test.go"];
	const runnerSources = [
		"internal/contractexec/runner/cli.go", "internal/contractexec/runner/evidence.go",
		"internal/contractexec/runner/runner.go", "internal/contractexec/runner/runner_darwin.go",
		"internal/contractexec/runner/runner_unsupported.go", "internal/contractexec/runner/scope_darwin.go",
	].map((path) => sources[path]).join("\n");
	const storeBridge = sources["internal/store/contract_run_bridge.go"];
	const interlock = sources["internal/store/execution_interlock.go"];
	const worldDarwin = sources["internal/world/process_darwin.go"];
	const worldAdapter = functionBody(worldDarwin, "runPlatformProcess");
	const packageFacts = Object.fromEntries(packageValues.map((value) => [value.ImportPath, packageBuildFacts(value)]));
	const testFiles = {};
	for (const path of Object.keys(expectedC4TestsByFile)) {
		const pattern = path.startsWith("internal/store/") ? /^func\s+(TestC4[A-Za-z0-9_]+)\s*\(/gmu :
			/^func\s+(Test[A-Za-z0-9_]+)\s*\(/gmu;
		testFiles[path] = sorted([...sources[path].matchAll(pattern)]
			.map((match) => match[1]).filter((name) => name !== "TestMain"));
	}
	const aggregateSurface = (paths) => sorted([...new Set(paths.flatMap((path) => exportedSurface(sources[path]))) ]);
	const consumeBody = functionBody(runnerDarwin, "consumeAndStart");
	const executeBody = functionBody(runnerDarwin, "executeCLI");
	const preparedTargetBody = functionBody(runnerDarwin, "samePreparedTarget");
	const preparedTargetIdentityFields = structFields(runnerEvidence, "preparedTargetIdentity");
	const preparedTargetIdentityGuardBody = functionBody(runnerDarwinTests, "assertPreparedTargetIdentityGuards");
	const prepareCLIBody = functionBody(runnerCLI, "prepareCLIExecution");
	const closeStartErrorBody = functionBody(runnerDarwin, "closeStartError");
	const persistBody = functionBody(runnerDarwin, "persistRunAndClassification");
	const resumeBody = functionBody(runnerDarwin, "resumeCLIClassification");
	const releaseBody = functionBody(interlock, "releaseInterlockAfterFinalizedRun");
	const scopeBody = functionBody(runnerEvidence, "buildScopeDrafts");
	const invocationInspectBody = functionBody(runnerEvidence, "inspectAndRetireCLIInvocationEvidence");
	const invocationReadBody = functionBody(runnerEvidence, "readCLIInvocationEvidence");
	const invocationRetireBody = functionBody(runnerEvidence, "retireCLIInvocationEvidence");
	const evidenceCapacityBody = functionBody(runnerEvidence, "preflightPrivateEvidenceCapacity");
	const evidenceCapacityValidateBody = methodBody(runnerEvidence, "cliEvidenceCapacity", "validate");
	const evidenceFrameBody = functionBody(runnerEvidence, "framePrivateEvidence");
	const childEvidenceBody = functionBody(runnerEvidence, "buildChildEvidenceDraft");
	const canonicalEvidenceBody = functionBody(runnerEvidence, "canonicalEvidence");
	const capacityTestBody = functionBody(runnerDarwinTests, "assertPrivateEvidenceCapacityAndFrame");
	const boundedResidueTestBody = functionBody(runnerDarwinTests, "assertBoundedForeignEvidenceAndScopeResidue");
	const assembleBody = methodBody(runnerEvidence, "evidenceDraft", "persistAndAssemble");
	const candidateInventoryEntryBody = goCodeOnly(functionBody(runnerScope, "snapshotCandidate"));
	const candidateInventoryBody = goCodeOnly(functionBody(runnerScope, "snapshotCandidateWithLimits"));
	const candidateDirectoryBody = goCodeOnly(functionBody(runnerScope, "readCandidateInventoryDirectory"));
	const candidateFileBody = goCodeOnly(functionBody(runnerScope, "digestCandidateInventoryFile"));
	const candidateInventorySource = goCodeOnly(runnerScope);
	const fixtureMaterializeBody = functionBody(runnerCLI, "materializeCLIFixtures");
	const fixtureValidateBody = functionBody(runnerCLI, "validateExistingCLIFixture");
	const fixtureWriteBody = functionBody(runnerCLI, "writeExactCLIFixture");
	const sharedSourceRepositoryBody = functionBody(cliFixture, "sharedSourceRepository");
	const newTargetBody = functionBody(cliFixture, "newTarget");
	const newReferenceTargetBody = functionBody(cliFixture, "NewReferenceTarget");
	const newControlTargetBody = functionBody(cliFixture, "NewTarget");
	const newMutationTargetBody = functionBody(cliFixture, "NewMutationTarget");
	const sharedControlResidueBody = functionBody(cliFixture, "sharedControlResidue");
	const newForbiddenTargetBody = functionBody(cliFixture, "NewForbiddenTarget");
	const sharedForbiddenResidueBody = functionBody(cliFixture, "sharedForbiddenResidue");
	const fixtureRunMainBody = functionBody(cliFixture, "RunMain");
	const scopeProbeBody = functionBody(runnerScope, "newScopeProbe");
	const shortSocketBody = functionBody(runnerScope, "newShortSocketRoot");
	const canaryCreateBody = functionBody(runnerScope, "newUnixCanary");
	const canaryCloseBody = methodBody(runnerScope, "unixCanary", "close");
	const canaryClassifyBody = functionBody(runnerScope, "classifyCanaryAcceptError");
	const boundedRosterBody = functionBody(runnerScope, "boundedDirectRoster");
	const removeRetainedLeafBody = functionBody(runnerScope, "removeRetainedLeaf");
	const removeRetainedDirectoryBody = functionBody(runnerScope, "removeRetainedEmptyDirectory");
	const scopeFinishBody = methodBody(runnerScope, "scopeProbe", "finish");
	const storeBoundedRosterBody = functionBody(storeNonhead, "boundedExactDirectoryNames");
	const inspectAttemptRootsBody = functionBody(storeNonhead, "inspectAttemptRoots");
	const boundedStoreTestBody = functionBody(storeNonheadTests, "TestC3ConformanceAttemptIsFreshDurableAndRestartReopenable");
	const acquireCallSites = await productionStoreOwnerReferenceSites(repositoryPackages);
	return structuredClone({
		packages: packageFacts,
		buildTags: Object.fromEntries(Object.keys(expectedC4BuildTags).map((path) => [path, goBuildTag(sources[path])])),
		testFiles,
		c3Problems: validateC3Facts(c3Facts),
		mechanics: {
			surface: aggregateSurface([
				"internal/processmechanics/capture.go", "internal/processmechanics/process.go",
				"internal/processmechanics/process_darwin.go",
			]),
			invocationFields: structFields(mechanicsCommon, "Invocation"),
			preparedFields: structFields(mechanicsCommon, "Prepared"),
			preparedStateFields: structFields(mechanicsCommon, "preparedState"),
			runningFields: structFields(mechanicsDarwin, "Running"),
			runningStateFields: structFields(mechanicsDarwin, "runningState"),
			unsupportedRunningFields: structFields(mechanicsUnsupported, "Running"),
			stdinFields: structFields(mechanicsCommon, "Stdin"),
			defensiveInputCopy: [
				"argv:        append([]string(nil), argv...)",
				"environment: append([]string(nil), environment...)",
				"stdin:       Stdin{presence: stdin.presence, bytes: append([]byte(nil), stdin.bytes...)}",
			].every((anchor) => functionBody(mechanicsCommon, "NewInvocation").includes(anchor)) &&
				ordered(functionBody(mechanicsCommon, "Prepare"), ["NewInvocation", "invocation.binding", "copyInvocation.binding"]),
			oneShot: functionBody(mechanicsCommon, "begin").includes("prepared.state.started.CompareAndSwap(false, true)") &&
				count(functionBody(mechanicsDarwin, "startPlatform"), /prepared\.begin\s*\(/gu) === 1 &&
				count(functionBody(mechanicsUnsupported, "startPlatform"), /prepared\.begin\s*\(/gu) === 1,
			copySafeState: methodBody(mechanicsDarwin, "Running", "Close").includes("state.closeOnce.Do") &&
				methodBody(mechanicsDarwin, "Running", "Close").includes("state.closePhysical()") &&
				methodBody(mechanicsDarwin, "Running", "abortBeforeClose").includes("state.mu.Lock()") &&
				functionBody(mechanicsDarwin, "startPlatform").includes("&Running{state: &runningState{") &&
				functionBody(mechanicsCommon, "Prepare").includes("&Prepared{state: &preparedState{") &&
				/type\s+Prepared\s+struct\s*\{\s*state\s+\*preparedState\s*\}/u.test(goCodeOnly(mechanicsCommon)) &&
				/type\s+Running\s+struct\s*\{\s*state\s+\*runningState\s*\}/u.test(goCodeOnly(mechanicsDarwin)),
			directSpawn: ordered(functionBody(mechanicsDarwin, "startPlatform"), [
				"command := &exec.Cmd{", "Path:        invocation.executable", "Args:        append([]string(nil), invocation.argv...)",
				"Env:         append([]string(nil), invocation.environment...)", "SysProcAttr: &syscall.SysProcAttr{Setpgid: true}",
				"command.Start()", "time.NewTimer(invocation.limits.Execution)",
			]) && !/(?:exec\.Command|LookPath|\/bin\/sh)/u.test(goCodeOnly(functionBody(mechanicsDarwin, "startPlatform"))),
			resultCopies: functionBody(mechanicsCommon, "cloneResult").includes("append([]byte(nil), result.Stdout...)") &&
				functionBody(mechanicsCommon, "cloneResult").includes("append([]byte(nil), result.Stderr...)") &&
				methodBody(mechanicsCommon, "StartError", "Result").includes("cloneResult") &&
				methodBody(mechanicsDarwin, "Running", "Close").includes("cloneResult"),
			noSemanticAuthority: ![
				contractPackagePath, packagePath, `${modulePath}/internal/domain`, storePackagePath,
			].some((path) => mechanics.includes(`\"${path}\"`)) &&
				!/(?:OfficialTarget|RunPermit|ContractExecution|StartClaim)/u.test(goCodeOnly(mechanics)),
			importers: sorted(repositoryPackages
				.filter((entry) => (entry.Imports ?? []).includes(processMechanicsPackagePath))
				.map((entry) => entry.ImportPath)),
		},
		worldAdapter: {
			chronology: ordered(worldAdapter, [
				"request.tool.revalidate", "processmechanics.NewInvocation", "processmechanics.Prepare",
				"prepared.Start(ctx)", "running.Close()",
			]),
			noDuplicateSpawn: !/(?:newDirectCommand|directExecOnly|exec\.Command|&exec\.Cmd)/u.test(goCodeOnly(worldAdapter)) &&
				!/(?:newDirectCommand|directExecOnly)/u.test(goCodeOnly(worldDarwin)),
			httpMechanicsRetained: ["teardownOwnedProcessGroup", "performFinalGroupProbe", "classifyWait"]
				.every((name) => functionBody(worldDarwin, name).length > 0),
		},
			runner: {
			surface: aggregateSurface([
				"internal/contractexec/runner/cli.go", "internal/contractexec/runner/evidence.go",
				"internal/contractexec/runner/runner.go", "internal/contractexec/runner/runner_darwin.go",
				"internal/contractexec/runner/scope_darwin.go",
			]),
			closedEntrypoints: /func\s+ExecuteCLI\s*\(\s*ctx\s+context\.Context,\s*retained\s+contractexec\.OfficialTarget,?\s*\)\s*\(store\.ContractExecutionRecord,\s*error\)/mu.test(runnerCommon) &&
				/func\s+ResumeCLIClassification\s*\(\s*ctx\s+context\.Context,\s*retained\s+contractexec\.OfficialTarget,?\s*\)\s*\(store\.ContractExecutionRecord,\s*error\)/mu.test(runnerCommon),
			permitAdjacent: count(runnerSources, /\.ConsumeForStart\s*\(/gu) === 1 &&
				count(runnerSources, /prepared\.Start\s*\(/gu) === 1 &&
				ordered(consumeBody, ["owner.ConsumeForStart(closureContext, binding)", "return prepared.Start(processContext)"]) &&
				/\}\s*return prepared\.Start\(processContext\)\s*$/u.test(goCodeOnly(consumeBody).trim()),
			spawnAdjacentRevalidation: count(executeBody, /contractexec\.ReopenOfficialTarget\s*\(/gu) === 4 &&
				count(executeBody, /samePreparedTarget\s*\(/gu) === 3 &&
				count(closeStartErrorBody, /samePreparedTarget\s*\(/gu) === 1 &&
				!runnerSources.includes("sameFreshTarget") &&
				exact(preparedTargetIdentityFields, ["model", "bundle", "roots", "candidateRoot", "markerPath"]) &&
				ordered(executeBody, [
					"store.AcquireContractRunOwner", "terminalClosureContext",
					"contractexec.ReopenOfficialTarget(closureContext, input.target)",
					"samePreparedTarget(input.cliExecutionPreparation, spawnTarget)", "input.target = spawnTarget", "consumeAndStart",
				]) && ordered(prepareCLIBody, [
					"model := target.Model()", "bundle := target.ContractBundle()", "roots := target.Roots()",
					"newPreparedTargetIdentity(model, bundle, roots)", "targetIdentity: identity",
				]) && ordered(preparedTargetBody, [
					"preparation.targetIdentity.valid()",
					"freshModel := fresh.Model()", "bytes.Equal(preparation.targetIdentity.model.CanonicalBytes(), freshModel.CanonicalBytes())",
					"freshRoots := fresh.Roots()", "preparation.targetIdentity.roots.AttemptRoot() == freshRoots.AttemptRoot()",
					"preparation.targetIdentity.roots.CandidateParent() == freshRoots.CandidateParent()",
					"preparation.targetIdentity.roots.FixtureRoot() == freshRoots.FixtureRoot()",
					"preparation.targetIdentity.roots.HomeRoot() == freshRoots.HomeRoot()",
					"preparation.targetIdentity.roots.TemporaryRoot() == freshRoots.TemporaryRoot()",
					"preparation.targetIdentity.roots.StateRoot() == freshRoots.StateRoot()",
					"preparation.targetIdentity.roots.EvidenceRoot() == freshRoots.EvidenceRoot()",
				]) && [
					"samePreparedTarget(cliExecutionPreparation{}, primary.target)",
					"contractexec.ReopenOfficialTarget(context.Background(), primary.target)",
					"!samePreparedTarget(primary.cliExecutionPreparation, reopened)",
					"reopened.Close()", "samePreparedTarget(primary.cliExecutionPreparation, reopened)",
					"primaryModel.Input().Tree != independentModel.Input().Tree",
					"primaryModel.Input().Attempt == independentModel.Input().Attempt",
					"primary.targetIdentity.roots.AttemptRoot() == independent.targetIdentity.roots.AttemptRoot()",
					"samePreparedTarget(primary.cliExecutionPreparation, independent.target)",
					"primaryModel, primary.targetIdentity.bundle, independent.targetIdentity.roots",
					"crossRootPreparation.targetIdentity = crossRootIdentity",
					"samePreparedTarget(crossRootPreparation, primary.target)",
				].every((anchor) => preparedTargetIdentityGuardBody.includes(anchor)) && ordered(functionBody(runnerDarwinTests, "assertBoundedForeignEvidenceAndScopeResidue"), [
					"evidenceInput := cliExecutionInput{", "assertPreparedTargetIdentityGuards(t, primary, evidenceInput)",
					"foreignEvidence := filepath.Join",
				]) && ordered(functionBody(runnerDarwinTests, "assertTargetReopenImmediatelyPrecedesPermitPath"), [
					'exactCallAssignment(window[0], token.DEFINE, []string{"spawnTarget", "err"}',
					'"contractexec.ReopenOfficialTarget", "closureContext", "input.target")',
					"!errGuard(window[1])", "!freshnessGuard(window[2])", "!targetReplacement(window[3])",
					"!targetCleanup(window[4])",
					'exactCallAssignment(window[5], token.DEFINE, []string{"physicalObservation", "running", "startErr"}',
					'"consumeAndStart", "ctx", "closureContext", "owner", "input.prepared", "input.binding")',
				]),
			soleOwnerAcquirer: acquireCallSites,
			physicalConforms: ordered(functionBody(cliFixture, "NewConformingTarget"), [
				"newTarget", "sharedConformingResidue", "ReferenceFiles",
			]) && ordered(functionBody(cliFixture, "sharedConformingResidue"), [
				"reference := sharedResidue(t)", "newResidue", "countercli.CLIFieldStdoutJSONMode", '"argv"',
				"countercli.CLIFieldStdoutJSONSource", '"argv"',
				"sharedCompilation.conforming.store == reference.store", "sharedCompilation.conforming.root == reference.root",
				"sharedCompilation.conforming.residue.BundleDigest() == reference.residue.BundleDigest()",
			]) && ordered(functionBody(cliFixture, "decisionForChoicepoint"), [
				"desiredFields", "card.Fields", "field.FieldID", "field.Text", "selectedAlias = card.Alias",
				"AllowedAliases: []string{selectedAlias}", "session.Finalize",
			]) && ordered(functionBody(cliFixture, "newResidue"), [
				"portableChoicepoint", "decisionForChoicepoint", "promotion.OpenRuling",
				"promotion.PreparePortableRuling", "node.PrepareCompilation", "node.CompilePrepared", "node.PublishPrepared",
			]) && ordered(cliReferenceExecutionBody, [
				"clitest.NewReferenceTarget", "runner.ExecuteCLI", "contractmodel.ResultContradicts",
				"contractmodel.ProcessClean", "contractmodel.CaptureProjected", "contractmodel.ScopeComplete",
				"contractmodel.DispositionEligibleClean", "runner.ResumeCLIClassification",
			]) && ordered(cliConformingExecutionBody, [
				"clitest.NewConformingTarget", "runner.ExecuteCLI", "contractmodel.ResultConforms", "contractmodel.ProcessClean",
				"contractmodel.CaptureProjected", "contractmodel.ScopeComplete", "contractmodel.DispositionEligibleClean",
				"conformingWitness.Observation().Tuple()", "len(conformingFields) != 2",
				"countercli.CLIFieldStdoutJSONMode", "countercli.CLIFieldStdoutJSONSource", 'text != "argv"',
			]),
			boundedPhysicalTestConcurrency: /const\s+c4PhysicalTestParallelism\s*=\s*2\b/u.test(cliPhysicalTests) &&
				/var\s+c4PhysicalTestSlots\s*=\s*make\(chan\s+struct\{\},\s*c4PhysicalTestParallelism\)/u.test(cliPhysicalTests) &&
				count(cliPhysicalTests, /\.Parallel\s*\(\s*\)/gu) === 4 &&
				count(cliClosureTestBody, /\.Parallel\s*\(\s*\)/gu) === 3 &&
				count(cliClosureTestBody, /runC4PhysicalTest\s*\(/gu) === 2 &&
				count(cliForbiddenTestBody, /runC4PhysicalTest\s*\(/gu) === 1 &&
				ordered(cliPhysicalSchedulerBody, [
					"c4PhysicalTestSlots <- struct{}{}", "defer func() { <-c4PhysicalTestSlots }()", "exercise()",
				]) && ordered(cliClosureTestBody, [
					"t.Parallel()", 't.Run("reference-contradicts"', "t.Parallel()",
					"runC4PhysicalTest(t, func() { assertReferenceContractExecution(t) })",
					't.Run("physical-conforms"', "t.Parallel()",
					"runC4PhysicalTest(t, func() { assertConformingContractExecution(t) })",
				]) && ordered(cliForbiddenTestBody, [
					"t.Parallel()", "runC4PhysicalTest(t, func() { assertForbiddenPositiveControls(t) })",
				]) && ordered(newReferenceTargetBody, [
					"newTarget", "sharedResidue", "ReferenceFiles",
				]) && ordered(newControlTargetBody, [
					"newTarget", "sharedControlResidue", "files", "label",
				]) && ordered(newMutationTargetBody, [
					"filepath.EvalSymlinks(t.TempDir())", "newResidue", "reference := sharedResidue(t)",
					"mutation.store == reference.store", "mutation.root == reference.root",
					"mutation.residue.BundleDigest() != reference.residue.BundleDigest()", "return newTarget(t, mutation, files, label)",
				]) && ordered(sharedControlResidueBody, [
					"reference := sharedResidue(t)", "sharedCompilation.controlOnce.Do", 'filepath.Join(sharedCompilation.root, "controls")',
					'newResidue(t, parent, "shared C4 hostile-control CLI contract", nil)', "control := sharedCompilation.control",
					"control.store == reference.store", "control.root == reference.root",
					"control.residue.BundleDigest() != reference.residue.BundleDigest()",
					"return control",
				]) && ordered(newForbiddenTargetBody, [
					"newTarget", "sharedForbiddenResidue", "ForbiddenCanaryFiles",
				]) && ordered(sharedForbiddenResidueBody, [
					"reference := sharedResidue(t)", "sharedCompilation.forbiddenOnce.Do", 'filepath.Join(sharedCompilation.root, "forbidden")',
					'newResidue(t, parent, "shared C4 forbidden CLI contract", nil)', "forbidden := sharedCompilation.forbidden",
					"forbidden.store == reference.store", "forbidden.root == reference.root",
					"forbidden.residue.BundleDigest() != reference.residue.BundleDigest()",
					"return forbidden",
				]) && cliForbiddenTestBody.includes("assertForbiddenPositiveControls") &&
				cliForbiddenTestBody.includes("runC4PhysicalTest") &&
				count(cliPhysicalTests, /clitest\.NewTarget\s*\(/gu) === 1 &&
				count(cliChildBindingTestBody, /clitest\.NewTarget\s*\(/gu) === 1 &&
				count(cliPhysicalTests, /clitest\.NewMutationTarget\s*\(/gu) === 1 &&
				count(cliTargetMutationTestBody, /clitest\.NewMutationTarget\s*\(/gu) === 1 &&
				cliPhysicalTests.includes("fixture := clitest.NewForbiddenTarget(t)"),
			immutableSourceCacheIsolation: exact(structFields(cliFixture, "sourceRepositoryFixture"), ["root", "ref"]) &&
				/func\s+sharedSourceRepository\s*\(\s*t\s+testing\.TB,\s*files\s+\[\]gitrepo\.File\s*\)\s+sourceRepositoryFixture/u.test(cliFixture) &&
				ordered(sharedSourceRepositoryBody, [
					"json.Marshal(files)", 'canon.DigestBytes("C4SourceRepositoryFixture", exact)', "key := digest.String()",
					"sharedCompilation.sourceMu.Lock()", "defer sharedCompilation.sourceMu.Unlock()",
					"if source, ok := sharedCompilation.sources[key]; ok", 'if sharedCompilation.root == ""',
					"gitrepo.Init", 'gitFixture.CommitFiles(context.Background(), files, "", "C4 cached source "+key)',
					"source := sourceRepositoryFixture{root: gitFixture.Root, ref: ref}",
					"sharedCompilation.sources[key] = source", "return source",
				]) && !/\blabel\b/u.test(sharedSourceRepositoryBody) && ordered(newTargetBody, [
					"source := sharedSourceRepository(t, files)", "gitobj.OpenRepository", "ScratchRoot:   t.TempDir()",
					"t.Cleanup(func() { _ = repository.Close() })", "contractexec.PublishOfficialTarget",
				]) && ordered(fixtureRunMainBody, [
					"code := m.Run()", 'if sharedCompilation.root != ""', "os.RemoveAll(sharedCompilation.root)", "return code",
				]),
			preOwnerEvidenceCapacity: /evidenceCapacity\s+cliEvidenceCapacity/u.test(runnerCLI) && [
				/privateEvidenceSummaryMaxBytes\s*=\s*int64\(1\s*<<\s*20\)/u,
				/privateEvidenceCaptureOverheadMaxBytes\s*=\s*int64\(1\s*<<\s*20\)/u,
				/privateEvidenceProjectionFrameMaxBytes\s*=\s*int64\(3\s*<<\s*20\)/u,
				/privateEvidenceCanonicalBodyCount\s*=\s*int64\(12\)/u,
				/privateEvidenceStoreAggregateMaxBytes\s*=\s*int64\(64\s*<<\s*20\)/u,
				/privateEvidenceMaximumUniqueBlobCount\s*=\s*14\b/u,
				/privateEvidenceMaximumChannelBytes\s*=\s*int64\(16\s*<<\s*20\)/u,
			].every((pattern) => pattern.test(runnerEvidence)) && ordered(evidenceCapacityBody, [
				"target.Valid()", "pre.valid()", "stdoutBytes > privateEvidenceMaximumChannelBytes",
				"privateEvidenceCaptureOverheadMaxBytes", "stdoutBytes", "stderrBytes",
				"privateEvidenceProjectionFrameMaxBytes",
				"privateEvidenceCanonicalBodyCount * privateEvidenceSummaryMaxBytes",
				"math.MaxInt64-next", "maximum > privateEvidenceStoreAggregateMaxBytes",
			]) && ordered(functionBody(runnerCLI, "prepareCLIExecution"), [
				"snapshotCandidate", "preflightPrivateEvidenceCapacity", "processmechanics.NewInvocation",
				"processmechanics.Prepare",
			]) && ordered(executeBody, [
				"prepareCLIExecution", "hostepoch.Measure", "store.AcquireContractRunOwner",
			]) && [
				"capacity.maximumUniqueBytes != 48<<20", "privateEvidenceMaximumChannelBytes + 1", "math.MaxInt64",
			].every((anchor) => capacityTestBody.includes(anchor)),
			actualDraftCapacityGate: ordered(persistBody, [
				"input.evidenceCapacity.validate(draft.bodies)",
				"owner.PersistPrivateRunManifest(ctx, draft.bodies)", "draft.persistAndAssemble",
			]) && count(executeBody, /persistRunAndClassification\s*\(/gu) === 1 &&
				count(closeStartErrorBody, /persistRunAndClassification\s*\(/gu) === 1 &&
				["hasDrain != hasCaptured", "bytes.Equal(drain, captured)", "sha256.Sum256(body)",
					"aggregate > capacity.maximumUniqueBytes-count", "len(unique) > privateEvidenceMaximumUniqueBlobCount",
				].every((anchor) => evidenceCapacityValidateBody.includes(anchor)),
			executionChronology: ordered(executeBody, [
				"contractexec.ReopenOfficialTarget", "prepareCLIExecution", "contractexec.ReopenOfficialTarget",
				"hostepoch.Measure", "store.AcquireContractRunOwner", "terminalClosureContext",
				"contractexec.ReopenOfficialTarget", "consumeAndStart",
				"owner.PersistSpawnObservation", "running.Close()", "input.scope.finish(closureContext)",
				"inspectAndRetireCLIInvocationEvidence", "snapshotCandidate", "contractexec.ReopenOfficialTarget",
				"buildChildEvidenceDraft", "persistRunAndClassification",
			]),
			startErrorChronology: ordered(closeStartErrorBody, [
				"owner.PersistSpawnObservation", "input.scope.finish(ctx)", "inspectAndRetireCLIInvocationEvidence",
				"snapshotCandidate", "contractexec.ReopenOfficialTarget", "buildStartErrorEvidenceDraft",
				"persistRunAndClassification",
			]),
			candidateEnvironment: ordered(functionBody(runnerCLI, "prepareCLIExecution"), [
				"evidenceRoot := roots.EvidenceRoot()", "preflightCLIInvocationEvidence(evidenceRoot, roots.MarkerPath())",
				"newCLIRuntimeAttemptID(model.Input().Attempt.ArtifactDigest)", "newScopeProbe", "buildCLIEnvironment",
			]) && ordered(functionBody(runnerCLI, "newCLIRuntimeAttemptID"), [
				"rand.Read(random[:])", "hex.EncodeToString(random[:])", "candidate != forbiddenID",
			]) && count(functionBody(runnerCLI, "newCLIRuntimeAttemptID"), /rand\.Read\s*\(/gu) === 1 &&
				functionBody(runnerCLI, "buildCLIEnvironment").includes("evidenceRoot != roots.EvidenceRoot()") &&
				functionBody(runnerCLI, "buildCLIEnvironment").includes("{evidenceRootEnvironment, evidenceRoot}") &&
				functionBody(runnerCLI, "buildCLIEnvironment").includes("{attemptIDEnvironment, attemptID}") &&
				!functionBody(runnerCLI, "buildCLIEnvironment").includes("filepath.Base(roots.AttemptRoot())") &&
				["sha256.Size*2", 'strings.HasPrefix(value, "attempt:")', "hex.DecodeString(hexText)",
					"len(decoded) == sha256.Size", "strings.ToLower(hexText) == hexText",
				].every((anchor) => functionBody(runnerEvidence, "validCLIRuntimeAttemptID").includes(anchor)) &&
				!goCodeOnly(fixtureMaterializeBody).includes("os.ReadFile(destination)") &&
				ordered(goCodeOnly(fixtureMaterializeBody), [
					"os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, permissions)",
					"validateExistingCLIFixture(destination, permissions, fixture.Contents())",
					"file.Chmod(permissions)", "writeExactCLIFixture(file, body)", "file.Sync()", "file.Close()",
					"validateExistingCLIFixture(destination, permissions, body)",
				]) && ordered(goCodeOnly(fixtureWriteBody), [
					"file.Write(body[offset:])", "offset += count", "if err != nil", "if count == 0", "io.ErrNoProgress",
				]) && ordered(goCodeOnly(fixtureValidateBody), [
					"os.Lstat(path)", "before.Mode().IsRegular()", "before.Size() != int64(len(expected))",
					"syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)",
					"handle.Stat()", "os.SameFile(before, opened)",
					"io.ReadAll(io.LimitReader(handle, int64(len(expected))+1))",
					"afterDescriptor, afterErr := handle.Stat()", "afterPath, pathErr := os.Lstat(path)",
					"os.SameFile(opened, afterDescriptor)", "os.SameFile(opened, afterPath)",
				]),
			detachedBoundedClosure: ordered(functionBody(runnerDarwin, "terminalClosureContext"), [
				"context.WithoutCancel(parent)", "context.WithTimeout",
			]) || functionBody(runnerDarwin, "terminalClosureContext").includes("context.WithTimeout(context.WithoutCancel(parent), budget)"),
			persistenceChronology: ordered(persistBody, [
				"owner.PersistPrivateRunManifest", "draft.persistAndAssemble", "contractmodel.NewFinalizedContractRun",
				"owner.PersistFinalizedRun", "closure.Release", "closure.FinalizedRun", "contractmodel.DeriveContractExecution",
				"store.PersistContractExecutionRecord",
			]),
			classificationOnlyRecovery: ordered(resumeBody, [
				"contractexec.ReopenOfficialTarget", "store.OpenFinalizedRunRecord", "store.OpenTerminalClosure",
				"closure.Release", "store.OpenFinalizedRunRecord", "contractmodel.DeriveContractExecution",
				"store.PersistContractExecutionRecord",
			]) && !/(?:processmechanics|prepareCLIExecution|AcquireContractRunOwner|consumeAndStart|\.Start\s*\()/u.test(goCodeOnly(resumeBody)),
		},
		terminalGraph: {
			storeSurface: exportedSurface(storeBridge),
			ownerPhases: ordered(storeBridge, [
				"contractRunOwnerAcquired", "contractRunOwnerStartConsumed", "contractRunOwnerSpawnObserved",
				"contractRunOwnerManifestPersisted", "contractRunOwnerFinalized",
			]),
			spawnBeforeManifest: methodBody(storeBridge, "ContractRunOwner", "PersistPrivateRunManifest")
				.includes("state.phase < contractRunOwnerSpawnObserved"),
			spawnBeforeFinalized: methodBody(storeBridge, "ContractRunOwner", "PersistFinalizedRun")
				.includes("durable spawn observation changed before finalized publication"),
			releaseOrder: ordered(releaseBody, [
				"replaceExecutionInterlock", "persistFinalizedRunReleaseLocked", "persistClearReceiptLocked",
			]) && count(releaseBody, /persistFinalizedRunReleaseLocked\s*\(/gu) === 2 &&
				count(releaseBody, /persistClearReceiptLocked\s*\(/gu) === 2,
			classificationGate: functionBody(storeBridge, "OpenFinalizedRunRecord").includes("validateFinalizedRunRelease") &&
				ordered(functionBody(storeBridge, "PersistContractExecutionRecord"), [
					"run.classificationReady()", "openStartClaim", "validateFinalizedRunRelease", "persistExecutionRecord",
				]),
		},
			evidence: {
			exactScopeOrder: ordered(scopeBody, [
				"contractmodel.ScopeTargetInventory", "contractmodel.ScopeChildBindings",
				"contractmodel.ScopeImportResolution", "contractmodel.ScopeServiceBindings",
				"contractmodel.ScopeSentinelInheritance",
			]) && scopeBody.includes("make([]scopeDraft, 0, 5)"),
			exactScopeCount: assembleBody.includes("len(draft.scope) != 5") &&
				assembleBody.includes("contractmodel.NewStandaloneScope(checks)"),
			manifestDerivedRefs: assembleBody.includes("manifest.EvidenceRef") &&
				!/(?:NewEvidenceRef|ParseDigest)/u.test(goCodeOnly(assembleBody)),
			boundedCandidateInventory: ordered(candidateInventoryEntryBody, [
				"snapshotCandidateWithLimits(root, inventoryLimits{",
				"entries: maxCandidateInventoryEntries", "pathBytes: maxCandidateInventoryPathBytes",
				"depth: maxCandidateInventoryDepth", "fileBytes: maxCandidateInventoryFileBytes",
				"aggregateBytes: maxCandidateInventoryAggregateBytes", "readBatch: candidateInventoryReadBatch",
			]) && [
				/\bcandidateInventoryReadBatch\s*=\s*128\b/u,
				/\bmaxCandidateInventoryEntries\s*=\s*20_000\b/u,
				/\bmaxCandidateInventoryPathBytes\s*=\s*32\s*\*\s*1024\s*\*\s*1024\b/u,
				/\bmaxCandidateInventoryDepth\s*=\s*128\b/u,
				/\bmaxCandidateInventoryFileBytes\s*=\s*64\s*\*\s*1024\s*\*\s*1024\b/u,
				/\bmaxCandidateInventoryAggregateBytes\s*=\s*64\s*\*\s*1024\s*\*\s*1024\b/u,
				/\bdarwinOpenNoFollowAny\s*=\s*0x20000000\b/u,
			].every((pattern) => pattern.test(candidateInventorySource)) &&
			!/\bvar\s+candidateInventoryLimits\b/u.test(candidateInventorySource) &&
			ordered(candidateInventoryBody, [
				"candidateInventoryState", "readCandidateInventoryDirectory", "os.Lstat(root)", "os.SameFile(rootInfo, afterRoot)",
			]) && ordered(candidateDirectoryBody, [
				"syscall.O_DIRECTORY", "darwinOpenNoFollowAny", "handle.ReadDir(limits.readBatch)",
				"limits.depth", "limits.entries", "limits.pathBytes", "limits.fileBytes", "limits.aggregateBytes",
				"digestCandidateInventoryFile", "handle.Stat()", "os.Lstat(directory.path)",
			]) && ordered(candidateFileBody, [
				"darwinOpenNoFollowAny", "syscall.O_NONBLOCK", "handle.Stat()",
				"io.Copy(digest, io.LimitReader(handle, opened.Size()))", "handle.Stat()", "os.Lstat(path)",
				"os.SameFile(opened, afterDescriptor)", "os.SameFile(opened, afterPath)",
			]) && !/(?:filepath\.WalkDir|os\.ReadDir|os\.ReadFile)/u.test(
				candidateInventoryEntryBody + candidateInventoryBody + candidateDirectoryBody + candidateFileBody,
			),
			rawFramedSingleCopy: !/(?:encoding\/base64|base64\.)/u.test(runnerEvidence) &&
				ordered(evidenceFrameBody, [
					"digest := sha256.Sum256(segment.body)", "canon.CanonicalizeTyped", "framingBytes :=",
					"binary.BigEndian.AppendUint64(body, uint64(len(metadata)))", "body = append(body, metadata...)",
					"binary.BigEndian.AppendUint64(body, uint64(len(segment.body)))", "body = append(body, segment.body...)",
				]) && count(childEvidenceBody, /framePrivateEvidence\s*\(/gu) === 2 && ordered(childEvidenceBody, [
					"captureFrame, err := framePrivateEvidence", "privateEvidenceSegment{name: \"stdout\"",
					"privateEvidenceSegment{name: \"stderr\"", "EvidenceDrainResult] = captureFrame",
					"EvidenceCapturedObservation] = captureFrame", "projectionFrame, frameErr := framePrivateEvidence",
				]) && canonicalEvidenceBody.includes("int64(len(body)) > privateEvidenceSummaryMaxBytes") &&
				["[]byte{0x00, 0xff, 'o'}", "binary.BigEndian.Uint64", "len(fullRoster) != 15",
					"maximumUniqueBytes: aggregate - 1",
				].every((anchor) => capacityTestBody.includes(anchor)) &&
				["reference.Kind() == contractmodel.EvidenceDrainResult", "witness.Observation().CapturedRef()",
					"drainRef.Digest() != capturedRef.Digest()",
				].every((anchor) => cliPhysicalTests.includes(anchor)),
			boundedStoreRosters: ordered(storeBoundedRosterBody, [
				"filepath.IsAbs(path)", "os.Lstat(path)", "os.Open(path)", "handle.Stat()",
				"handle.ReadDir(remaining)", "len(entries) > maximum", "len(batch) == 0",
				"afterDescriptor", "os.Lstat(path)", "os.SameFile(opened, afterPath)",
			]) && count(inspectAttemptRootsBody, /boundedExactDirectoryNames\s*\(/gu) === 2 &&
				ordered(inspectAttemptRootsBody, [
					"roots.attempt", "len(conformanceAttemptRootNames)", "roots.evidence", "contractAttemptMarker",
				]) && !/(?:os\.ReadDir|filepath\.WalkDir)/u.test(storeBoundedRosterBody + inspectAttemptRootsBody) &&
				["attempt-root-extra", "evidence-root-extra", 'filepath.Join(foreignParent, "foreign", "deep", "sentinel")',
					"foreign direct roster entry retained live attempt authority",
					"bounded roster validation traversed or removed foreign residue",
				].every((anchor) => boundedStoreTestBody.includes(anchor)),
			invocationClosure: [
				"cli-invocation.json", "64 << 10", "CANDIDATE_WRITTEN_EVIDENCE_IS_NOT_MALICIOUS_PROCESS_ATTESTATION",
				"ATTEMPT_ID_MISMATCH", "LOGICAL_ARGV_MISMATCH",
			].every((anchor) => runnerEvidence.includes(anchor)) &&
				exact(structFields(runnerEvidence, "invocationEvidenceAuthority"), ["root", "marker"]) &&
				ordered(functionBody(runnerEvidence, "preflightCLIInvocationEvidence"), [
					"filepath.IsAbs(root)", "os.Lstat(root)", "rootInfo.Mode().Perm() != 0o700",
					"boundedDirectRoster(context.Background()", "os.Lstat(markerPath)",
					"os.Lstat(filepath.Join(root, cliInvocationFilename))",
				]) && ordered(invocationInspectBody, [
					"os.Lstat(path)", "boundedDirectRoster(ctx, root, authority.root, expectedRoster)",
					"os.Lstat(markerPath)", "readCLIInvocationEvidence", "retireCLIInvocationEvidence",
				]) && ordered(invocationReadBody, ["syscall.Open", "syscall.O_NOFOLLOW", "io.LimitReader", "canon.Parse", "CanonicalChecked"]) &&
				["actualAttempt == attemptID", "text != logicalArgv[index]"].every((anchor) => invocationReadBody.includes(anchor)) &&
				ordered(invocationRetireBody, [
					"removeRetainedLeaf(ctx, path, observed)", "boundedDirectRoster(ctx, root, authority.root",
					"os.Lstat(markerPath)", "os.Lstat(path)",
				]) && !/(?:os\.ReadDir|os\.RemoveAll|filepath\.WalkDir)/u.test(
					functionBody(runnerEvidence, "preflightCLIInvocationEvidence") + invocationInspectBody + invocationRetireBody,
				) && scopeBody.includes("input.invocationEvidence.validated()") &&
				scopeBody.includes("input.invocationEvidence.contradictsBinding()") &&
				scopeBody.includes("enrolled && unchanged, false, contractmodel.ViolationTargetSourcePresent") &&
				methodBody(runnerEvidence, "invocationEvidence", "facts").includes("cliInvocationNonclaim") &&
				["invocationEvidenceValidated", 'evidence.Presence == "PRESENT"', "sha256.Size*2",
					"evidence.AttemptIDValidated", "evidence.LogicalArgvValidated",
				].every((anchor) => methodBody(runnerEvidence, "invocationEvidence", "validated").includes(anchor)) &&
				ordered(closeStartErrorBody, [
					"input.scope.finish(ctx)", "inspectAndRetireCLIInvocationEvidence", "snapshotCandidate",
					"contractexec.ReopenOfficialTarget", "buildStartErrorEvidenceDraft",
				]) && functionBody(runnerEvidence, "buildStartErrorEvidenceDraft").includes('"cli_invocation": input.invocationEvidence.facts()'),
		},
		scopeProbe: {
			attemptPrivateRoot: ordered(scopeProbeBody, [
				"roots.TemporaryRoot()", "os.Lstat(parent)", "parentInfo.Mode().Perm()&0o077", "os.MkdirTemp(parent, \"scope-\")",
				"os.Chmod(root, 0o700)", "newShortSocketRoot(root)",
				"newUnixCanary(filepath.Join(socketAlias, \"i\"))", "newUnixCanary(filepath.Join(socketAlias, \"s\"))",
			]) && scopeProbeBody.includes("probe.importModule = filepath.Join(root, \"outside-candidate.mjs\")"),
			shortPrivateAlias: runnerScope.includes('shortSocketParent            = "/private/tmp"') &&
				runnerScope.includes("maxDarwinUnixSocketPathBytes = 103") &&
				ordered(shortSocketBody, [
					"os.Lstat(shortSocketParent)", "parentInfo.Mode()&os.ModeSymlink", "parentInfo.Mode().Perm() != 0o777",
					"parentInfo.Mode()&os.ModeSticky", "parentStat.Uid != 0",
					"os.MkdirTemp(shortSocketParent, \"cs4-\")", "os.Chmod(root, 0o700)",
					"stat.Uid != uint32(os.Getuid())", "os.Symlink(attemptPrivateRoot, alias)", "os.Readlink(alias)",
					"filepath.EvalSymlinks(alias)",
					"maxDarwinUnixSocketPathBytes",
				]) && ordered(goCodeOnly(canaryCreateBody), [
					"net.ListenUnix", "listener.SetUnlinkOnClose(false)", "os.Lstat(path)",
					"identity.Mode()&os.ModeSocket", "identity: identity", "done: make(chan error, 1)",
					"AcceptUnix()", "canary.done <- err",
				]) &&
				ordered(goCodeOnly(canaryCloseBody), [
					"os.Lstat(canary.path)", "os.SameFile(canary.identity, currentIdentity)", "SetDeadline",
					"terminalErr := <-canary.done",
					"classifyCanaryAcceptError(terminalErr, true, false)", "canary.listener.Close()",
					"terminalErr := <-canary.done", "classifyCanaryAcceptError(terminalErr, false, true)",
					"errors.Join(integrityErr, deadlineErr, closeErr, acceptErr)",
				]) && !canaryCloseBody.includes("errors.Is(closeErr, net.ErrClosed)") &&
				["networkError.Timeout()", "errors.Is(err, net.ErrClosed)"].every((anchor) =>
					canaryClassifyBody.includes(anchor)) &&
				!/(?:closing\.Load|closing\.Store|canary\.acceptErr\s*=)/u.test(goCodeOnly(canaryCreateBody + canaryCloseBody)),
			boundedDescriptorRoster: ordered(boundedRosterBody, [
				"os.Lstat(root)", "syscall.Open", "syscall.O_DIRECTORY", "darwinOpenNoFollowAny", "syscall.O_NONBLOCK",
				"handle.Stat()", "os.SameFile(retained, opened)", "handle.ReadDir(remaining)",
				"if _, present := allowed[name]; !present", "len(observed) > len(want)", "len(batch) == 0",
				"handle.Sync()", "handle.Stat()", "handle.Close()", "os.Lstat(root)",
				"os.SameFile(opened, afterPath)",
			]) && !/(?:os\.ReadDir|os\.RemoveAll|filepath\.WalkDir)/u.test(boundedRosterBody),
			identityBoundTerminalCleanup: [
				"rootIdentity", "socketRootIdentity", "socketAliasIdentity", "importModuleIdentity",
			].every((field) => structFields(runnerScope, "scopeProbe").includes(field)) && ordered(scopeFinishBody, [
				"probe.root", "probe.rootIdentity", "probe.socketRoot", "probe.socketRootIdentity",
				"digestCandidateInventoryFile", "os.Readlink(probe.socketAlias)", "os.Lstat(probe.socketAlias)",
				"probe.importCanary.close()", "probe.serviceCanary.close()",
				"removeRetainedLeaf(ctx, probe.importCanary.path", "removeRetainedLeaf(ctx, probe.serviceCanary.path",
				"removeRetainedLeaf(ctx, probe.importModule", "removeRetainedEmptyDirectory(ctx, probe.root",
				"removeRetainedLeaf(ctx, probe.socketAlias", "removeRetainedEmptyDirectory(ctx, probe.socketRoot",
			]) && ordered(removeRetainedLeafBody, [
				"os.Lstat(path)", "os.SameFile(retained, current)", "os.Remove(path)", "os.Lstat(path)",
			]) && ordered(removeRetainedDirectoryBody, [
				"boundedDirectRoster(ctx, path, retained, nil)", "os.Remove(path)", "os.Lstat(path)",
			]) && !runnerScope.includes("os.RemoveAll"),
			residueHostiles: [
				'filepath.Join(evidenceInput.evidenceRoot, "foreign"',
				"foreign evidence residue did not fail closed", "foreign evidence residue was traversed or removed",
				'filepath.Join(probe.root, "foreign"', "foreign scope residue became a clean observation",
				"foreign scope residue was traversed or removed", "same-size scope module rewrite became clean",
			].every((anchor) => boundedResidueTestBody.includes(anchor)),
		},
		profiles: Object.fromEntries(c4ProfileNames.map((profile) => [profile, goJSONArguments(profile)])),
		claimMap,
	});
}

function violation(code, detail) { return Object.freeze({ code, detail }); }

export function validateC2Facts(facts) {
	const problems = [];
	const add = (code, detail) => problems.push(violation(code, detail));
	if (facts.package?.importPath !== storePackagePath || facts.package?.name !== "store" ||
		facts.package?.modulePath !== modulePath || facts.package?.moduleMain !== true) {
		add("P07B_C2_PACKAGE_IDENTITY", JSON.stringify(facts.package));
	}
	if (!exact(facts.package?.productionFiles, expectedC2ProductionFiles) ||
		!exact(facts.package?.testFiles, expectedC2TestFiles) || !exact(facts.package?.xTestFiles, expectedC2XTestFiles) ||
		!exact(facts.package?.ignoredGoFiles, ["head_unsupported.go"]) ||
		(facts.package?.invalidGoFiles ?? []).length !== 0 || (facts.package?.nonGoBuildFiles ?? []).length !== 0) {
		add("P07B_C2_STORE_TOPOLOGY", JSON.stringify(facts.package));
	}
	if (!exact(facts.package?.productionImports, expectedC2PackageImports)) {
		add("P07B_C2_IMPORT_ROSTER", `package:${JSON.stringify(facts.package?.productionImports)}`);
	}
	const expectedEntries = sorted([
		...expectedC2ProductionFiles, ...expectedC2TestFiles, ...expectedC2XTestFiles, "head_unsupported.go",
	].map((name) => `${name}:file`));
	if (!exact(facts.directoryEntries, expectedEntries)) add("P07B_C2_STORE_TOPOLOGY", JSON.stringify(facts.directoryEntries));
	for (const [file, expected] of Object.entries(expectedC2Imports)) {
		if (!exact(facts.imports?.[file], expected)) add("P07B_C2_IMPORT_ROSTER", `${file}:${JSON.stringify(facts.imports?.[file])}`);
	}
	const expectedNewProductionExports = {
		"internal/store/execution_interlock.go": [],
		"internal/store/nonhead_contract.go": expectedC3StoreSurface,
		"internal/store/private_contract_run.go": [],
	};
	if (!exact(facts.newProductionExports, expectedNewProductionExports) ||
		!exact(facts.objectStoreExports, expectedC2ObjectStoreSurface) || facts.compilerParsedSurface !== true) {
		add("P07B_C2_EXPORTED_SURFACE", JSON.stringify({
			new: facts.newProductionExports, store: facts.objectStoreExports, compilerParsed: facts.compilerParsedSurface,
		}));
	}
	for (const [name, expected] of Object.entries(expectedC2StructFields)) {
		if (!exact(facts.structs?.[name], expected)) add("P07B_C2_AUTHORITY_SHAPE", `${name}:${JSON.stringify(facts.structs?.[name])}`);
	}
	if (!exact(facts.testSymbols, expectedC2TestSymbols)) add("P07B_C2_TEST_SYMBOL_ROSTER", JSON.stringify(facts.testSymbols));
	for (const [path, expected] of Object.entries(expectedC2TestFilesByProfile)) {
		const combined = [...expected, ...(expectedC3StoreFilesByProfile[path] ?? [])];
		if (!exact(facts.testFiles?.[path], sorted(combined))) add("P07B_C2_TEST_FILE_ROSTER", `${path}:${JSON.stringify(facts.testFiles?.[path])}`);
	}
	if (!exact(facts.modelImporters, [
		"internal/store/contract_run_bridge.go", "internal/store/nonhead_contract.go",
	])) {
		add("P07B_C2_MODEL_IMPORTER_ROSTER", JSON.stringify(facts.modelImporters));
	}
	if ((facts.forbiddenSurface ?? []).length !== 0) add("P07B_C2_FORBIDDEN_PRODUCTION_SURFACE", facts.forbiddenSurface.join(","));
	if (!facts.namespaces?.paths || !facts.namespaces?.retained || !facts.namespaces?.replacementTest ||
		!facts.namespaces?.caseAliasGuard ||
		!exact(facts.namespaces?.fields, expectedC2ObjectStoreFields)) {
		add("P07B_C2_NAMESPACE_IDENTITY", JSON.stringify(facts.namespaces));
	}
	if (!facts.relations?.constants || !facts.relations?.keyRoster || !facts.relations?.algebraClosed ||
		!facts.relations?.persistenceOrder || !facts.relations?.openConvergence ||
		!facts.relations?.runManifestGate || !facts.relations?.executionProfileDerived ||
		!facts.relations?.identityGuards || !facts.relations?.identityReplacementTests ||
		!exact(facts.relations?.effects, ["AMBIGUOUS", "EXACT_CONVERGED", "KNOWN_NO_EFFECT"])) {
		add("P07B_C2_RELATION_ALGEBRA", JSON.stringify(facts.relations));
	}
	if (!facts.interlock?.bootJoin || !facts.interlock?.acquisitionOrder || !facts.interlock?.releaseOrder || !facts.interlock?.resetOrder ||
		!facts.interlock?.winnerFreshAndConsumed || !facts.interlock?.startClaimConvergence || !facts.interlock?.clearReceiptTransition ||
		!facts.interlock?.clearReceiptConvergence || !facts.interlock?.clearReceiptFaults || !facts.interlock?.identityBoundary) {
		add("P07B_C2_INTERLOCK_PROTOCOL", JSON.stringify(facts.interlock));
	}
	if (!exact(facts.interlock?.productionSeals, { attempt: 2, terminal: 0, reset: 0, lease: 1, winner: 1, manifest: 2 }) ||
		!exact(facts.interlock?.testSeals, { attempt: 1, terminal: 1, reset: 1 })) {
		add("P07B_C2_SEAL_OWNERSHIP", JSON.stringify({ production: facts.interlock?.productionSeals, test: facts.interlock?.testSeals }));
	}
	if (!facts.privateEvidence?.limits || !exact(facts.privateEvidence?.kinds, [
		"MATERIALIZATION_REVALIDATION", "RUNTIME_REVALIDATION", "PROCESS_RESULT", "WAIT_RESULT", "DRAIN_RESULT",
		"TEARDOWN_RESULT", "ORPHAN_CHECK", "FINALIZATION_MARKER", "CAPTURED_OBSERVATION", "PROJECTION_RESULT",
		"TARGET_INVENTORY", "CHILD_BINDINGS", "IMPORT_RESOLUTION", "SERVICE_BINDINGS",
		"NAMED_PARENT_SECRET_SENTINEL_INHERITANCE",
	]) || !exact(facts.privateEvidence?.states, ["MISSING_UNEXPECTED", "PURGED", "RETAINED"]) ||
		!facts.privateEvidence?.manifestClosure || !facts.privateEvidence?.manifestOpenConvergence || !facts.privateEvidence?.arithmeticBounded ||
		!facts.privateEvidence?.availabilityJoins || !facts.privateEvidence?.purgeOrder ||
		!facts.privateEvidence?.purgeCreateCount || !facts.privateEvidence?.freshPurgeOrder ||
		!facts.privateEvidence?.missingPackGate || !facts.privateEvidence?.noCanonicalMutation ||
		!facts.privateEvidence?.identityGuards || !facts.privateEvidence?.identityReplacementTests) {
		add("P07B_C2_PRIVATE_EVIDENCE", JSON.stringify(facts.privateEvidence));
	}
	return problems;
}

export function validateC3Facts(facts) {
	const problems = [];
	const add = (code, detail) => problems.push(violation(code, detail));
	for (const [importPath, expected] of Object.entries(expectedC3Packages)) {
		const actual = facts.packages?.[importPath];
		if (actual?.name !== expected.name || actual?.modulePath !== modulePath || actual?.moduleMain !== true ||
			!exact(actual?.go, expected.go) || !exact(actual?.cgo, expected.cgo) || !exact(actual?.test, expected.test) ||
			!exact(actual?.xtest, expected.xtest) || !exact(actual?.ignored, expected.ignored) ||
			(actual?.invalid ?? []).length !== 0 || !exact(actual?.imports, expected.imports)) {
			add("P07B_C3_PACKAGE_TOPOLOGY", `${importPath}:${JSON.stringify(actual)}`);
		}
	}
	if (!exact(facts.buildTags, expectedC3BuildTags)) {
		add("P07B_C3_BUILD_TAG_ROSTER", JSON.stringify(facts.buildTags));
	}
	for (const [path, expected] of Object.entries(expectedC3SourceImports)) {
		if (!exact(facts.sourceImports?.[path], expected)) {
			add("P07B_C3_IMPORT_ROSTER", `${path}:${JSON.stringify(facts.sourceImports?.[path])}`);
		}
		const expectedSurface = expectedC3Surfaces[path] ?? [];
		if (!exact(facts.sourceSurfaces?.[path], expectedSurface)) {
			add("P07B_C3_EXPORTED_SURFACE", `${path}:${JSON.stringify(facts.sourceSurfaces?.[path])}`);
		}
	}
	const expectedTestFiles = {
		"internal/contractexec/target_test.go": c3OfficialTargetTests,
		"internal/gitobj/single_target_test.go": c3SingleTargetTests,
		"internal/hostepoch/epoch_darwin_test.go": ["TestC3HostEpochDarwinLiveMeasurement"],
		"internal/hostepoch/epoch_test.go": c3HostEpochTests.filter((name) =>
			!["TestC3HostEpochDarwinLiveMeasurement", "TestC3HostEpochPublicSurfaceIsClosed"].includes(name)),
		"internal/hostepoch/public_api_test.go": ["TestC3HostEpochPublicSurfaceIsClosed"],
		"internal/noderuntime/probe_darwin_test.go": c3NodeRuntimeTests.filter((name) => name.startsWith("TestC3NodeRuntimeProbe")),
		"internal/noderuntime/public_api_test.go": ["TestC3NodeRuntimePublicSurfaceAndSoleSpawnEdge"],
		"internal/noderuntime/runtime_darwin_test.go": c3NodeRuntimeTests.filter((name) => name.startsWith("TestC3NodeRuntimeDarwin")),
		"internal/noderuntime/runtime_test.go": c3NodeRuntimeTests.filter((name) =>
			!name.startsWith("TestC3NodeRuntimeProbe") && !name.startsWith("TestC3NodeRuntimeDarwin") &&
			name !== "TestC3NodeRuntimePublicSurfaceAndSoleSpawnEdge"),
		"internal/store/nonhead_contract_test.go": expectedC3StoreFilesByProfile["internal/store/nonhead_contract_test.go"],
		"internal/store/public_api_test.go": expectedC3StoreFilesByProfile["internal/store/public_api_test.go"],
	};
	for (const [path, expected] of Object.entries(expectedTestFiles)) {
		if (!exact(facts.testFiles?.[path], sorted(expected))) {
			add("P07B_C3_TEST_FILE_ROSTER", `${path}:${JSON.stringify(facts.testFiles?.[path])}`);
		}
	}
	if ((facts.c2Problems ?? []).length !== 0) add("P07B_C3_INHERITED_C2", JSON.stringify(facts.c2Problems));
	if (!exact(facts.storeBridge?.modelImporters, [
		"internal/store/contract_run_bridge.go", "internal/store/nonhead_contract.go",
	]) ||
		!exact(facts.storeBridge?.exports, expectedC3StoreSurface) || !facts.storeBridge?.rootRoster ||
		!facts.storeBridge?.markerContract || !facts.storeBridge?.freshNonce || !facts.storeBridge?.exactReopen ||
		!facts.storeBridge?.targetJoin || !facts.storeBridge?.noIssuerOrProcessEdge) {
		add("P07B_C3_STORE_BRIDGE", JSON.stringify(facts.storeBridge));
	}
	if (!facts.gitTarget?.directOpaqueSource || !facts.gitTarget?.publicationOrder || !facts.gitTarget?.reopenOrder ||
		!facts.gitTarget?.ambiguousRecovery || !facts.gitTarget?.dirtyWorktreeTest) {
		add("P07B_C3_GIT_TARGET", JSON.stringify(facts.gitTarget));
	}
	if (!facts.hostEpoch?.twoSamples || !facts.hostEpoch?.canonicalUUID || !facts.hostEpoch?.typedDigest ||
		!facts.hostEpoch?.fixedDarwinSource || !facts.hostEpoch?.noFallbackOrProcess) {
		add("P07B_C3_HOST_EPOCH", JSON.stringify(facts.hostEpoch));
	}
	if (!exact(facts.nodeRuntime?.spawnOwners, ["internal/noderuntime/probe_darwin.go"]) ||
		!facts.nodeRuntime?.noAmbientPath || !facts.nodeRuntime?.measureProbeMeasure || !facts.nodeRuntime?.fixedProbe ||
		!facts.nodeRuntime?.executableIdentity) {
		add("P07B_C3_NODE_RUNTIME", JSON.stringify(facts.nodeRuntime));
	}
	if (!facts.officialTarget?.publishOrder || !facts.officialTarget?.openOrder || !facts.officialTarget?.finalRejoin ||
		!facts.officialTarget?.faultCoverage ||
		!facts.officialTarget?.treeJoinClosed || !facts.officialTarget?.preSpawnOnly ||
		!facts.officialTarget?.headPreservationTest) {
		add("P07B_C3_OFFICIAL_TARGET", JSON.stringify(facts.officialTarget));
	}
	if (!exact(facts.predecessor, expectedC3Predecessor)) {
		add("P07B_C3_PREDECESSOR", JSON.stringify(facts.predecessor));
	}
	if (!exact(facts.predecessorAuthority?.source_parent, expectedC3PredecessorAuthority.source_parent) ||
		!exact(facts.predecessorAuthority?.ancestry, expectedC3PredecessorAuthority.ancestry)) {
		add("P07B_C3_PREDECESSOR_AUTHORITY", JSON.stringify(facts.predecessorAuthority));
	}
	if (!exact(facts.predecessorAuthority?.chain, expectedC3PredecessorAuthority.chain)) {
		add("P07B_C3_PREDECESSOR_CHAIN", JSON.stringify(facts.predecessorAuthority?.chain));
	}
	return problems;
}

export function validateC4Facts(facts) {
	const problems = [];
	const add = (code, detail) => problems.push(violation(code, detail));
	if ((facts.c3Problems ?? []).length !== 0) {
		add("P07B_C4_INHERITED_C3", JSON.stringify(facts.c3Problems));
	}
	for (const [importPath, expected] of Object.entries(expectedC4Packages)) {
		const actual = facts.packages?.[importPath];
		if (actual?.name !== expected.name || actual?.modulePath !== modulePath || actual?.moduleMain !== true ||
			!exact(actual?.go, expected.go) || !exact(actual?.cgo, expected.cgo) || !exact(actual?.test, expected.test) ||
			!exact(actual?.xtest, expected.xtest) || !exact(actual?.ignored, expected.ignored) ||
			(actual?.invalid ?? []).length !== 0 || !exact(actual?.imports, expected.imports)) {
			add("P07B_C4_PACKAGE_TOPOLOGY", `${importPath}:${JSON.stringify(actual)}`);
		}
	}
	if (!exact(facts.buildTags, expectedC4BuildTags)) {
		add("P07B_C4_BUILD_TAG_ROSTER", JSON.stringify(facts.buildTags));
	}
	for (const [path, expected] of Object.entries(expectedC4TestsByFile)) {
		if (!exact(facts.testFiles?.[path], sorted(expected))) {
			add("P07B_C4_TEST_FILE_ROSTER", `${path}:${JSON.stringify(facts.testFiles?.[path])}`);
		}
	}
	if (!exact(facts.mechanics?.surface, expectedC4MechanicsSurface) ||
		!exact(facts.mechanics?.invocationFields, ["executable", "argv", "environment", "stdin", "cwd", "limits", "binding"]) ||
		!exact(facts.mechanics?.preparedFields, ["state"]) ||
		!exact(facts.mechanics?.preparedStateFields, ["invocation", "started"]) ||
		!exact(facts.mechanics?.runningFields, ["state"]) ||
		!exact(facts.mechanics?.unsupportedRunningFields, ["state"]) ||
		!exact(facts.mechanics?.runningStateFields, [
			"prepared", "ctx", "command", "stdoutPipe", "stderrPipe", "stdinWriter", "stdinResultC", "waitC",
			"stdoutDone", "stderrDone", "overflowC", "captures", "executionTimer", "mu", "result", "closing",
			"closeOnce", "final",
		]) ||
		!exact(facts.mechanics?.stdinFields, ["presence", "bytes"]) ||
		!facts.mechanics?.defensiveInputCopy || !facts.mechanics?.oneShot || !facts.mechanics?.directSpawn ||
		!facts.mechanics?.copySafeState || !facts.mechanics?.resultCopies || !facts.mechanics?.noSemanticAuthority) {
		add("P07B_C4_MECHANICS_AUTHORITY", JSON.stringify(facts.mechanics));
	}
	if (!exact(facts.mechanics?.importers, [contractRunnerPackagePath, `${modulePath}/internal/world`])) {
		add("P07B_C4_MECHANICS_IMPORTERS", JSON.stringify(facts.mechanics?.importers));
	}
	if (!facts.worldAdapter?.chronology || !facts.worldAdapter?.noDuplicateSpawn ||
		!facts.worldAdapter?.httpMechanicsRetained) {
		add("P07B_C4_WORLD_ADAPTER", JSON.stringify(facts.worldAdapter));
	}
	if (!exact(facts.runner?.surface, expectedC4RunnerSurface) || !facts.runner?.closedEntrypoints) {
		add("P07B_C4_RUNNER_SURFACE", JSON.stringify(facts.runner));
	}
	if (!facts.runner?.permitAdjacent || !facts.runner?.spawnAdjacentRevalidation ||
		!exact(facts.runner?.soleOwnerAcquirer, [
			"internal/contractexec/runner/runner_darwin.go:1", "internal/store/contract_run_bridge.go:1",
		]) ||
		!facts.runner?.detachedBoundedClosure) {
		add("P07B_C4_ADMISSION_ADJACENCY", JSON.stringify(facts.runner));
	}
	if (!facts.runner?.executionChronology || !facts.runner?.startErrorChronology ||
		!facts.runner?.immutableSourceCacheIsolation || !facts.runner?.boundedPhysicalTestConcurrency ||
		!facts.runner?.preOwnerEvidenceCapacity || !facts.runner?.actualDraftCapacityGate ||
		!facts.runner?.candidateEnvironment || !facts.runner?.physicalConforms || !facts.runner?.persistenceChronology ||
		!facts.runner?.classificationOnlyRecovery) {
		add("P07B_C4_EXECUTION_CHRONOLOGY", JSON.stringify(facts.runner));
	}
	if (!exact(facts.terminalGraph?.storeSurface, expectedC4StoreBridgeSurface) ||
		!facts.terminalGraph?.ownerPhases || !facts.terminalGraph?.spawnBeforeManifest ||
		!facts.terminalGraph?.spawnBeforeFinalized || !facts.terminalGraph?.releaseOrder ||
		!facts.terminalGraph?.classificationGate) {
		add("P07B_C4_TERMINAL_GRAPH", JSON.stringify(facts.terminalGraph));
	}
	if (!facts.evidence?.exactScopeOrder || !facts.evidence?.exactScopeCount || !facts.evidence?.manifestDerivedRefs ||
		!facts.evidence?.boundedCandidateInventory || !facts.evidence?.rawFramedSingleCopy ||
		!facts.evidence?.boundedStoreRosters ||
		!facts.evidence?.invocationClosure) {
		add("P07B_C4_EVIDENCE_SCOPE", JSON.stringify(facts.evidence));
	}
	if (!facts.scopeProbe?.attemptPrivateRoot || !facts.scopeProbe?.shortPrivateAlias ||
		!facts.scopeProbe?.boundedDescriptorRoster || !facts.scopeProbe?.identityBoundTerminalCleanup ||
		!facts.scopeProbe?.residueHostiles) {
		add("P07B_C4_SCOPE_ROOT", JSON.stringify(facts.scopeProbe));
	}
	if (!exact(Object.keys(facts.profiles ?? {}).sort(), [...c4ProfileNames].sort())) {
		add("P07B_C4_PROFILE_COMMAND", JSON.stringify(Object.keys(facts.profiles ?? {})));
	} else {
		for (const profileName of c4ProfileNames) {
			const args = facts.profiles[profileName];
			const expectedTests = expectedC4ProfileTests[profileName];
			const expectedPackage = goJSONProfiles[profileName].packageArgument;
			const expectedPattern = `^(?:${expectedTests.join("|")})$`;
			const raceCount = Array.isArray(args) ? args.filter((value) => value === "-race").length : -1;
			const expectedRace = profileName === "c4-processmechanics-parity" || profileName === "c4-authority-race";
			if (!Array.isArray(args) || args[0] !== "test" || args.at(-1) !== expectedPackage ||
				args.at(-2) !== expectedPattern || !args.includes("-json") || !args.includes("-count=1") ||
				raceCount !== (expectedRace ? 1 : 0)) {
				add("P07B_C4_PROFILE_COMMAND", `${profileName}:${JSON.stringify(args)}`);
			}
		}
	}
	if (!Array.isArray(facts.claimMap?.status) || !Array.isArray(facts.claimMap?.runbook) ||
		facts.claimMap.status.length !== 80 || facts.claimMap.runbook.length !== 80 ||
		!exact(facts.claimMap.status, facts.claimMap.runbook)) {
		add("P07B_C4_PROFILE_COMMAND", JSON.stringify(facts.claimMap));
	}
	return problems;
}

export function validateFacts(facts) {
	const problems = [];
	const add = (code, detail) => problems.push(violation(code, detail));
	if (
		facts.package?.importPath !== packagePath
		|| facts.package?.name !== "model"
		|| facts.package?.modulePath !== modulePath
		|| facts.package?.moduleMain !== true
	) add("P07B_C1_PACKAGE_IDENTITY", JSON.stringify(facts.package));
	if (!exact(facts.package?.productionFiles, expectedProductionFiles)) {
		add("P07B_C1_PRODUCTION_TOPOLOGY", JSON.stringify(facts.package?.productionFiles));
	}
	if (!exact(facts.package?.testFiles, expectedTestFiles) || !exact(facts.package?.xTestFiles, [])) {
		add("P07B_C1_TEST_TOPOLOGY", JSON.stringify({ test: facts.package?.testFiles, xTest: facts.package?.xTestFiles }));
	}
	if (!exact(facts.topology?.contractexecEntries, [
		"model:directory", "runner:directory", "target.go:file", "target_test.go:file",
	])) {
		add("P07B_C1_PRODUCTION_TOPOLOGY", JSON.stringify(facts.topology?.contractexecEntries));
	}
	if (!exact(facts.topology?.modelEntries, expectedModelEntries)) {
		add("P07B_C1_MODEL_TOPOLOGY", JSON.stringify(facts.topology?.modelEntries));
	}
	if (!exact(facts.package?.productionImports, expectedProductionImports)) {
		add("P07B_C1_PRODUCTION_IMPORT_ROSTER", JSON.stringify(facts.package?.productionImports));
	}
	if (!exact(facts.package?.testImports, expectedTestImports) || !exact(facts.package?.xTestImports, [])) {
		add("P07B_C1_TEST_IMPORT_ROSTER", JSON.stringify({ test: facts.package?.testImports, xTest: facts.package?.xTestImports }));
	}
	if (!exact(facts.package?.testSymbols, expectedTestSymbols)) {
		add("P07B_C1_TEST_SYMBOL_ROSTER", JSON.stringify(facts.package?.testSymbols));
	}
	if (
		(facts.package?.ignoredGoFiles ?? []).length > 0
		|| (facts.package?.invalidGoFiles ?? []).length > 0
		|| (facts.package?.nonGoBuildFiles ?? []).length > 0
	) add("P07B_C1_PRODUCTION_TOPOLOGY", "ignored, invalid, or foreign build input");
	if (!exact(facts.localDependencies, expectedLocalDependencies) || (facts.externalDependencies ?? []).length > 0) {
		add("P07B_C1_DEPENDENCY_CLOSURE", JSON.stringify({ local: facts.localDependencies, external: facts.externalDependencies }));
	}
	if (!exact(facts.productionImporters, [
		contractPackagePath, contractRunnerPackagePath, storePackagePath, contractCLITestPackagePath,
	])) {
		add("P07B_C1_IMPORTER_ROSTER", JSON.stringify(facts.productionImporters));
	}
	if ((facts.schema?.unclosedObjects ?? []).length > 0) add("P07B_C1_SCHEMA_CLOSURE", facts.schema.unclosedObjects.join(","));
	if ((facts.schema?.incompleteRequiredObjects ?? []).length > 0) {
		add("P07B_C1_SCHEMA_REQUIRED_ROSTER", facts.schema.incompleteRequiredObjects.join(","));
	}
	if ((facts.schema?.forbiddenMembers ?? []).length > 0) add("P07B_C1_OBJECT_ROSTER", facts.schema.forbiddenMembers.join(","));
	if (
		!exact(facts.schema?.targetRoot, expectedTargetRoot)
		|| !exact(facts.schema?.runRoot, expectedRunRoot)
		|| !exact(facts.schema?.executionRoot, expectedExecutionRoot)
		|| !exact(facts.schema?.objectKinds, expectedObjects)
		|| facts.schema?.startClaimKind !== "StartClaim"
	) add("P07B_C1_OBJECT_ROSTER", "root, kind, or StartClaim roster drift");
	if (
		!exact(facts.schema?.evidenceKinds, expectedEvidenceKinds)
		|| !exact(facts.schema?.startErrors, expectedStartErrors)
		|| !exact(facts.schema?.primaryReasons, expectedPrimaryReasons)
		|| !exact(facts.schema?.cleanupControls, ["TEARDOWN_ERROR", "ORPHAN_RISK"])
		|| !exact(facts.schema?.results, expectedResults)
	) add("P07B_C1_ENUM_ROSTER", "closed enum roster drift");
	if (
		!exact(facts.schema?.scopeDomains, expectedScopeDomains)
		|| !exact(facts.schema?.scopeStates, expectedScopeStates)
		|| !exact(facts.c0?.scopeDomains, expectedScopeDomains)
		|| !exact(facts.c0?.scopeStates, expectedScopeStates)
		|| !exact(facts.example?.scopeOrder, expectedScopeDomains)
	) add("P07B_C1_SCOPE_PROFILE", "scope declaration, schema, or example order drift");
	if (!exact(facts.c0?.objects, expectedObjects)) add("P07B_C1_C0_AUTHORITY", JSON.stringify(facts.c0?.objects));
	if (!exact(facts.schema?.constants, {
		targetScope: "IMMUTABLE_NONHEAD_PRESPAWN_AUTHORITY_V1",
		runScope: "IMMUTABLE_NONHEAD_FINALIZED_RUN_V1",
		executionScope: "IMMUTABLE_NONHEAD_CLASSIFICATION_V1",
		classifierProfile: "CONTRACT_EXECUTION_EXACT_TUPLE_V1",
	})) add("P07B_C1_OBJECT_ROSTER", JSON.stringify(facts.schema?.constants));
	if (!exact(facts.schema?.limits, {
		pathMax: 4096,
		pidMax: 2147483647,
		processEvidenceMax: 8,
		scopeCheckCount: 5,
		privateBlobMax: 16,
		privateByteMax: 67108864,
	})) add("P07B_C1_LIMIT_PROFILE", JSON.stringify(facts.schema?.limits));
	if (
		facts.example?.witnessReferenceCount !== 16
		|| !exact(facts.example?.witnessReferenceKinds, sorted(expectedEvidenceKinds))
	) add("P07B_C1_EXAMPLE_REFERENCE_ROSTER", JSON.stringify(facts.example?.witnessReferenceKinds));
	if (
		facts.example?.graph?.bundleToTarget !== true
		|| facts.example?.graph?.targetToRun !== true
		|| facts.example?.graph?.runToExecution !== true
		|| facts.example?.result !== "CONFORMS"
	) add("P07B_C1_EXAMPLE_GRAPH", JSON.stringify(facts.example));
	return problems;
}

async function snapshot(paths) {
	const result = {};
	for (const relativePath of sorted(paths)) {
		const absolute = resolve(repositoryRoot, relativePath);
		const fromRoot = relative(repositoryRoot, absolute);
		if (isAbsolute(fromRoot) || fromRoot === ".." || fromRoot.startsWith(`..${sep}`)) {
			throw new ArchitectureError("P07B_C1_SNAPSHOT_PATH", relativePath);
		}
		result[slash(relativePath)] = sha256(await readFile(absolute));
	}
	return result;
}

async function runIntersection(facts) {
	const paths = [
		"spec/schema/v1/contract-execution-target.schema.json",
		"spec/schema/v1/finalized-contract-run.schema.json",
		"spec/schema/v1/contract-execution.schema.json",
		"spec/examples/v1/contract-execution-target.valid.json",
		"spec/examples/v1/finalized-contract-run.valid.json",
		"spec/examples/v1/contract-execution.valid.json",
		...facts.topology.modelEntries.map((entry) => `internal/contractexec/model/${entry.slice(0, entry.lastIndexOf(":"))}`),
	];
	const before = await snapshot(paths);
	run(process.execPath, [resolve(repositoryRoot, "tools/generate-p07-planning-example.mjs"), "--check"], "P07B_C1_MODEL_EXAMPLE_INTERSECTION");
	run(process.execPath, [resolve(repositoryRoot, "tools/validate-planning.mjs")], "P07B_C1_MODEL_EXAMPLE_INTERSECTION");
	const after = await snapshot(paths);
	if (!exact(before, after)) throw new ArchitectureError("P07B_C1_SNAPSHOT_CHANGED", "model/schema/example inputs changed during read-only checks");
}

function runInheritedB() {
	const result = spawnSync(process.execPath, [resolve(repositoryRoot, "tools/check-p07b-b-architecture.mjs")], {
		cwd: repositoryRoot,
		encoding: "utf8",
		timeout: 300_000,
		maxBuffer: 32 * 1024 * 1024,
		env: process.env,
	});
	if (result.error || result.signal || result.status !== 0 || result.stderr !== "" ||
		result.stdout.trim() !== "P07B B architecture boundary OK") {
		throw new ArchitectureError(
			"P07B_C2_INHERITED_B_FAILED", `${result.status ?? result.signal}: ${result.stderr || result.stdout || result.error}`,
		);
	}
}

async function runC2Boundary() {
	const before = await snapshot(c2ReviewedPaths);
	runInheritedB();
	const facts = await collectC2Facts();
	const problems = validateC2Facts(facts);
	if (problems.length > 0) {
		for (const problem of problems) process.stderr.write(`${problem.code}: ${problem.detail}\n`);
		process.exitCode = 1;
		return;
	}
	runGoJSONProfile("c2-public-surface");
	const after = await snapshot(c2ReviewedPaths);
	if (!exact(before, after)) {
		throw new ArchitectureError("P07B_C2_SNAPSHOT_CHANGED", "C2 reviewed inputs changed during cumulative checks");
	}
	process.stdout.write("P07B-C C2 cumulative architecture boundary OK\n");
}

async function runC3Boundary() {
	const before = await snapshot(c3ReviewedPaths);
	runInheritedB();
	const facts = await collectC3Facts();
	const predecessorAuthorityBefore = structuredClone(facts.predecessorAuthority);
	const problems = validateC3Facts(facts);
	if (problems.length > 0) {
		for (const problem of problems) process.stderr.write(`${problem.code}: ${problem.detail}\n`);
		process.exitCode = 1;
		return;
	}
	for (const profile of [
		"c3-official-target", "c3-single-target", "c3-hostepoch", "c3-noderuntime", "c3-store-bridge",
	]) runGoJSONProfile(profile);
	const predecessorAuthorityAfter = collectC3PredecessorAuthority();
	if (!exact(predecessorAuthorityBefore, predecessorAuthorityAfter)) {
		throw new ArchitectureError("P07B_C3_PREDECESSOR_CHANGED", "C3 predecessor commit or notes authority changed during cumulative checks");
	}
	const after = await snapshot(c3ReviewedPaths);
	if (!exact(before, after)) {
		throw new ArchitectureError("P07B_C3_SNAPSHOT_CHANGED", "C3 reviewed inputs changed during cumulative checks");
	}
	process.stdout.write("P07B-C C3 cumulative architecture boundary OK\n");
}

async function runC4Boundary() {
	const before = await snapshot(c4BoundarySnapshotPaths);
	runInheritedB();
	const facts = await collectC4Facts();
	const problems = validateC4Facts(facts);
	if (problems.length > 0) {
		for (const problem of problems) process.stderr.write(`${problem.code}: ${problem.detail}\n`);
		process.exitCode = 1;
		return;
	}
	for (const profile of c4ProfileNames) runGoJSONProfile(profile);
	const after = await snapshot(c4BoundarySnapshotPaths);
	if (!exact(before, after)) {
		throw new ArchitectureError("P07B_C4_SNAPSHOT_CHANGED", "C4 reviewed inputs changed during cumulative checks");
	}
	process.stdout.write("P07B-C C4 cumulative architecture boundary OK\n");
}

async function main() {
	if (process.argv[2] === "--assert-go-json") {
		if (process.argv.length !== 4) {
			throw new ArchitectureError("P07B_C1_ARGUMENTS", "--assert-go-json requires one exact profile");
		}
		const result = validateGoJSONTranscript(process.argv[3], await readStandardInput());
		const phase = result.profile.startsWith("c4-") ? "C4" :
			result.profile.startsWith("c3-") ? "C3" : result.profile.startsWith("c2-") ? "C2" : "C1";
		process.stdout.write(`P07B-C ${phase} Go JSON target execution OK (${result.profile}: ${result.passed} passed, ${result.skipped} skipped)\n`);
		return;
	}
	if (process.argv[2] === "--run-go-json") {
		if (process.argv.length !== 4 || !/^c[234]-/u.test(process.argv[3])) {
			throw new ArchitectureError("P07B_C_GO_JSON_ARGUMENTS", "--run-go-json requires one exact C2, C3, or C4 profile");
		}
		const result = runGoJSONProfile(process.argv[3]);
		const phase = result.profile.startsWith("c4-") ? "C4" : result.profile.startsWith("c3-") ? "C3" : "C2";
		process.stdout.write(`P07B-C ${phase} Go JSON target execution OK (${result.profile}: ${result.passed} passed, ${result.skipped} skipped)\n`);
		return;
	}
	if (process.argv[2] === "--c2") {
		if (process.argv.length !== 3) throw new ArchitectureError("P07B_C2_ARGUMENTS", "--c2 accepts no other arguments");
		await runC2Boundary();
		return;
	}
	if (process.argv[2] === "--c3") {
		if (process.argv.length !== 3) throw new ArchitectureError("P07B_C3_ARGUMENTS", "--c3 accepts no other arguments");
		await runC3Boundary();
		return;
	}
	if (process.argv[2] === "--c4") {
		if (process.argv.length !== 3) throw new ArchitectureError("P07B_C4_ARGUMENTS", "--c4 accepts no other arguments");
		await runC4Boundary();
		return;
	}
	if (process.argv.length !== 2) throw new ArchitectureError("P07B_C1_ARGUMENTS", "no arguments accepted");
	const facts = await collectFacts();
	const problems = validateFacts(facts);
	if (problems.length > 0) {
		for (const problem of problems) process.stderr.write(`${problem.code}: ${problem.detail}\n`);
		process.exitCode = 1;
		return;
	}
	await runIntersection(facts);
	process.stdout.write("P07B-C C1 architecture boundary OK\n");
}

if (process.argv[1] && resolve(process.argv[1]) === checkerPath) {
	main().catch((error) => {
		process.stderr.write(`${error.stack ?? error}\n`);
		process.exitCode = 1;
	});
}

// Pure C6A authority for the two exact selected Go test profiles.

const noSkippedTests = Object.freeze([]);

const c5CrossTargetTests = Object.freeze([
	"TestHTTPExactTargetRunJoinsRefuseCrossTargetConfusion",
]);

const c4CLITests = Object.freeze([
	"TestCLIContractExecutionClosesStandaloneScope",
	"TestCLIContractExecutionForbiddenPositiveControls",
	"TestCLIContractExecutionChildBindingEvidenceStates",
	"TestCLIContractExecutionTargetMutationBlocksFinalization",
]);

const c5NodeParityTests = Object.freeze([
	"FuzzParseContractParityCorpusLine",
	"TestContractParityCorpus",
	"TestContractParityCorpusExactSizeBoundaries",
	"TestContractParityCorpusIsOrderAndOracleIndependent",
	"TestContractParityCorpusParserRejectsEnvelopeAliases",
	"TestContractParityCorpusRejectsClosedSchemaDrift",
	"TestContractParityManifestMatchesCopiedEntrypointParser",
	"TestContractParityOracleLeakMutantsAreKilledByRequestRoster",
	"TestDirectResultSelectorExhaustiveGoNodeMatrix",
	"TestGoAndNodeParityEvaluatorsMatchLiteralOracle",
	"TestNodeParityRunnerAcceptsExactWholeWireCap",
	"TestNodeParityRunnerRejectsInvalidFramesAtomically",
	"TestNodeParityRunnerRejectsMissingFinalLFWithoutStderr",
	"TestOwnerEligibilitySelectorExhaustiveGoNodeMatrix",
	"TestParityFramingRejectsExpandedSemanticResultAtomically",
	"TestParityResponseFramingExactBodyBoundary",
	"TestResultFrameBytesExactBoundaries",
]);

const c5HTTPBehaviorInternalTests = Object.freeze([
	"TestC5ProjectionRejectionAgreesWithProcessEvidence",
	"TestC5ResponseErrorControlMatrix",
]);

const c5HTTPBehaviorPublicTests = Object.freeze([
	"TestHTTPContractExecutionConformingDifferingCustom401AndAllowMany",
	"TestHTTPResponseControlMatrixRetainsBoundedCapture",
]);

function target(packageArgument, packagePath, pass) {
	return Object.freeze({
		packageArgument,
		packagePath,
		pass,
		skip: noSkippedTests,
	});
}

function descriptor(name, targets) {
	const tests = targets.flatMap((entry) => entry.pass);
	return Object.freeze({
		arguments: Object.freeze([
			"test",
			"-mod=readonly",
			"-buildvcs=false",
			"-p=1",
			"-count=1",
			"-json",
			"-run",
			`^(?:${tests.join("|")})$`,
			...targets.map((entry) => entry.packageArgument),
		]),
		name,
		targets,
	});
}

export const c6SelectedProfiles = Object.freeze([
	"c5-cross-profile-parity",
	"c5-http-behavior",
]);

export const c6SourceClosurePackages = Object.freeze([
	"./internal/contractexec/http",
	"./internal/emit/node/parity",
	"./testkit/contractexec/cli",
	"./testkit/contractexec/http",
]);

const crossProfileTargets = Object.freeze([
	target(
		"./testkit/contractexec/http",
		"github.com/nelsonwerd/countershape/testkit/contractexec/http",
		c5CrossTargetTests,
	),
	target(
		"./testkit/contractexec/cli",
		"github.com/nelsonwerd/countershape/testkit/contractexec/cli",
		c4CLITests,
	),
	target(
		"./internal/emit/node/parity",
		"github.com/nelsonwerd/countershape/internal/emit/node/parity",
		c5NodeParityTests,
	),
]);

const httpBehaviorTargets = Object.freeze([
	target(
		"./internal/contractexec/http",
		"github.com/nelsonwerd/countershape/internal/contractexec/http",
		c5HTTPBehaviorInternalTests,
	),
	target(
		"./testkit/contractexec/http",
		"github.com/nelsonwerd/countershape/testkit/contractexec/http",
		c5HTTPBehaviorPublicTests,
	),
]);

export const c6ProfileDescriptors = Object.freeze([
	descriptor(c6SelectedProfiles[0], crossProfileTargets),
	descriptor(c6SelectedProfiles[1], httpBehaviorTargets),
]);

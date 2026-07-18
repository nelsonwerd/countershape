package model

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	programv1 "github.com/nelsonwerd/countershape/internal/emit/node/program/v1"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
	"github.com/nelsonwerd/countershape/internal/projectionprofile"
)

const (
	BundleVersionV1            = "node-core-contract-bundle/v1"
	EmitterVersionV1           = "node-exact-emitter/v1"
	BundleDigestDomain         = "ContractBundle"
	ManifestPolicyV1           = "COVERS_OTHER_FIVE_EXCLUDES_SELF_V1"
	RuntimeDependencyProfileV1 = "NODE_CORE_ONLY_V1"
	CountershapeRuntimeBinding = "ABSENT_BY_CONSTRUCTION"
	PackageRegistryBinding     = "NONE"
	EnvironmentProfileV1       = "EXPLICIT_SPARSE_ALLOWLIST_V1"
	ExternalServiceBinding     = "NONE"
	DeterminismScopeV1         = "EMITTER_INVENTED_STRUCTURAL_FACTS_EXCLUDING_AUTHORIZED_INPUT_CONTENT"
	ContractSchemaVersionV1    = "countershape-contract/v1"
	CompiledDecisionKindV1     = "CompiledDecision"
	PortableFixtureKindV1      = "PortableFixture"
	PortableFixtureVersionV1   = "portable-source-fixture/v1"
	IntegrityManifestKindV1    = "IntegrityManifest"
	IntegrityManifestVersionV1 = "countershape-manifest/v1"
	ContractFileModeV1         = "100644"
	ContractFileCount          = 6
	MaxContractFileBytes       = 640 << 10
	MaxAggregateRawFileBytes   = 640 << 10
	MaxOuterNonContentBytes    = 160 << 10
	MaxGeneratedREADMEBytes    = 32 << 10
	CodeInvalidContractBundle  = "INVALID_CONTRACT_BUNDLE"
	CodeBundleDigestMismatch   = "CONTRACT_BUNDLE_DIGEST_MISMATCH"
	CodeCompilerLimitExceeded  = "COMPILER_LIMIT_EXCEEDED"
)

var contractFileRoster = [...]string{
	"README.md",
	programv1.ContractTestPath,
	"decision.json",
	"fixture.json",
	programv1.HarnessPath,
	"manifest.json",
}

type ContractFile struct {
	path       string
	mode       string
	byteSHA256 domain.Digest
	content    []byte
}

func (f ContractFile) Path() string              { return f.path }
func (f ContractFile) Mode() string              { return f.mode }
func (f ContractFile) ByteCount() int            { return len(f.content) }
func (f ContractFile) ByteSHA256() domain.Digest { return f.byteSHA256 }
func (f ContractFile) Content() []byte           { return append([]byte(nil), f.content...) }

type contractBundleSeal struct{}

var sealedContractBundle = &contractBundleSeal{}

// ContractBundle is an inert, recoverable six-file body at one externally
// supplied typed identity. It carries no store, ruling, publication, target,
// execution, or receipt authority.
type ContractBundle struct {
	digest            domain.Digest
	canonical         []byte
	decisionDigest    domain.Digest
	choicepointDigest domain.Digest
	action            string
	source            contractsource.PortableSource
	sourceProfile     SourceProfile
	predicate         Predicate
	files             []ContractFile
	seal              *contractBundleSeal
}

// ParseContractBundle strictly reconstructs a bundle only at the caller's
// expected external typed identity. A coherent body under a newly computed
// digest is a different inert bundle; this parser does not establish
// authenticity, authorship, currentness, or publication authority.
func ParseContractBundle(exact []byte, expectedDigest domain.Digest) (ContractBundle, error) {
	if !expectedDigest.Valid() || len(exact) == 0 || len(exact) > canon.MaxInputBytes {
		return ContractBundle{}, refuse(CodeInvalidContractBundle, "canonical body or expected digest is invalid", nil)
	}
	root, err := canon.Parse(exact)
	if err != nil {
		return ContractBundle{}, refuse(CodeInvalidContractBundle, "outer body is not strict JSON", err)
	}
	canonical, err := root.CanonicalChecked()
	if err != nil || !bytes.Equal(canonical, exact) {
		return ContractBundle{}, refuse(CodeInvalidContractBundle, "outer body is not exact canonical JSON", err)
	}
	digestRaw, err := canon.DigestBytes(BundleDigestDomain, exact)
	if err != nil || digestRaw.String() != expectedDigest.String() {
		return ContractBundle{}, refuse(CodeBundleDigestMismatch, "external bundle digest differs from the exact body", err)
	}
	if err := requireObjectRoster(root, []string{
		"schema_version", "kind", "bundle_version", "decision_record_digest", "choicepoint_digest",
		"portable_source_digest", "portable_profile_digest", "decision_action", "emitter_version",
		"source_profile", "predicate", "files", "manifest_policy", "runtime_dependency_profile",
		"countershape_runtime_binding", "package_registry_binding", "environment_profile",
		"external_service_binding", "determinism_profile", "confidentiality_established",
	}); err != nil {
		return ContractBundle{}, err
	}
	if err := requireStringConstants(root, map[string]string{
		"schema_version":               domain.SchemaVersion,
		"kind":                         "ContractBundle",
		"bundle_version":               BundleVersionV1,
		"emitter_version":              EmitterVersionV1,
		"manifest_policy":              ManifestPolicyV1,
		"runtime_dependency_profile":   RuntimeDependencyProfileV1,
		"countershape_runtime_binding": CountershapeRuntimeBinding,
		"package_registry_binding":     PackageRegistryBinding,
		"environment_profile":          EnvironmentProfileV1,
		"external_service_binding":     ExternalServiceBinding,
	}); err != nil {
		return ContractBundle{}, err
	}
	confidentiality, err := boolMember(root, "confidentiality_established")
	if err != nil || confidentiality {
		return ContractBundle{}, refuse(CodeInvalidContractBundle, "confidentiality_established must be false", err)
	}
	if err := parseDeterminismProfile(member(root, "determinism_profile")); err != nil {
		return ContractBundle{}, err
	}

	decisionDigest, err := digestMember(root, "decision_record_digest")
	if err != nil {
		return ContractBundle{}, err
	}
	choicepointDigest, err := digestMember(root, "choicepoint_digest")
	if err != nil {
		return ContractBundle{}, err
	}
	portableSourceDigest, err := digestMember(root, "portable_source_digest")
	if err != nil {
		return ContractBundle{}, err
	}
	portableProfileDigest, err := digestMember(root, "portable_profile_digest")
	if err != nil {
		return ContractBundle{}, err
	}
	action, err := stringMember(root, "decision_action")
	if err != nil || (action != "ALLOW_OBSERVED" && action != "CUSTOM_EXPECTATION") {
		return ContractBundle{}, refuse(CodeInvalidContractBundle, "decision action is outside the compilable roster", err)
	}

	files, encodedContentBytes, err := parseContractFiles(member(root, "files"))
	if err != nil {
		return ContractBundle{}, err
	}
	if len(exact)-encodedContentBytes > MaxOuterNonContentBytes {
		return ContractBundle{}, refuse(CodeCompilerLimitExceeded, "outer non-content bytes exceed the reviewed ceiling", nil)
	}
	fileByPath := make(map[string]ContractFile, len(files))
	for _, file := range files {
		fileByPath[file.path] = file
	}

	source, err := parsePortableFixture(fileByPath["fixture.json"].content, portableSourceDigest)
	if err != nil {
		return ContractBundle{}, err
	}
	derivedSourceProfile, err := NewSourceProfile(source)
	if err != nil {
		return ContractBundle{}, refuse(CodeInvalidContractBundle, "declared source profile could not be reconstructed", err)
	}
	outerSourceProfile := member(root, "source_profile")
	outerSourceProfileBytes, err := outerSourceProfile.CanonicalChecked()
	if err != nil || !bytes.Equal(outerSourceProfileBytes, derivedSourceProfile.CanonicalBytes()) {
		return ContractBundle{}, refuse(CodeInvalidContractBundle, "outer source profile differs from exact source authority", err)
	}
	if source.Profile().Digest() != portableProfileDigest {
		return ContractBundle{}, refuse(CodeInvalidContractBundle, "outer portable profile digest differs from exact source", nil)
	}
	predicate, err := parsePredicateValue(member(root, "predicate"), source.Profile(), source.StimulusDigest())
	if err != nil {
		return ContractBundle{}, err
	}
	if predicate.PortableProfileDigest() != portableProfileDigest {
		return ContractBundle{}, refuse(CodeInvalidContractBundle, "predicate and outer portable profiles differ", nil)
	}
	if action == "CUSTOM_EXPECTATION" && len(predicate.AllowedTuples()) != 1 {
		return ContractBundle{}, refuse(CodeInvalidContractBundle, "CUSTOM_EXPECTATION requires exactly one tuple", nil)
	}
	if err := parseCompiledDecision(fileByPath["decision.json"].content, action, predicate); err != nil {
		return ContractBundle{}, err
	}
	if err := parseIntegrityManifest(fileByPath["manifest.json"].content, files[:5]); err != nil {
		return ContractBundle{}, err
	}
	if err := compareFixedAssets(fileByPath); err != nil {
		return ContractBundle{}, err
	}
	readme, err := RenderREADME(action, source, predicate)
	if err != nil || !bytes.Equal(readme, fileByPath["README.md"].content) {
		return ContractBundle{}, refuse(CodeInvalidContractBundle, "README differs from the deterministic rerender", err)
	}

	bundle := ContractBundle{
		digest: expectedDigest, canonical: append([]byte(nil), exact...), decisionDigest: decisionDigest,
		choicepointDigest: choicepointDigest, action: action, source: source,
		sourceProfile: derivedSourceProfile, predicate: predicate, files: cloneContractFiles(files),
		seal: sealedContractBundle,
	}
	if !bundle.shallowValid() {
		return ContractBundle{}, refuse(CodeInvalidContractBundle, "reconstructed bundle is invalid", nil)
	}
	return bundle, nil
}

func (b ContractBundle) shallowValid() bool {
	return b.seal == sealedContractBundle && b.digest.Valid() && len(b.canonical) > 0 &&
		b.decisionDigest.Valid() && b.choicepointDigest.Valid() && b.source.Valid() &&
		b.sourceProfile.ValidFor(b.source) && b.predicate.ValidFor(b.source.Profile(), b.source.StimulusDigest()) &&
		len(b.files) == ContractFileCount
}

func (b ContractBundle) Valid() bool {
	if !b.shallowValid() {
		return false
	}
	rebuilt, err := ParseContractBundle(b.canonical, b.digest)
	return err == nil && rebuilt.digest == b.digest && bytes.Equal(rebuilt.canonical, b.canonical)
}

func (b ContractBundle) Digest() domain.Digest               { return b.digest }
func (b ContractBundle) CanonicalBytes() []byte              { return append([]byte(nil), b.canonical...) }
func (b ContractBundle) DecisionRecordDigest() domain.Digest { return b.decisionDigest }
func (b ContractBundle) ChoicepointDigest() domain.Digest    { return b.choicepointDigest }
func (b ContractBundle) DecisionAction() string              { return b.action }
func (b ContractBundle) PortableSource() contractsource.PortableSource {
	reparsed, _ := contractsource.Parse(b.source.CanonicalBytes())
	return reparsed
}
func (b ContractBundle) SourceProfile() SourceProfile { return b.sourceProfile }
func (b ContractBundle) Predicate() Predicate         { return b.predicate }
func (b ContractBundle) Files() []ContractFile        { return cloneContractFiles(b.files) }

func cloneContractFiles(files []ContractFile) []ContractFile {
	result := make([]ContractFile, len(files))
	for index, file := range files {
		result[index] = ContractFile{
			path: file.path, mode: file.mode, byteSHA256: file.byteSHA256,
			content: append([]byte(nil), file.content...),
		}
	}
	return result
}

func parseContractFiles(value canon.Value) ([]ContractFile, int, error) {
	items, ok := value.Elements()
	if !ok || len(items) != ContractFileCount {
		return nil, 0, refuse(CodeInvalidContractBundle, "files must contain exactly six entries", nil)
	}
	files := make([]ContractFile, len(items))
	aggregate := 0
	encoded := 0
	for index, item := range items {
		if err := requireObjectRoster(item, []string{"path", "mode", "byte_count", "byte_sha256", "content_base64"}); err != nil {
			return nil, 0, err
		}
		path, err := stringMember(item, "path")
		if err != nil || path != contractFileRoster[index] {
			return nil, 0, refuse(CodeInvalidContractBundle, "file roster or order differs", err)
		}
		mode, err := stringMember(item, "mode")
		if err != nil || mode != ContractFileModeV1 {
			return nil, 0, refuse(CodeInvalidContractBundle, "contract file mode differs", err)
		}
		count, err := intMember(item, "byte_count")
		if err != nil || count <= 0 {
			return nil, 0, refuse(CodeInvalidContractBundle, "contract file byte count must be one positive integer", err)
		}
		if count > MaxContractFileBytes {
			return nil, 0, refuse(CodeCompilerLimitExceeded, "contract file byte count exceeds the ceiling", nil)
		}
		digest, err := digestMember(item, "byte_sha256")
		if err != nil {
			return nil, 0, err
		}
		contentText, err := stringMember(item, "content_base64")
		if err != nil {
			return nil, 0, err
		}
		content, err := strictBase64(contentText, MaxContractFileBytes)
		if err != nil || len(content) != count || rawSHA256(content) != digest.String() {
			return nil, 0, refuse(CodeInvalidContractBundle, "file count, base64, or raw digest differs", err)
		}
		if err := validateGeneratedText(content); err != nil {
			return nil, 0, err
		}
		aggregate += len(content)
		encoded += len(contentText)
		if aggregate > MaxAggregateRawFileBytes {
			return nil, 0, refuse(CodeCompilerLimitExceeded, "aggregate raw file bytes exceed the ceiling", nil)
		}
		files[index] = ContractFile{path: path, mode: mode, byteSHA256: digest, content: content}
	}
	return files, encoded, nil
}

func parsePortableFixture(exact []byte, expectedDigest domain.Digest) (contractsource.PortableSource, error) {
	body, err := parseJSONFileEnvelope(exact)
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	if err := requireObjectRoster(body, []string{
		"schema_version", "kind", "fixture_version", "portable_source_digest", "portable_source_base64",
	}); err != nil {
		return contractsource.PortableSource{}, err
	}
	if err := requireStringConstants(body, map[string]string{
		"schema_version":  ContractSchemaVersionV1,
		"kind":            PortableFixtureKindV1,
		"fixture_version": PortableFixtureVersionV1,
	}); err != nil {
		return contractsource.PortableSource{}, err
	}
	digest, err := digestMember(body, "portable_source_digest")
	if err != nil || digest != expectedDigest {
		return contractsource.PortableSource{}, refuse(CodeInvalidContractBundle, "fixture and outer PortableSource digests differ", err)
	}
	encoded, err := stringMember(body, "portable_source_base64")
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	exactSource, err := strictBase64(encoded, contractsource.MaxSourceCanonicalBytes)
	if err != nil {
		return contractsource.PortableSource{}, refuse(CodeInvalidContractBundle, "fixture PortableSource base64 is invalid", err)
	}
	source, err := contractsource.Parse(exactSource)
	if err != nil || source.Digest() != digest || !bytes.Equal(source.CanonicalBytes(), exactSource) {
		return contractsource.PortableSource{}, refuse(CodeInvalidContractBundle, "fixture does not recover the exact PortableSource", err)
	}
	return source, nil
}

func parseCompiledDecision(exact []byte, action string, predicate Predicate) error {
	body, err := parseJSONFileEnvelope(exact)
	if err != nil {
		return err
	}
	if err := requireObjectRoster(body, []string{"schema_version", "kind", "decision_action", "predicate"}); err != nil {
		return err
	}
	if err := requireStringConstants(body, map[string]string{
		"schema_version":  ContractSchemaVersionV1,
		"kind":            CompiledDecisionKindV1,
		"decision_action": action,
	}); err != nil {
		return err
	}
	predicateBytes, err := member(body, "predicate").CanonicalChecked()
	if err != nil || !bytes.Equal(predicateBytes, predicate.CanonicalBytes()) {
		return refuse(CodeInvalidContractBundle, "decision and outer predicate bytes differ", err)
	}
	return nil
}

func parseIntegrityManifest(exact []byte, protected []ContractFile) error {
	body, err := ParseIntegrityManifestEnvelope(exact)
	if err != nil {
		return err
	}
	items, ok := member(body, "files").Elements()
	if !ok || len(items) != 5 || len(protected) != 5 {
		return refuse(CodeInvalidContractBundle, "manifest must cover exactly the other five files", nil)
	}
	for index, item := range items {
		if err := requireObjectRoster(item, []string{"path", "mode", "byte_count", "byte_sha256"}); err != nil {
			return err
		}
		path, err := stringMember(item, "path")
		if err != nil || path != protected[index].path || path == "manifest.json" {
			return refuse(CodeInvalidContractBundle, "manifest path roster differs or covers itself", err)
		}
		mode, err := stringMember(item, "mode")
		if err != nil || mode != protected[index].mode {
			return refuse(CodeInvalidContractBundle, "manifest mode differs", err)
		}
		count, err := intMember(item, "byte_count")
		if err != nil || count != len(protected[index].content) {
			return refuse(CodeInvalidContractBundle, "manifest byte count differs", err)
		}
		digest, err := digestMember(item, "byte_sha256")
		if err != nil || digest != protected[index].byteSHA256 {
			return refuse(CodeInvalidContractBundle, "manifest raw digest differs", err)
		}
	}
	return nil
}

// ParseIntegrityManifestEnvelope validates the exact inert five-file manifest
// schema without claiming that its entries agree with any external files. The
// bundle parser performs those content joins separately in
// parseIntegrityManifest. This split lets parity exercise the production
// manifest parser without fabricating protected-file authority.
func ParseIntegrityManifestEnvelope(exact []byte) (canon.Value, error) {
	if len(exact) == 0 || len(exact) > MaxContractFileBytes {
		return canon.Value{}, refuse(CodeInvalidContractBundle, "manifest envelope exceeds the contract-file ceiling", nil)
	}
	body, err := parseJSONFileEnvelope(exact)
	if err != nil {
		return canon.Value{}, err
	}
	if err := requireObjectRoster(body, []string{"schema_version", "kind", "manifest_version", "files"}); err != nil {
		return canon.Value{}, err
	}
	if err := requireStringConstants(body, map[string]string{
		"schema_version":   ContractSchemaVersionV1,
		"kind":             IntegrityManifestKindV1,
		"manifest_version": IntegrityManifestVersionV1,
	}); err != nil {
		return canon.Value{}, err
	}
	items, ok := member(body, "files").Elements()
	if !ok || len(items) != len(contractFileRoster)-1 {
		return canon.Value{}, refuse(CodeInvalidContractBundle, "manifest must cover exactly the other five files", nil)
	}
	for index, item := range items {
		if err := requireObjectRoster(item, []string{"path", "mode", "byte_count", "byte_sha256"}); err != nil {
			return canon.Value{}, err
		}
		path, err := stringMember(item, "path")
		if err != nil || path != contractFileRoster[index] || path == "manifest.json" {
			return canon.Value{}, refuse(CodeInvalidContractBundle, "manifest path roster differs or covers itself", err)
		}
		mode, err := stringMember(item, "mode")
		if err != nil || mode != ContractFileModeV1 {
			return canon.Value{}, refuse(CodeInvalidContractBundle, "manifest mode differs", err)
		}
		count, err := intMember(item, "byte_count")
		if err != nil || count <= 0 || count > MaxContractFileBytes {
			return canon.Value{}, refuse(CodeInvalidContractBundle, "manifest byte count is outside the reviewed ceiling", err)
		}
		if _, err := digestMember(item, "byte_sha256"); err != nil {
			return canon.Value{}, refuse(CodeInvalidContractBundle, "manifest raw digest is invalid", err)
		}
	}
	return body, nil
}

func compareFixedAssets(files map[string]ContractFile) error {
	for _, path := range []string{programv1.ContractTestPath, programv1.HarnessPath} {
		expected, err := programv1.Bytes(path)
		if err != nil || !bytes.Equal(expected, files[path].content) {
			return refuse(CodeInvalidContractBundle, "fixed Node program asset differs", err)
		}
	}
	return nil
}

// RenderREADME produces the only admitted human documentation body. All text
// is fixed or derived from closed source/profile/predicate fields.
func RenderREADME(action string, source contractsource.PortableSource, predicate Predicate) ([]byte, error) {
	if !source.Valid() || !predicate.ValidFor(source.Profile(), source.StimulusDigest()) ||
		(action != "ALLOW_OBSERVED" && action != "CUSTOM_EXPECTATION") {
		return nil, refuse(CodeInvalidContractBundle, "README inputs are invalid", nil)
	}
	profile, err := NewSourceProfile(source)
	if err != nil {
		return nil, err
	}
	var output strings.Builder
	output.WriteString("# Countershape experimental contract\n\n")
	output.WriteString("This contract reruns one observed behavior from the declared adapter profile and checks only the listed output fields. It is a Countershape-generated experimental selected-field contract.\n\n")
	output.WriteString("> WARNING: FULL USER AUTHORITY AND NETWORK ACCESS. This executes subject code with your user permissions and host network access. It is not a sandbox or containment boundary.\n\n")
	output.WriteString("**CONFIDENTIALITY NOT ESTABLISHED.** Exact fixture, input, environment, and source bytes may be sensitive.\n\n")
	output.WriteString("## What this contract checks\n\n")
	fmt.Fprintf(&output, "- Adapter: `%s`\n", profile.AdapterDomain())
	fmt.Fprintf(&output, "- Repository-relative subject entrypoint: `%s`\n", profile.SubjectEntrypoint())
	fmt.Fprintf(&output, "- Decision action: `%s`\n", action)
	fmt.Fprintf(&output, "- Scope: exact witnessed stimulus `%s`\n", source.StimulusDigest())
	output.WriteString("- Selected fields, in order:\n")
	for index, field := range predicate.SelectedFields() {
		fmt.Fprintf(&output, "  %d. `%s`\n", index+1, field)
	}
	output.WriteString("\nPlain English: the run passes only when the listed fields match one allowed exact combination from the original observation. Other context can change without failing this contract.\n\n")
	output.WriteString("## Before you run\n\n")
	output.WriteString("Countershape does not prepare the run directory in this release. Before running the contract, create a source-only copy of the target.\n\n")
	output.WriteString("The prepared target must:\n\n")
	output.WriteString("- Contain the repository-relative subject entrypoint shown above.\n")
	output.WriteString("- Contain no `.git` directory, symlink, hardlink, socket, device, FIFO, or other special entry.\n")
	output.WriteString("- Use only supported regular-file modes.\n")
	output.WriteString("- Remain distinct from and non-nested with the bundle directory.\n\n")
	output.WriteString("Use an existing reviewed source-export process. If you cannot make and inspect that copy, stop: this bundle does not provide a safe preparation command.\n\n")
	output.WriteString("Portability is not established. Environments without every required runtime, filesystem, watcher, and process-group facility report `INELIGIBLE_EXECUTION|ENVIRONMENT_INVALID`; a conforming run does not prove portability.\n\n")
	output.WriteString("## Run it\n\n")
	output.WriteString("Run these commands from a shell after preparing the separate target copy:\n\n")
	output.WriteString("```text\ncd \"<absolute-prepared-target-root>\" &&\n\"<absolute-node-executable>\" --test \\\n  --test-reporter=tap \\\n  \"<absolute-bundle-root>/contract.test.mjs\"\n```\n\n")
	output.WriteString("Replace only the three placeholder texts and leave the double quotes in place; the shown quoting preserves ordinary spaces. If a value contains a double quote, dollar sign, backtick, backslash, or newline, stop and use a reviewed argv-based launcher instead of editing this shell block. Direct invocation establishes no Git target identity.\n\n")
	output.WriteString("The selected TAP stream contains exactly one machine diagnostic:\n\n")
	output.WriteString("```text\n# COUNTERSHAPE_RESULT_V1|<outcome>|<reason>\n```\n\n")
	output.WriteString("Controlled runs keep outer stderr empty. Only `CONFORMS|NONE` exits zero.\n\n")
	output.WriteString("## Read the result\n\n")
	output.WriteString("- `CONFORMS|NONE`: the listed behavior matched one allowed exact combination.\n")
	output.WriteString("- `CONTRADICTS|PREDICATE_MISMATCH`: the run was usable, but the listed behavior differed.\n")
	output.WriteString("- `INELIGIBLE_EXECUTION|<reason>`: the run cannot support a behavioral conclusion.\n")
	output.WriteString("- `MALFORMED_CONTRACT`, `TAMPER_DETECTED`, or `HARNESS_FAILURE`: artifact, integrity, or control failure; do not interpret subject behavior.\n\n")
	output.WriteString("If behavior contradicts, inspect the listed fields before deciding whether the subject or contract should change. For an ineligible, malformed, tampered, or harness-failure result, repair the environment or recreate and verify the bundle through your trusted workflow, then rerun. Those outcomes are not subject-behavior results.\n\n")
	output.WriteString("## Integrity and nonclaims\n\n")
	output.WriteString("The unchanged contract entrypoint checks every protected companion file before it imports the harness. A mismatch with this bundle's expected bytes is reported as an integrity failure.\n\n")
	output.WriteString("Direct invocation does not establish:\n\n")
	output.WriteString("- Who created or approved the bundle.\n")
	output.WriteString("- Authenticity, Git identity, or currentness.\n")
	output.WriteString("- Correctness or portability beyond this exact run.\n")
	output.WriteString("- Resistance to coordinated replacement of the bundle.\n")
	output.WriteString("- Containment or long-term maintainability.\n")
	rendered := []byte(output.String())
	if len(rendered) == 0 || len(rendered) > MaxGeneratedREADMEBytes || rendered[len(rendered)-1] != '\n' ||
		bytes.IndexByte(rendered, '\r') >= 0 || bytes.IndexByte(rendered, 0) >= 0 {
		return nil, refuse(CodeCompilerLimitExceeded, "README exceeds its deterministic text envelope", nil)
	}
	return rendered, nil
}

func parsePredicateValue(value canon.Value, profile projectionprofile.Profile, stimulus domain.Digest) (Predicate, error) {
	if err := requireObjectRoster(value, []string{
		"kind", "scope", "stimulus_digest", "portable_profile_digest", "selected_fields", "allowed_tuples",
	}); err != nil {
		return Predicate{}, err
	}
	if err := requireStringConstants(value, map[string]string{
		"kind":                    PredicateKindV1,
		"scope":                   PredicateScopeV1,
		"stimulus_digest":         stimulus.String(),
		"portable_profile_digest": profile.Digest().String(),
	}); err != nil {
		return Predicate{}, err
	}
	selectedValues, ok := member(value, "selected_fields").Elements()
	if !ok || len(selectedValues) == 0 || len(selectedValues) > 64 {
		return Predicate{}, refuse(CodeInvalidContractBundle, "selected field roster is invalid", nil)
	}
	selected := make([]string, len(selectedValues))
	for index, item := range selectedValues {
		text, ok := item.Text()
		if !ok {
			return Predicate{}, refuse(CodeInvalidContractBundle, "selected field is not a string", nil)
		}
		selected[index] = text
	}
	tupleValues, ok := member(value, "allowed_tuples").Elements()
	if !ok || len(tupleValues) == 0 || len(tupleValues) > 4 {
		return Predicate{}, refuse(CodeInvalidContractBundle, "allowed tuple roster is invalid", nil)
	}
	tuples := make([]ExactTuple, len(tupleValues))
	for index, item := range tupleValues {
		tuple, err := parseExactTuple(item)
		if err != nil {
			return Predicate{}, err
		}
		tuples[index] = tuple
	}
	predicate, err := NewPredicate(stimulus, profile, selected, tuples)
	if err != nil {
		return Predicate{}, refuse(CodeInvalidContractBundle, "predicate reconstruction failed", err)
	}
	rebuilt, err := predicateValue(predicate)
	if err != nil {
		return Predicate{}, err
	}
	rebuiltBytes, _ := rebuilt.CanonicalChecked()
	inputBytes, _ := value.CanonicalChecked()
	if !bytes.Equal(rebuiltBytes, inputBytes) {
		return Predicate{}, refuse(CodeInvalidContractBundle, "predicate is not in exact canonical tuple order", nil)
	}
	return predicate, nil
}

// ParsePredicate reconstructs one exact canonical predicate body through the
// same closed parser used by ContractBundle recovery. It accepts no file
// envelope and performs no I/O.
func ParsePredicate(exact []byte, profile projectionprofile.Profile, stimulus domain.Digest) (Predicate, error) {
	if len(exact) == 0 || !profile.Valid() || !stimulus.Valid() {
		return Predicate{}, refuse(CodeInvalidContractBundle, "predicate authority or body is invalid", nil)
	}
	value, err := canon.Parse(exact)
	if err != nil {
		return Predicate{}, refuse(CodeInvalidContractBundle, "predicate body is invalid", err)
	}
	canonical, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonical, exact) {
		return Predicate{}, refuse(CodeInvalidContractBundle, "predicate body is not exact canonical bytes", err)
	}
	return parsePredicateValue(value, profile, stimulus)
}

func parseExactTuple(value canon.Value) (ExactTuple, error) {
	if err := requireObjectRoster(value, []string{"fields"}); err != nil {
		return ExactTuple{}, err
	}
	items, ok := member(value, "fields").Elements()
	if !ok || len(items) == 0 || len(items) > portablevalue.MaxTupleFields {
		return ExactTuple{}, refuse(CodeInvalidContractBundle, "tuple fields are invalid", nil)
	}
	fields := make([]ExactField, len(items))
	for index, item := range items {
		if err := requireObjectRoster(item, []string{"field_id", "value"}); err != nil {
			return ExactTuple{}, err
		}
		fieldID, err := stringMember(item, "field_id")
		if err != nil {
			return ExactTuple{}, err
		}
		exactValue, err := parseExactValue(member(item, "value"))
		if err != nil {
			return ExactTuple{}, err
		}
		fields[index], err = NewExactField(fieldID, exactValue)
		if err != nil {
			return ExactTuple{}, err
		}
	}
	return NewExactTuple(fields)
}

// ParseExactTuple reconstructs one exact canonical tuple body through the
// bundle parser's value algebra without granting it predicate membership.
func ParseExactTuple(exact []byte) (ExactTuple, error) {
	if len(exact) == 0 {
		return ExactTuple{}, refuse(CodeInvalidContractBundle, "tuple body is empty", nil)
	}
	value, err := canon.Parse(exact)
	if err != nil {
		return ExactTuple{}, refuse(CodeInvalidContractBundle, "tuple body is invalid", err)
	}
	canonical, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonical, exact) {
		return ExactTuple{}, refuse(CodeInvalidContractBundle, "tuple body is not exact canonical bytes", err)
	}
	return parseExactTuple(value)
}

func parseExactValue(value canon.Value) (ExactValue, error) {
	tagText, err := stringMember(value, "tag")
	if err != nil {
		return ExactValue{}, err
	}
	tag := portablevalue.Tag(tagText)
	var portable portablevalue.Value
	switch tag {
	case portablevalue.TagMissing:
		if err := requireObjectRoster(value, []string{"tag"}); err != nil {
			return ExactValue{}, err
		}
		portable = portablevalue.Missing()
	case portablevalue.TagNull:
		if err := requireObjectRoster(value, []string{"tag"}); err != nil {
			return ExactValue{}, err
		}
		portable = portablevalue.Null()
	case portablevalue.TagBoolean:
		if err := requireObjectRoster(value, []string{"tag", "value"}); err != nil {
			return ExactValue{}, err
		}
		boolean, err := boolMember(value, "value")
		if err != nil {
			return ExactValue{}, err
		}
		portable = portablevalue.Boolean(boolean)
	case portablevalue.TagInteger:
		if err := requireObjectRoster(value, []string{"tag", "canonical"}); err != nil {
			return ExactValue{}, err
		}
		canonical, err := stringMember(value, "canonical")
		if err != nil {
			return ExactValue{}, err
		}
		portable, err = portablevalue.Integer(canonical)
		if err != nil {
			return ExactValue{}, err
		}
	case portablevalue.TagString:
		if err := requireObjectRoster(value, []string{"tag", "value"}); err != nil {
			return ExactValue{}, err
		}
		text, err := stringMember(value, "value")
		if err != nil {
			return ExactValue{}, err
		}
		portable, err = portablevalue.String(text)
		if err != nil {
			return ExactValue{}, err
		}
	case portablevalue.TagBytes:
		if err := requireObjectRoster(value, []string{"tag", "base64"}); err != nil {
			return ExactValue{}, err
		}
		encoded, err := stringMember(value, "base64")
		if err != nil {
			return ExactValue{}, err
		}
		opaque, err := strictBase64(encoded, portablevalue.MaxBytesValueBytes)
		if err != nil {
			return ExactValue{}, err
		}
		portable, err = portablevalue.Bytes(opaque)
		if err != nil {
			return ExactValue{}, err
		}
	case portablevalue.TagOrderedStringList:
		if err := requireObjectRoster(value, []string{"tag", "values"}); err != nil {
			return ExactValue{}, err
		}
		items, ok := member(value, "values").Elements()
		if !ok || len(items) > portablevalue.MaxListMembers {
			return ExactValue{}, refuse(CodeInvalidContractBundle, "ordered string list is invalid", nil)
		}
		values := make([]string, len(items))
		for index, item := range items {
			text, ok := item.Text()
			if !ok {
				return ExactValue{}, refuse(CodeInvalidContractBundle, "ordered list member is not a string", nil)
			}
			values[index] = text
		}
		portable, err = portablevalue.OrderedStringList(values)
		if err != nil {
			return ExactValue{}, err
		}
	case portablevalue.TagCanonicalJSON:
		if err := requireObjectRoster(value, []string{"tag", "canonical_base64"}); err != nil {
			return ExactValue{}, err
		}
		encoded, err := stringMember(value, "canonical_base64")
		if err != nil {
			return ExactValue{}, err
		}
		canonical, err := strictBase64(encoded, portablevalue.MaxCanonicalJSONBytes)
		if err != nil {
			return ExactValue{}, err
		}
		portable, err = portablevalue.CanonicalJSON(canonical)
		if err != nil {
			return ExactValue{}, err
		}
	default:
		return ExactValue{}, refuse(CodeInvalidContractBundle, "exact value tag is unknown", nil)
	}
	return NewExactValue(portable)
}

func predicateValue(predicate Predicate) (canon.Value, error) {
	return canon.Parse(predicate.CanonicalBytes())
}

func parseDeterminismProfile(value canon.Value) error {
	if err := requireObjectRoster(value, []string{
		"scope", "emitter_introduces_time", "emitter_introduces_random_id",
		"emitter_introduces_absolute_path", "emitter_introduces_concrete_candidate_identity",
		"emitter_introduces_declared_secret_value", "emitter_introduces_host_runtime_fact",
		"emitter_introduces_execution_receipt",
	}); err != nil {
		return err
	}
	if err := requireStringConstants(value, map[string]string{"scope": DeterminismScopeV1}); err != nil {
		return err
	}
	for _, name := range []string{
		"emitter_introduces_time", "emitter_introduces_random_id", "emitter_introduces_absolute_path",
		"emitter_introduces_concrete_candidate_identity", "emitter_introduces_declared_secret_value",
		"emitter_introduces_host_runtime_fact", "emitter_introduces_execution_receipt",
	} {
		boolean, err := boolMember(value, name)
		if err != nil || boolean {
			return refuse(CodeInvalidContractBundle, "determinism structural flags must be false", err)
		}
	}
	return nil
}

func parseJSONFileEnvelope(exact []byte) (canon.Value, error) {
	if err := validateGeneratedText(exact); err != nil {
		return canon.Value{}, err
	}
	body := exact[:len(exact)-1]
	if len(body) == 0 {
		return canon.Value{}, refuse(CodeInvalidContractBundle, "JSON file body is empty", nil)
	}
	value, err := canon.Parse(body)
	if err != nil {
		return canon.Value{}, refuse(CodeInvalidContractBundle, "JSON file body is invalid", err)
	}
	canonical, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonical, body) {
		return canon.Value{}, refuse(CodeInvalidContractBundle, "JSON file body is not exact canonical bytes", err)
	}
	return value, nil
}

func validateGeneratedText(exact []byte) error {
	if len(exact) == 0 || !utf8.Valid(exact) || bytes.HasPrefix(exact, []byte{0xef, 0xbb, 0xbf}) ||
		bytes.IndexByte(exact, 0) >= 0 || bytes.IndexByte(exact, '\r') >= 0 || exact[len(exact)-1] != '\n' {
		return refuse(CodeInvalidContractBundle, "generated file is not BOM-free UTF-8 LF text", nil)
	}
	if len(exact) > 1 && exact[len(exact)-2] == '\n' {
		return refuse(CodeInvalidContractBundle, "generated file does not end in exactly one LF", nil)
	}
	return nil
}

func requireObjectRoster(value canon.Value, names []string) error {
	members, ok := value.Members()
	if !ok || len(members) != len(names) {
		return refuse(CodeInvalidContractBundle, "object member roster differs", nil)
	}
	expected := append([]string(nil), names...)
	sort.Strings(expected)
	for index, item := range members {
		if item.Name != expected[index] {
			return refuse(CodeInvalidContractBundle, "object member roster differs", nil)
		}
	}
	return nil
}

func requireStringConstants(value canon.Value, expected map[string]string) error {
	for name, want := range expected {
		got, err := stringMember(value, name)
		if err != nil || got != want {
			return refuse(CodeInvalidContractBundle, fmt.Sprintf("%s constant differs", name), err)
		}
	}
	return nil
}

func member(value canon.Value, name string) canon.Value {
	result, _ := value.LookupMember(name)
	return result
}

func stringMember(value canon.Value, name string) (string, error) {
	child, ok := value.LookupMember(name)
	if !ok {
		return "", refuse(CodeInvalidContractBundle, fmt.Sprintf("missing %s", name), nil)
	}
	text, ok := child.Text()
	if !ok {
		return "", refuse(CodeInvalidContractBundle, fmt.Sprintf("%s is not a string", name), nil)
	}
	return text, nil
}

func boolMember(value canon.Value, name string) (bool, error) {
	child, ok := value.LookupMember(name)
	if !ok {
		return false, refuse(CodeInvalidContractBundle, fmt.Sprintf("missing %s", name), nil)
	}
	boolean, ok := child.Boolean()
	if !ok {
		return false, refuse(CodeInvalidContractBundle, fmt.Sprintf("%s is not a boolean", name), nil)
	}
	return boolean, nil
}

func intMember(value canon.Value, name string) (int, error) {
	child, ok := value.LookupMember(name)
	if !ok {
		return 0, refuse(CodeInvalidContractBundle, fmt.Sprintf("missing %s", name), nil)
	}
	integer, ok := child.Int64()
	if !ok || integer < 0 || int64(int(integer)) != integer {
		return 0, refuse(CodeInvalidContractBundle, fmt.Sprintf("%s is not a bounded integer", name), nil)
	}
	return int(integer), nil
}

func digestMember(value canon.Value, name string) (domain.Digest, error) {
	text, err := stringMember(value, name)
	if err != nil {
		return domain.Digest(""), err
	}
	digest, err := domain.ParseDigest(text)
	if err != nil {
		return domain.Digest(""), refuse(CodeInvalidContractBundle, fmt.Sprintf("%s is not a digest", name), err)
	}
	return digest, nil
}

func strictBase64(encoded string, maximum int) ([]byte, error) {
	if len(encoded) > base64.StdEncoding.EncodedLen(maximum) {
		return nil, refuse(CodeInvalidContractBundle, "base64 value exceeds its bound", nil)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(decoded) > maximum || base64.StdEncoding.EncodeToString(decoded) != encoded {
		return nil, refuse(CodeInvalidContractBundle, "base64 value is not canonical padded encoding", err)
	}
	return decoded, nil
}

func rawSHA256(exact []byte) string {
	digest := sha256.Sum256(exact)
	return "sha256:" + hex.EncodeToString(digest[:])
}

// StaticCeilingProof reports the compile-time upper bound established by the
// reviewed aggregate and non-content limits.
func StaticCeilingProof() (encodedContentMaximum, outerMaximum int) {
	encodedContentMaximum = 4*((MaxAggregateRawFileBytes+2)/3) + 4*(ContractFileCount-1)
	outerMaximum = encodedContentMaximum + MaxOuterNonContentBytes
	return encodedContentMaximum, outerMaximum
}

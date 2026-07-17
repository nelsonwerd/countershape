package compiler

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/emit/node/internal/compilation"
	"github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
	contractfixtures "github.com/nelsonwerd/countershape/testkit/contracts"
)

func TestCompileRecoversExactSixFileBundle(t *testing.T) {
	input := testCompilationInput(t)
	bundle, err := Compile(input)
	if err != nil {
		t.Fatal(err)
	}
	if !bundle.Valid() || !bundle.Digest().Valid() || bundle.DecisionRecordDigest() != input.DecisionRecordDigest() ||
		bundle.ChoicepointDigest() != input.ChoicepointDigest() || bundle.DecisionAction() != string(input.Action()) {
		t.Fatal("compiled bundle did not preserve the exact inert authority links")
	}
	wantPaths := []string{"README.md", "contract.test.mjs", "decision.json", "fixture.json", "harness.mjs", "manifest.json"}
	files := bundle.Files()
	if len(files) != len(wantPaths) {
		t.Fatalf("file count = %d, want %d", len(files), len(wantPaths))
	}
	for index, file := range files {
		if file.Path() != wantPaths[index] || file.Mode() != model.ContractFileModeV1 ||
			file.ByteCount() == 0 || !file.ByteSHA256().Valid() || file.Content()[len(file.Content())-1] != '\n' {
			t.Fatalf("file %d is not the exact closed roster entry: %#v", index, file)
		}
	}
	if bundle.PortableSource().Digest() != input.Source().Digest() ||
		!bytes.Equal(bundle.PortableSource().CanonicalBytes(), input.Source().CanonicalBytes()) ||
		bundle.SourceProfile().Digest() != input.SourceProfile().Digest() ||
		!bytes.Equal(bundle.Predicate().CanonicalBytes(), input.Predicate().CanonicalBytes()) {
		t.Fatal("bundle did not recover source, declared profile, and predicate exactly")
	}
	reparsed, err := model.ParseContractBundle(bundle.CanonicalBytes(), bundle.Digest())
	if err != nil || !reparsed.Valid() || !bytes.Equal(reparsed.CanonicalBytes(), bundle.CanonicalBytes()) {
		t.Fatalf("strict reparse failed: %v", err)
	}
}

func TestCompileIsDeterministicAndDefensive(t *testing.T) {
	input := testCompilationInput(t)
	first, err := Compile(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Compile(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest() != second.Digest() || !bytes.Equal(first.CanonicalBytes(), second.CanonicalBytes()) {
		t.Fatal("repeated compilation changed bundle identity")
	}
	canonical := first.CanonicalBytes()
	canonical[0] ^= 0xff
	files := first.Files()
	content := files[0].Content()
	content[0] ^= 0xff
	if !first.Valid() || bytes.Equal(canonical, first.CanonicalBytes()) || bytes.Equal(content, first.Files()[0].Content()) {
		t.Fatal("caller mutation crossed the bundle copy boundary")
	}
	wrong := fixedDigest("f")
	if _, err := model.ParseContractBundle(first.CanonicalBytes(), wrong); !model.IsCode(err, model.CodeBundleDigestMismatch) {
		t.Fatalf("wrong expected digest error = %v", err)
	}
	tampered := first.CanonicalBytes()
	tampered[len(tampered)-1] ^= 1
	if parsed, err := model.ParseContractBundle(tampered, first.Digest()); err == nil || parsed.Valid() {
		t.Fatal("tampered canonical body parsed at the original identity")
	}
}

func TestParseContractBundleRejectsCoherentlyRehashedFixedAssetTamper(t *testing.T) {
	bundle, err := Compile(testCompilationInput(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"contract.test.mjs", "harness.mjs"} {
		t.Run(target, func(t *testing.T) {
			exact, digest := coherentlyRehashFixedAssetTamper(t, bundle.CanonicalBytes(), target)
			if digest == bundle.Digest() || bytes.Equal(exact, bundle.CanonicalBytes()) {
				t.Fatal("coherent tamper did not change the external bundle identity")
			}
			parsed, parseErr := model.ParseContractBundle(exact, digest)
			if !model.IsCode(parseErr, model.CodeInvalidContractBundle) || parsed.Valid() {
				t.Fatalf("coherently rehashed fixed asset parsed: valid=%t err=%v", parsed.Valid(), parseErr)
			}
		})
	}
}

func TestParseContractBundleMutationLadderReachesNamedJoins(t *testing.T) {
	bundle, err := Compile(testCompilationInput(t))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("original-identity-is-required", func(t *testing.T) {
		outer := decodeTestJSONObject(t, bundle.CanonicalBytes())
		outer["decision_record_digest"] = fixedDigest("e").String()
		exact, recomputed := recomputeTestBundleIdentity(t, outer)
		if bytes.Equal(exact, bundle.CanonicalBytes()) || recomputed == bundle.Digest() {
			t.Fatal("lineage mutation did not change the exact bundle identity")
		}
		if parsed, parseErr := model.ParseContractBundle(exact, bundle.Digest()); !model.IsCode(parseErr, model.CodeBundleDigestMismatch) || parsed.Valid() {
			t.Fatalf("body mutation parsed at the original identity: valid=%t err=%v", parsed.Valid(), parseErr)
		}
		parsed, parseErr := model.ParseContractBundle(exact, recomputed)
		if parseErr != nil || !parsed.Valid() || parsed.DecisionRecordDigest() != fixedDigest("e") {
			t.Fatalf("fully coherent inert lineage rewrite did not recover: valid=%t err=%v", parsed.Valid(), parseErr)
		}
	})

	t.Run("raw-file-metadata-join", func(t *testing.T) {
		outer := decodeTestJSONObject(t, bundle.CanonicalBytes())
		readme := testOuterFile(t, outer, "README.md")
		content := append(decodeTestBase64(t, readme["content_base64"]), []byte("hostile raw mismatch\n")...)
		readme["content_base64"] = base64.StdEncoding.EncodeToString(content)
		exact, digest := recomputeTestBundleIdentity(t, outer)
		if parsed, parseErr := model.ParseContractBundle(exact, digest); !model.IsCode(parseErr, model.CodeInvalidContractBundle) || parsed.Valid() {
			t.Fatalf("raw file metadata mismatch parsed: valid=%t err=%v", parsed.Valid(), parseErr)
		}
	})

	t.Run("deterministic-readme-rerender", func(t *testing.T) {
		outer := decodeTestJSONObject(t, bundle.CanonicalBytes())
		readme := testOuterFile(t, outer, "README.md")
		content := append(decodeTestBase64(t, readme["content_base64"]), []byte("hostile but coherently rehashed text\n")...)
		exact, digest := coherentlyRewriteTestFile(t, outer, "README.md", content)
		parsed, parseErr := model.ParseContractBundle(exact, digest)
		if !model.IsCode(parseErr, model.CodeInvalidContractBundle) || parsed.Valid() ||
			!strings.Contains(parseErr.Error(), "README differs from the deterministic rerender") {
			t.Fatalf("coherently rehashed README drift did not reach rerender join: valid=%t err=%v", parsed.Valid(), parseErr)
		}
	})
}

func TestParseContractBundleRejectsEveryGeneratedTextEnvelopeAlias(t *testing.T) {
	bundle, err := Compile(testCompilationInput(t))
	if err != nil {
		t.Fatal(err)
	}
	aliases := []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{"missing-lf", func(exact []byte) []byte { return append([]byte(nil), exact[:len(exact)-1]...) }},
		{"crlf", func(exact []byte) []byte { return append(append([]byte(nil), exact[:len(exact)-1]...), '\r', '\n') }},
		{"double-lf", func(exact []byte) []byte { return append(append([]byte(nil), exact...), '\n') }},
		{"bom", func(exact []byte) []byte { return append([]byte{0xef, 0xbb, 0xbf}, exact...) }},
		{"raw-nul", func(exact []byte) []byte {
			return append(append([]byte(nil), exact[:len(exact)-1]...), 0, '\n')
		}},
		{"invalid-utf8", func(exact []byte) []byte {
			return append(append([]byte(nil), exact[:len(exact)-1]...), 0xff, '\n')
		}},
	}
	paths := []string{"README.md", "contract.test.mjs", "decision.json", "fixture.json", "harness.mjs", "manifest.json"}
	for _, path := range paths {
		for _, alias := range aliases {
			t.Run(path+"/"+alias.name, func(t *testing.T) {
				outer := decodeTestJSONObject(t, bundle.CanonicalBytes())
				file := testOuterFile(t, outer, path)
				original := decodeTestBase64(t, file["content_base64"])
				mutated := alias.mutate(original)
				if bytes.Equal(original, mutated) {
					t.Fatal("text-envelope mutator was equivalent")
				}
				exact, digest := coherentlyRewriteTestFile(t, outer, path, mutated)
				parsed, parseErr := model.ParseContractBundle(exact, digest)
				if !model.IsCode(parseErr, model.CodeInvalidContractBundle) || parsed.Valid() {
					t.Fatalf("%s envelope alias parsed: valid=%t err=%v", alias.name, parsed.Valid(), parseErr)
				}
			})
		}
	}
}

func TestParseContractBundleRejectsSemanticCrossPairsAfterCoherentRehash(t *testing.T) {
	t.Run("predicate-source-stimulus", func(t *testing.T) {
		base, err := Compile(testCompilationInput(t))
		if err != nil {
			t.Fatal(err)
		}
		alternateStdin, err := cli.PresentStdin([]byte("different-stimulus"))
		if err != nil {
			t.Fatal(err)
		}
		alternateSource, err := contractfixtures.CLISourceWithStdin(alternateStdin)
		if err != nil {
			t.Fatal(err)
		}
		alternate, err := Compile(testCLICompilationInput(
			t, alternateSource, [][]byte{[]byte("ok\n")}, compilation.ActionAllowObserved,
		))
		if err != nil {
			t.Fatal(err)
		}
		outer := decodeTestJSONObject(t, base.CanonicalBytes())
		alternateOuter := decodeTestJSONObject(t, alternate.CanonicalBytes())
		outer["predicate"] = alternateOuter["predicate"]
		decisionFile := testOuterFile(t, outer, "decision.json")
		decision := decodeTestJSONObject(t, decodeTestBase64(t, decisionFile["content_base64"]))
		decision["predicate"] = alternateOuter["predicate"]
		decisionBytes := append(canonicalTestJSON(t, decision), '\n')
		exact, digest := coherentlyRewriteTestFile(t, outer, "decision.json", decisionBytes)
		if parsed, parseErr := model.ParseContractBundle(exact, digest); !model.IsCode(parseErr, model.CodeInvalidContractBundle) || parsed.Valid() {
			t.Fatalf("predicate/source stimulus cross-pair parsed: valid=%t err=%v", parsed.Valid(), parseErr)
		}
	})

	t.Run("descriptor-value-tag", func(t *testing.T) {
		bundle, err := Compile(testCompilationInput(t))
		if err != nil {
			t.Fatal(err)
		}
		outer := decodeTestJSONObject(t, bundle.CanonicalBytes())
		predicate, ok := outer["predicate"].(map[string]any)
		if !ok {
			t.Fatal("outer predicate is not an object")
		}
		tuples, ok := predicate["allowed_tuples"].([]any)
		if !ok || len(tuples) != 1 {
			t.Fatal("outer predicate does not have one allowed tuple")
		}
		tuple, ok := tuples[0].(map[string]any)
		if !ok {
			t.Fatal("allowed tuple is not an object")
		}
		fields, ok := tuple["fields"].([]any)
		if !ok || len(fields) != 1 {
			t.Fatal("allowed tuple does not have one selected field")
		}
		field, ok := fields[0].(map[string]any)
		if !ok {
			t.Fatal("selected field is not an object")
		}
		field["value"] = map[string]any{"tag": "STRING", "value": "ok\n"}
		decisionFile := testOuterFile(t, outer, "decision.json")
		decision := decodeTestJSONObject(t, decodeTestBase64(t, decisionFile["content_base64"]))
		decision["predicate"] = predicate
		decisionBytes := append(canonicalTestJSON(t, decision), '\n')
		exact, digest := coherentlyRewriteTestFile(t, outer, "decision.json", decisionBytes)
		if parsed, parseErr := model.ParseContractBundle(exact, digest); !model.IsCode(parseErr, model.CodeInvalidContractBundle) || parsed.Valid() {
			t.Fatalf("descriptor/value-tag cross-pair parsed: valid=%t err=%v", parsed.Valid(), parseErr)
		}
	})

	t.Run("custom-expectation-cardinality", func(t *testing.T) {
		source, err := contractfixtures.CLISource()
		if err != nil {
			t.Fatal(err)
		}
		bundle, err := Compile(testCLICompilationInput(
			t, source, [][]byte{[]byte("first\n"), []byte("second\n")}, compilation.ActionAllowObserved,
		))
		if err != nil {
			t.Fatal(err)
		}
		outer := decodeTestJSONObject(t, bundle.CanonicalBytes())
		outer["decision_action"] = string(compilation.ActionCustomExpectation)
		decisionFile := testOuterFile(t, outer, "decision.json")
		decision := decodeTestJSONObject(t, decodeTestBase64(t, decisionFile["content_base64"]))
		decision["decision_action"] = string(compilation.ActionCustomExpectation)
		decisionBytes := append(canonicalTestJSON(t, decision), '\n')
		exact, digest := coherentlyRewriteTestFile(t, outer, "decision.json", decisionBytes)
		if parsed, parseErr := model.ParseContractBundle(exact, digest); !model.IsCode(parseErr, model.CodeInvalidContractBundle) || parsed.Valid() {
			t.Fatalf("multi-tuple custom expectation parsed: valid=%t err=%v", parsed.Valid(), parseErr)
		}
	})
}

func coherentlyRehashFixedAssetTamper(t testing.TB, exact []byte, target string) ([]byte, domain.Digest) {
	t.Helper()
	outer := decodeTestJSONObject(t, exact)
	files, ok := outer["files"].([]any)
	if !ok || len(files) != model.ContractFileCount {
		t.Fatal("compiled outer body has no exact file roster")
	}
	var targetFile, manifestFile map[string]any
	for _, raw := range files {
		file, fileOK := raw.(map[string]any)
		if !fileOK {
			t.Fatal("compiled outer file is not an object")
		}
		switch file["path"] {
		case target:
			targetFile = file
		case "manifest.json":
			manifestFile = file
		}
	}
	if targetFile == nil || manifestFile == nil {
		t.Fatalf("compiled outer body omits %s or manifest.json", target)
	}
	targetBytes := decodeTestBase64(t, targetFile["content_base64"])
	targetBytes = append(targetBytes, []byte("// coherently rehashed hostile asset\n")...)
	setTestFileMetadata(targetFile, targetBytes)

	manifestBytes := decodeTestBase64(t, manifestFile["content_base64"])
	manifest := decodeTestJSONObject(t, manifestBytes)
	manifestFiles, ok := manifest["files"].([]any)
	if !ok || len(manifestFiles) != model.ContractFileCount-1 {
		t.Fatal("compiled manifest has no exact protected-file roster")
	}
	found := false
	for _, raw := range manifestFiles {
		entry, entryOK := raw.(map[string]any)
		if !entryOK {
			t.Fatal("compiled manifest file is not an object")
		}
		if entry["path"] == target {
			entry["byte_count"] = json.Number(strconv.Itoa(len(targetBytes)))
			entry["byte_sha256"] = testRawSHA256(targetBytes)
			found = true
		}
	}
	if !found {
		t.Fatalf("compiled manifest omits %s", target)
	}
	manifestBytes = canonicalTestJSON(t, manifest)
	manifestBytes = append(manifestBytes, '\n')
	setTestFileMetadata(manifestFile, manifestBytes)

	mutated := canonicalTestJSON(t, outer)
	rawDigest, err := canon.DigestBytes(model.BundleDigestDomain, mutated)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := domain.ParseDigest(rawDigest.String())
	if err != nil {
		t.Fatal(err)
	}
	return mutated, digest
}

func testOuterFile(t testing.TB, outer map[string]any, path string) map[string]any {
	t.Helper()
	files, ok := outer["files"].([]any)
	if !ok || len(files) != model.ContractFileCount {
		t.Fatal("compiled outer body has no exact file roster")
	}
	for _, raw := range files {
		file, fileOK := raw.(map[string]any)
		if !fileOK {
			t.Fatal("compiled outer file is not an object")
		}
		if file["path"] == path {
			return file
		}
	}
	t.Fatalf("compiled outer body omits %s", path)
	return nil
}

func coherentlyRewriteTestFile(
	t testing.TB,
	outer map[string]any,
	path string,
	content []byte,
) ([]byte, domain.Digest) {
	t.Helper()
	setTestFileMetadata(testOuterFile(t, outer, path), content)
	if path != "manifest.json" {
		manifestFile := testOuterFile(t, outer, "manifest.json")
		manifest := decodeTestJSONObject(t, decodeTestBase64(t, manifestFile["content_base64"]))
		entries, ok := manifest["files"].([]any)
		if !ok || len(entries) != model.ContractFileCount-1 {
			t.Fatal("compiled manifest has no exact protected-file roster")
		}
		found := false
		for _, raw := range entries {
			entry, entryOK := raw.(map[string]any)
			if !entryOK {
				t.Fatal("compiled manifest entry is not an object")
			}
			if entry["path"] == path {
				entry["byte_count"] = json.Number(strconv.Itoa(len(content)))
				entry["byte_sha256"] = testRawSHA256(content)
				found = true
			}
		}
		if !found {
			t.Fatalf("compiled manifest omits %s", path)
		}
		manifestBytes := append(canonicalTestJSON(t, manifest), '\n')
		setTestFileMetadata(manifestFile, manifestBytes)
	}
	return recomputeTestBundleIdentity(t, outer)
}

func recomputeTestBundleIdentity(t testing.TB, outer map[string]any) ([]byte, domain.Digest) {
	t.Helper()
	exact := canonicalTestJSON(t, outer)
	rawDigest, err := canon.DigestBytes(model.BundleDigestDomain, exact)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := domain.ParseDigest(rawDigest.String())
	if err != nil {
		t.Fatal(err)
	}
	return exact, digest
}

func decodeTestJSONObject(t testing.TB, exact []byte) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(exact))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func decodeTestBase64(t testing.TB, raw any) []byte {
	t.Helper()
	encoded, ok := raw.(string)
	if !ok {
		t.Fatal("compiled file content is not base64 text")
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || base64.StdEncoding.EncodeToString(decoded) != encoded {
		t.Fatalf("compiled file content is not exact standard base64: %v", err)
	}
	return decoded
}

func setTestFileMetadata(file map[string]any, content []byte) {
	file["byte_count"] = json.Number(strconv.Itoa(len(content)))
	file["byte_sha256"] = testRawSHA256(content)
	file["content_base64"] = base64.StdEncoding.EncodeToString(content)
}

func testRawSHA256(content []byte) string {
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func canonicalTestJSON(t testing.TB, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := canon.Canonicalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func TestCompileCeilingProofFitsCanonicalAuthority(t *testing.T) {
	encoded, outer := model.StaticCeilingProof()
	if encoded != 4*((model.MaxAggregateRawFileBytes+2)/3)+4*(model.ContractFileCount-1) {
		t.Fatalf("encoded ceiling = %d", encoded)
	}
	if outer != encoded+model.MaxOuterNonContentBytes || outer >= canon.MaxInputBytes {
		t.Fatalf("outer ceiling = %d, canonical maximum = %d", outer, canon.MaxInputBytes)
	}
	if _, err := Compile(compilation.Input{}); !IsCode(err, CodeInvalidCompilerInput) {
		t.Fatalf("zero compiler input error = %v", err)
	}
}

func TestRenderedREADMEStatesInvocationAndTrustCeilings(t *testing.T) {
	bundle, err := Compile(testCompilationInput(t))
	if err != nil {
		t.Fatal(err)
	}
	readme := string(bundle.Files()[0].Content())
	for _, required := range []string{
		"reruns one observed behavior from the declared adapter profile and checks only the listed output fields",
		"Countershape-generated experimental selected-field contract",
		"cd \"<absolute-prepared-target-root>\" &&\n\"<absolute-node-executable>\" --test \\\n  --test-reporter=tap \\\n  \"<absolute-bundle-root>/contract.test.mjs\"",
		"# COUNTERSHAPE_RESULT_V1|<outcome>|<reason>",
		"Controlled runs keep outer stderr empty",
		"Other context can change without failing this contract",
		"Countershape does not prepare the run directory in this release",
		"Contain no `.git` directory, symlink, hardlink, socket, device, FIFO, or other special entry",
		"If you cannot make and inspect that copy, stop",
		"CONFORMS|NONE", "CONTRADICTS|PREDICATE_MISMATCH", "INELIGIBLE_EXECUTION|<reason>",
		"MALFORMED_CONTRACT", "TAMPER_DETECTED", "HARNESS_FAILURE",
		"a conforming run does not prove portability",
		"user permissions and host network access",
		"FULL USER AUTHORITY AND NETWORK ACCESS",
		"CONFIDENTIALITY NOT ESTABLISHED",
		"the shown quoting preserves ordinary spaces",
		"use a reviewed argv-based launcher",
		"Those outcomes are not subject-behavior results",
		"Who created or approved the bundle", "Correctness or portability", "Containment or long-term maintainability",
	} {
		if !strings.Contains(readme, required) {
			t.Fatalf("README omits %q", required)
		}
	}
	if strings.Contains(readme, bundle.Digest().String()) {
		t.Fatal("README embeds the external bundle digest")
	}
	warning := strings.Index(readme, "> WARNING:")
	command := strings.Index(readme, "\"<absolute-node-executable>\" --test \\\n")
	if warning < 0 || command < 0 || warning >= command {
		t.Fatal("README does not place the full-authority warning before the copyable command")
	}
	if !strings.Contains(readme, "- Adapter: `CLI`") {
		t.Fatal("CLI README omits its declared adapter")
	}
	httpBundle, err := Compile(testHTTPCompilationInput(t))
	if err != nil {
		t.Fatal(err)
	}
	httpREADME := string(httpBundle.Files()[0].Content())
	if !strings.Contains(httpREADME, "- Adapter: `HTTP`") ||
		!strings.Contains(httpREADME, "reruns one observed behavior from the declared adapter profile") ||
		strings.Contains(httpREADME, "observed CLI behavior") {
		t.Fatal("HTTP README is not adapter-correct")
	}
}

func TestCompileAndRecoveryDoNotPolicyScanAuthorizedOpaqueContent(t *testing.T) {
	opaque := "candidate_key=/Users/example/private receipt=sha256:deadbeef secret=must-remain-authorized-data"
	source, err := contractfixtures.CLISourceWithEnvironmentValue(opaque)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(source.CanonicalBytes(), []byte(opaque)) {
		t.Fatal("opaque source fixture did not retain the policy-looking authorized bytes")
	}
	input := testCLICompilationInput(
		t, source,
		[][]byte{[]byte("/absolute-looking/output candidate receipt secret\n")},
		compilation.ActionAllowObserved,
	)
	bundle, err := Compile(input)
	if err != nil {
		t.Fatalf("compiler content-scanned authorized source or predicate bytes: %v", err)
	}
	parsed, err := model.ParseContractBundle(bundle.CanonicalBytes(), bundle.Digest())
	if err != nil || !parsed.Valid() || !bytes.Contains(parsed.PortableSource().CanonicalBytes(), []byte(opaque)) {
		t.Fatalf("recovery content-scanned or lost authorized opaque bytes: valid=%t err=%v", parsed.Valid(), err)
	}
}

func TestCompilationInputRejectsMultipleCustomExpectationTuples(t *testing.T) {
	source, err := contractfixtures.CLISource()
	if err != nil {
		t.Fatal(err)
	}
	allowMany := testCLICompilationInput(
		t, source, [][]byte{[]byte("first\n"), []byte("second\n")}, compilation.ActionAllowObserved,
	)
	input, err := compilation.New(
		allowMany.DecisionRecordDigest(), allowMany.ChoicepointDigest(), compilation.ActionCustomExpectation,
		allowMany.Source(), allowMany.SourceProfile(), allowMany.Predicate(),
	)
	if err == nil || input.Valid() || !strings.Contains(err.Error(), "CUSTOM_EXPECTATION requires exactly one tuple") {
		t.Fatalf("multi-tuple custom compilation input was admitted: valid=%t err=%v", input.Valid(), err)
	}
}

func testCompilationInput(t testing.TB) compilation.Input {
	t.Helper()
	source, err := contractfixtures.CLISource()
	if err != nil {
		t.Fatal(err)
	}
	return testCLICompilationInput(t, source, [][]byte{[]byte("ok\n")}, compilation.ActionAllowObserved)
}

func testCLICompilationInput(
	t testing.TB,
	source contractsource.PortableSource,
	allowedOutputs [][]byte,
	action compilation.DecisionAction,
) compilation.Input {
	t.Helper()
	profile, err := model.NewSourceProfile(source)
	if err != nil {
		t.Fatal(err)
	}
	tuples := make([]model.ExactTuple, len(allowedOutputs))
	for index, output := range allowedOutputs {
		value, valueErr := portablevalue.Bytes(output)
		if valueErr != nil {
			t.Fatal(valueErr)
		}
		exact, exactErr := model.NewExactValue(value)
		if exactErr != nil {
			t.Fatal(exactErr)
		}
		field, fieldErr := model.NewExactField("cli.stdout.bytes", exact)
		if fieldErr != nil {
			t.Fatal(fieldErr)
		}
		tuples[index], fieldErr = model.NewExactTuple([]model.ExactField{field})
		if fieldErr != nil {
			t.Fatal(fieldErr)
		}
	}
	predicate, err := model.NewPredicate(
		source.StimulusDigest(), source.Profile(), []string{"cli.stdout.bytes"}, tuples,
	)
	if err != nil {
		t.Fatal(err)
	}
	input, err := compilation.New(
		fixedDigest("d"), fixedDigest("c"), action,
		source, profile, predicate,
	)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func testCLICompletionCompilationInput(
	t testing.TB,
	source contractsource.PortableSource,
	completion string,
) compilation.Input {
	t.Helper()
	profile, err := model.NewSourceProfile(source)
	if err != nil {
		t.Fatal(err)
	}
	value, err := portablevalue.String(completion)
	if err != nil {
		t.Fatal(err)
	}
	exact, err := model.NewExactValue(value)
	if err != nil {
		t.Fatal(err)
	}
	field, err := model.NewExactField("cli.completion.kind", exact)
	if err != nil {
		t.Fatal(err)
	}
	tuple, err := model.NewExactTuple([]model.ExactField{field})
	if err != nil {
		t.Fatal(err)
	}
	predicate, err := model.NewPredicate(
		source.StimulusDigest(), source.Profile(), []string{"cli.completion.kind"}, []model.ExactTuple{tuple},
	)
	if err != nil {
		t.Fatal(err)
	}
	input, err := compilation.New(
		fixedDigest("d"), fixedDigest("c"), compilation.ActionAllowObserved,
		source, profile, predicate,
	)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func testHTTPCompilationInput(t testing.TB) compilation.Input {
	t.Helper()
	source, err := contractfixtures.HTTPSource()
	if err != nil {
		t.Fatal(err)
	}
	return testHTTPCompilationInputForSource(t, source, "200")
}

func testHTTPCompilationInputForSource(
	t testing.TB,
	source contractsource.PortableSource,
	expectedStatus string,
) compilation.Input {
	t.Helper()
	profile, err := model.NewSourceProfile(source)
	if err != nil {
		t.Fatal(err)
	}
	status, err := portablevalue.Integer(expectedStatus)
	if err != nil {
		t.Fatal(err)
	}
	contentTypes, err := portablevalue.OrderedStringList([]string{"application/json"})
	if err != nil {
		t.Fatal(err)
	}
	kind, err := portablevalue.String("ok")
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := portablevalue.CanonicalJSON([]byte(`{"mode":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	values := []portablevalue.Value{status, contentTypes, kind, metadata}
	fieldIDs := []string{
		"http.status", "http.header.content-type", "http.body.kind", "http.body.metadata",
	}
	fields := make([]model.ExactField, len(values))
	for index, value := range values {
		exact, exactErr := model.NewExactValue(value)
		if exactErr != nil {
			t.Fatal(exactErr)
		}
		fields[index], exactErr = model.NewExactField(fieldIDs[index], exact)
		if exactErr != nil {
			t.Fatal(exactErr)
		}
	}
	tuple, err := model.NewExactTuple(fields)
	if err != nil {
		t.Fatal(err)
	}
	predicate, err := model.NewPredicate(source.StimulusDigest(), source.Profile(), fieldIDs, []model.ExactTuple{tuple})
	if err != nil {
		t.Fatal(err)
	}
	input, err := compilation.New(
		fixedDigest("b"), fixedDigest("a"), compilation.ActionAllowObserved,
		source, profile, predicate,
	)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func FuzzParseContractBundle(f *testing.F) {
	bundle, err := Compile(testCompilationInput(f))
	if err != nil {
		f.Fatal(err)
	}
	seed := bundle.CanonicalBytes()
	f.Add([]byte{}, uint8(0), true)
	f.Add([]byte("coherent-readme-drift"), uint8(0), true)
	f.Add(seed, uint8(0), false)
	f.Add([]byte(`{"kind":"not-a-contract"}`), uint8(5), false)
	paths := []string{"README.md", "contract.test.mjs", "decision.json", "fixture.json", "harness.mjs", "manifest.json"}
	f.Fuzz(func(t *testing.T, mutation []byte, selector uint8, coherent bool) {
		if len(mutation) > 128 {
			mutation = mutation[:128]
		}
		exact := append([]byte(nil), mutation...)
		expected := bundle.Digest()
		if coherent {
			outer := decodeTestJSONObject(t, seed)
			path := paths[int(selector)%len(paths)]
			file := testOuterFile(t, outer, path)
			content := decodeTestBase64(t, file["content_base64"])
			if len(mutation) > 0 {
				content = append([]byte(nil), content...)
				offset := int(selector) % len(content)
				content[offset] ^= mutation[0] | 1
				content = append(content, mutation[1:]...)
			}
			exact, expected = coherentlyRewriteTestFile(t, outer, path, content)
		}
		parsed, parseErr := model.ParseContractBundle(exact, expected)
		if parseErr == nil && (!parsed.Valid() || parsed.Digest() != expected ||
			!bytes.Equal(parsed.CanonicalBytes(), exact)) {
			t.Fatal("successful fuzz parse did not return the exact defensive bundle")
		}
	})
}

func fixedDigest(character string) domain.Digest {
	digest, _ := domain.ParseDigest("sha256:" + strings.Repeat(character, 64))
	return digest
}

// Package compiler owns Countershape's pure deterministic Node contract
// compiler. It consumes only the sealed internal compilation input and returns
// one strictly reparsed inert bundle. Production code in this package performs
// no filesystem, process, environment, clock, randomness, runtime, Git, store,
// or network operation.
package compiler

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/emit/node/internal/compilation"
	"github.com/nelsonwerd/countershape/internal/emit/node/model"
	programv1 "github.com/nelsonwerd/countershape/internal/emit/node/program/v1"
)

const (
	CodeInvalidCompilerInput = "INVALID_COMPILER_INPUT"
	CodeCompilerLimit        = model.CodeCompilerLimitExceeded
	CodeCompilerInvariant    = "COMPILER_INVARIANT_FAILED"
	MaxPredicateBytes        = 128 << 10
	MaxSourceProfileBytes    = 8 << 10
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
	var target *Error
	return errors.As(err, &target) && target.Code == code
}

func refuse(code, detail string, cause error) error {
	return &Error{Code: code, Detail: detail, Cause: cause}
}

type generatedFile struct {
	path    string
	mode    string
	content []byte
	digest  domain.Digest
}

// Compile deterministically emits and strictly reparses exactly six in-memory
// files. It is intentionally partial over valid A2.1 inputs when the nested
// recovery body cannot fit the reviewed A2.2 ceilings.
func Compile(input compilation.Input) (model.ContractBundle, error) {
	if !input.Valid() {
		return model.ContractBundle{}, refuse(CodeInvalidCompilerInput, "sealed sanitized input is invalid", nil)
	}
	_, outerMaximum := model.StaticCeilingProof()
	if outerMaximum >= canon.MaxInputBytes {
		return model.ContractBundle{}, refuse(CodeCompilerInvariant, "static ceiling proof no longer fits canonical authority", nil)
	}
	predicateBytes := input.Predicate().CanonicalBytes()
	profileBytes := input.SourceProfile().CanonicalBytes()
	if len(predicateBytes) == 0 || len(predicateBytes) > MaxPredicateBytes ||
		len(profileBytes) == 0 || len(profileBytes) > MaxSourceProfileBytes {
		return model.ContractBundle{}, refuse(CodeCompilerLimit, "predicate or source profile exceeds the compiler ceiling", nil)
	}
	predicateValue, err := parseExactValue(predicateBytes, "predicate")
	if err != nil {
		return model.ContractBundle{}, err
	}
	profileValue, err := parseExactValue(profileBytes, "source profile")
	if err != nil {
		return model.ContractBundle{}, err
	}

	decision, err := buildDecisionFile(input, predicateValue)
	if err != nil {
		return model.ContractBundle{}, err
	}
	fixture, err := buildFixtureFile(input)
	if err != nil {
		return model.ContractBundle{}, err
	}
	readme, err := model.RenderREADME(string(input.Action()), input.Source(), input.Predicate())
	if err != nil {
		if model.IsCode(err, model.CodeCompilerLimitExceeded) {
			return model.ContractBundle{}, refuse(CodeCompilerLimit, "README exceeds the compiler ceiling", err)
		}
		return model.ContractBundle{}, refuse(CodeCompilerInvariant, "README render failed", err)
	}
	entrypoint, err := programv1.Bytes(programv1.ContractTestPath)
	if err != nil {
		return model.ContractBundle{}, refuse(CodeCompilerInvariant, "fixed entrypoint asset failed", err)
	}
	harness, err := programv1.Bytes(programv1.HarnessPath)
	if err != nil {
		return model.ContractBundle{}, refuse(CodeCompilerInvariant, "fixed harness asset failed", err)
	}
	files := []generatedFile{
		{path: "README.md", mode: model.ContractFileModeV1, content: readme},
		{path: programv1.ContractTestPath, mode: model.ContractFileModeV1, content: entrypoint},
		{path: "decision.json", mode: model.ContractFileModeV1, content: decision},
		{path: "fixture.json", mode: model.ContractFileModeV1, content: fixture},
		{path: programv1.HarnessPath, mode: model.ContractFileModeV1, content: harness},
	}
	if err := enforceFileCeilings(files, model.ContractFileCount-1); err != nil {
		return model.ContractBundle{}, err
	}
	if err := finalizeFileDigests(files); err != nil {
		return model.ContractBundle{}, err
	}
	manifest, err := buildManifestFile(files)
	if err != nil {
		return model.ContractBundle{}, err
	}
	files = append(files, generatedFile{
		path: "manifest.json", mode: model.ContractFileModeV1, content: manifest,
	})
	if err := enforceFileCeilings(files, model.ContractFileCount); err != nil {
		return model.ContractBundle{}, err
	}
	manifestDigest, err := parseRawDigest(rawSHA256(manifest))
	if err != nil {
		return model.ContractBundle{}, refuse(CodeCompilerInvariant, "manifest raw digest is invalid", err)
	}
	files[len(files)-1].digest = manifestDigest
	body, encodedContentBytes, err := buildOuterBody(input, profileValue, predicateValue, files)
	if err != nil {
		return model.ContractBundle{}, err
	}
	if len(body)-encodedContentBytes > model.MaxOuterNonContentBytes || len(body) > canon.MaxInputBytes {
		return model.ContractBundle{}, refuse(CodeCompilerLimit, "final canonical body exceeds the reviewed ceiling", nil)
	}
	digestRaw, err := canon.DigestBytes(model.BundleDigestDomain, body)
	if err != nil {
		return model.ContractBundle{}, refuse(CodeCompilerInvariant, "bundle digest failed", err)
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return model.ContractBundle{}, refuse(CodeCompilerInvariant, "bundle digest is outside the domain model", err)
	}
	bundle, err := model.ParseContractBundle(body, digest)
	if err != nil || !bundle.Valid() {
		return model.ContractBundle{}, refuse(CodeCompilerInvariant, "compiler output failed strict recovery", err)
	}
	return bundle, nil
}

func buildDecisionFile(input compilation.Input, predicate canon.Value) ([]byte, error) {
	value, err := compilerObject(
		compilerTextField("schema_version", model.ContractSchemaVersionV1),
		compilerTextField("kind", model.CompiledDecisionKindV1),
		compilerTextField("decision_action", string(input.Action())),
		compilerValueField("predicate", predicate),
	)
	if err != nil {
		return nil, refuse(CodeCompilerInvariant, "decision body assembly failed", err)
	}
	return jsonFileEnvelope(value, "decision")
}

func buildFixtureFile(input compilation.Input) ([]byte, error) {
	source := input.Source()
	value, err := compilerObject(
		compilerTextField("schema_version", model.ContractSchemaVersionV1),
		compilerTextField("kind", model.PortableFixtureKindV1),
		compilerTextField("fixture_version", model.PortableFixtureVersionV1),
		compilerTextField("portable_source_digest", source.Digest().String()),
		compilerTextField("portable_source_base64", base64.StdEncoding.EncodeToString(source.CanonicalBytes())),
	)
	if err != nil {
		return nil, refuse(CodeCompilerInvariant, "fixture body assembly failed", err)
	}
	return jsonFileEnvelope(value, "fixture")
}

func buildManifestFile(files []generatedFile) ([]byte, error) {
	entries := make([]canon.Value, len(files))
	for index, file := range files {
		count, err := canon.Integer(int64(len(file.content)))
		if err != nil {
			return nil, refuse(CodeCompilerInvariant, "manifest byte count is outside the canonical integer profile", err)
		}
		value, err := compilerObject(
			compilerTextField("path", file.path),
			compilerTextField("mode", file.mode),
			compilerValueField("byte_count", count),
			compilerTextField("byte_sha256", file.digest.String()),
		)
		if err != nil {
			return nil, refuse(CodeCompilerInvariant, "manifest entry assembly failed", err)
		}
		entries[index] = value
	}
	array, err := canon.Array(entries...)
	if err != nil {
		return nil, refuse(CodeCompilerInvariant, "manifest file roster failed", err)
	}
	value, err := compilerObject(
		compilerTextField("schema_version", model.ContractSchemaVersionV1),
		compilerTextField("kind", model.IntegrityManifestKindV1),
		compilerTextField("manifest_version", model.IntegrityManifestVersionV1),
		compilerValueField("files", array),
	)
	if err != nil {
		return nil, refuse(CodeCompilerInvariant, "manifest body assembly failed", err)
	}
	return jsonFileEnvelope(value, "manifest")
}

func buildOuterBody(
	input compilation.Input,
	profile, predicate canon.Value,
	files []generatedFile,
) ([]byte, int, error) {
	fileValues := make([]canon.Value, len(files))
	encodedContentBytes := 0
	for index, file := range files {
		encoded := base64.StdEncoding.EncodeToString(file.content)
		encodedContentBytes += len(encoded)
		count, err := canon.Integer(int64(len(file.content)))
		if err != nil {
			return nil, 0, refuse(CodeCompilerInvariant, "outer file byte count is outside the canonical integer profile", err)
		}
		entry, err := compilerObject(
			compilerTextField("path", file.path),
			compilerTextField("mode", file.mode),
			compilerValueField("byte_count", count),
			compilerTextField("byte_sha256", file.digest.String()),
			compilerTextField("content_base64", encoded),
		)
		if err != nil {
			return nil, 0, refuse(CodeCompilerInvariant, "outer file entry assembly failed", err)
		}
		fileValues[index] = entry
	}
	fileArray, err := canon.Array(fileValues...)
	if err != nil {
		return nil, 0, refuse(CodeCompilerInvariant, "outer file roster failed", err)
	}
	determinism, err := determinismValue()
	if err != nil {
		return nil, 0, err
	}
	root, err := compilerObject(
		compilerTextField("schema_version", domain.SchemaVersion),
		compilerTextField("kind", "ContractBundle"),
		compilerTextField("bundle_version", model.BundleVersionV1),
		compilerTextField("decision_record_digest", input.DecisionRecordDigest().String()),
		compilerTextField("choicepoint_digest", input.ChoicepointDigest().String()),
		compilerTextField("portable_source_digest", input.Source().Digest().String()),
		compilerTextField("portable_profile_digest", input.Predicate().PortableProfileDigest().String()),
		compilerTextField("decision_action", string(input.Action())),
		compilerTextField("emitter_version", model.EmitterVersionV1),
		compilerValueField("source_profile", profile),
		compilerValueField("predicate", predicate),
		compilerValueField("files", fileArray),
		compilerTextField("manifest_policy", model.ManifestPolicyV1),
		compilerTextField("runtime_dependency_profile", model.RuntimeDependencyProfileV1),
		compilerTextField("countershape_runtime_binding", model.CountershapeRuntimeBinding),
		compilerTextField("package_registry_binding", model.PackageRegistryBinding),
		compilerTextField("environment_profile", model.EnvironmentProfileV1),
		compilerTextField("external_service_binding", model.ExternalServiceBinding),
		compilerValueField("determinism_profile", determinism),
		compilerValueField("confidentiality_established", canon.Bool(false)),
	)
	if err != nil {
		return nil, 0, refuse(CodeCompilerInvariant, "outer bundle assembly failed", err)
	}
	canonical, err := root.CanonicalChecked()
	if err != nil {
		return nil, 0, refuse(CodeCompilerInvariant, "outer bundle canonicalization failed", err)
	}
	return canonical, encodedContentBytes, nil
}

func determinismValue() (canon.Value, error) {
	value, err := compilerObject(
		compilerTextField("scope", model.DeterminismScopeV1),
		compilerValueField("emitter_introduces_time", canon.Bool(false)),
		compilerValueField("emitter_introduces_random_id", canon.Bool(false)),
		compilerValueField("emitter_introduces_absolute_path", canon.Bool(false)),
		compilerValueField("emitter_introduces_concrete_candidate_identity", canon.Bool(false)),
		compilerValueField("emitter_introduces_declared_secret_value", canon.Bool(false)),
		compilerValueField("emitter_introduces_host_runtime_fact", canon.Bool(false)),
		compilerValueField("emitter_introduces_execution_receipt", canon.Bool(false)),
	)
	if err != nil {
		return canon.Value{}, refuse(CodeCompilerInvariant, "determinism profile assembly failed", err)
	}
	return value, nil
}

func enforceFileCeilings(files []generatedFile, expectedCount int) error {
	if len(files) != expectedCount {
		return refuse(CodeCompilerInvariant, "compiler constructed an unexpected file count", nil)
	}
	aggregate := 0
	for _, file := range files {
		if len(file.content) == 0 || len(file.content) > model.MaxContractFileBytes {
			return refuse(CodeCompilerLimit, fmt.Sprintf("%s exceeds the per-file ceiling", file.path), nil)
		}
		aggregate += len(file.content)
		if aggregate > model.MaxAggregateRawFileBytes {
			return refuse(CodeCompilerLimit, "aggregate generated bytes exceed the ceiling", nil)
		}
	}
	return nil
}

func finalizeFileDigests(files []generatedFile) error {
	for index := range files {
		digest, err := parseRawDigest(rawSHA256(files[index].content))
		if err != nil {
			return refuse(CodeCompilerInvariant, "raw file digest is invalid", err)
		}
		files[index].digest = digest
	}
	return nil
}

func parseExactValue(exact []byte, label string) (canon.Value, error) {
	value, err := canon.Parse(exact)
	if err != nil {
		return canon.Value{}, refuse(CodeInvalidCompilerInput, label+" is not strict JSON", err)
	}
	canonical, err := value.CanonicalChecked()
	if err != nil || string(canonical) != string(exact) {
		return canon.Value{}, refuse(CodeInvalidCompilerInput, label+" is not exact canonical JSON", err)
	}
	return value, nil
}

// jsonFileEnvelope renders exact canonical JSON plus one terminal LF.
func jsonFileEnvelope(value canon.Value, label string) ([]byte, error) {
	canonical, err := value.CanonicalChecked()
	if err != nil || len(canonical) == 0 {
		return nil, refuse(CodeCompilerInvariant, label+" canonical body failed", err)
	}
	return append(canonical, '\n'), nil
}

type compilerObjectField struct {
	name   string
	text   string
	value  canon.Value
	isText bool
}

func compilerTextField(name, value string) compilerObjectField {
	return compilerObjectField{name: name, text: value, isText: true}
}

func compilerValueField(name string, value canon.Value) compilerObjectField {
	return compilerObjectField{name: name, value: value}
}

func compilerObject(fields ...compilerObjectField) (canon.Value, error) {
	members := make([]canon.Member, len(fields))
	for index, field := range fields {
		value := field.value
		if field.isText {
			var err error
			value, err = canon.String(field.text)
			if err != nil {
				return canon.Value{}, err
			}
		}
		members[index] = canon.Member{Name: field.name, Value: value}
	}
	return canon.Object(members...)
}

func rawSHA256(exact []byte) string {
	digest := sha256.Sum256(exact)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func parseRawDigest(value string) (domain.Digest, error) {
	return domain.ParseDigest(value)
}

# Countershape

Countershape is a Darwin reference implementation for studying disagreements between trusted local implementations, reducing an exact witness, recording a human ruling, and preserving the distinction between observation, projection, decision, contract, execution, and receipt authority.

It is a research instrument, not a sandbox, CI service, agent runtime, or security product. Candidates run with the current user's permissions and host network access.

## Quick start

Requirements for the tested profile: macOS arm64, Go 1.26.5, Node 25.2.1, Apple Git 2.50.1, and the checked-in dependency tree.

```sh
make release
./dist/release/countershape-darwin-arm64/countershape --help
./dist/release/countershape-darwin-arm64/countershape studio
./dist/release/countershape-darwin-arm64/countershape export \
  --output "$PWD/countershape-report.html" \
  --acknowledge-confidentiality-not-established
```

The report is a local, inert, default-minimized presentation. **CONFIDENTIALITY NOT ESTABLISHED.** It is not safe, sanitized, anonymous, or shareable. Raw export additionally requires both `--include-raw` and `--i-understand-raw-may-contain-secrets`.

The single binary embeds the production studio assets and exposes help/version, studio, export, and the frozen HTTP/CLI study routes. Node is still the declared fixture/generated-contract runtime for the exercised studies; it is not hidden inside the Go binary.

## What the boundaries mean

- `TreeIdentity` identifies exact admitted source bytes. It is not a branch name or trust claim.
- Materialized files are a fresh execution root reconstructed from pinned objects.
- A comparison envelope states which dimensions are exact, recorded-only, or outside the claim.
- Captured evidence is private physical process/filesystem evidence.
- A projection is a typed, bounded view of captured evidence; it is not the capture itself.
- A ruling is a caller-attributed DecisionRecord over one exact Choicepoint.
- An emitted bundle is a standalone six-file contract residue, not an execution target.
- A `ContractExecutionTarget` is a fresh capability-only target; `FinalizedContractRun` is its exact target-bound run; `ContractExecution` is the matching classification.
- A didrun receipt records command evidence. Its grade is opaque and never upgraded into conformance.

Wake may supply refs or provenance, but Countershape is not a Wake UI, replay system, agent runtime, or fleet manager.

## Development

```sh
make test
make web
make release
make smoke
```

See [THREAT_MODEL.md](docs/THREAT_MODEL.md), [LIMITATIONS.md](docs/LIMITATIONS.md), [PERFORMANCE.md](docs/PERFORMANCE.md), [SECURITY.md](SECURITY.md), and [CONTRIBUTING.md](CONTRIBUTING.md).

Licensed under Apache-2.0. No trademark, domain, or package-name clearance is claimed.

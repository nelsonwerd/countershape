# Performance evidence

## Exact measured host

- Mac Studio `Mac14,13`, Apple M2 Max, 12 cores (8 performance, 4 efficiency), 32 GB RAM
- macOS 26.5.2 (25F84), arm64
- Go 1.26.5 darwin/arm64
- Node 25.2.1
- Apple Git 2.50.1 (Apple Git-155)
- Apple clang 17.0.0

All results are local snapshots for checked-in dependency-free fixtures. They are not imported-repository extrapolations.

## Budgets and repetitions

The U7 HTTP and CLI harnesses each execute three clean-root runs with 300 declared trials per domain and a 900,000 ms subject-process wall budget per domain. Fresh evidence, targets, runs, and classifications are required; caches and reused prepared roots are forbidden. The product target is under 15 minutes per study run. Exact U9 measurements are recorded in `evidence/performance/u9-measurements.json` and the hardening ledger.

Build/report/package timings use three repetitions unless the evidence file states otherwise. Wall time includes process startup and filesystem sync. CPU and memory observations are descriptive host snapshots, not enforced envelopes.

| Operation | Run 1 | Run 2 | Run 3 | Observed boundary |
| --- | ---: | ---: | ---: | --- |
| Native release-shaped build | 4.66 s | 3.55 s | 3.83 s | web production build plus Darwin arm64 Go binary and manifest |
| Default report export | 3.83 s | 3.73 s | 3.67 s | exclusive mode-0600 publication; identical SHA-256 `9f1f031de797f4da00bf8a487a48dfc951206a900ec975b115136f40e49645c0` |
| Clean-package smoke | 7.05 s | 7.05 s | 7.22 s | copied binary in a fresh private Git root through studio shutdown |

These timings are didrun event 1 of the U9 development ledger. They are exact local observations, not a signed release benchmark or another machine's expected latency.

## Fresh reference-study reproduction

Didrun event 3 ran the unchanged U7D harness v2 from a private archive of staged candidate tree `332a6d720845ded9e2ec763d9c9fca2de99a0c2f`. Each domain used three distinct clean Git fixtures and exactly 100 trials per run (300 per domain). The complete wrapper exited zero.

| Domain | Run 1 | Run 2 | Run 3 | Aggregate / budget | Peak RSS |
| --- | ---: | ---: | ---: | ---: | ---: |
| HTTP | 195.28 s | 177.45 s | 179.60 s | 552.33 / 900.00 s | 297,861,120 bytes |
| CLI | 259.65 s | 268.93 s | 276.84 s | 805.42 / 900.00 s | 267,976,704 bytes |

Each run includes fixture preparation, a fresh native compile, and product execution. The slowest complete run was 276.84 seconds, below the 15-minute per-study-run target. The harness required deterministic source-spec, Plan, ruling-projection, DecisionRecord-projection, and ContractBundle-projection digests to repeat within each domain while every world-instance, attempt, measurement, capture, confirmation, target, finalized-run, classification, fixture invocation, fixture Git directory, and evidence-manifest authority remained fresh and nonaliased. Its exact verdict remains `LOCAL_TWO_DOMAIN_PROCESS_ARTIFACT_GREEN_CLI_OFFICIAL_EXECUTION_HTTP_DIRECT_PROCESS_ONLY_FULL_STUDY_RESOURCE_UNRECEIPTED`; its honest fallback remains `ONE_DOMAIN_OFFICIAL_CONTRACT_EXECUTION_PLUS_HTTP_DIRECT_PROCESS_OBSERVATION_ONLY`. The harness observes product process/artifact facts and does not independently recompute product semantics or total harness resources.

An earlier unreceipted full-Go development attempt used the ambient macOS `/var/folders/...` temporary alias and Go's default ten-minute package timeout. It failed at the repository's deliberate all-component no-follow boundary and one long contract test's default alarm. Focused reruns using canonical `/private/tmp`, `GOMAXPROCS=2`, serial package execution, and a 20-minute alarm passed. The final matrix uses that same frozen profile; the ambient-path failure is not represented as a product-semantic result.

The final ordinary full-Go matrix completed in 2,484.886 seconds with serial package execution, `GOMAXPROCS=2`, test parallelism 2, fresh private caches, canonical `/private/tmp`, and a 20-minute per-package alarm. A subsequent universal `-race ./...` experiment ran for 8,998.027 seconds and emitted no race report, but three long existing physical suites exhausted their terminal or 30-minute package budgets under instrumentation. It is a retained failed event, not evidence of full-repository race freedom. A bounded race matrix covering every new U9 Go surface plus the three deterministic fixture-repair packages completed in 407.021 seconds. These observations establish neither an all-package race pass nor production resource capacity.

The local dependency/package gate found and repaired one exact peer mismatch: `@vitejs/plugin-react 5.1.4` did not declare Vite 8 support. Exact offline-cached `6.0.2` does; the complete web matrix passed and the production asset bytes remained unchanged. The final local inventory then completed in 22.850 seconds, including Go modules, the clean npm graph, declared package licenses, native release build and smoke, private/development-marker scans, and a Linux/amd64 compile-only artifact. It is not an SBOM, vulnerability scan, legal review, signature, provenance attestation, or Linux runtime result.

Two clean-root release builds were compared. The license and deterministic report bytes matched. The native cgo-linked Darwin binary and its digest-bearing manifest did not match because Apple's linker emitted a fresh Mach-O `LC_UUID`. A no-UUID experiment produced byte-identical binaries, but macOS refused to launch them; it was rejected. Countershape therefore makes no byte-reproducible-binary claim.

## Claim ceiling

Passing the timing target establishes only that these exact local fixtures completed within the declared budget on this host. It does not establish throughput, latency, scalability, production capacity, or another repository/platform's behavior.

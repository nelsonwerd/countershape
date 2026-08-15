# U9 hardening ledger

Status: `EVIDENCE_CLOSURE_RETRY_IN_PROGRESS`

## Pass 1 — export confidentiality and injection

- Added a package-issued immutable server snapshot over the exact resolved seed Choicepoint, reveal, and DecisionRecord.
- Added deterministic, script-free, no-network HTML with responsive/print/forced-colors CSS.
- Default output omits captured bodies, environment values, paths, process streams, credentials, candidate identities, and producer/model provenance; selected values receive a versioned known-pattern redaction pass.
- Raw export requires two explicit danger flags.
- Output uses an exclusive same-directory link commit, mode 0600, and refuses overwrite or a symlink final parent.
- Direct `file://` inspection was refused by the in-app browser URL policy. No policy bypass was attempted. Browser visual judgment therefore remains `UNRECEIPTED` until an actually supported product route is inspected.

## Pass 2 — package and documentation

- Added verified exact embedded-asset closure and single-binary help/version/studio/export/study dispatch.
- Added a Darwin/arm64 release builder and clean private-root package smoke.
- Added Apache-2.0 licensing and public boundary documentation.
- Two clean-root builds produced byte-identical license and report objects but different Darwin binaries/manifests because the native external linker emitted a fresh Mach-O UUID. Removing `LC_UUID` made the binary byte-identical but macOS refused to launch it, so that unsafe experiment was rejected. Binary reproducibility is not claimed.

## Pass 3 — adversarial, reproduction, and performance

- A three-repetition didrun-wrapped local measurement recorded release builds at 4.66/3.55/3.83 seconds, default exports at 3.83/3.73/3.67 seconds, and clean-package smoke at 7.05/7.05/7.22 seconds. All three default reports were byte-identical at SHA-256 `9f1f031de797f4da00bf8a487a48dfc951206a900ec975b115136f40e49645c0`.
- The complete web typecheck, lint, unit, production build, real-binary Playwright, security, responsive, and accessibility path completed under didrun.
- Didrun event 3 completed the exact U7D harness from private staged-tree archive `332a6d720845ded9e2ec763d9c9fca2de99a0c2f`: three fresh HTTP runs observed 195.28/177.45/179.60 seconds (552.33 of 900 seconds aggregate), and three fresh CLI runs observed 259.65/268.93/276.84 seconds (805.42 of 900 seconds aggregate). Both domains retained 300 exact trials, stable deterministic projections, and globally distinct fresh physical authorities. The exact bounded verdict/fallback remain unchanged and do not become independent product-semantic or full-harness-resource authority.
- An early direct `go test -count=1 ./...` development attempt used ambient `/var/folders/...` temporary roots and Go's default ten-minute alarm. It failed because the tested no-follow filesystem boundary deliberately refuses `/var`'s symlinked ancestor and because one long `internal/contractexec` test reached the default alarm. Exact focused reruns with the frozen canonical `/private/tmp` profile and a 20-minute package alarm passed. This is a development-profile failure, not a semantic receipt, and the final full command must use the canonical profile through didrun.
- Didrun events 4 and 5 were permanent failed full-Go attempts. Both exposed the same Darwin fixture defect at different layers: a setgid hostile applied under `/private/tmp` inherited group `wheel`, so Darwin silently cleared the bit before the validator could observe it. The testkit CLI/HTTP fixtures and the contract-execution scope fixture now rebind only the hostile file's group to the current process group before applying setgid. Focused reruns passed; event 6 then completed `go test -mod=readonly -buildvcs=false -p=1 -parallel=2 -count=1 -timeout=20m ./...` on candidate tree `3a4655a9d032eab8cc1846ed2979f0b9f047fec9` in 2,484.886 seconds.
- Didrun event 7 attempted the universal `-race ./...` matrix and terminated nonzero after 8,998.027 seconds. No race report was emitted. Three already-existing physical suites instead exhausted their execution/package budgets under race instrumentation: CLI contract execution returned terminal target/finalized-run deadline failures; HTTP contract execution reached its 30-minute package alarm; and the CLI precedence compilation barrier did not complete inside its bound. This event is retained as failed evidence and does not establish full-repository race freedom. Event 8 then ran the bounded U9/repaired-surface race roster—`cmd/countershape`, `internal/assets`, `internal/report`, `internal/server`, `internal/contractexec/scope`, `testkit/export`, `testkit/package`, and `testkit/reference`—and passed in 407.021 seconds on the same tree.
- The first dependency inventory (event 9) correctly refused the installed graph because `@vitejs/plugin-react 5.1.4` declared Vite support only through major 7 while the locked Vite was 8.2.1. The locally cached exact `@vitejs/plugin-react 6.0.2` supports Vite 8; the package and lock were updated offline, the production asset bytes remained unchanged, and event 10 reran typecheck, lint, unit, build, and real-binary E2E successfully on tree `450a5acdad1b...`.
- Event 11's additional binary strings scan rejected its own defensive literal `sourceMappingURL=`. The validator embeds that string specifically to refuse source-map markers, so raw strings cannot distinguish the guard from a packaged map. Event 12 retained binary scans for checkout/private/didrun/node_modules/Vite-development markers and separately scanned the exact asset files for map files and map/development references. It passed the Go module inventory, clean npm graph and local license inventory, native build, package smoke, native metadata scan, asset scan, and Linux/amd64 compilation-only check in 22.850 seconds. It is not an SBOM, vulnerability scan, signature, provenance attestation, Linux runtime receipt, or independent license review.
- The final staged integrity/credential gate first failed at event 13 because the shell expanded an `awk` `$1` under `set -u`; event 14 replaced that diagnostic-only parser with `cut` and passed on exact 31-path tree `d8b69a1bb708c65a9196e79534babfc5226504e1`. Event 15 then invoked the repository cumulative verifier. Row 1 passed, but row 2 refused the inherited sealed-parent HANDOFF capsule at `VERIFY_U7_PHASE_UNSUPPORTED: U7R`; no later row ran. U9 does not own the U7 architecture/control plane, so it neither weakens nor changes that inherited gate. Cumulative verification remains `UNRECEIPTED`; the complete ordinary Go, bounded race, web/E2E, fresh-study, package, and inventory evidence above remain separately scoped.
- The in-app Browser rejected direct `file://` navigation before it could load the generated report. No alternate browser, raw DevTools connection, or local-server workaround was used to bypass that policy. The report's structural/injection/CSP/responsive/print checks are automated; manual local report visual inspection remains `UNRECEIPTED`.

The final restaged integrity gate, didrun claims, commit, seal, and explicit-commit strict report remain pending. The inherited-U7R cumulative verifier and unsupported inventory/security tools remain `UNRECEIPTED`.

## First permanent failed postcommit evidence attempt

Product commit `f49f4bf4ee28c27511809701b64e86a8c46af07c`, tree `b57ac80e3f91f28482f7633a20092b6008563807`, direct parent `ed58cb6dd6f50fd1f2a889b2e3d6b728a43cd3b0`, and subject `feat: add U9 export and packaging hardening` preserves the complete U9 implementation. Its note blob is `23a9cc50d7821852ec32164cb27cdb43e45ae8d6` (body SHA-256 `514bb0e6b29f5cdc4e0b0384a4a6b3c500a9a12da3c64c960e3c3f9790217a05`) and discloses `secrets_override: true` after the separately claimed zero-finding structured scan.

That first seal was not strict-clean: six claims were declared against earlier candidate trees with pathspecs that omitted later documentation/repair files, so strict verification exited `1` with `1/7` `TREE-EXACT` and six `STALE` grades. The exact HTML is `.countershape/evidence/u9-final-f49f4bf4ee28.html` (9,452 bytes, SHA-256 `5376a8f467455ab9aa09c4b7821e804ab30bbdf0f01d8abffad8fa51fc171004`); the private failed ledger is preserved at `.didrun-history/u9-postcommit-failed-f49f4bf4ee28/.didrun`. Neither `STALE` grade is relabeled or transferred.

The repair is evidence-only: this child candidate changes only the tracked U9 handoff/ledger chronology, then reruns the seven successful U9 gates on one stable staged tree before any claim or commit. It changes no product, dependency, test, package, fixture, performance result, or limitation. U9 is not complete until that child seals with every declared claim `TREE-EXACT` and explicit-commit strict verification exits zero.

## Cross-unit edits

- `internal/server/export.go` is the only U8 production extension. It exposes immutable defensive bytes from the existing package-owned seed state and does not change studio behavior or semantic authority. Server and full suites must be rerun.
- `cmd/countershape/commands_other.go` is a compile-only non-Darwin refusal required by the packaging invariant. It exposes only inert help/version and carries no Linux runtime authority.
- `internal/contractexec/scope/probe_darwin_test.go`, `testkit/reference/cli_test.go`, and `testkit/reference/http_test.go` make the existing setgid hostiles deterministic under canonical `/private/tmp`; production behavior is unchanged.
- `web/package.json` and `web/package-lock.json` repair the pre-existing Vite 8 peer declaration with exact offline-cached `@vitejs/plugin-react 6.0.2`. The complete web/E2E suite was rerun and the checked-in production distribution remained byte-identical.

## Claim ceiling

This ledger is development chronology, not a receipt. The final U9 didrun note/strict report will be the command-evidence boundary; manual design/security/legal judgments remain outside command grades.

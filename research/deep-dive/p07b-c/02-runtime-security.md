# Lane 2 — runtime, process, and scoped standalone evidence

## Verdict

P07B-C needs three new physical capabilities before it may claim a standalone run:

- a direct single-target Git materialization issued from live `InspectedTree` authority;
- a Node-specific admitted-runtime capability retained through the spawn edge;
- an owner-produced closed-run witness that distinguishes complete, partial, and violated standalone checks.

The authoritative path must execute the recovered source through a Go-owned process edge. Running `node --test` and parsing TAP remains a useful black-box parity/operator check, not target/run authority.

The final admission correction also requires one measured Darwin boot-session identity and a store-private interlock. The higher contract runner—not low-level process mechanics—consumes official target/admission authority. Missing spawn observation or incomplete terminal closure holds that interlock on the same boot; a fresh target cannot bypass it.

## Runtime and process boundary

Reuse the proven Darwin mechanics below a new narrow `internal/processowner` layer: direct `exec.Cmd`, no shell, new process group, ownership check, bounded capture, TERM/grace/KILL, drains, direct-child wait, and final group probe. Do not route C through `WorldPlan`, `BoundCandidate`, or `WorldInstance`; those belong to historical 2–4-candidate comparison.

`AdmittedNodeRuntime` should privately retain:

- canonical absolute symlink-resolved executable path;
- device/inode/mode/size and executable-bytes digest;
- exact owned probe-program bytes/digest;
- measured `process.execPath`, version/major, platform, and architecture;
- pre-spawn and post-run revalidation facts.

This narrows substitution risk but does not establish descriptor-bound execution, immunity to coordinated same-user replacement, containment, or confidentiality.

## Scoped standalone meaning

The v1 claim is limited to the prepared target inventory and admitted child bindings. It requires typed observations for:

1. target inventory;
2. child executable/environment bindings;
3. import resolution;
4. service bindings;
5. named parent-secret sentinel inheritance.

It does not establish host-wide absence, network denial, registry denial, listener ownership, hostile-code containment, or resistance to session escape.

## Required lifecycle rule

Only `process clean + projected tuple + standalone COMPLETE without violation` is eligible. Early start/cancellation errors cannot manufacture import, child, or sentinel absence. A teardown/orphan failure remains ineligible even if useful bytes were captured.

## Key evidence

- Existing direct Git inspection: `internal/gitobj/inspect.go:438-502`.
- Comparison-only materializer/revalidator: `internal/gitobj/materialize.go:410-430`, `internal/gitobj/validate.go:37-47`.
- Generic runtime admission: `internal/world/tools.go:47-66,104-180`.
- Darwin process ownership: `internal/world/process_darwin.go:97-113,221-244,474-529,708-788`.
- TAP must remain inert: `docs/prompts/P07-U6-STANDALONE-CONTRACT.md:173-204`.
- Scoped nonclaims: `docs/THREAT_MODEL.md:239-247`.

## Confidence

**7.8/10.** Ten of thirteen conclusions are directly grounded in current source; the package extraction, runtime capability shape, and detector implementation are engineering bets.

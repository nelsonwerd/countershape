# Limitations and fallback products

- Darwin/arm64 is the only natively exercised runtime profile. Linux is compilation-only and `UNRECEIPTED` as runtime evidence.
- Node 25 is exercised as fixture/generated-contract runtime. Countershape does not embed Node.
- Candidates are trusted local code with full user permissions and host network access.
- Process-group cleanup does not contain deliberately detached descendants.
- Loopback authentication excludes ordinary unauthenticated requests, not same-user malware or hostile browser extensions.
- Default export is fixture-profile minimization, not arbitrary-secret detection. `CONFIDENTIALITY NOT ESTABLISHED`.
- Raw export is dangerous and local-only.
- No Docker path, cloud service, remote execution, fleet management, agent runtime, replay system, hostile isolation, universal repository support, signed binary, SBOM, provenance attestation, independent security review, or cross-platform runtime matrix.
- Observed finite stability is not determinism. Deterministic wrapper equality does not transfer raw physical lineage.
- Unknown receipt grades remain verbatim; absent evidence is `UNRECEIPTED`.
- The local Collector's artifact-byte closure does not establish harness process topology, stdout, environment, resource, or receipt authority.
- External visual criticism was unavailable during P10; its local visual hardening fallback remains the honest result.

If a higher boundary fails, the permitted lower products are: no export; checkout-required developer build; CLI-only presentation; DecisionRecord-only residue; one-domain instrument; or observation/approval prototype. The cut must be named rather than silently inheriting the larger claim.

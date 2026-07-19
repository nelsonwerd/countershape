# C3 runtime and host-epoch audit

## Finding

The pre-C3 target contract named `DARWIN_KERN_BOOTTIME_V1` as if Darwin's `kern.boottime` were a stable boot-session identity. That is false. Apple XNU's calendar-set path explicitly adjusts `clock_boottime` by the wall-clock delta, and the `kern.boottime` sysctl returns that mutable value. A time correction during one boot can therefore change the serialized identity and incorrectly resemble a reboot.

Apple XNU separately generates a boot-session UUID during root-domain boot initialization and exposes `kern.bootsessionuuid` as a read-only sysctl. The running Darwin host also exposes that key. This is the narrower source for the interlock/reset fence.

Primary sources:

- [XNU clock calendar adjustment](https://github.com/apple-oss-distributions/xnu/blob/main/osfmk/kern/clock.c#L720-L838)
- [XNU boot-session UUID generation](https://github.com/apple-oss-distributions/xnu/blob/main/iokit/Kernel/IOPMrootDomain.cpp#L4159-L4175)
- [XNU read-only boot-session sysctl](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/kern/kern_sysctl.c#L2921-L2949)

## Ruling

The exact v1 profile is `DARWIN_KERN_BOOTSESSIONUUID_V1`. It means that the private host-epoch owner reads only `kern.bootsessionuuid`, validates each sample against the closed 36-character hyphenated UUID shape, rejects the all-zero value, lowercases each sample, compares two consecutive normalized values, and hashes one typed `DarwinBootSessionIdentity` body. The canonical target stores only that digest. It never stores or accepts a caller-authored UUID, timestamp, boot digest, or success Boolean.

The old `DARWIN_KERN_BOOTTIME_V1` bytes are rejected. The implementation must not silently read `kern.bootsessionuuid` under the old label, fall back to `kern.boottime`, spawn `/usr/sbin/sysctl`, consult PATH, or infer boot identity from wall time, uptime, PID, a file, or process-local state. Darwin production measurement will use a bounded fixed-name system-call edge. Other platforms refuse explicitly.

## Why this is a separate boundary

C1 sealed inert target bytes before any production official target existed. C2 persisted only test-issued generic semantic objects and digests; it never measured a boot session or issued `OfficialTarget`. Changing the frozen profile before C3 is therefore a pre-authority semantic correction, not a reinterpretation of a live production record. The historical C1/C2 commits and their receipt claims remain truthful evidence about those exact historical trees.

C3P changes the model constant, target schema, the three joined checked examples, planning validator, controlling documentation, exact unit rosters, and receipt machinery together. It is a full `SOURCE_FULL` boundary. C3 remains blocked until C3P and its separate C3PB receipt reconciliation are both committed, sealed, note-present, and strict-clean.

## C3 design corrections carried forward

The audit found two additional scope defects that the C3 brief must absorb:

1. C3's Git path must materialize and reopen directly from opaque `gitobj.InspectedTree`; manufacturing a one-member `WorldPlan` or `BoundCandidate` is forbidden.
2. C3 necessarily evolves the compiler-frozen store surface with a narrow inert attempt/target bridge, so `internal/store/public_api_test.go` belongs to the exact C3 roster. The bridge grants no `OfficialTarget`, permit, process, discovery, or head authority.

C3's former directory prefixes are replaced by an exact forty-path roster. This makes the final source-integrity and credential gates admissible instead of relying on a weaker lexical-prefix check. A separate C3B receipt unit is declared before C4.

## Honest ceiling

The corrected UUID profile establishes only that Darwin currently reports the same or a different boot-session UUID. A different value does not independently prove a physical reboot, child absence, VM non-rollback, kernel trust, UUID collision impossibility, or safe reset. C4 still requires explicit operator action for ambiguous reset, repeats the live measurement immediately before admission, and makes no same-boot survivor-absence claim.

Node admission likewise proves only the exact locally measured path/file/probe tuple at its checkpoints. It does not prove publisher authenticity, dynamic-library closure, descriptor-bound later execution, hostile same-user isolation, containment, or restart-time inode continuity.

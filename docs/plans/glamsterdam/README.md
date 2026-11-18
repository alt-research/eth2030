# Glamsterdam completion program

Status: implementation started in an isolated current-mainline worktree; revalidation is in progress.

The original audit baseline below is historical. Implementation starts from `origin/master` at `38ea9259` on 2026-11-18. That mainline already contains Amsterdam header/slot repairs and canonical persisted Engine bodies; these must be revalidated rather than reimplemented. See [implementation tracking](IMPLEMENTATION.md).

This is the consolidated entry point for finishing Glamsterdam in ETH2030. It replaces historical percentage-based progress claims with explicit requirements, development tasks and acceptance gates. Project documentation is in English; protocol changes and a new devnet run are outside this documentation-only delivery.

## Read in this order

1. [Scope and gap matrix](GAP_MATRIX.md): the 18 scheduled core EIPs, their dependencies, supporting network protocols and production consumers.
2. [Development plan](DEVELOPMENT_PLAN.md): task IDs, dependencies, implementation boundaries, verification, sequencing and effort assumptions.
3. [Upstream decision register](UPSTREAM_DECISIONS.md): what is settled, what was proposed, and which decisions affect implementation or release acceptance.

## Baseline and authority

The implementation baseline is `feat/track2-hegota` at `c9e0597b`, with protocol code unchanged from `83e3860d`. Evidence was collected on 2026-11-18; this planning pass organizes that snapshot and does not claim another live upstream refresh.

| Source | Revision or role |
| --- | --- |
| Upstream EIPs | `3db38a3725105cb67b8804ac3e147967113bc3ca` |
| Upstream consensus-specs observed head | `2ea9d96a420927eb1a35d7c9399c8b3691c69f6a` |
| Local EIPs reference | `49beda901dcf3747ed4b91f853a333595fb2cbc4`; older than the audited rules |
| Local consensus-specs reference | `14e6ce5a5334910b0d0575d782e468305ad92373` |
| Local Lighthouse inspected source | `f85fa05b6`; source inspection only, not a fresh CL qualification |
| [Initial audit](../../audits/glamsterdam-2026-11-18/README.md) | Initial per-EIP comparison and native gas observations; read its correction notice |
| [Discussion follow-up](../../audits/glamsterdam-upstream-2026-11-18/README.md) | Corrected BAL/slot findings, upstream discussions and consumer gaps |
| [Discussion index](../../audits/glamsterdam-upstream-2026-11-18/discussion-index.json) | 27 PRs, seven issues, six comment threads, observed states and revisions |

Normative requirements come from the selected EIP, Engine, devp2p and consensus specifications together with recorded fork decisions. EIP descriptions, meeting agendas, source comments, PR approval and issue closure alone do not establish adopted consensus behavior. Before implementation, GL-001 records the complete compatible specification and fixture set. References are read-only; refresh inputs in an isolated checkout or versioned fixture workspace rather than editing reference files in place.

## Completion target

The target is a native execution client that builds, imports, persists, replays and serves Glamsterdam blocks correctly, interoperating with an explicitly qualified CL and external builder. Acceptance includes canonical bytes and hashes, gas/state/receipt agreement, Engine/RPC/P2P consumers, fork activation and recovery. It is not satisfied by existing helper functions, a green geth-backed vector run or sustained empty/transfer-only block production.

The baseline core classification is **10 confirmed mismatches, two missing EL paths, three CL acceptance boundaries and three foundations**. This is an evidence classification, not a percentage or a count of equal-sized tasks. The gap matrix includes 7495/7916 dependencies and five network EIPs separately.

Core consensus acceptance, the operational client release and the broader networking deliverable have separate gates. The default program includes all five listed network EIPs in its final target. A narrower intermediate release must name unsupported protocol versions and leave those tasks open; it must not claim complete program acceptance.

FOCIL, frame transactions, future SSZ execution-block activation, Quick Slots, PQ migration and other later-fork work do not count toward Glamsterdam completion. Existing implementations must continue to pass affected regressions, but this program does not reopen every later-fork design. Required consensus SSZ dependencies remain in scope.

## Start now

The first implementation batch is GL-001/002/003, followed by GL-010/011: establish the spec/test baseline, add failing native regression assertions, repair the stale tagged BAL expectation, and fix canonical header commitments and slot propagation. Independent consumer work can follow the task dependencies while upstream gas and ePBS choices remain unresolved.

The plan uses existing fork scheduling, without adding environment variables or feature toggles for accepted functionality. A fork activates the agreed feature set together on the test/release schedule. Do not schedule an unannounced public-network fork or bundle later-fork experimental behavior into that feature set.

Track delivery in [CHECK.md](../../../CHECK.md) and [STATUS.md](../../../STATUS.md), using the GL task IDs. Every completed task needs a scoped commit and an evidence record; no implementation task is marked complete by creating this plan.

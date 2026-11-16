# proofs/prover — STARK prover and distributed prover assignment

## Overview

This package provides two complementary systems for the mandatory 3-of-5 proof
requirement (K+ roadmap). The first is a full STARK proof system over the
Goldilocks field (p = 2^64 - 2^32 + 1) with FRI (Fast Reed-Solomon IOP) folding,
Merkle commitments over execution traces, and Fiat-Shamir challenges binding
public inputs to the proof. The second is a distributed prover pool with
reputation-based selection, geographic diversity, and capacity tracking.

The STARK prover compiles algebraic constraints over multi-column execution
traces, commits the trace and constraint evaluations to Merkle trees, generates
FRI layer commitments with deduplication-safe query selection, and verifies query
responses with Merkle authentication paths.

## Functionality

**STARK types and functions**

- `FieldElement` — Goldilocks field element with `NewFieldElement(v int64)`
- `STARKConstraint` — degree and coefficient vector for an algebraic constraint
- `STARKProofData` — full proof: `TraceCommitment`, `FRICommitments`,
  `QueryResponses`, `ConstraintEvalCommitment`, `BlowupFactor`, `NumQueries`
- `STARKProver` — created with `NewSTARKProver()` or `NewSTARKProverWithParams(...)`
- `GenerateSTARKProof(trace, constraints) (*STARKProofData, error)`
- `VerifySTARKProof(proof, publicInputs) (bool, error)`
- `(*STARKProofData).ProofSize() int`

**Validation-frame proof boundary (`validation_frame_circuit.go`)**

- `STARKValidationFrameProver` returns `ErrFrameProofUnavailable`: the prior trace did not prove EVM execution or `APPROVE`.
- `StubValidationFrameProver` is a test double and does not prove validity. The VM requires a separately configured block-bound provider and does not accept this stub for block proof verification.

**Prover pool types and functions**

- `ProverCandidate` — ID, region, capacity, in-flight count, reputation score
- `AssignmentResult` — selected prover IDs, scores, and regions per block
- `ReputationScorer` — `RecordSuccess`, `RecordFailure`, `Decay(factor)`
- `ProverPool` — created with `NewProverPool(minProvers int)`
- `RegisterProver(id, capacity)` / `RegisterProverWithRegion(id, capacity, region)`
- `AssignProvers(blockNum, count) (*AssignmentResult, error)`
- `RecordSuccess(proverID, blockNum)` / `RecordFailure(proverID, blockNum)`
- `DecayAllReputations(factor float64) error`

**Parent package:** [proofs](../)

# crypto/pqc — Post-quantum cryptography (Dilithium3, Falcon512, SPHINCS+, ML-DSA-65)

[← crypto](../README.md)

## Overview

The `PQSigner` API wraps CIRCL ML-DSA and SLH-DSA, and go-fn-dsa FN-DSA-512. CIRCL also supplies ML-KEM-768. The registry exposes the selected algorithms, and hybrid signing combines ECDSA with a PQ signer.

## Functionality

**Core types**
- `PQAlgorithm` — enum: `DILITHIUM3`, `FALCON512`, `SPHINCSSHA256`
- `PQKeyPair` — `Algorithm`, `PublicKey []byte`, `SecretKey []byte`
- `PQSignature` — `Algorithm`, `PublicKey []byte`, `Signature []byte`
- `PQSigner` interface — `GenerateKey() (*PQKeyPair, error)`, `Sign(sk, msg []byte) ([]byte, error)`, `Verify(pk, msg, sig []byte) bool`

**Signers**
- `GetSigner(alg PQAlgorithm) PQSigner` — returns the configured ML-DSA-65, FN-DSA-512 or SLH-DSA-SHA2-128f wrapper
- `MLDSASigner` — FIPS 204 ML-DSA-65 via cloudflare/circl
- `SPHINCSSigner` — stateless hash-based signatures
- `HybridSigner` — ECDSA + PQ combined signature

**Supporting infrastructure**
- `PQAlgorithmRegistry` — project-local experimental algorithm capability registry
- `PQTxSigner` — signs Ethereum transactions with PQ keys
- `LatticeCommitment` / `BlobCommitment` — MLWE lattice-based commitments for blobs
- `HybridThresholdScheme` — combines threshold crypto with PQ
- `CustodyReplacer` / `CustodyReplacerV2` — upgrades custody proofs to PQ
- `BatchBlobVerifier` — verifies multiple blob commitments in batch
- `SignaturePipeline` — end-to-end PQ signing pipeline
- `PubKeyRegistry` — maps lean-sig public keys to PQ counterparts

The size constants derive from the public providers. Existing historical identifiers select ML-DSA-65, the prospective go-fn-dsa FN-DSA-512 format, and SLH-DSA-SHA2-128f; they do not establish wire compatibility with round-3 Dilithium3, Falcon-512, or SPHINCS+. The SLH-DSA wrapper is checked against a public NIST ACVP deterministic-signature case. These identifiers are not the pinned EIP-7932 registry identifiers.

## Usage

```go
signer := pqc.GetSigner(pqc.DILITHIUM3)
kp, _ := signer.GenerateKey()
sig, _ := signer.Sign(kp.SecretKey, msg)
ok := signer.Verify(kp.PublicKey, msg, sig)
```

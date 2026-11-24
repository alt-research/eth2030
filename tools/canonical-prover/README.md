# Canonical RV32IM execution proofs

The native node exposes a versioned execution-proof service. A fixed RISC Zero
3.1.0 image executes the supplied RV32IM program and commits its program/input
SHA-256 digests, output, output digest, cycles, zero exit code, terminal PC,
consumed input, complete register/RAM state digest and initial resource/layout
configuration. The Go producer independently executes and compares the entire
statement before publishing a result. Verification checks the actual receipt;
it does not substitute replay, proof length or a hash-shaped fixture.

This proves execution of the selected program under the documented machine ABI.
It does not assert that an arbitrary program implements Ethereum's STF. A future
canonical Ethereum STF program or mandatory block-proof fork requires its own
adopted program identity and conformance evidence. No public fork parameter or
existing Engine contract is changed here.

## Build the complete native bundle

Qualified platform: Linux x86_64, Docker with BuildKit, Go 1.26.11 and Rust.
Use the same `RISC0_HOME` when bootstrapping and building. The scripts default
to a dedicated ETH2030 toolchain cache and pin Rust 1.97.0 and SDK/r0vm 3.1.0.

```sh
tools/canonical-prover/bootstrap.sh
tools/canonical-prover/build.sh
tools/canonical-prover/target/node-bundle/eth2030 --datadir /path/to/node
```

The prover and `r0vm` ship beside the native executable. The `zkvm` HTTP/WS
namespace is included in native defaults. Explicit module allowlists still
apply. A missing prover returns an availability error; synthetic fallback is
never enabled. Build manifests retain revision, dirty state, image and binary
hashes. To build a runtime image:

```sh
docker build -f tools/canonical-prover/Dockerfile \
  -t eth2030-native:canonical-proof tools/canonical-prover/target/node-bundle
```

Guest builds use the SDK's container at the committed digest, under fixed `/src`
and target/registry roots. `image-id.txt` pins the resulting identity. A different
image is a build failure, not an automatically trusted new key. Both Cargo lock
files are retained. Changes to the ABI/image require a reviewed identity and new
independent proof fixtures. Development-mode fake receipts are always rejected.

## Native RPC

Submit one object to `zkvm_proveExecution`:

```json
{"program":"0x...","input":"0x2a","config":{"max_cycles":100,"memory_pages":4,"program_base":0,"entry_pc":0,"stack_base":2147483648,"input_base":0,"load_input":false}}
```

The result is a job `{id,status}`. `zkvm_getProof([id])` returns its current
status and a verified artifact on completion. `zkvm_cancelProof([id])` cancels
ongoing work. `zkvm_verifyExecution` accepts the same program/input/config plus
`proof` and returns `{valid,statement}`; changed inputs, outputs, state, resource
bounds, images or seals cannot be adopted as valid results.

One proof runs at a time, with no unbounded queue; two concurrent verifications
are allowed. Busy calls fail promptly. Eight terminal jobs are retained, and
each proof has a five-minute deadline. Shutdown cancels and joins the complete
Linux process group, including `r0vm`. Job IDs belong to one node lifetime;
exported proofs remain independently verifiable after restart. Programs, input
and output are bounded to 1 MiB each, guest cycles to 2^24 and RAM to 16,384 pages.

The v1 ABI uses little-endian RV32IM, sparse zero-initialized 4 KiB pages,
single-threaded FENCE and ECALL 0 (zero-code halt), 1 (output byte), 2 (input byte,
0xffffffff at EOF). High MMIO writes are no-ops without external callbacks.
Compressed, privileged, CSR and custom crypto-syscall extensions are not v1 ABI.

## Acceptance

The dedicated CI job installs the real components and requires positive proving;
it never treats an unavailable backend or a skipped test as acceptance.
Go-only jobs continue to exercise rejection and lifecycle behavior. With the
bundle directory on PATH:

```sh
cargo test --manifest-path tools/canonical-prover/Cargo.toml --release --locked
cd pkg
go test -p 1 -tags canonicalproof_integration ./zkvm/... ./node -timeout 90m
```

The retained real receipt is independently verified, its actual STARK seal is
mutated, and an SDK fake receipt is rejected even with developer mode enabled.
Differential vectors compare full Go/Rust machine states; native TCP acceptance
generates a real proof while ordinary RPC remains responsive, rejects altered
claims and verifies the exported artifact after a verifier restart.


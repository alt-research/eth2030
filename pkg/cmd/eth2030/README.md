# cmd/eth2030 — Native ETH2030 Ethereum client binary

## Overview

`cmd/eth2030` is the primary CLI binary for the ETH2030 Ethereum client. It wires together the native `node` package (custom EVM, Engine API, JSON-RPC, P2P) and exposes all runtime configuration through command-line flags. It is the production entry point for networks that use the ETH2030-native execution engine rather than the go-ethereum adapter.

Fork activations (Glamsterdam, Hogota, I+) are driven entirely by `ChainConfig` timestamps set in `node.New()` via flag overrides; no precompile injection step is needed at this layer (unlike `eth2030-geth`).

## Functionality

- `parseFlags(args []string) (node.Config, bool, int)` — builds a `node.Config` from CLI arguments; returns an exit code if `--version` or a parse error is encountered.
- `newFlagSet(cfg *node.Config) *flagSet` — registers all flags against the config struct.
- `applyZeroForkOverrides(args, cfg)` — handles the `--override.*=0` edge case where `flag.Uint64` cannot distinguish "not set" from "zero".
- `setupLogging(verbosity int)` — configures the `slog` default logger (verbosity 0–5).
- `logForkOverrides(cfg)` — logs active fork timestamp overrides at startup.

Custom `flagSet` helpers: `Uint64Var`, `Uint64PtrVar` (optional pointer-to-uint64 for fork overrides), `StringSliceVar` (comma-separated lists).

## Usage

```
eth2030 [flags]

Node flags:
  --datadir <path>           Data directory (default ~/.eth2030)
  --network <name>           mainnet | sepolia | holesky
  --port <n>                 P2P port (default 30303)
  --maxpeers <n>             Max P2P peers
  --syncmode full|snap
  --gcmode archive|full
  --verbosity 0-5

HTTP-RPC:
  --http.addr <addr>         (default 0.0.0.0)
  --http.port <n>            (default 8545)
  --http.vhosts <list>
  --http.corsdomain <list>
  --http.api <modules>

Engine API:
  --authrpc.addr <addr>      (default 0.0.0.0)
  --authrpc.port <n>         (default 8551)
  --authrpc.vhosts <list>
  --authrpc.jwtsecret <path>

WebSocket:
  --ws  --ws.addr  --ws.port  --ws.api  --ws.origins

P2P:
  --bootnodes <enodes>
  --discovery.port <n>
  --nat <method>

Metrics:
  --metrics  --metrics.addr  --metrics.port

Fork overrides (Kurtosis / devnet):
  --override.genesis <path>
  --override.glamsterdam <timestamp>
  --override.hogota <timestamp>
  --override.iplus <timestamp>

Experimental:
  --frame-mempool conservative|aggressive
  --lean-available-chain  --lean-available-validators <n>
  --stark-validation-frames
  --slot-duration 4s|6s
  --mixnet simulated|tor|nym
  --bls-backend blst|pure-go
```

Build and run:

```sh
cd pkg && go build -o eth2030 ./cmd/eth2030/
./eth2030 --network sepolia --authrpc.jwtsecret /run/jwtsecret
```

---

Parent: [`cmd`](../)

## Offline binary state migration

```sh
eth2030 binary-migrate -source /path/to/flat-state -destination /path/to/binary-state \
  -source-root 0x<32-byte-mpt-root> -batch-size 1000
```

The source must be an existing FileDB containing committed TrieStateDB account,
storage and code tables. Supply its trusted MPT root. Both databases are locked
for the duration of the command. The command prints the source root, binary root,
account count and completion flag as JSON.

Use `-max-batches 1` to stop after one durable checkpoint. Run the same command
again to resume. This is a full offline conversion of a quiescent source. The
current native node keeps its execution state in memory, and its `chaindata`
directory alone does not contain these flat state tables. The command produces
a binary snapshot for inspection and recovery. Fork transitions use the
separate overlay and conversion rules referenced by EIP-7864.

## Private-chain PQ draft profile

Set `config.pqDraft` in genesis to enable the pinned EIP-7932 and EIP-8051
verifiers at a chosen timestamp. The profile requires Berlin and three distinct,
unoccupied addresses. For example, a private chain can use the following
configuration.

```json
{
  "time": 1000,
  "sigRecover": "0x0000000000000000000000000000000000007001",
  "mldsa": "0x0000000000000000000000000000000000007002",
  "mldsaEth": "0x0000000000000000000000000000000000007003"
}
```

For recovery, allocate code `0x00 || expandedPublicKey` at the address formed
by the last 20 bytes of Keccak256 of that key. The leading STOP makes the
account inert when called. The registry checks the key hash and the signature.
Recovery charges the draft base and algorithm gas plus a Berlin or Glamsterdam
cold or warm account access and 1926 gas for copying the code. The direct
verifiers charge the pinned draft's 4500 gas and take the key in calldata.

The private profile addresses and key account encoding are ETH2030 choices.
The pinned drafts overlap at address `0x12`, also occupied by the pinned
EIP-7997 factory. Reserved addresses and active precompile collisions are
rejected. Normal network configurations have no PQ draft profile.

# Fixed specifications and isolated profiles

HGT-001. The [source lock](../../../specs/hegota/source-lock.json) retains 15 repository records, 108 source-file hashes and historical reference provenance. The original review remains [at its historical revision](https://github.com/alt-research/eth2030/blob/6a17479/docs/plans/hegota-a/specification-profiles.md).

| Source | Fixed revision / role |
| --- | --- |
| EIPs | `ade9e9cab7e148bba4cd293f916a4a189cd7a8aa`; 72 proposal/discussion/admission records. Unmerged proposals remain explicitly identified pinned copies. |
| consensus-specs | `d2021d2719235f0df4726a1d330502cb15aa8ea3`; Gloas/Heze research, not deployment evidence. |
| EELS / current test source | `b7e5b4bda2e1e62004b491dd74f119fe6622f0b3`, `forks/bogota`; chain-ID/nonce-width disagreement stays D09. |
| Archived EEST | `10eaa63d5da2f50b63d4359968f36542212f9f50`; distinct from current execution-specs test source. |
| Engine API | `1ca0818569467cff85b7f41372f22e728d5daa14`; method-specific wire qualification remains explicit. |
| Direct reference | **Geth v1.17.10**, `1df30b18d9ed35ee9d28ffd4648442c2204554ad`; module checksum in `active_reference` and `pkg/go.sum`. Lock, profile, module and runner identity are cross-checked. v1.17.1 is explicitly historical. |
| Legacy fixtures | `1f581b8ccdc4c63acf5f2c5c1b155c690c32a8eb`; two unmodified files and 12 mandatory cases. |
| Lighthouse | Stable `57c2abc1`, Bogota descriptor `8e0aeef8`, Heze descriptor `f66bd5ac`; full SHAs locked. Distinct roles, not one validated build. |
| Devnet / tools | ethereum-package `90d0f4e7`, spamoor `97fe0b9f`, Bogota descriptor `24ef46c3`, Heze descriptor `7dd667c6`; full source/file hashes locked. Mutable image tags remain unqualified. |

| Profile | Qualified scope | Activation limit |
| --- | --- | --- |
| `native-history-v1` | Twelve pinned historical GST cases, native versus direct Geth v1.17.10: roots/logs/gas | Explicit fixture forks; no full historical/Prague/Glamsterdam/Hegota/BAL/receipt/network claim. |
| `legacy-frame-prototype-v0` | Frozen local format/approval semantics | Experimental opt-in plus block/time prerequisites; never the current canonical envelope. |
| `8141-base-research-v1` | Schema, independent encoding/arithmetic and concrete semantic goldens | Research profile enables nothing. Existing master canonical execution is tested separately; full selected-spec conformance is not implied. |
| `bogota-devnet-2-descriptor` | Source-pinned GLOAS epoch 0 / HEZE disabled descriptor | Not executed as a qualified network. |
| `heze-devnet-2-descriptor` | Source-pinned HEZE epoch 5 descriptor | No current-master parameter parity or combined Frames interoperability claim. |
| `hegota-release` | No fully qualified admitted release composition | Activation remains `none`; specification, implementation and release gates stay open. |

D01–D09 in [profiles.json](../../../specs/hegota/profiles.json) retain owners, affected tasks and closure conditions for nonce fees/envelope, PQ composition, FOCIL/network scope, claims, code adoption, assertion rollback, paired SELFDESTRUCT changes, state-growth alternatives and EIP/EELS widths. Independent 8250/8272 rule goldens do not resolve composition or activate public-network state changes.

EIPs 8141, 8037 and 8038 are byte-identical at master's implementation pin `dc5d5fa6` and research pin `ade9e9ca`: reuse the implementation and qualify deltas. Source changes require appended dated old/new impact decisions; immutable URLs cannot silently acquire new digests. References under `refs/` remain unmodified.

## D01 resolution addendum (2026-11-26)

The earlier isolated-envelope assessment is superseded for the selected native
8141/8250 profile by merged EIPs #11968/#12414 and actual EELS source at
6013c4eaed5c9cc1cbb8428541972d6976a8cabd. The decision history is retained in
profiles.json. See [COMPOSITION.md](../hegota-c/COMPOSITION.md) for the two-fee
schema, explicit optional activation, native consumers and remaining upstream
canonical-paymaster gate. Frozen A/B oracle provenance is historical and is not
a declaration that the current selected public profile is still blocked by D01.

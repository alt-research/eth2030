# Preserved failed and superseded qualification attempts

1. Initial enclave: startup rejected the inherited privileged WS modules. No runtime pass.
2. Second enclave: runtime capture lost the CL endpoint after Docker reassigned its published port on crash/start. No complete-window pass.
3. Third enclave: runtime passed, but retrospective BLS first used the mainnet Gloas activation value. The timing-correction attempt was aborted. A further attempt could not retrieve pruned early non-checkpoint states. No whole-window BLS pass.
4. Final retained enclave: runtime passed. The first offline verifier rejected an actual unconfirmed slot-72 orphan because it assumed a linear slot sequence. The corrected verifier authenticates every captured block/state/signature, checks all parent links, and independently walks the retained canonical-tip header. The second offline verification passed, retaining the orphan and all earlier failures.

Artifacts remain under `/projects/eth2030/qualification/fcr-proof-20261124-attempt{1,2,3,4}`. The final canonical tip was captured in a separately timestamped follow-up after the original runtime window; the raw window and initial failed BLS report were not overwritten. See WINDOW_FILES.json and BLS_WINDOW.json.

These collector/reference defects and the expected unconfirmed-block reorg do not establish the historical original-04 waiting-chain root cause. No new 48h/scale pass is claimed.

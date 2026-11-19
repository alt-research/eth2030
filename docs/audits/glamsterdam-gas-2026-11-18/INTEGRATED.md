# Aggregate signed gas observations

Measured native revision: `24e0bd542ebb36676ab6831a33bfb43cd4d6f2db` (clean). Reference: `github.com/ethereum/go-ethereum v1.17.10`.

signed transaction transitions; pool and outer block validation are not invoked.

Counts (total, differing observable fields): `{"amsterdam_merged_text_candidate": [22, 19], "osaka_control": [22, 5]}`.

The ordinary Osaka creation/destruction control now agrees after PR 72, and the existing-value Amsterdam transfer agrees after PR 71. The remaining Osaka high-limit cases compare exported transition checks; native outer block validation already enforces its Osaka cap and is outside this corpus. They do not establish an Osaka production bypass.

The Amsterdam candidate still differs in 19 of 22 cases. Examples include new-account runtime OOG (native success versus reference included failure at 21,000 gas), new-account value transfer (21,000 versus 204,600 user gas), new storage (43,106 versus 125,026 user gas, including 97,920 reference state gas), and funded-reservoir exceptional settlement. Differences in fee-dependent state roots do not by themselves establish another storage defect.

Full frozen transactions, pre-state/config/context, observations and per-field differences: [native-integrated.json](native-integrated.json). The gas model remains unqualified. No scenario or mismatch was waived to close a release gate.

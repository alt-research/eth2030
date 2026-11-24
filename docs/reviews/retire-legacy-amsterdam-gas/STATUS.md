# Status

G04 against master 5fedb7e5. No active canonical caller uses the obsolete 4500-base schedule/floor helpers; later experimental tests still consume the table. Preserve source compatibility while removing their false current-consensus authority. Current transaction admission/settlement uses CalculateAmsterdamIntrinsic and applyAmsterdamMessage, and opcode pricing is installed by amsterdam_gas_table.go.

Local validation passes: all four affected race suites, both node builds, 157 static pages and export guards. The tracker resolves all 422 source paths for 149 entries. Both modified Go files have identical non-comment token streams to master, proving the retained compatibility behavior is unchanged. Current source pointers/titles/status metadata now match the active Amsterdam transition and current EIP names. Final hosted CI and PR conflict checks remain publication gates.

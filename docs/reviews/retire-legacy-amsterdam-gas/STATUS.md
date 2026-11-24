# Status

G04 against master 5fedb7e5. No active canonical caller uses the obsolete 4500-base schedule/floor helpers; later experimental tests still consume the table. Preserve source compatibility while removing their false current-consensus authority. Current transaction admission/settlement uses CalculateAmsterdamIntrinsic and applyAmsterdamMessage, and opcode pricing is installed by amsterdam_gas_table.go.

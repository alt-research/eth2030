# Confirmed process-level defect

The final process campaign observes a real EISDIR failure at the head file after the durable WAL commit. The first forkchoice returns an error, but the next one returns SYNCING because HasValidatedBlock hides the fenced DB read error. Preserve the storage/resource distinction before Engine head/parent lookup shortcuts. EIP-3675 atomic head/finality authority is unchanged. Original failed attempt process-lifecycle-02 remains retained.

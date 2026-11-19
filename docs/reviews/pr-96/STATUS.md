# Simulation system deadline correction qualified

PR 90 CI exposed a real error-classification gap: post-execution request system calls can return a wrapped context deadline before the following context check, becoming generic RPC -32603 instead of timeout -32016. The five-second work bound fired, but its public error depended on which phase returned first.

The shared simulation failure boundary now recognizes wrapped deadline/cancellation errors and an expired execution context before generic RPC wrapping. Genuine system reverts preserve their distinct errors. Direct native pre-execution, builder-deposit and builder-exit system loops exercise actual timed cancellation; the two builder request phases failed before the correction. The original HTTP timeout/state-isolation regression passes three consecutive runs.

Full Go tests, integration variant, Node/backend race checks, full vet and build pass locally. This changes error classification only: it does not extend the five-second request budget or weaken transaction/system validation. Protocol system behavior remains based on selected EIP-8037 system budgets and EIP-8282 request system calls. Request-owned simulation state remains separate from canonical state.

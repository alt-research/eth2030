# Simulation deadline CI correction in progress

PR 90 CI exposed a real error-classification gap: post-execution request system calls can return a wrapped context deadline before the following context check, becoming generic RPC -32603 instead of timeout -32016. The five-second work bound fired, but the public error depends on where it fired. This review preserves the deadline and fixes deterministic classification across system and transaction phases.

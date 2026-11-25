# Persisted runtime configuration and restart authority

Branch `feat/glamsterdam-config-recovery`, independently based on `38ea9259`.
This supplies the production restart consumer of configuration compatibility,
composing with PR 78's canonical startup and Engine validation.

## Source and policy

Trace Node construction through genesis loading/CLI overrides, NewBlockchain
history restoration and state adoption. Existing core.CommitGenesis already
writes `ETH2030-chain-config`; actual Node construction did not read or write
that authority, and CheckConfigCompatible had no production consumer. A changed
future activation moved into existing history could therefore be replayed under
different rules without a prior configuration check.

Read [EIP-7892](https://github.com/ethereum/EIPs/blob/3db38a3725105cb67b8804ac3e147967113bc3ca/EIPS/eip-7892.md)
in full (file blob `af10bf37`). BPO activation is part of the execution schedule
and can interleave with named upgrades. The fork registry now includes the
existing Osaka, Muir/Arrow/Gray Glacier, IPlus and BPO fields that were omitted.
NextForkAfter compares actual activation points, preserving the distinction
between block-based and timestamp-based schedules. BPOs require prior/simultaneous
Prague and increasing configured BPO order; no requirement places every BPO
before Amsterdam or requires an unused BPO1 when only BPO2 is configured.
This does not implement configurable blob parameter values or qualify EL/CL
epoch alignment; those remain separate acceptance work.

## Actual startup behavior

Before NewBlockchain can restore/replay history, Node validates the resolved
configuration and reads the existing configuration and head. The head's stored
number/header/hash must agree. Active differences on either side reject startup;
chain ID is fixed once recorded. Registered genuinely future forks can change,
including disabling an inactive fork. After executed history exists, other
parameters are immutable without an explicit migration rule, including changed
contract addresses and unregistered research settings.

After successful blockchain restoration, Node persists the resolved runtime
configuration, including overrides, and propagates write failures. It copies
network defaults before applying node-local overrides. Persisted config parsing rejects
invalid/unknown fields and trailing data; metadata or storage failures do not
silently select defaults. Failed compatibility checks do not overwrite existing
configuration or select a new head.

A fresh database can initialize normally. A verified genesis-only legacy database
can record its configuration after NewBlockchain checks genesis identity. An
executed legacy database without configuration authority is rejected: the code
does not infer historical rules from the new input. Its existing files are
preserved; a separately initialized directory with a verified genesis and replay
is the supported recovery boundary. Orphan metadata or a stored genesis without
a readable head is not treated as a fresh database. No migration flag, silent
rewrite or destructive cleanup is introduced.

## Evidence

The actual Node HTTP test independently signs a Geth legacy transaction, submits
it through the authenticated endpoint, builds/imports a real payload and selects
its head. It stops the node and checks actual FileDB configuration. Restarts with
an activation moved into history, a different chain ID or a different deposit
contract address are rejected before replay. The configuration bytes and head
remain unchanged. Moving a future activation from time 100 to 200 succeeds;
another fresh Node reads that setting and the original signed-transaction balance,
head and state root.

Additional tests cover missing/invalid/unknown/trailing metadata, orphan authority,
injected reads/writes, head absence, either-side activation, removed active Osaka,
future BPO updates, new non-schedule parameters and mutation of network globals.
Each of the 24 fields allowed to change as a scheduled fork must also participate
in compatibility authority. Optional fork differences are symmetric; BPOs on
either side of Amsterdam produce the correct next activation.

All 210 tested Go packages, affected tests, six race-tested packages, vet and
local build pass. Four compiling behavioral mutations fail their intended real
consumer tests: omit startup checking, omit configuration persistence, permit a
historical contract-address change, or apply overrides to shared network defaults.
Sources are restored and affected tests pass. Logs are
`/tmp/glamsterdam-config-recovery-{all,race,final-affected,mutation-*}.log`.

This qualifies actual Node configuration adoption and restart. It does not claim
unrun power-loss/interprocess-storage guarantees, automatically migrate unknown
legacy history, qualify all direct library constructors or close final gas,
complete blob schedules, multi-client devnets and release gates.

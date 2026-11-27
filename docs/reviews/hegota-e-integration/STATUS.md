# Group E status

Production source: `63d673ed216d08058c56404ee1daa06c19ad1d73`, based directly on merged PR #171/master
`cc31afffaa12919bbc08dc854c03af1e1928dc3c`. No base conflicts or changes to
read-only refs or independent A/B/C/D expected oracles.

HGT-040–045 native implementation is delivered. The selected current Heze
companion authenticates and retains branch-keyed ILs, separates timely/late
views and detects equivocation. Native Bogotá builds private IL transactions,
retries nonce/funding cascades, enforces gas/complete-block RLP capacity and
retains execution/satisfaction decisions across branch selection and restart.

The 36-criterion register names 27 mandatory native consumers. Profile 1 checks
actual signatures and full-width account state without EVM execution. Payload
execution validity remains independent from IL satisfaction. Replacements cancel
unfinished builds; context cancellation rolls back trials and aborts publication.

HGT-046 is explicitly gated by D04: EIP-8369 remains Informational and does not
define enough to activate Profile 2. Defined reference predicates and thirteen
activation prerequisites are implemented, but no production partial-stateless
replay or consensus claims encoding is asserted. HGT-047 qualifies local native
and CL decision adapters; external beacon transport/CL/ePBS network acceptance
remains HGT-070/080 work. HGT-048 measures real canonical direct-8504 bytes and
verification/local pipeline cost; omitted witness transport and network tail
latency remain unqualified. These gates prevent a claim that all Group E public
activation functionality is complete.

The website homepage, roadmap and EIP-7805 progress now display this precise
implementation/gate split. GitHub Pages deploys the repository site after merge
through the existing master workflow; this branch does not claim a live deploy.

Verification and final English PR/check links will be recorded after the exact
source runs complete. The first concurrent full/race attempt hit the existing
one-second native commit-barrier timeout. A separate full run passed without
changing that test. Header-copy vet findings were fixed before final reruns.

Final path audit also added authenticated pooled-witness retrieval for canonical PQ IL builds; a native RPC/Engine regression detects its removal.

Restart restores only the still-required retention window, rejecting future cached lists and requiring authenticated state for retained lists; long shutdowns no longer require obsolete beacon state.

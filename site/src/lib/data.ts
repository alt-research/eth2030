// Merges, measured runtimes and hosted CI retain distinct pinned records.
export const developmentSnapshot = {
  date: 'November 23, 2026',
  revision: 'd743925e3c4e7deb3fdc00648cb2296372910b28',
  pullRequest: 141,
  title: 'Frozen development scope delivered; readiness gates open',
  recordRevision: '9a094830b8d11ab541df4f439a3730941311d430',
  record: 'docs/reviews/glamsterdam-postmerge-20261121/RESULTS.json',
  reviewRevision: '1317a6624e6246a8e0937f56d2758b300ccbcfcb',
  stateRecord: 'docs/reviews/remaining-20261123/CURRENT_STATE.json',
  latestRecord: 'docs/reviews/diagnostic-threshold-20261123/COMPLETED_RUN.json',
  plan: 'docs/plans/glamsterdam/NEXT_STEPS_20261123.md',
  companion: { revision: 'fa16f70abee7d31c9817ab8d782af73bce241484', url: 'https://github.com/jiayaoqijia/lighthouse/commit/fa16f70abee7d31c9817ab8d782af73bce241484' },
  milestones: [
    { title: 'Frozen development scope', value: '33/33 task IDs', detail: 'Merged LIFO and signed Gloas target. EL PRs 131–140 and the companion CL integration are merged.' },
    { title: 'Completed bounded run', value: '6h45m05s', detail: 'After the ready-observed 5.5h threshold: 144 healthy samples and 72 full stacks. No new sustained stall captured.' },
    { title: 'Independent signed finality', value: '14 BLS windows', detail: '8068 observed slots; unchanged-reference replays and linked anchors. 73 early missing slots remain recorded.' },
    { title: 'Actual stopped endpoint', value: 'slot 97868', detail: '10 slots beyond the last health sample. Persisted CL body, native header/state and 160 live + 2 stopped account proofs agree.' },
  ],
  currentCampaign: { detail: 'Post-repair cumulative code: 91s native RPC, 212s catchup, 301s ready; 102 signed slots, actual stopped slot 113651, 4 live + 2 stopped proofs and 60 matching height-gauge reads. The initial 300s full catchup failed; this separately declared 900s recovery keeps it failed. Earlier 30m: 604 observed slots and actual stopped slot 110547, with its own executable and receipts.', record: 'docs/reviews/remaining-20261123/POST_REPAIR_RECOVERY.json', reviews: [142, 143, 144, 145, 146] },
  historicalMilestones: ['4 roles under 300s', '400/400 + 60/60 slots', '544 KZG cell proofs'],
  followUp: { title: 'Merged repairs; historical attribution remains open', revision: '84200d5322398ebe473ff26c6f54de2665cd2586', reviews: [135, 138, 139, 140], detail: 'The captured FCU fsync/read-lock convoy and snap-history corrections are merged. Rounds 03/04 remain failed. Original 04 has no contemporaneous waiting stack: its exact holder and physical I/O cause remain unconfirmed. The bounded run is not a 48-hour pass.' },
  gates: [
    { id: 'G-01', state: 'closed', title: 'Integrated CL CI', detail: 'Final head completed 27 test-suite jobs and local-testnet (9 passed, 1 conditional checkpoint skip). The repaired baseline completed 27 hosted jobs. 6s primary timing retains the full aggregate assertion and original 3s failure.', url: 'https://github.com/jiayaoqijia/lighthouse/actions/runs/45166237988' },
    { id: 'G-02', state: 'unconfirmed', title: 'Original incident attribution', detail: 'Preserve request-correlated stacks and storage timing on any recurrence. No-recurrence and controlled recovery cannot reconstruct the missing original waiter.' },
    { id: 'G-03', state: 'open', title: 'Duration and capacity', detail: 'Actual 48h acceptance is absent. Eight finite KV/WAL cases pass. A real ext4 flat-directory limit is repaired in PR146; all 2707772 snapshot values and independent header/MPT readback agree. Fresh cumulative recovery qualification and unrestricted capacity stay separate.' },
    { id: 'G-04', state: 'open', title: 'Upstream Lodestar', detail: 'PRs 10693 and 10725 await maintainer workflow approval. Local full unit run: 5547 passed, 9 skipped; corrected TS coverage has 821 records. These are not hosted upstream passes.', url: 'https://github.com/ChainSafe/lodestar/pull/10693' },
    { id: 'G-05', state: 'reserved', title: 'Public target and activation', detail: 'Final gas/model, compatible specification/client tuple and public activation require an explicit decision and affected-path requalification.' },
    { id: 'G-06', state: 'conditional', title: 'Coverage expansion', detail: 'Add specifically adopted formats and consumers beyond the finite qualified corpora. Skipped BLS cases and inactive handlers are not executed passes.' },
  ],
  boundary: 'This qualifies the selected frozen development scope, not whole-public-fork or production readiness. Historical tracker percentages count mapped labels. Frame/FOCIL, service migration and research features retain separate acceptance gates.',
};

export const evidence = [
  { title: 'Block-level access lists', spec: 'EIP-7928', eip: '7928', code: 'pkg/core/chain/block_access_list.go', test: 'pkg/core/chain/snap_history_retention_test.go', qualification: 'Frozen target qualified', summary: 'Canonical BALs are execution-validated, persisted and served through independent historical hash/range consumers. Snap history resumes after legal expiry without weakening retained-byte or corruption checks.', limit: 'Downloaded pre-pivot history does not gain state or execution authority. Public activation and later specification changes require their own review.' },
  { title: 'Enshrined proposer-builder separation', spec: 'EIP-7732', eip: '7732', revision: '84200d5322398ebe473ff26c6f54de2665cd2586', code: 'pkg/engine/server.go', test: 'pkg/engine/legacy_builder_retired_test.go', qualification: 'Signed Gloas route qualified', summary: 'Actual external signed builder bids, payments, withheld/late payloads, reorgs, registry reuse and independent signed finality have retained process evidence.', limit: 'PR 127 retires unsupported legacy Builder Engine extensions, including synthetic stake and the signature-dropping header route. This source is merged; standard CL signatures and finalized registry/collateral stay authoritative. Upstream Lodestar and public review remain open.' },
  { title: 'Inclusion lists (FOCIL)', spec: 'EIP-7805', eip: '7805', code: 'pkg/engine/bogota.go', test: 'pkg/node/frame_8141_integration_test.go', qualification: 'Paired later-fork control', summary: 'Signed inclusion lists delivered a sponsored Frame between isolated EL mempools; both pairs finalized it and agreed on settlement.', limit: 'Independent Frame/Bogota verdict agreement and the Dora production consumer remain separate later-fork acceptance gates.' },
  { title: 'Frame transactions', spec: 'EIP-8141', eip: '8141', code: 'pkg/core/execution/frame_8141.go', test: 'pkg/node/frame_8141_integration_test.go', qualification: 'Pinned draft qualified locally', summary: 'Merged native seven-field execution, split gas, approvals, atomic rollback, blob admission and a reproducible paymaster v1 candidate.', limit: 'The pinned draft and local paymaster ABI passed the paired control. Independent-client agreement and production service cutover remain open.' },
  { title: 'Native rollups', spec: 'EIP-8079', eip: '8079', code: 'pkg/rollup/execute', test: 'pkg/rollup/execute_test.go', qualification: 'Research implementation', summary: 'Experimental EXECUTE-related code and tests.', limit: 'This is not evidence of a complete, production-ready implementation of the draft.' },
];

export const gettingStartedCommands = [
  { comment: '# Clone the repository', command: 'git clone https://github.com/alt-research/eth2030.git' },
  { comment: '# Build and test the Go module', command: 'cd eth2030/pkg && go build ./... && go test ./...' },
  { comment: '# Build the experimental node', command: 'go build -o eth2030-geth ./cmd/eth2030-geth/' },
];

// Merged implementation snapshot; runtime revisions are retained in the run record.
export const developmentSnapshot = {
  date: 'November 22, 2026',
  revision: '9a094830b8d11ab541df4f439a3730941311d430',
  pullRequest: 125,
  title: 'Glamsterdam development target merged and qualified',
  record: 'docs/reviews/glamsterdam-postmerge-20261121/RESULTS.json',
  plan: 'docs/audits/merged-glamsterdam-20261122/README.md',
  milestones: [
    { title: 'Frozen development scope', value: '33/33 task IDs', detail: 'Merged LIFO and signed Gloas target; 219 package suites and 108 official RPC requests pass.' },
    { title: 'Original-datadir recovery', value: '4 roles under 300s', detail: 'Both native pairs and full/snap observers preserve finalized commitments and independent proofs.' },
    { title: 'Steady and pressure', value: '400/400 + 60/60 slots', detail: 'Unchanged 20-minute and 3-minute budgets; independent BLS finality and offline replay pass.' },
    { title: 'External-builder blob', value: '544 KZG cell proofs', detail: 'Two ordered rows, authenticated builder signature and useful partial acquisition with getBlobs disabled.' },
  ],
  boundary: 'Qualified for the frozen merged LIFO / signed Gloas development target. Public protocol/model approval, upstream Lodestar integration and production-scale operation remain separate. Frame/FOCIL and service migration retain their own later-fork acceptance gates.',
};

export const evidence = [
  { title: 'Block-level access lists', spec: 'EIP-7928', eip: '7928', code: 'pkg/core/chain/block_access_list.go', test: 'pkg/core/chain/snap_history_retention_test.go', qualification: 'Frozen target qualified', summary: 'Canonical BALs are execution-validated, persisted and served through independent historical hash/range consumers. Snap history resumes after legal expiry without weakening retained-byte or corruption checks.', limit: 'Downloaded pre-pivot history does not gain state or execution authority. Public activation and later specification changes require their own review.' },
  { title: 'Enshrined proposer-builder separation', spec: 'EIP-7732', eip: '7732', code: 'pkg/engine/builder_api.go', test: 'pkg/engine/gloas_payload_e2e_test.go', qualification: 'Signed Gloas route qualified', summary: 'Actual external signed builder bids, payments, withheld/late payloads, reorgs, registry reuse and independent signed finality have retained process evidence.', limit: 'Legacy EL builder-registration/bid extensions still lack signature authentication; they are a separate code gap from the qualified CL signed-envelope route. Upstream Lodestar review remains open.' },
  { title: 'Inclusion lists (FOCIL)', spec: 'EIP-7805', eip: '7805', code: 'pkg/engine/bogota.go', test: 'pkg/node/frame_8141_integration_test.go', qualification: 'Paired later-fork control', summary: 'Signed inclusion lists delivered a sponsored Frame between isolated EL mempools; both pairs finalized it and agreed on settlement.', limit: 'Independent Frame/Bogota verdict agreement and the Dora production consumer remain separate later-fork acceptance gates.' },
  { title: 'Frame transactions', spec: 'EIP-8141', eip: '8141', code: 'pkg/core/execution/frame_8141.go', test: 'pkg/node/frame_8141_integration_test.go', qualification: 'Pinned draft qualified locally', summary: 'Merged native seven-field execution, split gas, approvals, atomic rollback, blob admission and a reproducible paymaster v1 candidate.', limit: 'The pinned draft and local paymaster ABI passed the paired control. Independent-client agreement and production service cutover remain open.' },
  { title: 'Native rollups', spec: 'EIP-8079', eip: '8079', code: 'pkg/rollup/execute', test: 'pkg/rollup/execute_test.go', qualification: 'Research implementation', summary: 'Experimental EXECUTE-related code and tests.', limit: 'This is not evidence of a complete, production-ready implementation of the draft.' },
];

export const gettingStartedCommands = [
  { comment: '# Clone the repository', command: 'git clone https://github.com/alt-research/eth2030.git' },
  { comment: '# Build and test the Go module', command: 'cd eth2030/pkg && go build ./... && go test ./...' },
  { comment: '# Build the experimental node', command: 'go build -o eth2030-geth ./cmd/eth2030-geth/' },
];

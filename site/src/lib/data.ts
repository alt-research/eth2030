// Merged implementation snapshot; runtime revisions are retained in the run record.
export const developmentSnapshot = {
  date: 'November 18, 2026',
  revision: '911afb0c42a2e33c9e65881b31ed1b1c446390b8',
  pullRequest: 61,
  record: 'docs/verification/frame-8141-20261118-paymaster-history-final.json',
  plan: 'docs/next-stage-work-plan.md',
  milestones: [
    { title: 'Sponsored FOCIL', value: 'Slot 66', detail: 'Coded paymaster Frame finalized on both node pairs.' },
    { title: 'Blob Frame', value: 'Slot 165', detail: 'Finalized with matching commitments and sponsor fees.' },
    { title: 'Fresh CL history', value: 'Joined at slot 256', detail: 'A new consensus database read both original finalized envelopes.' },
    { title: 'Receiver restart', value: 'Readback passed', detail: 'Both receipts, historical storage and payer balances survived restart.' },
  ],
  boundary: 'Qualified on paired ETH2030 / Lighthouse nodes. The new CL used a synchronized EL. Independent EL interoperability, Dora consumer migration and state-sync acceptance remain next steps.',
};

export const evidence = [
  { title: 'Block-level access lists', spec: 'EIP-7928', eip: '7928', code: 'pkg/core/chain/block_access_list.go', test: 'pkg/core/chain/block_access_list_test.go', summary: 'Execution-validated RLP BALs persist with blocks and remain available through historical Engine body APIs after restart.', limit: 'The paired history control passed; full current-spec conformance and independent execution-client validation remain open.' },
  { title: 'Enshrined proposer-builder separation', spec: 'EIP-7732', eip: '7732', code: 'pkg/epbs', test: 'pkg/epbs/validation_test.go', summary: 'Builder bid and payload validation, payment state and Engine API code.', limit: 'Bid validation permits zero signatures or public keys in its testing path; end-to-end consensus interoperability is not established here.' },
  { title: 'Inclusion lists (FOCIL)', spec: 'EIP-7805', eip: '7805', code: 'pkg/engine/bogota.go', test: 'pkg/node/frame_8141_integration_test.go', summary: 'Signed inclusion lists delivered a sponsored Frame between isolated EL mempools; both pairs finalized it and agreed on settlement.', limit: 'Independent execution-client verdicts, alternate-peer recovery cases and the updated Dora consumer remain separate acceptance gates.' },
  { title: 'Frame transactions', spec: 'EIP-8141', eip: '8141', code: 'pkg/core/execution/frame_8141.go', test: 'pkg/node/frame_8141_integration_test.go', summary: 'Merged native seven-field execution, split gas, approvals, atomic rollback, blob admission and a reproducible paymaster v1 candidate.', limit: 'The pinned draft and local paymaster ABI passed the paired control. Independent-client agreement and production service cutover remain open.' },
  { title: 'Native rollups', spec: 'EIP-8079', eip: '8079', code: 'pkg/rollup/execute', test: 'pkg/rollup/execute_test.go', summary: 'Experimental EXECUTE-related code and tests.', limit: 'This is not evidence of a complete, production-ready implementation of the draft.' },
];

export const gettingStartedCommands = [
  { comment: '# Clone the repository', command: 'git clone https://github.com/alt-research/eth2030.git' },
  { comment: '# Build and test the Go module', command: 'cd eth2030/pkg && go build ./... && go test ./...' },
  { comment: '# Build the experimental node', command: 'go build -o eth2030-geth ./cmd/eth2030-geth/' },
];

#!/usr/bin/env python3
"""Run compiled semantic-removal controls without editing production or tests."""
import argparse,json,subprocess
from pathlib import Path
R=Path(__file__).resolve().parents[2]
parser=argparse.ArgumentParser()
parser.add_argument('--output',type=Path,required=True)
parser.add_argument('--only',action='append',help='Run named controls when resuming a qualified unchanged-source batch')
args=parser.parse_args()
D=args.output.resolve();D.mkdir(parents=True,exist_ok=True)
controls=[
('BLS-authentication','pkg/focil/heze/validation.go','if !crypto.BLSVerify(pubkey, root[:], s.Signature) {','if false && !crypto.BLSVerify(pubkey, root[:], s.Signature) {','./focil/heze','TestGroupEHezeAuthenticatedGossipAndFailureAtomicity'),
('strict-timeliness','pkg/focil/heze/types.go','DueBasisPoints      = 7500','DueBasisPoints      = 7501','./focil/heze','TestGroupEHezeBranchTimelinessEquivocationAndOwnership'),
('nonce-cascade','pkg/core/block/inclusion_queue.go','if !c.inclusion {','if true {','./core/block','TestGroupEBuilderReverseNonceILCascade'),
('RLP-byte-cap','pkg/core/block/inclusion_size.go','size > MaxRLPBlockSize','size > MaxRLPBlockSize * 2','./core/block','TestGroupEBuilderExactRLPBudgetBoundaryAndRejectedAppendRollback'),
('signature-budget','pkg/focil/vops_reference.go','r.Reason = "invalid protocol signature"','remaining += r.SignatureCost; r.Reason = "invalid protocol signature"','./focil','TestGroupEVOPSOccurrenceBudgetSignatureDebitAndShapeFailure'),
('omission-verdict','pkg/node/backend/engine_bogota.go','// EIPs PR #12723 proposes a different size rule and remains unmerged.\n\t\treturn false, nil','// EIPs PR #12723 proposes a different size rule and remains unmerged.\n\t\treturn true, nil','./node','TestGroupEHezeGossipNativeEngineForkchoiceAndRestart'),
]
controls.append(('pooled-IL-witness','pkg/node/backend/engine_backend.go','return r.pool.Get(hash)','return nil','./node','TestGroupECanonicalPQILUsesAuthenticatedPoolWitness'))
controls.append(('expired-cache-pruning','pkg/focil/heze/store.go','if e.First.Message.Slot < minimum {','if false && e.First.Message.Slot < minimum {','./focil/heze','TestGroupEHezeRequestsRetentionAndReauthenticatedRestart'))
controls.extend([
('ignored-type3','pkg/node/backend/engine_bogota.go','if err != nil || tx.Type() == types.BlobTxType {','if err == nil && tx.Type() == types.BlobTxType { return false, nil }; if err != nil {','./node','TestGroupELatest7805Type3IgnoredAndPoolBlobAvailability'),
('failed-id-retry','pkg/node/backend/engine_fcu.go','retry = existing.err != nil','retry = false','./node','TestGroupECanonicalPQILUsesAuthenticatedPoolWitness'),
('IL-type3-cannot-stall-builder','pkg/core/block/inclusion_queue.go','if err != nil || tx.Type() == types.BlobTxType {','if err != nil {','./node','TestGroupELatest7805Type3IgnoredAndPoolBlobAvailability'),
('malformed-proof-tolerance','pkg/node/backend/engine_bogota.go','canonical && frame.ValidateWithExtensions(bc.Config().FrameExtensions(block.Time())) != nil','false && canonical && frame.ValidateWithExtensions(bc.Config().FrameExtensions(block.Time())) != nil','./node','TestGroupECanonicalPQILUsesAuthenticatedPoolWitness'),
])
controls.append(('append-header-metadata','pkg/core/types/block_append_size.go','if header.BlobGasUsed != nil {','if false && header.BlobGasUsed != nil {','./core/types','TestGroupEExactAppendRLPAllPrefixAndGasBoundaries'))
controls.extend([
('finalized-verdict','pkg/core/chain/bogota_payload.go','if len(record.Block) == 0 {','if false && len(record.Block) == 0 {','./node','TestBogotaDeferredWireAcrossRestart'),
('reconfigured-authentication','pkg/node/heze_inclusion.go','store, err = heze.Restore(data, view)\n\t\tif err != nil {','_ = data\n\t\tstore = n.hezeLists\n\t\terr = nil\n\t\tif err != nil {','./node','TestGroupEHezeReconfigureReauthenticatesMemoryAtomically'),
])

controls.extend([
('transaction-end-accounts','pkg/node/backend/inclusion_state.go','return !e.replayed && (','return false && !e.replayed && (','./node','TestGroupEWithdrawalDoesNotFundSameBlockOmission'),
('pre-execution-Frame-enforcement','pkg/node/backend/engine_bogota.go','if preconditionErr != nil {','if preconditionErr != nil || canonical {','./node','TestGroupELatest7805FramePreExecutionAndVersionMigration'),
('transaction-end-prefix','pkg/core/execution/processor.go','return p.processWithBAL(block, statedb, false)','return p.processWithBAL(block, statedb, true)','./core/execution','TestGroupETransactionEndIncludesPreSystemsAndExcludesPostSystems'),
('pruned-BAL-boundary','pkg/node/backend/inclusion_state.go','if len(encoded) == 0 {','if len(encoded) == 0 { e.known = true;','./node','TestGroupEWithdrawalDoesNotFundSameBlockOmission'),
('legacy-live-boundary','pkg/node/backend/engine_bogota.go','if record.SatisfactionVersion != chain.BogotaSatisfactionVersion {','if false && record.SatisfactionVersion != chain.BogotaSatisfactionVersion {','./node','TestGroupEWithdrawalDoesNotFundSameBlockOmission'),
('legacy-compact-boundary','pkg/node/backend/engine_bogota.go','if !record.HasInclusionListView() {','if false && !record.HasInclusionListView() {','./node','TestGroupEWithdrawalDoesNotFundSameBlockOmission'),
('compact-boundary-version','pkg/core/chain/bogota_payload.go','}{record.Number, record.Satisfied, record.SatisfactionVersion})','}{record.Number, record.Satisfied, 0})','./core/chain','TestBogotaPayloadStorageRetainsFinalizedAndPendingViews'),
])

controls.extend([
('header-only-ancestry','pkg/node/backend/inclusion_state.go','header := bc.GetHeaderByHash(hash)','var header *types.Header; if body := bc.GetBlock(hash); body != nil { header = body.Header() }','./node/backend','TestGroupEInclusionAncestryUsesAuthenticatedHeadersWithoutBodies'),
('missing-header-verdict','pkg/node/backend/inclusion_state.go','return e.ancestryErr','return nil','./node/backend','TestGroupEInclusionAncestryUsesAuthenticatedHeadersWithoutBodies'),
])

controls.extend([
('foreign-context-omission','pkg/node/backend/engine_bogota.go','canonical && execution.ValidateFrameContext8141(bc.Config(), block.Header(), frame) != nil','false && canonical && execution.ValidateFrameContext8141(bc.Config(), block.Header(), frame) != nil','./node','TestGroupECanonicalPQILUsesAuthenticatedPoolWitness'),
('foreign-context-build','pkg/core/block/builder.go','canonical && execution.ValidateFrameContext8141(b.config, header, frame) != nil','false && canonical && execution.ValidateFrameContext8141(b.config, header, frame) != nil','./node','TestGroupECanonicalPQILUsesAuthenticatedPoolWitness'),
])

controls.extend([
('frame-preconditions-omission','pkg/node/backend/engine_bogota.go','if preconditionErr != nil {','if false && preconditionErr != nil {','./node','TestGroupEFramePreconditionsBeforeMissingWitness'),
('frame-preconditions-build','pkg/core/block/builder.go','if preconditionErr != nil {','if false && preconditionErr != nil {','./node','TestGroupEFramePreconditionsBeforeMissingWitness'),
('ordinary-state-capacity','pkg/core/types/receipt.go','state += receipt.StateGasUsed','state += 0','./node','TestGroupEFrameCapacityIncludesOrdinaryStateGas'),
('Amsterdam-capacity-replay','pkg/node/backend/inclusion_state.go','|| e.bc.Config().IsAmsterdam(e.block.Time())','|| false && e.bc.Config().IsAmsterdam(e.block.Time())','./node','TestGroupEFrameCapacityIncludesOrdinaryStateGas'),
('predicate-version-three','pkg/core/chain/bogota_payload.go','const BogotaSatisfactionVersion uint8 = 3','const BogotaSatisfactionVersion uint8 = 2','./node','TestGroupELatest7805FramePreExecutionAndVersionMigration'),
('preconditions-read-only','pkg/core/execution/frame_preconditions_8141.go','if !gp.FitsFrame(prepared.budget.ExecutionReservation, prepared.budget.StateReservation) {','if gp.ReserveFrame(prepared.budget.ExecutionReservation, prepared.budget.StateReservation) != nil {','./core/execution','TestGroupEFramePreconditionsAreReadOnlyAndFailClosed'),
('preconditions-read-error','pkg/core/execution/frame_preconditions_8141.go','if readErr := state.ReadError(db); readErr != nil {','if readErr := state.ReadError(db); false && readErr != nil {','./core/execution','TestGroupEFramePreconditionsAreReadOnlyAndFailClosed'),
('dimensional-RLP','pkg/core/gaspool/gas_pool.go','return max(gasLimit-gp.execution, gasLimit-gp.state)','return gasLimit-gp.execution','./core/block','TestGroupEBuilderExactRLPIndependentGasPools'),
('ingress-publish-after-persist','pkg/node/heze_inclusion.go','store := n.hezeLists.Clone()','store := n.hezeLists; if store == nil { store = heze.NewStore() }','./node','TestGroupEHezeIngressCommitsAfterPersistence'),
('ingress-one-clock','pkg/node/heze_inclusion.go','clock, err := store.GossipWithClock(data, n.hezeView)','clock, err := store.GossipWithClock(data, n.hezeView); if err == nil { clock, err = n.hezeView.Clock() }','./node','TestGroupEHezeIngressCommitsAfterPersistence'),
])

controls.extend([
('unmerged-size-exemption','pkg/node/backend/engine_bogota.go','// EIPs PR #12723 proposes a different size rule and remains unmerged.\n\t\treturn false, nil','// EIPs PR #12723 proposes a different size rule and remains unmerged.\n data, sizeErr := block.EncodeRLP(); if sizeErr == nil && len(data) + len(raw) + 8 > blockpkg.MaxRLPBlockSize { return true, nil }; return false, nil','./node/backend','TestGroupELatest7805SizeIsNotAnOmissionExcuse'),
('independent-ordinary-reservation','pkg/focil/profile1.go','result.ExecutionReservation = min(tx.Gas(), vm.AmsterdamMaxExecutionGas)','result.ExecutionReservation = tx.Gas()','./focil','TestGroupELatest7805IndependentOrdinaryGasReservations'),
('IL-generation-no-blob','pkg/core/eips/inclusion_list.go','if tx == nil || tx.Type() == types.BlobTxType || len(tx.BlobHashes()) > 0 {','if tx == nil {','./node','TestGroupELatest7805Type3IgnoredAndPoolBlobAvailability'),
])

controls.extend([
('EIP2718-no-block-wrapper-budget','pkg/core/eips/inclusion_list.go','if totalSize+len(encoded) > maxBytes {','overhead := 0; if tx.Type() != types.LegacyTxType { overhead = 3 }; if totalSize+len(encoded)+overhead > maxBytes {','./node','TestGroupELatest2718ExactEnvelopeBudget'),
('EIP2718-canonical-not-pooled','pkg/core/eips/inclusion_list.go','encoded, err := tx.EncodeRLP()\n\t\tif err != nil {\n\t\t\tcontinue\n\t\t}\n\t\tif totalSize+len(encoded) > maxBytes {','encoded, err := types.EncodePooledTransaction(tx)\n\t\tif err != nil {\n\t\t\tcontinue\n\t\t}\n\t\tif totalSize+len(encoded) > maxBytes {','./node','TestGroupELatest2718ExactEnvelopeBudget'),
('EIP2718-one-over-rejected','pkg/core/eips/inclusion_list.go','if totalSize+len(encoded) > maxBytes {','if totalSize+len(encoded) > maxBytes+1 {','./node','TestGroupELatest2718ExactEnvelopeBudget'),
])

# The shared preflight deliberately repeats cheap context/static checks before
# witness handling. Removing an early fast filter alone is now redundant; these
# historical semantic removals disable both protections, keeping their original
# unchanged native consumers and counting only compiled behavioral failures.
entry='func ValidateFramePreconditions8141(cfg *config.ChainConfig, db state.StateDB, header *types.Header, tx *types.FrameTx8141, gp *gaspool.GasPool) error {'
extra_targets={name:[('pkg/core/execution/frame_preconditions_8141.go',entry,entry+'\n if gp != nil { return nil }')] for name in ['malformed-proof-tolerance','foreign-context-omission','foreign-context-build']}

if args.only:
 requested=set(args.only); known={c[0] for c in controls}
 if not requested<=known:raise ValueError(('unknown control',sorted(requested-known)))
 controls=[c for c in controls if c[0] in requested]

for name,path,old,new,pkg,test in controls:
 replacements={}
 for index,(target,before,after) in enumerate([(path,old,new)]+extra_targets.get(name,[])):
  source=(R/target).read_text()
  if before not in source and name=='strict-timeliness':
   import re
   before=re.search(r'DueBasisPoints\s*= 7500',source).group();after=before.replace('7500','7501')
  if source.count(before)!=1:raise ValueError(('control target must be unique',name,target,source.count(before)))
  replacement=D/(name+('-extra'+str(index) if index else '')+'.go');replacement.write_text(source.replace(before,after));replacements[str(R/target)]=str(replacement)
 overlay=D/(name+'.json');overlay.write_text(json.dumps({'Replace':replacements}))
 log=D/(name+'.jsonl')
 with log.open('w')as f:
  result=subprocess.run(['env','-u','GOROOT','GOFLAGS=-p=4','go','test','-json','-overlay',str(overlay),pkg,'-run','^'+test+'$','-count=1'],cwd=R/'pkg',stdout=f,stderr=subprocess.STDOUT)
 events=[json.loads(l)for l in log.read_text().splitlines()]
 failures=[e for e in events if e['Action']=='fail'and e.get('Test')==test]
 if result.returncode!=1 or not failures or any(e['Action']=='build-fail'for e in events):raise ValueError(('control did not compile and fail its intended consumer',name,result.returncode))
 print(name,'detected',flush=True)

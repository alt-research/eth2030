#!/usr/bin/env python3
"""Run compiled semantic-removal controls without editing production or tests."""
import argparse,json,subprocess
from pathlib import Path
R=Path(__file__).resolve().parents[2]
parser=argparse.ArgumentParser()
parser.add_argument('--output',type=Path,required=True)
args=parser.parse_args()
D=args.output.resolve();D.mkdir(parents=True,exist_ok=True)
controls=[
('BLS-authentication','pkg/focil/heze/validation.go','if !crypto.BLSVerify(pubkey, root[:], s.Signature) {','if false && !crypto.BLSVerify(pubkey, root[:], s.Signature) {','./focil/heze','TestGroupEHezeAuthenticatedGossipAndFailureAtomicity'),
('strict-timeliness','pkg/focil/heze/types.go','DueBasisPoints      = 7500','DueBasisPoints      = 7501','./focil/heze','TestGroupEHezeBranchTimelinessEquivocationAndOwnership'),
('nonce-cascade','pkg/core/block/inclusion_queue.go','if !c.inclusion {','if true {','./core/block','TestGroupEBuilderReverseNonceILCascade'),
('RLP-byte-cap','pkg/core/block/inclusion_size.go','size > MaxRLPBlockSize','size > MaxRLPBlockSize * 2','./core/block','TestGroupEBuilderExactRLPBudgetBoundaryAndRejectedAppendRollback'),
('signature-budget','pkg/focil/vops_reference.go','r.Reason = "invalid protocol signature"','remaining += r.SignatureCost; r.Reason = "invalid protocol signature"','./focil','TestGroupEVOPSOccurrenceBudgetSignatureDebitAndShapeFailure'),
('omission-verdict','pkg/node/backend/engine_bogota.go','maximum <= blockpkg.MaxRLPBlockSize {\n\t\t\treturn false, nil','maximum <= blockpkg.MaxRLPBlockSize {\n\t\t\treturn true, nil','./node','TestGroupEHezeGossipNativeEngineForkchoiceAndRestart'),
]
controls.append(('pooled-IL-witness','pkg/node/backend/engine_backend.go','return r.pool.Get(hash)','return nil','./node','TestGroupECanonicalPQILUsesAuthenticatedPoolWitness'))
controls.append(('expired-cache-pruning','pkg/focil/heze/store.go','if e.First.Message.Slot < minimum {','if false && e.First.Message.Slot < minimum {','./focil/heze','TestGroupEHezeRequestsRetentionAndReauthenticatedRestart'))
controls.extend([
('base-7805-blob','pkg/node/backend/engine_bogota.go','if result.Kind == focil.Profile1Outside {','if false && result.Kind == focil.Profile1Outside {','./node','TestGroupEBaseFOCILBlobValidityAvailabilityAndSameIDRetry'),
('failed-id-retry','pkg/node/backend/engine_fcu.go','retry = existing.err != nil','retry = false','./node','TestGroupEBaseFOCILBlobValidityAvailabilityAndSameIDRetry'),
('blob-sidecar-priority','pkg/core/block/inclusion_queue.go','tx.HasDependencies8504() || len(tx.BlobHashes()) > 0','tx.HasDependencies8504()','./node','TestGroupEBaseFOCILBlobValidityAvailabilityAndSameIDRetry'),
('malformed-proof-tolerance','pkg/node/backend/engine_bogota.go','ok && frame.ValidateWithExtensions(bc.Config().FrameExtensions(payload.Time())) != nil','false && ok && frame.ValidateWithExtensions(bc.Config().FrameExtensions(payload.Time())) != nil','./node','TestGroupECanonicalPQILUsesAuthenticatedPoolWitness'),
])
controls.append(('append-header-metadata','pkg/core/types/block_append_size.go','if header.BlobGasUsed != nil {','if false && header.BlobGasUsed != nil {','./core/types','TestGroupEExactAppendRLPAllPrefixAndGasBoundaries'))
controls.extend([
('finalized-verdict','pkg/core/chain/bogota_payload.go','if len(record.Block) == 0 {','if false && len(record.Block) == 0 {','./node','TestBogotaDeferredWireAcrossRestart'),
('reconfigured-authentication','pkg/node/heze_inclusion.go','store, err = heze.Restore(data, view)\n\t\tif err != nil {','store = n.hezeLists\n\t\terr = nil\n\t\tif err != nil {','./node','TestGroupEHezeReconfigureReauthenticatesMemoryAtomically'),
])

for name,path,old,new,pkg,test in controls:
 s=(R/path).read_text()
 if old not in s and name=='strict-timeliness':
  import re
  old=re.search(r'DueBasisPoints\s*= 7500',s).group();new=old.replace('7500','7501')
 if s.count(old)!=1:raise ValueError(('control target must be unique',name,s.count(old)))
 replacement=D/(name+'.go');replacement.write_text(s.replace(old,new));overlay=D/(name+'.json');overlay.write_text(json.dumps({'Replace':{str(R/path):str(replacement)}}))
 log=D/(name+'.jsonl')
 with log.open('w')as f:
  result=subprocess.run(['env','-u','GOROOT','GOFLAGS=-p=4','go','test','-json','-overlay',str(overlay),pkg,'-run','^'+test+'$','-count=1'],cwd=R/'pkg',stdout=f,stderr=subprocess.STDOUT)
 events=[json.loads(l)for l in log.read_text().splitlines()]
 failures=[e for e in events if e['Action']=='fail'and e.get('Test')==test]
 if result.returncode!=1 or not failures or any(e['Action']=='build-fail'for e in events):raise ValueError(('control did not compile and fail its intended consumer',name,result.returncode))
 print(name,'detected',flush=True)

#!/usr/bin/env python3
"""Verify the unchanged, state-authenticated signed CL route after EL retirement."""
import argparse,copy,io,json,sys
from pathlib import Path
from eth_consensus_specs.gloas import mainnet as spec
from eth_consensus_specs.utils import bls
sys.path.insert(0,str(Path(__file__).resolve().parents[3]/"pkg/devnet/kurtosis/qualification/reference"))
from ssz_rest import parsed
p=argparse.ArgumentParser();p.add_argument("--capture",type=Path,required=True);p.add_argument("--output",type=Path,required=True);args=p.parse_args();assert not args.output.exists()
d=args.capture/"final-blob";anchor=json.loads((d/"anchor.json").read_text());bundle=json.loads((d/"bundle.json").read_text());node=bundle["nodes"]["cl-1-lighthouse-geth"]
block=parsed(node["block"]["data"],spec.SignedBeaconBlock)
parent=parsed(json.loads((d/"parent-header.json").read_text())["header"],spec.SignedBeaconBlockHeader)
raw=(d/"parent-state.ssz").read_bytes();state=spec.BeaconState.deserialize(io.BytesIO(raw),len(raw))
assert bls.bls_active and spec.hash_tree_root(state)==parent.message.state_root
assert spec.hash_tree_root(parent.message)==block.message.parent_root
assert "0x"+bytes(spec.hash_tree_root(block.message)).hex()==anchor["beacon_root"]
signed=block.message.body.signed_execution_payload_bid;index=int(signed.message.builder_index)
assert spec.verify_execution_payload_bid_signature(state,signed)
assert spec.is_active_builder(state,index) and state.builders[index].version==spec.PAYLOAD_BUILDER_VERSION
assert spec.can_builder_cover_bid(state,index,signed.message.value)
# SSZ round-trip must preserve the signed message and every commitment exactly.
stream=io.BytesIO();signed.serialize(stream);wire=stream.getvalue();decoded=spec.SignedExecutionPayloadBid.deserialize(io.BytesIO(wire),len(wire));assert bytes(decoded.signature)==bytes(signed.signature) and spec.hash_tree_root(decoded)==spec.hash_tree_root(signed)
rejected=[]
def verify(st,bid):
 try:return spec.verify_execution_payload_bid_signature(st,bid)
 except (AssertionError,ValueError,RuntimeError):return False
for field in ["value","execution_payment","gas_limit"]:
 changed=copy.deepcopy(signed);setattr(changed.message,field,int(getattr(changed.message,field))^1);assert not verify(state,changed);rejected.append(field)
changed=copy.deepcopy(signed);changed.message.execution_requests_root=spec.Root(b"\x01"*32);assert changed.message.execution_requests_root!=signed.message.execution_requests_root and not verify(state,changed);rejected.append("execution_requests_root")
changed=copy.deepcopy(signed);changed.signature=spec.BLSSignature(b"\x00"*96);assert not verify(state,changed);rejected.append("zero_signature")
changed=copy.deepcopy(signed);sig=bytearray(bytes(changed.signature));sig[0]^=1;changed.signature=spec.BLSSignature(sig);assert not verify(state,changed);rejected.append("mutated_signature")
changed=copy.deepcopy(signed);changed.message.blob_kzg_commitments[0],changed.message.blob_kzg_commitments[1]=changed.message.blob_kzg_commitments[1],changed.message.blob_kzg_commitments[0];assert not verify(state,changed);rejected.append("commitment_order")
changed=copy.deepcopy(state);changed.builders[index].pubkey=state.validators[0].pubkey;assert not verify(changed,signed);rejected.append("changed_registry_key")
changed=copy.deepcopy(state);changed.fork.current_version=spec.Version(b"\x01\x00\x00\x00");assert not verify(changed,signed);rejected.append("wrong_fork_domain")
changed=copy.deepcopy(state);changed.genesis_validators_root=spec.Root(b"\x02"*32);assert not verify(changed,signed);rejected.append("wrong_genesis_domain")
changed=copy.deepcopy(state);changed.builders[index].deposit_epoch=changed.finalized_checkpoint.epoch;assert not spec.is_active_builder(changed,index);rejected.append("unfinalized_deposit")
changed=copy.deepcopy(state);changed.builders[index].withdrawable_epoch=spec.Epoch(0);assert not spec.is_active_builder(changed,index);rejected.append("exited_builder")
changed=copy.deepcopy(state);changed.builders[index].balance=spec.Gwei(0);assert not spec.can_builder_cover_bid(changed,index,signed.message.value);rejected.append("no_collateral")
assert verify(state,signed) and spec.is_active_builder(state,index) and spec.can_builder_cover_bid(state,index,signed.message.value)
report={"passed":True,"reference":"2ea9d96a420927eb1a35d7c9399c8b3691c69f6a","bls_active":True,"original_slot":anchor["slot"],"builder_index":index,"original_parent_state_root":"0x"+bytes(parent.message.state_root).hex(),"positive_signature_and_active_funded_registry":True,"signed_ssz_roundtrip_preserved":True,"rejected_controls":rejected,"scope":"Original independently-finalized captured CL state and signed bid through unchanged reference rules. Companion captured finality replay is required separately. No new live process result or EL-side collateral authority is invented."};args.output.write_text(json.dumps(report,indent=2)+"\n");print(json.dumps(report))

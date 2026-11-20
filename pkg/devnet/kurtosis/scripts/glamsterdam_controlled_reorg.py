#!/usr/bin/env python3
import os,json,time,subprocess,urllib.request,urllib.error,importlib.util,hashlib
from pathlib import Path
import argparse,uuid
import glamsterdam_campaign as c
parser=argparse.ArgumentParser(description="Run one private validator-free observer reorg without altering signed messages.")
parser.add_argument("--output",type=Path,required=True)
parser.add_argument("--builder-binary",type=Path,required=True)
parser.add_argument("--image",default="eth2030-cl-reveal-control:qualification")
args=parser.parse_args()
out=args.output;out.mkdir(exist_ok=False);tag=uuid.uuid4().hex[:12];control="eth2030-qualified-reorg-"+tag;kt="kt-eth2030-glamsterdam-completion-20261119";nodes=c.inventory();c.assert_private_network(nodes)
def info(cid):return json.loads(subprocess.check_output(["docker","inspect",cid],text=True))[0]
def role(name):
 ids=subprocess.check_output(["docker","ps","-q","--filter","label=kurtosis_service_name="+name],text=True).split();assert len(ids)==1;return ids[0]
def run(*args,input=None,timeout=60):return subprocess.run(["docker",*args],input=input,capture_output=True,check=True,timeout=timeout)
aux_cl=role("glamsterdam-qualified-snap-cl");aux_el=role("glamsterdam-qualified-snap-el");builder=nodes["buildoor"]["container"];source=nodes["cl-1-lighthouse-geth"]["container"];old_network=info(aux_cl)["NetworkSettings"]["Networks"][kt];source_ip=info(source)["NetworkSettings"]["Networks"][kt]["IPAddress"];observer_original="http://"+old_network["IPAddress"]+":4000";builder_original=args.builder_binary
trace=(out/"exchanges.jsonl").open("x")
def builder_url():return "http://127.0.0.1:"+info(builder)["NetworkSettings"]["Ports"]["8080/tcp"][0]["HostPort"]
base=builder_url()
def get(url):
 r=c.exchange(url);trace.write(json.dumps(r)+"\n");trace.flush();return c.decoded(r)
def post_plan(value):
 req=urllib.request.Request(base+"/api/v1/action-plan",json.dumps(value).encode(),{"Content-Type":"application/json"},method="POST")
 try:
  with c.OPENER.open(req,timeout=10) as response:body=response.read().decode();status=response.status
 except urllib.error.HTTPError as error:body=error.read().decode();status=error.code
 trace.write(json.dumps({"at":c.now(),"action":"set-single-slot-plan","request":value,"status":status,"response":body})+"\n");trace.flush();assert status==200,(status,body)
def restore_settings():
 current=get(base+"/api/v1/config")
 if current["lifecycle_enabled"]!=original_config["lifecycle_enabled"]:
  value={"lifecycle_enabled":original_config["lifecycle_enabled"]};req=urllib.request.Request(base+"/api/v1/config",json.dumps(value).encode(),{"Content-Type":"application/json"},method="PATCH")
  with c.OPENER.open(req,timeout=10) as response:body=response.read().decode();status=response.status
  trace.write(json.dumps({"at":c.now(),"action":"restore-original-runtime-settings","request":value,"status":status,"response":body})+"\n");trace.flush();assert status==200
 assert get(base+"/api/v1/config")==original_config
original_config=get(base+"/api/v1/config");original_status=get(base+"/api/v1/status");assert get(base+"/api/v1/action-plan/rules")["rules"]==[];clock=int(original_status["current_slot"]);assert get(base+f"/api/v1/action-plan?min_slot={clock}&max_slot={clock+64}")["plans"]==[]
health=get(observer_original+"/eth/v1/node/syncing")["data"];assert not any(health[k] for k in ["is_syncing","is_optimistic","el_offline"]) and int(health["sync_distance"])<=1
run("cp",builder+":/usr/local/bin/buildoor",str(out/"buildoor-before"));assert hashlib.file_digest((out/"buildoor-before").open("rb"),"sha256").hexdigest()==hashlib.file_digest(builder_original.open("rb"),"sha256").hexdigest()
profile={"declared_at":c.now(),"scope":"One isolated validator-free observer, unmodified signed SSZ Contents and mandatory original headers, original real builder key, ordinary main-network delivery 3 seconds later. No consensus, BLS, KZG, state/hash check or circuit breaker is disabled.","observer_cl":aux_cl,"observer_el":aux_el,"source_cl":source,"builder":builder,"original_network":old_network,"control_network":control,"original_config":original_config,"original_builder_status":original_status,"preflight_observer":health,"window_seconds_max":300,"instrumentation":"Temporarily wrap the unchanged builder executable only to override its literal-IP CL endpoint. Runtime settings and original executable are restored afterwards. Original slot-result artifacts were captured before restart.","steady_window_started":False};(out/"profile.json").write_text(json.dumps(profile,indent=2)+"\n")
connected=[];isolated=False;proxy=None;network_created=False;builder_changed=False
try:
 assert not subprocess.check_output(["docker","network","ls","-q","--filter","name=^"+control+"$"],text=True).strip()
 run("network","create","--label","eth2030.qualification=controlled-reorg",control);network_created=True
 for cid,alias in [(aux_el,"glamsterdam-qualified-snap-el"),(aux_cl,"glamsterdam-qualified-snap-cl")]:run("network","connect","--alias",alias,control,cid);connected.append(cid)
 run("network","disconnect",kt,aux_cl);isolated=True;observer_ip=info(aux_cl)["NetworkSettings"]["Networks"][control]["IPAddress"];observer="http://"+observer_ip+":4000"
 slot=1;profile.update(observer_url=observer,preference_warmup_seconds_max=180);(out/"profile.json").write_text(json.dumps(profile,indent=2)+"\n")
 proxy=run("run","-d","--name","eth2030-reveal-control-"+tag,"--label","eth2030.qualification=controlled-reorg","--network",kt,"--user",str(os.getuid())+":"+str(os.getgid()),"--mount","type=bind,source="+str(out)+",target=/evidence",args.image,"--upstream=http://"+source_ip+":4000","--observer="+observer,"--slot="+str(slot),"--output=/evidence/proxy-exchanges.jsonl","--slot-file=/evidence/selected-slot.txt").stdout.decode().strip();run("network","connect",control,proxy);proxy_ip=info(proxy)["NetworkSettings"]["Networks"][kt]["IPAddress"]
 wrapper=out/"buildoor-start";wrapper.write_text('#!/bin/sh\nset -eu\nexec /usr/local/bin/buildoor-original-aa3-qualification "$@" --cl-client=http://'+proxy_ip+':4000\n');wrapper.chmod(0o755)
 run("cp",str(builder_original),builder+":/usr/local/bin/buildoor-original-aa3-qualification");run("stop","--time","45",builder);run("cp",str(wrapper),builder+":/usr/local/bin/buildoor");builder_changed=True;run("start",builder);base=builder_url();ready=time.monotonic()+60
 while time.monotonic()<ready:
  try:
   status=get(base+"/api/v1/status")
   if status["running"] and status["is_registered"] and status["builder_pubkey"]==original_status["builder_pubkey"]:break
  except Exception:pass
  time.sleep(1)
 else:raise RuntimeError("same builder identity did not resume")
 restore_settings()
 args=subprocess.check_output(["docker","top",builder,"-eo","pid,args"],text=True);assert "--cl-client=http://"+proxy_ip+":4000" in args
 warm=time.monotonic()+180
 while time.monotonic()<warm:
  now=int(get(base+"/api/v1/status")["current_slot"]);prefs=get(base+"/api/v1/registrations/proposer-preferences")["preferences"];epoch=int(get("http://"+source_ip+":4000/eth/v1/node/syncing")["data"]["head_slot"])//32;duties=[]
  for e in [epoch,epoch+1]:duties+=get("http://"+source_ip+f":4000/eth/v1/validator/duties/proposer/{e}")["data"]
  eligible={int(d["slot"]):int(d["validator_index"]) for d in duties if int(d["validator_index"])<128 and int(d["slot"])>=now+8};cached=[p for p in prefs if int(p["slot"]) in eligible and int(p["validator_index"])==eligible[int(p["slot"])]]
  if cached:
   chosen=min(cached,key=lambda p:int(p["slot"]));slot=int(chosen["slot"]);profile.update(slot=slot,actual_cached_proposer_preference=chosen,preference_selected_at=c.now());(out/"profile.json").write_text(json.dumps(profile,indent=2)+"\n");temp=out/"selected-slot.tmp";temp.write_text(str(slot)+"\n");temp.replace(out/"selected-slot.txt");break
  time.sleep(1)
 else:raise RuntimeError("producer did not receive an actual eligible signed-preference event within warm-up budget")
 actual_clock=int(get(base+"/api/v1/status")["current_slot"]);assert slot>=actual_clock+3
 plan={"updates":[{"slots":[slot],"bid":{"mode":"custom","bid_value_gwei":1000000000,"bids_per_key":1},"builder_api":{"mode":"custom","total_value_override_gwei":1000000000},"reveal":{"mode":"custom","gate_mode":"time","reveal_time_ms":1250,"targets":[{"urls":["http://cl-1-lighthouse-geth:4000","http://cl-2-lodestar-geth:4000","http://cl-3-lighthouse-geth:4000"],"delay_ms":3000}]}}]};post_plan(plan)
 genesis=get("http://"+source_ip+":4000/eth/v1/beacon/genesis")["data"];epoch_time=int(genesis["genesis_time"]);rpc=c.endpoint({"ports":info(aux_el)["NetworkSettings"]["Ports"]},"8545/tcp");samples=(out/"observations.jsonl").open("x");start=time.monotonic();last=None;reorg=[];heads={};beacons={}
 while time.monotonic()-start<300:
  clock=int((time.time()-epoch_time)//3);sample={"at":c.now(),"clock_slot":clock,"el":c.exchange(rpc,"eth_getBlockByNumber",["latest",False]),"observer_syncing":c.exchange(observer+"/eth/v1/node/syncing")}
  try:
   head=c.decoded(sample["el"])["result"]
   if last and head["hash"]!=last["hash"] and int(head["number"],16)==int(last["number"],16):reorg.append({"at":c.now(),"height":head["number"],"old":last,"new":head})
   last=head;heads[head["hash"]]=head
  except Exception:pass
  samples.write(json.dumps(sample)+"\n");samples.flush()
  if slot-1<=clock<=slot+4:
   for n in range(max(slot-1,clock-1),min(slot+4,clock)+1):
    if n in beacons:continue
    try:beacons[n]=get("http://"+source_ip+":4000/eth/v2/beacon/blocks/"+str(n));(out/("block-"+str(n)+".json")).write_text(json.dumps(beacons[n])+"\n")
    except Exception:pass
  if clock>slot+5:break
  time.sleep(.05 if clock>=slot-1 else .5)
 (out/"same-height-reorgs.json").write_text(json.dumps(reorg,indent=2)+"\n");(out/"seen-execution-heads.json").write_text(json.dumps(heads,indent=2)+"\n")
 result=get(base+f"/api/v1/slot-results?min_slot={slot-1}&max_slot={slot+5}");(out/"slot-results.json").write_text(json.dumps(result,indent=2)+"\n")
 for kind in ["payload","envelope","bids"]:
  r=c.exchange(base+f"/api/v1/slot-results/{slot}/{kind}");trace.write(json.dumps(r)+"\n");trace.flush()
  if r.get("http_status")==200:(out/("builder-"+kind+".json")).write_text(r["response"])
 (out/"capture-result.json").write_text(json.dumps({"captured":True,"same_height_reorgs":len(reorg),"slot":slot,"acceptance_evaluated":False},indent=2)+"\n");print("controlled reorg observations",len(reorg),flush=True)
finally:
 if builder_changed:
  run("stop","--time","45",builder);run("cp",str(out/"buildoor-before"),builder+":/usr/local/bin/buildoor");run("start",builder);base=builder_url()
  restore_deadline=time.monotonic()+45
  while True:
   try:restore_settings();break
   except Exception:
    if time.monotonic()>restore_deadline:raise
    time.sleep(1)
 if isolated:run("network","connect","--ip",old_network["IPAddress"],"--alias","glamsterdam-qualified-snap-cl",kt,aux_cl)
 if proxy:run("rm","-f",proxy)
 for cid in reversed(connected):run("network","disconnect",control,cid)
 if network_created:run("network","rm",control)
 trace.close();(out/"restored.json").write_text(json.dumps({"at":c.now(),"original_observer_ip":info(aux_cl)["NetworkSettings"]["Networks"][kt]["IPAddress"],"builder_executable_restored":builder_changed,"original_runtime_settings_restored":True},indent=2)+"\n")

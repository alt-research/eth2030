"""Observe an owned validator-free pair; no result is a full-fork release claim."""
import argparse, hashlib, json, shutil, subprocess, time, urllib.request
from datetime import datetime, timezone
from pathlib import Path

def utc(): return datetime.now(timezone.utc).isoformat()
def docker(*args): return subprocess.check_output(["docker", *args], text=True).strip()
def inspect(cid): return json.loads(docker("inspect",cid))[0]
def endpoint(info,port): return "http://127.0.0.1:"+info["NetworkSettings"]["Ports"][str(port)+"/tcp"][0]["HostPort"]
def request(url,method=None,params=None):
 data=None if method is None else json.dumps({"jsonrpc":"2.0","id":1,"method":method,"params":params}).encode()
 req=urllib.request.Request(url,data=data,headers={"Content-Type":"application/json"})
 start=time.monotonic()
 with urllib.request.urlopen(req,timeout=10) as response: value=json.load(response)
 row={"at":utc(),"url":url,"method":method,"params":params,"elapsed_seconds":time.monotonic()-start,"response":value}
 with (output/"exchanges.jsonl").open("a") as f:f.write(json.dumps(row)+"\n")
 if method and "error" in value:raise RuntimeError("RPC error "+json.dumps(value["error"]))
 return value if method is None else value["result"]
def resources(role,info):
 assert info["State"]["Running"] and not info["State"]["OOMKilled"]
 assert info["HostConfig"]["Memory"]==4096000000 and info["HostConfig"]["NanoCpus"]==2000000000
 assert info["RestartCount"]==0 and info["State"]["StartedAt"]==identities[role]["started"]
 raw=docker("exec",info["Id"],"cat","/proc/1/status","/sys/fs/cgroup/memory.events","/sys/fs/cgroup/memory.max","/sys/fs/cgroup/memory.peak")
 events={k:int(v) for k,v in (line.split() for line in raw.splitlines() if len(line.split())==2 and line.split()[0] in ["oom","oom_kill","oom_group_kill","max"])}
 assert events["oom"]==events["oom_kill"]==0
 numbers=[int(line) for line in raw.splitlines() if line.isdecimal()]
 assert numbers[-2]==4096000000
 hwm=int(next(line for line in raw.splitlines() if line.startswith("VmHWM:")).split()[1])*1024
 assert hwm<4096000000
 return {"events":events,"memory_max":numbers[-2],"memory_peak":numbers[-1],"process_hwm":hwm}
def identify(role,info):
 actual=docker("exec",info["Id"],"sha256sum","/proc/1/exe").split()[0]
 expected=profile["binary_sha256"] if role=="el" else profile["cl_binary_sha256"]
 assert actual==expected,(role,actual,expected)
 return {"source":profile["source"] if role=="el" else profile["cl_source"],"binary_sha256":actual,"started":info["State"]["StartedAt"],"image":info["Image"]}
def sample(prove=False):
 info={role:inspect(record[role]) for role in ["el","cl"]}
 mem={role:resources(role,info[role]) for role in info}
 assert shutil.disk_usage(output).free>20*(1<<30),"shared filesystem below predeclared20GiB stop margin"
 sync=request(clurl+"/eth/v1/node/syncing")["data"]
 if sync["is_syncing"] or sync["is_optimistic"] or sync["el_offline"]:return {"ready":False,"sync":sync,"memory":mem}
 header=request(elurl,"eth_getBlockByNumber",["finalized",False]); assert header and int(header["number"],16)>0
 ref=request(gethurl,"eth_getBlockByNumber",[header["number"],False]); assert ref and ref["hash"]==header["hash"] and ref["stateRoot"]==header["stateRoot"],"independent finalized execution disagreement"
 tip=request(gethurl,"eth_getBlockByNumber",["finalized",False]);lag=int(tip["number"],16)-int(header["number"],16)
 assert 0<=lag<=128,("finalized lag",lag)
 if prove:
  proofpath=output/("proofs-"+str(len(list(output.glob("proofs-*.jsonl"))))+".jsonl")
  rows=[]
  for address in ["0x7e5f4552091a69125d5dfcb7b8c2659029395bdf","0xee9ede56b61f213e69233a001b17fd6414c93675"]:
   params=[address,[],{"blockHash":header["hash"],"requireCanonical":True}]
   proof=request(elurl,"eth_getProof",params)
   rows.append({"kind":"proof","service":"isolated48h-observer","expected_hash":header["hash"],"header":header,"request":{"method":"eth_getProof","params":params},"response":{"jsonrpc":"2.0","id":1,"result":proof}})
  proofpath.write_text("".join(json.dumps(row)+"\n" for row in rows))
  p=subprocess.run([str(args.ledger),"proof-stream",str(proofpath)],capture_output=True,text=True,check=True)
  checked=json.loads(p.stdout); assert checked["passed"] and checked["independent_proofs"]==2
  (proofpath.with_suffix(".checked.json")).write_text(json.dumps(checked)+"\n")
 return {"ready":True,"sync":sync,"memory":mem,"finalized_hash":header["hash"],"finalized_root":header["stateRoot"],"finalized_number":int(header["number"],16),"reference_lag":lag}

if __name__=="__main__":
 parser=argparse.ArgumentParser(description=__doc__);parser.add_argument("--record",type=Path,required=True);parser.add_argument("--ledger",type=Path,required=True);parser.add_argument("--output",type=Path,required=True);args=parser.parse_args()
 record=json.loads(args.record.read_text());profile=record["profile"];output=args.output;output.mkdir(exist_ok=False)
 assert profile["duration_seconds"]==172800 and profile["catchup_budget_seconds"]==3600 and profile["sampling_seconds"]==30
 infos={role:inspect(record[role]) for role in ["el","cl"]}
 for info in infos.values():assert info["Config"]["Labels"]["eth2030.qualification"]=="remaining-code-20261122"
 identities={role:identify(role,info) for role,info in infos.items()};(output/"identities.json").write_text(json.dumps(identities,indent=2)+"\n")
 elurl=endpoint(infos["el"],8545);clurl=endpoint(infos["cl"],4000);gethurl=record["geth_rpc"]
 report={"passed":False,"status":"catchup","started_at":utc(),"profile":profile,"scope":"48-hour owned observer operating profile only; not public/mainnet capacity or independent BLS finality across the entire span."};(output/"result.json").write_text(json.dumps(report,indent=2)+"\n")
 try:
  deadline=time.monotonic()+3600
  while True:
   try: row=sample()
   except Exception as error:row={"ready":False,"error":type(error).__name__+": "+str(error)}
   with (output/"catchup.jsonl").open("a") as f:f.write(json.dumps({"at":utc(),**row})+"\n")
   if row.get("ready"):break
   if time.monotonic()>=deadline:raise TimeoutError("catchup did not qualify within3600s")
   time.sleep(30)
  start=time.monotonic();first=sample(True);last=first["finalized_number"];advanced=start
  report.update(status="running",accepted_started_at=utc(),initial=first,samples=0,elapsed_seconds=0)
  (output/"result.json").write_text(json.dumps(report,indent=2)+"\n")
  count=0
  while True:
   time.sleep(30);count+=1;row=sample(count%10==0);assert row["ready"],"accepted observer became syncing/optimistic/offline"
   elapsed=time.monotonic()-start
   if row["finalized_number"]>last:advanced=time.monotonic();last=row["finalized_number"]
   assert time.monotonic()-advanced<600,"finalized execution did not advance within600s"
   if count%20==0:
    for role in infos:assert identify(role,inspect(record[role]))==identities[role]
   with (output/"samples.jsonl").open("a") as f:f.write(json.dumps({"at":utc(),"elapsed_seconds":elapsed,**row})+"\n")
   report.update(samples=count,elapsed_seconds=elapsed,last=row,updated_at=utc());(output/"result.json").write_text(json.dumps(report,indent=2)+"\n")
   if elapsed>=172800:break
  assert last>first["finalized_number"];sample(True);report.update(passed=True,status="completed operating profile; final span BLS replay remains separate",finished_at=utc())
 except Exception as error:
  report.update(status="failed",failure={"class":type(error).__name__,"message":str(error)},finished_at=utc());raise
 finally:
  for role in ["cl","el"]:
   info=inspect(record[role]);assert info["Config"]["Labels"]["eth2030.qualification"]=="remaining-code-20261122"
   subprocess.run(["docker","stop","--time","30",record[role]],capture_output=True)
   logs=subprocess.run(["docker","logs","--tail","500",record[role]],capture_output=True,text=True);(output/(role+"-final.log")).write_text(logs.stdout+logs.stderr)
  (output/"result.json").write_text(json.dumps(report,indent=2)+"\n")

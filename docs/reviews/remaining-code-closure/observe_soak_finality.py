"""Collect bounded contiguous signed finality windows during an owned soak."""
import argparse,json,subprocess,time
from datetime import datetime,timezone
from pathlib import Path

def now():return datetime.now(timezone.utc).isoformat()
if __name__=="__main__":
 p=argparse.ArgumentParser(description=__doc__);p.add_argument("--operating-result",type=Path,required=True);p.add_argument("--reference-python",type=Path,required=True);p.add_argument("--verifier",type=Path,required=True);p.add_argument("--trusted-bootstrap",type=Path,required=True);p.add_argument("--output",type=Path,required=True);args=p.parse_args()
 out=args.output;out.mkdir(exist_ok=False);report={"passed":False,"status":"waiting","windows":[],"scope":"Contiguous signed finality for the accepted observed slot span, using unchanged reference with its4096-link bound. The operating controller remains an independent required gate."};next_slot=None;last_capture=0
 def save():(out/"result.json").write_text(json.dumps(report,indent=2)+"\n")
 try:
  save()
  while True:
   try:operating=json.loads(args.operating_result.read_text())
   except json.JSONDecodeError:time.sleep(1);continue
   if operating["status"]=="failed":raise RuntimeError("operating gate failed; do not qualify its duration")
   if "initial" not in operating:time.sleep(30);continue
   if next_slot is None:next_slot=int(operating["initial"]["sync"]["head_slot"]);report["first_slot"]=next_slot
   last=int(operating.get("last",operating["initial"])["sync"]["head_slot"])
   complete=operating["passed"]
   through=last if complete else last-128
   due=complete or time.monotonic()-last_capture>=3600
   if due and through>=next_slot:
    directory=out/("window-"+str(len(report["windows"])))
    cmd=[str(args.reference_python),str(args.verifier),"--capture",str(directory),"--trusted-bootstrap",str(args.trusted_bootstrap),"--authenticate-from",str(next_slot),"--authenticate-through",str(through)]
    with (out/(directory.name+".log")).open("x") as log:subprocess.run(cmd,stdout=log,stderr=subprocess.STDOUT,check=True,timeout=900)
    live=json.loads((directory/"finality-result.json").read_text());assert live["passed"] and live["bls_active"]
    replay=out/(directory.name+"-replay.json")
    with (out/(directory.name+"-replay.log")).open("x") as log:subprocess.run([str(args.reference_python),str(args.verifier),"--capture",str(directory),"--trusted-bootstrap",str(args.trusted_bootstrap),"--replay","--output",str(replay)],stdout=log,stderr=subprocess.STDOUT,check=True,timeout=120)
    offline=json.loads(replay.read_text());assert offline["passed"] and offline["bls_active"]
    window=live["beacon_window"];assert window["first"]==next_slot and window["stop"]==through+1
    report["windows"].append({"directory":str(directory),"first":next_slot,"through":through,"live_passed":True,"replay_passed":True,"participants":window["signature_participants"],"links":window["parent_hash_links"]});next_slot=through+1;last_capture=time.monotonic();report.update(status="collecting",covered_through=through,updated_at=now());save()
   if complete and next_slot>last:report.update(passed=True,status="completed",finished_at=now());save();break
   time.sleep(30)
 except Exception as error:report.update(status="failed",failure={"class":type(error).__name__,"message":str(error)},finished_at=now());save();raise

#!/usr/bin/env python3
"""Supervise the existing collectors without restarting the measured daemons."""

import copy
import json
import subprocess
import time
from datetime import datetime, timezone
from pathlib import Path

import glamsterdam_soak_audit as evidence


def stop_owned(record):
    results = {}
    for role in ("cl", "el"):
        try:
            info = evidence.run_json(["docker", "inspect", record[role]], 15)[0]
            evidence.require(info["Config"]["Labels"].get("eth2030.qualification") == evidence.LABEL,
                             "refusing cleanup of an unowned container")
            stopped = subprocess.run(["docker", "stop", "--time", "30", record[role]],
                                     capture_output=True, text=True, timeout=40)
            final = evidence.run_json(["docker", "inspect", record[role]], 15)[0]["State"]
            results[role] = {"stop_exit": stopped.returncode, "state": final,
                             "clean": stopped.returncode == 0 and not final["Running"]}
        except Exception as error:
            results[role] = {"clean": False, "error": str(error)}
    return results


def supervise(args):
    args.output.mkdir(parents=True, exist_ok=False)
    # Freeze the allowed container IDs; a later edited record cannot redirect
    # cleanup to the original producer or validator fleet.
    original_record = args.record.read_bytes()
    record = json.loads(original_record)
    evidence.write_json(args.output / "input-record.json", record)
    result = {"passed": False, "status": "auditing", "started_at": evidence.utc()}
    evidence.write_json(args.output / "result.json", result)
    cache = {}
    audit_args = copy.copy(args)
    audit_args.output = args.output / "audit"
    try:
        while True:
            evidence.require(args.record.read_bytes() == original_record, "owned container record changed")
            operating = evidence.read_json(args.run / "result.json")
            finality = evidence.read_json(args.finality / "result.json")
            evidence.require(operating.get("status") != "failed", "operating controller failed")
            evidence.require(finality.get("status") != "failed", "signed finality collector failed")
            now = datetime.now(timezone.utc).timestamp()
            if "accepted_started_at" not in operating:
                evidence.require(now - evidence.timestamp(operating["started_at"]) <= 3600,
                                 "catchup acceptance deadline expired")
                time.sleep(30)
                continue
            started = evidence.timestamp(operating["accepted_started_at"])
            if not operating.get("passed") and operating.get("samples", 0) == 0:
                evidence.require(now - started <= evidence.MAX_SAMPLE_GAP,
                                 "first accepted sample did not arrive within60s")
                time.sleep(1)
                continue
            # The legacy process may be shutting down before writing its final
            # result. Allow that bounded transition, never turn it into a pass.
            if not operating.get("passed") and now >= started + evidence.DURATION:
                evidence.require(now <= started + evidence.DURATION + 120,
                                 "operating completion/cleanup did not finish within120s")
                time.sleep(10)
                continue
            evidence.require(now <= started + evidence.DURATION + 1200,
                             "final signed closure exceeded the post-duration deadline")
            if finality.get("windows"):
                evidence.require(now - evidence.timestamp(finality["updated_at"]) <= 4800,
                                 "signed finality collector stopped extending its hourly coverage")
            else:
                evidence.require(now - started <= 1200,
                                 "first signed finality window did not arrive within1200s")
            audit_args.complete = operating.get("passed") is True and finality.get("passed") is True
            checked = evidence.audit(audit_args, cache)
            result.update(status="complete" if audit_args.complete else "running",
                          updated_at=evidence.utc(), evidence=checked,
                          passed=checked["passed"] and checked["cleanup_verified"])
            evidence.write_json(args.output / "result.json", result)
            with (args.output / "history.jsonl").open("a") as history:
                history.write(json.dumps({"at": result["updated_at"], "samples": checked["operating"]["samples"],
                                          "seconds": checked["operating"]["observed_seconds"],
                                          "covered_through": checked["covered_through"],
                                          "complete": result["passed"]}) + "\n")
            if result["passed"]:
                return result
            time.sleep(30)
    except Exception as error:
        result.update(passed=False, status="failed", finished_at=evidence.utc(),
                      failure={"class": type(error).__name__, "message": str(error)})
        # A cleanup correction cannot retrospectively turn failed qualification
        # into a pass. Keep the original failure and the actual stop results.
        result["cleanup_after_failure"] = stop_owned(record)
        evidence.write_json(args.output / "result.json", result)
        raise


if __name__ == "__main__":
    print(json.dumps(supervise(evidence.arguments().parse_args())))

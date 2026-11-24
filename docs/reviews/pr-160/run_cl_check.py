"""Compile an external probe against unchanged, pinned actual Lighthouse code.

Arguments: Lighthouse checkout, oracle output directory. Run cl_oracle.py first.
This writes build/probe artifacts only to the output directory and the existing
Cargo target directory; it never edits the CL checkout or reference sources.
"""
import json
import os
from pathlib import Path
import subprocess
import sys

cl, out = map(lambda x: Path(x).resolve(), sys.argv[1:3])
tree = subprocess.check_output(["git", "-C", str(cl), "rev-parse", "HEAD^{tree}"], text=True).strip()
assert tree == "9fba2d606e87e9fb70e49ec13ee7eb08434711f3", tree
assert not subprocess.check_output(["git", "-C", str(cl), "status", "--porcelain"], text=True)
target = Path(os.environ["CARGO_TARGET_DIR"]).resolve() / "release"
env = dict(os.environ, CARGO_BUILD_JOBS="2")
build = subprocess.run(["cargo", "build", "--release", "-p", "types", "--message-format=json"],
                       cwd=cl, env=env, capture_output=True, text=True, check=True)
(out / "cl-library-build.jsonl").write_text(build.stdout)
(out / "cl-library-build.log").write_text(build.stderr)
libs = {}
for line in build.stdout.splitlines():
    event = json.loads(line)
    if event.get("reason") == "compiler-artifact":
        for name in event["filenames"]:
            if name.endswith(".rlib"):
                libs[event["target"]["name"]] = name
cmd = ["rustc", "--edition=2024", "-O", str(Path(__file__).with_name("cl_check.rs")),
       "-o", str(out / "cl-check"), "-L", "dependency=" + str(target / "deps"),
       "--extern", "types=" + libs["types"], "--extern", "serde_json=" + libs["serde_json"]]
for native in (target / "build").glob("*/out"):
    cmd += ["-L", "native=" + str(native)]
compile_result = subprocess.run(cmd, cwd=cl, env=env, capture_output=True, text=True)
(out / "cl-check-compile.log").write_text(compile_result.stdout + compile_result.stderr)
assert compile_result.returncode == 0, compile_result.stderr
oracle_path, state_path = out / "CL_ORACLE.json", out / "cl-captured-state.ssz"
run = subprocess.run([str(out / "cl-check"), str(oracle_path), str(state_path)],
                     capture_output=True, text=True)
(out / "cl-check.log").write_text(run.stdout + run.stderr)
assert run.returncode == 0, run.stderr
checks = json.loads(run.stdout)
checks["lighthouse_tree"] = tree
checks["out_of_decision_window_query_rejected"] = True
(out / "CL_CHECKS.json").write_text(json.dumps(checks, indent=2) + "\n")
original = json.loads(oracle_path.read_text())
controls = []
for kind in ["proposer", "churn"]:
    value = json.loads(json.dumps(original))
    if kind == "proposer":
        value["proposer_cases"][0]["expected"][0] = (value["proposer_cases"][0]["expected"][0] + 1) % 384
    else:
        value["churn_cases"][0]["exit"] += 1
    changed = out / ("cl-wrong-" + kind + ".json")
    changed.write_text(json.dumps(value))
    result = subprocess.run([str(out / "cl-check"), str(changed), str(state_path)], capture_output=True, text=True)
    assert result.returncode != 0 and "assertion" in result.stderr and "failed" in result.stderr, result.stderr
    controls.append({"changed_expectation": kind, "exit_code": result.returncode, "rejected_by_assertion": True})
(out / "CL_NEGATIVE_CONTROLS.json").write_text(json.dumps(controls, indent=2) + "\n")
print(json.dumps({"checks": checks, "negative_controls": controls}))

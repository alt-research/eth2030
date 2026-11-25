#!/usr/bin/env python3
"""Finalize a sponsored canonical Frame through Heze, restart an EL and clean up."""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import signal
from uuid import uuid4

SCRIPTS = Path(__file__).resolve().parent
REPO = SCRIPTS.parents[3]


def load(name, filename):
    spec = importlib.util.spec_from_file_location(name, SCRIPTS / filename)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


control = load("frame_control_base", "run-track4-control.py")
verifier = load("frame_focil_verifier", "verify-heze-focil.py")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--package", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--frame-tool", type=Path, required=True)
    args = parser.parse_args()
    package = args.package.resolve()
    frame_tool = args.frame_tool.resolve()
    if control.run("git", "-C", str(package), "rev-parse", "HEAD") != control.PACKAGE_REV:
        raise ValueError("Frame control requires the pinned ethereum-package")
    control.run("git", "-C", str(package), "diff", "--quiet", "HEAD")
    if control.enclave_networks():
        raise RuntimeError("finish other Kurtosis enclaves before launching this control")
    args.output.mkdir(parents=True, exist_ok=False)
    enclave = f"eth2030-frame-control-{uuid4().hex[:8]}"
    was_running = control.engine_running()
    signal.signal(signal.SIGTERM, control.interrupt)
    evidence = {"enclave": enclave, "package": control.PACKAGE_REV,
                "source": control.run("git", "-C", str(REPO), "rev-parse", "HEAD")}
    try:
        with (args.output / "launch.log").open("w") as log:
            control.run_job(["bash", str(SCRIPTS / "run-devnet.sh"), "frame-heze-control", enclave],
                            log, timeout=420,
                            env={**os.environ, "KURTOSIS_PACKAGE": str(package), "SKIP_EL_IMAGE_BUILD": "1"})
        evidence["enclave_id"] = control.enclave_id(f"kt-{enclave}")
        evidence["nodes"] = control.collect_images(enclave)
        evidence["execution_source"] = control.run("docker", "image", "inspect", "eth2030:frame-control", "--format", '{{index .Config.Labels "commit"}}')
        evidence["consensus_source"] = control.run("docker", "image", "inspect", "eth2030cl:frame-control", "--format", '{{index .Config.Labels "source_commit"}}')
        evidence["result"] = verifier.verify(enclave, 900, frame_tool, args.output)
        result = evidence["result"]
        control.run("kurtosis", "service", "stop", enclave, control.EL[1], timeout=75)
        control.run("kurtosis", "service", "start", enclave, control.EL[1], timeout=75)
        evidence["after_restart"] = control.wait_for(
            lambda: verifier.frame_readback(control.endpoint(enclave, control.EL[1], "rpc"), result["transaction"]),
            120, "restarted EL did not recover the canonical Frame")
        if evidence["after_restart"] != result["frame"]:
            raise RuntimeError("Frame receipt, charge or historical state changed after EL restart")
    finally:
        try:
            control.cleanup(enclave, evidence, was_running)
        finally:
            (args.output / "evidence.json").write_text(json.dumps(evidence, indent=2) + "\n")
    print(f"Verified and removed {enclave}. Evidence: {args.output / 'evidence.json'}")


if __name__ == "__main__":
    main()

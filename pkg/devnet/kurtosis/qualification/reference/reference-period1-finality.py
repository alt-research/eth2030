import argparse
import json, copy, sys
from pathlib import Path
from urllib.parse import urlsplit
from eth_consensus_specs.gloas import mainnet as spec
from eth_consensus_specs.fulu import mainnet as fulu
from eth_consensus_specs.utils import bls

sys.path.insert(0, str(Path(__file__).resolve().parent))
from ssz_rest import parsed

parser = argparse.ArgumentParser(
    description="Replay authentic period-one finality and payment ancestry using the unchanged reference"
)
parser.add_argument("--artifact-dir", required=True, type=Path)
args = parser.parse_args()
root = args.artifact_dir
out = root / "replay-reference-period1-finality.json"
trace_path = root / "replay-reference-period1-finality.jsonl"
assert not out.exists() and not trace_path.exists()
trace = trace_path.open("x")
report = {
    "passed": False,
    "spec_revision": "2ea9d96a420927eb1a35d7c9399c8b3691c69f6a",
    "reference_source_unmodified": True,
    "bls_active": bls.bls_active,
}
captured = {}
for line in (
    (root / "attempt4-reference-light-client-advertised-baseline.jsonl")
    .read_text()
    .splitlines()
):
    record = json.loads(line)
    url = urlsplit(record["url"])
    path = url.path + ("?" + url.query if url.query else "")
    captured.setdefault(path, record)


def old(path):
    r = captured[path]
    assert r["status"] == 200
    return json.loads(r["response_bytes"])


extension = {}
for line in (
    (root / "attempt4-reference-period1-finality.jsonl").read_text().splitlines()
):
    record = json.loads(line)
    extension.setdefault(record["url_path"], []).append(record)
positions = {}


def get(path):
    index = positions.get(path, 0)
    records = extension[path]
    assert index < len(records), "capture exhausted: " + path
    record = records[index]
    positions[path] = index + 1
    trace.write(
        json.dumps({"source": "replayed_frozen_period1_capture", "original": record})
        + "\n"
    )
    trace.flush()
    assert record["status"] == 200
    return json.loads(record["response_bytes"])


def save():
    out.write_text(json.dumps(report, indent=2) + "\n")


def hx(value):
    return "0x" + bytes(value).hex()


def root_of(header):
    return hx(spec.hash_tree_root(header))


try:
    assert bls.bls_active
    network = get("/eth/v1/config/spec")["data"]
    genesis = get("/eth/v1/beacon/genesis")["data"]
    assert genesis == old("/eth/v1/beacon/genesis")["data"]
    updates = {}
    for name, existing in spec.config._asdict().items():
        if name in network and (
            name.endswith(("_FORK_VERSION", "_FORK_EPOCH"))
            or name == "SLOT_DURATION_MS"
        ):
            updates[name] = parsed(network[name], type(existing))
    spec.config = spec.config._replace(**updates)
    trusted = "0x45499f221a955bd11e78aff4bea50928c75e08182e1db28aa453f6c06be950d7"
    trusted_root = spec.Root(bytes.fromhex(trusted[2:]))
    bootstrap = parsed(
        old("/eth/v1/beacon/light_client/bootstrap/" + trusted)["data"],
        spec.LightClientBootstrap,
    )
    store = spec.initialize_light_client_store(trusted_root, bootstrap)
    best = old("/eth/v1/beacon/light_client/updates?start_period=0&count=1")[0]
    if best["version"] == "fulu":
        full = spec.upgrade_lc_update_to_gloas(
            parsed(best["data"], fulu.LightClientUpdate)
        )
    else:
        assert best["version"] == "gloas"
        full = parsed(best["data"], spec.LightClientUpdate)
    candidate = get("/eth/v1/beacon/light_client/finality_update")
    assert candidate["version"] == "gloas"
    current = spec.Slot(int(candidate["data"]["signature_slot"]) + 1)
    gvr = spec.Root(bytes.fromhex(genesis["genesis_validators_root"][2:]))
    spec.process_light_client_update(store, full, current, gvr)
    assert spec.is_next_sync_committee_known(store)
    previous = copy.deepcopy(store)
    update = parsed(candidate["data"], spec.LightClientFinalityUpdate)
    spec.process_light_client_finality_update(store, update, current, gvr)
    bundle = json.loads(
        (root / "attempt4-builder-ledger-fresh-20261119T234834Z.json").read_text()
    )
    required = bundle["after"]["beacon_root"]
    required_slot = bundle["after"]["slot"]
    assert int(store.finalized_header.beacon.slot) >= required_slot
    report["progress"] = {
        "trusted_slot": int(bootstrap.header.beacon.slot),
        "signature_slot": int(update.signature_slot),
        "signature_period": int(update.signature_slot)
        // (int(spec.SLOTS_PER_EPOCH) * int(spec.EPOCHS_PER_SYNC_COMMITTEE_PERIOD)),
        "finalized_slot": int(store.finalized_header.beacon.slot),
        "finalized_root": root_of(store.finalized_header.beacon),
        "participation": sum(
            x.bit_count() for x in bytes(update.sync_aggregate.sync_committee_bits)
        ),
        "next_committee_known_after_rotation": spec.is_next_sync_committee_known(store),
    }
    assert report["progress"]["signature_period"] == 1
    save()
    cursor = store.finalized_header.beacon
    links = 0
    while root_of(cursor) != required:
        assert int(cursor.slot) > required_slot and links < 4096
        parent = hx(cursor.parent_root)
        entry = get("/eth/v1/beacon/headers/" + parent)["data"]
        cursor = parsed(entry["header"]["message"], spec.BeaconBlockHeader)
        assert root_of(cursor) == parent
        links += 1
    report["actual_payment_finalized_ancestor"] = {
        "slot": required_slot,
        "root": required,
        "independent_parent_hash_links": links,
    }
    save()
    controls = []
    for name in [
        "period1_finality_branch",
        "period1_signature",
        "period1_genesis_domain",
    ]:
        altered = copy.deepcopy(candidate["data"])
        domain = gvr
        if name == "period1_finality_branch":
            value = altered["finality_branch"][0]
            altered["finality_branch"][0] = value[:-2] + format(
                int(value[-2:], 16) ^ 1, "02x"
            )
        elif name == "period1_signature":
            value = altered["sync_aggregate"]["sync_committee_signature"]
            altered["sync_aggregate"]["sync_committee_signature"] = value[:-2] + format(
                int(value[-2:], 16) ^ 1, "02x"
            )
        else:
            domain = spec.Root(bytes([bytes(gvr)[0] ^ 1]) + bytes(gvr)[1:])
        try:
            spec.process_light_client_finality_update(
                copy.deepcopy(previous),
                parsed(altered, spec.LightClientFinalityUpdate),
                current,
                domain,
            )
        except (AssertionError, ValueError):
            controls.append(name)
        else:
            raise AssertionError("reference accepted " + name)
    report["rejected_controls"] = controls
    report["passed"] = True
    save()
    print(
        json.dumps(
            {
                "passed": True,
                "progress": report["progress"],
                "ancestor": report["actual_payment_finalized_ancestor"],
                "rejections": controls,
            }
        ),
        flush=True,
    )
except Exception as error:
    report["failure"] = {"type": type(error).__name__, "message": str(error)}
    save()
    raise
finally:
    trace.close()

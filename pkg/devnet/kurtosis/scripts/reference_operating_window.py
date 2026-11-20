#!/usr/bin/env python3
import sys, json, copy, time, math, argparse, hashlib
from urllib.parse import urlsplit
from pathlib import Path
from datetime import datetime, timezone
from eth_consensus_specs.gloas import mainnet as spec
from eth_consensus_specs.fulu import mainnet as fulu
from eth_consensus_specs.utils import bls

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "qualification/reference"))
from ssz_rest import parsed
import glamsterdam_campaign as c

parser = argparse.ArgumentParser(
    description="Authenticate a declared operating window through unchanged light-client reference code."
)
parser.add_argument("--capture", type=Path, required=True)
parser.add_argument("--trusted-bootstrap", type=Path, required=True)
parser.add_argument(
    "--replay",
    action="store_true",
    help="Use original captured exchanges only; do not discover Docker or access a server",
)
parser.add_argument("--output", type=Path)
args = parser.parse_args()
if args.replay and args.output is None:
    parser.error("--replay requires a new --output path")
root = args.capture
profile = json.loads((root / "profile.json").read_text())
latency = json.loads((root / "latency-result.json").read_text())
out = args.output or root / "finality-result.json"
assert not out.exists()
trace_path = (
    out.with_suffix(".exchanges.jsonl")
    if args.replay
    else root / "reference-finality-exchanges.jsonl"
)
trace = trace_path.open("x")
report = {
    "passed": False,
    "spec": "2ea9d96a420927eb1a35d7c9399c8b3691c69f6a",
    "bls_active": bls.bls_active,
    "latency_passed": latency["latency_passed"],
}


def save():
    out.write_text(json.dumps(report, indent=2) + "\n")


def hx(v):
    return "0x" + bytes(v).hex()


replay_clock = None
if args.replay:
    original_trace = root / "reference-finality-exchanges.jsonl"
    original_records = [
        json.loads(line) for line in original_trace.read_text().splitlines()
    ]
    records = iter(
        record
        for record in original_records
        if "original_trusted_bootstrap" not in record
    )
    first_url = urlsplit(original_records[0]["url"])
    base = first_url.scheme + "://" + first_url.netloc
    nodes = None
    report["replay_input_sha256"] = hashlib.sha256(
        original_trace.read_bytes()
    ).hexdigest()
else:
    nodes = c.inventory()
    base = c.endpoint(nodes["cl-1-lighthouse-geth"], "4000/tcp")


def captured_exchange(url):
    global replay_clock
    if args.replay:
        record = next(records)
        wanted, actual = urlsplit(url), urlsplit(record["url"])
        assert (wanted.path, wanted.query) == (actual.path, actual.query)
        replay_clock = datetime.fromisoformat(
            record["started"].replace("Z", "+00:00")
        ).timestamp()
    else:
        record = c.exchange(url)
    trace.write(json.dumps(record) + "\n")
    trace.flush()
    return record


def get(path):
    return c.decoded(captured_exchange(base + path))


try:
    assert bls.bls_active
    network = get("/eth/v1/config/spec")["data"]
    genesis = get("/eth/v1/beacon/genesis")["data"]
    assert (
        genesis["genesis_validators_root"]
        == "0x0888cde6ee0c51578046793b377d6f6474908f7812b8db4d87a736d73e539d2c"
    )
    changes = {
        name: parsed(network[name], type(value))
        for name, value in spec.config._asdict().items()
        if name in network
        and (
            name.endswith(("_FORK_VERSION", "_FORK_EPOCH"))
            or name == "SLOT_DURATION_MS"
        )
    }
    spec.config = spec.config._replace(**changes)
    trusted = "0x45499f221a955bd11e78aff4bea50928c75e08182e1db28aa453f6c06be950d7"
    old = args.trusted_bootstrap
    captured = [json.loads(x) for x in old.read_text().splitlines()]
    original = next(
        x
        for x in captured
        if x.get("status") == 200
        and x["url"].endswith("/light_client/bootstrap/" + trusted)
    )
    trace.write(json.dumps({"original_trusted_bootstrap": original}) + "\n")
    trace.flush()
    bootstrap = parsed(
        json.loads(original["response_bytes"])["data"], spec.LightClientBootstrap
    )
    store = spec.initialize_light_client_store(
        spec.Root(bytes.fromhex(trusted[2:])), bootstrap
    )
    gvr = spec.Root(bytes.fromhex(genesis["genesis_validators_root"][2:]))

    def current_slot():
        return spec.Slot(
            int(
                (
                    (replay_clock if args.replay else time.time())
                    - int(genesis["genesis_time"])
                )
                * 1000
            )
            // int(network["SLOT_DURATION_MS"])
        )

    period = int(current_slot()) // (
        int(spec.SLOTS_PER_EPOCH) * int(spec.EPOCHS_PER_SYNC_COMMITTEE_PERIOD)
    )
    updates = get(
        f"/eth/v1/beacon/light_client/updates?start_period=0&count={period+1}"
    )
    assert len(updates) == period + 1
    for raw in updates:
        full = (
            spec.upgrade_lc_update_to_gloas(parsed(raw["data"], fulu.LightClientUpdate))
            if raw["version"] == "fulu"
            else parsed(raw["data"], spec.LightClientUpdate)
        )
        spec.process_light_client_update(store, full, current_slot(), gvr)
    begin = datetime.fromisoformat(profile["declared_at"]).timestamp()
    end = begin + profile["duration_seconds"]
    first = math.ceil(
        (begin - int(genesis["genesis_time"])) * 1000 / int(network["SLOT_DURATION_MS"])
    )
    stop = math.ceil(
        (end - int(genesis["genesis_time"])) * 1000 / int(network["SLOT_DURATION_MS"])
    )
    assert stop - first == int(
        profile["duration_seconds"] * 1000 / int(network["SLOT_DURATION_MS"])
    )
    deadline = time.monotonic() + 420
    while time.monotonic() < deadline:
        raw = get("/eth/v1/beacon/light_client/finality_update")
        assert raw["version"] == "gloas"
        candidate = parsed(raw["data"], spec.LightClientFinalityUpdate)
        previous = copy.deepcopy(store)
        spec.process_light_client_finality_update(store, candidate, current_slot(), gvr)
        if int(store.finalized_header.beacon.slot) >= stop - 1:
            break
        save()
        if not args.replay:
            time.sleep(3)
    else:
        raise RuntimeError("original 420-second authenticated finality wait failed")
    finalized = store.finalized_header.beacon
    anchor = hx(spec.hash_tree_root(finalized))
    cursor = finalized
    present = []
    links = 0
    while int(cursor.slot) >= first:
        if first <= int(cursor.slot) < stop:
            present.append(int(cursor.slot))
        parent = hx(cursor.parent_root)
        entry = get("/eth/v1/beacon/headers/" + parent)["data"]
        cursor = parsed(entry["header"]["message"], spec.BeaconBlockHeader)
        assert hx(spec.hash_tree_root(cursor)) == parent
        links += 1
        assert links < 4096
    missing = sorted(set(range(first, stop)) - set(present))
    report["beacon_window"] = {
        "first": first,
        "stop": stop,
        "slots": stop - first,
        "finalized_root": anchor,
        "finalized_slot": int(finalized.slot),
        "signature_slot": int(candidate.signature_slot),
        "signature_participants": sum(
            x.bit_count() for x in bytes(candidate.sync_aggregate.sync_committee_bits)
        ),
        "parent_hash_links": links,
        "missing_slots": missing,
        "missing_fraction": len(missing) / (stop - first),
    }
    canonical_urls = []
    for role in ["cl-2-lodestar-geth", "cl-3-lighthouse-geth"]:
        role_base = base if args.replay else c.endpoint(nodes[role], "4000/tcp")
        rec = captured_exchange(role_base + "/eth/v1/beacon/headers/" + anchor)
        canonical_urls.append(rec["url"])
        d = c.decoded(rec)["data"]
        assert d["canonical"] and d["root"] == anchor
    assert len(set(canonical_urls)) == 2 and all(
        not value.startswith(base + "/") for value in canonical_urls
    )
    if args.replay:
        assert next(records, None) is None, "unconsumed original finality exchanges"
    rejects = []
    for name in ["signature", "finality_branch", "genesis_domain"]:
        altered = copy.deepcopy(raw["data"])
        domain = gvr
        if name == "signature":
            v = altered["sync_aggregate"]["sync_committee_signature"]
            altered["sync_aggregate"]["sync_committee_signature"] = v[:-2] + format(
                int(v[-2:], 16) ^ 1, "02x"
            )
        elif name == "finality_branch":
            v = altered["finality_branch"][0]
            altered["finality_branch"][0] = v[:-2] + format(int(v[-2:], 16) ^ 1, "02x")
        else:
            domain = spec.Root(bytes([bytes(gvr)[0] ^ 1]) + bytes(gvr)[1:])
        try:
            spec.process_light_client_finality_update(
                copy.deepcopy(previous),
                parsed(altered, spec.LightClientFinalityUpdate),
                current_slot(),
                domain,
            )
        except (AssertionError, ValueError):
            rejects.append(name)
        else:
            raise AssertionError("unchanged reference accepted mutated " + name)
    report["rejected_controls"] = rejects
    report["passed"] = (
        bool(latency["latency_passed"])
        and len(missing) / (stop - first) <= profile["missing_slot_fraction_max"]
    )
    save()
    print(json.dumps(report), flush=True)
except Exception as error:
    report["failure"] = {"class": type(error).__name__, "message": str(error)}
    save()
    raise

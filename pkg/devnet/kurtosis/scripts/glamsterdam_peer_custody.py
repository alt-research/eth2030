#!/usr/bin/env python3
"""Observe normal custody increases, provider loss/reconnect and restart in a private observer."""

import argparse, json, subprocess, time, urllib.request
from pathlib import Path
import glamsterdam_campaign as c


def inspect(cid):
    return json.loads(c.docker("inspect", cid))[0]


def service(role):
    ids = c.docker(
        "ps", "-q", "--filter", "label=kurtosis_service_name=" + role
    ).split()
    assert len(ids) == 1
    return ids[0]


def run(output, binary):
    output.mkdir(exist_ok=False)
    nodes = c.inventory()
    c.assert_private_network(nodes)
    observer_id = service("glamsterdam-qualified-snap-cl")
    observer = inspect(observer_id)
    base = c.endpoint({"ports": observer["NetworkSettings"]["Ports"]}, "4000/tcp")
    network = "kt-eth2030-glamsterdam-completion-20261119"
    provider_id = nodes["cl-3-lighthouse-geth"]["container"]
    provider_network = inspect(provider_id)["NetworkSettings"]["Networks"][network]
    trace = (output / "exchanges.jsonl").open("x")

    def get(url):
        row = c.exchange(url)
        trace.write(json.dumps(row) + "\n")
        trace.flush()
        return c.decoded(row)

    def post(path, body):
        request = urllib.request.Request(
            base + path, json.dumps(body).encode(), {"Content-Type": "application/json"}
        )
        with c.OPENER.open(request, timeout=10) as response:
            row = {
                "at": c.now(),
                "url": base + path,
                "request": body,
                "http_status": response.status,
                "response": response.read().decode(),
            }
        trace.write(json.dumps(row) + "\n")
        trace.flush()
        assert row["http_status"] == 200

    before = get(base + "/eth/v1/node/identity")
    assert before["data"]["metadata"]["custody_group_count"] == "4"
    assert "--disable-get-blobs" in observer["Config"]["Cmd"]
    network_spec = get(base + "/eth/v1/config/spec")["data"]
    profile = {
        "declared_at": c.now(),
        "scope": "Validator-free observer receives ordinary proposer-preparation declarations for existing public validator indices. No validator secrets or signing clients are copied. Normal custody growth 4->8->16, provider network loss/reconnect, actual new two-row blobs, historical column backfill and normal restart. All native validation stays active. Main steady/pressure windows are separate.",
        "observer": observer_id,
        "provider": provider_id,
        "provider_network": provider_network,
        "custody_stages": [8, 16],
        "stage_budget_seconds": 600,
        "restart_budget_seconds": 300,
        "provider_loss_seconds": 6,
        "network_spec": network_spec,
    }
    (output / "profile.json").write_text(json.dumps(profile, indent=2) + "\n")
    rows = []
    for count in [8, 16]:
        stage = output / ("cgc-" + str(count))
        stage.mkdir()
        start = time.monotonic()
        before_slot = int(get(base + "/eth/v1/node/syncing")["data"]["head_slot"])
        effective_epoch = (before_slot + 30 // 3) // 32 + 1
        balances = [
            int(
                get(base + "/eth/v1/beacon/states/finalized/validators/" + str(i))[
                    "data"
                ]["validator"]["effective_balance"]
            )
            for i in range(count)
        ]
        expected = min(
            max(
                sum(balances)
                // int(network_spec["BALANCE_PER_ADDITIONAL_CUSTODY_GROUP"]),
                int(network_spec["VALIDATOR_CUSTODY_REQUIREMENT"]),
            ),
            int(network_spec["NUMBER_OF_CUSTODY_GROUPS"]),
        )
        assert expected == count
        post(
            "/eth/v1/validator/prepare_beacon_proposer",
            [
                {
                    "validator_index": str(i),
                    "fee_recipient": "0x0000000000000000000000000000000000000000",
                }
                for i in range(count)
            ],
        )
        advertised_deadline = time.monotonic() + 10
        while time.monotonic() < advertised_deadline:
            identity = get(base + "/eth/v1/node/identity")
            if int(identity["data"]["metadata"]["custody_group_count"]) == count:
                break
            time.sleep(0.1)
        else:
            raise RuntimeError("normal custody change was not advertised within 10 seconds")
        while time.monotonic() - start < 600:
            health = get(base + "/eth/v1/node/syncing")["data"]
            if int(health["head_slot"]) >= effective_epoch * 32 + 2 and not any(
                health[k] for k in ["is_syncing", "is_optimistic", "el_offline"]
            ):
                break
            time.sleep(2)
        else:
            raise RuntimeError("custody DA activation exceeded original stage budget")
        # Disconnect one real provider and restore the same address. This is a
        # process network interruption; no payload, signature or custody check changes.
        detached = False
        try:
            c.docker("network", "disconnect", network, provider_id)
            detached = True
            time.sleep(6)
        finally:
            if detached:
                c.docker(
                    "network",
                    "connect",
                    "--ip",
                    provider_network["IPAddress"],
                    "--alias",
                    "cl-3-lighthouse-geth",
                    network,
                    provider_id,
                )
        time.sleep(12)
        # Rediscover public bindings after changing a real container network.
        nodes = c.inventory()
        metric_nodes = {
            "target": nodes["cl-1-lighthouse-geth"],
            "observer": {"ports": inspect(observer_id)["NetworkSettings"]["Ports"]},
        }
        for label, node in metric_nodes.items():
            (stage / (label + "-before.prom")).write_bytes(
                c.OPENER.open(
                    c.endpoint(node, "5054/tcp") + "/metrics", timeout=10
                ).read()
            )
        command = [
            str(binary),
            "--rpc="
            + ",".join(
                c.endpoint(node, "8545/tcp")
                for name, node in nodes.items()
                if name.startswith("el-")
            ),
            "--genesis=" + c.GENESIS,
            "--output=" + str(stage / "live.jsonl"),
            "--timeout=300s",
            "blobs",
        ]
        with (stage / "live.log").open("x") as log:
            subprocess.run(
                command,
                stdout=log,
                stderr=subprocess.STDOUT,
                check=True,
                timeout=max(1, 600 - (time.monotonic() - start)),
            )
        complete = next(
            row
            for row in map(json.loads, (stage / "live.jsonl").read_text().splitlines())
            if row.get("kind") == "completed_multi_blob_campaign"
        )
        rpc = c.endpoint(nodes["el-1-geth-lighthouse"], "8545/tcp")
        header = c.decoded(
            c.exchange(rpc, "eth_getBlockByHash", [complete["last_block"], False])
        )["result"]
        slot = int(header["slotNumber"], 16)
        beacon = get(base + "/eth/v1/beacon/headers/" + str(slot))["data"]
        root = beacon["root"]
        bundle = {"nodes": {}}
        for name, url in [
            (
                "cl-1-lighthouse-geth",
                c.endpoint(nodes["cl-1-lighthouse-geth"], "4000/tcp"),
            ),
            (
                "cl-3-lighthouse-geth",
                c.endpoint(nodes["cl-3-lighthouse-geth"], "4000/tcp"),
            ),
            ("observer", base),
        ]:
            bundle["nodes"][name] = {
                kind: get(url + path + root)
                for kind, path in [
                    ("block", "/eth/v2/beacon/blocks/"),
                    ("envelope", "/eth/v1/beacon/execution_payload_envelopes/"),
                    ("columns", "/eth/v1/debug/beacon/data_column_sidecars/"),
                ]
            }
        observer_columns = [
            int(col["index"]) for col in bundle["nodes"]["observer"]["columns"]["data"]
        ]
        assert len(observer_columns) >= count
        (stage / "bundle.json").write_text(json.dumps(bundle) + "\n")
        (stage / "anchor.json").write_text(
            json.dumps(
                {
                    "completion": complete,
                    "execution_header": header,
                    "beacon_header": beacon,
                    "beacon_root": root,
                    "slot": slot,
                },
                indent=2,
            )
            + "\n"
        )
        for label, node in metric_nodes.items():
            (stage / (label + "-after.prom")).write_bytes(
                c.OPENER.open(
                    c.endpoint(node, "5054/tcp") + "/metrics", timeout=10
                ).read()
            )
        custody = get(base + "/lighthouse/custody/info")
        rows.append(
            {
                "cgc": count,
                "metadata": identity,
                "effective_epoch": effective_epoch,
                "actual_blob_slot": slot,
                "actual_columns": observer_columns,
                "custody_backfill_observation": custody,
                "elapsed_seconds": time.monotonic() - start,
            }
        )
        assert time.monotonic() - start < 600
        (output / "progress.json").write_text(json.dumps(rows, indent=2) + "\n")
    # The spec retains the highest advertised custody even when declarations age
    # out. A normal process restart must preserve that persisted requirement.
    start = time.monotonic()
    c.docker("restart", "--time", "45", observer_id)
    while time.monotonic() - start < 300:
        try:
            base = c.endpoint(
                {"ports": inspect(observer_id)["NetworkSettings"]["Ports"]}, "4000/tcp"
            )
            identity = get(base + "/eth/v1/node/identity")
            health = get(base + "/eth/v1/node/syncing")["data"]
            if int(
                identity["data"]["metadata"]["custody_group_count"]
            ) == 16 and not any(
                health[k] for k in ["is_syncing", "is_optimistic", "el_offline"]
            ):
                break
        except (OSError, RuntimeError):
            pass
        time.sleep(1)
    else:
        raise RuntimeError("custody persistence restart exceeded original budget")
    result = {
        "passed": True,
        "stages": rows,
        "highest_cgc_survived_restart": 16,
        "normal_restart_seconds": time.monotonic() - start,
        "independent_bls_kzg_evaluated": False,
    }
    (output / "process-result.json").write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps(result))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--live-binary", required=True, type=Path)
    args = parser.parse_args()
    run(args.output, args.live_binary)

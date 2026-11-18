# Observer only: no validator keys or validator client are attached.
def run(plan, args):
    plan.add_service(
        name="frame-history-observer",
        config=ServiceConfig(
            image="eth2030cl:frame-control",
            max_memory=2048,
            files={"/jwt": "jwt_file", "/network-configs": "el_cl_genesis_data"},
            entrypoint=["lighthouse"],
            cmd=[
                "beacon_node", "--datadir=/data/history-observer", "--testnet-dir=/network-configs",
                "--checkpoint-sync-url=http://cl-1-lighthouse-geth:4000",
                "--execution-endpoints=http://el-2-geth-lighthouse:8551", "--jwt-secrets=/jwt/jwtsecret",
                "--http", "--http-address=0.0.0.0", "--http-port=4000", "--listen-address=0.0.0.0",
                "--port=9000", "--quic-port=9001", "--enr-address=KURTOSIS_IP_ADDR_PLACEHOLDER",
                "--enr-tcp-port=9000", "--enr-udp-port=9000", "--enr-quic-port=9001",
                "--boot-nodes=" + args["bootnodes"], "--target-peers=2", "--disable-peer-scoring",
                "--enable-private-discovery", "--disable-packet-filter", "--allow-insecure-genesis-sync",
                "--disable-backfill-rate-limiting", "--subscribe-all-subnets",
            ],
            private_ip_address_placeholder="KURTOSIS_IP_ADDR_PLACEHOLDER",
            ports={
                "http": PortSpec(number=4000, transport_protocol="TCP", application_protocol="http"),
                "tcp-discovery": PortSpec(number=9000, transport_protocol="TCP"),
                "udp-discovery": PortSpec(number=9000, transport_protocol="UDP"),
                "quic-discovery": PortSpec(number=9001, transport_protocol="UDP"),
            },
            ready_conditions=ReadyCondition(recipe=GetHttpRequestRecipe(port_id="http", endpoint="/eth/v1/node/health"), field="code", assertion="IN", target_value=[200, 206], timeout="5m"),
        ),
    )

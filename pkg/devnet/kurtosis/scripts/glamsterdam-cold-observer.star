# Fresh databases and no validator client: both actual EL and CL synchronize independently.
def run(plan, args):
    mode = args.get("syncmode", "full")
    if mode not in ["full", "snap"]:
        fail("observer syncmode must be full or snap")
    prefix = args.get("name_prefix", "glamsterdam-cold") + "-" + mode
    el_name = prefix + "-el"
    cl_name = prefix + "-cl"
    if not args.get("reuse_existing_el", False):
        plan.add_service(
            name=el_name,
            config=ServiceConfig(
                image=args["el_image"],
                max_memory=4096,
                max_cpu=2000,
                files={"/jwt": "jwt_file", "/network-configs": args["network_artifact"]},
                entrypoint=["geth"],
                cmd=[
                    "--override.genesis=/network-configs/genesis.json", "--datadir=/data/geth/execution-data",
                    "--http", "--http.addr=0.0.0.0", "--http.port=8545", "--http.vhosts=*",
                    "--http.api=admin,engine,net,eth,web3,debug,txpool", "--rpc.rate_limit=0",
                    "--authrpc.port=8551", "--authrpc.addr=0.0.0.0", "--authrpc.jwtsecret=/jwt/jwtsecret",
                    "--syncmode=" + mode, "--networkid=1337", "--port=30303",
                    "--nat=extip:KURTOSIS_IP_ADDR_PLACEHOLDER", "--bootnodes=" + args["el_bootnodes"],
                    "--metrics", "--metrics.addr=0.0.0.0", "--metrics.port=9001",
                ],
                private_ip_address_placeholder="KURTOSIS_IP_ADDR_PLACEHOLDER",
                ports={
                    "rpc": PortSpec(number=8545, transport_protocol="TCP", application_protocol="http"),
                    "engine": PortSpec(number=8551, transport_protocol="TCP", application_protocol="http"),
                    "metrics": PortSpec(number=9001, transport_protocol="TCP", application_protocol="http"),
                    "p2p-tcp": PortSpec(number=30303, transport_protocol="TCP"),
                    "p2p-udp": PortSpec(number=30303, transport_protocol="UDP"),
                },
                ready_conditions=ReadyCondition(recipe=GetHttpRequestRecipe(port_id="rpc", endpoint="/"), field="code", assertion="IN", target_value=[200, 400, 405], timeout="2m"),
            ),
        )
    plan.add_service(
        name=cl_name,
        config=ServiceConfig(
            image=args["cl_image"],
            max_memory=4096,
            max_cpu=2000,
            files={"/jwt": "jwt_file", "/network-configs": args["network_artifact"]},
            entrypoint=["lighthouse"],
            cmd=[
                "beacon_node", "--debug-level=debug", "--datadir=/data/checkpoint-observer",
                "--testnet-dir=/network-configs", "--checkpoint-sync-url=http://cl-1-lighthouse-geth:4000",
                "--execution-endpoints=http://" + el_name + ":8551", "--jwt-secrets=/jwt/jwtsecret",
                "--http", "--http-address=0.0.0.0", "--http-port=4000", "--listen-address=0.0.0.0",
                "--port=9000", "--quic-port=9001", "--enr-address=KURTOSIS_IP_ADDR_PLACEHOLDER",
                "--enr-tcp-port=9000", "--enr-udp-port=9000", "--enr-quic-port=9001",
                "--boot-nodes=" + args["cl_bootnodes"], "--target-peers=3",
                "--enable-private-discovery", "--disable-packet-filter",
                "--metrics", "--metrics-address=0.0.0.0", "--metrics-port=5054",
            ] + (["--disable-get-blobs"] if args.get("disable_get_blobs", False) else []),
            private_ip_address_placeholder="KURTOSIS_IP_ADDR_PLACEHOLDER",
            ports={
                "http": PortSpec(number=4000, transport_protocol="TCP", application_protocol="http"),
                "p2p-tcp": PortSpec(number=9000, transport_protocol="TCP"),
                "p2p-udp": PortSpec(number=9000, transport_protocol="UDP"),
                "quic": PortSpec(number=9001, transport_protocol="UDP"),
                "metrics": PortSpec(number=5054, transport_protocol="TCP", application_protocol="http"),
            },
            ready_conditions=ReadyCondition(recipe=GetHttpRequestRecipe(port_id="http", endpoint="/eth/v1/node/health"), field="code", assertion="IN", target_value=[200, 206], timeout="5m"),
        ),
    )

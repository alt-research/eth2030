# ETH2030 devnet explorers and third-party services

Inventory checked on 2026-11-17 against the 11 top-level and 40 feature YAML
profiles on the CL roadmap branch, the companion Heze profile, the pinned
ethereum-package source, the workspace Git checkouts and the historical run
records. All workspace paths in this document are relative to `/projects/eth2030/`.

## Maintained forks and reference checkouts

| Component | Local source and revision | Ownership and use |
| --- | --- | --- |
| Dora | `dora/`, `eth2030` at [`6f702f2`][dora-old]. `dora-heze-inclusion/`, `feat/heze-inclusion-history` at [`12a0198`][dora-current]. | One fork of [ethpandaops/dora][dora-upstream] with two worktrees. The March branch contains the initial FOCIL UI and historical Type 6 display. The Heze branch reads the current Lighthouse payload bid, stores inclusion views by proposal root and serves them from its database after restart. Use the Heze revision for the retained-list verifier. Gloas and Heze BeaconState SSZ epoch statistics remain open. |
| go-eth2-client | The existing checkout `dora/go-eth2-client/` is at `21c05ba`. The Heze Dora gitlink pins [`79989d2`][go-eth2-client] and its worktree submodule is uninitialized. | The Heze Dora build uses the `fyInALT/go-eth2-client` fork through the local Go module replacement and a tracked submodule mapping. The March Dora tree has a gitlink without a `.gitmodules` mapping. A clean checkout of that revision requires resolving that mapping before its documented local build can be reproduced. |
| Spamoor | `spamoor/`, `feat/support-native-aa` at [`4908a05`][spamoor]. | Fork of [ethpandaops/spamoor][spamoor-upstream]. Adds raw and UserOp native-AA scenarios, the historical Frame builder, wallet funding and receipt handling. Its Frame encoder still has separate execution and blob fee caps, so the pinned seven-field draft needs a generator update. `eth2030/refs/spamoor/` is the upstream reference at `7e0d24d`. |
| Lighthouse | `lighthouse-eth2030/`, `eth2030` at [`1bf89a4fc`][lighthouse]. | Fork of [sigp/lighthouse][lighthouse-upstream]. Provides the BN, VC, and current Heze data consumed by Dora. `lighthouse/` and `lighthouse-*/` integration worktrees share the same Git repository. Build the intended branch explicitly before selecting `eth2030cl:local`. |
| ethereum-package | `eth2030/refs/ethereum-package/`, upstream version `6.1.0` at [`c4a8e15d`][package]. | Read-only submodule from ethpandaops. The devnet and feature runners default to it. This checkout contains the service launchers, image defaults, port wiring and genesis jobs. A roadmap worktree needs its own initialized copy of the submodule before it can run the default launcher. |
| Frame reference stack | `eip8141-implementation/` at [`ed68710`][frame-ref] and `revm-eip8141/` at [`2c5f7089`][revm-ref]. | External research checkouts. The Frame reference pins `foundry-eip8141` at `7fc6109`, `revm-eip8141` at `2c5f7089`, and ETHDILITHIUM at `e1d788b`. The standalone revm checkout and the nested gitlink point at the same source revision. These are references for Frame tests and anvil, outside the tracked Kurtosis service profiles. |

The Dora, Spamoor and Lighthouse branch tips above match their fork remotes.
`dora-config/` is the historical local Dora deployment directory. Its
`run-dora-local.sh` script hard-codes the `dora/` worktree, copies network
configuration from a running devnet and deletes the SQLite file and Pebble
database on launch. The Heze retention test builds from the Heze worktree and
reuses the existing Dora database across restart. The binaries, databases,
logs and generated `el_cl_genesis_data/` snapshots are runtime artifacts.

## Explorers, load generators and assertions

Profile counts are explicit entries across the 51 tracked YAMLs above. They
measure configuration. Live observations are recorded separately in each row.

| Service | Profile or image selection | Evidence and use |
| --- | --- | --- |
| **Dora** | 48 profiles. `full-feature`, `full-feature-eth2030` and `single-client` explicitly select `eth2030/dora:v0.0.1`. The package otherwise selects upstream Dora and can resolve it to `ethpandaops/dora:gloas-support` or `ethpandaops/dora:heze-support`. | Beacon slots, validator and bid data, and FOCIL committee positions. The capped Heze run used the fork's retained-view changes and verified transaction bytes after BN pruning and Dora restart. The status entry records source revisions and the result. The exact image IDs and generated profile overrides are unavailable in that entry. [Run record][heze-dora-status] · [Image selection][dora-launcher] |
| **Blockscout** | Enabled in `full-feature`. The upstream launcher deploys the backend, frontend, contract verifier and PostgreSQL. `full-feature-eth2030` dropped it in [`c4dc9aec`][blockscout-removal] after flaky verifier startup. | EL transactions, addresses, contracts and receipts. Present in an older test deployment inventory. No Blockscout verifier acceptance result is retained in the CL status. Keep its backend and verifier version separate from CL acceptance. [Upstream][blockscout] · [Launcher][blockscout-launcher] |
| **Blobscan** | Enabled in `peerdas-throughput`. The package uses the upstream API, web and indexer images plus PostgreSQL and Redis. | Blob transaction and sidecar indexing. The profile and all three application images are present locally. Successful indexing and PeerDAS custody still need result evidence from the relevant APIs. The package exposes the UI as `blobscan-web`; the generic runner looks up `blobscan`. [Upstream][blobscan] · [Launcher][blobscan-launcher] |
| **Assertoor** | 49 profiles. Upstream image defaults to `ethpandaops/assertoor:latest`; the pinned launcher selects `gloas-support` when Gloas is configured and the default image is used. | Historical full-feature runs exposed the service. The standalone checker polls `/api/v1/test_runs`. The feature runner labels any nonzero checker result `TIMEOUT`, and the overall result depends on consensus and feature checks. A historical overall `PASS` cannot establish completion of the Assertoor checks. Retain named runs and their actual terminal statuses when a test relies on it. [Upstream][assertoor] · [Launcher][assertoor-launcher] · [Feature runner][feature-runner] |
| **Spamoor** | 39 profiles. `full-feature` and `full-feature-eth2030` explicitly select `eth2030/spamoor:v0.0.1`; other configured profiles use the package default image unless overridden. | EOA, token, blob and diagnostic traffic. A full-feature fork run recorded Spamoor EOA transfers landing. The historical `full-feature` YAML also configures EIP-7701 raw and UserOp modes, which belong to the earlier draft. The pinned Frame case needs generator-produced bytes, canonical receipts and sponsor balance checks. [Run record][spamoor-status] · [Full profile][full-profile] · [Frame encoder][frame-encoder] |
| **Rakoon** | Five profiles. This is the `rakoon` binary from the package's upstream `ethpandaops/fuzztools:main` image. The dedicated EIP-7702 profile and historical Native AA profiles select it for SetCode transaction coverage. | The devnet README records an EIP-7702 profile pass. The available summary gives no per-transaction outcome set for Rakoon. Preserve the seed and concrete accepted or rejected transaction observations in a run that claims Rakoon coverage. [Upstream][fuzztools] · [Profile][eip7702-profile] · [Devnet README][devnet-readme] |

The local `eth2030/dora:v0.0.1` and `eth2030/spamoor:v0.0.1` images lack a
registry digest and an OCI revision label on this host. The tags cannot be
mapped to a source commit from image metadata. The historical Heze Dora result
is tied to the source revision recorded in its run, and the broad full-feature
configuration continues to use its older image tag.

## Orchestration, monitoring and optional services

| Component | Use and current evidence |
| --- | --- |
| **Kurtosis and Docker** | The host has Kurtosis CLI `1.15.2` and Docker `29.7.2`. The runners invoke the local read-only package by default, build or reuse the EL image, and expose mapped ports. The separate Compose entry point starts the EL and Lighthouse with a JWT initialization job. It has no explorer or assertion services configured. [Kurtosis package][package] · [Devnet runner][devnet-runner] · [Compose][compose] |
| **Genesis generator and validator-key jobs** | Bundled package jobs produce EL genesis, CL SSZ, the fork configuration and validator keystores. The pinned package defaults to generator `5.3.5`. The direct Gloas-genesis run needed `6.2.8` for CL SSZ decoding and a subsequent EL genesis-header fix. Record the generator revision and produced genesis identities with each run. [Package defaults][package-constants] · [Generator v6.2.8][generator] · [Gloas evidence][gloas-genesis-status] |
| **Prometheus and Grafana** | Upstream scrape storage and dashboards. Both appeared in past deployment inventories. None of the 51 current profiles lists either as an additional service. The PeerDAS profile's `grafana_params` supplies settings when Grafana is enabled, and the package starts it only when selected. [Package service dispatch][package] · [PeerDAS profile][peerdas-profile] |
| **Bootnodoor and Checkpointz** | Upstream discovery bootstrap and checkpoint HTTP service. An earlier [`197f456a` change][bootnodoor-history] explicitly added Bootnodoor for CL peer discovery. Neither is listed in the current profiles. The cached Checkpointz image establishes local availability; independent checkpoint/late-join validation remains a separate gap. [Bootnodoor][bootnodoor] · [Checkpointz][checkpointz] |
| **Forky, Forkmon, Tracoor and RPC Snooper** | Package-supported diagnostics for CL fork choice, EL head divergence, traces and RPC capture. The runner probes for several optional dashboard URLs. None is enabled by the audited profiles. RPC Snooper has a separate enable path in package configuration. [Package][package] · [Devnet runner][devnet-runner] · [RPC Snooper][snooper] |
| **Other package integrations** | The package supports external builder and relay stacks, remote Web3Signer, the full Beaconchain explorer and additional gateways. These are upstream choices in the package. The tested Gloas evidence covers self-build, with external builder payment still open. [Package defaults][package-constants] · [CL gap audit][cl-gap-audit] |
| **Foundry and reference clients** | The host had `cast` `1.5.1-stable` for RPC probes and transaction submission. The forked Foundry reference is nested under `eip8141-implementation/foundry/`. The `geth` profile uses the package's upstream Geth default, while ETH2030 images also set `el_type: geth` to select the launcher. A Prysm profile is available as an alternative CL configuration. Pin executable images before using either as independent acceptance evidence. [Geth profile][geth-profile] · [Prysm profile][prysm-profile] |

Blockscout, Blobscan, Assertoor, Rakoon and the monitoring services above use
upstream images and package integration. The workspace contains no checked-out
source fork for those services. Container images, worktrees, nested Git sources
and runtime outputs have separate provenance.

## Which services to use for each acceptance run

| Scenario | Services and evidence |
| --- | --- |
| Fulu Track 4 and isolated Heze FOCIL | Start with the corresponding two-pair profile and no additional services. Retain the EL RPC and Beacon API results from both pairs, fork and finality state, and transaction receipt. The Track 4 wrapper injects genesis allocations. The Heze runbook requires a genesis generator that emits `bogotaTime` and links the signed-list verifier. The capped Dora run added the explorer for its persistence check. [Track 4 runbook][track4-runbook] · [Heze runbook][heze-runbook] · [Heze profile][heze-profile] |
| Heze explorer history | Use the Lighthouse retained-view API introduced at `5d4500270` and preserved on `eth2030`. Run `git submodule update --init go-eth2-client` in the Dora `12a0198` worktree before building it. Add Dora to the Heze profile, match the signed list and bid bits to the proposal root, and reopen Dora against the same database after the BN prunes its view. |
| Execution explorer | Use Blockscout when the test requires an indexed receipt, contract view, or verifier result. Compare indexed data with canonical EL RPC and retain the backend, frontend and verifier image identities separately. |
| Blob and data availability | Use Spamoor blob traffic for load and Blobscan for indexing evidence. Compare versioned hashes and sidecars with Beacon and EL observations. Record availability and custody checks from the relevant client APIs. |
| Gloas failure, divergence and resource diagnosis | Capture Engine and Beacon responses and metrics. Enable the applicable diagnostics for the scenario. Record actual external builder bids, envelopes and PTC decisions for ePBS acceptance. |
| Broad feature smoke or load | Use Assertoor for defined assertions and Spamoor or Rakoon for the intended traffic. Keep service completion, protocol results and app-level explorer checks visible as individual results. |

For a repeatable run, retain the effective package and profile revisions,
service image digests or local build revisions, EL and CL versions, genesis
identities, and the result from each service used as a test oracle. Resolve
ports from the active service names. Use bounded EL and CL node memory for
the paired test, and stop the enclave once the evidence has been collected.
The [CL gap audit][cl-gap-audit] tracks the protocol-specific acceptance gaps.

[dora-old]: https://github.com/jiayaoqijia/dora/commit/6f702f24e8445d9e4b764d5bf141c604b1c99d47
[dora-current]: https://github.com/jiayaoqijia/dora/commit/12a0198f237c8bf1973256ae5c2498d7cef90587
[dora-upstream]: https://github.com/ethpandaops/dora
[go-eth2-client]: https://github.com/fyInALT/go-eth2-client/commit/79989d2591dc7a964990097a1e048db49d4d3890
[spamoor]: https://github.com/jiayaoqijia/spamoor/commit/4908a05d77d3836f61524aeda6c4ee79f8c36097
[spamoor-upstream]: https://github.com/ethpandaops/spamoor
[lighthouse]: https://github.com/jiayaoqijia/lighthouse/commit/1bf89a4fc933d58828c39bcda3fe6eb481d680f5
[lighthouse-upstream]: https://github.com/sigp/lighthouse
[package]: https://github.com/ethpandaops/ethereum-package/blob/c4a8e15d40f0965f3728f776d72f1954440ad96b/main.star
[package-constants]: https://github.com/ethpandaops/ethereum-package/blob/c4a8e15d40f0965f3728f776d72f1954440ad96b/src/package_io/constants.star
[frame-ref]: https://github.com/aalimsahin/eip8141-implementation/commit/ed68710b4d975e825f71b6c23f4128c8fce2800b
[revm-ref]: https://github.com/aalimsahin/revm-eip8141/commit/2c5f708968acbb9e13478296ecbc48258274b625
[heze-dora-status]: ../STATUS.md#pinned-frame-encoding-and-inclusion-history-2026-11-16
[dora-launcher]: https://github.com/ethpandaops/ethereum-package/blob/c4a8e15d40f0965f3728f776d72f1954440ad96b/src/dora/dora_launcher.star
[blockscout-removal]: https://github.com/alt-research/eth2030/commit/c4dc9aec256b140e1e7aad07975fd48e1b54f25c
[blockscout]: https://github.com/blockscout/blockscout
[blockscout-launcher]: https://github.com/ethpandaops/ethereum-package/blob/c4a8e15d40f0965f3728f776d72f1954440ad96b/src/blockscout/blockscout_launcher.star
[blobscan]: https://github.com/Blobscan/blobscan
[blobscan-launcher]: https://github.com/ethpandaops/ethereum-package/blob/c4a8e15d40f0965f3728f776d72f1954440ad96b/src/blobscan/blobscan_launcher.star
[assertoor]: https://github.com/ethpandaops/assertoor
[assertoor-launcher]: https://github.com/ethpandaops/ethereum-package/blob/c4a8e15d40f0965f3728f776d72f1954440ad96b/src/assertoor/assertoor_launcher.star
[feature-runner]: ../pkg/devnet/kurtosis/scripts/run-feature-tests.sh
[spamoor-status]: ../STATUS.md#hegotá-track-2-closure-2026-11-12
[full-profile]: ../pkg/devnet/kurtosis/configs/full-feature.yaml
[frame-encoder]: https://github.com/jiayaoqijia/spamoor/blob/4908a05d77d3836f61524aeda6c4ee79f8c36097/txbuilder/frametx_rlp.go
[fuzztools]: https://github.com/ethpandaops/fuzztools
[eip7702-profile]: ../pkg/devnet/kurtosis/configs/eip7702-test.yaml
[devnet-readme]: ../pkg/devnet/kurtosis/README.md
[devnet-runner]: ../pkg/devnet/kurtosis/scripts/run-devnet.sh
[compose]: ../pkg/devnet/docker-compose.yml
[generator]: https://github.com/ethpandaops/ethereum-genesis-generator/releases/tag/v6.2.8
[gloas-genesis-status]: ../STATUS.md#bogotá-engine-and-heze-inclusion-recovery-2026-11-16
[peerdas-profile]: ../pkg/devnet/kurtosis/configs/peerdas-throughput.yaml
[bootnodoor-history]: https://github.com/alt-research/eth2030/commit/197f456a7fb4de691edc4f5f1fa40274b0c16bb1
[bootnodoor]: https://github.com/ethpandaops/bootnodoor
[checkpointz]: https://github.com/ethpandaops/checkpointz
[snooper]: https://github.com/ethpandaops/rpc-snooper
[cl-gap-audit]: lighthouse-cl-gap-audit.md
[geth-profile]: ../pkg/devnet/kurtosis/configs/geth.yaml
[prysm-profile]: ../pkg/devnet/kurtosis/configs/full-feature-prysm.yaml
[track4-runbook]: ../pkg/devnet/kurtosis/README.md#track-4-private-devnet
[heze-runbook]: https://github.com/fyInALT/eth2030/blob/e628c770df98ba9ad269de74344afcdf4e6a92a3/pkg/devnet/kurtosis/README.md#heze-compulsory-inclusion-devnet
[heze-profile]: https://github.com/fyInALT/eth2030/blob/e628c770df98ba9ad269de74344afcdf4e6a92a3/pkg/devnet/kurtosis/configs/heze-focil.yaml

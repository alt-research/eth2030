# PR143 CI format follow-up

The hosted Build/Vet/Lint/Test job stopped at gofmt, reporting engine/server.go. Its other8 jobs passed. This patch applies gofmt to that file; no Engine behavior or validation changes. Paris forkchoice rules4-6 remain unchanged. The independent cumulative PR142/143/144 control passed219 Go packages,3 affected race packages and native build before this whitespace-only repair. Exact new-head hosted verdict remains separate.

Affected Engine package tests, full package gofmt scan and native Go1.26.11 build pass after the repair.

## Address follow-up

Actual bind review found empty HTTP hosts were classified as loopback despite listening on all interfaces, and127-prefixed DNS names/invalid literals were accepted as loopback without parsing. IPv6 loopback endpoints were also formatted as ambiguous `::1:port`. New controls fail on all four bad classification examples and the IPv6 endpoint before the repair.

Use net.ParseIP/IsLoopback without DNS and net.JoinHostPort for HTTP, Engine, WebSocket and metrics listeners. Empty hosts retain wildcard semantics and require the existing explicit privileged-module authorization. IPv4 and public-module defaults are unchanged. The existing helper's IPv4 hostname:port and unbracketed IPv6-host contracts remain intact. No Ethereum payload/hash/signature or consensus rule changes.

Final source passes219 full Go packages,3 affected race packages and explicit builds of native cmd/eth2030 plus Geth-backed cmd/eth2030-geth. The earlier quoted geth command was the wrapper build; the two entrypoints are now explicitly distinguished.

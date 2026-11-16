#!/usr/bin/env bash
set -euo pipefail
PKG="$(cd "$(dirname "$0")/../../.." && pwd)"
ENCLAVE="${1:-eth2030-track4}"

endpoint() {
    local url
    url=$(kurtosis port print "$ENCLAVE" "$1" "$2")
    case "$url" in
        http://*|https://*) printf '%s' "$url" ;;
        *) printf 'http://%s' "$url" ;;
    esac
}

rpc1=$(endpoint el-1-geth-lighthouse rpc)
rpc2=$(endpoint el-2-geth-lighthouse rpc)
cl1=$(endpoint cl-1-lighthouse-geth http)
cl2=$(endpoint cl-2-lighthouse-geth http)
ETH2030_TRACK4_RPC="$rpc1 $rpc2" ETH2030_TRACK4_BEACON="$cl1 $cl2"     go -C "$PKG" test ./node -run '^TestTrack4LiveDevnet$' -v -count=1

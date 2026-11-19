#!/bin/bash
set -euo pipefail
forward=()
for arg in "$@"; do
    case "$arg" in
        --authrpc.port=*) forward+=("--authrpc.port=18551") ;;
        *) forward+=("$arg") ;;
    esac
done
mkdir -p /data
/usr/local/bin/engine-capture --listen=0.0.0.0:8551 --upstream=http://127.0.0.1:18551 --output=/data/engine-exchanges.jsonl &
exec /usr/local/bin/eth2030 "${forward[@]}"

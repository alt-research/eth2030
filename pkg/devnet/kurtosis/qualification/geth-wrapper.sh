#!/bin/bash
set -euo pipefail
forward=()
init_flags=()
genesis=""
datadir=""
for arg in "$@"; do
    case "$arg" in
        --override.genesis=*) genesis="${arg#*=}" ;;
        --datadir=*) datadir="${arg#*=}"; forward+=("$arg") ;;
        --state.scheme=*) init_flags+=("$arg"); forward+=("$arg") ;;
        --authrpc.port=*) forward+=("--authrpc.port=18551") ;;
        *) forward+=("$arg") ;;
    esac
done
if [[ -n "$genesis" ]]; then
    [[ -n "$datadir" ]] || { echo "explicit datadir required" >&2; exit 1; }
    /usr/local/bin/geth-real "--datadir=$datadir" "${init_flags[@]}" init "$genesis"
    /usr/local/bin/engine-capture --listen=0.0.0.0:8551 --upstream=http://127.0.0.1:18551 --output=/data/engine-exchanges.jsonl &
fi
exec /usr/local/bin/geth-real "${forward[@]}"

#!/bin/bash
set -euo pipefail
forward=()
genesis=""
datadir=""
for arg in "$@"; do
    case "$arg" in
        --override.genesis=*) genesis="${arg#*=}" ;;
        --datadir=*) datadir="${arg#*=}"; forward+=("$arg") ;;
        *) forward+=("$arg") ;;
    esac
done
if [[ -n "$genesis" ]]; then
    [[ -n "$datadir" ]] || { echo "explicit datadir required" >&2; exit 1; }
    /usr/local/bin/geth-real "--datadir=$datadir" init "$genesis"
fi
exec /usr/local/bin/geth-real "${forward[@]}"

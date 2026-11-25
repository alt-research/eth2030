#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)/$(uname -m)" != "Linux/x86_64" ]]; then
  echo "The qualified canonical prover bundle requires Linux x86_64." >&2
  exit 2
fi
export RISC0_HOME="${RISC0_HOME:-${XDG_CACHE_HOME:-$HOME/.cache}/eth2030/risc0-3.1.0}"
mkdir -p "$RISC0_HOME/bin"
PROVER_RZUP="$RISC0_HOME/bin/rzup"
PROVER_RZUP_SHA=2cbb8081730ca42acac6818a279c7f218878b571a7f9ff76970f60688f54ddf5
if [[ ! -x "$PROVER_RZUP" ]]; then
  curl --fail --http1.1 --location --retry 3 https://artifacts.risczero.net/rzup/prod/Linux-X64/rzup -o "$PROVER_RZUP.tmp"
  printf '%s  %s\n' "$PROVER_RZUP_SHA" "$PROVER_RZUP.tmp" | sha256sum -c -
  chmod 755 "$PROVER_RZUP.tmp"
  mv "$PROVER_RZUP.tmp" "$PROVER_RZUP"
fi
printf '%s  %s\n' "$PROVER_RZUP_SHA" "$PROVER_RZUP" | sha256sum -c -
"$PROVER_RZUP" install rust 1.97.0
"$PROVER_RZUP" install cargo-risczero 3.1.0
"$PROVER_RZUP" install r0vm 3.1.0


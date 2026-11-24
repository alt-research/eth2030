#!/usr/bin/env bash
set -euo pipefail
PROVER_ROOT="$(cd "$(dirname "$0")" && pwd)"
NODE_ROOT="$(cd "$PROVER_ROOT/../.." && pwd)"
DIST_DIR="${1:-$PROVER_ROOT/target/node-bundle}"
export RISC0_HOME="${RISC0_HOME:-${XDG_CACHE_HOME:-$HOME/.cache}/eth2030/risc0-3.1.0}"
export CARGO_BUILD_JOBS="${CARGO_BUILD_JOBS:-2}"
export RISC0_BUILD_LOCKED=1
export PATH="$HOME/.cargo/bin:$PATH"

command -v docker >/dev/null
command -v rustup >/dev/null
if ! r0vm --version | grep -q '3\.1\.0'; then
  echo "Install the pinned RISC Zero components with tools/canonical-prover/bootstrap.sh." >&2
  exit 2
fi
cargo build --manifest-path "$PROVER_ROOT/Cargo.toml" --release --locked -p eth2030-prover
PROVER_TARGET="${CARGO_TARGET_DIR:-$PROVER_ROOT/target}"
mkdir -p "$DIST_DIR"
DIST_DIR="$(cd "$DIST_DIR" && pwd)"
install -m 755 "$PROVER_TARGET/release/eth2030-prover" "$DIST_DIR/eth2030-prover"
install -m 755 "$(command -v r0vm)" "$DIST_DIR/r0vm"
PROVER_REVISION="$(git -C "$NODE_ROOT" rev-parse HEAD)"
PROVER_DIRTY=false
if [[ -n "$(git -C "$NODE_ROOT" status --porcelain --untracked-files=normal -- pkg tools/canonical-prover)" ]]; then
  PROVER_DIRTY=true
fi
(
  cd "$NODE_ROOT/pkg"
  GOTOOLCHAIN="${GOTOOLCHAIN:-go1.26.11}" go build -trimpath -ldflags "-X main.commit=$PROVER_REVISION" -o "$DIST_DIR/eth2030" ./cmd/eth2030
)
printf '{"action":"metadata"}' | "$DIST_DIR/eth2030-prover" > "$DIST_DIR/prover-image.json"
python3 - "$DIST_DIR" "$PROVER_REVISION" "$PROVER_DIRTY" <<'PYTHON'
import hashlib,json,sys
from pathlib import Path
root=Path(sys.argv[1])
result={'source_revision':sys.argv[2],'source_dirty':sys.argv[3]=='true','proof_system':'risc0-3.1.0','image':json.loads((root/'prover-image.json').read_text()),'binaries':{}}
for name in ['eth2030','eth2030-prover','r0vm']:
    result['binaries'][name]=hashlib.sha256((root/name).read_bytes()).hexdigest()
(root/'build-info.json').write_text(json.dumps(result,indent=2)+'\n')
PYTHON
echo "Native node and real local prover bundle: $DIST_DIR"


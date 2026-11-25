#!/bin/sh
# The Lighthouse genesis uses the original MPT root. Track 4 activates later.
set -eu

for arg do
    case "$arg" in
        --override.genesis=*) original=${arg#*=} ;;
    esac
done
: "${original:?missing Kurtosis execution genesis}"
timestamp=$(jq -er '.timestamp' "$original")
activation=$((timestamp + 12))
conversion=$((timestamp + 24))
profile=$(mktemp)
jq --argjson activation "$activation" --argjson conversion "$conversion" '
    .config.binaryOverlayTime = $activation
    | .config.binaryConversionTime = $conversion
    | .config.binaryConversionStride = 1
    | .config.pqDraft = {
        time: $activation,
        sigRecover: "0x0000000000000000000000000000000000007001",
        mldsa: "0x0000000000000000000000000000000000007002",
        mldsaEth: "0x0000000000000000000000000000000000007003"
    }
' "$original" > "$profile"

# Preserve argument boundaries when replacing the genesis path.
for arg do
    shift
    case "$arg" in
        --override.genesis=*) set -- "$@" "--override.genesis=$profile" ;;
        *) set -- "$@" "$arg" ;;
    esac
done
exec /usr/local/bin/eth2030 "$@"

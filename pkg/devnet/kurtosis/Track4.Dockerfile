ARG EL_IMAGE=eth2030:track4-devnet
FROM ${EL_IMAGE}
RUN apk add --no-cache jq && rm /usr/local/bin/geth
COPY --chmod=755 devnet/kurtosis/scripts/track4-geth.sh /usr/local/bin/geth

FROM ubuntu:24.04
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates libstdc++6 zlib1g && rm -rf /var/lib/apt/lists/*
COPY eth2030 eth2030-prover r0vm /usr/local/bin/
COPY build-info.json prover-image.json /usr/local/share/eth2030/
RUN ln -s /usr/local/bin/eth2030 /usr/local/bin/geth
EXPOSE 30303 8545 8546 8551 9001
ENTRYPOINT ["eth2030"]


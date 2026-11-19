# Hash-selected execution RPC correction in progress

The actual independent-client application campaign passes four signed transactions and their complete header/transaction/receipt commitments, then Native eth_call rejects an EIP-1898 object with blockHash and requireCanonical as an invalid block number. Unchanged Geth accepts the same selection. This review traces execution entrypoints and fixes exact historical/canonical context selection; no numeric-selector workaround is substituted in the campaign. Implementation, real consumer regressions and qualification remain in progress.

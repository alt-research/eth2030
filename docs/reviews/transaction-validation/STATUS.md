# Native transaction validation in progress

The complete official Osaka/Amsterdam state campaign exposed missing high-S, full-width chain/parity and historical nonce/empty-authorization guards, plus mandatory-recipient decoding that normalized prohibited creation envelopes. This task traces the actual raw admission/import paths and retains earlier-fork and ecrecover behavior. Implementation, consumer negatives, complete-corpus qualification and PR publication remain in progress.

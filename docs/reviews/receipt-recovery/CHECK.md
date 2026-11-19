# Historical transaction and receipt recovery

- [x] Read receipt/envelope/fee authority and reproduce missing derived fields after actual Node restart and cold receipt-cache reads.
- [x] Restore transaction senders, effective fees, blob fields and creation addresses using the exact historical header; preserve consensus receipt bytes and all private envelope handling.
- [x] Independently verify derived fields, receipts commitments and actual HTTP results before/after restart across relevant envelope types and fork contexts.
- [ ] Run affected/full/integration/race/vet/build checks and publish a separate production correction.
- [ ] Compose the qualified correction into the owned runtime and resume the original exact historical application sequence.

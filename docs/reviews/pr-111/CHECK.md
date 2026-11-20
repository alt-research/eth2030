# Verified persistent-head recovery

- [x] Preserve the actual completed 8,085-root index and failed 300-second ordinary restart; trace startup replay and repeated validated-block writes.
- [x] Restore only an indexed, fully authenticated original head after checking its durable execution-validation marker and complete canonical ancestry/header/body commitments.
- [x] Preserve legacy/snap fallback, finality and forkchoice authority; keep initial network imports on normal execution validation.
- [x] Exercise restart/exact historical readback, corrupted nodes/bodies/ancestry, missing validation markers, branch context and ordinary next-block import with independent roots/proofs.
- [x] Run affected/full/race/integration/vet/build checks, publish the separate review and repeat the unchanged actual restart budget.

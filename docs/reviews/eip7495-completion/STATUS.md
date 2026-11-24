# Status

Implementing current ProgressiveContainer(active_fields) semantics from EIP-7495 with EIP-7916 progressive Merkleization. The audit reproduced 477 incorrect mixed-root gindices in 487 progressive-list proof cases; correcting the root-prefix in the diagnostic makes all 487 verify. Four old conformance tests also accepted ordinary binary Merkleization in a temporary mutant copy. These are retained as pre-fix failures rather than counted as completed validation.

The new API will own an immutable schema, reject illegal type layouts and reuse existing canonical Container serialization. The old optional-field root utility will be explicitly deprecated as a compatibility helper; it will not serve as a ProgressiveContainer type declaration. Existing production EL/CL roots and wire formats must remain independently equal.

# Composed native runtime candidate

Source 7e2c66e17611765fe2c75dd2a06783e945bddce0 retains the original independent feature reviews and composes PR 97 transaction context, PR 98 real application sequences, PR 96 simulation timeout classification and PR 99 asynchronous sparse-promotion observation. No EL review is merged into master. Full composed Go tests, core/P2P integration, vet and build pass locally; the native command runtime also builds with goethkzg and reports the exact candidate source.

The native ELF SHA-256 is d7ba7306b8efb7a85f45bac7e16a5bae07def38d63edfb1cf33d9cb0a5b86942, image ID 784be4f54e2dc481601ef032d5e14ae290804dab49af567cf7b56ea4b47ef0c3. It executes cmd/eth2030, not the geth-backed bridge. The independent Geth binary reports 1.17.10-stable, and Lighthouse production runtime reports 0262e8f. The separate process campaign pins their actual image/artifact identities and independent Lodestar/Buildoor digests.

PR 97 latest CI passes all eight jobs; PR 98 latest CI passes all nine jobs, including complete native official state fixtures and actual application sequence evidence. This composition remains a draft process-campaign base. The thirteen-package inventory still has API/lifecycle/payment/proof/peer/durability/resource and final release obligations; runtime builds and matching component roots do not close those gates.

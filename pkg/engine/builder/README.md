# Retired legacy Builder Engine extension

The local `engine/builder` registry, synthetic one-ETH stake, non-SSZ bid hash and unsigned Engine auction/header methods were retired by explicit user selection. This directory remains as a migration notice, not an implementation package.

Builder identity, finalized deposits, balance/reservations, activation/exit and BLS domains are owned by the selected Gloas CL. EL deposits/exits continue through canonical EIP-8282 contracts and execution requests. The normal Amsterdam payload/body/blob Engine methods remain available. No replacement EL registry or new signing protocol is introduced.

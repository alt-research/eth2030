# PQClean historical signature providers

The `falcon/`, `sphincs/`, and `common/` files are subsets of
[PQClean at 0586a824fc0d49df0b6b6e9179d8d15d06d0974f](https://github.com/PQClean/PQClean/tree/0586a824fc0d49df0b6b6e9179d8d15d06d0974f).
The scheme sources are from `crypto_sign/falcon-padded-512/clean` and
`crypto_sign/sphincs-sha2-128f-simple/clean`. Their license notices are included
in their directories. Two SPHINCS+ files have trailing blank lines removed. The shared SHA2 and FIPS202 implementations contain
their public domain source notices.

Each top-level C translation unit includes one source file, preserving
upstream file boundaries and private symbols. The Go bridge provides
PQClean's random-byte callback using the system CSPRNG via `crypto/rand`.
The wrapper checks wire sizes before calling the C APIs. With CGO disabled,
key generation and signing return an unavailable-provider error and
verification returns false.

The public API uses PQClean's tagged 897-byte Falcon public key. EIP-8052
describes an EVM verification interface and lists the 896-byte packed
polynomial. The legacy signer does not register an EVM precompile. Existing
FN-DSA and SLH-DSA transaction algorithms keep their original encodings.

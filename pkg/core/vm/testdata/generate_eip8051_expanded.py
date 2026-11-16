"""Generate expanded-key fixtures from the pinned independent Python reference.

Run with PYTHONPATH pointing to ETHDILITHIUM's pythonref and NTT's
assets/pythonref. See eip8051_expanded.md for the reference revisions.
"""

import gzip
import json
import struct
from pathlib import Path

from dilithium_py.dilithium.default_parameters import Dilithium2
from dilithium_py.keccak_prng.keccak_prng_wrapper import Keccak256PRNG
from dilithium_py.shake.shake_wrapper import shake128, shake256


def pack_polynomials(matrix):
    return b"".join(
        struct.pack(">I", coefficient)
        for row in matrix._data
        for polynomial in row
        for coefficient in polynomial.coeffs
    )


def make_vector(name, xof, matrix_xof, scale):
    scheme = Dilithium2
    message = bytes(range(32))
    public, secret = scheme.key_derive(
        bytes([len(name)]) * 32, _xof=xof, _xof2=matrix_xof
    )
    rho, t1 = scheme._unpack_pk(public)
    tr = scheme._h(public, 32, _xof=xof)
    matrix = scheme._expand_matrix_from_seed(rho, _xof=matrix_xof)
    t1_hat = t1.scale(scale).to_ntt()

    # Adapt only the message and tr to the EIP's explicit wire fields.
    # The reference signs the formatted message passed to _sign_internal.
    unpack, hash_fn = scheme._unpack_sk, scheme._h

    def unpack_draft(key):
        rho, k, _, s1, s2, t0 = unpack(key)
        return rho, k, tr, s1, s2, t0

    def hash_draft(data, length, **kwargs):
        if data == public and length == 64:
            return tr
        return hash_fn(data, length, **kwargs)

    scheme._unpack_sk, scheme._h = unpack_draft, hash_draft
    try:
        signature = scheme._sign_internal(
            secret, message, bytes(32), _xof=xof, _xof2=matrix_xof
        )
        assert scheme._verify_internal(
            public, message, signature, _xof=xof, _xof2=matrix_xof
        )
    finally:
        scheme._unpack_sk, scheme._h = unpack, hash_fn

    expanded = pack_polynomials(matrix) + tr + pack_polynomials(t1_hat)
    assert len(expanded) == 20512 and len(signature) == 2420
    return dict(name=name, message=message.hex(), signature=signature.hex(), key=expanded.hex())


if __name__ == "__main__":
    vectors = [
        make_vector("SHAKE", shake256, shake128, 1),
        make_vector("Keccak", Keccak256PRNG, Keccak256PRNG, 1 << 13),
    ]
    Path(__file__).with_name("eip8051_expanded.json.gz").write_bytes(
        gzip.compress(json.dumps(vectors).encode(), mtime=0)
    )

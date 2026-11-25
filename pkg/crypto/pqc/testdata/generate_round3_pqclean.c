/* Fixture generator only. The deterministic RNG must never be used for production keys. */
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "crypto_sign/falcon-padded-512/clean/api.h"
#include "crypto_sign/sphincs-sha2-128f-simple/clean/api.h"

int PQCLEAN_randombytes(uint8_t *out, size_t n) {
    static uint32_t state = 42;
    for (size_t i = 0; i < n; ++i) {
        state = state * 1664525u + 1013904223u;
        out[i] = (uint8_t)(state >> 24);
    }
    return 0;
}

static void hex(const uint8_t *data, size_t size) {
    for (size_t i = 0; i < size; ++i) {
        printf("%02x", data[i]);
    }
    putchar('\n');
}

int main(int argc, char **argv) {
    uint8_t pk[897], sk[1281], sig[17088];
    size_t len;
    const uint8_t message[] = "eth2030 historical signature fixture";
    int falcon = argc == 2 && strcmp(argv[1], "falcon") == 0;
    if (falcon) {
        if (PQCLEAN_FALCONPADDED512_CLEAN_crypto_sign_keypair(pk, sk)
                || PQCLEAN_FALCONPADDED512_CLEAN_crypto_sign_signature(sig, &len, message, sizeof message - 1, sk)
                || PQCLEAN_FALCONPADDED512_CLEAN_crypto_sign_verify(sig, len, message, sizeof message - 1, pk)) {
            return 1;
        }
        hex(pk, 897);
        hex(sk, 1281);
    } else {
        if (PQCLEAN_SPHINCSSHA2128FSIMPLE_CLEAN_crypto_sign_keypair(pk, sk)
                || PQCLEAN_SPHINCSSHA2128FSIMPLE_CLEAN_crypto_sign_signature(sig, &len, message, sizeof message - 1, sk)
                || PQCLEAN_SPHINCSSHA2128FSIMPLE_CLEAN_crypto_sign_verify(sig, len, message, sizeof message - 1, pk)) {
            return 1;
        }
        hex(pk, 32);
        hex(sk, 64);
    }
    hex(sig, len);
    return 0;
}

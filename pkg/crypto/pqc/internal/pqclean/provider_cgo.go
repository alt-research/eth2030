//go:build cgo

package pqclean

/*
#cgo CFLAGS: -I${SRCDIR}/common -std=c99
#include "falcon/api.h"
#include "sphincs/api.h"
*/
import "C"

import (
	"crypto/rand"
	"unsafe"
)

//export PQCLEAN_randombytes
func PQCLEAN_randombytes(out *C.uint8_t, length C.size_t) C.int {
	// crypto/rand.Read fills the buffer or terminates the process.
	_, _ = rand.Read(unsafe.Slice((*byte)(unsafe.Pointer(out)), int(length)))
	return 0
}

func pointer(data []byte) *C.uint8_t {
	// Some C routines use memcpy even for a zero-length message.
	if len(data) == 0 {
		return new(C.uint8_t)
	}
	return (*C.uint8_t)(unsafe.Pointer(unsafe.SliceData(data)))
}

func (s Scheme) GenerateKey() ([]byte, []byte, error) {
	publicSize, privateSize, _ := s.sizes()
	public := make([]byte, publicSize)
	private := make([]byte, privateSize)
	var result C.int
	switch s {
	case Falcon512:
		result = C.PQCLEAN_FALCONPADDED512_CLEAN_crypto_sign_keypair(pointer(public), pointer(private))
	case SPHINCS128f:
		result = C.PQCLEAN_SPHINCSSHA2128FSIMPLE_CLEAN_crypto_sign_keypair(pointer(public), pointer(private))
	default:
		return nil, nil, ErrOperation
	}
	if result != 0 {
		return nil, nil, ErrOperation
	}
	return public, private, nil
}

func (s Scheme) Sign(private, message []byte) ([]byte, error) {
	_, privateSize, signatureSize := s.sizes()
	if privateSize == 0 || len(private) != privateSize {
		return nil, ErrKeySize
	}
	signature := make([]byte, signatureSize)
	var written C.size_t
	var result C.int
	switch s {
	case Falcon512:
		result = C.PQCLEAN_FALCONPADDED512_CLEAN_crypto_sign_signature(pointer(signature), &written, pointer(message), C.size_t(len(message)), pointer(private))
	case SPHINCS128f:
		result = C.PQCLEAN_SPHINCSSHA2128FSIMPLE_CLEAN_crypto_sign_signature(pointer(signature), &written, pointer(message), C.size_t(len(message)), pointer(private))
	}
	if result != 0 || written != C.size_t(signatureSize) {
		return nil, ErrOperation
	}
	return signature, nil
}

func (s Scheme) Verify(public, message, signature []byte) bool {
	publicSize, _, signatureSize := s.sizes()
	if len(public) != publicSize || len(signature) != signatureSize {
		return false
	}
	switch s {
	case Falcon512:
		return C.PQCLEAN_FALCONPADDED512_CLEAN_crypto_sign_verify(pointer(signature), C.size_t(len(signature)), pointer(message), C.size_t(len(message)), pointer(public)) == 0
	case SPHINCS128f:
		return C.PQCLEAN_SPHINCSSHA2128FSIMPLE_CLEAN_crypto_sign_verify(pointer(signature), C.size_t(len(signature)), pointer(message), C.size_t(len(message)), pointer(public)) == 0
	default:
		return false
	}
}

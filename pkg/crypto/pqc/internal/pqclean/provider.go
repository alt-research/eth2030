// Package pqclean provides the historical signature formats through pinned PQClean code.
package pqclean

import "errors"

var (
	ErrUnavailable = errors.New("historical PQ signature provider requires CGO")
	ErrKeySize     = errors.New("historical PQ signature provider: invalid private key length")
	ErrOperation   = errors.New("historical PQ signature provider: key generation or signing failed")
)

type Scheme uint8

const (
	Falcon512 Scheme = iota
	SPHINCS128f
)

func (s Scheme) sizes() (public, private, signature int) {
	switch s {
	case Falcon512:
		return 897, 1281, 666
	case SPHINCS128f:
		return 32, 64, 17088
	default:
		return 0, 0, 0
	}
}

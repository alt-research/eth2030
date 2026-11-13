//go:build !cgo

package pqclean

func (Scheme) GenerateKey() ([]byte, []byte, error) { return nil, nil, ErrUnavailable }
func (Scheme) Sign([]byte, []byte) ([]byte, error)  { return nil, ErrUnavailable }
func (Scheme) Verify([]byte, []byte, []byte) bool   { return false }

package rlp

import "testing"

func TestStreamLengthAdditionCannotOverflow(t *testing.T) {
	for _, prefix := range []byte{0xbf, 0xff} {
		t.Run(string([]byte{prefix}), func(t *testing.T) {
			input := []byte{prefix, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
			defer func() {
				if p := recover(); p != nil {
					t.Fatalf("declared length overflow panicked: %v", p)
				}
			}()
			s := NewStreamFromBytes(input)
			if prefix == 0xff {
				if _, err := s.List(); err == nil {
					t.Fatal("accepted overflowing list length")
				}
			} else {
				if _, err := s.Bytes(); err == nil {
					t.Fatal("accepted overflowing string length")
				}
			}
		})
	}
}

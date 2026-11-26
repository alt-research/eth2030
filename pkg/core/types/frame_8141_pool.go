package types

import (
	"encoding/binary"
	"errors"
)

var FrameExpiryVerifier = HexToAddress("0x81413f0cF12e9b6a49B1D0439E081c577D57FfFf")

var FrameExpiryCode = []byte{0x60, 0x08, 0x36, 0x14, 0x60, 0x0a, 0x57, 0x5f, 0x5f, 0xfd, 0x5b, 0x5f, 0x35, 0x60, 0xc0, 0x1c, 0x42, 0x11, 0x60, 0x16, 0x57, 0x00, 0x5b, 0x5f, 0x5f, 0xfd}

// ValidationPrefix8141 locates the end of the public-mempool validation prefix.
// This checks structure and declared work; successful approval is checked by execution.
func (tx *FrameTx8141) ValidationPrefix8141(timestamp uint64) (int, error) {
	return tx.ValidationPrefix8141WithExtensions(timestamp, FrameExtensions{})
}

func (tx *FrameTx8141) ValidationPrefix8141WithExtensions(timestamp uint64, extensions FrameExtensions) (int, error) {
	if err := tx.ValidateWithExtensions(extensions); err != nil {
		return 0, err
	}
	return tx.validationPrefix8141(timestamp)
}

func (tx *FrameTx8141) validationPrefix8141(timestamp uint64) (int, error) {
	if len(tx.Frames) == 0 {
		return 0, errors.New("Frame validation prefix is missing")
	}
	if tx.HasProofFrames8504() {
		logical, indices := tx.withoutProofFrames8504()
		end, err := logical.validationPrefix8141(timestamp)
		if err != nil {
			return 0, err
		}
		return indices[end], nil
	}
	start := 0
	if tx.Frames[0].Target != nil && *tx.Frames[0].Target == FrameExpiryVerifier {
		frame := tx.Frames[0]
		if frame.Mode != ModeVerify || len(frame.Data) != 8 || binary.BigEndian.Uint64(frame.Data) < timestamp {
			return 0, errors.New("Frame validation has expired")
		}
		start++
	}
	if start < len(tx.Frames) && tx.Frames[start].Target != nil && *tx.Frames[start].Target == RecentRootAddress {
		if !tx.Frames[start].IsRecentRootVerifier() {
			return 0, errors.New("invalid recent-root validation frame")
		}
		start++
	}
	if start < len(tx.Frames) && tx.Frames[start].Mode == ModeDefault && tx.Frames[start].Flags == 0 {
		start++
	}
	if start >= len(tx.Frames) {
		return 0, errors.New("Frame validation prefix is missing")
	}
	verify := tx.Frames[start]
	target := tx.Sender
	if verify.Target != nil {
		target = *verify.Target
	}
	if verify.Mode != ModeVerify || target != tx.Sender {
		return 0, errors.New("Frame validation prefix must verify the sender")
	}
	end := start
	switch verify.Flags {
	case FrameApproveExecution | FrameApprovePayment:
	case FrameApproveExecution:
		end++
		if end >= len(tx.Frames) || tx.Frames[end].Mode != ModeVerify || tx.Frames[end].Flags != FrameApprovePayment {
			return 0, errors.New("Frame validation prefix is missing payment approval")
		}
	default:
		return 0, errors.New("Frame validation prefix has invalid approval flags")
	}
	var execution, state uint64
	for _, signature := range tx.Signatures {
		switch signature.Scheme {
		case FrameSignatureArbitrary:
			execution += 100
		case FrameSignatureSecp256k1:
			execution += 2800
		case FrameSignatureP256:
			execution += 6700
		}
	}
	for _, frame := range tx.Frames[:end+1] {
		execution += frame.Limits.Execution
		state += frame.Limits.State
	}
	if execution > 100000 || state > 500000 {
		return 0, errors.New("Frame validation prefix exceeds the public work budget")
	}
	for _, frame := range tx.Frames[end+1:] {
		if frame.Mode == ModeVerify {
			return 0, errors.New("Frame public validation cannot continue after payment")
		}
	}
	return end, nil
}

// ValidationDeployIndex8141 skips the protocol verifiers. Call only after
// ValidationPrefix8141 has accepted the grammar.
func (tx *FrameTx8141) ValidationDeployIndex8141() int {
	if tx.HasProofFrames8504() {
		logical, indices := tx.withoutProofFrames8504()
		index := logical.ValidationDeployIndex8141()
		if index < 0 {
			return -1
		}
		return indices[index]
	}
	index := 0
	if index < len(tx.Frames) && tx.Frames[index].Target != nil && *tx.Frames[index].Target == FrameExpiryVerifier {
		index++
	}
	if index < len(tx.Frames) && tx.Frames[index].IsRecentRootVerifier() {
		index++
	}
	if index < len(tx.Frames) && tx.Frames[index].Mode == ModeDefault && tx.Frames[index].Flags == 0 {
		return index
	}
	return -1
}

// ValidationPrefix8250 uses the replacement signing hash and nonce rules.
func (tx *FrameTx8250Delta) ValidationPrefix8250(timestamp uint64) (int, error) {
	return tx.ValidationPrefix8250WithExtensions(timestamp, FrameExtensions{})
}

func (tx *FrameTx8250Delta) ValidationPrefix8250WithExtensions(timestamp uint64, extensions FrameExtensions) (int, error) {
	if err := tx.ValidateWithExtensions(extensions); err != nil {
		return 0, err
	}
	return tx.ExecutionFields().validationPrefix8141(timestamp)
}

func (tx *FrameTx8141) withoutProofFrames8504() (*FrameTx8141, []int) {
	logical := *tx
	logical.Frames = nil
	var indices []int
	for i, f := range tx.Frames {
		if f.Mode != ModeProof8504 {
			logical.Frames = append(logical.Frames, f)
			indices = append(indices, i)
		}
	}
	return &logical, indices
}

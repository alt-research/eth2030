package types

import (
	"errors"
	"math"
)

var (
	ErrNonceMax            = errors.New("sender nonce overflow")
	ErrEmptyAuthorizations = errors.New("set-code transaction requires authorizations")
	ErrBlobTxCreate        = errors.New("blob transaction requires recipient")
	ErrSetCodeTxCreate     = errors.New("set-code transaction requires recipient")
	ErrRecipientLength     = errors.New("invalid transaction recipient length")
)

// ValidateOrdinaryEnvelope applies fork-independent field invariants to the
// standard Ethereum envelopes. Experimental Frame/AA nonce policies remain
// with their own validators.
func ValidateOrdinaryEnvelope(tx *Transaction) error {
	if tx.Type() > SetCodeTxType {
		return nil
	}
	if tx.Nonce() == math.MaxUint64 {
		return ErrNonceMax
	}
	if tx.Type() == SetCodeTxType && len(tx.AuthorizationList()) == 0 {
		return ErrEmptyAuthorizations
	}
	return nil
}

// DecodeBlockTransaction rejects pooled blob wrappers at consensus ingress.
// Pool/RPC transport keeps using DecodePooledTransaction to retain sidecars.
func DecodeBlockTransaction(raw []byte) (*Transaction, error) {
	tx, err := DecodeTxRLP(raw)
	if err != nil {
		return nil, err
	}
	if tx.BlobSidecar() != nil {
		return nil, errors.New("blob network wrapper is not valid in a block body")
	}
	return tx, nil
}

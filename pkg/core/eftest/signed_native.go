package eftest

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/gaspool"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/core/vm"
	"github.com/eth2030/eth2030/crypto"
)

// RunSigned consumes the exact signed EEST envelope rather than synthesizing a
// legacy/dynamic-fee transaction from a subset of the fixture fields. It never
// treats a parser failure, panic or unrelated transition error as an expected
// protocol exception. Osaka and Amsterdam use this path by default.
func (st *StateTest) RunSigned(sub StateSubtest) (result *RunResult) {
	result = &RunResult{Name: st.Name, Fork: sub.Fork, Index: sub.Index}
	// A panic is always a harness failure, including on an expected-invalid case.
	defer func() {
		if r := recover(); r != nil {
			result.Passed = false
			result.Error = fmt.Errorf("native fixture panic: %v", r)
		}
	}()
	posts, ok := st.json.Post[sub.Fork]
	if !ok || sub.Index < 0 || sub.Index >= len(posts) {
		result.Error = fmt.Errorf("invalid subtest %s/%d", sub.Fork, sub.Index)
		return
	}
	post := posts[sub.Index]
	cfg := ForkConfig(sub.Fork)
	if cfg == nil {
		result.Error = fmt.Errorf("unsupported fork: %s", sub.Fork)
		return
	}
	fail := func(err error) *RunResult { result.Error = err; return result }
	root, err := fixtureBytes(post.Hash, 32)
	if err != nil {
		return fail(fmt.Errorf("expected state root: %w", err))
	}
	logs, err := fixtureBytes(post.Logs, 32)
	if err != nil {
		return fail(fmt.Errorf("expected logs hash: %w", err))
	}
	result.ExpectedRoot = types.BytesToHash(root)
	result.ExpectedLogs = types.BytesToHash(logs)
	if result.ExpectedRoot == (types.Hash{}) {
		return fail(errors.New("missing nonzero expected state root"))
	}
	// Signed bytes are authoritative, including access lists, blobs and every
	// authorization. Missing bytes must not silently fall back to a different type.
	raw, err := fixtureBytes(post.TxBytes, -1)
	if err != nil || len(raw) == 0 {
		return fail(fmt.Errorf("missing or invalid signed txbytes: %v", err))
	}
	tx, err := types.DecodeTxRLP(raw)
	if err != nil {
		return fail(fmt.Errorf("decode signed envelope: %w", err))
	}
	if tx.Type() > types.SetCodeTxType {
		return fail(fmt.Errorf("unsupported fixture envelope %d", tx.Type()))
	}
	sender, recoveryErr := types.MakeSigner(1, tx.Type()).Sender(tx)
	var signatureErr error
	if recoveryErr != nil {
		kind := errFixtureSignature
		if recoveryErr.Error() == "invalid chain ID for signer" {
			kind = errFixtureChainID
		}
		signatureErr = fmt.Errorf("%w: %v", kind, recoveryErr)
	}
	if st.json.Tx.Sender != "" {
		b, e := fixtureBytes(st.json.Tx.Sender, 20)
		if e != nil {
			return fail(e)
		}
		if signatureErr == nil && sender != types.BytesToAddress(b) {
			return fail(errors.New("fixture sender disagrees with signed envelope"))
		}
	}
	tx.SetSender(sender)
	// Validate selection metadata even though bytes, not the parameter arrays,
	// supply execution input. Malformed selectors are fixture errors.
	idx := post.Indexes
	if idx.Data < 0 || idx.Data >= len(st.json.Tx.Data) || idx.Gas < 0 || idx.Gas >= len(st.json.Tx.GasLimit) || idx.Value < 0 || idx.Value >= len(st.json.Tx.Value) {
		return fail(errors.New("fixture transaction selector out of bounds"))
	}
	data, err := fixtureBytes(st.json.Tx.Data[idx.Data], -1)
	if err != nil {
		return fail(err)
	}
	limit, err := fixtureNumber(st.json.Tx.GasLimit[idx.Gas], 64)
	if err != nil {
		return fail(err)
	}
	value, err := fixtureNumber(st.json.Tx.Value[idx.Value], 256)
	if err != nil {
		return fail(err)
	}
	if hex.EncodeToString(data) != hex.EncodeToString(tx.Data()) || limit.Uint64() != tx.Gas() || value.Cmp(tx.Value()) != 0 {
		return fail(errors.New("signed envelope disagrees with selected data/gas/value"))
	}
	db := state.NewMemoryStateDB()
	for addrHex, account := range st.json.Pre {
		addrBytes, e := fixtureBytes(addrHex, 20)
		if e != nil {
			return fail(e)
		}
		addr := types.BytesToAddress(addrBytes)
		balance, e := fixtureNumber(account.Balance, 256)
		if e != nil {
			return fail(e)
		}
		nonce, e := fixtureNumber(account.Nonce, 64)
		if e != nil {
			return fail(e)
		}
		code, e := fixtureBytes(account.Code, -1)
		if e != nil {
			return fail(e)
		}
		db.CreateAccount(addr)
		db.AddBalance(addr, balance)
		db.SetNonce(addr, nonce.Uint64())
		db.SetCode(addr, code)
		for keyHex, valueHex := range account.Storage {
			key, e := fixtureNumber(keyHex, 256)
			if e != nil {
				return fail(e)
			}
			value, e := fixtureNumber(valueHex, 256)
			if e != nil {
				return fail(e)
			}
			db.SetState(addr, types.BytesToHash(key.Bytes()), types.BytesToHash(value.Bytes()))
		}
	}
	db.FinalizePreState()
	env := st.json.Env
	for name, field := range map[string]string{"number": env.CurrentNumber, "timestamp": env.CurrentTimestamp, "gas limit": env.CurrentGasLimit} {
		if _, e := fixtureNumber(field, 64); e != nil {
			return fail(fmt.Errorf("environment %s: %w", name, e))
		}
	}
	if env.CurrentSlotNumber != "" {
		if _, err := fixtureNumber(env.CurrentSlotNumber, 64); err != nil {
			return fail(err)
		}
	}
	if cfg.IsLondon(hexToBigInt(env.CurrentNumber)) && env.CurrentBaseFee == "" {
		return fail(errors.New("missing London base fee"))
	}
	if cfg.IsMerge() && env.CurrentRandom == "" {
		return fail(errors.New("missing post-merge randomness"))
	}
	for _, field := range []string{env.CurrentBaseFee, env.CurrentDifficulty, env.ExcessBlobGas} {
		if field != "" {
			if _, e := fixtureNumber(field, 256); e != nil {
				return fail(e)
			}
		}
	}
	if _, e := fixtureBytes(env.CurrentCoinbase, 20); e != nil {
		return fail(e)
	}
	if env.CurrentRandom != "" {
		if _, e := fixtureBytes(env.CurrentRandom, 32); e != nil {
			return fail(e)
		}
	}
	header := st.buildHeader(cfg)
	db.SetTxContext(tx.Hash(), 0)
	applyErr := signatureErr
	if applyErr == nil {
		applyErr = gas.ValidateTransactionGasForFork(cfg, header.Time, tx)
	}
	if applyErr == nil {
		_, _, applyErr = execution.ApplyTransactionWithBALGetHash(cfg, func(n uint64) types.Hash { return crypto.Keccak256Hash([]byte(strconv.FormatUint(n, 10))) }, db, header, tx, new(gaspool.GasPool).AddGas(header.GasLimit), nil)
	}
	if post.ExpectException != "" {
		if !fixtureExceptionMatches(post.ExpectException, applyErr, tx, cfg, header) {
			return fail(fmt.Errorf("expected %q; actual native error: %v", post.ExpectException, applyErr))
		}
	} else if applyErr != nil {
		return fail(fmt.Errorf("unexpected native transaction error: %w", applyErr))
	}
	// Commit the actual native transition without repairing its account lifecycle.
	result.GotRoot, err = db.Commit()
	if err != nil {
		return fail(err)
	}
	result.StateDB = db
	result.GotLogs = computeLogsHash(db.GetLogs(tx.Hash()))
	if result.GotRoot != result.ExpectedRoot {
		return fail(fmt.Errorf("state root mismatch: expected %s, got %s", result.ExpectedRoot, result.GotRoot))
	}
	if result.GotLogs != result.ExpectedLogs {
		return fail(fmt.Errorf("logs hash mismatch: expected %s, got %s", result.ExpectedLogs, result.GotLogs))
	}
	result.Passed = true
	return
}

func fixtureBytes(s string, size int) ([]byte, error) {
	if !strings.HasPrefix(s, "0x") {
		return nil, fmt.Errorf("fixture bytes require 0x prefix: %q", s)
	}
	b, err := hex.DecodeString(s[2:])
	if err != nil {
		return nil, err
	}
	if size >= 0 && len(b) != size {
		return nil, fmt.Errorf("fixture bytes length %d, expected %d", len(b), size)
	}
	return b, nil
}
func fixtureNumber(s string, bits int) (*big.Int, error) {
	if !strings.HasPrefix(s, "0x") || len(s) == 2 {
		return nil, fmt.Errorf("invalid fixture quantity %q", s)
	}
	n, ok := new(big.Int).SetString(s[2:], 16)
	if !ok || n.Sign() < 0 || n.BitLen() > bits {
		return nil, fmt.Errorf("fixture quantity exceeds uint%d: %q", bits, s)
	}
	return n, nil
}

var errFixtureSignature = errors.New("native signed-envelope recovery rejected signature")
var errFixtureChainID = errors.New("native signed-envelope recovery rejected chain ID")

func fixtureExceptionMatches(expected string, err error, tx *types.Transaction, cfg *config.ChainConfig, header *types.Header) bool {
	if err == nil {
		return false
	}
	// Only explicitly classified errors count. Unknown identifiers fail closed.
	for _, item := range strings.Split(expected, "|") {
		item = strings.TrimSpace(item)
		switch item {
		case "TransactionException.GAS_LIMIT_EXCEEDS_MAXIMUM":
			if errors.Is(err, gas.ErrTxGasLimitExceeded) {
				return true
			}
		case "TransactionException.NONCE_MISMATCH_TOO_LOW":
			if errors.Is(err, execution.ErrNonceTooLow) {
				return true
			}
		case "TransactionException.NONCE_MISMATCH_TOO_HIGH":
			if errors.Is(err, execution.ErrNonceTooHigh) {
				return true
			}
		case "TransactionException.INSUFFICIENT_ACCOUNT_FUNDS":
			if errors.Is(err, execution.ErrInsufficientBalance) {
				return true
			}
		case "TransactionException.INTRINSIC_GAS_TOO_LOW", "TransactionException.INTRINSIC_GAS_BELOW_FLOOR_GAS_COST":
			intrinsic := execution.IntrinsicGas(tx.Data(), tx.To() == nil, cfg.IsShanghai(header.Time), uint64(len(tx.AuthorizationList())))
			for _, tuple := range tx.AccessList() {
				intrinsic += 2400 + 1900*uint64(len(tuple.StorageKeys))
			}

			floor := execution.CalldataFloorGas(tx.Data(), tx.To() == nil)
			if cfg.IsAmsterdam(header.Time) {
				sender, err := types.MakeSigner(1, tx.Type()).Sender(tx)
				if err != nil {
					return false
				}
				cost, err := gas.CalculateAmsterdamIntrinsic(sender, tx.To(), tx.Value(), tx.Data(), tx.AccessList(), uint64(len(tx.AuthorizationList())))
				if err != nil {
					return false
				}
				intrinsic, floor = cost.Gas, cost.Floor
			}

			if item == "TransactionException.INTRINSIC_GAS_TOO_LOW" && cfg.IsAmsterdam(header.Time) && strings.HasPrefix(err.Error(), "Amsterdam intrinsic/floor exceeds execution cap:") && max(intrinsic, floor) > vm.AmsterdamMaxExecutionGas {
				return true
			}
			// The legacy executor exposes the same error text for these two guards;
			// classify the actual violated bound, not merely any non-nil error.
			if strings.HasPrefix(err.Error(), "intrinsic gas too low: have ") || errors.Is(err, execution.ErrIntrinsicGasTooLow) {
				if item == "TransactionException.INTRINSIC_GAS_TOO_LOW" && tx.Gas() < intrinsic {
					return true
				}
				if item == "TransactionException.INTRINSIC_GAS_BELOW_FLOOR_GAS_COST" && tx.Gas() >= intrinsic && tx.Gas() < floor {
					return true
				}
			}

		case "TransactionException.INVALID_SIGNATURE_VRS":
			if errors.Is(err, errFixtureSignature) {
				return true
			}
		case "TransactionException.INVALID_CHAINID":
			if errors.Is(err, errFixtureChainID) {
				return true
			}
		case "TransactionException.SENDER_NOT_EOA":
			if strings.HasPrefix(err.Error(), "sender not an EOA") {
				return true
			}
		case "TransactionException.NONCE_IS_MAX":
			if tx.Nonce() == math.MaxUint64 && err.Error() == "sender nonce overflow" {
				return true
			}
		case "TransactionException.INITCODE_SIZE_EXCEEDED":
			if tx.To() == nil && errors.Is(err, vm.ErrMaxInitCodeSizeExceeded) {
				return true
			}
		case "TransactionException.INSUFFICIENT_MAX_FEE_PER_GAS":
			if header.BaseFee != nil && tx.GasFeeCap().Cmp(header.BaseFee) < 0 && strings.HasPrefix(err.Error(), "max fee per gas less than block base fee") {
				return true
			}
		case "TransactionException.PRIORITY_GREATER_THAN_MAX_FEE_PER_GAS":
			if tx.GasTipCap().Cmp(tx.GasFeeCap()) > 0 && (strings.HasPrefix(err.Error(), "max priority fee per gas higher") || err.Error() == "invalid transaction fee caps") {
				return true
			}
		case "TransactionException.TYPE_3_TX_ZERO_BLOBS":
			if tx.Type() == types.BlobTxType && len(tx.BlobHashes()) == 0 && err.Error() == "invalid blob transaction recipient or count" {
				return true
			}
		case "TransactionException.TYPE_3_TX_INVALID_BLOB_VERSIONED_HASH":
			if tx.Type() == types.BlobTxType && err.Error() == "invalid blob versioned hash" {
				for _, hash := range tx.BlobHashes() {
					if hash[0] != 1 {
						return true
					}
				}
			}
		case "TransactionException.INSUFFICIENT_MAX_FEE_PER_BLOB_GAS":
			if tx.Type() == types.BlobTxType && strings.HasPrefix(err.Error(), "blob fee cap below active blob base fee") {
				return true
			}
		case "TransactionException.TYPE_4_EMPTY_AUTHORIZATION_LIST":
			if tx.Type() == types.SetCodeTxType && tx.To() != nil && len(tx.AuthorizationList()) == 0 && err.Error() == "set-code transaction requires destination and authorizations" {
				return true
			}
		case "TransactionException.GAS_ALLOWANCE_EXCEEDED":
			if tx.Gas() > header.GasLimit && err.Error() == "gas pool exhausted" {
				return true
			}
		case "TransactionException.TYPE_3_TX_BLOB_COUNT_EXCEEDED":
			if err.Error() == "invalid blob transaction recipient or count" && tx.Type() == types.BlobTxType && uint64(len(tx.BlobHashes())) > gas.MaxBlobsPerTransaction(cfg, header.Time) && uint64(len(tx.BlobHashes())) <= gas.MaxBlobsForBlock(cfg, header.Time) {
				return true
			}
		case "TransactionException.TYPE_3_TX_MAX_BLOB_GAS_ALLOWANCE_EXCEEDED":
			if err.Error() == "invalid blob transaction recipient or count" && tx.Type() == types.BlobTxType && uint64(len(tx.BlobHashes())) > gas.MaxBlobsForBlock(cfg, header.Time) {
				return true
			}
		}
	}
	return false
}

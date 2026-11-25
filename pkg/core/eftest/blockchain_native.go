package eftest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/eth2030/eth2030/bal"
	"github.com/eth2030/eth2030/core/block"
	"github.com/eth2030/eth2030/core/chain"
	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/gas"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/state"
	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

// BlockchainFixture uses the signed block bytes as execution authority. JSON
// headers, receipts and sidecars are independent expected observations.
type BlockchainFixture struct {
	Network       string                     `json:"network"`
	GenesisHeader map[string]json.RawMessage `json:"genesisBlockHeader"`
	GenesisRLP    string                     `json:"genesisRLP"`
	Pre           map[string]stAccount       `json:"pre"`
	Post          map[string]stAccount       `json:"postState"`
	PostHash      string                     `json:"postStateHash"`
	LastHash      string                     `json:"lastblockhash"`
	Config        struct {
		Network string `json:"network"`
		ChainID string `json:"chainid"`
		Blobs   map[string]struct {
			Target   string `json:"target"`
			Max      string `json:"max"`
			Fraction string `json:"baseFeeUpdateFraction"`
		} `json:"blobSchedule"`
	} `json:"config"`
	Blocks []BlockchainFixtureBlock `json:"blocks"`
}
type BlockchainFixtureBlock struct {
	RLP               string                     `json:"rlp"`
	ExpectedException string                     `json:"expectException"`
	Header            map[string]json.RawMessage `json:"blockHeader"`
	BAL               json.RawMessage            `json:"blockAccessList"`
	Receipts          []struct {
		RLP  string `json:"rlp"`
		Hash string `json:"transactionHash"`
	} `json:"receipts"`
	Decoded *BlockchainFixtureBlock `json:"rlp_decoded"`
}
type BlockchainBlockResult struct {
	Index              int    `json:"index"`
	Hash               string `json:"hash,omitempty"`
	Accepted           bool   `json:"accepted"`
	ExpectedException  string `json:"expected_exception,omitempty"`
	AlternateRejection bool   `json:"alternate_rejection_for_independently_verified_multi_fault,omitempty"`
	Rejection          string `json:"rejection,omitempty"`
	StateRoot          string `json:"state_root,omitempty"`
	ReceiptsRoot       string `json:"receipts_root,omitempty"`
	BALHash            string `json:"bal_hash,omitempty"`
	HeaderGas          uint64 `json:"header_gas"`
}
type BlockchainResult struct {
	Passed       bool                    `json:"passed"`
	Error        string                  `json:"error,omitempty"`
	ExpectedHead string                  `json:"expected_head"`
	GotHead      string                  `json:"got_head,omitempty"`
	ExpectedRoot string                  `json:"expected_root,omitempty"`
	GotRoot      string                  `json:"got_root,omitempty"`
	Blocks       []BlockchainBlockResult `json:"blocks"`
}

func LoadBlockchainFixtures(path string) (map[string]*BlockchainFixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fixtures map[string]*BlockchainFixture
	if err = json.Unmarshal(data, &fixtures); err != nil {
		return nil, err
	}
	if len(fixtures) == 0 {
		return nil, errors.New("empty blockchain fixture file")
	}
	return fixtures, nil
}

func fixtureState(accounts map[string]stAccount) (*state.MemoryStateDB, error) {
	db := state.NewMemoryStateDB()
	for text, a := range accounts {
		raw, err := fixtureBytes(text, 20)
		if err != nil {
			return nil, err
		}
		address := types.BytesToAddress(raw)
		balance, err := fixtureNumber(a.Balance, 256)
		if err != nil {
			return nil, err
		}
		nonce, err := fixtureNumber(a.Nonce, 64)
		if err != nil {
			return nil, err
		}
		code, err := fixtureBytes(a.Code, -1)
		if err != nil {
			return nil, err
		}
		db.CreateAccount(address)
		db.AddBalance(address, balance)
		db.SetNonce(address, nonce.Uint64())
		db.SetCode(address, code)
		for key, value := range a.Storage {
			k, err := fixtureNumber(key, 256)
			if err != nil {
				return nil, err
			}
			v, err := fixtureNumber(value, 256)
			if err != nil {
				return nil, err
			}
			db.SetState(address, types.BytesToHash(k.Bytes()), types.BytesToHash(v.Bytes()))
		}
	}
	db.FinalizePreState()
	return db, nil
}

func (f *BlockchainFixture) nativeConfig() (*config.ChainConfig, error) {
	if f.Network != "Osaka" && f.Network != "Amsterdam" {
		return nil, fmt.Errorf("unsupported exact blockchain network %q", f.Network)
	}
	cfg := ForkConfig(f.Network)
	if f.Config.Network != f.Network {
		return nil, errors.New("fixture config network differs")
	}
	id, err := fixtureNumber(f.Config.ChainID, 256)
	if err != nil {
		return nil, err
	}
	cfg.ChainID = id
	cfg.RequireRequestPredeploys = true
	cfg.BlobScheduleConfig = &config.BlobScheduleConfig{}
	parsed := map[string]config.BlobConfig{}
	for name, b := range f.Config.Blobs {
		target, e := fixtureNumber(b.Target, 64)
		if e != nil {
			return nil, e
		}
		maximum, e := fixtureNumber(b.Max, 64)
		if e != nil {
			return nil, e
		}
		fraction, e := fixtureNumber(b.Fraction, 64)
		if e != nil {
			return nil, e
		}
		entry := &config.BlobConfig{Target: target.Uint64(), Max: maximum.Uint64(), UpdateFraction: fraction.Uint64()}
		parsed[name] = *entry
		switch name {
		case "Cancun":
			cfg.BlobScheduleConfig.Cancun = entry
		case "Prague":
			cfg.BlobScheduleConfig.Prague = entry
		case "Osaka":
			// Osaka is an inherited-name observation, not another parameter activation.
		case "BPO1":
			cfg.BlobScheduleConfig.BPO1 = entry
		case "BPO2":
			cfg.BlobScheduleConfig.BPO2 = entry
		case "Amsterdam":
			// Amsterdam inherits the latest BPO parameters.
		default:
			return nil, fmt.Errorf("unsupported blob schedule %q", name)
		}
	}
	for alias, inherited := range map[string]string{"Osaka": "Prague", "Amsterdam": "BPO2"} {
		if value, ok := parsed[alias]; ok {
			prior, present := parsed[inherited]
			if !present || value != prior {
				return nil, fmt.Errorf("fixture blob alias %s differs from %s", alias, inherited)
			}
		}
	}
	return cfg, cfg.ValidateBlobSchedule()
}

// RunNative imports actual encoded blocks through the node's Blockchain. A
// panic, fixture inconsistency or unclassified error never satisfies an expected
// invalid block. Rejected blocks must leave the canonical head/state unchanged.
func (f *BlockchainFixture) RunNative() (result BlockchainResult) {
	result.ExpectedHead = f.LastHash
	result.Blocks = []BlockchainBlockResult{}
	defer func() {
		if p := recover(); p != nil {
			result.Passed = false
			result.Error = fmt.Sprintf("native blockchain fixture panic: %v", p)
		}
	}()
	fail := func(err error) BlockchainResult { result.Error = err.Error(); return result }
	cfg, err := f.nativeConfig()
	if err != nil {
		return fail(err)
	}
	db, err := fixtureState(f.Pre)
	if err != nil {
		return fail(fmt.Errorf("pre-state: %w", err))
	}
	raw, err := fixtureBytes(f.GenesisRLP, -1)
	if err != nil {
		return fail(err)
	}
	genesis, err := types.DecodeBlockRLP(raw)
	if err != nil {
		return fail(fmt.Errorf("genesis RLP: %w", err))
	}
	if genesis.NumberU64() != 0 || len(genesis.Transactions()) != 0 {
		return fail(errors.New("fixture genesis is not empty block zero"))
	}
	if db.GetRoot() != genesis.Root() {
		return fail(errors.New("pre-state does not match genesis commitment"))
	}
	if err = checkFixtureHeader(f.GenesisHeader, genesis.Header()); err != nil {
		return fail(fmt.Errorf("genesis metadata: %w", err))
	}
	bc, err := chain.NewBlockchain(cfg, genesis, db, rawdb.NewMemoryDB())
	if err != nil {
		return fail(err)
	}
	for index, fixture := range f.Blocks {
		record := BlockchainBlockResult{Index: index, ExpectedException: fixture.ExpectedException}
		oldHead, oldRoot := bc.CurrentBlock().Hash(), bc.State().GetRoot()
		raw, err := fixtureBytes(fixture.RLP, -1)
		if err != nil {
			return fail(fmt.Errorf("block %d fixture bytes: %w", index, err))
		}
		candidate, decodeErr := types.DecodeBlockRLP(raw)
		var admissionErr error
		if decodeErr != nil {
			admissionErr = decodeErr
		} else {
			record.Hash = candidate.Hash().Hex()
			record.HeaderGas = candidate.GasUsed()
			// Independent framing check detects acceptance of malformed/trailing RLP.
			var reference []gethrlp.RawValue
			if err := gethrlp.DecodeBytes(raw, &reference); err != nil {
				return fail(fmt.Errorf("native accepted malformed block framing: %w", err))
			}
			sidecar := &fixture
			if fixture.Decoded != nil {
				sidecar = fixture.Decoded
			}
			if cfg.IsAmsterdam(candidate.Time()) {
				var encoded []byte
				encoded, admissionErr = fixtureBAL(sidecar.BAL)
				if admissionErr == nil {
					_, admissionErr = bal.DecodeForHeader(encoded, candidate.Header())
				}
			}
			if admissionErr == nil {
				var executed *execution.ProcessResult
				executed, admissionErr = bc.InsertBlockWithResult(candidate)
				if admissionErr == nil && fixture.ExpectedException == "" {
					if err = checkFixtureHeader(fixture.Header, candidate.Header()); err != nil {
						return fail(fmt.Errorf("block %d metadata: %w", index, err))
					}
					if executed == nil {
						return fail(errors.New("valid fixture block was already known and not executed"))
					}
					if len(fixture.Receipts) > 0 && len(fixture.Receipts) != len(executed.Receipts) {
						return fail(errors.New("fixture receipt count mismatch"))
					}
					for i, want := range fixture.Receipts {
						expected, e := fixtureBytes(want.RLP, -1)
						if e != nil {
							return fail(e)
						}
						got, e := executed.Receipts[i].EncodeRLP()
						if e != nil {
							return fail(e)
						}
						if !bytes.Equal(got, expected) {
							return fail(fmt.Errorf("block %d receipt %d bytes differ", index, i))
						}
						if want.Hash != "" && want.Hash != candidate.Transactions()[i].Hash().Hex() {
							return fail(errors.New("fixture receipt transaction hash differs"))
						}
					}
					record.StateRoot = candidate.Root().Hex()
					record.ReceiptsRoot = candidate.Header().ReceiptHash.Hex()
					if executed.BlockAccessList != nil {
						got, e := executed.BlockAccessList.EncodeRLP()
						if e != nil {
							return fail(e)
						}
						want, e := fixtureBAL(fixture.BAL)
						if e != nil {
							return fail(e)
						}
						if !bytes.Equal(got, want) {
							return fail(fmt.Errorf("block %d actual BAL bytes differ from reference", index))
						}
						stored, e := bc.BlockAccessList(candidate.Hash())
						if e != nil || !bytes.Equal(stored, want) {
							return fail(fmt.Errorf("block %d persisted BAL differs: %v", index, e))
						}
						record.BALHash = crypto.Keccak256Hash(got).Hex()
					}
				}
			}
		}
		record.Accepted = admissionErr == nil
		if fixture.ExpectedException == "" {
			if admissionErr != nil {
				return fail(fmt.Errorf("block %d rejected: %w", index, admissionErr))
			}
		} else {
			if admissionErr == nil {
				return fail(fmt.Errorf("block %d accepted expected exception %s", index, fixture.ExpectedException))
			}
			if !blockchainExceptionMatches(fixture.ExpectedException, admissionErr, decodeErr, candidate, cfg) {
				if !blockchainMultiFaultRejection(fixture.ExpectedException, admissionErr, candidate, cfg) {
					return fail(fmt.Errorf("block %d wrong rejection for %s: %w", index, fixture.ExpectedException, admissionErr))
				}
				record.AlternateRejection = true
			}
			if bc.CurrentBlock().Hash() != oldHead || bc.State().GetRoot() != oldRoot {
				return fail(errors.New("rejected block changed canonical state/head"))
			}
			record.Rejection = admissionErr.Error()
		}
		result.Blocks = append(result.Blocks, record)
	}
	result.GotHead = bc.CurrentBlock().Hash().Hex()
	result.GotRoot = bc.State().GetRoot().Hex()
	if result.GotHead != f.LastHash {
		return fail(errors.New("final canonical head differs"))
	}
	if f.Post != nil {
		expected, e := fixtureState(f.Post)
		if e != nil {
			return fail(e)
		}
		result.ExpectedRoot = expected.GetRoot().Hex()
	} else if f.PostHash != "" {
		result.ExpectedRoot = f.PostHash
	} else {
		return fail(errors.New("fixture missing final state expectation"))
	}
	if result.GotRoot != result.ExpectedRoot {
		return fail(errors.New("final full post-state differs"))
	}
	result.Passed = true
	return result
}

func checkFixtureHeader(fields map[string]json.RawMessage, h *types.Header) error {
	if len(fields) == 0 {
		return errors.New("missing expected header")
	}
	values := map[string]string{"parentHash": h.ParentHash.Hex(), "uncleHash": h.UncleHash.Hex(), "coinbase": h.Coinbase.Hex(), "stateRoot": h.Root.Hex(), "transactionsTrie": h.TxHash.Hex(), "receiptTrie": h.ReceiptHash.Hex(), "bloom": fmt.Sprintf("0x%x", h.Bloom[:]), "mixHash": h.MixDigest.Hex(), "nonce": fmt.Sprintf("0x%x", h.Nonce[:]), "extraData": fmt.Sprintf("0x%x", h.Extra), "hash": h.Hash().Hex()}
	numbers := map[string]*big.Int{"difficulty": h.Difficulty, "number": h.Number, "gasLimit": new(big.Int).SetUint64(h.GasLimit), "gasUsed": new(big.Int).SetUint64(h.GasUsed), "timestamp": new(big.Int).SetUint64(h.Time), "baseFeePerGas": h.BaseFee}
	for key, p := range map[string]*types.Hash{"withdrawalsRoot": h.WithdrawalsHash, "parentBeaconBlockRoot": h.ParentBeaconRoot, "requestsHash": h.RequestsHash, "blockAccessListHash": h.BlockAccessListHash} {
		if p != nil {
			values[key] = p.Hex()
		}
	}
	for key, p := range map[string]*uint64{"blobGasUsed": h.BlobGasUsed, "excessBlobGas": h.ExcessBlobGas, "slotNumber": h.SlotNumber} {
		if p != nil {
			numbers[key] = new(big.Int).SetUint64(*p)
		}
	}
	for key, data := range fields {
		var expected string
		if err := json.Unmarshal(data, &expected); err != nil {
			return err
		}
		if got, ok := numbers[key]; ok {
			n, e := fixtureNumber(expected, 256)
			if e != nil || got == nil || n.Cmp(got) != 0 {
				return fmt.Errorf("header %s differs", key)
			}
			continue
		}
		got, ok := values[key]
		if !ok {
			return fmt.Errorf("unsupported/absent expected header field %s", key)
		}
		if !strings.EqualFold(expected, got) {
			return fmt.Errorf("header %s differs: expected %s actual %s", key, expected, got)
		}
	}
	return nil
}

// fixtureBAL preserves supplied order and values; sorting here would repair an
// invalid reference sidecar and hide a client format-validation gap.
func fixtureBAL(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, errors.New("fixture missing BAL")
	}
	type change struct {
		Index   string `json:"blockAccessIndex"`
		Balance string `json:"postBalance"`
		Nonce   string `json:"postNonce"`
		Code    string `json:"newCode"`
		Value   string `json:"postValue"`
	}
	var accounts []struct {
		Address string `json:"address"`
		Storage []struct {
			Slot    string   `json:"slot"`
			Changes []change `json:"slotChanges"`
		} `json:"storageChanges"`
		Reads    []string `json:"storageReads"`
		Balances []change `json:"balanceChanges"`
		Nonces   []change `json:"nonceChanges"`
		Codes    []change `json:"codeChanges"`
	}
	if err := json.Unmarshal(raw, &accounts); err != nil {
		return nil, err
	}
	result := make([]any, 0, len(accounts))
	for _, a := range accounts {
		address, e := fixtureBytes(a.Address, 20)
		if e != nil {
			return nil, e
		}
		storage, reads, balances, nonces, codes := []any{}, []any{}, []any{}, []any{}, []any{}
		idx := func(c change) (*big.Int, error) { return fixtureNumber(c.Index, 32) }
		for _, s := range a.Storage {
			key, e := fixtureNumber(s.Slot, 256)
			if e != nil {
				return nil, e
			}
			changes := []any{}
			for _, c := range s.Changes {
				i, e := idx(c)
				if e != nil {
					return nil, e
				}
				v, e := fixtureNumber(c.Value, 256)
				if e != nil {
					return nil, e
				}
				changes = append(changes, []any{i, v})
			}
			storage = append(storage, []any{key, changes})
		}
		for _, key := range a.Reads {
			n, e := fixtureNumber(key, 256)
			if e != nil {
				return nil, e
			}
			reads = append(reads, n)
		}
		for _, c := range a.Balances {
			i, e := idx(c)
			if e != nil {
				return nil, e
			}
			v, e := fixtureNumber(c.Balance, 256)
			if e != nil {
				return nil, e
			}
			balances = append(balances, []any{i, v})
		}
		for _, c := range a.Nonces {
			i, e := idx(c)
			if e != nil {
				return nil, e
			}
			v, e := fixtureNumber(c.Nonce, 64)
			if e != nil {
				return nil, e
			}
			nonces = append(nonces, []any{i, v})
		}
		for _, c := range a.Codes {
			i, e := idx(c)
			if e != nil {
				return nil, e
			}
			v, e := fixtureBytes(c.Code, -1)
			if e != nil {
				return nil, e
			}
			codes = append(codes, []any{i, v})
		}
		result = append(result, []any{address, storage, reads, balances, nonces, codes})
	}
	return gethrlp.EncodeToBytes(result)
}

func blockchainExceptionMatches(expected string, err, decodeErr error, b *types.Block, cfg *config.ChainConfig) bool {
	for _, kind := range strings.Split(expected, "|") {
		if decodeErr != nil && fixtureDecodeExceptionMatches(kind, decodeErr) {
			return true
		}
		if strings.HasPrefix(kind, "TransactionException.") && b != nil {
			for _, tx := range b.Transactions() {
				for cause := err; cause != nil; cause = errors.Unwrap(cause) {
					if fixtureExceptionMatches(kind, cause, tx, cfg, b.Header()) {
						return true
					}
					text := cause.Error()
					switch kind {
					case "TransactionException.GAS_ALLOWANCE_EXCEEDED":
						if text == "gas pool exhausted" {
							return true
						}
					case "TransactionException.INVALID_SIGNATURE_VRS":
						if text == "invalid transaction signature" || text == "invalid signature values" || strings.HasPrefix(text, "invalid signature:") {
							return true
						}
					case "TransactionException.INVALID_CHAINID":
						if text == "invalid chain ID for signer" {
							return true
						}
					case "TransactionException.TYPE_3_TX_MAX_BLOB_GAS_ALLOWANCE_EXCEEDED", "TransactionException.TYPE_3_TX_BLOB_COUNT_EXCEEDED":
						if strings.HasPrefix(text, "block blob gas used exceeds maximum:") {
							var count uint64
							for _, item := range b.Transactions() {
								count += uint64(len(item.BlobHashes()))
							}
							if kind == "TransactionException.TYPE_3_TX_MAX_BLOB_GAS_ALLOWANCE_EXCEEDED" && count > gas.MaxBlobsForBlock(cfg, b.Time()) {
								return true
							}
							if tx.Type() == types.BlobTxType && uint64(len(tx.BlobHashes())) > gas.MaxBlobsPerTransaction(cfg, b.Time()) {
								return true
							}
						}
					}
				}
			}
		}

		switch kind {
		case "BlockException.RLP_STRUCTURES_ENCODING":
			if decodeErr != nil {
				return true
			}
		case "BlockException.INVALID_STATE_ROOT":
			if errors.Is(err, block.ErrInvalidStateRoot) {
				return true
			}
		case "BlockException.INVALID_GAS_USED", "BlockException.GAS_USED_OVERFLOW":
			if errors.Is(err, block.ErrInvalidGasUsed) || errors.Is(err, block.ErrInvalidGasUsedTotal) {
				return true
			}
		case "BlockException.INVALID_GASLIMIT":
			if errors.Is(err, block.ErrInvalidGasLimit) {
				return true
			}
		case "BlockException.INVALID_BASEFEE_PER_GAS":
			if errors.Is(err, block.ErrInvalidBaseFee) {
				return true
			}
		case "BlockException.INVALID_WITHDRAWALS_ROOT":
			if errors.Is(err, block.ErrInvalidWithdrawalsRoot) {
				return true
			}
		case "BlockException.INVALID_REQUESTS":
			if errors.Is(err, block.ErrInvalidRequestHash) {
				return true
			}
		case "BlockException.INVALID_BLOCK_ACCESS_LIST", "BlockException.INVALID_BAL_HASH":
			if errors.Is(err, block.ErrInvalidBlockAccessList) || strings.Contains(err.Error(), "BAL ") || strings.Contains(err.Error(), "block access list") {
				return true
			}
		case "BlockException.BLOCK_ACCESS_LIST_GAS_LIMIT_EXCEEDED":
			if errors.Is(err, bal.ErrBALSizeExceeded) {
				return true
			}
		case "BlockException.INCORRECT_BLOCK_FORMAT":
			if strings.Contains(err.Error(), "BAL ") {
				return true
			}
		case "BlockException.RLP_BLOCK_LIMIT_EXCEEDED":
			if strings.HasPrefix(err.Error(), "RLP block size ") {
				return true
			}
		case "BlockException.INVALID_DEPOSIT_EVENT_LAYOUT":
			if strings.Contains(err.Error(), "deposit requests:") {
				return true
			}
		case "BlockException.SYSTEM_CONTRACT_CALL_FAILED":
			if strings.Contains(err.Error(), "system call") || strings.Contains(err.Error(), "no code at") {
				return true
			}
		case "BlockException.INCORRECT_EXCESS_BLOB_GAS":
			if strings.Contains(err.Error(), "excess blob gas") {
				return true
			}
		case "BlockException.INCORRECT_BLOB_GAS_USED", "BlockException.BLOB_GAS_USED_ABOVE_LIMIT":
			if strings.Contains(err.Error(), "blob gas used") || strings.Contains(err.Error(), "blob gas limit") {
				return true
			}
		default:
			if strings.HasPrefix(kind, "TransactionException.") && b != nil {
				for _, tx := range b.Transactions() {
					if fixtureExceptionMatches(kind, err, tx, cfg, b.Header()) {
						return true
					}
				}
			}
		}
	}
	return false
}

// Some upstream typed-envelope vectors deliberately have two invalid fields
// (e.g. invalid V/R/S and an empty type-4 authorization list). Error precedence
// is not a consensus rule. Qualify only these independently proven alternatives,
// and report them separately rather than claiming the signature branch ran.
func blockchainMultiFaultRejection(expected string, err error, b *types.Block, cfg *config.ChainConfig) bool {
	if b == nil {
		return false
	}
	for cause := err; cause != nil; cause = errors.Unwrap(cause) {
		recovery := cause.Error() == "public key recovery failed"
		for _, tx := range b.Transactions() {
			empty := errors.Is(cause, types.ErrEmptyAuthorizations) && tx.Type() == types.SetCodeTxType && len(tx.AuthorizationList()) == 0
			if !empty && !recovery {
				continue
			}
			raw, e := tx.EncodeRLP()
			if e != nil {
				continue
			}
			var independent gethtypes.Transaction
			if independent.UnmarshalBinary(raw) != nil {
				continue
			}
			_, sigErr := gethtypes.Sender(gethtypes.LatestSignerForChainID(cfg.ChainID), &independent)
			if sigErr == nil {
				continue
			}
			for _, kind := range strings.Split(expected, "|") {
				if kind == "TransactionException.INVALID_CHAINID" && errors.Is(sigErr, gethtypes.ErrInvalidChainId) {
					return true
				}
				if kind == "TransactionException.INVALID_SIGNATURE_VRS" && !errors.Is(sigErr, gethtypes.ErrInvalidChainId) {
					return true
				}
			}
		}
	}
	return false
}

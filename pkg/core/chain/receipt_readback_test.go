package chain

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/config"
	"github.com/eth2030/eth2030/core/rawdb"
	"github.com/eth2030/eth2030/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
	gethtrie "github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

func TestColdDiskReceiptDerivedFieldsMatchIndependentSDK(t *testing.T) {
	key, err := gethcrypto.HexToECDSA(fmt.Sprintf("%064x", 1))
	if err != nil {
		t.Fatal(err)
	}
	target := gethcommon.Address{19: 0x42}
	zero := uint64(0)
	cfg := &config.ChainConfig{ChainID: big.NewInt(1337), CancunTime: &zero}
	sdkConfig := *params.AllDevChainProtocolChanges
	sdkConfig.PragueTime, sdkConfig.OsakaTime, sdkConfig.AmsterdamTime, sdkConfig.BogotaTime = nil, nil, nil, nil
	auth, err := gethtypes.SignSetCode(key, gethtypes.SetCodeAuthorization{ChainID: *uint256.NewInt(1337), Address: target, Nonce: 9})
	if err != nil {
		t.Fatal(err)
	}
	candidates := []struct {
		name      string
		tx        gethtypes.TxData
		signer    gethtypes.Signer
		base      *big.Int
		postState bool
		highS     bool
	}{
		{"legacy protected create", &gethtypes.LegacyTx{Nonce: 128, GasPrice: big.NewInt(71), Gas: 150000, Value: new(big.Int)}, gethtypes.NewEIP155Signer(big.NewInt(1337)), big.NewInt(31), false, false},
		{"frontier high S", &gethtypes.LegacyTx{Nonce: 2, To: &target, GasPrice: big.NewInt(71), Gas: 50000, Value: new(big.Int)}, gethtypes.FrontierSigner{}, nil, true, true},
		{"access list", &gethtypes.AccessListTx{ChainID: big.NewInt(1337), Nonce: 3, To: &target, GasPrice: big.NewInt(71), Gas: 50000, Value: new(big.Int)}, gethtypes.LatestSignerForChainID(big.NewInt(1337)), big.NewInt(31), false, false},
		{"dynamic tip", &gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Nonce: 4, To: &target, GasFeeCap: big.NewInt(100), GasTipCap: big.NewInt(7), Gas: 50000, Value: new(big.Int)}, gethtypes.LatestSignerForChainID(big.NewInt(1337)), big.NewInt(31), false, false},
		{"dynamic fee cap failed create", &gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Nonce: 5, GasFeeCap: big.NewInt(35), GasTipCap: big.NewInt(7), Gas: 150000, Value: new(big.Int)}, gethtypes.LatestSignerForChainID(big.NewInt(1337)), big.NewInt(31), false, false},
		{"dynamic zero base", &gethtypes.DynamicFeeTx{ChainID: big.NewInt(1337), Nonce: 6, To: &target, GasFeeCap: big.NewInt(100), GasTipCap: big.NewInt(7), Gas: 50000, Value: new(big.Int)}, gethtypes.LatestSignerForChainID(big.NewInt(1337)), new(big.Int), false, false},
		{"blob", &gethtypes.BlobTx{ChainID: uint256.NewInt(1337), Nonce: 7, To: target, GasFeeCap: uint256.NewInt(100), GasTipCap: uint256.NewInt(7), Gas: 50000, Value: new(uint256.Int), BlobFeeCap: uint256.NewInt(100), BlobHashes: []gethcommon.Hash{{0: 1, 31: 1}, {0: 1, 31: 2}}}, gethtypes.LatestSignerForChainID(big.NewInt(1337)), big.NewInt(31), false, false},
		{"set code", &gethtypes.SetCodeTx{ChainID: uint256.NewInt(1337), Nonce: 8, To: target, GasFeeCap: uint256.NewInt(100), GasTipCap: uint256.NewInt(7), Gas: 150000, Value: new(uint256.Int), AuthList: []gethtypes.SetCodeAuthorization{auth}}, gethtypes.LatestSignerForChainID(big.NewInt(1337)), big.NewInt(31), false, false},
	}
	for _, tc := range candidates {
		t.Run(tc.name, func(t *testing.T) {
			signed, err := gethtypes.SignTx(gethtypes.NewTx(tc.tx), tc.signer, key)
			if err != nil {
				t.Fatal(err)
			}
			if tc.highS {
				v, r, s := signed.RawSignatureValues()
				signed = gethtypes.NewTx(&gethtypes.LegacyTx{Nonce: signed.Nonce(), To: signed.To(), Value: signed.Value(), Gas: signed.Gas(), GasPrice: signed.GasPrice(), V: new(big.Int).SetUint64(27 + ((v.Uint64() - 27) ^ 1)), R: r, S: new(big.Int).Sub(gethcrypto.S256().Params().N, s)})
			}
			raw, err := signed.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			native, err := types.DecodeTxRLP(raw)
			if err != nil {
				t.Fatal(err)
			}
			receipt := &types.Receipt{Type: signed.Type(), Status: 0, CumulativeGasUsed: 42424, Logs: []*types.Log{{Address: types.Address(target), Topics: []types.Hash{{31: 7}}, Data: []byte{1, 2, 3}}}}
			if tc.postState {
				receipt.PostState = bytes.Repeat([]byte{0x19}, 32)
			}
			receipt.Bloom = types.LogsBloom(receipt.Logs)
			encodedReceipt, err := receipt.EncodeRLP()
			if err != nil {
				t.Fatal(err)
			}
			var independent gethtypes.Receipt
			if err := independent.UnmarshalBinary(encodedReceipt); err != nil {
				t.Fatal(err)
			}
			excess := uint64(20000000)
			header := &types.Header{Number: big.NewInt(41), Time: 100, BaseFee: tc.base, GasLimit: 1000000, Difficulty: new(big.Int), ReceiptHash: types.Hash(gethtypes.DeriveSha(gethtypes.Receipts{&independent}, gethtrie.NewStackTrie(nil)))}
			if tc.base != nil {
				withdrawals := types.EmptyRootHash
				header.WithdrawalsHash = &withdrawals
				header.ExcessBlobGas = &excess
				header.BlobGasUsed = &zero
			}
			encodedHeader, err := header.EncodeRLP()
			if err != nil {
				t.Fatal(err)
			}
			var sdkHeader gethtypes.Header
			if err := gethrlp.DecodeBytes(encodedHeader, &sdkHeader); err != nil {
				t.Fatal(err)
			}
			if types.Hash(sdkHeader.Hash()) != header.Hash() {
				t.Fatal("complete header commitment mismatch")
			}
			var blobPrice *big.Int
			if signed.Type() == gethtypes.BlobTxType {
				blobPrice = eip4844.CalcBlobFee(&sdkConfig, &sdkHeader)
			}
			independent.DeriveFields(tc.signer, gethtypes.DeriveReceiptContext{BlockHash: sdkHeader.Hash(), BlockNumber: 41, BlockTime: 100, BaseFee: tc.base, BlobGasPrice: blobPrice, GasUsed: 42424, Tx: signed})
			path := t.TempDir()
			db, err := rawdb.NewFileDB(path)
			if err != nil {
				t.Fatal(err)
			}
			bc := &Blockchain{db: db, config: cfg}
			if err := rawdb.WriteHeader(db, 41, header.Hash(), encodedHeader); err != nil {
				t.Fatal(err)
			}
			body, err := encodeBlockBody(&types.Body{Transactions: []*types.Transaction{native}})
			if err != nil {
				t.Fatal(err)
			}
			if err := rawdb.WriteBody(db, 41, header.Hash(), body); err != nil {
				t.Fatal(err)
			}
			bc.writeReceipts(41, header.Hash(), []*types.Receipt{receipt})
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			db, err = rawdb.NewFileDB(path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			bc = &Blockchain{db: db, config: cfg}
			block := bc.readBlock(header.Hash())
			if block == nil {
				t.Fatal("cold body missing")
			}
			got := bc.readReceiptsFromDB(41, header.Hash(), block.Transactions())
			if len(got) != 1 {
				t.Fatal("cold receipt missing")
			}
			r := got[0]
			if r.EffectiveGasPrice == nil || r.EffectiveGasPrice.Cmp(independent.EffectiveGasPrice) != 0 {
				t.Fatalf("cold effectiveGasPrice=%v, independent=%v", r.EffectiveGasPrice, independent.EffectiveGasPrice)
			}
			from, err := gethtypes.Sender(tc.signer, signed)
			if err != nil {
				t.Fatal(err)
			}
			if block.Transactions()[0].Sender() == nil || gethcommon.Address(*block.Transactions()[0].Sender()) != from {
				t.Fatal("cold signed transaction lost authenticated sender")
			}
			if gethcommon.Address(r.ContractAddress) != independent.ContractAddress {
				t.Fatalf("cold creation address=%s, independent=%s", r.ContractAddress, independent.ContractAddress)
			}
			if r.BlobGasUsed != independent.BlobGasUsed || ((r.BlobGasPrice == nil) != (independent.BlobGasPrice == nil)) || r.BlobGasPrice != nil && r.BlobGasPrice.Cmp(independent.BlobGasPrice) != 0 {
				t.Fatalf("cold blob fee fields=%d/%v, independent=%d/%v", r.BlobGasUsed, r.BlobGasPrice, independent.BlobGasUsed, independent.BlobGasPrice)
			}
			readBytes, err := r.EncodeRLP()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(readBytes, encodedReceipt) {
				t.Fatal("metadata restoration changed consensus receipt bytes")
			}
			if r.TxHash != native.Hash() || r.GasUsed != 42424 || r.Logs[0].TxHash != native.Hash() || r.Logs[0].BlockHash != header.Hash() {
				t.Fatal("receipt/log locations lost")
			}
		})
	}
}

package eth

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/p2p"
	getheth "github.com/ethereum/go-ethereum/eth/protocols/eth"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

func TestSparseCellsFlatIndependentSDKWire(t *testing.T) {
	var mask types.CellMask
	mask[0] = 1
	mask[15] = 0x80
	request := GetCellsPacket{RequestID: 23, Hashes: []types.Hash{{31: 7}}, Mask: mask}
	raw, err := gethrlp.EncodeToBytes(request)
	if err != nil {
		t.Fatal(err)
	}
	var sdk getheth.GetCellsRequestPacket
	if err := gethrlp.DecodeBytes(raw, &sdk); err != nil || sdk.RequestId != 23 || sdk.Mask != mask || len(sdk.Hashes) != 1 {
		t.Fatalf("flat SDK request: %x %v", raw, err)
	}
	reply := CellsPacket{RequestID: 23, Hashes: request.Hashes, Cells: [][][]byte{{make([]byte, 2048), make([]byte, 2048)}}, Mask: mask}
	reply.Cells[0][1][31] = 7
	raw, err = gethrlp.EncodeToBytes(reply)
	if err != nil {
		t.Fatal(err)
	}
	var response getheth.CellsPacket
	if err := gethrlp.DecodeBytes(raw, &response); err != nil {
		t.Fatal(err)
	}
	groups, err := response.Cells.Items()
	if err != nil {
		t.Fatal(err)
	}
	cells, err := groups[0].Items()
	if err != nil || len(cells) != 2 || cells[1][31] != 7 || response.RequestId != 23 || response.Mask != mask {
		t.Fatalf("SDK cell ordering: %v", err)
	}
}
func TestSparsePooledBlobsAreRawEmptyList(t *testing.T) {
	tx := types.NewTransaction(&types.BlobTx{ChainID: big.NewInt(1), GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(100), Gas: 100000, BlobFeeCap: big.NewInt(10), BlobHashes: []types.Hash{{0: 1}}, Value: new(big.Int), V: new(big.Int), R: big.NewInt(1), S: big.NewInt(1)})
	tx.SetBlobSidecar(&types.BlobSidecarData{Version: 1, Blobs: [][]byte{make([]byte, 131072)}, Commitments: [][]byte{make([]byte, 48)}, Proofs: make([][]byte, 128)})
	for i := range tx.BlobSidecar().Proofs {
		tx.BlobSidecar().Proofs[i] = make([]byte, 48)
	}
	raw, err := (pooledTransactions72{RequestID: 7, Transactions: []*types.Transaction{tx}}).EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	var outer struct {
		ID  uint64
		Txs []gethrlp.RawValue
	}
	if err := gethrlp.DecodeBytes(raw, &outer); err != nil {
		t.Fatal(err)
	}
	var encoded []byte
	if err := gethrlp.DecodeBytes(outer.Txs[0], &encoded); err != nil {
		t.Fatal(err)
	}
	var fields []gethrlp.RawValue
	if err := gethrlp.DecodeBytes(encoded[1:], &fields); err != nil || !bytes.Equal(fields[2], []byte{0xc0}) {
		t.Fatalf("wrong elision %x %v", fields, err)
	}
	var sdk getheth.PooledTransactionsPacket
	if err := gethrlp.DecodeBytes(raw, &sdk); err != nil {
		t.Fatal(err)
	}
	items, err := sdk.List.Items()
	if err != nil || len(items) != 1 || len(items[0].BlobTxSidecar().Blobs) != 0 || len(items[0].BlobTxSidecar().Proofs) != 128 {
		t.Fatalf("SDK metadata consumer: %v", err)
	}
	var packet pooledTransactions72
	if err := packet.DecodeRLP(raw); err != nil {
		t.Fatal(err)
	}
	fields[2] = []byte{0x80}
	badBody, err := gethrlp.EncodeToBytes(fields)
	if err != nil {
		t.Fatal(err)
	}
	outer.Txs[0], err = gethrlp.EncodeToBytes(append([]byte{3}, badBody...))
	if err != nil {
		t.Fatal(err)
	}
	bad, err := gethrlp.EncodeToBytes(outer)
	if err != nil {
		t.Fatal(err)
	}
	if err := packet.DecodeRLP(bad); err == nil {
		t.Fatal("empty string accepted for blobs")
	}
	full, err := (p2p.PooledTransactionsPacket{RequestID: 7, Transactions: []*types.Transaction{tx}}).EncodeRLP()
	if err != nil {
		t.Fatal(err)
	}
	if err := packet.DecodeRLP(full); err == nil {
		t.Fatal("eth/72 accepted non-elided blobs")
	}
	if len(tx.BlobSidecar().Blobs) != 1 {
		t.Fatal("sender's full sidecar mutated")
	}
}

func TestSparseWireRejectsMaskAndCellAliases(t *testing.T) {
	for _, width := range []int{0, 1, 15, 17, 32} {
		raw, _ := gethrlp.EncodeToBytes([]any{uint64(1), []types.Hash{{1}}, make([]byte, width)})
		var request GetCellsPacket
		if err := request.DecodeRLP(raw); err == nil {
			t.Fatalf("mask width %d accepted", width)
		}
	}
	for _, width := range []int{0, 1, 2047, 2049} {
		raw, _ := gethrlp.EncodeToBytes(CellsPacket{RequestID: 1, Hashes: []types.Hash{{1}}, Cells: [][][]byte{{make([]byte, width)}}, Mask: types.CellMask{1}})
		var response CellsPacket
		if err := response.DecodeRLP(raw); err == nil {
			t.Fatalf("cell width %d accepted", width)
		}
	}
}

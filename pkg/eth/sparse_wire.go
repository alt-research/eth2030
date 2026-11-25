package eth

import (
	"bytes"
	"fmt"

	"github.com/eth2030/eth2030/core/types"
	"github.com/eth2030/eth2030/crypto/bls"
	"github.com/eth2030/eth2030/p2p"
	gethrlp "github.com/ethereum/go-ethereum/rlp"
)

const ETH72 = 72
const GetCellsMsg = 0x14
const CellsMsg = 0x15
const MaxCellHashes = 64
const CellResponseLimit = 2 << 20

type NewPooledTxHashesMsg72 struct {
	Types  []byte
	Sizes  []uint32
	Hashes []types.Hash
	Mask   types.CellMask
}

// Fields are flat: devp2p eth.md and merged Geth PR 35428 include request IDs.
type GetCellsPacket struct {
	RequestID uint64
	Hashes    []types.Hash
	Mask      types.CellMask
}
type CellsPacket struct {
	RequestID uint64
	Hashes    []types.Hash
	Cells     [][][]byte
	Mask      types.CellMask
}

func sparseMetadata(tx *types.Transaction) (*types.Transaction, error) {
	if tx.Type() != types.BlobTxType {
		return tx, nil
	}
	raw, err := tx.EncodeRLP()
	if err != nil {
		return nil, err
	}
	out, err := types.DecodeTxRLP(raw)
	if err != nil {
		return nil, err
	}
	side := tx.BlobSidecar()
	if side == nil {
		return nil, fmt.Errorf("missing blob metadata")
	}
	copySide := &types.BlobSidecarData{Version: 1}
	for _, c := range side.Commitments {
		copySide.Commitments = append(copySide.Commitments, bytes.Clone(c))
	}
	switch side.Version {
	case 1:
		for _, p := range side.Proofs {
			copySide.Proofs = append(copySide.Proofs, bytes.Clone(p))
		}
	case 0:
		if !types.HasFullBlobSidecar(tx) {
			return nil, fmt.Errorf("legacy blob data is incomplete")
		}
		for _, blob := range side.Blobs {
			_, proofs, err := bls.DefaultKZGBackend().ComputeCellsAndProofs(blob)
			if err != nil {
				return nil, err
			}
			for _, p := range proofs {
				copySide.Proofs = append(copySide.Proofs, bytes.Clone(p[:]))
			}
		}
	default:
		return nil, fmt.Errorf("unsupported blob wrapper version")
	}
	out.SetBlobSidecar(copySide)
	return out, nil
}

type pooledTransactions72 p2p.PooledTransactionsPacket

func (p pooledTransactions72) EncodeRLP() ([]byte, error) {
	packet := p2p.PooledTransactionsPacket{RequestID: p.RequestID}
	for _, tx := range p.Transactions {
		meta, err := sparseMetadata(tx)
		if err != nil {
			return nil, err
		}
		packet.Transactions = append(packet.Transactions, meta)
	}
	return packet.EncodeRLP()
}
func (p *pooledTransactions72) DecodeRLP(data []byte) error {
	var wire struct {
		ID           uint64
		Transactions []gethrlp.RawValue
	}
	if err := gethrlp.DecodeBytes(data, &wire); err != nil {
		return err
	}
	if len(wire.Transactions) > 256 {
		return fmt.Errorf("too many pooled transactions")
	}
	var packet p2p.PooledTransactionsPacket
	if err := packet.DecodeRLP(data); err != nil {
		return err
	}
	for i, tx := range packet.Transactions {
		if tx.Type() != types.BlobTxType {
			continue
		}
		var raw []byte
		if err := gethrlp.DecodeBytes(wire.Transactions[i], &raw); err != nil {
			return err
		}
		var fields []gethrlp.RawValue
		if len(raw) < 2 {
			return fmt.Errorf("missing blob wrapper")
		}
		if err := gethrlp.DecodeBytes(raw[1:], &fields); err != nil {
			return err
		}
		canonical, err := types.EncodePooledTransaction(tx)
		if err != nil || !bytes.Equal(canonical, raw) {
			return fmt.Errorf("noncanonical sparse pooled transaction")
		}
		if len(fields) != 5 || !bytes.Equal(fields[1], []byte{1}) || !bytes.Equal(fields[2], []byte{0xc0}) {
			return fmt.Errorf("eth/72 requires version one with blobs elided as c0")
		}
	}
	*p = pooledTransactions72(packet)
	return nil
}

func (p *GetCellsPacket) DecodeRLP(raw []byte) error {
	type plain GetCellsPacket
	var packet plain
	if len(raw) > 64*33+64 {
		return fmt.Errorf("cell request too large")
	}
	if err := gethrlp.DecodeBytes(raw, &packet); err != nil {
		return err
	}
	if len(packet.Hashes) > MaxCellHashes {
		return fmt.Errorf("too many cell hashes")
	}
	*p = GetCellsPacket(packet)
	return nil
}
func (p *CellsPacket) DecodeRLP(raw []byte) error {
	type plain CellsPacket
	var packet plain
	if len(raw) > CellResponseLimit+65536 {
		return fmt.Errorf("cell response too large")
	}
	if err := gethrlp.DecodeBytes(raw, &packet); err != nil {
		return err
	}
	if len(packet.Hashes) > MaxCellHashes || len(packet.Hashes) != len(packet.Cells) {
		return fmt.Errorf("invalid cell group dimensions")
	}
	for _, group := range packet.Cells {
		if len(group) > 6*128 {
			return fmt.Errorf("too many cells")
		}
		for _, cell := range group {
			if len(cell) != 2048 {
				return fmt.Errorf("invalid cell width")
			}
		}
	}
	*p = CellsPacket(packet)
	return nil
}

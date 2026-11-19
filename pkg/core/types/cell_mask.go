package types

import gethtypes "github.com/ethereum/go-ethereum/core/types"

// CellMask uses the canonical 128-bit little-endian custody bitmap shared with
// the independent SDK. Wire encoding remains exactly 16 bytes.
type CellMask = gethtypes.CustodyBitmap

var AllCells = gethtypes.CustodyBitmapAll

// BlobCells is an ordered projection of requested indices. Nil cells/proofs
// indicate unavailable data; proofs correspond to the same positions as cells.
type BlobCells struct {
	Cells  [][]byte
	Proofs [][]byte
}

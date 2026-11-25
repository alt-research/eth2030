// Reconstruct the supplied official RPC fixture pre-states using the pinned
// independent Geth SDK, then authenticate each complete native snapshot root.
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	native "github.com/eth2030/eth2030/core/state"
	nt "github.com/eth2030/eth2030/core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"io"
	"math/big"
	"os"
	"path/filepath"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	source := flag.String("source", ".", "unchanged official genesis.json and chain.rlp directory")
	outputArg := flag.String("output", "", "new output directory for authenticated snapshots")
	flag.Parse()
	if *outputArg == "" {
		return fmt.Errorf("new output directory is required")
	}
	base := *source
	raw, err := os.ReadFile(filepath.Join(base, "genesis.json"))
	if err != nil {
		return err
	}
	var genesis core.Genesis
	if err = json.Unmarshal(raw, &genesis); err != nil {
		return err
	}
	db := rawdb.NewMemoryDatabase()
	defer db.Close()
	cfg := core.DefaultConfig().WithArchive(true).WithStateScheme(rawdb.HashScheme)
	cfg.Preimages = true
	chain, err := core.NewBlockChain(db, &genesis, beacon.New(ethash.NewFaker()), cfg)
	if err != nil {
		return err
	}
	defer chain.Stop()
	f, err := os.Open(filepath.Join(base, "chain.rlp"))
	if err != nil {
		return err
	}
	defer f.Close()
	stream := rlp.NewStream(f, 0)
	var blocks types.Blocks
	for {
		var b types.Block
		err := stream.Decode(&b)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		blocks = append(blocks, &b)
	}
	if n, err := chain.InsertChain(blocks); err != nil {
		return fmt.Errorf("SDK import %d: %w", n, err)
	}
	fmt.Printf("SDK imported %d actual fixture blocks\n", len(blocks))
	output := *outputArg
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	for _, number := range []uint64{1, 44, 45} {
		b := chain.GetBlockByNumber(number)
		if b == nil {
			return fmt.Errorf("block %d missing", number)
		}
		st, err := chain.StateAt(b.Root(), new(big.Int).SetUint64(number), b.Time())
		if err != nil {
			return err
		}
		it, err := st.Database().Iteratee(b.Root())
		if err != nil {
			return err
		}
		accounts, err := it.NewAccountIterator(common.Hash{})
		if err != nil {
			return err
		}
		snapshot := native.NewStateSnapshot()
		for accounts.Next() {
			account := accounts.Account()
			encoded, err := rlp.EncodeToBytes(account)
			if err != nil {
				return err
			}
			hash := nt.Hash(accounts.Hash())
			snapshot.Accounts[hash] = encoded
			codeHash := common.BytesToHash(account.CodeHash)
			if codeHash != types.EmptyCodeHash {
				snapshot.Codes[nt.Hash(codeHash)] = rawdb.ReadCode(db, codeHash)
			}
			slots, err := it.NewStorageIterator(accounts.Hash(), common.Hash{})
			if err != nil {
				return err
			}
			for slots.Next() {
				value := slots.Slot()
				encoded, err := rlp.EncodeToBytes(bytes.TrimLeft(value[:], "\x00"))
				if err != nil {
					return err
				}
				if snapshot.Storage[hash] == nil {
					snapshot.Storage[hash] = make(map[nt.Hash][]byte)
				}
				snapshot.Storage[hash][nt.Hash(slots.Hash())] = encoded
			}
			if err := slots.Error(); err != nil {
				return err
			}
			slots.Release()
		}
		if err := accounts.Error(); err != nil {
			return err
		}
		accounts.Release()
		if err := snapshot.Validate(nt.Hash(b.Root())); err != nil {
			return err
		}
		wire := struct {
			Root     string
			Accounts map[string][]byte
			Storage  map[string]map[string][]byte
			Codes    map[string][]byte
		}{Root: b.Root().Hex(), Accounts: make(map[string][]byte), Storage: make(map[string]map[string][]byte), Codes: make(map[string][]byte)}
		for k, v := range snapshot.Accounts {
			wire.Accounts[k.Hex()] = v
		}
		for a, slots := range snapshot.Storage {
			wire.Storage[a.Hex()] = make(map[string][]byte)
			for k, v := range slots {
				wire.Storage[a.Hex()][k.Hex()] = v
			}
		}
		for k, v := range snapshot.Codes {
			wire.Codes[k.Hex()] = v
		}
		data, err := json.Marshal(wire)
		if err != nil {
			return err
		}
		p := filepath.Join(output, fmt.Sprintf("state-%d.json.gz", number))
		f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		z := gzip.NewWriter(f)
		if _, err := z.Write(data); err != nil {
			return err
		}
		if err := z.Close(); err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		fmt.Printf("Verified complete snapshot %d root %s accounts %d\n", number, b.Root(), len(snapshot.Accounts))
	}
	return nil
}

package txpool

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/eth2030/eth2030/core/execution"
	"github.com/eth2030/eth2030/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestFrameCDeploymentStorageRootCacheAndCandidate(t *testing.T) {
	for _, create2 := range []bool{false, true} {
		t.Run(fmt.Sprintf("create2_%t", create2), func(t *testing.T) {
			pool, db, frame := framePoolFixture(t)
			factory := types.Address{19: 0x90}
			verifier := []byte{0x60, 3, 0x60, 0x0a, 0xb0, 0xb3, 0x5f, 0x5f, 0xaa}
			init := append([]byte{0x60, 9, 0x60, 10, 0x5f, 0x39, 0x60, 9, 0x5f, 0xf3}, verifier...)
			code := []byte{0x36, 0x5f, 0x5f, 0x37}
			frame.Sender = types.Address(gethcrypto.CreateAddress([20]byte(factory), 0))
			if create2 {
				code = append(code, 0x5f)
				frame.Sender = types.Address(gethcrypto.CreateAddress2([20]byte(factory), [32]byte{}, gethcrypto.Keccak256(init)))
			}
			code = append(code, 0x36, 0x5f, 0x5f)
			op := byte(0xf0)
			if create2 {
				op = 0xf5
			}
			db.SetCode(factory, append(code, op, 0x50, 0))
			db.AddBalance(frame.Sender, big.NewInt(100_000_000))
			frame.Frames = []types.Frame8141{
				{Mode: types.ModeDefault, Target: &factory, Data: init, Limits: types.FrameGasLimits{Execution: 30000, State: 13770}},
				{Mode: types.ModeVerify, Flags: 3, Limits: types.FrameGasLimits{Execution: 1000}},
			}
			transaction := types.NewTransaction(frame)
			if err := pool.AddRemote(transaction); err != nil {
				t.Fatal("empty-root deployment rejected", err)
			}
			before := pool.lookup.frameValidations[transaction.Hash()]
			dep := execution.FrameDependency{Address: frame.Sender, Kind: execution.FrameStorageRootDependency}
			if _, found := before.Dependencies[dep]; !found || len(pool.lookup.frameDependencies[dep]) != 1 {
				t.Fatal("deployment did not retain its complete pre-state storage-root dependency")
			}
			key := types.Hash(gethcrypto.Keccak256Hash([]byte("unread high storage slot")))
			// Unrelated writes retain the validated prefix without executing it again.
			db.SetState(types.Address{19: 0x99}, key, types.Hash{31: 1})
			pool.Reset(db)
			if pool.lookup.frameValidations[transaction.Hash()] != before {
				t.Fatal("unrelated root change re-executed validation")
			}
			// Candidate state differs without changing canonical pool dependencies.
			candidate := db.Dup()
			candidate.SetState(frame.Sender, key, types.Hash{31: 1})
			header := &types.Header{Time: 100, Number: big.NewInt(1), GasLimit: 30000000, BaseFee: big.NewInt(1)}
			root := db.GetRoot()
			if err := pool.ValidateFrameForBuild(transaction, candidate, header); err == nil {
				t.Fatal("candidate storage collision reused cached acceptance")
			}
			if db.GetRoot() != root || pool.lookup.frameValidations[transaction.Hash()] != before {
				t.Fatal("candidate revalidation changed canonical state or cache")
			}
			// A canonical root-only change invalidates acceptance and releases all indices.
			db.SetState(frame.Sender, key, types.Hash{31: 1})
			pool.Reset(db)
			if pool.Get(transaction.Hash()) != nil || len(pool.lookup.frameDependencies) != 0 || len(pool.lookup.framePayers) != 0 || len(pool.lookup.frameValidations) != 0 {
				t.Fatal("root-only collision retained acceptance or reservations")
			}
			if err := pool.AddRemote(transaction); err == nil || len(pool.lookup.frameDependencies) != 0 {
				t.Fatal("fresh collision admission succeeded or left provisional metadata")
			}
			// A rollback to the actual empty root permits admission again.
			db.SetState(frame.Sender, key, types.Hash{})
			pool.Reset(db)
			if err := pool.AddRemote(transaction); err != nil {
				t.Fatal("empty-root rollback prevented readmission", err)
			}
		})
	}
}

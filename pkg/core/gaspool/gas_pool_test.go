package gaspool

import (
	"errors"
	"testing"
)

func TestGasPool_AddSub(t *testing.T) {
	pool := new(GasPool).AddGas(1000)

	pool.AddGas(500)
	if pool.Gas() != 1500 {
		t.Fatalf("after AddGas(500): got %d, want 1500", pool.Gas())
	}

	if err := pool.SubGas(300); err != nil {
		t.Fatalf("SubGas(300): %v", err)
	}
	if pool.Gas() != 1200 {
		t.Fatalf("after SubGas(300): got %d, want 1200", pool.Gas())
	}
}

func TestGasPool_SubExhausted(t *testing.T) {
	pool := new(GasPool).AddGas(100)

	err := pool.SubGas(200)
	if !errors.Is(err, ErrGasPoolExhausted) {
		t.Fatalf("expected ErrGasPoolExhausted, got %v", err)
	}
	if pool.Gas() != 100 {
		t.Fatalf("pool should be unchanged after failed SubGas: got %d", pool.Gas())
	}
}

func TestGasPool_ZeroGas(t *testing.T) {
	var pool GasPool

	err := pool.SubGas(1)
	if !errors.Is(err, ErrGasPoolExhausted) {
		t.Fatalf("zero pool: expected ErrGasPoolExhausted, got %v", err)
	}
}

func TestFrameIndependentBlockReservations(t *testing.T) {
	pool := new(GasPool).AddGas(100000)
	if err := pool.ReserveFrame(90000, 90000); err != nil {
		t.Fatal("independent dimensions were summed", err)
	}
	before := *pool
	if err := pool.ReserveFrame(1, 10001); err != ErrGasPoolExhausted || *pool != before {
		t.Fatal("failed state reservation partially debited the pool")
	}
	pool.RefundFrame(20000, 80000)
	if pool.Gas() != 30000 || pool.StateGas() != 90000 {
		t.Fatal("dimension refill crossed pools")
	}
	if err := pool.ReserveFrame(30000, 90000); err != nil {
		t.Fatal(err)
	}
	if pool.Gas() != 0 || pool.StateGas() != 0 {
		t.Fatal("exact block limits were not consumed")
	}
}

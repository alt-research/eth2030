package vm

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
)

// These byte strings and outcomes are published execution vectors in EIP-8024,
// rather than fixtures generated from this implementation's decoder.
func TestEIP8024PublishedExecutionVectors(t *testing.T) {
	for _, tc := range []struct {
		code string
		want []int64
		fail bool
	}{
		{"60016000808080808080808080808080808080e680", []int64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, false},
		{"600160008080808080808080808080808080806002e780", []int64{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, false},
		{"600260008080808080600160008080808080808080e8", []int64{1, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0}, false},
		{"600060016002e88e", []int64{1, 0, 2}, false},
		{"600080808080808080808080808080808080808080808080808080808060016002e88f", []int64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}, false},
		{"e75b", nil, true}, {"600456e65b", nil, false},
		{"60008080e88e15", []int64{0, 0, 1}, false}, {"e852", nil, true},
		{"6000808080808080808080808080808080e680", nil, true},
	} {
		t.Run(tc.code, func(t *testing.T) {
			code, err := hex.DecodeString(tc.code)
			if err != nil {
				t.Fatal(err)
			}
			evm := NewEVM(BlockContext{}, TxContext{}, Config{})
			evm.SetJumpTable(NewGlamsterdanJumpTable())
			contract := NewContract([20]byte{}, [20]byte{}, new(big.Int), 100000)
			contract.Code = code
			// Check published final stacks directly, then run the same raw code through the interpreter.
			if tc.want != nil {
				stack := NewStack()
				memory := NewMemory()
				for pc := uint64(0); pc < uint64(len(code)); pc++ {
					op := evm.jumpTable[OpCode(code[pc])]
					if op == nil {
						t.Fatalf("undefined opcode at %d", pc)
					}
					if stack.Len() < op.minStack || stack.Len() > op.maxStack {
						err = ErrStackUnderflow
						break
					}
					if !contract.UseGas(op.constantGas) {
						err = ErrOutOfGas
						break
					}
					_, err = op.execute(&pc, evm, contract, memory, stack)
					if err != nil {
						break
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				if stack.Len() != len(tc.want) {
					t.Fatalf("stack length=%d want=%d", stack.Len(), len(tc.want))
				}
				for i, v := range tc.want {
					if stack.Data()[i].Int64() != v {
						t.Fatalf("stack[%d]=%s want=%d", i, stack.Data()[i], v)
					}
				}
			}
			contract.Gas = 100000
			_, err = evm.Run(contract, nil)
			if (err != nil) != tc.fail {
				t.Fatalf("execution error=%v want failure=%t", err, tc.fail)
			}
		})
	}
}

func TestEIP8024EveryEncodedDepthAndStackBoundary(t *testing.T) {
	for n := 17; n <= 235; n++ {
		for _, opcode := range []OpCode{DUPN, SWAPN} {
			t.Run(fmt.Sprintf("%s/%d", opcode, n), func(t *testing.T) {
				need := n
				if opcode == SWAPN {
					need++
				}
				for _, depth := range []int{need - 1, need} {
					stack := NewStack()
					for i := 1; i <= depth; i++ {
						stack.Push(big.NewInt(int64(i)))
					}
					contract := NewContract([20]byte{}, [20]byte{}, nil, 3)
					contract.Code = []byte{byte(opcode), byte(n + 111)}
					evm := NewEVM(BlockContext{}, TxContext{}, Config{})
					op := NewAmsterdamJumpTable()[opcode]
					var pc uint64
					_, err := op.execute(&pc, evm, contract, NewMemory(), stack)
					if depth < need {
						if err != ErrStackUnderflow {
							t.Fatalf("underflow: %v", err)
						}
						continue
					}
					if err != nil || stack.Peek().Int64() != 1 || pc != 1 {
						t.Fatalf("error=%v top=%s pc=%d", err, stack.Peek(), pc)
					}
					if opcode == SWAPN && stack.Data()[0].Int64() != int64(need) {
						t.Fatal("SWAPN did not preserve former top")
					}
				}
			})
		}
	}
	for n := 1; n <= 14; n++ {
		for m := n + 1; m <= 30-n; m++ {
			t.Run(fmt.Sprintf("exchange/%d/%d", n, m), func(t *testing.T) {
				// Assemble with the inverse encoding published in EIP-8024.
				q, r := n-1, m-1
				if m > 16 {
					q, r = 29-m, n-1
				}
				immediate := byte(16*q+r) ^ 143
				for _, depth := range []int{m, m + 1} {
					stack := NewStack()
					for i := 1; i <= depth; i++ {
						stack.Push(big.NewInt(int64(i)))
					}
					contract := NewContract([20]byte{}, [20]byte{}, nil, 3)
					contract.Code = []byte{byte(EXCHANGE), immediate}
					var pc uint64
					_, err := opExchange(&pc, NewEVM(BlockContext{}, TxContext{}, Config{}), contract, NewMemory(), stack)
					if depth == m {
						if err != ErrStackUnderflow {
							t.Fatalf("underflow: %v", err)
						}
						continue
					}
					top := depth - 1
					if err != nil || stack.Data()[top-n].Int64() != int64(depth-m) || stack.Data()[top-m].Int64() != int64(depth-n) || stack.Peek().Int64() != int64(depth) {
						t.Fatalf("wrong exchange %d/%d immediate=%x error=%v", n, m, immediate, err)
					}
				}
			})
		}
	}
}

func TestEIP8024ReservedImmediatesAndOverflow(t *testing.T) {
	for _, opcode := range []OpCode{DUPN, SWAPN, EXCHANGE} {
		start := 91
		if opcode == EXCHANGE {
			start = 82
		}
		for x := start; x < 128; x++ {
			stack := NewStack()
			for i := 0; i < 1024; i++ {
				stack.Push(new(big.Int))
			}
			contract := NewContract([20]byte{}, [20]byte{}, nil, 3)
			contract.Code = []byte{byte(opcode), byte(x)}
			var pc uint64
			_, err := NewAmsterdamJumpTable()[opcode].execute(&pc, NewEVM(BlockContext{}, TxContext{}, Config{}), contract, NewMemory(), stack)
			if err != ErrInvalidOpCode {
				t.Fatalf("%s immediate=%x error=%v", opcode, x, err)
			}
		}
	}
	evm := NewEVM(BlockContext{}, TxContext{}, Config{})
	evm.SetJumpTable(NewAmsterdamJumpTable())
	contract := NewContract([20]byte{}, [20]byte{}, nil, 100000)
	contract.Code = append(bytes.Repeat([]byte{byte(PUSH0)}, 1024), byte(DUPN), 0x80)
	if _, err := evm.Run(contract, nil); err != ErrStackOverflow {
		t.Fatalf("1024-entry DUPN: %v", err)
	}
}

func TestEIP8024JumpdestCompatibility(t *testing.T) {
	for _, tc := range []struct {
		code   []byte
		target int
		valid  bool
	}{
		{[]byte{0xe6, 0x5b}, 1, true}, {[]byte{0xe7, 0x5b}, 1, true}, {[]byte{0xe8, 0x5b}, 1, true},
		{[]byte{0xe6, 0x60, 0x5b}, 2, false}, {[]byte{0xe6, 0x80, 0x5b}, 2, true},
		{[]byte{0xe7, 0x61, 0, 0, 0x5b}, 4, true},
	} {
		contract := NewContract([20]byte{}, [20]byte{}, nil, 100000)
		contract.Code = tc.code
		if got := contract.validJumpdest(big.NewInt(int64(tc.target))); got != tc.valid {
			t.Fatalf("code=%x target=%d valid=%t want=%t", tc.code, tc.target, got, tc.valid)
		}
	}
}

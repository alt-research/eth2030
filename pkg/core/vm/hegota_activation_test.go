package vm

import "testing"

func TestFrameOpcodesAreIsolatedFromHistoricalForks(t *testing.T) {
	for _, r := range []ForkRules{{IsPrague: true}, {IsGlamsterdan: true}, {IsIPlus: true}, {IsVerkle: true}} {
		table := SelectJumpTable(r)
		for _, op := range []OpCode{APPROVE, TXPARAMLOAD, TXPARAMSIZE, TXPARAMCOPY} {
			if table[op] != nil {
				t.Fatalf("prototype opcode %x leaked into %+v", op, r)
			}
		}
	}
	table := SelectJumpTable(ForkRules{IsGlamsterdan: true, IsLegacyFrames: true})
	for _, op := range []OpCode{APPROVE, TXPARAMLOAD, TXPARAMSIZE, TXPARAMCOPY} {
		if table[op] == nil {
			t.Fatalf("explicit legacy profile lacks %x", op)
		}
	}
}

func TestFrameActivationPreservesOtherForkTables(t *testing.T) {
	for _, r := range []ForkRules{{IsIPlus: true, IsLegacyFrames: true}, {IsVerkle: true, IsLegacyFrames: true}} {
		table := SelectJumpTable(r)
		for _, op := range []OpCode{APPROVE, TXPARAMLOAD, TXPARAMSIZE, TXPARAMCOPY} {
			if table[op] == nil {
				t.Fatalf("active prototype missing %x under %+v", op, r)
			}
		}
		if r.IsIPlus && table[RVCREATE] == nil {
			t.Fatal("Frame activation removed IPlus instructions")
		}
		if r.IsVerkle && table[SLOAD].constantGas != 0 {
			t.Fatal("Frame activation replaced Verkle access pricing")
		}
	}
}

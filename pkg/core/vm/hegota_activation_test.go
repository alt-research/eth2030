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

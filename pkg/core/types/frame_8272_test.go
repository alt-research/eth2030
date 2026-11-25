package types

import (
	"bytes"
	"testing"
)

func TestRecentRoot8272ReferenceVector(t *testing.T) {
	ref := RecentRootReference{Source: RecentRootSource(Address{19: 1}, Hash{}), Slot: 1, Root: Hash{31: 2}}
	if ref.Source != HexToHash("0xb9382d35273c75a50631a3e84d3c75ec9266e2b18c35a627e16cdbf26a18ca85") ||
		ref.EntryHash() != HexToHash("0x0a0d1254c851be5a133b4c9a9e300f5602fc0f43dbe65aa6a66930d4ca0a51b8") ||
		ref.StorageKey() != HexToHash("0x5f027aa1cbe2df279bf6518edd4b44ea5409fd800189ec35224e10ab05e574c3") {
		t.Fatal("EIP reference vector mismatch")
	}
	decoded, err := DecodeRecentRootReferences(ref.AppendTo(nil))
	if err != nil || len(decoded) != 1 || decoded[0] != ref {
		t.Fatal(decoded, err)
	}
	for _, slot := range []uint64{0, 1, 8193, ^uint64(0)} {
		if ref.IsRecent(slot) {
			t.Fatalf("accepted slot %d", slot)
		}
	}
	if !ref.IsRecent(2) || !ref.IsRecent(8192) {
		t.Fatal("usable bounds rejected")
	}
	alias := ref
	alias.Slot += RecentRootLength
	if alias.StorageKey() != ref.StorageKey() || alias.EntryHash() == ref.EntryHash() {
		t.Fatal("ring alias loses entry slot binding")
	}
	frame := Frame8141{Mode: ModeVerify, Target: &RecentRootAddress, Data: ref.AppendTo(nil)}
	if !frame.IsRecentRootVerifier() {
		t.Fatal("canonical frame not classified")
	}
	frame.Flags = 1
	if frame.IsRecentRootVerifier() {
		t.Fatal("noncanonical frame classified")
	}
	if bytes.Equal(recentRootEntryDomain, recentRootStorageDomain) {
		t.Fatal("domains collide")
	}
}

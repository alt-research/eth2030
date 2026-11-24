package fastconfirm

import (
	"testing"
	"time"
)

func TestVotesForAnotherRootCannotConfirmCandidate(t *testing.T) {
	tracker := NewFastConfirmTracker(&FastConfirmConfig{QuorumThreshold: .75, MinAttesters: 1, ConfirmTimeout: time.Minute, MaxTrackedSlots: 4, TotalValidators: 4})
	root, other := fcHash(1), fcHash(2)
	if err := tracker.AddAttestation(1, root, 0); err != nil {
		t.Fatal(err)
	}
	for _, index := range []ValidatorIndex{1, 2} {
		if err := tracker.AddAttestation(1, other, index); err == nil {
			t.Errorf("accepted validator %d for a conflicting root", index)
		}
	}
	if tracker.IsConfirmed(1, root) || tracker.AttestationCount(1) != 1 {
		t.Fatalf("foreign votes changed candidate: confirmed=%v count=%d", tracker.IsConfirmed(1, root), tracker.AttestationCount(1))
	}
	// A rejected foreign-root observation must not consume the candidate vote.
	for _, index := range []ValidatorIndex{1, 2} {
		if err := tracker.AddAttestation(1, root, index); err != nil {
			t.Fatal(err)
		}
	}
	if !tracker.IsConfirmed(1, root) || tracker.IsConfirmed(1, other) {
		t.Fatal("candidate did not reach its own quorum")
	}
}

func TestUnknownValidatorCannotSupplyQuorum(t *testing.T) {
	tracker := NewFastConfirmTracker(&FastConfirmConfig{QuorumThreshold: .75, MinAttesters: 1, ConfirmTimeout: time.Minute, MaxTrackedSlots: 4, TotalValidators: 4})
	for _, index := range []ValidatorIndex{4, 5, ^ValidatorIndex(0)} {
		if err := tracker.AddAttestation(1, fcHash(1), index); err == nil {
			t.Errorf("accepted out-of-range validator %d", index)
		}
	}
	if tracker.IsConfirmed(1, fcHash(1)) || tracker.TrackedSlots() != 0 {
		t.Fatal("unknown validators created confirmation state")
	}
}

func TestTrackerOwnsConfiguration(t *testing.T) {
	cfg := &FastConfirmConfig{QuorumThreshold: .75, MinAttesters: 1, ConfirmTimeout: time.Minute, MaxTrackedSlots: 4, TotalValidators: 4}
	tracker := NewFastConfirmTracker(cfg)
	cfg.QuorumThreshold = .1
	cfg.TotalValidators = 1
	if err := tracker.AddAttestation(1, fcHash(1), 0); err != nil {
		t.Fatal(err)
	}
	if tracker.IsConfirmed(1, fcHash(1)) || tracker.Config().TotalValidators != 4 {
		t.Fatal("caller mutation changed the tracker quorum")
	}
}

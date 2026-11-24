package canonicalproof

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCancellationAndCloseJoinTheRealProcessGroup(t *testing.T) {
	for _, stop := range []string{"cancel", "close", "parent"} {
		t.Run(stop, func(t *testing.T) {
			dir := t.TempDir()
			binary := filepath.Join(dir, "blocked-prover")
			// This process deliberately does not produce a proof. It tests process
			// ownership and cleanup, not positive cryptographic acceptance.
			script := "#!/bin/sh\nsleep 300 &\nprintf '%s' \"$!\" > \"$(dirname \"$0\")/child.pid\"\nwait\n"
			if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			parent, cancel := context.WithCancel(context.Background())
			defer cancel()
			service := NewService(parent, Prover{Binary: binary})
			defer service.Close()
			job, err := service.Submit([]byte{0x73}, nil, DefaultConfig())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.Submit([]byte{0x73}, nil, DefaultConfig()); !errors.Is(err, ErrBusy) {
				t.Fatal("unbounded proof concurrency", err)
			}
			deadline := time.Now().Add(3 * time.Second)
			var pid int
			for time.Now().Before(deadline) {
				data, _ := os.ReadFile(filepath.Join(dir, "child.pid"))
				pid, _ = strconv.Atoi(string(data))
				if pid > 0 {
					break
				}
				time.Sleep(5 * time.Millisecond)
			}
			if pid == 0 {
				t.Fatal("prover child did not start")
			}
			switch stop {
			case "cancel":
				if err := service.Cancel(job.ID); err != nil {
					t.Fatal(err)
				}
			case "parent":
				cancel()
			case "close":
				service.Close()
			}
			deadline = time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				state, _ := service.Job(job.ID)
				data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "status"))
				dead := os.IsNotExist(err) || strings.Contains(string(data), "State:\tZ")
				if state.Status == "cancelled" && dead {
					return
				}
				time.Sleep(5 * time.Millisecond)
			}
			t.Fatal("cancel/shutdown left a proving subprocess running")
		})
	}
}

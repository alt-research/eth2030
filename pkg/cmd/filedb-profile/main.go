// filedb-profile measures an explicitly owned synthetic FileDB fixture.
// Fixture preparation is separate from database open/recovery measurements.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/eth2030/eth2030/core/rawdb"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"
)

type fixture struct {
	Format         int    `json:"format"`
	Keys           int    `json:"keys"`
	ValueBytes     int    `json:"value_bytes"`
	WALGenerations int    `json:"wal_generations"`
	SHA256         string `json:"expected_sha256"`
}

func key(i int) []byte { return []byte(fmt.Sprintf("profile/%09d", i)) }
func value(i, generation, size int) []byte {
	b := bytes.Repeat([]byte{byte(i % 251)}, size)
	binary.LittleEndian.PutUint64(b, uint64(i))
	binary.LittleEndian.PutUint64(b[8:], uint64(generation))
	return b
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	mode := flag.String("mode", "open", "prepare or open")
	dir := flag.String("dir", "", "new owned fixture directory")
	keys := flag.Int("keys", 1000, "fixture key count")
	size := flag.Int("value-bytes", 4096, "bytes per fixture value")
	generations := flag.Int("wal-generations", 0, "committed overwrite generations in the fixture WAL")
	flag.Parse()
	if *dir == "" {
		return fmt.Errorf("--dir is required")
	}
	marker := filepath.Join(*dir, "PROFILE_FIXTURE.json")
	if *mode == "prepare" {
		if *keys < 1 || *size < 16 || *generations < 0 {
			return fmt.Errorf("invalid fixture parameters")
		}
		// Never overwrite a pre-existing directory or point fixture creation at a node.
		if err := os.Mkdir(*dir, 0700); err != nil {
			return err
		}
		data := filepath.Join(*dir, "data")
		if err := os.Mkdir(data, 0700); err != nil {
			return err
		}
		h := sha256.New()
		for i := 0; i < *keys; i++ {
			k := key(i)
			v := value(i, *generations, *size)
			if err := os.WriteFile(filepath.Join(data, hex.EncodeToString(k)), v, 0600); err != nil {
				return err
			}
			h.Write(k)
			h.Write(v)
		}
		if *generations > 0 {
			f, err := os.Create(filepath.Join(*dir, "wal"))
			if err != nil {
				return err
			}
			w := bufio.NewWriterSize(f, 1<<20)
			var n [4]byte
			for generation := 1; generation <= *generations; generation++ {
				for i := 0; i < *keys; i++ {
					k := key(i)
					v := value(i, generation, *size)
					if err := w.WriteByte(1); err != nil {
						return err
					}
					binary.BigEndian.PutUint32(n[:], uint32(len(k)))
					w.Write(n[:])
					w.Write(k)
					binary.BigEndian.PutUint32(n[:], uint32(len(v)))
					w.Write(n[:])
					w.Write(v)
				}
				if err := w.WriteByte(3); err != nil {
					return err
				}
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if err := f.Sync(); err != nil {
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
		meta := fixture{1, *keys, *size, *generations, hex.EncodeToString(h.Sum(nil))}
		body, _ := json.MarshalIndent(meta, "", "  ")
		if err := os.WriteFile(marker, body, 0600); err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(meta)
	}
	if *mode != "open" {
		return fmt.Errorf("unknown mode")
	}
	body, err := os.ReadFile(marker)
	if err != nil {
		return fmt.Errorf("not an owned fixture: %w", err)
	}
	var meta fixture
	if err := json.Unmarshal(body, &meta); err != nil {
		return err
	}
	if meta.Format != 1 || meta.Keys < 1 || meta.ValueBytes < 16 {
		return fmt.Errorf("invalid fixture marker")
	}
	// Measurement-only GC budget; production FileDB configuration is unchanged.
	debug.SetMemoryLimit(1 << 30)
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	walBytes := int64(0)
	if info, err := os.Stat(filepath.Join(*dir, "wal")); err == nil {
		walBytes = info.Size()
	}
	start := time.Now()
	db, err := rawdb.NewFileDB(*dir)
	if err != nil {
		return err
	}
	openTime := time.Since(start)
	defer db.Close()
	runtime.GC()
	runtime.ReadMemStats(&after)
	if db.Len() != meta.Keys {
		return fmt.Errorf("key count=%d want%d", db.Len(), meta.Keys)
	}
	h := sha256.New()
	readStart := time.Now()
	for i := 0; i < meta.Keys; i++ {
		k := key(i)
		actual, err := db.Get(k)
		if err != nil {
			return err
		}
		if !bytes.Equal(actual, value(i, meta.WALGenerations, meta.ValueBytes)) {
			return fmt.Errorf("key%d differs", i)
		}
		h.Write(k)
		h.Write(actual)
	}
	if hex.EncodeToString(h.Sum(nil)) != meta.SHA256 {
		return fmt.Errorf("full fixture digest differs")
	}
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		return err
	}
	result := map[string]any{"passed": true, "scope": "Synthetic KV storage/recovery profile only; fixture preparation excluded, no blockchain conformance or multi-day claim.", "fixture": meta, "open_seconds": openTime.Seconds(), "read_all_seconds": time.Since(readStart).Seconds(), "heap_alloc_before": before.HeapAlloc, "heap_alloc_after_open": after.HeapAlloc, "wal_input_bytes": walBytes, "max_rss_kib": usage.Maxrss, "reopening_budget_seconds": 300, "rss_budget_bytes": 1 << 30, "budget_passed": openTime < 300*time.Second && usage.Maxrss*1024 < 1<<30}
	return json.NewEncoder(os.Stdout).Encode(result)
}

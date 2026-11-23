package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/eth2030/eth2030/core/rawdb"
	"os"
	"sync"
	"syscall"
	"time"
)

func main() {
	dir := os.Args[1]
	if _, e := os.Stat(dir); !os.IsNotExist(e) {
		panic("exclusive fresh fixture required")
	}
	d, e := rawdb.NewFileDB(dir)
	must(e)
	b := d.NewBatch()
	n := 12056
	for i := range n {
		k := make([]byte, 9)
		k[0] = 'c'
		binary.BigEndian.PutUint64(k[1:], uint64(94503+i))
		v := sha256.Sum256(k)
		must(b.Put(k, v[:]))
	}
	var mu sync.Mutex
	published := false
	start := time.Now()
	must(rawdb.WriteBatchAndPublish(b, &mu, func() { published = true }))
	duration := time.Since(start).Seconds()
	if !published {
		panic("not published")
	}
	must(d.Close())
	d, e = rawdb.NewFileDB(dir)
	must(e)
	defer d.Close()
	for i := range n {
		k := make([]byte, 9)
		k[0] = 'c'
		binary.BigEndian.PutUint64(k[1:], uint64(94503+i))
		want := sha256.Sum256(k)
		got, e := d.Get(k)
		must(e)
		if !bytes.Equal(got, want[:]) {
			panic("reopened value differs")
		}
	}
	var st syscall.Stat_t
	must(syscall.Stat(dir, &st))
	h, e := os.ReadFile(os.Args[0])
	must(e)
	v := sha256.Sum256(h)
	bts, e := json.Marshal(map[string]any{"passed": true, "entries": n, "value_bytes": 32, "write_and_publish_seconds": duration, "reopen_all_values_verified": true, "device": st.Dev, "binary_sha256": hex.EncodeToString(v[:]), "scope": "Fresh captured canonical batch shape, not actual chain replay, runtime duration or original04 attribution"})
	must(e)
	fmt.Println(string(bts))
}
func must(e error) {
	if e != nil {
		panic(e)
	}
}

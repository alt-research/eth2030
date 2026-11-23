package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eth2030/eth2030/core/rawdb"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type item struct {
	key  []byte
	name string
}
type receipt struct {
	source, target []byte
	bytes          int64
	reused         bool
	err            error
}

func must(e error) {
	if e != nil {
		panic(e)
	}
}

var buffers = sync.Pool{New: func() any { return make([]byte, 32768) }}

func hash(f *os.File) ([]byte, int64, error) {
	h := sha256.New()
	b := buffers.Get().([]byte)
	defer buffers.Put(b)
	n, e := io.CopyBuffer(h, struct{ io.Reader }{f}, b)
	return h.Sum(nil), n, e
}
func syncDirectory(p string) error {
	f, e := os.Open(p)
	if e != nil {
		return e
	}
	return errors.Join(f.Sync(), f.Close())
}
func main() {
	if len(os.Args) != 5 {
		panic("source-DB new-destination-DB owned-reuse-data-or-dash receipt")
	}
	source, dest, reuse, out := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	start := time.Now()
	report := map[string]any{"passed": false, "scope": "Read-only closed legacy source; complete per-value source/destination digest comparison; only verified owned reusable copies move. No source hardlinks or source writes.", "source": source, "destination": dest}
	defer func() {
		if p := recover(); p != nil {
			report["error"] = fmt.Sprint(p)
		}
		report["elapsed_seconds"] = time.Since(start).Seconds()
		b, e := json.MarshalIndent(report, "", "  ")
		if e != nil {
			panic(e)
		}
		must(os.WriteFile(out, append(b, '\n'), 0600))
		fmt.Println(string(b))
	}()
	lock, e := os.Open(filepath.Join(source, "LOCK"))
	must(e)
	defer lock.Close()
	must(syscall.Flock(int(lock.Fd()), syscall.LOCK_SH|syscall.LOCK_NB))
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	walInfo, e := os.Lstat(filepath.Join(source, "wal"))
	must(e)
	if !walInfo.Mode().IsRegular() {
		panic("only legacy regular-journal inputs are supported")
	}
	srcData := filepath.Join(source, "data")
	if reuse != "-" {
		a, e := os.Stat(srcData)
		must(e)
		b, e := os.Stat(reuse)
		must(e)
		if os.SameFile(a, b) {
			panic("reuse directory aliases source")
		}
	}
	if _, e := os.Stat(dest); !os.IsNotExist(e) {
		panic("exclusive fresh destination required")
	}
	db, e := rawdb.NewFileDB(dest)
	must(e)
	must(db.Close())
	dir, e := os.Open(srcData)
	must(e)
	defer dir.Close()
	sourceAll, targetAll := sha256.New(), sha256.New()
	var count, total, reused int64
	process := func(it item) (r receipt) {
		src, e := os.Open(filepath.Join(srcData, it.name))
		if e != nil {
			r.err = e
			return
		}
		defer src.Close()
		sourceInfo, e := src.Stat()
		if e != nil || !sourceInfo.Mode().IsRegular() {
			r.err = fmt.Errorf("non-regular source %q: %v", it.name, e)
			return
		}
		digest, size, e := hash(src)
		if e != nil {
			r.err = e
			return
		}
		target := rawdb.FileDBValuePath(dest, it.key)
		if e = os.MkdirAll(filepath.Dir(target), 0755); e != nil {
			r.err = e
			return
		}
		if reuse != "-" {
			path := filepath.Join(reuse, it.name)
			f, err := os.OpenFile(path, os.O_RDWR, 0)
			if err == nil {
				info, err := f.Stat()
				if err != nil {
					f.Close()
					r.err = err
					return
				}
				if os.SameFile(sourceInfo, info) {
					f.Close()
					r.err = errors.New("source-linked reuse file rejected")
					return
				}
				d, n, err := hash(f)
				if err == nil && n == size && bytes.Equal(d, digest) {
					err = errors.Join(f.Sync(), f.Close())
					if err == nil {
						err = os.Rename(path, target)
					}
					if err != nil {
						r.err = err
						return
					}
					r.reused = true
				} else {
					f.Close()
				}
			} else if !os.IsNotExist(err) {
				r.err = err
				return
			}
		}
		if !r.reused {
			if _, e = src.Seek(0, 0); e != nil {
				r.err = e
				return
			}
			dst, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
			if e != nil {
				r.err = e
				return
			}
			n, e := io.Copy(dst, src)
			closeErr := errors.Join(dst.Sync(), dst.Close())
			if e == nil {
				e = closeErr
			}
			if e != nil || n != size {
				r.err = fmt.Errorf("copy count%d/%d: %v", n, size, e)
				return
			}
		}
		check, e := os.Open(target)
		if e != nil {
			r.err = e
			return
		}
		d, n, e := hash(check)
		check.Close()
		if e != nil || n != size || !bytes.Equal(d, digest) {
			r.err = fmt.Errorf("destination differs for %x: %v", it.key, e)
			return
		}
		record := binary.BigEndian.AppendUint64(nil, uint64(len(it.key)))
		record = append(record, it.key...)
		record = binary.BigEndian.AppendUint64(record, uint64(size))
		r.source = append(append([]byte{}, record...), digest...)
		r.target = append(append([]byte{}, record...), d...)
		r.bytes = size
		return
	}
	for {
		entries, readErr := dir.ReadDir(128)
		items := make([]item, 0, len(entries))
		for _, entry := range entries {
			if !entry.Type().IsRegular() {
				panic("unexpected source entry " + entry.Name())
			}
			key, e := hex.DecodeString(entry.Name())
			must(e)
			if len(key) == 0 || hex.EncodeToString(key) != entry.Name() {
				panic("noncanonical legacy filename")
			}
			items = append(items, item{key, entry.Name()})
		}
		results := make([]receipt, len(items))
		jobs := make(chan int)
		var wait sync.WaitGroup
		for range 32 {
			wait.Add(1)
			go func() {
				defer wait.Done()
				for i := range jobs {
					results[i] = process(items[i])
				}
			}()
		}
		for i := range items {
			jobs <- i
		}
		close(jobs)
		wait.Wait()
		for _, r := range results {
			must(r.err)
			sourceAll.Write(r.source)
			targetAll.Write(r.target)
			count++
			total += r.bytes
			if r.reused {
				reused++
			}
		}
		if count > 0 && count%65536 == 0 {
			fmt.Fprintln(os.Stderr, "verified", count, "values")
		}
		if readErr == io.EOF {
			break
		}
		must(readErr)
	}
	// Preserve the exact closed source journal; the new reader handles its
	// committed/torn prefix after all physical values have been independently copied.
	srcWal, e := os.Open(filepath.Join(source, "wal"))
	must(e)
	defer srcWal.Close()
	tmp := rawdb.FileDBJournalPath(dest) + ".copy.tmp"
	dstWal, e := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	must(e)
	_, e = io.Copy(dstWal, srcWal)
	must(e)
	must(errors.Join(dstWal.Sync(), dstWal.Close()))
	must(os.Rename(tmp, rawdb.FileDBJournalPath(dest)))
	must(syncDirectory(filepath.Join(dest, "data")))
	buckets, e := os.ReadDir(filepath.Join(dest, "data-v2"))
	must(e)
	for _, b := range buckets {
		must(syncDirectory(filepath.Join(dest, "data-v2", b.Name())))
	}
	must(syncDirectory(filepath.Join(dest, "data-v2")))
	must(syncDirectory(dest))
	if !bytes.Equal(sourceAll.Sum(nil), targetAll.Sum(nil)) {
		panic("ordered aggregate differs")
	}
	report["passed"] = true
	report["keys"] = count
	report["value_bytes"] = total
	report["verified_reused_values"] = reused
	report["copied_values"] = count - reused
	report["source_order_sha256"] = hex.EncodeToString(sourceAll.Sum(nil))
	report["destination_same_order_sha256"] = hex.EncodeToString(targetAll.Sum(nil))
	report["order_scope"] = "Source directory enumeration order; each key/value independently checked before hashing; not a canonical cross-filesystem digest"
	report["buckets"] = len(buckets)
}

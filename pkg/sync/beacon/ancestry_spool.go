package beacon

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/eth2030/eth2030/core/types"
)

const ancestryRecordBytes = 72
const retainedAncestryPaths = 2
const retainedAncestrySegments = 32

type ancestrySegment struct {
	file         *os.File
	start, count uint64
}
type ancestryPath struct {
	segments          []ancestrySegment
	count, headNumber uint64
	headHash, parent  types.Hash
	writable          *os.File
}

type ancestrySpool struct {
	paths []*ancestryPath
	files map[*os.File]struct{}
	dir   string
}

func (s *ancestrySpool) append(p *ancestryPath, h downloadedHeader) error {
	if p.count > 0 && (p.parent != h.hash || p.count > p.headNumber || h.number != p.headNumber-p.count) {
		return fmt.Errorf("beacon execution sync: disconnected staged ancestry")
	}
	if p.writable == nil {
		f, err := os.CreateTemp(s.dir, "eth2030-ancestry-*")
		if err != nil {
			return fmt.Errorf("beacon execution sync: create ancestry staging: %w", err)
		}
		// Staging is disposable, never validation authority. Unlink while open
		// so a process crash also releases it without orphaning a large file.
		if err := os.Remove(f.Name()); err != nil {
			f.Close()
			return err
		}
		if s.files == nil {
			s.files = make(map[*os.File]struct{})
		}
		s.files[f] = struct{}{}
		p.writable = f
		p.segments = append(p.segments, ancestrySegment{file: f})
	}
	var data [ancestryRecordBytes]byte
	copy(data[:32], h.hash[:])
	copy(data[32:64], h.parent[:])
	binary.BigEndian.PutUint64(data[64:], h.number)
	if _, err := p.writable.Write(data[:]); err != nil {
		return fmt.Errorf("beacon execution sync: write ancestry staging: %w", err)
	}
	if p.count == 0 {
		p.headNumber, p.headHash = h.number, h.hash
	}
	p.count++
	p.parent = h.parent
	p.segments[len(p.segments)-1].count++
	return nil
}

func (p *ancestryPath) at(index uint64) (downloadedHeader, error) {
	if index >= p.count || index > p.headNumber {
		return downloadedHeader{}, fmt.Errorf("beacon execution sync: invalid staged ancestry offset")
	}
	expected := p.headNumber - index
	for _, seg := range p.segments {
		if index >= seg.count {
			index -= seg.count
			continue
		}
		var data [ancestryRecordBytes]byte
		if seg.start+index > uint64(^uint64(0)>>1)/ancestryRecordBytes {
			return downloadedHeader{}, fmt.Errorf("beacon execution sync: staged ancestry offset overflow")
		}
		if _, err := seg.file.ReadAt(data[:], int64((seg.start+index)*ancestryRecordBytes)); err != nil {
			return downloadedHeader{}, fmt.Errorf("beacon execution sync: read ancestry staging: %w", err)
		}
		h := downloadedHeader{hash: types.BytesToHash(data[:32]), parent: types.BytesToHash(data[32:64]), number: binary.BigEndian.Uint64(data[64:])}
		if h.number != expected || h.hash == (types.Hash{}) {
			return downloadedHeader{}, fmt.Errorf("beacon execution sync: corrupt staged ancestry")
		}
		return h, nil
	}
	return downloadedHeader{}, fmt.Errorf("beacon execution sync: incomplete ancestry staging")
}

func (p *ancestryPath) suffix(index uint64) (*ancestryPath, error) {
	h, err := p.at(index)
	if err != nil {
		return nil, err
	}
	out := &ancestryPath{count: p.count - index, headNumber: h.number, headHash: h.hash, parent: p.parent}
	for _, seg := range p.segments {
		if index >= seg.count {
			index -= seg.count
			continue
		}
		seg.start += index
		seg.count -= index
		index = 0
		out.segments = append(out.segments, seg)
	}
	return out, nil
}

func (s *ancestrySpool) find(hash types.Hash, number uint64, known bool) (*ancestryPath, error) {
	for _, p := range s.paths {
		if p.count == 0 {
			continue
		}
		if !known {
			if p.headHash != hash {
				continue
			}

			head, err := p.at(0)
			if err != nil {
				return nil, err
			}
			if head.hash != hash {
				return nil, fmt.Errorf("beacon execution sync: corrupt cached head")
			}
			return p.suffix(0)
		}
		if number > p.headNumber || p.headNumber-number >= p.count {
			continue
		}
		index := p.headNumber - number
		h, err := p.at(index)
		if err != nil {
			return nil, err
		}
		if h.hash == hash {
			return p.suffix(index)
		}
	}
	return nil, nil
}

func (p *ancestryPath) join(suffix *ancestryPath) error {
	if p.count != 0 && (p.parent != suffix.headHash || p.count > p.headNumber || p.headNumber-p.count != suffix.headNumber) {
		return fmt.Errorf("beacon execution sync: disconnected cached ancestry")
	}
	if p.count == 0 {
		p.headNumber, p.headHash = suffix.headNumber, suffix.headHash
	}
	p.segments = append(p.segments, suffix.segments...)
	p.count += suffix.count
	p.parent = suffix.parent
	p.writable = nil
	return nil
}

func (p *ancestryPath) pop(h downloadedHeader) {
	last := len(p.segments) - 1
	p.segments[last].count--
	if p.segments[last].count == 0 {
		p.segments = p.segments[:last]
	}
	p.count--
	p.parent = h.hash
	p.writable = nil
}

func (s *ancestrySpool) retain(p *ancestryPath) error {
	// Bound RAM even under indefinite head replacement. Evict only the newest
	// prefix of cached metadata; keep the deep walk frontier. A later attempt
	// re-fetches that short prefix and joins the retained hash-linked suffix.
	for len(p.segments) > retainedAncestrySegments {
		first := p.segments[0]
		p.count -= first.count
		p.headNumber -= first.count
		p.segments = p.segments[1:]
		h, err := p.at(0)
		if err != nil {
			return err
		}
		p.headNumber, p.headHash = h.number, h.hash
	}
	p.writable = nil
	paths := []*ancestryPath{p}
	for _, old := range s.paths {
		if old.headHash != p.headHash && len(paths) < retainedAncestryPaths {
			paths = append(paths, old)
		}
	}
	s.paths = paths
	s.collect()
	return nil
}

func (s *ancestrySpool) collect() {
	live := make(map[*os.File]bool)
	for _, p := range s.paths {
		for _, seg := range p.segments {
			live[seg.file] = true
		}
	}
	for f := range s.files {
		if !live[f] {
			f.Close()
			delete(s.files, f)
		}
	}
}
func (s *ancestrySpool) clear() {
	s.paths = nil
	s.collect()
}

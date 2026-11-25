package beacon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eth2030/eth2030/core/types"
	"github.com/ethereum/go-ethereum/common/lru"
)

// ExecutionTarget is nominated by the CL through Engine forkchoiceUpdated.
// Peer announcements are not an authority for choosing this target (EIP-3675).
type ExecutionTarget struct{ Head, Safe, Finalized types.Hash }

const maxExecutionAncestry = 65536
const executionBodyCacheBytes = 64 << 20

type downloadedHeader struct {
	hash, parent types.Hash
	number       uint64
}

type ExecutionChain interface {
	GetBlock(types.Hash) *types.Block
	InsertBlock(*types.Block) error
}

type ExecutionFetcher interface {
	FetchBlock(context.Context, types.Hash) (*types.Block, error)
}

type executionProgress struct {
	active         bool
	start, highest uint64
}

// ExecutionSyncer owns one replaceable CL target and one joined worker. Failed
// downloads remain pending and retry as peers become available.
type ExecutionSyncer struct {
	progress                  atomic.Pointer[executionProgress]
	mu                        sync.Mutex
	chain                     ExecutionChain
	fetcher                   ExecutionFetcher
	apply                     func(ExecutionTarget) error
	pivotSync                 func(context.Context, *types.Block) (bool, error)
	target                    ExecutionTarget
	lastErr                   error
	running, complete         bool
	wake                      chan struct{}
	cancel                    context.CancelFunc
	requestCancel             context.CancelFunc
	pivotActive               bool
	startHeight, targetHeight uint64
	done                      chan struct{}
	// Downloaded data is not an execution-validation or forkchoice authority.
	downloadHeaders   *lru.Cache[types.Hash, downloadedHeader]
	downloadBodies    *lru.SizeConstrainedCache[types.Hash, []byte]
	downloadBodyLimit uint64
	ancestry          ancestrySpool
}

func NewExecutionSyncer(chain ExecutionChain, fetcher ExecutionFetcher, apply func(ExecutionTarget) error, stagingDirectory ...string) *ExecutionSyncer {
	dir := ""
	if len(stagingDirectory) > 0 {
		dir = stagingDirectory[0]
	}
	return &ExecutionSyncer{
		chain: chain, fetcher: fetcher, apply: apply, wake: make(chan struct{}, 1),
		ancestry:          ancestrySpool{dir: dir},
		downloadHeaders:   lru.NewCache[types.Hash, downloadedHeader](maxExecutionAncestry),
		downloadBodyLimit: executionBodyCacheBytes,
		downloadBodies:    lru.NewSizeConstrainedCacheWithKeySize[types.Hash, []byte](executionBodyCacheBytes, func(types.Hash) uint64 { return 32 }),
	}
}

// SetPivotSync installs the snap bootstrap path before Start. The pivot is
// restricted to the finalized hash nominated by the CL.
func (s *ExecutionSyncer) SetPivotSync(fn func(context.Context, *types.Block) (bool, error)) {
	s.pivotSync = fn
}

func (s *ExecutionSyncer) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}
	if s.done != nil {
		select {
		case <-s.done:
		default:
			return
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel, s.done, s.running = cancel, make(chan struct{}), true
	go s.loop(ctx, s.done)
}

func (s *ExecutionSyncer) Stop() {
	s.mu.Lock()
	if s.done == nil {
		s.ancestry.clear()
		s.mu.Unlock()
		return
	}
	cancel, done := s.cancel, s.done
	s.running = false
	cancel()
	s.mu.Unlock()
	<-done
}

func (s *ExecutionSyncer) Request(target ExecutionTarget, startingHeight ...uint64) {
	if target.Head == (types.Hash{}) {
		return
	}
	s.mu.Lock()
	if (s.target.Head == (types.Hash{}) || s.complete) && len(startingHeight) > 0 {
		s.startHeight = startingHeight[0]
	}
	if target != s.target {
		s.targetHeight = 0
		// Finish an authenticated finalized snapshot even while the CL head
		// advances. Restarting it for every slot prevents bootstrap progress.
		if s.requestCancel != nil && !s.pivotActive {
			s.requestCancel()
		}
		s.target, s.complete, s.lastErr = target, false, nil
	}
	s.publishProgressLocked()
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// Progress reports the last coherently published target without waiting for
// application callbacks that may be blocked on durable storage. It is telemetry,
// not an execution-validation or canonical-selection authority.
func (s *ExecutionSyncer) Progress() (bool, uint64, uint64) {
	if value := s.progress.Load(); value != nil {
		return value.active, value.start, value.highest
	}
	return false, 0, 0
}
func (s *ExecutionSyncer) publishProgressLocked() {
	s.progress.Store(&executionProgress{active: s.target.Head != (types.Hash{}) && !s.complete, start: s.startHeight, highest: s.targetHeight})
}

func (s *ExecutionSyncer) Status() (ExecutionTarget, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.target, s.complete, s.lastErr
}

// ApplyKnown serializes a synchronous FCU with worker publication. Failed FCUs
// leave the last accepted target unchanged and cannot populate a VALID cache.
func (s *ExecutionSyncer) ApplyKnown(target ExecutionTarget) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.apply != nil {
		if err := s.apply(target); err != nil {
			return err
		}
	}
	if s.requestCancel != nil {
		s.requestCancel()
	}
	s.target, s.complete, s.lastErr = target, true, nil
	s.publishProgressLocked()
	return nil
}

func (s *ExecutionSyncer) AdoptFinalized(ctx context.Context, pivot *types.Block, adopt func() error) error {
	if pivot == nil || pivot.Header().Number == nil || !pivot.Header().Number.IsUint64() {
		return fmt.Errorf("beacon execution sync: invalid pivot")
	}
	hash := pivot.Hash()
	for {
		s.mu.Lock()
		finalized := s.target.Finalized
		s.mu.Unlock()
		if err := ctx.Err(); err != nil {
			return err
		}
		// A snapshot may take longer than a finality interval. An older CL
		// finalized pivot is still usable only if it is an ancestor of the
		// latest finalized block. Verify the hash-linked path before adopting.
		if finalized != hash {
			if err := s.verifyFinalizedAncestor(ctx, finalized, pivot); err != nil {
				return err
			}
		}
		s.mu.Lock()
		if err := ctx.Err(); err != nil {
			s.mu.Unlock()
			return err
		}
		if s.target.Finalized != finalized {
			s.mu.Unlock()
			continue
		}
		err := adopt()
		s.mu.Unlock()
		return err
	}
}

func (s *ExecutionSyncer) verifyFinalizedAncestor(ctx context.Context, descendant types.Hash, pivot *types.Block) error {
	ancestor := pivot.Hash()
	if descendant == (types.Hash{}) || ancestor == (types.Hash{}) {
		return context.Canceled
	}
	var childNumber uint64
	for count := 0; descendant != ancestor; count++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if count >= 65536 {
			return fmt.Errorf("beacon execution sync: finalized ancestry exceeds limit")
		}
		block := s.chain.GetBlock(descendant)
		if block == nil {
			var err error
			block, err = s.fetcher.FetchBlock(ctx, descendant)
			if err != nil {
				return err
			}
		}
		if block == nil || block.Header().Number == nil || !block.Header().Number.IsUint64() || block.Hash() != descendant || block.NumberU64() == 0 || (count > 0 && block.NumberU64()+1 != childNumber) {
			return fmt.Errorf("beacon execution sync: invalid finalized ancestry at %s", descendant)
		}
		childNumber = block.NumberU64()
		descendant = block.ParentHash()
	}
	if childNumber == 0 || childNumber-1 != pivot.NumberU64() {
		return fmt.Errorf("beacon execution sync: non-contiguous finalized pivot")
	}
	return nil
}

func (s *ExecutionSyncer) validated(hash types.Hash) bool {
	if chain, ok := s.chain.(interface{ HasValidatedBlock(types.Hash) bool }); ok {
		return chain.HasValidatedBlock(hash)
	}
	return s.chain.GetBlock(hash) != nil
}

func (s *ExecutionSyncer) loop(ctx context.Context, done chan struct{}) {
	defer close(done)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.wake:
		case <-ticker.C:
		}
		s.mu.Lock()
		target := s.target
		if target.Head == (types.Hash{}) || s.complete {
			s.mu.Unlock()
			continue
		}
		requestCtx, cancel := context.WithCancel(ctx)
		s.requestCancel = cancel
		s.mu.Unlock()
		err := s.synchronize(requestCtx, target)
		cancel()
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.Warn("execution sync failed", "head", target.Head.Hex(), "finalized", target.Finalized.Hex(), "err", err)
		}
		s.mu.Lock()
		if s.target == target {
			s.lastErr, s.complete = err, err == nil
			s.publishProgressLocked()
		}
		s.requestCancel = nil
		s.mu.Unlock()
	}
}

// downloadedBlock reuses completed content-addressed transfers across target
// replacement. A block still has to pass the chain's full execution validation.
func (s *ExecutionSyncer) downloadedBlock(ctx context.Context, hash types.Hash) (*types.Block, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	block := s.chain.GetBlock(hash)
	if block == nil {
		if encoded, ok := s.downloadBodies.Get(hash); ok {
			var err error
			block, err = types.DecodeBlockRLP(encoded)
			if err != nil {
				return nil, fmt.Errorf("beacon execution sync: staged block: %w", err)
			}
		}
	}
	if block == nil {
		var err error
		block, err = s.fetcher.FetchBlock(ctx, hash)
		if err != nil {
			return nil, err
		}
	}
	if block == nil || block.Header().Number == nil || !block.Header().Number.IsUint64() || block.Hash() != hash || block.NumberU64() == 0 {
		return nil, fmt.Errorf("beacon execution sync: invalid ancestry at %s", hash)
	}
	encoded, err := block.EncodeRLP()
	if err != nil {
		return nil, err
	}
	// The upstream byte LRU deliberately permits one oversized value, so enforce
	// the strict operation budget before adding. Oversized bodies may be streamed.
	if uint64(len(encoded))+32 <= s.downloadBodyLimit {
		s.downloadBodies.Add(hash, encoded)
	}
	s.downloadHeaders.Add(hash, downloadedHeader{hash: hash, parent: block.ParentHash(), number: block.NumberU64()})
	return block, nil
}

func (s *ExecutionSyncer) downloadedHeader(ctx context.Context, hash types.Hash) (downloadedHeader, error) {
	if header, ok := s.downloadHeaders.Get(hash); ok {
		return header, nil
	}
	block, err := s.downloadedBlock(ctx, hash)
	if err != nil {
		return downloadedHeader{}, err
	}
	return downloadedHeader{hash: hash, parent: block.ParentHash(), number: block.NumberU64()}, nil
}

func (s *ExecutionSyncer) synchronize(ctx context.Context, target ExecutionTarget) (result error) {
	path := &ancestryPath{}
	usable := true
	defer func() {
		if !usable || path.count == 0 {
			s.ancestry.clear()
			return
		}
		if err := s.ancestry.retain(path); err != nil {
			s.ancestry.clear()
			if result == nil {
				result = err
			}
		}
	}()
	next := target.Head
	var expected uint64
	known := false
	// A pivot callback is a bootstrap path, not a reason to discard the
	// ancestry cursor after restoring authenticated non-genesis state. Both
	// node pivot implementations reject adoption over that existing state.
	bootstrap := s.pivotSync != nil
	if chain, ok := s.chain.(interface{ CurrentBlock() *types.Block }); ok {
		if head := chain.CurrentBlock(); head != nil && head.NumberU64() > 0 {
			bootstrap = false
		}
	}
	// Empty-chain snap adoption still has to observe the CL-finalized hash on
	// this exact target path. Restored chains execute the complete linked path.
	reuse := !bootstrap || s.validated(target.Finalized)
	for !s.validated(next) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if reuse {
			cached, err := s.ancestry.find(next, expected, known)
			if err != nil {
				usable = false
				return err
			}
			if cached != nil {
				if path.count == 0 {
					s.mu.Lock()
					if s.target == target {
						s.targetHeight = cached.headNumber
						s.publishProgressLocked()
					}
					s.mu.Unlock()
				}
				if err := path.join(cached); err != nil {
					return err
				}
				next = path.parent
				expected = path.headNumber - path.count
				known = true
				continue
			}
		}
		header, err := s.downloadedHeader(ctx, next)
		if err != nil {
			return err
		}
		if known && header.number != expected {
			return fmt.Errorf("beacon execution sync: non-contiguous ancestry")
		}
		if path.count == 0 {
			s.mu.Lock()
			if s.target == target {
				s.targetHeight = header.number
				s.publishProgressLocked()
			}
			s.mu.Unlock()
		}
		if reuse {
			cached, err := s.ancestry.find(header.hash, header.number, true)
			if err != nil {
				usable = false
				return err
			}
			if cached != nil {
				if path.count == 0 {
					s.mu.Lock()
					if s.target == target {
						s.targetHeight = cached.headNumber
						s.publishProgressLocked()
					}
					s.mu.Unlock()
				}
				if err := path.join(cached); err != nil {
					return err
				}
				next = path.parent
				expected = path.headNumber - path.count
				known = true
				continue
			}
		}
		if err := s.ancestry.append(path, header); err != nil {
			usable = false
			return err
		}
		if bootstrap && header.hash == target.Finalized {
			block, err := s.downloadedBlock(ctx, header.hash)
			if err != nil {
				return err
			}
			s.mu.Lock()
			s.pivotActive = true
			s.mu.Unlock()
			adopted, err := s.pivotSync(ctx, block)
			s.mu.Lock()
			s.pivotActive = false
			s.mu.Unlock()
			if err != nil {
				return err
			}
			if adopted {
				path.pop(header)
				next = header.hash
				break
			}
		}
		next = header.parent
		expected = header.number - 1
		known = true
	}
	ancestor := s.chain.GetBlock(next)
	if ancestor == nil || ancestor.Header().Number == nil || !ancestor.Header().Number.IsUint64() {
		return fmt.Errorf("beacon execution sync: missing validated anchor")
	}
	parentHash, parentNumber := ancestor.Hash(), ancestor.NumberU64()
	for path.count > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := path.at(path.count - 1)
		if err != nil {
			usable = false
			return err
		}
		if header.parent != parentHash || parentNumber == ^uint64(0) || header.number != parentNumber+1 {
			usable = false
			return fmt.Errorf("beacon execution sync: disconnected forward ancestry")
		}

		if path.count > 1 {
			child, err := path.at(path.count - 2)
			if err != nil {
				usable = false
				return err
			}
			if child.parent != header.hash {
				usable = false
				return fmt.Errorf("beacon execution sync: corrupt staged hash linkage")
			}
		} else if header.hash != path.headHash {
			usable = false
			return fmt.Errorf("beacon execution sync: corrupt staged head")
		}
		block, err := s.downloadedBlock(ctx, header.hash)
		if err != nil {
			return err
		}
		if block.ParentHash() != header.parent || block.NumberU64() != header.number {
			return fmt.Errorf("beacon execution sync: corrupt staged block linkage")
		}
		if !s.validated(header.hash) {
			insert := s.chain.InsertBlock
			if chain, ok := s.chain.(interface{ InsertSyncedBlock(*types.Block) error }); ok {
				insert = chain.InsertSyncedBlock
			}
			if err := insert(block); err != nil {
				return fmt.Errorf("beacon execution sync: block %d: %w", block.NumberU64(), err)
			}
		}
		parentHash, parentNumber = block.Hash(), block.NumberU64()
		path.pop(header)
		s.downloadHeaders.Remove(header.hash)
	}
	if parentHash != target.Head {
		return fmt.Errorf("beacon execution sync: staged path did not reach nominated head")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.apply != nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.target != target || ctx.Err() != nil {
			return context.Canceled
		}
		return s.apply(target)
	}
	return nil
}

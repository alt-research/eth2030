package beacon

import (
	"context"
	"fmt"
	"sync"
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

// ExecutionSyncer owns one replaceable CL target and one joined worker. Failed
// downloads remain pending and retry as peers become available.
type ExecutionSyncer struct {
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
}

func NewExecutionSyncer(chain ExecutionChain, fetcher ExecutionFetcher, apply func(ExecutionTarget) error) *ExecutionSyncer {
	return &ExecutionSyncer{
		chain: chain, fetcher: fetcher, apply: apply, wake: make(chan struct{}, 1),
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
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *ExecutionSyncer) Progress() (bool, uint64, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.target.Head != (types.Hash{}) && !s.complete, s.startHeight, s.targetHeight
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
		s.mu.Lock()
		if s.target == target {
			s.lastErr, s.complete = err, err == nil
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

func (s *ExecutionSyncer) synchronize(ctx context.Context, target ExecutionTarget) error {
	var ancestry []downloadedHeader
	next := target.Head
	// Keep only hash-linked header records on the walk. Body bytes have their own
	// strict budget and are reloaded on forward validation if evicted.
	for !s.validated(next) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(ancestry) >= maxExecutionAncestry {
			return fmt.Errorf("beacon execution sync: ancestry exceeds limit")
		}
		header, err := s.downloadedHeader(ctx, next)
		if err != nil {
			return err
		}
		if len(ancestry) > 0 && (header.number == ^uint64(0) || ancestry[len(ancestry)-1].number != header.number+1) {
			return fmt.Errorf("beacon execution sync: non-contiguous ancestry")
		}
		if len(ancestry) == 0 {
			s.mu.Lock()
			if s.target == target {
				s.targetHeight = header.number
			}
			s.mu.Unlock()
		}
		ancestry = append(ancestry, header)
		if s.pivotSync != nil && header.hash == target.Finalized {
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
				break
			}
		}
		next = header.parent
	}
	for i := len(ancestry) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !s.validated(ancestry[i].hash) {
			block, err := s.downloadedBlock(ctx, ancestry[i].hash)
			if err != nil {
				return err
			}
			insert := s.chain.InsertBlock
			if chain, ok := s.chain.(interface{ InsertSyncedBlock(*types.Block) error }); ok {
				insert = chain.InsertSyncedBlock
			}
			if err := insert(block); err != nil {
				return fmt.Errorf("beacon execution sync: block %d: %w", block.NumberU64(), err)
			}
		}
		s.downloadHeaders.Remove(ancestry[i].hash)
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

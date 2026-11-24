package canonicalproof

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	ErrBusy       = errors.New("canonical prover is busy")
	ErrClosed     = errors.New("canonical prover service is closed")
	ErrJobUnknown = errors.New("unknown canonical proof job")
)

const maxJobs = 8

type Job struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Proof  *Proof `json:"proof,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Service admits one CPU proof at a time. Submission is asynchronous because a
// real proof can exceed HTTP write timeouts. Jobs are node-lifetime records;
// returned proof artifacts remain independently verifiable across restarts.
type Service struct {
	prover     Prover
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.Mutex
	closed     bool
	busy       bool
	jobs       map[string]*Job
	jobCancels map[string]context.CancelFunc
	order      []string
	workers    sync.WaitGroup
	verifiers  chan struct{}
}

func NewService(parent context.Context, prover Prover) *Service {
	ctx, cancel := context.WithCancel(parent)
	return &Service{prover: prover, ctx: ctx, cancel: cancel, jobs: make(map[string]*Job), jobCancels: make(map[string]context.CancelFunc), verifiers: make(chan struct{}, 2)}
}

func (s *Service) Submit(program, input []byte, cfg Config) (Job, error) {
	if err := validateInputs(program, input, cfg); err != nil {
		return Job{}, err
	}
	if !s.prover.Available() {
		return Job{}, ErrUnavailable
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return Job{}, err
	}
	id := hex.EncodeToString(random[:])
	s.mu.Lock()
	if s.closed || s.ctx.Err() != nil {
		s.mu.Unlock()
		return Job{}, ErrClosed
	}
	if s.busy {
		s.mu.Unlock()
		return Job{}, ErrBusy
	}
	if len(s.order) == maxJobs {
		delete(s.jobs, s.order[0])
		s.order = s.order[1:]
	}
	job := &Job{ID: id, Status: "running"}
	s.jobs[id] = job
	s.order = append(s.order, id)
	s.busy = true
	s.workers.Add(1)
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	s.jobCancels[id] = cancel
	ownedProgram, ownedInput := append([]byte(nil), program...), append([]byte(nil), input...)
	s.mu.Unlock()
	go func() {
		defer s.workers.Done()
		defer cancel()
		proof, err := s.prover.Prove(ctx, ownedProgram, ownedInput, cfg)
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.busy = false
		delete(s.jobCancels, id)
		if err != nil {
			job.Status = "failed"
			job.Error = err.Error()
			if errors.Is(err, context.Canceled) {
				job.Status = "cancelled"
			}
			return
		}
		job.Status = "complete"
		job.Proof = proof
	}()
	return Job{ID: id, Status: "running"}, nil
}

func (s *Service) Cancel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[id]
	if job == nil {
		return ErrJobUnknown
	}
	if cancel := s.jobCancels[id]; cancel != nil {
		job.Status = "cancelling"
		cancel()
	}
	return nil
}

func (s *Service) Job(id string) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[id]
	if job == nil {
		return Job{}, ErrJobUnknown
	}
	copy := *job
	if job.Proof != nil {
		proof := *job.Proof
		copy.Proof = &proof
	}
	return copy, nil
}

func (s *Service) Verify(ctx context.Context, proof *Proof, expected Expected) error {
	s.mu.Lock()
	if s.closed || s.ctx.Err() != nil {
		s.mu.Unlock()
		return ErrClosed
	}
	s.workers.Add(1)
	s.mu.Unlock()
	defer s.workers.Done()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	stop := context.AfterFunc(s.ctx, cancel)
	defer stop()
	select {
	case s.verifiers <- struct{}{}:
		defer func() { <-s.verifiers }()
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrBusy
	}
	return s.prover.Verify(ctx, proof, expected)
}

func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	s.cancel()
	s.mu.Unlock()
	s.workers.Wait()
}

package scheduler

import (
	"github.com/akaspin/soil/agent/allocation"
	"github.com/pkg/errors"
	"sync"
)

var (
	AllocationNotFoundError = errors.New("allocation is not found")
	AllocationNotUnique     = errors.New("allocation is not unique")
)

type ExecutorState struct {
	ready      map[string]*allocation.Pod // finished evaluations
	inProgress map[string]*allocation.Pod //
	pending    map[string]*allocation.Pod

	mu *sync.Mutex
}

func NewExecutorState(initial []*allocation.Pod) (s *ExecutorState) {
	s = &ExecutorState{
		ready:      map[string]*allocation.Pod{},
		inProgress: map[string]*allocation.Pod{},
		pending:    map[string]*allocation.Pod{},
		mu:         &sync.Mutex{},
	}
	for _, a := range initial {
		s.ready[a.Header.Name] = a
	}
	return
}

// Submit allocation to pending. Use <nil> for destroy.
// Submit returns ok if pods actually submitted.
func (s *ExecutorState) Submit(name string, pending *allocation.Pod) (ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	latest := s.getLatest(name)
	if !allocation.IsEqual(latest, pending) {
		s.pending[name] = pending
		ok = true
	}
	return
}

// Promote allocation from pending to inProgress and return ready and inProgress pair.
// or error if evaluation is not possible at this time.
func (s *ExecutorState) Promote(name string) (ready, active *allocation.Pod, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var ok bool
	if _, ok = s.inProgress[name]; ok {
		err = errors.Wrapf(AllocationNotUnique, "can't promote pending %s", name)
		return
	}
	if active, ok = s.pending[name]; !ok {
		err = errors.Wrapf(AllocationNotFoundError, "can't promote pending %s", name)
		return
	}

	ready = s.ready[name]
	s.inProgress[name] = active
	delete(s.pending, name)

	return
}

// Commit inProgress to ready
func (s *ExecutorState) Commit(name string, failures []error) (destroyed bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	active, ok := s.inProgress[name]
	if !ok {
		err = errors.Wrapf(AllocationNotFoundError, "can't commit %s", name)
		return
	}
	destroyed = active == nil
	if destroyed {
		delete(s.ready, name)
	} else {
		s.ready[name] = active
	}
	delete(s.inProgress, name)
	return
}

// List
func (s *ExecutorState) ListActual() (res map[string]*allocation.Header) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res = map[string]*allocation.Header{}

	for _, what := range []map[string]*allocation.Pod{
		s.pending, s.inProgress, s.ready,
	} {
		for k, v := range what {
			if _, ok := res[k]; !ok {
				if v == nil {
					res[k] = nil
					continue
				}
				res[k] = v.Header
			}
		}
	}
	for k, v := range res {
		if v == nil {
			delete(res, k)
		}
	}
	return
}

// returns latest (done/inProgress/pending) pod
func (s *ExecutorState) getLatest(name string) (res *allocation.Pod) {
	var ok bool
	if res, ok = s.pending[name]; ok {
		return
	}
	if res, ok = s.inProgress[name]; ok {
		return
	}
	if res, ok = s.ready[name]; ok {
		return
	}
	return
}

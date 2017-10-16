package provision

import (
	"context"
	"github.com/akaspin/logx"
	"github.com/akaspin/soil/agent/allocation"
	"github.com/akaspin/soil/agent/metrics"
	"github.com/akaspin/soil/manifest"
	"github.com/akaspin/supervisor"
	"github.com/coreos/go-systemd/dbus"
	"sync"
)

type Evaluator struct {
	*supervisor.Control
	log         *logx.Log
	systemPaths allocation.SystemPaths
	reporter    metrics.Reporter

	state *EvaluatorState
}

func NewEvaluator(ctx context.Context, log *logx.Log, systemPaths allocation.SystemPaths, state allocation.Recovery, reporter metrics.Reporter) (e *Evaluator) {
	e = &Evaluator{
		Control:     supervisor.NewControl(ctx),
		log:         log.GetLog("provision", "evaluator"),
		systemPaths: systemPaths,
		reporter:    reporter,
		state:       NewEvaluatorState(state),
	}
	return
}

func (e *Evaluator) Open() (err error) {
	err = e.Control.Open()
	e.log.Debug("opened")
	return
}

func (e *Evaluator) Close() (err error) {
	err = e.Control.Close()
	e.log.Debug("closed")
	return
}

// GetConstraint returns defined pod constraints with constraints for
// required resources.
func (e *Evaluator) GetConstraint(pod *manifest.Pod) (res manifest.Constraint) {
	res = pod.GetResourceAllocationConstraint()
	return
}

func (e *Evaluator) Allocate(pod *manifest.Pod, env map[string]string) {
	alloc := allocation.NewPod(e.systemPaths)
	if err := alloc.FromManifest(pod, env); err != nil {
		e.log.Error(err)
		return
	}
	e.submitAllocation(pod.Name, alloc)
}

func (e *Evaluator) Deallocate(name string) {
	e.submitAllocation(name, nil)
}

func (e *Evaluator) submitAllocation(name string, pod *allocation.Pod) {
	next := e.state.Submit(name, pod)
	e.fanOut(next)
}

func (e *Evaluator) fanOut(next []*Evaluation) {
	if len(next) > 0 {
		for _, evaluation := range next {
			go e.executeEvaluation(evaluation)
		}
	}
}

func (e *Evaluator) executeEvaluation(evaluation *Evaluation) {
	var failures []error
	e.log.Debugf("begin: %s", evaluation)
	conn, err := dbus.New()
	if err != nil {
		e.log.Error(err)
		return
	}
	defer conn.Close()

	currentPhase := -1
	var phase []Instruction
	for _, instruction := range evaluation.Plan() {
		if currentPhase < instruction.Phase() {
			currentPhase = instruction.Phase()
			failures = append(failures, e.executePhase(phase, conn)...)
			phase = []Instruction{}
		}
		phase = append(phase, instruction)
	}
	failures = append(failures, e.executePhase(phase, conn)...)

	e.log.Debugf("plan done: %s:%s (failures:%v)", evaluation, evaluation.plan, failures)
	e.log.Infof("evaluation done: %s (failures:%v)", evaluation, failures)
	e.reporter.Count("provision.evaluations", 1)
	e.reporter.Count("provision.failures", int64(len(failures)))
	next := e.state.Commit(evaluation.Name())
	e.fanOut(next)
	return
}

func (e *Evaluator) executePhase(phase []Instruction, conn *dbus.Conn) (failures []error) {
	if len(phase) == 0 {
		return
	}
	e.log.Tracef("begin phase %v", phase)
	ch := make(chan error, len(phase))
	wg := &sync.WaitGroup{}
	wg.Add(len(phase))
	for _, instruction := range phase {
		go func(instruction Instruction) {
			defer wg.Done()
			e.log.Tracef("begin instruction %v", instruction)
			var iErr error
			if iErr = instruction.Execute(conn); iErr != nil {
				e.log.Errorf("error while execute instruction %v: %s", instruction, iErr)
			}
			e.log.Tracef("finish instruction %s", instruction)
			ch <- iErr
		}(instruction)
	}
	go func() {
		for res := range ch {
			if res != nil {
				failures = append(failures, res)
			}
		}
	}()
	wg.Wait()
	e.log.Tracef("finish phase %v", phase)

	return
}

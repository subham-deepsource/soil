package scheduler

import (
	"context"
	"github.com/akaspin/logx"
	"github.com/akaspin/soil/agent/bus"
	"github.com/akaspin/soil/manifest"
	"github.com/akaspin/supervisor"
)

type NotifyFn func(error, bus.Message)

type arbiterEntity struct {
	id         string
	constraint manifest.Constraint
	notifyFn   func(error, bus.Message)
}

type Arbiter struct {
	*supervisor.Control
	log      *logx.Log
	name     string
	required manifest.Constraint

	state    bus.Message
	entities map[string]arbiterEntity

	messageChan chan bus.Message
	bindChan    chan arbiterEntity
	unbindChan  chan arbiterEntity
}

func NewArbiter(ctx context.Context, log *logx.Log, name string, required manifest.Constraint) (a *Arbiter) {
	a = &Arbiter{
		Control:     supervisor.NewControl(ctx),
		log:         log.GetLog("arbiter", name),
		required:    required,
		state:       bus.NewMessage(name, nil),
		entities:    map[string]arbiterEntity{},
		messageChan: make(chan bus.Message),
		bindChan:    make(chan arbiterEntity),
		unbindChan:  make(chan arbiterEntity),
	}
	return
}

func (a *Arbiter) Open() (err error) {
	go a.loop()
	err = a.Control.Open()
	return
}

// Bind entity to arbiter
func (a *Arbiter) Bind(id string, constraint manifest.Constraint, callback func(error, bus.Message)) {
	select {
	case <-a.Control.Ctx().Done():
	case a.bindChan <- arbiterEntity{
		id:         id,
		constraint: constraint,
		notifyFn:   callback,
	}:
	}
}

// Unbind entity from arbiter
func (a *Arbiter) Unbind(id string, callback func()) {
	select {
	case <-a.Control.Ctx().Done():
	case a.unbindChan <- arbiterEntity{
		id: id,
		notifyFn: func(err error, message bus.Message) {
			callback()
		},
	}:
	}
}

func (a *Arbiter) ConsumeMessage(message bus.Message) {
	select {
	case <-a.Control.Ctx().Done():
	case a.messageChan <- message:
	}
}

func (a *Arbiter) loop() {
	log := a.log.GetLog(a.log.Prefix(), append(a.log.Tags(), "loop")...)
LOOP:
	for {
		select {
		case <-a.Control.Ctx().Done():
			break LOOP
		case message := <-a.messageChan:
			log.Tracef("message: %v", message)
			if a.state.IsEqual(message) {
				log.Debugf("skipping update: message is equal")
			}
			a.state = message
			for _, entity := range a.entities {
				a.notify(entity)
			}
		case req := <-a.bindChan:
			log.Debugf("got bind %v", req)
			a.entities[req.id] = req
			log.Infof(`registered "%s" with constraint: %v`, req.id, req.constraint)
			a.notify(req)
		case req := <-a.unbindChan:
			log.Tracef("got unbind %v", req)
			delete(a.entities, req.id)
			log.Infof(`unregistered "%s"`, req.id)
			req.notifyFn(nil, bus.NewMessage(a.name, nil))
		}
	}
}

func (a *Arbiter) notify(entity arbiterEntity) {
	if a.state.IsEmpty() {
		a.log.Tracef(`skipping arbitrate "%s": state is empty`, entity.id)
		return
	}
	a.log.Tracef(`evaluating "%s"`, entity.id)

	if a.required != nil {
		if err := a.required.Check(a.state.GetPayload()); err != nil {
			a.log.Warningf(`notifying "%s" (required): %v`, entity.id, err)
			entity.notifyFn(err, bus.NewMessage(a.name, nil))
			return
		}
	}
	if err := entity.constraint.Check(a.state.GetPayload()); err != nil {
		a.log.Debugf(`notifying "%s": %v`, entity.id, err)
		entity.notifyFn(err, bus.NewMessage(a.name, nil))
		return
	}
	a.log.Debugf(`notifying "%s": ok:%x`, entity.id, a.state.GetPayloadMark())
	entity.notifyFn(nil, a.state)
}

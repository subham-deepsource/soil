package cluster

import (
	"context"
	"fmt"
	"github.com/akaspin/logx"
	"github.com/da-moon/soil/agent/bus"
	"io"
	"net/url"
	"time"
)

const (
	backendLocal  = "local"
	backendConsul = "consul"
)

type BackendConfig struct {
	Kind    string
	ID      string
	Address string
	Chroot  string
	TTL     time.Duration
}

type WatchRequest struct {
	Key string
	Ctx context.Context
}

type WatchResult struct {
	Key  string
	Data map[string][]byte
}

type Backend interface {
	io.Closer

	Ctx() context.Context      // Backend context closes on backend is not available to accept operations
	FailCtx() context.Context  // Fail context closes then backend is failed
	ReadyCtx() context.Context // Ready context closes then backend is ready to accept operations
	Submit(ops []StoreOp)      // Submit ops to backend
	Subscribe(req []WatchRequest)
	CommitChan() chan []StoreCommit
	WatchResultsChan() chan WatchResult
	Leave() // Leave cluster
}

type BackendFactory func(ctx context.Context, log *logx.Log, config Config) (c Backend, err error)

func DefaultBackendFactory(ctx context.Context, log *logx.Log, config Config) (c Backend, err error) {
	kvConfig := BackendConfig{
		Kind:    "local",
		Chroot:  "soil",
		ID:      config.NodeID,
		Address: "localhost",
		TTL:     config.TTL,
	}
	u, err := url.Parse(config.BackendURL)
	if err != nil {
		log.Error(err)
	}
	if err == nil {
		kvConfig.Kind = u.Scheme
		kvConfig.Address = u.Host
		kvConfig.Chroot = NormalizeKey(u.Path)
	}
	kvLog := log.GetLog("cluster", "backend", config.BackendURL, config.NodeID)
	if kvConfig.ID == "" {
		err = fmt.Errorf(`empty node id`)
		return
	}
	switch kvConfig.Kind {
	case backendConsul:
		c = NewConsulBackend(ctx, kvLog, kvConfig)
	default:
		c = NewZeroBackend(ctx, kvLog)
	}
	return
}

type StoreOp struct {
	Message bus.Message
	WithTTL bool
}

type StoreCommit struct {
	ID      string
	Hash    uint64
	WithTTL bool
}

//go:build ide || test_unit
// +build ide test_unit

package cluster_test

import (
	"context"
	"github.com/akaspin/logx"
	"github.com/da-moon/soil/agent/bus"
	"github.com/da-moon/soil/agent/cluster"
	"github.com/da-moon/soil/fixture"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestKV_Configure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer := bus.NewTestingConsumer(ctx)
	backendCfg := cluster.TestingBackendConfig{
		Consumer:    consumer,
		ReadyChan:   make(chan struct{}, 1),
		CrashChan:   make(chan struct{}, 1),
		MessageChan: make(chan map[string]map[string]interface{}),
	}
	kv := cluster.NewKV(ctx, logx.GetLog("test"), cluster.NewTestingBackendFactory(backendCfg))
	assert.NoError(t, kv.Open())

	kvConfig := cluster.DefaultConfig()
	kvConfig.NodeID = "localhost"
	kvConfig.RetryInterval = time.Millisecond * 10

	watcherCtx, _ := context.WithCancel(context.Background())
	watcher := bus.NewTestingConsumer(ctx)

	waitConfig := fixture.DefaultWaitConfig()

	t.Run(`configure and crash after 100ms`, func(t *testing.T) {
		kv.Configure(kvConfig)
		go func() {
			<-time.After(time.Millisecond * 100)
			backendCfg.CrashChan <- struct{}{}
		}()
	})
	t.Run(`store and watch`, func(t *testing.T) {
		kv.Submit([]cluster.StoreOp{
			{bus.NewMessage("pre-volatile", map[string]string{"1": "1"}), true},
		})
		kv.Submit([]cluster.StoreOp{
			{bus.NewMessage("pre-permanent", map[string]string{"1": "1"}), false},
		})
		kv.SubscribeKey("down", watcherCtx, watcher)
	})
	t.Run(`start backend`, func(t *testing.T) {
		go func() {
			<-time.After(time.Millisecond * 200)
			backendCfg.ReadyChan <- struct{}{}
		}()
		fixture.WaitNoErrorT(t, waitConfig, consumer.ExpectMessagesFn(
			bus.NewMessage("crash", map[string]interface{}{}),
			bus.NewMessage("test", map[string]interface{}{
				"pre-volatile": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  true,
				},
				"pre-permanent": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  false,
				},
			}),
		))
	})
}

func TestKV_Submit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer := bus.NewTestingConsumer(ctx)
	backendCfg := cluster.TestingBackendConfig{
		Consumer:    consumer,
		ReadyChan:   make(chan struct{}, 1),
		CrashChan:   make(chan struct{}, 1),
		MessageChan: make(chan map[string]map[string]interface{}),
	}
	kv := cluster.NewKV(ctx, logx.GetLog("test"), cluster.NewTestingBackendFactory(backendCfg))
	assert.NoError(t, kv.Open())

	waitConfig := fixture.DefaultWaitConfig()

	t.Run(`submit on zero`, func(t *testing.T) {
		kv.Submit([]cluster.StoreOp{
			{bus.NewMessage("pre-volatile", map[string]string{"1": "1"}), true},
		})
		kv.Submit([]cluster.StoreOp{
			{bus.NewMessage("pre-permanent", map[string]string{"1": "1"}), false},
		})
		fixture.WaitNoErrorT(t, waitConfig, consumer.ExpectMessagesFn())
	})
	t.Run(`configure 1`, func(t *testing.T) {
		config := cluster.DefaultConfig()
		config.NodeID = "localhost"
		config.RetryInterval = time.Millisecond * 10
		kv.Configure(config)
		go func() {
			<-time.After(time.Millisecond * 100)
			backendCfg.ReadyChan <- struct{}{}
		}()
	})
	t.Run(`ensure submit after config`, func(t *testing.T) {
		fixture.WaitNoErrorT(t, waitConfig, consumer.ExpectMessagesFn(
			bus.NewMessage("test", map[string]interface{}{
				"pre-volatile": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  true,
				},
				"pre-permanent": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  false,
				},
			}),
		))
	})
	t.Run(`ensure resubmit volatile after crash`, func(t *testing.T) {
		backendCfg.CrashChan <- struct{}{}
		go func() {
			<-time.After(time.Millisecond * 100)
			backendCfg.ReadyChan <- struct{}{}
		}()
		fixture.WaitNoErrorT(t, waitConfig, consumer.ExpectMessagesFn(
			bus.NewMessage("test", map[string]interface{}{
				"pre-volatile": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  true,
				},
				"pre-permanent": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  false,
				},
			}),
			bus.NewMessage("crash", map[string]interface{}{}),
			bus.NewMessage("test", map[string]interface{}{
				"pre-volatile": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  true,
				},
			}),
		))
	})
	t.Run(`ensure noop after equal config`, func(t *testing.T) {
		config := cluster.DefaultConfig()
		config.NodeID = "localhost"
		config.RetryInterval = time.Millisecond * 10
		kv.Configure(config)

		fixture.WaitNoErrorT(t, waitConfig, consumer.ExpectMessagesFn(
			bus.NewMessage("test", map[string]interface{}{
				"pre-volatile": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  true,
				},
				"pre-permanent": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  false,
				},
			}),
			bus.NewMessage("crash", map[string]interface{}{}),
			bus.NewMessage("test", map[string]interface{}{
				"pre-volatile": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  true,
				},
			}),
		))
	})
	t.Run(`remove`, func(t *testing.T) {
		kv.Submit([]cluster.StoreOp{
			{bus.NewMessage("pre-volatile", nil), true},
		})

		fixture.WaitNoErrorT(t, waitConfig, consumer.ExpectMessagesFn(
			bus.NewMessage("test", map[string]interface{}{
				"pre-volatile": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  true,
				},
				"pre-permanent": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  false,
				},
			}),
			bus.NewMessage("crash", map[string]interface{}{}),
			bus.NewMessage("test", map[string]interface{}{
				"pre-volatile": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  true,
				},
			}),
			bus.NewMessage("test", map[string]interface{}{
				"pre-volatile": map[string]interface{}{
					"Data": nil,
					"TTL":  true,
				},
			}),
		))
	})
	t.Run(`add`, func(t *testing.T) {
		kv.Submit([]cluster.StoreOp{
			{bus.NewMessage("post-volatile", map[string]string{"1": "1"}), true},
		})

		fixture.WaitNoErrorT(t, waitConfig, consumer.ExpectMessagesFn(
			bus.NewMessage("test", map[string]interface{}{
				"pre-volatile": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  true,
				},
				"pre-permanent": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  false,
				},
			}),
			bus.NewMessage("crash", map[string]interface{}{}),
			bus.NewMessage("test", map[string]interface{}{
				"pre-volatile": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  true,
				},
			}),
			bus.NewMessage("test", map[string]interface{}{
				"pre-volatile": map[string]interface{}{
					"Data": nil,
					"TTL":  true,
				},
			}),
			bus.NewMessage("test", map[string]interface{}{
				"post-volatile": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  true,
				},
			}),
		))
	})

	kv.Close()
	kv.Wait()
}

func TestKV_Subscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	consumer := bus.NewTestingConsumer(ctx)
	msgChan := make(chan map[string]map[string]interface{})

	backendCfg := cluster.TestingBackendConfig{
		Consumer:    consumer,
		ReadyChan:   make(chan struct{}, 1),
		CrashChan:   make(chan struct{}, 1),
		MessageChan: msgChan,
	}
	kv := cluster.NewKV(ctx, logx.GetLog("test"), cluster.NewTestingBackendFactory(backendCfg))
	assert.NoError(t, kv.Open())
	backendCfg.ReadyChan <- struct{}{}

	cons1 := bus.NewTestingConsumer(ctx)
	ctx1, _ := context.WithCancel(context.Background())
	cons2 := bus.NewTestingConsumer(ctx)
	ctx2, cancel2 := context.WithCancel(context.Background())

	waitConfig := fixture.DefaultWaitConfig()

	t.Run(`subscribe 1`, func(t *testing.T) {
		kv.SubscribeKey("test/1", ctx1, cons1)
		time.Sleep(time.Millisecond * 100)
	})
	t.Run(`configure 1`, func(t *testing.T) {
		config := cluster.DefaultConfig()
		config.NodeID = "localhost"
		config.RetryInterval = time.Millisecond * 10
		kv.Configure(config)
		time.Sleep(time.Millisecond * 100)
	})
	t.Run(`put 1`, func(t *testing.T) {
		msgChan <- map[string]map[string]interface{}{
			"test/1": {"1": "1"},
		}
		fixture.WaitNoErrorT(t, waitConfig, cons1.ExpectMessagesFn(
			bus.NewMessage("test/1", map[string]string{"1": "1"}),
		))
	})
	t.Run(`put 1 duplicate`, func(t *testing.T) {
		msgChan <- map[string]map[string]interface{}{
			"test/1": {"1": "1"},
		}
		fixture.WaitNoErrorT(t, waitConfig, cons1.ExpectMessagesFn(
			bus.NewMessage("test/1", map[string]string{"1": "1"}),
		))
	})
	t.Run(`subscribe 2`, func(t *testing.T) {
		kv.SubscribeKey("test/1", ctx2, cons2)
		fixture.WaitNoErrorT(t, waitConfig, cons2.ExpectMessagesFn(
			bus.NewMessage("test/1", map[string]string{"1": "1"}),
		))
	})
	t.Run(`crash`, func(t *testing.T) {
		backendCfg.CrashChan <- struct{}{}
		go func() {
			time.After(time.Millisecond * 100)
			backendCfg.ReadyChan <- struct{}{}
		}()
		fixture.WaitNoErrorT(t, waitConfig, cons1.ExpectMessagesFn(
			bus.NewMessage("test/1", map[string]string{"1": "1"}),
		))
		fixture.WaitNoErrorT(t, waitConfig, cons2.ExpectMessagesFn(
			bus.NewMessage("test/1", map[string]string{"1": "1"}),
		))
	})
	t.Run(`unsubscribe 2`, func(t *testing.T) {
		cancel2()
		time.Sleep(time.Millisecond * 100)
		msgChan <- map[string]map[string]interface{}{
			"test/1": {"1": "2"},
		}
		fixture.WaitNoErrorT(t, waitConfig, cons1.ExpectMessagesFn(
			bus.NewMessage("test/1", map[string]string{"1": "1"}),
			bus.NewMessage("test/1", map[string]string{"1": "2"}),
		))
		fixture.WaitNoErrorT(t, waitConfig, cons2.ExpectMessagesFn(
			bus.NewMessage("test/1", map[string]string{"1": "1"}),
		))
	})
	t.Run(`close`, func(t *testing.T) {
		kv.Close()
		time.Sleep(time.Second)
	})

}

func TestStore_ConsumeMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	consumer := bus.NewTestingConsumer(ctx)
	backendCfg := cluster.TestingBackendConfig{
		Consumer:    consumer,
		ReadyChan:   make(chan struct{}, 1),
		CrashChan:   make(chan struct{}, 1),
		MessageChan: nil,
	}
	kv := cluster.NewKV(context.Background(), logx.GetLog("test"), cluster.NewTestingBackendFactory(backendCfg))
	assert.NoError(t, kv.Open())
	backendCfg.ReadyChan <- struct{}{}

	t.Run(`configure`, func(t *testing.T) {
		config := cluster.DefaultConfig()
		config.NodeID = "localhost"
		config.RetryInterval = time.Millisecond * 10
		kv.Configure(config)
		time.Sleep(time.Millisecond * 100)
	})
	t.Run(`volatile with prefix`, func(t *testing.T) {
		store := kv.VolatileStore("prefix")
		store.ConsumeMessage(bus.NewMessage("1", map[string]string{
			"1": "1",
		}))
		fixture.WaitNoErrorT(t, fixture.DefaultWaitConfig(), consumer.ExpectMessagesFn(
			bus.NewMessage("test", map[string]interface{}{
				"prefix/1": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  true,
				},
			}),
		))
		store.ConsumeMessage(bus.NewMessage("", map[string]string{
			"1": "2",
		}))
		fixture.WaitNoErrorT(t, fixture.DefaultWaitConfig(), consumer.ExpectMessagesFn(
			bus.NewMessage("test", map[string]interface{}{
				"prefix/1": map[string]interface{}{
					"Data": map[string]string{"1": "1"},
					"TTL":  true,
				},
			}),
			bus.NewMessage("test", map[string]interface{}{
				"prefix": map[string]interface{}{
					"Data": map[string]string{"1": "2"},
					"TTL":  true,
				},
			}),
		))
	})
}

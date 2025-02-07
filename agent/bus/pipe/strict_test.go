//go:build ide || test_unit
// +build ide test_unit

package pipe_test

import (
	"context"
	"github.com/akaspin/logx"
	"github.com/da-moon/soil/agent/bus"
	"github.com/da-moon/soil/agent/bus/pipe"
	"github.com/da-moon/soil/fixture"
	"testing"
)

func TestCompositePipe_ConsumeMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dummy := bus.NewTestingConsumer(ctx)
	strictPipe := pipe.NewStrict("test", logx.GetLog("test"), dummy, "1", "2")

	t.Run("1", func(t *testing.T) {
		strictPipe.ConsumeMessage(bus.NewMessage("1", map[string]string{
			"1": "1",
		}))
		fixture.WaitNoErrorT(t, fixture.DefaultWaitConfig(), dummy.ExpectMessagesFn(
			bus.NewMessage("test", nil),
		))
	})
	t.Run("2", func(t *testing.T) {
		strictPipe.ConsumeMessage(bus.NewMessage("2", map[string]string{
			"2": "2",
		}))
		fixture.WaitNoErrorT(t, fixture.DefaultWaitConfig(), dummy.ExpectMessagesFn(
			bus.NewMessage("test", nil),
			bus.NewMessage("test", map[string]string{
				"1.1": "1",
				"2.2": "2",
			}),
		))
	})
	t.Run("2 off", func(t *testing.T) {
		strictPipe.ConsumeMessage(bus.NewMessage("2", nil))
		fixture.WaitNoErrorT(t, fixture.DefaultWaitConfig(), dummy.ExpectMessagesFn(
			bus.NewMessage("test", nil),
			bus.NewMessage("test", map[string]string{
				"1.1": "1",
				"2.2": "2",
			}),
			bus.NewMessage("test", nil),
		))
	})
	t.Run("2 on", func(t *testing.T) {
		strictPipe.ConsumeMessage(bus.NewMessage("2", map[string]string{
			"2": "3",
		}))
		fixture.WaitNoErrorT(t, fixture.DefaultWaitConfig(), dummy.ExpectMessagesFn(
			bus.NewMessage("test", nil),
			bus.NewMessage("test", map[string]string{
				"1.1": "1",
				"2.2": "2",
			}),
			bus.NewMessage("test", nil),
			bus.NewMessage("test", map[string]string{
				"1.1": "1",
				"2.2": "3",
			}),
		))
	})
}

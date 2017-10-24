// +build ide test_unit

package bus_test

import (
	"github.com/akaspin/soil/agent/bus"
	"testing"
)

func TestCompositePipe_ConsumeMessage(t *testing.T) {
	dummy := &bus.DummyConsumer{}
	pipe := bus.NewCompositePipe("test", dummy, "1", "2")

	t.Run("1", func(t *testing.T) {
		pipe.ConsumeMessage(bus.NewMessage("1", map[string]string{
			"1": "1",
		}))
		dummy.AssertMessages(t, bus.NewMessage("test", nil))
	})
	t.Run("2", func(t *testing.T) {
		pipe.ConsumeMessage(bus.NewMessage("2", map[string]string{
			"2": "2",
		}))
		dummy.AssertMessages(t,
			bus.NewMessage("test", nil),
			bus.NewMessage("test", map[string]string{
				"1.1": "1",
				"2.2": "2",
			}))
	})
	t.Run("2 off", func(t *testing.T) {
		pipe.ConsumeMessage(bus.NewMessage("2", nil))
		dummy.AssertMessages(t,
			bus.NewMessage("test", nil),
			bus.NewMessage("test", map[string]string{
				"1.1": "1",
				"2.2": "2",
			}),
			bus.NewMessage("test", nil),
		)
	})
	t.Run("2 on", func(t *testing.T) {
		pipe.ConsumeMessage(bus.NewMessage("2", map[string]string{
			"2": "3",
		}))
		dummy.AssertMessages(t,
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
		)
	})
}
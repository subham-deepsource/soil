//go:build ide || test_unit
// +build ide test_unit

package resource_test

import (
	"context"
	"fmt"
	"github.com/akaspin/logx"
	"github.com/da-moon/soil/agent/allocation"
	"github.com/da-moon/soil/agent/bus"
	"github.com/da-moon/soil/agent/bus/pipe"
	"github.com/da-moon/soil/agent/resource"
	"github.com/da-moon/soil/fixture"
	"github.com/da-moon/soil/manifest"
	"testing"
)

func TestSandbox(t *testing.T) {
	for i := 0; i < 5; i++ {
		t.Run(fmt.Sprintf("i-%d", i), func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cons := bus.NewTestingConsumer(ctx)
			upstream := bus.NewTestingConsumer(ctx)
			sb := resource.NewSandbox(
				resource.SandboxConfig{
					Ctx:        ctx,
					Log:        logx.GetLog("test"),
					Downstream: pipe.NewLift("0", cons),
					Upstream:   upstream,
				},
				"pod1.test",
				&allocation.Provider{
					Name: "test",
					Kind: "blackhole",
				})

			t.Run(`check blackbox kind`, func(t *testing.T) {
				fixture.WaitNoErrorT10(t, upstream.ExpectMessagesFn(
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "blackhole",
					}),
				))

			})

			t.Run(`recover 1 and 2`, func(t *testing.T) {
				sb.Create("1", &allocation.Resource{
					Request: manifest.Resource{
						Name:     "1",
						Provider: "pod1.test1",
					},
					Values: manifest.FlatMap{
						"allocated": "true",
						"value":     "8080",
					},
				})
				sb.Create("2", &allocation.Resource{
					Request: manifest.Resource{
						Name:     "2",
						Provider: "pod1.test1",
					},
					Values: manifest.FlatMap{
						"allocated": "true",
						"value":     "8081",
					},
				})
				fixture.WaitNoErrorT10(t, cons.ExpectMessagesFn())
			})
			t.Run(`reconfigure in range`, func(t *testing.T) {
				sb.Configure(&allocation.Provider{
					Name: "test",
					Kind: "range",
					Config: map[string]interface{}{
						"min": 3000,
						"max": 9000,
					},
				})
				fixture.WaitNoErrorT10(t, cons.ExpectLastMessageFn(
					bus.NewMessage("0", map[string]string{
						"1.allocated": "true",
						"1.provider":  "pod1.test",
						"1.value":     "8080",
						"2.allocated": "true",
						"2.provider":  "pod1.test",
						"2.value":     "8081",
					}),
				))
				fixture.WaitNoErrorT10(t, upstream.ExpectMessagesFn(
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "blackhole",
					}),
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "range",
					}),
				))
			})
			t.Run(`destroy 1`, func(t *testing.T) {
				sb.Destroy(`1`)
				fixture.WaitNoErrorT10(t, cons.ExpectLastMessageFn(
					bus.NewMessage("0", map[string]string{
						"2.allocated": "true",
						"2.provider":  "pod1.test",
						"2.value":     "8081",
					}),
				))
			})
			t.Run(`reconfigure not in range`, func(t *testing.T) {
				sb.Configure(&allocation.Provider{
					Name: "test",
					Kind: "range",
					Config: map[string]interface{}{
						"min": 3000,
						"max": 4000,
					},
				})
				fixture.WaitNoErrorT10(t, cons.ExpectLastMessageFn(
					bus.NewMessage("0", map[string]string{
						"2.allocated": "true",
						"2.provider":  "pod1.test",
						"2.value":     "3000",
					}),
				))
				fixture.WaitNoErrorT10(t, upstream.ExpectMessagesFn(
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "blackhole",
					}),
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "range",
					}),
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "range",
					}),
				))
			})
			t.Run("create 3", func(t *testing.T) {
				sb.Create("3", &allocation.Resource{
					Request: manifest.Resource{
						Provider: "test",
						Name:     "3",
					},
				})
				fixture.WaitNoErrorT10(t, cons.ExpectLastMessageFn(
					bus.NewMessage("0", map[string]string{
						"2.allocated": "true",
						"2.provider":  "pod1.test",
						"2.value":     "3000",
						"3.allocated": "true",
						"3.provider":  "pod1.test",
						"3.value":     "3001",
					}),
				))
			})
			t.Run("destroy 2", func(t *testing.T) {
				sb.Destroy("2")
				fixture.WaitNoErrorT10(t, cons.ExpectLastMessageFn(
					bus.NewMessage("0", map[string]string{
						"3.allocated": "true",
						"3.provider":  "pod1.test",
						"3.value":     "3001",
					}),
				))
			})
			t.Run(`invalid`, func(t *testing.T) {
				sb.Configure(&allocation.Provider{
					Name: "test",
					Kind: "bad",
					Config: map[string]interface{}{
						"min": 3000,
						"max": 4000,
					},
				})
				fixture.WaitNoErrorT10(t, upstream.ExpectMessagesFn(
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "blackhole",
					}),
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "range",
					}),
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "range",
					}),
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "bad",
					}),
				))

				fixture.WaitNoErrorT10(t, cons.ExpectLastMessageFn(
					bus.NewMessage("0", map[string]string{
						"3.allocated": "false",
						"3.provider":  "pod1.test",
						"3.failure":   "invalid-provider-kind",
					}),
				))
			})
			t.Run(`range`, func(t *testing.T) {
				sb.Configure(&allocation.Provider{
					Name: "test",
					Kind: "range",
					Config: map[string]interface{}{
						"min": 3000,
						"max": 4000,
					},
				})
				fixture.WaitNoErrorT10(t, upstream.ExpectMessagesFn(
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "blackhole",
					}),
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "range",
					}),
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "range",
					}),
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "bad",
					}),
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "range",
					}),
				))
				fixture.WaitNoErrorT10(t, cons.ExpectLastMessageFn(
					bus.NewMessage("0", map[string]string{
						"3.allocated": "true",
						"3.provider":  "pod1.test",
						"3.value":     "3000",
					}),
				))
			})
			t.Run(`shutdown`, func(t *testing.T) {
				sb.Shutdown()
				fixture.WaitNoErrorT10(t, cons.ExpectLastMessageFn(
					bus.NewMessage("0", map[string]string{}),
				))
				fixture.WaitNoErrorT10(t, upstream.ExpectMessagesFn(
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "blackhole",
					}),
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "range",
					}),
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "range",
					}),
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "bad",
					}),
					bus.NewMessage("pod1.test", map[string]string{
						"allocated": "true",
						"kind":      "range",
					}),
					bus.NewMessage("pod1.test", nil),
				))
			})
			t.Run(`close`, func(t *testing.T) {
				sb.Close()
			})
		})
	}

}

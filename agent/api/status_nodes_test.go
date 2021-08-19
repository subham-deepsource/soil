//go:build ide || test_unit
// +build ide test_unit

package api_test

import (
	"context"
	"fmt"
	"github.com/akaspin/logx"
	"github.com/da-moon/soil/agent/api"
	"github.com/da-moon/soil/agent/bus"
	"github.com/da-moon/soil/fixture"
	"github.com/da-moon/soil/proto"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestClusterNodesProcessor_Process(t *testing.T) {
	processor := api.NewClusterNodesGet(logx.GetLog("test")).Processor()

	nodes := proto.NodesInfo{
		{
			ID:        "1",
			Advertise: "one",
			Version:   "0.1",
			API:       "v1",
		},
		{
			ID:        "2",
			Advertise: "two",
			Version:   "0.1",
			API:       "v1",
		},
	}

	t.Run(`empty`, func(t *testing.T) {
		res, _ := processor.Process(context.Background(), nil, nil)
		assert.Nil(t, res)
	})
	t.Run(`with nodes`, func(t *testing.T) {
		processor.(bus.Consumer).ConsumeMessage(bus.NewMessage("1", nodes))
		fixture.WaitNoErrorT10(t, func() error {
			res, _ := processor.Process(context.Background(), nil, nil)
			if !reflect.DeepEqual(res, nodes) {
				return fmt.Errorf(`not equal`)
			}
			return nil
		})
	})
}

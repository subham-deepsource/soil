package scheduler_test

import (
	"context"
	"github.com/akaspin/logx"
	"github.com/akaspin/soil/agent"
	"github.com/akaspin/soil/agent/scheduler"
	"github.com/akaspin/soil/agent/source"
	"github.com/akaspin/soil/fixture"
	"github.com/akaspin/soil/manifest"
	"github.com/akaspin/supervisor"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewScheduler(t *testing.T) {
	fixture.RunTestUnless(t, "TEST_INTEGRATION")
	fixture.RunTestIf(t, "TEST_SYSTEMD")

	sd := fixture.NewSystemd("/run/systemd/system", "pod")
	defer sd.Cleanup()

	ctx := context.Background()
	log := logx.GetLog("test")

	t.Run("0", func(t *testing.T) {
		agentSource := source.NewMap(ctx, log, "agent", true, manifest.Constraint{
			"${agent.drain}": "false",
		})
		metaSource := source.NewMap(ctx, log, "meta", true, manifest.Constraint{})
		sourceSV := supervisor.NewGroup(ctx, agentSource, metaSource)

		sink, schedulerSV := scheduler.New(ctx, log, []agent.Source{agentSource, metaSource}, nil)
		sv := supervisor.NewChain(ctx, sourceSV, schedulerSV)

		assert.NoError(t, sv.Open())
		// premature init arbiters
		metaSource.Set(map[string]string{
			"first_private":  "1",
			"second_private": "1",
			"third_public":   "1",
		}, true)
		agentSource.Set(map[string]string{
			"id":       "one",
			"pod_exec": "ExecStart=/usr/bin/sleep inf",
			"drain":    "false",
		}, true)
		private, err := manifest.ParseFromFiles("private", "testdata/scheduler_test_0_private.hcl")
		assert.NoError(t, err)
		sink.Sync("private", private)
		public, err := manifest.ParseFromFiles("public", "testdata/scheduler_test_0_public.hcl")
		assert.NoError(t, err)
		sink.Sync("public", public)

		time.Sleep(time.Second)

		res, err := sd.ListPods()
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{
			"first":  "/run/systemd/system/pod-private-first.service",
			"second": "/run/systemd/system/pod-private-second.service",
			"third":  "/run/systemd/system/pod-public-third.service",
		}, res)

		assert.NoError(t, sv.Close())
		assert.NoError(t, sv.Wait())
	})

	// create new arbiter

	agentSource := source.NewMap(ctx, log, "agent", true, manifest.Constraint{})
	metaSource := source.NewMap(ctx, log, "meta", true, manifest.Constraint{})
	sourceSV := supervisor.NewGroup(ctx, agentSource, metaSource)
	sink, schedulerSv := scheduler.New(ctx, log, []agent.Source{agentSource, metaSource}, nil)
	sv := supervisor.NewChain(ctx, sourceSV, schedulerSv)
	// premature init arbiters
	assert.NoError(t, sv.Open())
	metaSource.Set(map[string]string{
		"first_private":  "1",
		"second_private": "1",
	}, true)
	agentSource.Set(map[string]string{
		"id":       "one",
		"pod_exec": "ExecStart=/usr/bin/sleep inf",
	}, true)

	t.Run("1", func(t *testing.T) {
		// assert all pods are still running
		res, err := sd.ListPods()
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{
			"first":  "/run/systemd/system/pod-private-first.service",
			"second": "/run/systemd/system/pod-private-second.service",
			"third":  "/run/systemd/system/pod-public-third.service",
		}, res)
	})
	t.Run("2", func(t *testing.T) {
		// re sync private
		private, err := manifest.ParseFromFiles("private", "testdata/scheduler_test_2_private.hcl")
		assert.NoError(t, err)
		sink.Sync("private", private)
		time.Sleep(time.Second)

		// Deploy first in public namespace
		public, err := manifest.ParseFromFiles("public", "testdata/scheduler_test_2_public.hcl")
		assert.NoError(t, err)
		sink.Sync("public", public)
		time.Sleep(time.Second)

		// assert first pod is not overrided by public
		// assert third pod is gone
		res, err := sd.ListPods()
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{
			"first":  "/run/systemd/system/pod-private-first.service",
			"second": "/run/systemd/system/pod-private-second.service",
		}, res)
	})
	t.Run("3", func(t *testing.T) {
		// Remove first private
		private, err := manifest.ParseFromFiles("private", "testdata/scheduler_test_3_private.hcl")
		assert.NoError(t, err)
		sink.Sync("private", private)
		time.Sleep(time.Second)

		// ensure first is gone
		res, err := sd.ListPods()
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{
			"second": "/run/systemd/system/pod-private-second.service",
		}, res)
	})
	t.Run("4", func(t *testing.T) {
		// modify meta
		metaSource.Set(map[string]string{
			"first_private":  "1",
			"first_public":   "1",
			"second_private": "1",
		}, true)
		time.Sleep(time.Second)

		// ensure first public is deployed
		res, err := sd.ListPods()
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{
			"first":  "/run/systemd/system/pod-public-first.service",
			"second": "/run/systemd/system/pod-private-second.service",
		}, res)
	})
	t.Run("5", func(t *testing.T) {
		// reenter first private
		private, err := manifest.ParseFromFiles("private", "testdata/scheduler_test_5_private.hcl")
		assert.NoError(t, err)
		sink.Sync("private", private)
		time.Sleep(time.Second)

		// ensure first is private
		res, err := sd.ListPods()
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{
			"first":  "/run/systemd/system/pod-private-first.service",
			"second": "/run/systemd/system/pod-private-second.service",
		}, res)
	})
	t.Run("6", func(t *testing.T) {
		// remove first private
		private, err := manifest.ParseFromFiles("private", "testdata/scheduler_test_6_private.hcl")
		assert.NoError(t, err)
		sink.Sync("private", private)
		time.Sleep(time.Second)

		// ensure first is public
		res, err := sd.ListPods()
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{
			"first":  "/run/systemd/system/pod-public-first.service",
			"second": "/run/systemd/system/pod-private-second.service",
		}, res)
	})
	t.Run("7", func(t *testing.T) {
		// update private and meta
		private, err := manifest.ParseFromFiles("private", "testdata/scheduler_test_7_private.hcl")
		assert.NoError(t, err)
		sink.Sync("private", private)

		metaSource.Set(map[string]string{
			"first_private":  "2",
			"first_public":   "1",
			"second_private": "1",
		}, true)
		time.Sleep(time.Second)

		// ensure first is public
		res, err := sd.ListPods()
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{
			"second": "/run/systemd/system/pod-private-second.service",
		}, res)
	})

	assert.NoError(t, sv.Close())
	assert.NoError(t, sv.Wait())
}

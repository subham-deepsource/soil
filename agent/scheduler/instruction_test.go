package scheduler_test

import (
	"fmt"
	"github.com/akaspin/soil/agent/allocation"
	"github.com/akaspin/soil/agent/scheduler"
	"github.com/akaspin/soil/fixture"
	"github.com/coreos/go-systemd/dbus"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestWantsInstruction_Execute(t *testing.T) {
	sd := fixture.NewSystemd("/run/systemd/system", "pod")
	defer sd.Cleanup()
	assert.NoError(t, sd.DeployPod("test-1", 3))
	assert.NoError(t, sd.DeployPod("test-2", 3))

	conn, err := dbus.New()
	assert.NoError(t, err)
	defer conn.Close()

	unitFile := &allocation.UnitFile{
		Path: "/run/systemd/system/test-1-0.service",
	}
	assert.NoError(t, unitFile.Read())

	t.Run("disable", func(t *testing.T) {
		assert.NoError(t, scheduler.NewDisableUnitInstruction(unitFile).Execute(conn))
		_, err = os.Stat("/run/systemd/system/multi-user.target.wants/test-1-0.service")
		assert.Error(t, err)
	})
	t.Run("enable", func(t *testing.T) {
		assert.NoError(t, scheduler.NewEnableUnitInstruction(unitFile).Execute(conn))
		_, err = os.Stat("/run/systemd/system/multi-user.target.wants/test-1-0.service")
		assert.NoError(t, err)
	})
}

func TestExecuteCommandInstruction_Execute(t *testing.T) {
	sd := fixture.NewSystemd("/run/systemd/system", "pod")
	defer sd.Cleanup()
	assert.NoError(t, sd.DeployPod("test-1", 3))
	assert.NoError(t, sd.DeployPod("test-2", 3))

	conn, err := dbus.New()
	assert.NoError(t, err)
	defer conn.Close()

	unitFile := &allocation.UnitFile{
		Path: "/run/systemd/system/test-1-0.service",
	}
	assert.NoError(t, unitFile.Read())

	testCommand := func(command string, state string) (err error) {
		c := scheduler.NewCommandInstruction(0, unitFile, command)
		if err = c.Execute(conn); err != nil {
			return
		}
		var res []dbus.UnitStatus
		if res, err = conn.ListUnitsByNames([]string{unitFile.UnitName()}); err != nil {
			return
		}
		if res[0].ActiveState != state {
			err = fmt.Errorf("%s != %s", state, res[0].ActiveState)
		}
		return
	}

	t.Run("stop", func(t *testing.T) {
		assert.NoError(t, testCommand("stop", "inactive"))
	})
	t.Run("start", func(t *testing.T) {
		assert.NoError(t, testCommand("start", "active"))
	})
	t.Run("restart", func(t *testing.T) {
		assert.NoError(t, testCommand("restart", "active"))
	})
	t.Run("reload", func(t *testing.T) {
		assert.NoError(t, testCommand("reload", "active"))
	})
	t.Run("try-restart", func(t *testing.T) {
		// stop unit first
		ch := make(chan string)
		conn.StopUnit(unitFile.UnitName(), "replace", ch)
		<-ch

		assert.NoError(t, testCommand("try-restart", "inactive"))
	})
}

func TestFSInstruction_Execute(t *testing.T) {
	sd := fixture.NewSystemd("/run/systemd/system", "pod")
	defer sd.Cleanup()
	assert.NoError(t, sd.DeployPod("test-1", 3))
	assert.NoError(t, sd.DeployPod("test-2", 3))

	conn, err := dbus.New()
	assert.NoError(t, err)
	defer conn.Close()

	unitFile := &allocation.UnitFile{
		Path: "/run/systemd/system/test-1-0.service",
	}
	assert.NoError(t, unitFile.Read())

	t.Run("delete", func(t *testing.T) {
		assert.NoError(t, scheduler.NewCommandInstruction(0, unitFile, "stop").Execute(conn))
		assert.NoError(t, scheduler.NewDisableUnitInstruction(unitFile).Execute(conn))
		assert.NoError(t, scheduler.NewDeleteUnitInstruction(unitFile).Execute(conn))
		_, err := os.Stat(unitFile.Path)
		assert.Error(t, err)
	})
	t.Run("write", func(t *testing.T) {
		assert.NoError(t, scheduler.NewWriteUnitInstruction(unitFile).Execute(conn))
		_, err := os.Stat(unitFile.Path)
		assert.NoError(t, err)
		assert.NoError(t, scheduler.NewCommandInstruction(0, unitFile, "start").Execute(conn))
	})

}

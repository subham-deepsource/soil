package allocation_test

import (
	"github.com/akaspin/soil/agent"
	"github.com/akaspin/soil/agent/scheduler/allocation"
	"github.com/akaspin/soil/manifest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewFromManifest(t *testing.T) {
	m := &manifest.Pod{
		Runtime: true,
		Name: "pod-1",
		Target: "multi-user.target",
		Units: []*manifest.Unit{
			{
				Name: "unit-1.service",
				Source: `# ${meta.consul}`,
				Transition: manifest.Transition{
					Create: "start",
					Destroy: "stop",
				},
			},
			{
				Name: "unit-2.service",
				Source: `# ${meta.consul}`,
				Transition: manifest.Transition{
					Create: "start",
					Destroy: "stop",
				},
			},
		},
	}
	e := agent.NewEnvironment(map[string]string{
		"meta.consul": "true",
		"agent.pod.exec": "ExecStart=/usr/bin/sleep inf",
	})

	res, err := allocation.NewFromManifest("private", m, e)
	assert.NoError(t, err)
	assert.Equal(t, &allocation.Pod{
		PodHeader: &allocation.PodHeader{
			Name: "pod-1",
			PodMark: 12378424092920473463,
			AgentMark: 10913175000834197307,
			Namespace: "private",
		},
		File: &allocation.File{
			Path: "/run/systemd/system/pod-private-pod-1.service",
			Source: "### POD pod-1 {\"AgentMark\":10913175000834197307,\"Namespace\":\"private\",\"PodMark\":12378424092920473463}\n### UNIT /run/systemd/system/unit-1.service {\"Create\":\"start\",\"Update\":\"\",\"Destroy\":\"stop\",\"Permanent\":false}\n### UNIT /run/systemd/system/unit-2.service {\"Create\":\"start\",\"Update\":\"\",\"Destroy\":\"stop\",\"Permanent\":false}\n\n[Unit]\nDescription=pod-1\nBefore=unit-1.service unit-2.service\n[Service]\nExecStart=/usr/bin/sleep inf\n[Install]\nWantedBy=multi-user.target\n",
		},
		Units: []*allocation.Unit{
			{
				UnitHeader: &allocation.UnitHeader{
					Permanent: false,
					Transition: manifest.Transition{
						Create: "start",
						Destroy: "stop",
					},
				},
				File: &allocation.File{
					Path: "/run/systemd/system/unit-1.service",
					Source: "# true",
				},
			},
			{
				UnitHeader: &allocation.UnitHeader{
					Permanent: false,
					Transition: manifest.Transition{
						Create: "start",
						Destroy: "stop",
					},
				},
				File: &allocation.File{
					Path: "/run/systemd/system/unit-2.service",
					Source: "# true",
				},
			},
		},
	}, res)
}

func TestHeader_Unmarshal(t *testing.T) {
	src := `### POD pod-1 {"AgentMark":123,"Namespace":"private","PodMark":345}
### UNIT /etc/systemd/system/unit-1.service {"Create":"start","Update":"","Destroy":"","Permanent":true}
### UNIT /etc/systemd/system/unit-2.service {"Create":"","Update":"start","Destroy":"","Permanent":false}
[Unit]
`
	header := &allocation.PodHeader{}
	units, err := header.Unmarshal(src)
	assert.NoError(t, err)
	assert.Equal(t, []*allocation.Unit{
		{
			File: &allocation.File{
				Path: "/etc/systemd/system/unit-1.service",
			},
			UnitHeader: &allocation.UnitHeader{
				Permanent: true,
				Transition: manifest.Transition{
					Create: "start",
				},
			},
		},
		{
			File: &allocation.File{
				Path: "/etc/systemd/system/unit-2.service",
			},
			UnitHeader: &allocation.UnitHeader{
				Permanent: false,
				Transition: manifest.Transition{
					Update: "start",
				},
			},
		},
	}, units)
	assert.Equal(t, &allocation.PodHeader{
		Name:      "pod-1",
		AgentMark: 123,
		PodMark:   345,
		Namespace: "private",
	}, header)
}

func TestHeader_Marshal(t *testing.T) {
	units := []*allocation.Unit{
		{
			UnitHeader: &allocation.UnitHeader{
				Permanent: true,
				Transition: manifest.Transition{
					Create: "start",
				},
			},
			File: &allocation.File{
				Path: "/etc/systemd/system/unit-1.service",
			},
		},
	}
	h := &allocation.PodHeader{
		Namespace: "private",
		AgentMark: 234,
		PodMark:   123,
	}
	res, err := h.Marshal("pod-1", units)
	assert.NoError(t, err)
	assert.Equal(t, "### POD pod-1 {\"AgentMark\":234,\"Namespace\":\"private\",\"PodMark\":123}\n### UNIT /etc/systemd/system/unit-1.service {\"Create\":\"start\",\"Update\":\"\",\"Destroy\":\"\",\"Permanent\":true}\n", res)
}



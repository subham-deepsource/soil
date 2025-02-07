package fixture

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	uuid "github.com/nu7hatch/gouuid"
)

type ConsulServerConfig struct {
	RepoTag       string `json:"-"`
	NodeName      string `json:"node_name"`
	NodeID        string `json:"node_id"`
	AdvertiseAddr string `json:"advertise_addr"`
	ClientAddr    string `json:"client_addr"`
	Bootstrap     bool   `json:"bootstrap"`
	Server        bool   `json:"server"`
	UI            bool   `json:"ui"`
	Performance   struct {
		RaftMultiplier int `json:"raft_multiplier"`
	} `json:"performance"`
	SessionTTLMin string `json:"session_ttl_min"`
	Ports         struct {
		HTTP int
	}
}

type ConsulServer struct {
	ctx    context.Context
	cancel context.CancelFunc

	t           *testing.T
	addr        string
	Config      *ConsulServerConfig
	wd          string
	dockerCli   *client.Client
	containerID string
}

func NewConsulServer(t *testing.T, configFn func(config *ConsulServerConfig)) (s *ConsulServer) {
	t.Helper()
	wd := "/tmp"
	wd = filepath.Join(wd, fmt.Sprintf(".test_consul_%s", TestName(t)))
	id, _ := uuid.NewV5(uuid.NamespaceDNS, []byte(wd))
	ports := RandomPorts(t, 3)
	s = &ConsulServer{
		t:    t,
		wd:   wd,
		addr: GetLocalIP(t),
		Config: &ConsulServerConfig{
			RepoTag:    "docker.io/library/consul:1.0.0",
			NodeName:   TestName(t),
			NodeID:     id.String(),
			ClientAddr: "0.0.0.0",
			Bootstrap:  true,
			Server:     true,
			UI:         true,
			Performance: struct {
				RaftMultiplier int `json:"raft_multiplier"`
			}{RaftMultiplier: 1},
			SessionTTLMin: ".5s",
			Ports: struct {
				HTTP int
			}{
				HTTP: ports[0],
			},
		},
	}
	if configFn != nil {
		configFn(s.Config)
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	// wait for cli
	WaitNoErrorT(t, WaitConfig{
		Retry:   time.Millisecond * 500,
		Retries: 100,
	}, func() (err error) {
		s.dockerCli, err = client.NewEnvClient()
		return
	})
	s.cleanupContainer()
	os.RemoveAll(s.wd)
	return
}

func (s *ConsulServer) Address() (res string) {
	res = fmt.Sprintf("%s:%d", s.addr, s.Config.Ports.HTTP)
	return
}

func (s *ConsulServer) Up() {
	s.t.Helper()
	s.cleanupContainer()
	s.writeConfig()
	resp, err := s.dockerCli.ImagePull(s.ctx, s.Config.RepoTag, types.ImagePullOptions{})
	if err != nil {
		s.t.Error(err)
		s.t.Fail()
		return
	}
	_, err = ioutil.ReadAll(resp)
	if err != nil {
		s.t.Error(err)
		s.t.Fail()
	}
	resp.Close()
	res, err := s.dockerCli.ContainerCreate(s.ctx,
		&container.Config{
			Image: s.Config.RepoTag,
			Cmd: []string{
				"agent",
				"-config-file", "/opt/config/consul.json",
			},
			ExposedPorts: nat.PortSet{
				nat.Port(fmt.Sprintf("%d/tcp", s.Config.Ports.HTTP)): struct{}{},
			},
			AttachStderr: true,
			AttachStdout: true,
		},
		&container.HostConfig{
			AutoRemove: true,
			PortBindings: nat.PortMap{
				nat.Port(fmt.Sprintf("%d/tcp", s.Config.Ports.HTTP)): []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: fmt.Sprintf("%d", s.Config.Ports.HTTP),
					},
				},
			},
			Binds: []string{
				fmt.Sprintf("%s/config:/opt/config", s.wd),
				fmt.Sprintf("%s/consul:/consul/data", s.wd),
			},
		},
		nil,
		nil,
		TestName(s.t))
	if err != nil {
		s.t.Error(err)
		s.t.FailNow()
		return
	}
	s.containerID = res.ID
	err = s.dockerCli.ContainerStart(s.ctx, s.containerID, types.ContainerStartOptions{})
	if err != nil {
		s.t.Error(err)
		s.t.FailNow()
		return
	}
	s.t.Logf(`started: %s on %s`, TestName(s.t), s.Address())
}

func (s *ConsulServer) Pause() {
	s.t.Helper()
	ctx := context.Background()
	filterBy := filters.NewArgs(filters.Arg("name", TestName(s.t)))

	list, err := s.dockerCli.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filterBy,
	})
	if err != nil {
		s.t.Error(err)
		s.t.FailNow()
	}
	for _, orphan := range list {
		s.dockerCli.ContainerPause(ctx, orphan.ID)
		s.t.Logf(`paused container: %s`, orphan.ID)
	}
}

func (s *ConsulServer) Unpause() {
	s.t.Helper()
	ctx := context.Background()
	filterBy := filters.NewArgs(filters.Arg("name", TestName(s.t)))
	list, err := s.dockerCli.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filterBy,
	})
	if err != nil {
		s.t.Error(err)
		s.t.FailNow()
	}
	for _, orphan := range list {
		s.dockerCli.ContainerUnpause(ctx, orphan.ID)
		s.t.Logf(`resumed container: %s`, orphan.ID)
	}
}

func (s *ConsulServer) cleanupContainer() {
	s.t.Helper()
	ctx := context.Background()
	filterBy := filters.NewArgs(filters.Arg("name", TestName(s.t)))

	list, err := s.dockerCli.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filterBy,
	})
	if err != nil {
		s.t.Error(err)
		s.t.Fail()
	}
	for _, orphan := range list {
		s.dockerCli.ContainerStop(ctx, orphan.ID, nil)
		s.dockerCli.ContainerWait(ctx, orphan.ID, container.WaitConditionNotRunning)
		s.dockerCli.ContainerRemove(ctx, orphan.ID, types.ContainerRemoveOptions{
			Force: true,
		})
		s.t.Logf(`removed container: %s`, orphan.ID)
	}
}

func (s *ConsulServer) Down() {
	s.t.Helper()
	s.cleanupContainer()
}

func (s *ConsulServer) Clean() {
	s.t.Helper()
	s.cleanupContainer()
	os.RemoveAll(s.wd)
	s.cancel()
}

func (s *ConsulServer) WaitAlive() {
	s.t.Helper()

	WaitNoErrorT(s.t, WaitConfig{
		Retry:   time.Millisecond * 500,
		Retries: 100,
	}, func() (err error) {
		resp, err := http.Get(fmt.Sprintf("http://%s/v1/agent/self", s.Address()))
		if err != nil {
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			err = fmt.Errorf("bad status code: %d", resp.StatusCode)
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		if string(body) == "" {
			err = fmt.Errorf(`empty api`)
		}
		return
	})
	s.t.Logf(`consul %s is alive`, s.Address())
}

func (s *ConsulServer) WaitLeader() {
	s.t.Helper()

	var index int64
	WaitNoErrorT(s.t, WaitConfig{
		Retry:   time.Millisecond * 500,
		Retries: 100,
	}, func() (err error) {
		resp, err := http.Get(fmt.Sprintf("http://%s/v1/catalog/nodes?index=%d", s.Address(), index))
		if err != nil {
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			err = fmt.Errorf("bad status code: %d", resp.StatusCode)
			return
		}

		// Ensure we have a leader and a node registration.
		if leader := resp.Header.Get("X-Consul-KnownLeader"); leader != "true" {
			err = fmt.Errorf("consul leader status: %#v", leader)
			return
		}
		index, err = strconv.ParseInt(resp.Header.Get("X-Consul-Index"), 10, 64)
		if err != nil {
			err = fmt.Errorf("bad consul index: %v", err)
			return
		}
		if index == 0 {
			err = fmt.Errorf("consul index is 0")
			return
		}

		// Watch for the anti-entropy sync to finish.
		var v []map[string]interface{}
		dec := json.NewDecoder(resp.Body)
		if err = dec.Decode(&v); err != nil {
			return
		}
		if len(v) < 1 {
			err = fmt.Errorf("no nodes")
			return
		}
		taggedAddresses, ok := v[0]["TaggedAddresses"].(map[string]interface{})
		if !ok {
			err = fmt.Errorf("missing tagged addresses")
			return
		}
		if _, ok = taggedAddresses["lan"]; !ok {
			err = fmt.Errorf("no lan tagged addresses")
		}
		return
	})
	s.t.Logf(`consul %s leader is alive`, s.Address())
}

func (s *ConsulServer) writeConfig() {
	s.t.Helper()
	if err := os.MkdirAll(s.wd, 0777); err != nil {
		s.t.Error(err)
		s.t.FailNow()
		return
	}
	os.MkdirAll(filepath.Join(s.wd, "config"), 0777)
	f, err := os.Create(filepath.Join(s.wd, "config", "consul.json"))
	if err != nil {
		s.t.Error(err)
		s.t.FailNow()
		return
	}
	if err = json.NewEncoder(f).Encode(s.Config); err != nil {
		s.t.Error(err)
		s.t.FailNow()
		return
	}
}

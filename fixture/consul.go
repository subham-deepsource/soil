package fixture

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/eapache/go-resiliency/retrier"
	"github.com/moby/moby/client"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type ConsulServerConfig struct {
	RepoTag       string   `json:"-"`
	Join          []string `json:"retry_join"`
	NodeName      string   `json:"node_name"`
	NodeID        string   `json:"node_id"`
	AdvertiseAddr string   `json:"advertise_addr"`
	ClientAddr    string   `json:"client_addr"`
	Bootstrap     bool     `json:"bootstrap"`
	Server        bool     `json:"server"`
	UI            bool     `json:"ui"`
	Performance   struct {
		RaftMultiplier int `json:"raft_multiplier"`
	} `json:"performance"`
	SessionTTLMin string `json:"session_ttl_min"`
	Ports         struct {
		HTTP    int
		Server  int
		SerfLan int `json:"serf_lan"`
	}
}

type ConsulServer struct {
	ctx    context.Context
	cancel context.CancelFunc

	t           *testing.T
	Config      *ConsulServerConfig
	wd          string
	dockerCli   *client.Client
	containerID string
}

func NewConsulServer(t *testing.T, configFn func(config *ConsulServerConfig)) (s *ConsulServer) {
	t.Helper()
	wd, _ := os.Getwd()
	wd = filepath.Join(wd, "testdata", fmt.Sprintf(".test_%s", TestName(t)))
	id, _ := uuid.NewV5(uuid.NamespaceDNS, []byte(wd))
	ip := GetLocalIP(t)
	s = &ConsulServer{
		t:  t,
		wd: wd,
		Config: &ConsulServerConfig{
			RepoTag:       "docker.io/library/consul",
			NodeName:      TestName(t),
			NodeID:        id.String(),
			AdvertiseAddr: ip,
			Join:          []string{ip},
			ClientAddr:    "0.0.0.0",
			Bootstrap:     true,
			Server:        true,
			UI:            true,
			Performance: struct {
				RaftMultiplier int `json:"raft_multiplier"`
			}{RaftMultiplier: 1},
			SessionTTLMin: ".5s",
			Ports: struct {
				HTTP    int
				Server  int
				SerfLan int `json:"serf_lan"`
			}{
				HTTP:    RandomPort(t),
				Server:  RandomPort(t),
				SerfLan: RandomPort(t),
			},
		},
	}
	if configFn != nil {
		configFn(s.Config)
	}
	if err := os.MkdirAll(s.wd, 0777); err != nil {
		t.Error(err)
		t.FailNow()
		return
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	var err error
	if s.dockerCli, err = client.NewEnvClient(); err != nil {
		t.Error(err)
		t.FailNow()
		return
	}
	return
}

func (s *ConsulServer) Address() (res string) {
	res = fmt.Sprintf("%s:%d", s.Config.AdvertiseAddr, s.Config.Ports.HTTP)
	return
}

func (s *ConsulServer) Up() {
	s.t.Helper()
	s.writeConfig()
	resp, err := s.dockerCli.ImagePull(s.ctx, s.Config.RepoTag, types.ImagePullOptions{})
	if err != nil {
		s.t.Error(err)
		s.t.FailNow()
		return
	}
	ioutil.ReadAll(resp)
	resp.Close()

	res, err := s.dockerCli.ContainerCreate(s.ctx,
		&container.Config{
			Image: s.Config.RepoTag,
			Cmd: []string{
				"agent",
				"-config-file", "/opt/config/consul.json",
			},
			ExposedPorts: nat.PortSet{
				nat.Port(fmt.Sprintf("%d/tcp", s.Config.Ports.HTTP)):    struct{}{},
				nat.Port(fmt.Sprintf("%d/tcp", s.Config.Ports.Server)):  struct{}{},
				nat.Port(fmt.Sprintf("%d/tcp", s.Config.Ports.SerfLan)): struct{}{},
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
				nat.Port(fmt.Sprintf("%d/tcp", s.Config.Ports.Server)): []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: fmt.Sprintf("%d", s.Config.Ports.Server),
					},
				},
				nat.Port(fmt.Sprintf("%d/tcp", s.Config.Ports.SerfLan)): []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: fmt.Sprintf("%d", s.Config.Ports.SerfLan),
					},
				},
			},
			Binds: []string{
				fmt.Sprintf("%s/config:/opt/config", s.wd),
				fmt.Sprintf("%s/consul:/consul/data", s.wd),
			},
		},
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
}

func (s *ConsulServer) Down() {
	s.t.Helper()
	s.dockerCli.ContainerStop(context.Background(), s.containerID, nil)
	s.dockerCli.ContainerWait(context.Background(), s.containerID)
	s.dockerCli.ContainerRemove(context.Background(), s.containerID, types.ContainerRemoveOptions{
		Force: true,
	})
}

func (s *ConsulServer) Clean() {
	s.t.Helper()
	s.Down()
	os.RemoveAll(s.wd)
	s.cancel()
}

func (s *ConsulServer) WaitAlive() {
	s.t.Helper()
	rtr := retrier.New(retrier.ConstantBackoff(100, time.Millisecond*50), retrier.DefaultClassifier{})
	err := rtr.Run(func() (err error) {
		resp, err := http.Get(fmt.Sprintf("http://%s/v1/agent/self", s.Address()))
		if err != nil {
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		if string(body) == "" {
			err = fmt.Errorf(`empty api`)
		}
		return
	})
	if err != nil {
		s.t.Error(err)
		s.t.FailNow()
		return
	}
}

func (s *ConsulServer) writeConfig() {
	s.t.Helper()
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

func (s *ConsulServer) waitForLeader() {

}

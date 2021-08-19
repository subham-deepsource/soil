package main

import (
	"github.com/akaspin/cut"
	agent "github.com/da-moon/soil/cmd/soil/agent"
	version "github.com/da-moon/soil/cmd/soil/version"
	"github.com/spf13/cobra"
	"io"
)

type Soil struct {
	*cut.Environment
}

func (c *Soil) Bind(cc *cobra.Command) {
	cc.Use = "soil"
}

func run(stderr, stdout io.Writer, stdin io.Reader, args ...string) (err error) {
	env := &cut.Environment{
		Stderr: stderr,
		Stdin:  stdin,
		Stdout: stdout,
	}
	configs := &agent.AgentOptions{}

	cmd := cut.Attach(
		&Soil{env}, []cut.Binder{env},
		cut.Attach(
			&agent.Agent{
				Environment:  env,
				AgentOptions: configs,
			}, []cut.Binder{configs},
		),
		cut.Attach(
			&version.Version{env}, nil,
		),
	)
	cmd.SetArgs(args)
	cmd.SetOutput(stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err = cmd.Execute()
	return
}

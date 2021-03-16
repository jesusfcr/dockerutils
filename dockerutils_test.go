/*
Copyright 2019 Adevinta
*/

package dockerutils

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
)

const (
	registryServer = "containers.mpi-internal.com"
	testImage      = "containers.mpi-internal.com/alpine:3.4"
)

func TestDockerIntegration(t *testing.T) {
	checks := []struct {
		command         []string
		wantStdout      []byte
		wantStderr      []byte
		createThenStart bool
	}{
		{
			command:    []string{"sh", "-c", "echo stdout works"},
			wantStdout: []byte("stdout works\n"),
			wantStderr: nil,
		},
		{
			command:         []string{"sh", "-c", "echo stdout works in two steps"},
			wantStdout:      []byte("stdout works in two steps\n"),
			wantStderr:      nil,
			createThenStart: true,
		},
		{
			command:    []string{"sh", "-c", "echo stderr works >&2"},
			wantStdout: nil,
			wantStderr: []byte("stderr works\n"),
		},
	}

	envCli, err := client.NewEnvClient()
	if err != nil {
		t.Fatalf("NewEnvClient error: %v", err)
	}

	ctx := context.Background()

	dc := NewClient(envCli)

	for _, check := range checks {
		if err := dc.Login(
			ctx,
			registryServer,
			os.Getenv("ARTIFACTORY_USER"),
			os.Getenv("ARTIFACTORY_PASSWORD"),
		); err != nil {
			t.Fatalf("Login error: %v", err)
		}
		if err := dc.Pull(ctx, testImage); err != nil {
			t.Fatalf("Pull error: %v", err)
		}

		runConfig := RunConfig{
			ContainerConfig: &container.Config{
				Image: testImage,
				Cmd:   strslice.StrSlice(check.command),
			},
			HostConfig:            &container.HostConfig{},
			NetConfig:             &network.NetworkingConfig{},
			ContainerStartOptions: types.ContainerStartOptions{},
		}

		var contID string
		var err error
		if check.createThenStart {
			contID, err = dc.Create(ctx, runConfig, "")
			if err != nil {
				t.Fatalf("Create error: %v", err)
			}

			if err := dc.RunExisting(ctx, runConfig, contID); err != nil {
				t.Fatalf("RunExisting error: %v", err)
			}
		} else {
			contID, err = dc.Run(ctx, runConfig, "")
			if err != nil {
				t.Fatalf("Run error: %v", err)
			}
		}

		stdout, stderr, err := dc.Logs(ctx, contID, true)
		if bytes.Compare(stdout, check.wantStdout) != 0 {
			t.Errorf("cmd=%q stdout=%q want=%q",
				check.command, stdout, check.wantStdout)
		}
		if bytes.Compare(stderr, check.wantStderr) != 0 {
			t.Errorf("cmd=%q stderr=%q want=%q",
				check.command, stderr, check.wantStderr)
		}

		removeConfig := types.ContainerRemoveOptions{}
		if err := dc.ContainerRemove(ctx, contID, removeConfig); err != nil {
			t.Fatalf("ContainerRemove: %v", err)
		}
	}
}

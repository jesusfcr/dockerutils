/*
Copyright 2019 Adevinta
*/

package dockerutils

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Client struct {
	*client.Client
	authConfig types.AuthConfig
}

type RunConfig struct {
	ContainerConfig       *container.Config
	HostConfig            *container.HostConfig
	NetConfig             *network.NetworkingConfig
	ContainerStartOptions types.ContainerStartOptions
}

type LogsOutput struct {
	Stdout []byte
	Stderr []byte
}

// NewClient creates a new Client instance.
func NewClient(cli *client.Client) *Client {
	return &Client{Client: cli}
}

// Login authenticates against the docker registry.
func (c *Client) Login(ctx context.Context, server, username, password string) error {
	cfg := types.AuthConfig{
		Username:      username,
		Password:      password,
		ServerAddress: server,
	}
	_, err := c.RegistryLogin(ctx, cfg)
	if err != nil {
		return err
	}
	c.authConfig = cfg
	return nil
}

// Pull pulls a container from docker registry.
func (c *Client) Pull(ctx context.Context, imageRef string) error {
	buf, err := json.Marshal(c.authConfig)
	if err != nil {
		return err
	}
	encodedAuth := base64.URLEncoding.EncodeToString(buf)

	pullOpts := types.ImagePullOptions{
		RegistryAuth: encodedAuth,
	}

	respBody, err := c.ImagePull(ctx, imageRef, pullOpts)
	if err != nil {
		return err
	}
	defer respBody.Close()

	if _, err := io.Copy(ioutil.Discard, respBody); err != nil {
		return err
	}

	return nil
}

// Run creates and executes a container.
func (c *Client) Run(ctx context.Context, cfg RunConfig, name string) (contID string, err error) {
	containerResp, err := c.ContainerCreate(
		ctx, cfg.ContainerConfig, cfg.HostConfig, cfg.NetConfig, name,
	)
	if err != nil {
		return "", err
	}

	if err := c.ContainerStart(
		ctx, containerResp.ID, cfg.ContainerStartOptions,
	); err != nil {
		return "", err
	}

	return containerResp.ID, nil
}

// RunExisting starts executing an already existing container.
func (c *Client) RunExisting(ctx context.Context, cfg RunConfig, containerID string) (err error) {
	return c.ContainerStart(
		ctx, containerID, cfg.ContainerStartOptions,
	)
}

// Create creates a new container, but not starts executing it.
func (c *Client) Create(ctx context.Context, cfg RunConfig, name string) (contID string, err error) {
	containerResp, err := c.ContainerCreate(
		ctx, cfg.ContainerConfig, cfg.HostConfig, cfg.NetConfig, name,
	)
	if err != nil {
		return "", err
	}
	return containerResp.ID, nil
}

// Logs gets the logs of a container.
func (c *Client) Logs(ctx context.Context, contID string, follow bool) (stdout, stderr []byte, err error) {
	bout, berr := &bytes.Buffer{}, &bytes.Buffer{}

	opts := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
	}

	logs, err := c.ContainerLogs(ctx, contID, opts)
	if err != nil {
		return nil, nil, err
	}
	defer logs.Close()

	_, err = stdcopy.StdCopy(bout, berr, logs)
	if err != nil {
		return nil, nil, err
	}

	return bout.Bytes(), berr.Bytes(), nil
}

/*
Copyright (C) 2018 Expedia Group.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package docker

import (
	"errors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"io"
	"github.com/HotelsDotCom/go-logger"
	"testing"
	"time"
)

func TestNewClientShouldCreateDockerClientAndContext(t *testing.T) {
	nd, err := NewDocker()
	require.NoError(t, err)
	client := nd.(*docker)
	assert.NotNil(t, client.cli)
	assert.NotNil(t, client.ctx)
}

func TestNewClientShouldReturnErrorWhenUnableToCreateNewClient(t *testing.T) {
	of := newDockerClient
	defer func() { newDockerClient = of }()

	newDockerClient = func() (dockerClient, error) {
		return nil, errors.New("an error")
	}

	_, err := NewDocker()

	require.EqualError(t, err, "an error")
}

func TestRunShouldReturnContainerAndNoError(t *testing.T) {
	of := newDockerClient
	defer func() { newDockerClient = of }()

	newDockerClient = func() (dockerClient, error) {
		return &mockDockerClient{}, nil
	}

	c, err := NewDocker()
	require.NoError(t, err)
	require.NotNil(t, c)

	container, err := c.Run("name", "path", nil, nil)

	require.NoError(t, err)
	require.NotNil(t, container)

	err = container.StopAndRemove()
	require.NoError(t, err)
}

func TestRunShouldPullRequestedImageWhenNeeded(t *testing.T) {
	calledImagePull := false
	imagePuller := func(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error) {
		calledImagePull = true
		return &mockReadCloser{}, nil
	}

	mdc := &mockDockerClient{imagePuller: imagePuller}
	c := &docker{cli: mdc}

	c.Run("name", "path", nil, nil)

	assert.True(t, calledImagePull)
}

func TestRunShouldReturnErrorWhenHasImageFails(t *testing.T) {

	mdc := &mockDockerClient{imageLister: func(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error) {
		return []types.ImageSummary{}, errors.New("the error")
	}}
	c := &docker{cli: mdc}

	_, err := c.Run("name", "path", nil, nil)

	assert.EqualError(t, err, "the error")
}

func TestShouldFilterImageListToRequiredImageName(t *testing.T) {
	calledImageList := false
	imageLister := func(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error) {
		calledImageList = true
		assert.True(t, options.Filters.Len() > 0, "should have at least one filter")
		assert.True(t, options.Filters.ExactMatch("reference", "imagePath"), "should have a reference filter for the image path")
		return []types.ImageSummary{}, nil
	}

	mdc := &mockDockerClient{imageLister: imageLister}
	c := &docker{cli: mdc}

	c.Run("containerName", "imagePath", nil, nil)

	assert.True(t, calledImageList)
}

func TestShouldCloseReadCloserWhenPullingRegistryImage(t *testing.T) {
	closeCalled := false

	closer := &mockReadCloser{closer: func() error {
		closeCalled = true
		return nil
	}}

	imagePuller := func(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error) {
		return closer, nil
	}

	mdc := &mockDockerClient{imagePuller: imagePuller}
	c := &docker{cli: mdc}

	c.Run("foo", "externalImage", nil, nil)

	assert.True(t, closeCalled, "close should have been called when pulling the image")
}

func TestImagePullShouldReturnErrorWhenPullOfExternalImageFails(t *testing.T) {

	imagePuller := func(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error) {
		return &mockReadCloser{}, errors.New("imagePull Failed")
	}

	mdc := &mockDockerClient{imagePuller: imagePuller}
	c := &docker{cli: mdc}

	_, err := c.Run("mongo", "mongo", nil, nil)

	assert.EqualError(t, err, "imagePull Failed")
}

func TestImagePullShouldLogErrorWhenPullOfExternalImageFails(t *testing.T) {

	imagePuller := func(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error) {
		return &mockReadCloser{}, errors.New("imagePull Failed")
	}

	mdc := &mockDockerClient{imagePuller: imagePuller}
	c := &docker{cli: mdc}

	loggerCalled := false
	defer func(ol func(m string, args ...interface{})) { logger.Errorf = ol }(logger.Errorf)

	logger.Errorf = func(msg string, args ...interface{}) {
		loggerCalled = true
		require.Equal(t, "unable to pull image: %s", msg)
		require.EqualError(t, args[0].(error), "imagePull Failed")
	}

	c.Run("name", "imagePath", nil, nil)

	assert.True(t, loggerCalled, "logger should be called when image pull fails")
}

func TestContainerCreateShouldCreateContainerWhenCalled(t *testing.T) {

	calledContainerCreate := false

	containerCreate := func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
		calledContainerCreate = true
		require.True(t, config.Image == "imagePath", "should be imagePath")
		require.Equal(t, config.Env, []string{"BANANA=YELLOW"})
		require.Equal(t, config.ExposedPorts, nat.PortSet{nat.Port("8080/tcp"): {}})
		return container.ContainerCreateCreatedBody{}, nil
	}

	mdc := &mockDockerClient{containerCreate: containerCreate}

	c := &docker{cli: mdc}

	c.Run("name", "imagePath", []string{"BANANA=YELLOW"}, []string{"8080/tcp"})

	assert.True(t, calledContainerCreate, "containerCreate Should be Called")

}

func TestCreateContainerShouldReturnErrorWhenContainerCreateFails(t *testing.T) {

	containerCreate := func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
		return container.ContainerCreateCreatedBody{}, errors.New("containerCreate Failed")
	}

	mdc := &mockDockerClient{containerCreate: containerCreate}

	c := &docker{cli: mdc}

	_, err := c.Run("name", "imagePath", nil, nil)

	assert.EqualError(t, err, "containerCreate Failed")

}

func TestCreateContainerShouldHaveEnvironmentVariablesSet(t *testing.T) {

	containerCreate := func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
		require.Equal(t, config.Env, []string{"BANANA=YELLOW"})
		return container.ContainerCreateCreatedBody{}, nil
	}

	var usedContainerId string
	containerStart := func(ctx context.Context, containerID string, options types.ContainerStartOptions) error {
		usedContainerId = containerID
		return nil
	}

	containerInspect := func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
		return types.ContainerJSON{
			Config: &container.Config{
				Env: []string{"BANANA=YELLOW"},
			},
		}, nil
	}

	mdc := &mockDockerClient{containerCreate: containerCreate, containerStart: containerStart, containerInspect: containerInspect}
	c := &docker{cli: mdc}

	_, err := c.Run("name", "imagePath", []string{"BANANA=YELLOW"}, []string{"8080/tcp"})
	assert.NoError(t, err)

	resp, err := c.cli.ContainerInspect(c.ctx, usedContainerId)
	assert.NoError(t, err)

	assert.NotNil(t, resp.Config.Env)
	require.Contains(t, resp.Config.Env, "BANANA=YELLOW")
}

func TestRunShouldStartAndReturnContainerWithCorrectContainerID(t *testing.T) {

	containerCreate := func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
		return container.ContainerCreateCreatedBody{ID: "aContainerId"}, nil
	}

	didCallContainerStart := false
	containerStart := func(ctx context.Context, containerID string, options types.ContainerStartOptions) error {
		didCallContainerStart = true
		require.Equal(t, "aContainerId", containerID)
		return nil
	}

	mdc := &mockDockerClient{containerCreate: containerCreate, containerStart: containerStart}

	c := &docker{cli: mdc}

	container, err := c.Run("name", "imagePath", nil, nil)

	assert.True(t, didCallContainerStart, "should have called dockerContainer start")
	assert.NotNil(t, container)
	assert.NoError(t, err)

	assert.IsType(t, &dockerContainer{}, container)
	oc := container.(*dockerContainer)
	assert.Equal(t, oc.id, "aContainerId")

	err = container.StopAndRemove()
	require.NoError(t, err)
}

func TestStartContainerShouldReturnErrorWhenFails(t *testing.T) {

	containerStart := func(ctx context.Context, containerID string, options types.ContainerStartOptions) error {
		return errors.New("the error")
	}

	mdc := &mockDockerClient{containerStart: containerStart}

	c := &docker{cli: mdc}

	_, err := c.Run("name", "imagePath", nil, nil)

	assert.EqualError(t, err, "the error")
}

type mockReadCloser struct {
	closer func() error
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (m *mockReadCloser) Close() error {
	if m.closer != nil {
		return m.closer()
	}
	return nil
}

type mockDockerClient struct {
	imageLister      func(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error)
	imagePuller      func(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error)
	containerCreate  func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error)
	containerStart   func(ctx context.Context, containerID string, options types.ContainerStartOptions) error
	containerInspect func(ctx context.Context, containerID string) (types.ContainerJSON, error)
	containerStop    func(ctx context.Context, containerID string, timeout *time.Duration) error
	containerRemove  func(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error
}

func (m *mockDockerClient) ImageList(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error) {
	if m.imageLister != nil {
		return m.imageLister(ctx, options)
	}
	return []types.ImageSummary{}, nil
}

func (m *mockDockerClient) ImagePull(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error) {
	if m.imagePuller != nil {
		return m.imagePuller(ctx, refStr, options)
	}
	return &mockReadCloser{}, nil
}

func (m *mockDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
	if m.containerCreate != nil {
		return m.containerCreate(ctx, config, hostConfig, networkingConfig, containerName)
	}
	return container.ContainerCreateCreatedBody{}, nil
}

func (m *mockDockerClient) ContainerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error {
	if m.containerStart != nil {
		return m.containerStart(ctx, containerID, options)
	}
	return nil
}

func (m *mockDockerClient) ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error {
	if m.containerStop != nil {
		return m.containerStop(ctx, containerID, nil)
	}
	return nil
}

func (m *mockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	if m.containerInspect != nil {
		return m.containerInspect(ctx, containerID)
	}
	return types.ContainerJSON{}, nil
}

func (m *mockDockerClient) ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
	if m.containerRemove != nil {
		return m.containerRemove(ctx, containerID, options)
	}
	return nil
}

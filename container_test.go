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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"testing"
	"time"
)

func TestStopAndRemoveShouldBeCalledAndReturnAnError(t *testing.T) {
	iCalledStop := false
	containerStopper := func(ctx context.Context, containerID string, timeout *time.Duration) error {
		iCalledStop = true
		return errors.New("the error")
	}

	mdc := &mockDockerClient{containerStop: containerStopper}

	c := &docker{cli: mdc}
	dockerContainer, _ := c.Run("name", "path", nil, nil)

	err := dockerContainer.StopAndRemove()

	assert.True(t, iCalledStop)
	assert.EqualError(t, err, "the error")
}

func TestStopAndRemoveShouldBeCalledWithASpecificContainerId(t *testing.T) {
	containerCreate := func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
		return container.ContainerCreateCreatedBody{ID: "aContainerId"}, nil
	}

	var usedContainerId string

	containerStopper := func(ctx context.Context, containerID string, timeout *time.Duration) error {
		usedContainerId = containerID
		return nil
	}

	mdc := &mockDockerClient{containerCreate: containerCreate, containerStop: containerStopper}

	c := &docker{cli: mdc}
	dockerContainer, _ := c.Run("name", "path", nil, nil)

	err := dockerContainer.StopAndRemove()
	require.Equal(t, "aContainerId", usedContainerId)

	assert.NoError(t, err)
}

func TestGetIPShouldBeCalledAndPassAnErrorOnFailure(t *testing.T) {
	iCalledGetIp := false
	containerInspect := func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
		iCalledGetIp = true
		return types.ContainerJSON{}, errors.New("the error")
	}

	mdc := &mockDockerClient{containerInspect: containerInspect}

	c := &docker{cli: mdc}
	dockerContainer, _ := c.Run("name", "path", nil, nil)
	defer dockerContainer.StopAndRemove()

	_, err := dockerContainer.GetIP()

	assert.True(t, iCalledGetIp)
	assert.EqualError(t, err, "the error")
}

func TestGetIPShouldBeCalledWithASpecificContainerId(t *testing.T) {
	containerCreate := func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
		return container.ContainerCreateCreatedBody{ID: "aContainerId"}, nil
	}

	var usedContainerId string

	containerInspect := func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
		usedContainerId = containerID
		return types.ContainerJSON{}, errors.New("the error")
	}

	mdc := &mockDockerClient{containerInspect: containerInspect, containerCreate: containerCreate}

	c := &docker{cli: mdc}
	dockerContainer, _ := c.Run("name", "path", nil, nil)
	defer dockerContainer.StopAndRemove()

	dockerContainer.GetIP()

	require.Equal(t, "aContainerId", usedContainerId)
}

func TestGetIPShouldReturnAnIPAddress(t *testing.T) {

	containerCreate := func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
		return container.ContainerCreateCreatedBody{ID: "aContainerId"}, nil
	}

	containerInspect := func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
		return types.ContainerJSON{
			NetworkSettings: &types.NetworkSettings{
				DefaultNetworkSettings: types.DefaultNetworkSettings{
					IPAddress: "10.0.0.1",
				},
			},
		}, nil
	}

	mdc := &mockDockerClient{containerCreate: containerCreate, containerInspect: containerInspect}

	c := &docker{cli: mdc}
	dockerContainer, _ := c.Run("name", "path", nil, nil)
	defer dockerContainer.StopAndRemove()

	ip, err := dockerContainer.GetIP()

	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.1", ip)
}

func TestGetIPShouldBeEmptyIfNetworkSettingsIsNil(t *testing.T) {
	containerCreate := func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
		return container.ContainerCreateCreatedBody{ID: "aContainerId"}, nil
	}

	containerInspect := func(ctx context.Context, containerID string) (types.ContainerJSON, error) {
		return types.ContainerJSON{
			NetworkSettings: nil,
		}, nil
	}

	mdc := &mockDockerClient{containerCreate: containerCreate, containerInspect: containerInspect}

	c := &docker{cli: mdc}
	dockerContainer, _ := c.Run("name", "path", nil, nil)
	defer dockerContainer.StopAndRemove()

	ip, err := dockerContainer.GetIP()

	assert.NoError(t, err)
	assert.Empty(t, ip)
}

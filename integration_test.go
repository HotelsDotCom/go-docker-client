// +build integration

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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log"
	"os/exec"
	"regexp"
	"testing"
)

func TestContainer(t *testing.T) {
	d, err := NewDocker()
	require.NoError(t, err)
	require.NotNil(t, d)

	c, err := d.Run("mongo", "mongo", []string{"BANANA=YELLOW"}, []string{"27017:27017"})

	require.NoError(t, err)
	require.NotNil(t, c)

	ip, err := c.GetIP()
	require.NoError(t, err)

	assert.Regexp(t, regexp.MustCompile("^(\\d+\\.){3}\\d+$"), ip)

	inspected := inspect("mongo")
	require.Contains(t, inspected[0].Config.Env, "BANANA=YELLOW")

	err = c.StopAndRemove()
	assert.NoError(t, err)
}

func inspect(container string) response {
	cmd := exec.Command("docker", "inspect", container)

	var out bytes.Buffer
	cmd.Stdout = &out

	var response response

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	err = json.NewDecoder(&out).Decode(&response)
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
	return response
}

type response []struct {
	Config struct {
		Env []string `json:"Env"`
	}

	ID string `json:"Id"`
}

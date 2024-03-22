// Copyright 2024 Humanitec
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"context"
	"crypto/rand"
	rand2 "math/rand"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type exampleTestCase struct {
	subDir   string
	adds     []string
	expected string
}

func TestExample(t *testing.T) {
	for _, tc := range []exampleTestCase{
		{
			subDir: "01-hello",
			adds:   []string{"score.yaml"},
			expected: `name: 01-hello
services:
    hello-world-hello:
        command:
            - -c
            - while true; do echo Hello World!; sleep 5; done
        entrypoint:
            - /bin/sh
        image: busybox
`,
		},
		{
			subDir: "02-environment",
			adds:   []string{"score.yaml"},
			expected: `name: 02-environment
services:
    hello-world-hello:
        command:
            - -c
            - while true; do echo $${GREETING} $${NAME}!; sleep 5; done
        entrypoint:
            - /bin/sh
        environment:
            GREETING: Hello
            NAME: ${NAME}
            WORKLOAD_NAME: hello-world
        image: busybox
`,
		},
		{
			subDir: "03-files",
			adds:   []string{"score.yaml"},
			expected: `name: 03-files
services:
    hello-world-hello:
        command:
            - -c
            - while true; do cat /fileA.txt; cat /fileB.txt; sleep 5; done
        entrypoint:
            - /bin/sh
        image: busybox
        volumes:
            - type: bind
              source: .score-compose/mounts/files/hello-world-files-0-fileA.txt
              target: /fileA.txt
            - type: bind
              source: .score-compose/mounts/files/hello-world-files-1-fileB.txt
              target: /fileB.txt
`,
		},
		{
			subDir: "04-multiple-workloads",
			adds: []string{
				"score.yaml",
				"score2.yaml",
			},
			expected: `name: 04-multiple-workloads
services:
    hello-world-2-first:
        environment:
            NGINX_PORT: "8080"
        image: nginx:latest
    hello-world-first:
        environment:
            NGINX_PORT: "8080"
        image: nginx:latest
    hello-world-second:
        environment:
            NGINX_PORT: "8081"
        image: nginx:latest
        network_mode: service:hello-world-first
`,
		},
		{
			subDir: "05-volume-mounts",
			adds:   []string{"score.yaml"},
			expected: `name: 05-volume-mounts
services:
    hello-world-first:
        image: nginx:latest
        volumes:
            - type: volume
              source: hello-world-data-2saOb4
              target: /data
volumes:
    hello-world-data-2saOb4:
        name: hello-world-data-2saOb4
        driver: local
        labels:
            dev.score.compose.res.uid: volume.default#hello-world.data
`,
		},
		{
			subDir: "06-resource-provisioning",
			adds:   []string{"score.yaml", "score2.yaml"},
		},
		{
			subDir: "07-overrides",
			adds:   []string{"score.yaml --override-property containers.web.variables.DEBUG=\"true\""},
			expected: `name: 07-overrides
services:
    hello-world-web:
        environment:
            DEBUG: "true"
        image: nginx
`,
		},
		{
			subDir: "09-dns-and-route",
			adds:   []string{"score.yaml"},
		},
	} {
		t.Run(tc.subDir, func(t *testing.T) {
			oldReader := rand.Reader
			t.Cleanup(func() {
				rand.Reader = oldReader
			})
			rand.Reader = rand2.New(rand2.NewSource(0))

			changeToDir(t, "../../examples/"+tc.subDir)
			require.NoError(t, os.RemoveAll(".score-compose"))
			require.NoError(t, os.RemoveAll("compose.yaml"))

			stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
			require.NoError(t, err)
			assert.Equal(t, "", stdout)

			for _, add := range tc.adds {
				stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, strings.Split("generate "+add, " "))
				require.NoError(t, err)
				assert.Equal(t, "", stdout)
			}

			if tc.expected != "" {
				raw, err := os.ReadFile("compose.yaml")
				require.NoError(t, err)
				assert.Equal(t, tc.expected, string(raw))
			}

			if os.Getenv("NO_DOCKER") == "" {
				dockerCmd, err := exec.LookPath("docker")
				require.NoError(t, err)
				cmd := exec.Command(dockerCmd, "compose", "-f", "compose.yaml", "convert", "--quiet", "--dry-run")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				assert.NoError(t, cmd.Run())
			}

		})
	}
}

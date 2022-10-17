package compose

import (
	"bufio"
	"bytes"
	"testing"

	compose "github.com/compose-spec/compose-go/types"
	assert "github.com/stretchr/testify/assert"
)

func TestYamlEncode(t *testing.T) {
	var tests = []struct {
		Name   string
		Source *compose.Project
		Output []byte
		Error  error
	}{
		{
			Name: "Should encode the docker-compose spec",
			Source: &compose.Project{
				Services: compose.Services{
					{
						Name:  "test",
						Image: "busybox",
						Command: compose.ShellCommand{
							"/bin/sh",
							"-c",
							"while true; echo ...sleeping 10 sec...; sleep 10; done",
						},
					},
				},
			},
			Output: []byte(`services:
  test:
    command:
      - /bin/sh
      - -c
      - while true; echo ...sleeping 10 sec...; sleep 10; done
    image: busybox
`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			buf := bytes.Buffer{}
			w := bufio.NewWriter(&buf)

			err := WriteYAML(tt.Source, w)
			w.Flush()

			if tt.Error != nil {
				// On Error
				//
				assert.ErrorContains(t, err, tt.Error.Error())
			} else {
				// On Success
				//
				assert.NoError(t, err)
				assert.Equal(t, tt.Output, buf.Bytes())
			}
		})
	}
}

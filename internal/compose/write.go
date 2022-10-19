package compose

import (
	"io"

	compose "github.com/compose-spec/compose-go/types"
	yaml "gopkg.in/yaml.v3"
)

// WriteYAML exports docker-compose specification in YAML.
func WriteYAML(w io.Writer, proj *compose.Project) error {
	var enc = yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc.Encode(proj)
}

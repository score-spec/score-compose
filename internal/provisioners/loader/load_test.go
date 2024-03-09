package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/util"
)

func TestLoadProvisioners(t *testing.T) {

	t.Run("empty", func(t *testing.T) {
		p, err := LoadProvisioners([]byte(`[]`))
		require.NoError(t, err)
		assert.Len(t, p, 0)
	})

	t.Run("nominal", func(t *testing.T) {
		p, err := LoadProvisioners([]byte(`
- uri: template://example
  type: thing
  class: default
  id: specific
  state: |
    number: {{ 1 }}
`))
		require.NoError(t, err)
		assert.Len(t, p, 1)
		assert.Equal(t, "template://example", p[0].Uri())
		assert.True(t, p[0].Match(project.NewResourceUid("w", "r", "thing", nil, util.Ref("specific"))))
	})

	t.Run("unknown schema", func(t *testing.T) {
		_, err := LoadProvisioners([]byte(`
- uri: blah://example
  type: thing
`))
		require.EqualError(t, err, "0: unsupported provisioner type 'blah'")
	})

	t.Run("missing uri", func(t *testing.T) {
		_, err := LoadProvisioners([]byte(`
- type: thing
`))
		require.EqualError(t, err, "0: missing uri schema ''")
	})

}

func TestLoadProvisionersFromDirectory(t *testing.T) {
	td := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(td, "00.p.yaml"), []byte(`
- uri: template://example-a
  type: thing
`), 0600))
	assert.NoError(t, os.WriteFile(filepath.Join(td, "01.p.yaml"), []byte(`
- uri: template://example-b
  type: thing
`), 0600))

	p, err := LoadProvisionersFromDirectory(td, ".p.yaml")
	require.NoError(t, err)
	uris := make([]string, len(p))
	for i, prv := range p {
		uris[i] = prv.Uri()
	}
	assert.Equal(t, []string{"template://example-a", "template://example-b"}, uris)
}

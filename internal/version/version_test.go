package version

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssertVersion_good(t *testing.T) {
	for _, tup := range [][2]string{
		{"=1.2.3", "v1.2.3"},
		{">=1.2.3", "v1.2.3"},
		{">=1.2.3", "v1.2.4"},
		{">1.2.3", "v1.2.4"},
		{">=1.1", "1.1.0"},
		{">=1.1", "1.2.0"},
		{">=1", "1.0.0"},
		{">1", "2.0.0"},
	} {
		t.Run(fmt.Sprintf("%v", tup), func(t *testing.T) {
			assert.NoError(t, AssertVersion(tup[0], tup[1]))
		})
	}
}

func TestAssertVersion_bad(t *testing.T) {
	for _, tup := range [][3]string{
		{"=1.2.3", "v1.2.0", "current version v1.2.0 does not match requested constraint =1.2.3"},
		{">2", "v1.2.0", "current version v1.2.0 does not match requested constraint >2"},
		{">1.2", "v1.2.0", "current version v1.2.0 does not match requested constraint >1.2"},
	} {
		t.Run(fmt.Sprintf("%v", tup), func(t *testing.T) {
			assert.EqualError(t, AssertVersion(tup[0], tup[1]), tup[2])
		})
	}
}

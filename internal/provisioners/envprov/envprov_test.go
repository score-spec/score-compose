// Copyright 2024 The Score Authors
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

package envprov

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/score-spec/score-compose/internal/provisioners"
)

func TestProvisioner(t *testing.T) {
	p := new(Provisioner)

	t.Run("test match", func(t *testing.T) {
		assert.True(t, p.Match("environment.default#w.r"))
		assert.False(t, p.Match("environment.default#thing"))
		assert.False(t, p.Match("environment.foo#w.r"))
		assert.False(t, p.Match("other.default#w.r"))
	})

	t.Run("provision", func(t *testing.T) {
		po, err := p.Provision(context.Background(), new(provisioners.Input))
		assert.NoError(t, err)

		for _, tc := range [][]string{
			{"thing", "environment variable 'thing' must be resolved"},
			{"HELLO", "environment variable 'HELLO' must be resolved"},
			{"HELLO", "world", "environment resource only supports a single lookup key"},
		} {
			keys, expected := tc[:len(tc)-1], tc[len(tc)-1]
			t.Run(strings.Join(tc, "."), func(t *testing.T) {
				res, err := po.OutputLookupFunc(keys...)
				if err != nil {
					assert.EqualError(t, err, expected)
				} else {
					assert.Equal(t, expected, res)
				}
			})
		}
	})

	t.Run("sub resource", func(t *testing.T) {
		sub := p.GenerateSubProvisioner("thing", "thing2.default#w.thing")

		t.Run("test match", func(t *testing.T) {
			assert.True(t, sub.Match("thing2.default#w.thing"))
			assert.False(t, sub.Match("thing2.default#w.thing2"))
			assert.False(t, sub.Match("thing2.default#thing3"))
		})

		t.Run("provision", func(t *testing.T) {
			po, err := sub.Provision(context.Background(), new(provisioners.Input))
			assert.NoError(t, err)

			for _, tc := range [][]string{
				{"thing", "environment variable 'THING_THING' must be resolved"},
				{"HELLO", "environment variable 'THING_HELLO' must be resolved"},
				{"HELLO", "world", "environment variable 'THING_HELLO_WORLD' must be resolved"},
				{"at least one output lookup key is required"},
			} {
				keys, expected := tc[:len(tc)-1], tc[len(tc)-1]
				t.Run(strings.Join(tc, "."), func(t *testing.T) {
					res, err := po.OutputLookupFunc(keys...)
					if err != nil {
						assert.EqualError(t, err, expected)
					} else {
						assert.Equal(t, expected, res)
					}
				})
			}
		})

		t.Run("access", func(t *testing.T) {
			assert.Equal(t, map[string]string{
				"HELLO": "", "THING_HELLO": "", "THING_HELLO_WORLD": "", "THING_THING": "", "thing": "",
			}, p.Accessed())
		})

	})

}

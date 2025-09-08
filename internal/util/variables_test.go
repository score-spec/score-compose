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

package util

import (
	"reflect"
	"testing"
)

func TestPrepareEnvVariables(t *testing.T) {
	tests := []struct {
		name string
		arr  []string
		want []string
	}{
		{"PrepareEnvVariables only",
			[]string{
				"$A",
				"$VAR",
				"${VARIABLE}",
			},
			[]string{
				"$$A",
				"$$VAR",
				"$${VARIABLE}",
			},
		},
		{"PrepareEnvVariables with prefix",
			[]string{
				"PREFIX-$VAR",
				"PREFIX-${VARIABLE}",
				"PREFIX_$VAR",
				"PREFIX_${VARIABLE}",
				"PREFIX${VARIABLE}",
			},
			[]string{
				"PREFIX-$$VAR",
				"PREFIX-$${VARIABLE}",
				"PREFIX_$$VAR",
				"PREFIX_$${VARIABLE}",
				"PREFIX$${VARIABLE}",
			},
		},
		{"PrepareEnvVariables with prefix and special character",
			[]string{
				"PREFIX@${VARIABLE}",
				"PREFIX#${VARIABLE}",
				"PREFIX!${VARIABLE}",
				"PREFIX&${VARIABLE}",
				"PREFIX%${VARIABLE}",
				"PREFIX:${VARIABLE}",
				"PREFIX;${VARIABLE}",
				"PREFIX=${VARIABLE}",
				"PREFIX|${VARIABLE}",
				"PREFIX~${VARIABLE}",
				"'PREFIX'${VARIABLE}",
				"PREFIX.${VARIABLE}",
				"PREFIX*${VARIABLE}",
				"PREFIX?${VARIABLE}",
				"PREFIX^${VARIABLE}",
				`PREFIX"${VARIABLE}"`,
			},
			[]string{
				"PREFIX@$${VARIABLE}",
				"PREFIX#$${VARIABLE}",
				"PREFIX!$${VARIABLE}",
				"PREFIX&$${VARIABLE}",
				"PREFIX%$${VARIABLE}",
				"PREFIX:$${VARIABLE}",
				"PREFIX;$${VARIABLE}",
				"PREFIX=$${VARIABLE}",
				"PREFIX|$${VARIABLE}",
				"PREFIX~$${VARIABLE}",
				"'PREFIX'$${VARIABLE}",
				"PREFIX.$${VARIABLE}",
				"PREFIX*$${VARIABLE}",
				"PREFIX?$${VARIABLE}",
				"PREFIX^$${VARIABLE}",
				`PREFIX"$${VARIABLE}"`,
			},
		},
		{"PrepareEnvVariables with prefix and slashes",
			[]string{
				"PREFIX/${VARIABLE}",
				"PREFIX\\${VARIABLE}",
			},
			[]string{
				"PREFIX/$${VARIABLE}",
				"PREFIX\\$${VARIABLE}",
			},
		},
		{"PrepareEnvVariables with prefix and brackets",
			[]string{
				"(PREFIX)${VARIABLE}",
				"PREFIX(${VARIABLE})",
				"[PREFIX]${VARIABLE}",
				"PREFIX[${VARIABLE}]",
				"{PREFIX}${VARIABLE}",
				"PREFIX{${VARIABLE}}",
			},
			[]string{
				"(PREFIX)$${VARIABLE}",
				"PREFIX($${VARIABLE})",
				"[PREFIX]$${VARIABLE}",
				"PREFIX[$${VARIABLE}]",
				"{PREFIX}$${VARIABLE}",
				"PREFIX{$${VARIABLE}}",
			},
		},
		{"PrepareEnvVariables with suffix",
			[]string{
				"$VAR-SUFFIX",
				"${VARIABLE}-SUFFIX",
				"$VAR_SUFFIX",
				"${VARIABLE}_SUFFIX",
				"${VARIABLE}SUFFIX",
			},
			[]string{
				"$$VAR-SUFFIX",
				"$${VARIABLE}-SUFFIX",
				"$$VAR_SUFFIX",
				"$${VARIABLE}_SUFFIX",
				"$${VARIABLE}SUFFIX",
			},
		}, {"PrepareEnvVariables with one letter",
			[]string{
				"echo hello $A ${B} $C world",
				"echo hello $A-${B}-$C world",
				"echo hello $A_${B}_$C world",
				"echo hello $A${B}$C world",
			},
			[]string{
				"echo hello $$A $${B} $$C world",
				"echo hello $$A-$${B}-$$C world",
				"echo hello $$A_$${B}_$$C world",
				"echo hello $$A$${B}$$C world",
			},
		},
		{"PrepareEnvVariables with several letters",
			[]string{
				"echo hello $AAA ${BBB} $CCC world",
				"echo hello $AAA-${BBB}-$CCC world",
				"echo hello $AAA_${BBB}_$CCC world",
				"echo hello $AAA${BBB}$CCC world",
			},
			[]string{
				"echo hello $$AAA $${BBB} $$CCC world",
				"echo hello $$AAA-$${BBB}-$$CCC world",
				"echo hello $$AAA_$${BBB}_$$CCC world",
				"echo hello $$AAA$${BBB}$$CCC world",
			},
		},
		{"PrepareEnvVariables complex example",
			[]string{
				"curl https://${SUBDOMAIN}.${HOST}:${PORT}/${PATH}?${PARAM1}=${VALUE1}&${PARAM2}=${VALUE2}#${REFID}",
				"mysql://${USER}:${PASSWORD}@${HOST}:${PORT}/${DATABASE}?${KEY1}=${VALUE1}&${KEY2}=${VALUE2}&${KEY3}=${VALUE3}",
			},
			[]string{
				"curl https://$${SUBDOMAIN}.$${HOST}:$${PORT}/$${PATH}?$${PARAM1}=$${VALUE1}&$${PARAM2}=$${VALUE2}#$${REFID}",
				"mysql://$${USER}:$${PASSWORD}@$${HOST}:$${PORT}/$${DATABASE}?$${KEY1}=$${VALUE1}&$${KEY2}=$${VALUE2}&$${KEY3}=$${VALUE3}",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PrepareEnvVariables(tt.arr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PrepareEnvVariables()\nget  = %v\nwant = %v\n", got, tt.want)
			}
		})
	}
}

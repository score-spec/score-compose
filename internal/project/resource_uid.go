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

package project

import (
	"fmt"
	"strings"

	"github.com/score-spec/score-compose/internal/util"
)

type ResourceUid string

func NewResourceUid(workloadName string, resName string, resType string, resClass *string, resId *string) ResourceUid {
	if resClass == nil {
		resClass = util.Ref("default")
	}
	if resId != nil {
		return ResourceUid(fmt.Sprintf("%s.%s#%s", resType, *resClass, *resId))
	}
	return ResourceUid(fmt.Sprintf("%s.%s#%s.%s", resType, *resClass, workloadName, resName))
}

func (r ResourceUid) Type() string {
	return strings.SplitN(string(r), ".", 2)[0]
}

func (r ResourceUid) Class() string {
	return strings.SplitN(strings.SplitN(string(r), "#", 2)[0], ".", 2)[1]
}

func (r ResourceUid) Id() string {
	return strings.SplitN(string(r), "#", 2)[1]
}

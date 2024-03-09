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

package project

import (
	"fmt"

	score "github.com/score-spec/score-go/types"

	"github.com/score-spec/score-compose/internal/ref"
)

type ResourceCoordinate struct {
	resType  string
	resClass string
	resId    string
}

func (r ResourceCoordinate) Uid() string {
	return fmt.Sprintf("%s.%s#%s", r.resType, r.resClass, r.resId)
}

func (r ResourceCoordinate) Type() string {
	return r.resType
}

func (r ResourceCoordinate) Class() string {
	return r.resClass
}

func (r ResourceCoordinate) Id() string {
	return r.resId
}

func NewResourceCoordinate(workloadName string, resName string, resource *score.Resource) ResourceCoordinate {
	resId := fmt.Sprintf("%s.%s", workloadName, resName)
	if resource.Id != nil {
		resId = *resource.Id
	}
	return ResourceCoordinate{
		resType:  resource.Type,
		resClass: ref.DerefOr(resource.Class, "default"),
		resId:    resId,
	}
}

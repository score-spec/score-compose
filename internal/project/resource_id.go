package project

import (
	"fmt"

	score "github.com/score-spec/score-go/types"

	"github.com/score-spec/score-compose/internal/ref"
)

func GenerateResourceClassAndId(workloadName string, resName string, resource *score.Resource) (string, string) {
	resClass := ref.DerefOr(resource.Class, "default")
	resId := GenerateResourceId(workloadName, resName, resource)
	return resClass, resId
}

func GenerateResourceId(workloadName string, resName string, resource *score.Resource) string {
	if resource.Id != nil {
		return *resource.Id
	}
	// NOTE that the schema validation prevents the resource id from containing '.', so it cannot collide with these
	// shared variables here.
	return fmt.Sprintf("%s.%s", workloadName, resName)
}

// GenerateResourceUidFromParts generates a deterministic resource id for a resource we've matched.
func GenerateResourceUidFromParts(resType, resClass, resId string) string {
	return fmt.Sprintf("%s.%s#%s", resType, resClass, resId)
}

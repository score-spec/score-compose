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
	// TODO: if we support shared resources - use a shared name
	return fmt.Sprintf("%s.%s", workloadName, resName)
}

// GenerateResourceUidFromParts generates a deterministic resource id for a resource we've matched.
func GenerateResourceUidFromParts(resType, resClass, resId string) string {
	return fmt.Sprintf("%s.%s#%s", resType, resClass, resId)
}

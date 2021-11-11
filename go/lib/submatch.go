package lib

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"strings"
)

func FilterSubmatches(recognisedEntities []*pb.Entity) []*pb.Entity {
	filteredEntities := make([]*pb.Entity, 0, len(recognisedEntities))

OuterLoopLabel:
	for _, submatchCanditate := range recognisedEntities {
		for _, re := range recognisedEntities {
			if IsSubmatch(submatchCanditate, re) {
				continue OuterLoopLabel
			}
		}
		filteredEntities = append(filteredEntities, submatchCanditate)
	}
	return filteredEntities
}

func IsSubmatch(canditate, entity *pb.Entity) bool {
	return len(canditate.Name) < len(entity.Name) &&
		canditate.Xpath == entity.Xpath &&
		strings.Contains(entity.Name, canditate.Name)
}


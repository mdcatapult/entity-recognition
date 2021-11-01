package lib

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"strings"
)

func FilterSubmatches(recognisedEntities []*pb.RecognizedEntity) []*pb.RecognizedEntity {
	filteredEntities := make([]*pb.RecognizedEntity, 0, len(recognisedEntities))

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

func IsSubmatch(canditate, entity *pb.RecognizedEntity) bool {
	return len(canditate.Entity) < len(entity.Entity) &&
		canditate.Xpath == entity.Xpath &&
		strings.Contains(entity.Entity, canditate.Entity)
}


/*
 * Copyright 2022 Medicines Discovery Catapult
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lib

import (
	"strings"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
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

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
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

func Test_isSubmatch(t *testing.T) {
	type args struct {
		canditate *pb.Entity
		entity    *pb.Entity
	}
	tests := []struct {
		name     string
		args     args
		expected bool
	}{
		{
			name: "is a submatch",
			args: args{
				canditate: &pb.Entity{
					Name:  "sub",
					Xpath: "/html/*[1]/*[2]",
				},
				entity: &pb.Entity{
					Name:  "substantial",
					Xpath: "/html/*[1]/*[2]",
				},
			},
			expected: true,
		},
		{
			name: "is not a submatch, longer entity",
			args: args{
				canditate: &pb.Entity{
					Name:  "substantially",
					Xpath: "/html/*[1]/*[2]",
				},
				entity: &pb.Entity{
					Name:  "substantial",
					Xpath: "/html/*[1]/*[2]",
				},
			},
			expected: false,
		},
		{
			name: "is not a submatch, different xpath",
			args: args{
				canditate: &pb.Entity{
					Name:  "sub",
					Xpath: "/html/*[1]/*[2]",
				},
				entity: &pb.Entity{
					Name:  "substantial",
					Xpath: "/html/*[1]/*[3]",
				},
			},
			expected: false,
		},
		{
			name: "is not a submatch, no substring match",
			args: args{
				canditate: &pb.Entity{
					Name:  "dave",
					Xpath: "/html/*[1]/*[2]",
				},
				entity: &pb.Entity{
					Name:  "substantial",
					Xpath: "/html/*[1]/*[2]",
				},
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		actual := IsSubmatch(tt.args.canditate, tt.args.entity)
		assert.Equal(t, tt.expected, actual, tt.name)
	}
}

func Test_filterSubmatches(t *testing.T) {
	type args struct {
		recognisedEntities []*pb.Entity
	}
	tests := []struct {
		name     string
		args     args
		expected []*pb.Entity
	}{
		{
			name: "",
			args: args{
				recognisedEntities: []*pb.Entity{
					{
						Name:  "substantially",
						Xpath: "/html/*[1]",
					},
					{
						// small submatch (should be removed)
						Name:  "sub",
						Xpath: "/html/*[1]",
					},
					{
						// longer submatch (should be removed)
						Name:  "substantial",
						Xpath: "/html/*[1]",
					},
					{
						// different xpath
						Name:  "sub",
						Xpath: "/html/*[2]",
					},
					{
						// Doesn't match substring
						Name:  "dave",
						Xpath: "/html/*[1]",
					},
				},
			},
			expected: []*pb.Entity{
				{
					Name:  "sub",
					Xpath: "/html/*[2]",
				},
				{
					Name:  "substantially",
					Xpath: "/html/*[1]",
				},
				{
					Name:  "dave",
					Xpath: "/html/*[1]",
				},
			},
		},
	}
	for _, tt := range tests {
		actual := FilterSubmatches(tt.args.recognisedEntities)
		assert.ElementsMatch(t, tt.expected, actual, tt.name)
	}
}

package lib

import (
	"github.com/stretchr/testify/assert"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"testing"
)

func Test_isSubmatch(t *testing.T) {
	type args struct {
		canditate *pb.RecognizedEntity
		entity    *pb.RecognizedEntity
	}
	tests := []struct {
		name string
		args args
		expected bool
	}{
		{
			name: "is a submatch",
			args: args{
				canditate: &pb.RecognizedEntity{
					Entity:      "sub",
					Xpath:       "/html/*[1]/*[2]",
				},
				entity:    &pb.RecognizedEntity{
					Entity:      "substantial",
					Xpath:       "/html/*[1]/*[2]",
				},
			},
			expected: true,
		},
		{
			name: "is not a submatch, longer entity",
			args: args{
				canditate: &pb.RecognizedEntity{
					Entity:      "substantially",
					Xpath:       "/html/*[1]/*[2]",
				},
				entity:    &pb.RecognizedEntity{
					Entity:      "substantial",
					Xpath:       "/html/*[1]/*[2]",
				},
			},
			expected: false,
		},
		{
			name: "is not a submatch, different xpath",
			args: args{
				canditate: &pb.RecognizedEntity{
					Entity:      "sub",
					Xpath:       "/html/*[1]/*[2]",
				},
				entity:    &pb.RecognizedEntity{
					Entity:      "substantial",
					Xpath:       "/html/*[1]/*[3]",
				},
			},
			expected: false,
		},
		{
			name: "is not a submatch, no substring match",
			args: args{
				canditate: &pb.RecognizedEntity{
					Entity:      "dave",
					Xpath:       "/html/*[1]/*[2]",
				},
				entity:    &pb.RecognizedEntity{
					Entity:      "substantial",
					Xpath:       "/html/*[1]/*[2]",
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
		recognisedEntities []*pb.RecognizedEntity
	}
	tests := []struct {
		name string
		args args
		expected []*pb.RecognizedEntity
	}{
		{
			name: "",
			args: args{
				recognisedEntities: []*pb.RecognizedEntity{
					{
						Entity: "substantially",
						Xpath: "/html/*[1]",
					},
					{
						// small submatch (should be removed)
						Entity: "sub",
						Xpath: "/html/*[1]",
					},
					{
						// longer submatch (should be removed)
						Entity: "substantial",
						Xpath: "/html/*[1]",
					},
					{
						// different xpath
						Entity: "sub",
						Xpath: "/html/*[2]",
					},
					{
						// Doesn't match substring
						Entity: "dave",
						Xpath: "/html/*[1]",
					},
				},
			},
			expected: []*pb.RecognizedEntity{
				{
					Entity: "sub",
					Xpath: "/html/*[2]",
				},
				{
					Entity: "substantially",
					Xpath: "/html/*[1]",
				},
				{
					Entity: "dave",
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
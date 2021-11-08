package blacklist

import (
	"github.com/stretchr/testify/assert"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"testing"
)

func Test_Blacklist(t *testing.T) {
	//blacklist = map[string]interface{}{
	//	"an entity": true,
	//}
	snip := pb.Snippet{
		Text:           "a-snippet",
		NormalisedText: "",
		Offset:         0,
		Xpath:          "",
	}
	res, err := Ok(&snip)
	assert.NoError(t, err)
	assert.Equal(t, true, res)
}

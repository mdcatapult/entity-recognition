package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/unicode/norm"
)

type htmlTextWithLocationOffests struct {
	Text []byte `json:"rawtext"`
	Offsets []int `json:"offsets"`
}

func main() {
	r := gin.Default()
	r.POST("/", func(c *gin.Context) {
		b, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			_ = c.AbortWithError(400, err); return
		}

		var textWithOffsets htmlTextWithLocationOffests
		if err := json.Unmarshal(b, &textWithOffsets); err != nil {
			_ = c.AbortWithError(400, err); return
		}

		b = norm.NFKD.Bytes(textWithOffsets.Text)
		c.Data(200, "text/plain", b)
	})
	_ = r.Run()
}

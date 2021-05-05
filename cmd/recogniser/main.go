package main

import (
	"github.com/gin-gonic/gin"
)


func init() {

}

func main() {
	r := gin.Default()
	r.POST("/", func(c *gin.Context) {
		c.JSON(200, "not implemented yet")
	})
	_ = r.Run(":8083")
}
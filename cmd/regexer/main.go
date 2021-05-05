package main

import (
	"io/ioutil"
	"regexp"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
)

var regexps map[string]*regexp.Regexp
var regexpStringMap = make(map[string]string)

type result struct {
	Name string `json:"name"`
	Regexp string `json:"regexp"`
}

func init() {
	b, err := ioutil.ReadFile("regexps.yml")
	if err != nil {
		panic(err)
	}

	if err := yaml.Unmarshal(b, &regexpStringMap); err != nil {
		panic(err)
	}

	regexps = make(map[string]*regexp.Regexp)
	for name, uncompiledRegexp := range regexpStringMap {
		regexps[name] = regexp.MustCompile(uncompiledRegexp)
	}
}

func main() {
	r := gin.Default()
	r.POST("/", func(c *gin.Context) {
		b, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			_ = c.AbortWithError(400, err); return
		}

		results := make([]result, 0, len(regexps))
		for name, re := range regexps {
			if re.Match(b) {
				results = append(results, result{
					Name: name,
					Regexp: regexpStringMap[name],
				})
			}
		}

		c.JSON(200, results)
	})

	_ = r.Run(":8082")
}
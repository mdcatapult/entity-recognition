package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	. "github.com/onsi/gomega"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

func GetEntities(host, port, source, contentType string) []pb.Entity {
	reader := strings.NewReader(source)
	res, err := http.Post(fmt.Sprintf("http://%s:%s/entities?recogniser=dictionary", host, port), contentType, reader)

	Expect(err).Should(BeNil())

	var b []byte
	_, err = res.Body.Read(b)

	Expect(err).Should(BeNil())
	Expect(res.StatusCode).Should(Equal(200))

	body, err := ioutil.ReadAll(res.Body)
	Expect(err).Should(BeNil())

	var entities []pb.Entity
	err = json.Unmarshal(body, &entities)
	Expect(err).Should(BeNil())

	return entities
}

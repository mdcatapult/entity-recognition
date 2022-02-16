package api_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

// This must be set for these tests to run
const envVar = "NER_API_TEST"

func TestMain(m *testing.M) {

	if os.Getenv(envVar) == "" {
		fmt.Println("SKIPPING API TESTS: set NER_API_TEST to run API tests")
	}

	os.Exit(m.Run())
}

func TestAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Suite")
}

var _ = Describe("Entity Recognition API", func() {

	var _ = Describe("should recognise in html", func() {

		It("plain entity", func() {

			html := "<html>calcium</html>"

			entities := getEntities(html)

			Expect(len(entities)).Should(Equal(1))
			Expect(entities[0].Name).Should(Equal("calcium"))
		})

		It("entity needing normalization", func() {

			html := "<html>calcium)</html>"

			entities := getEntities(html)

			Expect(len(entities)).Should(Equal(1))
			Expect(entities[0].Name).Should(Equal("calcium"))
		})
	})
})

func getEntities(html string) []pb.Entity {
	htmlReader := strings.NewReader(html)
	res, err := http.Post("http://localhost:8080/entities?recogniser=dictionary", "text/html", htmlReader)

	Expect(err).Should(BeNil())
	Expect(res.StatusCode).Should(Equal(200))

	body, err := ioutil.ReadAll(res.Body)
	Expect(err).Should(BeNil())

	var entities []pb.Entity
	json.Unmarshal(body, &entities)
	return entities
}

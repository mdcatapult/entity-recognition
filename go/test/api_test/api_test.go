package apitest

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

const (
	envVar = "NER_API_TEST" // This must be set for these tests to run
	host   = "localhost"
	port   = "8080"
)

func TestMain(m *testing.M) {
	if os.Getenv(envVar) == "" {
		fmt.Printf("SKIPPING API TESTS: set %s to run API tests", envVar)
		return
	}

	os.Exit(m.Run())
}

func TestAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Suite")
}

var _ = Describe("Entity Recognition API", func() {

	var _ = Describe("bad requests", func() {

		It("should return bad request for invalid recogniser", func() {
			htmlReader := strings.NewReader("<html>calcium</html>")
			res, err := http.Post(fmt.Sprintf("http://%s:%s/entities?recogniser=invalid-recogniser", host, port), "text/html", htmlReader)

			Expect(err).Should(BeNil())
			Expect(res.StatusCode).Should(Equal(400))

		})

		It("should return bad request for missing recogniser", func() {
			htmlReader := strings.NewReader("<html>calcium</html>")
			res, err := http.Post(fmt.Sprintf("http://%s:%s/entities?recogniser", host, port), "text/html", htmlReader)

			Expect(err).Should(BeNil())
			Expect(res.StatusCode).Should(Equal(400))
		})

		It("should return bad request for invalid content type", func() {

			contentType := "nonsense"
			htmlReader := strings.NewReader("<html>calcium</html>")
			res, err := http.Post(fmt.Sprintf("http://%s:%s/entities?recogniser=dictionary", host, port), contentType, htmlReader)

			Expect(err).Should(BeNil())
			Expect(res.StatusCode).Should(Equal(400))
		})

		It("should return bad request for missing source text", func() {

			htmlReader := strings.NewReader("")
			res, err := http.Post(fmt.Sprintf("http://%s:%s/entities?recogniser=dictionary", host, port), "text/html", htmlReader)

			Expect(err).Should(BeNil())
			Expect(res.StatusCode).Should(Equal(400))
		})
	})

	var _ = Describe("should recognise in html", func() {

		It("plain entity", func() {

			html := "<html>calcium</html>"
			entities := getEntities(html, "text/html")

			Expect(len(entities)).Should(Equal(1))
			Expect(entities[0].Name).Should(Equal("calcium"))
			Expect(hasIdentifier(&entities[0], "ca")).Should(BeTrue())
		})

		It("multiple entities", func() {

			html := "<html>calcium entity</html>"
			entities := getEntities(html, "text/html")

			Expect(len(entities)).Should(Equal(2))

			for _, entity := range []string{
				"calcium",
				"entity",
			} {
				Expect(hasEntity(entities, entity)).Should(BeTrue())
			}
		})

		It("no recognised entities", func() {

			html := "<html>nonsense</html>"
			entities := getEntities(html, "text/html")

			Expect(len(entities)).Should(Equal(0))
		})

		It("entity needing normalization", func() {

			html := "<html>calcium)</html>"
			entities := getEntities(html, "text/html")

			Expect(len(entities)).Should(Equal(1))
			Expect(entities[0].Name).Should(Equal("calcium"))
		})

		It("nested xpath", func() {
			html := "<html><div>nonsense</div><div><span>calcium</span></div></html>"
			entities := getEntities(html, "text/html")

			Expect(len(entities)).Should(Equal(1))
			Expect(entities[0].GetXpath()).Should(Equal("/html/*[2]"))
		})
	})

	var _ = Describe("should recognise in plaintext", func() {

		It("plain entity", func() {

			text := "calcium"
			entities := getEntities(text, "text/plain")

			Expect(len(entities)).Should(Equal(1))
			Expect(entities[0].Name).Should(Equal("calcium"))
			Expect(hasIdentifier(&entities[0], "ca")).Should(BeTrue())
		})

		It("multiple entities", func() {

			text := "calcium entity"
			entities := getEntities(text, "text/plain")

			Expect(len(entities)).Should(Equal(2))

			for _, entity := range []string{
				"calcium",
				"entity",
			} {
				Expect(hasEntity(entities, entity)).Should(BeTrue())
			}
		})

	})
})

func getEntities(source, contentType string) []pb.Entity {
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

func hasEntity(entities []pb.Entity, entity string) bool {
	for i := range entities {
		if entities[i].GetName() == entity {
			return true
		}
	}
	return false
}

func hasIdentifier(entity *pb.Entity, identifier string) bool {
	for k := range entity.GetIdentifiers() {
		if k == identifier {
			return true
		}
	}
	return false
}

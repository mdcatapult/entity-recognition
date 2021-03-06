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

package apitest

import (
	"fmt"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"net/http"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/test/api_test/util"
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
			entities := util.GetEntities(host, port, html, "text/html")

			Expect(len(entities)).Should(Equal(1))
			Expect(entities[0].Name).Should(Equal("calcium"))
			Expect(hasIdentifier(entities[0], "ca")).Should(BeTrue())
		})

		It("multiple entities", func() {

			html := "<html>calcium entity</html>"
			entities := util.GetEntities(host, port, html, "text/html")

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
			entities := util.GetEntities(host, port, html, "text/html")

			Expect(len(entities)).Should(Equal(0))
		})

		It("entity needing normalization", func() {

			html := "<html>calcium)</html>"
			entities := util.GetEntities(host, port, html, "text/html")

			Expect(len(entities)).Should(Equal(1))
			Expect(entities[0].Name).Should(Equal("calcium"))
		})

		It("nested xpath", func() {
			html := "<html><div>nonsense</div><div><span>calcium</span></div></html>"
			entities := util.GetEntities(host, port, html, "text/html")

			Expect(len(entities)).Should(Equal(1))
			Expect(entities[0].Positions[0].Xpath).Should(Equal("/html/*[2]"))
		})
	})

	var _ = Describe("should recognise in plaintext", func() {

		It("plain entity", func() {

			text := "calcium"
			entities := util.GetEntities(host, port, text, "text/plain")

			Expect(len(entities)).Should(Equal(1))
			Expect(entities[0].Name).Should(Equal("calcium"))
			Expect(hasIdentifier(entities[0], "ca")).Should(BeTrue())
		})

		It("multiple entities", func() {

			text := "calcium entity"
			entities := util.GetEntities(host, port, text, "text/plain")

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

func hasEntity(entities []lib.APIEntity, entity string) bool {
	for i := range entities {
		if entities[i].Name == entity {
			return true
		}
	}
	return false
}

func hasIdentifier(entity lib.APIEntity, identifier string) bool {
	for k := range entity.Identifiers {
		if k == identifier {
			return true
		}
	}
	return false
}

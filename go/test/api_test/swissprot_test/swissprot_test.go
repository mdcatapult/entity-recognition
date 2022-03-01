package swissprot

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
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

func TestSwissprot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Swissprot Suite")
}

var _ = Describe("Swissprot", func() {

	var entities []pb.Entity
	const species = "Drosophila melanogaster"

	BeforeEach(func() {
		html := "<html>Q540X7</html>"
		entities = util.GetEntities(host, port, html, "text/html")

	})

	It("should return entities", func() {
		Expect(len(entities)).Should(Equal(1))
	})

	It("should populate entity identifiers by species", func() {

		identifiers := entities[0].GetIdentifiers()
		Expect(identifiers).ShouldNot(BeEmpty())
		Expect(identifiers[species]).ShouldNot(BeNil())

		var speciesIdentifiers map[string]string
		err := json.Unmarshal([]byte(identifiers[species]), &speciesIdentifiers)
		Expect(err).Should(BeNil())

		accession := speciesIdentifiers["Accession"]
		Expect(accession).Should(Equal("P02574"))

	})

	It("should populate entity metadata by species", func() {

		jsonMetadata := entities[0].GetMetadata()
		Expect(jsonMetadata).ShouldNot(BeEmpty())

		var metadata map[string]map[string]string
		err := json.Unmarshal([]byte(jsonMetadata), &metadata)
		Expect(err).Should(BeNil())

		speciesMetadata := metadata[species]
		Expect(speciesMetadata).ShouldNot(BeNil())
		Expect(speciesMetadata["Accessions"]).Should(Equal("P02574, Q540X7, Q9VNW5"))

	})
})

package swissprot

import (
	"fmt"
	"os"
	"reflect"
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
		fmt.Printf("SKIPPING SWISSPROT API TESTS: set %s to run API tests", envVar)
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

	BeforeEach(func() {
		html := "<html>Q540X7</html>"
		entities = util.GetEntities(host, port, html, "text/html")

	})

	It("should return entities", func() {
		Expect(len(entities)).Should(Equal(1))

		expectedEntity := pb.Entity{
			Name:       "Q540X7",
			Position:   6,
			Xpath:      "/html",
			Recogniser: "dictionary",
			Identifiers: map[string]string{
				"Drosophila melanogaster": "{\"Accession\":\"P02574\",\"BioGRID\":\"65684\",\"ExpressionAtlas\":\"P02574\",\"GeneTree\":\"ENSGT00940000175284\",\"IntAct\":\"P02574\",\"InterPro\":\"IPR004000, IPR020902, IPR004001, IPR043129\",\"KEGG\":\"dme:Dmel_CG7478\",\"Pfam\":\"PF00022\",\"PrimaryGeneName\":\"Act79B\",\"RefSeq\":\"NP_001262200.1, NP_524210.1\"}",
			},
			Metadata: `{"Drosophila melanogaster":{"Accessions":"P02574, Q540X7, Q9VNW5","Primary Gene Name":"Act79B","Protein Name":"Actin, larval muscle","Scientific Organism Name":"Drosophila melanogaster","sequence":"MCDEEASALVVDNGSGMCKAGFAGDDAPRAVFPSIVGRPRHQGVMVGMGQKDCYVGDEAQSKRGILSLKYPIEHGIITNWDDMEKVWHHTFYNELRVAPEEHPVLLTEAPLNPKANREKMTQIMFETFNSPAMYVAIQAVLSLYASGRTTGIVLDSGDGVSHTVPIYEGYALPHAILRLDLAGRDLTDYLMKILTERGYSFTTTAEREIVRDIKEKLCYVALDFEQEMATAAASTSLEKSYELPDGQVITIGNERFRTPEALFQPSFLGMESCGIHETVYQSIMKCDVDIRKDLYANNVLSGGTTMYPGIADRMQKEITALAPSTIKIKIIAPPERKYSVWIGGSILASLSTFQQMWISKQEYDESGPGIVHRKCF","sequence length":"376","sequence mass":"41787","subcellular location":"Cytoplasm, Cytoskeleton"}}`,
		}

		Expect(reflect.DeepEqual(&entities[0], &expectedEntity)).Should(BeTrue())

	})

})

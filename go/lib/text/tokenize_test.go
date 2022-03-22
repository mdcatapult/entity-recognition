package text

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

func TestTokeniser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tokeniser Suite")
}

var _ = Describe("Tokenise", func() {

	var tokens []*pb.Snippet
	onTokenCallback := func(snippet *pb.Snippet) error {
		tokens = append(tokens, snippet)
		return nil
	}

	BeforeEach(func() {
		tokens = make([]*pb.Snippet, 0)
	})

	var _ = Describe("Exact match", func() {

		const exactMatch = true

		It("special char and preceding/trailing spaces", func() {

			snippet := &pb.Snippet{
				Text: " £ some text ",
			}
			expectedTexts := []string{"£", "some", "text"}
			expectedOffsets := []uint32{1, 3, 8}

			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())
			assertTokenize(tokens, expectedTexts, expectedOffsets)

		})

		It("text with special char", func() {

			snippet := &pb.Snippet{
				Text: "some-text$ ",
			}
			expectedTexts := []string{"some-text$"}
			expectedOffsets := []uint32{0}

			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

		It("text starting with non alpha char, containing alpha and non alpha, ending in space", func() {
			snippet := &pb.Snippet{
				Text: "- apple !@£ pie-face ",
			}
			expectedTexts := []string{"-", "apple", "!@£", "pie-face"}
			expectedOffsets := []uint32{0, 2, 8, 12}
			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

		It("with existing offset", func() {
			snippet := &pb.Snippet{
				Text:   "Halogen-bonding-triggered supramolecular gel formation.",
				Offset: 100,
			}
			expectedTexts := []string{"Halogen-bonding-triggered", "supramolecular", "gel", "formation."}
			expectedOffsets := []uint32{100, 126, 141, 145}
			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

		It("greek characters", func() {
			snippet := &pb.Snippet{
				Text: "βωα -νπψ- lamb ανπψ",
			}
			expectedTexts := []string{"βωα", "-νπψ-", "lamb", "ανπψ"}
			expectedOffsets := []uint32{0, 4, 10, 15}
			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

		It("chemicals with ( and ;", func() {
			snippet := &pb.Snippet{
				Text: "(MDMA; Ecstasy)",
			}
			expectedTexts := []string{"(MDMA;", "Ecstasy)"}
			expectedOffsets := []uint32{0, 7}
			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

		It("chemicals with / and (", func() {
			snippet := &pb.Snippet{
				Text: "pluronic/poly(acrylic acid)",
			}
			expectedTexts := []string{"pluronic/poly(acrylic", "acid)"}
			expectedOffsets := []uint32{0, 22}
			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

		It("chemicals with +", func() {
			snippet := &pb.Snippet{
				Text: "verapamil+bufadienolide",
			}
			expectedTexts := []string{"verapamil+bufadienolide"}
			expectedOffsets := []uint32{0}
			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

		It("two interesting characters: zhōng wén", func() {
			snippet := &pb.Snippet{
				Text: "中文",
			}
			expectedTexts := []string{"中文"}
			expectedOffsets := []uint32{0}

			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

	})

	var _ = Describe("non-exact match", func() {

		const exactMatch = false

		It("should break text on '-'", func() {
			snippet := &pb.Snippet{
				Text: "some-text",
			}
			expectedTexts := []string{"some", "-", "text"}
			expectedOffsets := []uint32{0, 4, 5}
			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

		It("should break text on '-' with existing offset", func() {
			snippet := &pb.Snippet{
				Text:   "some-text",
				Offset: 100,
			}
			expectedTexts := []string{"some", "-", "text"}
			expectedOffsets := []uint32{100, 104, 105}
			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

		It("should handle spaces'", func() {
			snippet := &pb.Snippet{
				Text: "some text",
			}
			expectedTexts := []string{"some", "text"}
			expectedOffsets := []uint32{0, 5}
			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

		It("should handle special chars", func() {
			snippet := &pb.Snippet{
				Text: "βωα βωα hello",
			}
			expectedTexts := []string{"βωα", "βωα", "hello"}
			expectedOffsets := []uint32{0, 4, 8}
			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

		It("should handle trailing and leading spaces", func() {
			snippet := &pb.Snippet{
				Text: " some -text some-text ",
			}
			expectedTexts := []string{"some", "-", "text", "some", "-", "text"}
			expectedOffsets := []uint32{1, 6, 7, 12, 16, 17}
			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

		It("chemicals with ( and ;", func() {
			snippet := &pb.Snippet{
				Text: "(MDMA; Ecstasy)",
			}
			expectedTexts := []string{"(", "MDMA", ";", "Ecstasy", ")"}
			expectedOffsets := []uint32{0, 1, 5, 7, 14}
			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

		It("chemicals with / and (", func() {
			snippet := &pb.Snippet{
				Text: "pluronic/poly(acrylic acid)",
			}
			expectedTexts := []string{"pluronic", "/", "poly", "(", "acrylic", "acid", ")"}
			expectedOffsets := []uint32{0, 8, 9, 13, 14, 22, 26}
			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

		It("chemicals with +", func() {
			snippet := &pb.Snippet{
				Text: "verapamil+bufadienolide",
			}
			expectedTexts := []string{"verapamil", "+", "bufadienolide"}
			expectedOffsets := []uint32{0, 9, 10}
			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

		It("two interesting characters: zhōng wén", func() {
			snippet := &pb.Snippet{
				Text: "中文",
			}
			expectedTexts := []string{"中", "文"}
			expectedOffsets := []uint32{0, 1}

			err := Tokenize(snippet, onTokenCallback, exactMatch)
			Expect(err).Should(BeNil())

			assertTokenize(tokens, expectedTexts, expectedOffsets)
		})

	})
})

func assertTokenize(tokens []*pb.Snippet, expectedTexts []string, expectedOffsets []uint32) {
	Expect(len(tokens)).Should(Equal(len(expectedTexts)))
	Expect(hasTokens(expectedTexts, tokens)).Should(BeTrue())
	Expect(hasOffsets(expectedOffsets, tokens)).Should(BeTrue())
}

func hasTokens(expected []string, actual []*pb.Snippet) bool {
	for i, expectedText := range expected {
		if actual[i].Text != expectedText {
			return false
		}

	}
	return true
}

func hasOffsets(expected []uint32, actual []*pb.Snippet) bool {
	for i, expectedOffset := range expected {
		if actual[i].Offset != expectedOffset {
			return false
		}
	}
	return true
}

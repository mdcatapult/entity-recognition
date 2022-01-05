package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var router *gin.Engine
var ginContext *gin.Context

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = Describe("GetRecognisers", func() {

	var _ = Describe("Status codes", Ordered, func() {

		var _ = BeforeAll(func() {
			ginContext, router = gin.CreateTestContext(httptest.NewRecorder())

			router.GET("/statusCodeTests", server{}.GetRecognisers)

			go router.Run("localhost:9999")

			// wait for server to start
			time.Sleep(1 * time.Second)
		})

		var _ = It("Should be a bad request when no recognisers are specified", func() {
			res, err := http.Get("http://localhost:9999/statusCodeTests")

			Ω(err).Should(BeNil())
			Ω(res.StatusCode).Should(Equal(http.StatusBadRequest))
		})

		var _ = It("Should return status OK", func() {
			res, err := http.Get("http://localhost:9999/statusCodeTests?recogniser=something")

			Ω(err).Should(BeNil())
			Ω(res.StatusCode).Should(Equal(http.StatusOK))
		})
	})

	var _ = Describe("Adding recognisers to context", Ordered, func() {

		recogniser1 := "recogniser1"
		recogniser2 := "recogniser2"

		var _ = BeforeAll(func() {
			_, router = gin.CreateTestContext(httptest.NewRecorder())

			singleRecogniserAsserter := func(c *gin.Context) {
				receivedRecogniser, ok := c.Get(recognisersKey)
				Ω(ok).Should(Equal(true))

				recognisers, ok := receivedRecogniser.([]lib.RecogniserOptions)

				Ω(ok).Should(Equal(true))
				Ω(len(recognisers)).Should(Equal(1))
				Ω(recognisers[0].Name).Should(Equal(recogniser1))
			}

			multipleRecogniserAsserter := func(c *gin.Context) {

				receivedRecognisers, ok := c.Get(recognisersKey)
				Ω(ok).Should(Equal(true))

				recognisers, ok := receivedRecognisers.([]lib.RecogniserOptions)
				//
				Ω(ok).Should(Equal(true))
				Ω(len(recognisers)).Should(Equal(2))
				Ω(recognisers[0].Name).Should(Equal(recogniser1))
				Ω(recognisers[1].Name).Should(Equal(recogniser2))
				fmt.Println("multip recog: ", receivedRecognisers)
			}

			router.GET("/singleRecogniser", server{}.GetRecognisers, singleRecogniserAsserter)
			router.GET("/multipleRecogniser", server{}.GetRecognisers, multipleRecogniserAsserter)
			go router.Run("localhost:9998")
		})

		var _ = It("Should add single recogniser to context", func() {
			res, err := http.Get(fmt.Sprintf("http://localhost:9998/singleRecogniser?%v=%v", "recogniser", recogniser1))
			Ω(err).Should(BeNil())
			Ω(res.StatusCode).Should(Equal(http.StatusOK))
		})

		var _ = It("Should add multiple recognisers to context", func() {
			res, err := http.Get(fmt.Sprintf("http://localhost:9998/multipleRecogniser?%v=%v&%v=%v", "recogniser", recogniser1, "recogniser", recogniser2))
			Ω(err).Should(BeNil())
			Ω(res.StatusCode).Should(Equal(http.StatusOK))
		})
	})
})

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

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	http_recogniser "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/cmd/recognition-api/http-recogniser"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/blocklist"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/recogniser"
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

			go func() {
				_ = router.Run("localhost:9999")
			}()

			// wait for server to start
			time.Sleep(1 * time.Second)
		})

		var _ = It("Should be a bad request when no recognisers are specified", func() {
			res, err := http.Get("http://localhost:9999/statusCodeTests")

			Expect(err).Should(BeNil())
			Expect(res.StatusCode).Should(Equal(http.StatusBadRequest))
		})

		var _ = It("Should return status OK", func() {
			res, err := http.Get("http://localhost:9999/statusCodeTests?recogniser=something")

			Expect(err).Should(BeNil())
			Expect(res.StatusCode).Should(Equal(http.StatusOK))
		})
	})

	var _ = Describe("Adding recognisers to context", Ordered, func() {

		recogniser1 := "recogniser1"
		recogniser2 := "recogniser2"
		recogniser3 := "recogniser3"

		testServer := server{
			controller: &controller{
				// recogniser3 on the controller will be used to test that the allRecognisers flag causes recogniser3 to be used.
				recognisers: map[string]recogniser.Client{
					recogniser3: http_recogniser.NewLeadmineClient(recogniser3, "", blocklist.Blocklist{}),
				},
			},
		}

		// set up server, routes and handler functions with assertions
		var _ = BeforeAll(func() {
			_, router = gin.CreateTestContext(httptest.NewRecorder())

			singleRecogniserAsserter := func(c *gin.Context) {
				receivedRecogniser, ok := c.Get(recognisersKey)
				Expect(ok).Should(Equal(true))

				recognisers, ok := receivedRecogniser.([]lib.RecogniserOptions)

				Expect(ok).Should(Equal(true))
				Expect(len(recognisers)).Should(Equal(1))
				Expect(recognisers[0].Name).Should(Equal(recogniser1))
			}

			multipleRecognisersAsserter := func(c *gin.Context) {

				receivedRecognisers, ok := c.Get(recognisersKey)
				Expect(ok).Should(Equal(true))

				recognisers, ok := receivedRecognisers.([]lib.RecogniserOptions)

				Expect(ok).Should(Equal(true))
				Expect(len(recognisers)).Should(Equal(2))
				Expect(recognisers[0].Name).Should(Equal(recogniser1))
				Expect(recognisers[1].Name).Should(Equal(recogniser2))
			}

			allRecognisersAsserter := func(c *gin.Context) {
				receivedRecogniser, ok := c.Get(recognisersKey)
				Expect(ok).Should(Equal(true))

				recognisers, ok := receivedRecogniser.([]lib.RecogniserOptions)

				Expect(ok).Should(Equal(true))
				Expect(len(recognisers)).Should(Equal(1))
				Expect(recognisers[0].Name).Should(Equal(recogniser3))
			}

			router.GET("/singleRecogniser", server{}.GetRecognisers, singleRecogniserAsserter)
			router.GET("/multipleRecognisers", server{}.GetRecognisers, multipleRecognisersAsserter)
			router.GET("/allRecognisers", testServer.GetRecognisers, allRecognisersAsserter)
			go func() {
				_ = router.Run("localhost:9998")
			}()

			// wait for server to start
			time.Sleep(1 * time.Second)
		})

		var _ = It("Should add single recogniser to context", func() {
			res, err := http.Get(fmt.Sprintf("http://localhost:9998/singleRecogniser?%v=%v", "recogniser", recogniser1))
			Expect(err).Should(BeNil())
			Expect(res.StatusCode).Should(Equal(http.StatusOK))
		})

		var _ = It("Should add multiple recognisers to context", func() {
			res, err := http.Get(fmt.Sprintf("http://localhost:9998/multipleRecognisers?%v=%v&%v=%v", "recogniser", recogniser1, "recogniser", recogniser2))
			Expect(err).Should(BeNil())
			Expect(res.StatusCode).Should(Equal(http.StatusOK))
		})

		var _ = It("Should use the allRecognisers flag to use all available recognisers", func() {
			res, err := http.Get("http://localhost:9998/allRecognisers?allRecognisers=true")
			Expect(err).Should(BeNil())
			Expect(res.StatusCode).Should(Equal(http.StatusOK))
		})
	})
})

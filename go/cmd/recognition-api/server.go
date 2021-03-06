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

// Package classification NER API.
//
// Documentation of NER API.
//     Version: 1.0.0
//
// swagger:meta
package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
)

const recognisersKey = "recognisers"

type HttpError struct {
	code int
	error
}

func (e HttpError) Error() string {
	return e.error.Error()
}

func NewHttpError(code int, err error) HttpError {
	return HttpError{
		code:  code,
		error: err,
	}
}

type server struct {
	controller *controller
}

func (s server) RegisterRoutes(engine *gin.Engine) {
	engine.POST("/text", validateBody, s.HTMLToText)
	engine.POST("/tokens", validateBody, s.getParams, s.Tokenise)
	engine.POST("/entities", validateBody, s.getParams, s.GetRecognisers, s.Recognize)
	engine.GET("/recognisers", s.ListRecognisers)
}

// swagger:route GET /recognisers Endpoints recognisers
// GetRecognisers returns a list of all configured recognisers.
//
//	Produces:
//		- application/json
//
//	responses:
//      200: description: A list of the names of all configured recognisers
func (s server) ListRecognisers(c *gin.Context) {
	c.JSON(200, s.controller.ListRecognisers())
}

// GetRecognisers is a gin middleware func which uses the "recogniser" query param to populate recognisers. (may be specified multiple times).
// The x-{recogniserName} header is also read to add config options to the relevant recogniser.
func (s server) GetRecognisers(c *gin.Context) {

	var requestedRecognisers []string
	allRecognisersFlag, ok := c.GetQuery("allRecognisers")
	if ok && allRecognisersFlag == "true" {
		requestedRecognisers = s.controller.ListRecognisers()
	} else {
		requestedRecognisers, ok = c.GetQueryArray("recogniser")
		if !ok {
			handleError(c, NewHttpError(400, errors.New("you must set at least one recogniser query parameter")))
			return
		}
	}

	recognisers := make([]lib.RecogniserOptions, len(requestedRecognisers))
	for i, recogniser := range requestedRecognisers {

		recognisers[i] = lib.RecogniserOptions{Name: recogniser}

		header := c.GetHeader(fmt.Sprintf("x-%s", recogniser))
		if header == "" {
			continue
		}

		headerBytes, err := base64.StdEncoding.DecodeString(header)
		if err != nil {
			handleError(c, NewHttpError(400, errors.New("invalid request header - must be base64 encoded")))
			return
		}

		var opts lib.RecogniserOptions
		if err := json.Unmarshal(headerBytes, &opts); err != nil {
			handleError(c, NewHttpError(400, errors.New("invalid request header - must be valid json (base64 encoded)")))
			return
		}
		recognisers[i].HttpOptions = opts.HttpOptions
	}

	c.Set(recognisersKey, recognisers)
	c.Next()
}

// swagger:route POST /entities Endpoints entities
//
//	/entities takes an HTML or text document and returns entities in the document by communicating with recoginsers via HTTP or GRPC.
//
//	The recognisers to use can be specified in query params.
//
//	The workflow is as follows:
//	1) 	Work out the recognisers to use based on query params.
//	2)	Ask the controller to perform recognition with the specified recognisers.
//	3)  Read the body of the HTTP request into snippets. This involves either the HTMLReader or TextReader depending on Content-Type header. The end result of this is many snippet containing parts of the document's text, for example the contents of a \<p\> tag.
//  4)  Send the snippets to Tokenise(). This will further break down the snippets into tokens (also of type *pb.Snippet). The exact-match query paramater controls how fine-grained tokenising is.
//  5)	Send tokens to each recogniser. If a token matches a key in the recogniser's dictionary, an entity will be returned from this step.
//  6)  The previous 3 steps are done in parallel, so wait for them all to complete.
//  7)  Collect all the entities returned from all the recognisers and return them in the HTTP response.
//
//	Parameters:
//    + name: recogniser
//      description: a recogniser to use for entity recognition. May be specified more than once with different values. Hit /recognisers for a list of all configured recognisers.
//      in: query
//      type: string
//      required: true
//
//    + name: exact-match
//      description:  Boolean value of whether to perform exact matching during tokenising.  With exact matching, "some-text" is a single token - "some-text". Without exact matching, tokenising is more fine grained. "some-text" would be three tokens: "some", "-", "text".
//      in: query
//      type: boolean
//      required: false
//
//	 + name: Body
//  	description: The HTML document to scan for entities
//  	in: body
//		required: true
//
// 	Consumes:
//		- text/html
//		- text/plain
//
//	Produces:
//		- application/json
//
//	responses:
//      200: []Entity
//  	400: description: Bad request - invalid content type or missing / invalid recogniser
func (s server) Recognize(c *gin.Context) {
	requestedRecognisers, ok := c.Get(recognisersKey)
	if !ok {
		handleError(c, errors.New("recognisers are unset"))
	}

	contentType, ok := allowedContentTypeEnumMap[c.ContentType()]
	if !ok {
		handleError(c, NewHttpError(400, errors.New("invalid content type - must be text/html or text/plain")))
	}

	recognisers := requestedRecognisers.([]lib.RecogniserOptions)

	// TODO: next line blocks until all of the recognisers return. If a recogniser dies then this will get stuck.
	entities, err := s.controller.Recognize(c.Request.Body, contentType, recognisers)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, entities)
}

// swagger:route POST /tokens Endpoints tokens
// /tokens splits an HTML or plain text document into tokens.
// Tokens are the segments of text from the source document which can be used to query
// a recogniser.
//
//	Parameters:
//    + name: exact-match
//      description:  Boolean value of whether to perform exact matching during tokenising.  With exact matching, "some-text" is a single token - "some-text". Without exact matching, tokenising is more fine grained. "some-text" would be three tokens: "some", "-", "text".
//
//      in: query
//      type: boolean
//      required: false
//
//	 + name: Body
//  	description: The text document to tokenise
//  	in: body
//		required: true
//
// 	Consumes:
//		- text/html
//		- text/plain
//
//	Produces:
//		- application/json
//
//	responses:
//      200: []Snippet
//  	400: description: Bad request - invalid content type.
func (s server) Tokenise(c *gin.Context) {
	contentType, ok := allowedContentTypeEnumMap[c.ContentType()]
	if !ok {
		handleError(c, NewHttpError(400, errors.New("invalid content type - must be text/html or text/plain")))
	}

	tokens, err := s.controller.Tokenize(c.Request.Body, contentType)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, tokens)
}

// swagger:route POST /text Endpoints text
// HTMLToText converts an HTML document into plain text.
//	Parameters:
//	 + name: Body
//  	description: The HTML document to convert
//  	in: body
//		required: true
//
// 	Consumes:
//		- text/html
//
//	Produces:
//		- text/plain
//
//	responses:
//      200: description: OK
//  	400: description: Bad request - invalid content type.
func (s server) HTMLToText(c *gin.Context) {
	contentType := allowedContentTypeEnumMap[c.ContentType()]
	if contentType != contentTypeHTML {
		handleError(c, NewHttpError(400, errors.New("invalid content type - must be text/html")))
	}

	data, err := s.controller.HTMLToText(c.Request.Body)
	if err != nil {
		handleError(c, err)
		return
	}

	c.Data(200, "text/plain", data)
}

func (s *server) getParams(c *gin.Context) {
	s.controller.exactMatch = c.Query("exact-match") == "true"
	c.Next()
}

func validateBody(c *gin.Context) {
	if c.Request.Body == nil {
		handleError(c, NewHttpError(400, errors.New("request body missing")))
	} else if _, err := c.Request.Body.Read(nil); err == io.EOF {
		handleError(c, NewHttpError(400, errors.New("request body missing")))
	} else {
		c.Next()
	}
}

func handleError(c *gin.Context, err error) {
	if err == nil {
		abort(c, 500, errors.New("abort called on nil error"))
	}
	switch e := err.(type) {
	case HttpError:
		abort(c, e.code, e.error)
	default:
		abort(c, 500, e)
	}
}

func abort(c *gin.Context, code int, err error) {
	switch {
	case code <= 500:
		c.JSON(code, map[string]interface{}{
			"status":  code,
			"message": err.Error(),
		})
		c.Abort()
	default:
		_ = c.AbortWithError(code, err)
	}
}

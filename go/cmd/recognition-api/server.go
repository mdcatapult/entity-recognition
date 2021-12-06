package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"io"
)

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
	controller controller
}

func (s server) RegisterRoutes(r *gin.Engine) {
	r.POST("/text", validateBody, s.HTMLToText)
	r.POST("/tokens", validateBody, s.Tokenize)
	r.POST("/entities", validateBody, s.GetRecognisers, s.Recognize)
	r.GET("/recognisers", s.ListRecognisers)
}

func (s server) ListRecognisers(c *gin.Context) {
	c.JSON(200, s.controller.ListRecognisers())
}

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

	recognisers := make(map[string]lib.RecogniserOptions, len(requestedRecognisers))
	for _, recogniser := range requestedRecognisers {
		recognisers[recogniser] = lib.RecogniserOptions{}

		header := c.GetHeader(fmt.Sprintf("x-%s", recogniser))
		if header == "" {
			continue
		}

		b, err := base64.StdEncoding.DecodeString(header)
		if err != nil {
			handleError(c, NewHttpError(400, errors.New("invalid request header - must be base64 encoded")))
			return
		}

		var opts lib.RecogniserOptions
		if err := json.Unmarshal(b, &opts); err != nil {
			handleError(c, NewHttpError(400, errors.New("invalid request header - must be valid json (base64 encoded)")))
			return
		}
		recognisers[recogniser] = opts
	}

	c.Set("recognisers", recognisers)
	c.Next()
}

func (s server) Recognize(c *gin.Context) {
	r, ok := c.Get("recognisers")
	if !ok {
		handleError(c, errors.New("recognisers are unset"))
	}

	recognisers := r.(map[string]lib.RecogniserOptions)

	contentType, ok := allowedContentTypeEnumMap[c.ContentType()]
	if !ok {
		handleError(c, NewHttpError(400, errors.New("invalid content type - must be text/html or text/plain")))
	}

	entities, err := s.controller.Recognize(c.Request.Body, contentType, recognisers)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, entities)
}

func (s server) Tokenize(c *gin.Context) {
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

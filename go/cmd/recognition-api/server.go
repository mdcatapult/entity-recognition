package main

import (
	"errors"
	"io"

	"github.com/gin-gonic/gin"
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
	r.POST("/html/text", validateBody, s.HTMLToText)
	r.POST("/html/tokens", validateBody, s.TokenizeHTML)
	r.POST("/html/entities", validateBody, s.RecognizeInHTML)
}

func (s server) RecognizeInHTML(c *gin.Context) {
	entities, err := s.controller.RecognizeInHTML(c.Request.Body)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, entities)
}

func (s server) TokenizeHTML(c *gin.Context) {
	tokens, err := s.controller.TokenizeHTML(c.Request.Body)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, tokens)
}

func (s server) HTMLToText(c *gin.Context) {
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
	case code < 500:
		c.JSON(code, map[string]interface{}{
			"status":  code,
			"message": err.Error(),
		})
		c.Abort()
	default:
		_ = c.AbortWithError(code, err)
	}
}

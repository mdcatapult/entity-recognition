package main

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
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
	r.POST("/html/text", s.HTMLToText)
	r.POST("/html/tokens", s.TokenizeHTML)
	r.POST("/html/entities", s.RecognizeInHTML)
}

func (s server) RecognizeInHTML(c *gin.Context) {
	if c.Request.Body == nil {
		handleError(c, NewHttpError(400, errors.New("request body missing")))
		return
	}

	entities, err := s.controller.RecognizeInHTML(c.Request.Body)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, entities)
}

func (s server) TokenizeHTML(c *gin.Context) {
	if c.Request.Body == nil {
		handleError(c, NewHttpError(400, errors.New("request body missing")))
		return
	}

	tokens, err := s.controller.TokenizeHTML(c.Request.Body)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, tokens)
}

func (s server) HTMLToText(c *gin.Context) {
	if c.Request.Body == nil {
		handleError(c, NewHttpError(400, errors.New("request body missing")))
		return
	}

	data, err := s.controller.HTMLToText(c.Request.Body)
	if err != nil {
		handleError(c, err)
		return
	}

	c.Data(200, "text/plain", data)
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
	err = c.AbortWithError(code, err)
	if err != nil {
		log.Error().Err(err).Send()
	}
}

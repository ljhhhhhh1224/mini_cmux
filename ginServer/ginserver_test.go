package ginServer

import (
	"net/http"
	"testing"

	"github.com/go-playground/assert/v2"

	"github.com/appleboy/gofight/v2"
)

func TestGet(t *testing.T) {
	r := gofight.New()
	r.GET("/get").Run(SetupRouter(), func(response gofight.HTTPResponse, request gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, response.Code)
	})
}

func TestStop(t *testing.T) {
	r := gofight.New()
	r.GET("/stop").Run(SetupRouter(), func(response gofight.HTTPResponse, request gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, response.Code)
	})
}

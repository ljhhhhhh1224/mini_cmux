package ginServer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGet(t *testing.T) {
	r := SetupRouter()
	Convey("Test gin handler /get", t, func() {
		req := httptest.NewRequest(
			"GET", "/get", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
		//assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		err := json.Unmarshal([]byte(w.Body.String()), &resp)
		So(err, ShouldBeNil)
		So(resp["message"], ShouldEqual, "get message successfully")
	})
}

func TestStop(t *testing.T) {
	r := SetupRouter()
	Convey("Test gin handler /stop", t, func() {
		req := httptest.NewRequest(
			"GET", "/stop", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
		var resp map[string]string
		err := json.Unmarshal([]byte(w.Body.String()), &resp)
		So(err, ShouldBeNil)
		So(resp["message"], ShouldEqual, "stop successfully")
	})
}

//func TestGet(t *testing.T) {
//	r := gofight.New()
//	r.GET("/get").Run(SetupRouter(), func(response gofight.HTTPResponse, request gofight.HTTPRequest) {
//		assert.Equal(t, http.StatusOK, response.Code)
//	})
//}

//func TestStop(t *testing.T) {
//	r := gofight.New()
//	r.GET("/stop").Run(SetupRouter(), func(response gofight.HTTPResponse, request gofight.HTTPRequest) {
//		assert.Equal(t, http.StatusOK, response.Code)
//	})
//}

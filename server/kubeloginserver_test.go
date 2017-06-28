package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSpecs(t *testing.T) {
	Convey("Kubelogin Server Tests", t, func() {
		testServer := httptest.NewServer(http.HandlerFunc(handler))
		response, _ := http.Get(testServer.URL)
		Convey("The server should get a 200 response", func() {
			So(response.StatusCode, ShouldEqual, 200)
		})
		Convey("The postHandler shouldn't be able to know who the source is", func() {
			resp, _ := http.Post(testServer.URL, "application/x-www-form-urlencoded", strings.NewReader("hello world"))
			b, _ := ioutil.ReadAll(resp.Body)
			So(string(b), ShouldEqual, "Unkown Post source :( \n")
		})
		Convey("The postHandler should know that the source is a client", func() {
			v := url.Values{}
			v.Set("post_from", "client")
			s := v.Encode()
			req, _ := http.NewRequest("POST", testServer.URL, strings.NewReader(s))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			c := &http.Client{}
			resp, _ := c.Do(req)
			data, _ := ioutil.ReadAll(resp.Body)
			So(string(data), ShouldEqual, "Post from client! r.PostForm = map[post_from:[client]]\n")
		})
		Convey("The get request should be hoopla", func() {
			response, _ := http.Get(testServer.URL)
			body, _ := ioutil.ReadAll(response.Body)
			So(string(body), ShouldEqual, "hoopla")
		})
	})
}

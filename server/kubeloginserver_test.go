package main

import (
	"bytes"
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSpecs(t *testing.T) {
	Convey("Kubelogin Server", t, func() {
		redirectTestServer := httptest.NewServer(http.HandlerFunc(cliGetRedirectHandler))
		responseTestServer := httptest.NewServer(http.HandlerFunc(responseHandler))
		cliGetTestServer := httptest.NewServer(http.HandlerFunc(cliGetHandler))
		authPostTestServer := httptest.NewServer(http.HandlerFunc(authPostHandler))
		authPostJwtTestServer := httptest.NewServer(http.HandlerFunc(authPostJwtHandler))
		Convey("The server should get a 200 response upon a successful URL", func() {
			response, _ := http.Get(responseTestServer.URL)
			So(response.StatusCode, ShouldEqual, 200)
		})
		Convey("The cliGetRedirectHandler should receive a status code 200 from the webpage after redirect", func() {
			url := redirectTestServer.URL + "/login/auth?clusterID=mycluster&port=8000"
			resp, _ := http.Get(url)
			resp.Body.Close()
			So(resp.StatusCode, ShouldEqual, 200)

		})
		Convey("The cliGetHandler should receive clientID and port from the CLI", func() {
			url := cliGetTestServer.URL + "/login/auth?clusterID=mycluster&port=8000"
			resp, _ := http.Get(url)
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			s := buf.String()
			resp.Body.Close()
			So(s, ShouldEqual, "mycluster,8000,127.0.0.1")
		})
		Convey("If the port is missing the cliGetHandler should return a 400 error", func() {
			url := cliGetTestServer.URL + "/login/auth?clusterID=myclient&port="
			resp, _ := http.Get(url)
			resp.Body.Close()
			So(resp.StatusCode, ShouldEqual, 400)
		})
		Convey("If the ID is missing the cliGetHandler should return a 400 error", func() {
			url := cliGetTestServer.URL + "/login/auth?clusterID=&port=8000"
			resp, _ := http.Get(url)
			resp.Body.Close()
			So(resp.StatusCode, ShouldEqual, 400)
		})

		/*Convey("The server should keep a map of clientID's to secrets for later validation", func() {
			response, _ := http.Get(testServer.URL)
			body, _ := ioutil.ReadAll(response.Body)
			So(string(body), ShouldEqual, "hoopla")
		})
		Convey("The server should keep a map of http session id's/local ports to confirm things are getting sent to right location", func() {
			response, _ := http.Get(testServer.URL)
			body, _ := ioutil.ReadAll(response.Body)
			So(string(body), ShouldEqual, "hoopla")
		})*/
		Convey("Server should listen for callback from the authZ server (assuming a POST)", func() {
			values := url.Values{}
			values.Set("authCode", "myauthcode")
			encode := values.Encode()
			resp, _ := http.Post(authPostTestServer.URL, "application/x-www-form-urlencoded", strings.NewReader(encode))
			b, _ := ioutil.ReadAll(resp.Body)
			result := string(b)
			resp.Body.Close()
			So(result, ShouldEqual, "good news everyone")
		})
		/*Convey("Server should receive the auth code from auth server POST", func() {
			response, _ := http.Get(testServer.URL)
			body, _ := ioutil.ReadAll(response.Body)
			So(string(body), ShouldEqual, "hoopla")
		})*/
		Convey("Server should make a POST request to authZ server containing clientID, clientSecret and Authcode", func() {
			response := postToAuthHandler("id", "secret", "authCode")
			So(response, ShouldEqual, nil)
		})
		Convey("Server should listen for the response by the authZ server containing the JWT", func() {
			values := url.Values{}
			values.Set("jwtToken", "myjwttoken")
			encode := values.Encode()
			resp, _ := http.Post(authPostJwtTestServer.URL, "application/x-www-form-urlencoded", strings.NewReader(encode))
			b, _ := ioutil.ReadAll(resp.Body)
			result := string(b)
			resp.Body.Close()
			So(result, ShouldEqual, "good news everyone")
		})
		Convey("Server should finally redirect JWT to CLI at CLI's local port", func() {
			response := postTokenToCliHandler("jwtToken")
			So(response, ShouldEqual, nil)
		})
	})
}

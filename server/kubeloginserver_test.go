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
		responseTestServer := httptest.NewServer(http.HandlerFunc(handler))
		cliPostTestServer := httptest.NewServer(http.HandlerFunc(cliPostHandler))
		authPostTestServer := httptest.NewServer(http.HandlerFunc(authPostHandler))
		authPostJwtTestServer := httptest.NewServer(http.HandlerFunc(authPostJwtHandler))
		Convey("The server should get a 200 response", func() {
			response, _ := http.Get(responseTestServer.URL)
			So(response.StatusCode, ShouldEqual, 200)
		})
		Convey("The cliPostHandler should receive clientID and port from the CLI", func() {
			values := url.Values{}
			values.Set("clientID", "myclient")
			values.Set("portNum", "8000")
			encode := values.Encode()
			resp, _ := http.Post(cliPostTestServer.URL, "application/x-www-form-urlencoded", strings.NewReader(encode))
			b, _ := ioutil.ReadAll(resp.Body)
			result := string(b)

			So(result, ShouldEqual, "myclient,8000")
		})
		Convey("If the port is missing the cliPostHandler should return a 400 error", func() {
			values := url.Values{}
			values.Set("clientID", "myclient")
			values.Set("portNum", "")
			encode := values.Encode()
			resp, _ := http.Post(cliPostTestServer.URL, "application/x-www-form-urlencoded", strings.NewReader(encode))
			b, _ := ioutil.ReadAll(resp.Body)
			result := string(b)

			So(result, ShouldEqual, "400 Bad request")
		})
		Convey("If the ID is missing the cliPostHandler should return a 400 error", func() {
			values := url.Values{}
			values.Set("clientID", "")
			values.Set("portNum", "8000")
			encode := values.Encode()
			resp, _ := http.Post(cliPostTestServer.URL, "application/x-www-form-urlencoded", strings.NewReader(encode))
			b, _ := ioutil.ReadAll(resp.Body)
			result := string(b)

			So(result, ShouldEqual, "400 Bad request")
		})
		Convey("Once the clientID is received, should verify if it exists", func() {
			response := verifyID("myclient")
			So(response, ShouldEqual, true)
		})
		Convey("If the clientID does not exist, return 404 error", func() {
			values := url.Values{}
			values.Set("clientID", "wrongclient")
			values.Set("portNum", "8000")
			encode := values.Encode()
			resp, _ := http.Post(cliPostTestServer.URL, "application/x-www-form-urlencoded", strings.NewReader(encode))
			b, _ := ioutil.ReadAll(resp.Body)
			err := string(b)

			So(err, ShouldEqual, "404 Not found")
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
			So(result, ShouldEqual, "good news everyone")
		})
		Convey("Server should finally redirect JWT to CLI at CLI's local port", func() {
			response := postTokenToCliHandler("jwtToken")
			So(response, ShouldEqual, nil)
		})
	})
}

package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coreos/go-oidc"
	. "github.com/smartystreets/goconvey/convey"
)

func incorrectURL(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "404 Page Not Found", http.StatusNotFound)
}

func TestServerSpecs(t *testing.T) {
	Convey("Kubelogin Server", t, func() {
		provider := &oidc.Provider{}
		authClient := authClientSetup("foo", "bar", provider)
		unitTestServer := httptest.NewServer(getMux(authClient))
		Convey("The incorrectURL handler should return a 404 if a user doesn't specify a path", func() {
			response, _ := http.Get(unitTestServer.URL)
			response.Body.Close()
			So(response.StatusCode, ShouldEqual, 404)
		})
		Convey("The cliHandleLogin function", func() {
			Convey("should get a status code 303 for a correct redirect", func() {
				url := unitTestServer.URL + "/login?port=8000"
				authClient.client = &http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
				}
				request, _ := http.NewRequest("GET", url, nil)
				resp, _ := authClient.client.Do(request)
				log.Print(resp.StatusCode)
				resp.Body.Close()
				So(resp.StatusCode, ShouldEqual, 303)
			})
			Convey("should return a 400 error if the port is missing", func() {
				url := unitTestServer.URL + "/login?port="
				resp, _ := http.Get(url)
				resp.Body.Close()
				So(resp.StatusCode, ShouldEqual, 400)
			})
		})
		Convey("callbackHandler", func() {
			Convey("should return a bad request if no code or state is in the url", func() {
				url := unitTestServer.URL + "/callback"
				request, _ := http.NewRequest("GET", url, nil)
				response, _ := authClient.client.Do(request)
				response.Body.Close()
				So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
			})
			Convey("should return a internal server error if the authcode is not valid", func() {
				fakeCodeURL := unitTestServer.URL + "/callback?code=asdf123&state=3000"
				request, _ := http.NewRequest("GET", fakeCodeURL, nil)
				response, _ := authClient.client.Do(request)
				response.Body.Close()
				So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})
		})
		Convey("the local listener should return a message saying that a jwt has been received", func() {
			resp, _ := http.Get(unitTestServer.URL + "/local")
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			So(string(bodyBytes), ShouldEqual, "got a jwt")
		})
		Convey("the redirect listener should return a message saying it's back at local", func() {
			resp, _ := http.Get(unitTestServer.URL + "/redirect")
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			So(string(bodyBytes), ShouldEqual, "back at local")
		})
	})
}

func TestGetField(t *testing.T) {
	Convey("getFields", t, func() {
		url := "localhost/callback?code=myawesomecode&state=8000"
		newReq, _ := http.NewRequest("GET", url, nil)

		Convey("returns the code field value from the url", func() {
			result := getField(newReq, authCodeField)
			So(result, ShouldEqual, "myawesomecode")
		})
		Convey("returns the state field value", func() {
			result := getField(newReq, stateField)
			So(result, ShouldEqual, "8000")
		})
		Convey("returns an empty string upon a missing field", func() {
			result := getField(newReq, "helloworld")
			So(result, ShouldEqual, "")
		})

	})
}

func TestVerifyJWT(t *testing.T) {
	Convey("verifyJwt", t, func() {
		testJwt := "https://claims.nordstrom.com/nauth/groups, https://claims.nordstrom.com/nauth/username, @nordstrom.com"
		testResult := verifyJWT(testJwt)
		Convey("should return true upon the username, usernameSpec, and groups fields being present", func() {
			So(testResult, ShouldEqual, true)
		})
		Convey("should return false if a field is missing", func() {
			testResult = verifyJWT("nordy")
			So(testResult, ShouldEqual, false)
		})
	})
}

func TestJwtToString(t *testing.T) {
	Convey("jwtToString", t, func() {
		var w http.ResponseWriter

		Convey("returns a string if a valid byte array is given", func() {
			testJwt := jwtToString([]byte{123, 34, 97, 108, 103, 34, 58, 34, 82, 83, 50, 53, 54, 34, 125}, w)
			So(testJwt, ShouldContainSubstring, "alg")
		})
		Convey("returns an EOF because there is a missing } as the delimiting byte", func() {
			failedJwt := jwtToString([]byte{123, 34, 97, 108, 103, 34, 58, 34, 82, 83, 50, 53, 54, 34}, w)
			So(failedJwt, ShouldEqual, "EOF")
		})
	})
}

func TestAuthClientSetup(t *testing.T) {
	Convey("authClientSetup", t, func() {
		provider := &oidc.Provider{}
		Convey("authClientSetup should return a serverApp struct with the clientid/secret, redirect URL, and defaultClient info filled in", func() {
			testClient := authClientSetup("foo", "bar", provider)
			correctID := testClient.clientID == "foo"
			correctSec := testClient.clientSecret == "bar"
			correctURI := testClient.redirectURI == "http://localhost:3000/callback"
			correctClient := testClient.client == http.DefaultClient
			overallCorrect := correctClient && correctID && correctSec && correctURI
			So(overallCorrect, ShouldEqual, true)
		})
	})
}

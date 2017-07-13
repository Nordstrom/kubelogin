package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coreos/go-oidc"
	. "github.com/smartystreets/goconvey/convey"
)

func getFieldsTest(w http.ResponseWriter, r *http.Request) {
	authCode := getField(r, "code")
	state := getField(r, "state")
	returnedItems := authCode + ", " + state
	fmt.Fprint(w, returnedItems)
}

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

		Convey("The cliHandleLogin should get a status code 303 for a correct redirect", func() {
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

			Convey("If the port is missing the cliHandleLogin should return a 400 error", func() {
				url := unitTestServer.URL + "/login?port="
				resp, _ := http.Get(url)
				resp.Body.Close()
				So(resp.StatusCode, ShouldEqual, 400)
			})
		})

		Convey("jwtChecker should return true upon the username, usernameSpec, and groups fields being present", func() {
			testJwt := "https://claims.nordstrom.com/nauth/groups, https://claims.nordstrom.com/nauth/username, @nordstrom.com"
			testResult := verifyJWT(testJwt)
			So(testResult, ShouldEqual, true)
			Convey("jwtChecker should fail if the groups field is missing", func() {
				testJwt = "https://claims.nordstrom.com/nauth/, https://claims.nordstrom.com/nauth/username, @nordstrom.com"
				testResult = verifyJWT(testJwt)
				So(testResult, ShouldEqual, false)
				Convey("jwtChecker should fail if the username field is missing", func() {
					testJwt = "https://claims.nordstrom.com/nauth/groups, https://claims.nordstrom.com/nauth/, @nordstrom.com"
					testResult = verifyJWT(testJwt)
					So(testResult, ShouldEqual, false)
					Convey("jwtChecker should fail if the usernameSpec is missing/incorrect", func() {
						testJwt = "https://claims.nordstrom.com/nauth/groups, https://claims.nordstrom.com/nauth/, @nordy.com"
						testResult = verifyJWT(testJwt)
						So(testResult, ShouldEqual, false)
					})
				})
			})
		})
		Convey("authClientSetup should return a serverApp struct with the clientid/secret, redirect URL, and defaultClient info filled in", func() {
			provider := &oidc.Provider{}
			testClient := authClientSetup("foo", "bar", provider)
			correctID := testClient.clientID == "foo"
			correctSec := testClient.clientSecret == "bar"
			correctURI := testClient.redirectURI == "http://localhost:3000/callback"
			correctClient := testClient.client == http.DefaultClient
			overallCorrect := correctClient && correctID && correctSec && correctURI
			So(overallCorrect, ShouldEqual, true)
		})
		Convey("the local listener should return a message saying that a jwt has been received", func() {
			resp, _ := http.Get(unitTestServer.URL + "/local")
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			So(string(bodyBytes), ShouldEqual, "got a jwt")
		})
		Convey("Test that the jwtToString function returns a string we can search", func() {
			var w http.ResponseWriter
			claims := []byte{123, 34, 97, 108, 103, 34, 58, 34, 82, 83, 50, 53, 54, 34, 125}
			testJwt := jwtToString(claims, w)
			So(testJwt, ShouldContainSubstring, "alg")

			Convey("Test that the jwtToString function fails because there is a missing } as the delimiting byte", func() {
				claims = []byte{123, 34, 97, 108, 103, 34, 58, 34, 82, 83, 50, 53, 54, 34}
				failedJwt := jwtToString(claims, w)
				So(failedJwt, ShouldEqual, "EOF")
			})
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

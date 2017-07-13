package main

import (
	"context"
	"fmt"
	"github.com/coreos/go-oidc"
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
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

func TestSpecs(t *testing.T) {
	Convey("Kubelogin Server", t, func() {

		var app authOClient
		app.clientID = "myawesomeid"
		app.clientSecret = "myawesomesecret"
		app.redirectURI = "http://localhost:3000/callback"
		app.client = http.DefaultClient
		contxt := oidc.ClientContext(context.Background(), app.client)
		provider, _ := oidc.NewProvider(contxt, "https://nauth-test.auth0.com/")
		app.provider = provider
		//app.verifier = provider.Verifier(&oidc.Config{ClientID: app.clientID})

		unitTestServer := httptest.NewServer(getMux(app))
		Convey("The incorrectURL handler should return a 404 if a user doesn't specify a path", func() {
			response, _ := http.Get(unitTestServer.URL)
			response.Body.Close()
			So(response.StatusCode, ShouldEqual, 404)
		})

		Convey("The cliHandleLogin should get a status code 303 for a correct redirect", func() {
			url := unitTestServer.URL + "/login?port=8000"
			app.client = &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}
			request, _ := http.NewRequest("GET", url, nil)
			resp, _ := app.client.Do(request)
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
		Convey("getFields", func() {
			url := unitTestServer.URL + "/callback?code=myawesomecode&state=8000"
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
			testClient := authClientSetup()
			correctID := testClient.clientID == os.Getenv("CLIENT_ID")
			correctSec := testClient.clientSecret == os.Getenv("CLIENT_SEC")
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

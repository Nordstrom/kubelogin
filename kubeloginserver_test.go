package main

import (
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
		authClient := newAuthClient("foo", "bar", "redirect", provider)
		unitTestServer := httptest.NewServer(getMux(authClient))
		Convey("The incorrectURL handler should return a 404 if a user doesn't specify a path", func() {
			response, _ := http.Get(unitTestServer.URL)
			response.Body.Close()
			So(response.StatusCode, ShouldEqual, 404)
		})
		Convey("The handleCliLogin function", func() {
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
		Convey("exchangeHandler", func() {
			Convey("should return a internal server error due to dealing with redis offline", func() {
				err := makeRedisClient()
				log.Print(err)
				exchangeURL := unitTestServer.URL + "/exchange?token=hoopla"
				response, _ := http.Get(exchangeURL)
				response.Body.Close()
				So(response.StatusCode, ShouldEqual, http.StatusUnauthorized)
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

func TestMakeRedisClient(t *testing.T) {
	Convey("makeRedisClient", t, func() {
		Convey("should fail since it cant find the environment variable holding the address", func() {
			err := makeRedisClient()
			So(err, ShouldNotEqual, nil)
		})
	})
}

func TestGenerateToken(t *testing.T) {
	Convey("generateToken", t, func() {
		Convey("should pass since we are just returning a string", func() {
			token, _ := generateToken("hoopla")
			So(token, ShouldNotEqual, nil)
		})
	})
}

func TestExchangeToken(t *testing.T) {
	Convey("exchangeToken", t, func() {
		Convey("should error out since we can't access the redis cache offline", func() {
			_, err := exchangeToken("hoopla")
			So(err, ShouldNotEqual, nil)
		})
	})
}

func TestGenerateSendBackURL(t *testing.T) {
	Convey("generateSendBackURL", t, func() {
		Convey("should pass since we will encounter errors when trying to add our value to redis", func() {
			_, err := generateSendBackURL("hoopla", "3000")
			log.Printf("The err is %s", err)
			So(err, ShouldNotEqual, nil)
		})
	})
}

func TestNewAuthClient(t *testing.T) {
	Convey("authClientSetup", t, func() {
		provider := &oidc.Provider{}
		Convey("authClientSetup should return a serverApp struct with the clientid/secret, redirect URL, and defaultClient info filled in", func() {
			testClient := newAuthClient("foo", "bar", "redirect", provider)
			correctID := testClient.clientID == "foo"
			correctSec := testClient.clientSecret == "bar"
			correctURI := testClient.redirectURI == "redirect"
			correctClient := testClient.client == http.DefaultClient

			overallCorrect := correctClient && correctID && correctSec && correctURI
			So(overallCorrect, ShouldEqual, true)
		})
	})
}

func TestHealthHandler(t *testing.T) {
	Convey("healthHandler", t, func() {
		provider := &oidc.Provider{}
		authClient := newAuthClient("foo", "bar", "redirect", provider)
		unitTestServer := httptest.NewServer(getMux(authClient))
		Convey("Should write back to the response writer a statusOK", func() {
			resp, _ := http.Get(unitTestServer.URL + "/health")
			So(resp.StatusCode, ShouldEqual, 200)

		})
	})
}

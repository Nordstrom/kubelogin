package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coreos/go-oidc"
	. "github.com/smartystreets/goconvey/convey"
)

func TestServerSpecs(t *testing.T) {
	Convey("Kubelogin Server", t, func() {
		app := setAppMemberFields("redisFoo", "redisBar", "redisFoobar", "foo", "bar", "redirect", &oidc.Provider{}, "fooGroups", "barUsers")
		unitTestServer := httptest.NewServer(getMux(app, "/downoad"))
		Convey("The handleCLILogin function", func() {
			Convey("should get a status code 303 for a correct redirect", func() {
				url := unitTestServer.URL + "/login?port=8000"
				app.authClient.client = &http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
				}
				request, _ := http.NewRequest("GET", url, nil)
				resp, _ := app.authClient.client.Do(request)
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
				response, _ := app.authClient.client.Do(request)
				response.Body.Close()
				So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
			})
			Convey("should return a internal server error if the authcode is not valid", func() {
				fakeCodeURL := unitTestServer.URL + "/callback?code=asdf123&state=3000"
				request, _ := http.NewRequest("GET", fakeCodeURL, nil)
				response, _ := app.authClient.client.Do(request)
				response.Body.Close()
				So(response.StatusCode, ShouldEqual, http.StatusInternalServerError)
			})
		})
		Convey("defaultHandler", func() {
			Convey("should have basic html written on the page i.e., body is not nil", func() {
				request, _ := http.NewRequest("GET", unitTestServer.URL, nil)
				response, _ := app.authClient.client.Do(request)
				response.Body.Close()
				So(response.Body, ShouldNotEqual, nil)
			})
			Convey("should return a 200 status code upon a successful connection", func() {
				response, _ := http.Get(unitTestServer.URL)
				So(response.StatusCode, ShouldEqual, 200)
			})
		})
		Convey("exchangeHandler", func() {
			Convey("should return a internal server error due to Redis not being available", func() {
				app.redisValues.makeRedisClient()
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
		app := setAppMemberFields("redisFoo", "redisBar", "redisFoobar", "foo", "bar", "redirect", &oidc.Provider{}, "fooGroups", "barUsers")
		Convey("should fail since no Redis address environment variable was set", func() {
			err := app.redisValues.makeRedisClient()
			So(err, ShouldNotEqual, nil)
		})
	})
}

func TestGenerateToken(t *testing.T) {
	Convey("generateToken", t, func() {
		app := setAppMemberFields("redisFoo", "redisBar", "redisFoobar", "foo", "bar", "redirect", &oidc.Provider{}, "fooGroups", "barUsers")
		Convey("should pass since we are just returning a string", func() {
			token, _ := app.redisValues.generateToken("hoopla")
			So(token, ShouldNotEqual, nil)
		})
	})
}

func TestFetchJWTForToken(t *testing.T) {
	Convey("fetchJWTForToken", t, func() {
		app := setAppMemberFields("redisFoo", "redisBar", "redisFoobar", "foo", "bar", "redirect", &oidc.Provider{}, "fooGroups", "barUsers")
		app.redisValues.makeRedisClient()
		Convey("should error out since we can't access the Redis cache offline", func() {
			_, err := app.redisValues.fetchJWTForToken("hoopla")
			So(err, ShouldNotEqual, nil)
		})
	})
}

func TestGenerateSendBackURL(t *testing.T) {
	Convey("generateSendBackURL", t, func() {
		app := setAppMemberFields("redisFoo", "redisBar", "redisFoobar", "foo", "bar", "redirect", &oidc.Provider{}, "fooGroups", "barUsers")
		Convey("should pass since we will encounter errors when trying to add our value to Redis", func() {
			_, err := app.redisValues.generateSendBackURL("hoopla", "3000")
			log.Printf("The err is %s", err)
			So(err, ShouldNotEqual, nil)
		})
	})
}

func TestNewAuthClient(t *testing.T) {
	Convey("authClientSetup", t, func() {
		provider := &oidc.Provider{}
		Convey("authClientSetup should return a serverApp struct with the clientid/secret, redirect URL, and defaultClient info filled in", func() {
			testClient := newAuthClient("foo", "bar", "redirect", provider, "fooGroups", "fooUsers")
			correctID := testClient.clientID == "foo"
			correctSec := testClient.clientSecret == "bar"
			correctURI := testClient.redirectURI == "redirect"
			correctClient := testClient.client == http.DefaultClient
			correctUser := testClient.userClaim == "fooUsers"
			correctGroup := testClient.groupsClaim == "fooGroups"
			overallCorrect := correctClient && correctID && correctSec && correctURI && correctUser && correctGroup
			So(overallCorrect, ShouldEqual, true)
		})
	})
}

func TestHealthHandler(t *testing.T) {
	Convey("healthHandler", t, func() {
		app := setAppMemberFields("redisFoo", "redisBar", "redisFoobar", "foo", "bar", "redirect", &oidc.Provider{}, "fooGroups", "barUsers")
		unitTestServer := httptest.NewServer(getMux(app, "/download"))
		Convey("Should write back to the response writer a statusOK", func() {
			resp, _ := http.Get(unitTestServer.URL + "/health")
			So(resp.StatusCode, ShouldEqual, 200)

		})
	})
}

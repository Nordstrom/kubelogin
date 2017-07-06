package main

import (
	"context"
	"github.com/coreos/go-oidc"
	. "github.com/smartystreets/goconvey/convey"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	"testing"
)

func TestSpecs(t *testing.T) {
	Convey("Kubelogin Server", t, func() {

		var app serverApp
		app.clientID = os.Getenv("CLIENT_ID")
		app.clientSecret = os.Getenv("CLIENT_SECRET")
		app.redirectURI = "http://localhost:3000/callback"
		app.client = http.DefaultClient
		contxt := oidc.ClientContext(context.Background(), app.client)
		provider, _ := oidc.NewProvider(contxt, "https://nauth-test.auth0.com/")
		app.provider = provider
		app.verifier = provider.Verifier(&oidc.Config{ClientID: app.clientID})

		//callbackTest := httptest.NewServer(http.HandlerFunc(app.callbackHandler))
		redirectTestServer := httptest.NewServer(http.HandlerFunc(cliGetRedirectHandler))
		responseTestServer := httptest.NewServer(http.HandlerFunc(responseHandler))
		cliGetTestServer := httptest.NewServer(http.HandlerFunc(app.handleCliLogin))

		Convey("The server should get a 200 response upon a successful URL", func() {
			response, _ := http.Get(responseTestServer.URL)
			So(response.StatusCode, ShouldEqual, 200)
		})
		Convey("The cliGetRedirectHandler should receive a status code 301 from the webpage after redirect", func() {
			url := redirectTestServer.URL + "/login/port?port=8000"
			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}
			request, _ := http.NewRequest("GET", url, nil)
			resp, _ := client.Do(request)
			log.Print(resp.Request.URL.String())
			resp.Body.Close()
			//log.Print(resp.StatusCode)
			So(resp.StatusCode, ShouldEqual, 301)

		})
		Convey("The cliHandleLogin should get a status code 200 for a correct redirect", func() {
			url := cliGetTestServer.URL + "/login/port?port=8000"
			resp, _ := http.Get(url)
			So(resp.StatusCode, ShouldEqual, 200)
		})
		Convey("If the port is missing the cliHandleLogin should return a 400 error", func() {
			url := cliGetTestServer.URL + "/login/port?port="
			resp, _ := http.Get(url)
			resp.Body.Close()
			So(resp.StatusCode, ShouldEqual, 400)
		})

		Convey("Server should finally redirect JWT to CLI at CLI's local port", func() {
			response := postTokenToCliHandler("jwtToken")
			So(response, ShouldEqual, nil)
		})
	})
}

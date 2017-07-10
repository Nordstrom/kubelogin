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
	"testing"
)

func callbackItems(w http.ResponseWriter, r *http.Request) {
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

		var app serverApp
		app.clientID = "myawesomeid"
		app.clientSecret = "myawesomesecret"
		app.redirectURI = "http://localhost:3000/callback"
		app.client = http.DefaultClient
		contxt := oidc.ClientContext(context.Background(), app.client)
		provider, _ := oidc.NewProvider(contxt, "https://nauth-test.auth0.com/")
		app.provider = provider
		app.verifier = provider.Verifier(&oidc.Config{ClientID: app.clientID})

		//callbackTest := httptest.NewServer(http.HandlerFunc(app.callbackHandler))
		incorrectPathServer := httptest.NewServer(http.HandlerFunc(incorrectURL))
		cliGetTestServer := httptest.NewServer(http.HandlerFunc(app.handleCliLogin))
		callbackItemsTestServer := httptest.NewServer(http.HandlerFunc(callbackItems))

		Convey("The incorrectURL handler should return a 404 if a user doesn't specify a path", func() {
			response, _ := http.Get(incorrectPathServer.URL)
			response.Body.Close()
			So(response.StatusCode, ShouldEqual, 404)
		})

		Convey("The cliHandleLogin should get a status code 303 for a correct redirect", func() {
			url := cliGetTestServer.URL + "/login/port?port=8000"
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
		})
		Convey("If the port is missing the cliHandleLogin should return a 400 error", func() {
			url := cliGetTestServer.URL + "/login/port?port="
			resp, _ := http.Get(url)
			resp.Body.Close()
			So(resp.StatusCode, ShouldEqual, 400)
		})

		Convey("getFields should read correct field values", func() {
			url := callbackItemsTestServer.URL + "/callback?code=myawesomecode&state=8000"
			resp, _ := http.Get(url)
			responseData, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			bodyString := string(responseData)
			So(bodyString, ShouldEqual, "myawesomecode, 8000")
		})
		Convey("getFields should fail to read missing code field", func() {
			url := callbackItemsTestServer.URL + "/callback?code=&state=8000"
			resp, _ := http.Get(url)
			responseData, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			bodyString := string(responseData)
			So(bodyString, ShouldEqual, ", 8000")
		})
		Convey("getFields should fail to read missing state field", func() {
			url := callbackItemsTestServer.URL + "/callback?code=myawesomecode&state="
			resp, _ := http.Get(url)
			responseData, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			bodyString := string(responseData)
			So(bodyString, ShouldEqual, "myawesomecode, ")
		})
	})
}

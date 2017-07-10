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
	"net/url"
	"strings"
	"testing"
)

func dummyVerify(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	value, _ := url.ParseQuery(string(body))
	if value.Get("jwt") != "" {
		fmt.Fprint(w, "valid jwt")
	} else {
		fmt.Fprint(w, "not a valid jwt")
	}
}

func dummyExchange(w http.ResponseWriter, r *http.Request) {
	//r.ParseForm()
	body, _ := ioutil.ReadAll(r.Body)
	values, _ := url.ParseQuery(string(body))
	if values.Get("clientID") != "" && values.Get("clientSecret") != "" && values.Get("authCode") != "" {
		fmt.Fprint(w, "{id_token: myawesomeJWT}")
	} else {
		fmt.Fprint(w, "Error! Error!")
	}

}

func callbackItems(w http.ResponseWriter, r *http.Request) {
	authCode := r.FormValue("code")
	state := r.FormValue("state")
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
		exchangeTestServer := httptest.NewServer(http.HandlerFunc(dummyExchange))
		verifyTestServer := httptest.NewServer(http.HandlerFunc(dummyVerify))
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

		Convey("Upon logging into auth page, callbackItems should return the authCode and state given in the url", func() {
			url := callbackItemsTestServer.URL + "/callback?code=myawesomecode&state=myawesomestate"
			resp, _ := http.Get(url)
			responseData, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			bodyString := string(responseData)
			So(bodyString, ShouldEqual, "myawesomecode, myawesomestate")
		})

		Convey("After the authCode callback, should exchange authCode and other items to get a token with an id_token field containing a raw jwt", func() {
			form := url.Values{}
			form.Add("clientID", app.clientID)
			form.Add("clientSecret", app.clientSecret)
			form.Add("authCode", "myawesomeauthcode")
			resp, _ := http.Post(exchangeTestServer.URL, "application/x-www-form-encoded", strings.NewReader(form.Encode()))
			responseData, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			bodyString := string(responseData)
			So(bodyString, ShouldEqual, "{id_token: myawesomeJWT}")
		})

		Convey("After the authCode callback, should exchange authCode and other items to get a token with an id_token field containing a raw jwt unless a value is missing", func() {
			form := url.Values{}
			form.Add("clientID", app.clientID)
			form.Add("authCode", app.clientSecret)
			resp, _ := http.Post(exchangeTestServer.URL, "application/x-www-form-encoded", strings.NewReader(form.Encode()))
			responseData, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			bodyString := string(responseData)
			So(bodyString, ShouldEqual, "Error! Error!")
		})

		Convey("After getting the jwt, it should be verified and be valid", func() {
			form := url.Values{}
			form.Add("jwt", "myawesomejwt")
			resp, _ := http.Post(verifyTestServer.URL, "application/x-www-form-encoded", strings.NewReader(form.Encode()))
			responseData, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			bodyString := string(responseData)
			So(bodyString, ShouldEqual, "valid jwt")
		})

		Convey("After getting the jwt, if it's not valid then an error occurs", func() {
			form := url.Values{}
			form.Add("jwt", "")
			resp, _ := http.Post(verifyTestServer.URL, "application/x-www-form-encoded", strings.NewReader(form.Encode()))
			responseData, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			bodyString := string(responseData)
			So(bodyString, ShouldEqual, "not a valid jwt")
		})

		Convey("Server should finally redirect JWT to CLI at CLI's local port", func() {
			response := postTokenToCliHandler("jwtToken", "8000")
			So(response, ShouldEqual, nil)
		})
	})
}

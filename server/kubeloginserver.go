package main

//handlelogin with dex handles the get & redirectURL
//handle callback handles the get from the auth server, gets the code, exchanges for the jwt.
//provider is the n-auth link minus the login

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type serverApp struct {
	/*
	   struct that contains necessary oauth/oidc information
	*/
	clientID     string
	clientSecret string
	redirectURI  string
	verifier     *oidc.IDTokenVerifier
	provider     *oidc.Provider
	client       *http.Client
}

func (app *serverApp) oauth2Config(scopes []string) *oauth2.Config {
	/*
	   the config for oauth2, scopes contain info we want back from the auth server
	*/
	return &oauth2.Config{
		ClientID:     app.clientID,
		ClientSecret: app.clientSecret,
		Endpoint:     app.provider.Endpoint(),
		Scopes:       scopes,
		RedirectURL:  app.redirectURI,
	}
}

func (app *serverApp) handleCliLogin(w http.ResponseWriter, r *http.Request) {
	/*
	   handles the get request from the client clicking the link they receive from the CLI
	   this will grab the port and sets it as the state for later use
	   we set the scopes to be openid, username, and groups so we get a jwt later with the needed info
	   we then redirect to the login page with the necessary info.
	*/
	portState := r.FormValue("port")
	if portState == "" {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
	var scopes []string
	scopes = append(scopes, "openid", " https://claims.nordstrom.com/nauth/username ", " https://claims.nordstrom.com/nauth/groups ")
	authCodeURL := app.oauth2Config(scopes).AuthCodeURL(portState)
	http.Redirect(w, r, authCodeURL, http.StatusSeeOther)
}

func getField(r *http.Request, fieldName string) string {
	/*
	   used to grab the field from the callback request
	*/
	if r.FormValue(fieldName) != "" {
		log.Print("fieldname: [" + r.FormValue(fieldName) + "]")
		return r.FormValue(fieldName)
	}
	return ""
}

func jwtToString(claims json.RawMessage, w http.ResponseWriter) string {
	/*
	   converts the jwt from bytes to a readable string
	*/
	buff := new(bytes.Buffer)
	json.Indent(buff, []byte(claims), "", "  ")
	jwt, err := buff.ReadString('}')
	if err != nil {
		log.Print(err)
		http.Error(w, fmt.Sprintf("Failed to transribe claims into string"), http.StatusInternalServerError)
		return ""
	}
	return jwt
}

func jwtChecker(jwt string) bool {
	/*
	   checks to make sure the jwt contains necessary info to send back to the client
	*/
	groups := strings.Contains(jwt, "groups")
	username := strings.Contains(jwt, "username")
	validUsername := strings.Contains(jwt, "@nordstrom.com")
	log.Print(groups, validUsername, username)
	if groups && username && validUsername {
		return true
	}
	return false
}

func (app *serverApp) callbackHandler(w http.ResponseWriter, r *http.Request) {
	/*
	   handles the callback from the auth server, exchanges the authcode, clientID, clientSecret for a rawToken which holds an id_token
	   field that has the JWT. Upon verification of the jwt, we pull the claims out which is the info that is needed to send back to the client
	*/
	var (
		err   error
		token *oauth2.Token
	)
	contxt := oidc.ClientContext(r.Context(), app.client)

	oauth2Config := app.oauth2Config(nil)
	authCode := getField(r, "code")
	port := getField(r, "state")
	if authCode == "" || port == "" {
		http.Error(w, "400: Bad Request", http.StatusBadRequest)
	}

	token, err = oauth2Config.Exchange(contxt, authCode)
	if err != nil {
		log.Print(err)
		http.Error(w, fmt.Sprintf("failed to get token: %v", err), http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token in token response", http.StatusInternalServerError)
		return
	}

	idToken, err := app.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to verify ID token"), http.StatusInternalServerError)
		log.Print(err)
		return
	}

	var claims json.RawMessage
	if claimErr := idToken.Claims(&claims); err != nil {
		log.Print(claimErr)
		http.Error(w, fmt.Sprintf("Failed to get claims from JWT"), http.StatusInternalServerError)
		return
	}

	jwt := jwtToString(claims, w)
	validData := jwtChecker(jwt)
	if validData {
		//sendBack(w, r, jwt, port)
		log.Print(jwt)
		log.Print(port)
	} else {
		log.Print(jwt)
		http.Error(w, fmt.Sprintf("Jwt does not contain necessary data"), http.StatusInternalServerError)
		return
	}

}

func sendBack(w http.ResponseWriter, r *http.Request, jwt string, port string) {
	/*
	   this will take the jwt and pass it back to the client using the port given earlier and lastly redirect to the clients localhost
	*/
	form := url.Values{}
	form.Add("jwt", jwt)
	url := "localhost:" + port
	resp, err := http.Post(url, "application/x-www-form-encoded", strings.NewReader(form.Encode()))
	if resp.StatusCode != 200 || err != nil {
		http.Error(w, "Couldnt post to url: ["+url+"]", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func main() {
	/*
				   sets up a new mux. upon a user clicking the link to our server, it will be handled by the handleLogin function.
			       When the auth server posts to our server it will be controlled by the callbackHandler. This is also initial setup for
	               the struct to contain necessary information
	*/
	var app serverApp
	app.clientID = os.Getenv("CLIENT_ID")
	app.clientSecret = os.Getenv("CLIENT_SEC")
	app.redirectURI = "http://localhost:3000/callback"
	app.client = http.DefaultClient
	contxt := oidc.ClientContext(context.Background(), app.client)
	provider, err := oidc.NewProvider(contxt, "https://nauth-test.auth0.com/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err.Error())
	}
	app.provider = provider
	app.verifier = provider.Verifier(&oidc.Config{ClientID: app.clientID})

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", app.callbackHandler)
	mux.HandleFunc("/login/", app.handleCliLogin)
	if err := http.ListenAndServe(":3000", mux); err != nil {
		log.Fatal(err)
	}
}

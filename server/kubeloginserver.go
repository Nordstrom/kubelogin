package main

//handlelogin with dex handles the get & redirectURL
//handle callback handles the get from the auth server, gets the code, exchanges for the jwt.
//provider is the n-auth link minus the login

import (
	"context"
	"fmt"
	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"os"
)

type serverApp struct {
	clientID     string
	clientSecret string
	redirectURI  string
	verifier     *oidc.IDTokenVerifier
	provider     *oidc.Provider
	client       *http.Client
}

func (app *serverApp) oauth2Config(scopes []string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     app.clientID,
		ClientSecret: app.clientSecret,
		Endpoint:     app.provider.Endpoint(),
		Scopes:       scopes,
		RedirectURL:  app.redirectURI,
	}
}

func responseHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello world")
}

func cliGetRedirectHandler(w http.ResponseWriter, r *http.Request) {
	redirectURL := "https://nauth-test.auth0.com/login?client=" + os.Getenv("CLIENT_ID")
	http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)

}

func (app *serverApp) handleCliLogin(w http.ResponseWriter, r *http.Request) {
	portState := r.FormValue("port")
	if portState == "" {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
	var scopes []string
	scopes = append(scopes, "groups", "email", "password")
	authCodeURL := app.oauth2Config(scopes).AuthCodeURL(portState)
	log.Print(authCodeURL)
	http.Redirect(w, r, authCodeURL, http.StatusSeeOther)
}

func (app *serverApp) callbackHandler(w http.ResponseWriter, r *http.Request) {
	var (
		err   error
		token *oauth2.Token
	)
	contxt := oidc.ClientContext(r.Context(), app.client)

	log.Print(contxt)
	oauth2Config := app.oauth2Config(nil)
	authCode := r.FormValue("code")

	log.Print(r.FormValue("state"))
	log.Print(authCode)
	/*form := url.Values{}
	form.Add("grant_type", "authorization_code")
	form.Add("client_id", app.clientID)
	form.Add("client_secret", app.clientSecret)
	form.Add("redirect_uri", app.redirectURI)
	req, err := http.NewRequest("POST", "https://nauth-test.auth0.com/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		log.Print(err)
		return
	}
	resp, err := app.client.Do(req)
	if err != nil {
		log.Print(err)
	}
	log.Print(resp)*/
	token, err = oauth2Config.Exchange(contxt, authCode)

	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get token: %v", err), http.StatusInternalServerError)
		return
	}
	log.Print(token)
	rawIDToken, ok := token.Extra("access_token").(string)
	log.Print(rawIDToken)
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
	log.Print(idToken)
}

func postTokenToCliHandler(jwtToken string) error {
	//will be easier to implement with a base cli created to communicate with this server
	fmt.Print("send back to the client")
	return nil
}

func main() {
	/*
			   sets up a new mux. upon a user clicking the link to our server, it will be handled by the cliGetHandler.
		       When the auth server posts to our server it should be controlled by the authPostHandler.
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

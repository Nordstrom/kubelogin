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
	app.clientSecret = os.Getenv("CLIENT_SECRET")
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

var htmlStr = `
html stuff
`

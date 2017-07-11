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
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type serverSideClient struct {
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

func (authClient *serverSideClient) oauth2Config(scopes []string) *oauth2.Config {
	/*
	   the config for oauth2, scopes contain info we want back from the auth server
	*/
	return &oauth2.Config{
		ClientID:     authClient.clientID,
		ClientSecret: authClient.clientSecret,
		Endpoint:     authClient.provider.Endpoint(),
		Scopes:       scopes,
		RedirectURL:  authClient.redirectURI,
	}
}

func (authClient *serverSideClient) handleCliLogin(writer http.ResponseWriter, request *http.Request) {
	/*
	   handles the get request from the client clicking the link they receive from the CLI
	   this will grab the port and sets it as the state for later use
	   we set the scopes to be openid, username, and groups so we get a jwt later with the needed info
	   we then redirect to the login page with the necessary info.
	*/
	portState := request.FormValue("port")
	if portState == "" {
		http.Error(writer, "400 Bad Request", http.StatusBadRequest)
		return
	}
	var scopes []string
	scopes = append(scopes, "openid", " https://claims.nordstrom.com/nauth/username ", " https://claims.nordstrom.com/nauth/groups ")
	authCodeURL := authClient.oauth2Config(scopes).AuthCodeURL(portState)
	http.Redirect(writer, request, authCodeURL, http.StatusSeeOther)
}

func getField(request *http.Request, fieldName string) string {
	/*
	   used to grab the field from the callback request
	*/
	if request.FormValue(fieldName) != "" {
		log.Print("fieldname: [" + request.FormValue(fieldName) + "]")
		return request.FormValue(fieldName)
	}
	return ""
}

func jwtToString(claims json.RawMessage, writer http.ResponseWriter) string {
	/*
	   converts the jwt from bytes to a readable string
	*/
	buff := new(bytes.Buffer)
	json.Indent(buff, []byte(claims), "", "  ")
	jwt, err := buff.ReadString('}')
	if err != nil {
		log.Print(err)
		http.Error(writer, fmt.Sprintf("Failed to transribe claims into string"), http.StatusInternalServerError)
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

func (authClient *serverSideClient) callbackHandler(writer http.ResponseWriter, request *http.Request) {
	/*
	   handles the callback from the auth server, exchanges the authcode, clientID, clientSecret for a rawToken which holds an id_token
	   field that has the JWT. Upon verification of the jwt, we pull the claims out which is the info that is needed to send back to the client
	*/
	var (
		err   error
		token *oauth2.Token
	)
	contxt := oidc.ClientContext(request.Context(), authClient.client)

	oauth2Config := authClient.oauth2Config(nil)
	authCode := getField(request, "code")
	port := getField(request, "state")
	if authCode == "" || port == "" {
		http.Error(writer, "400: Bad Request", http.StatusBadRequest)
	}

	token, err = oauth2Config.Exchange(contxt, authCode)
	if err != nil {
		log.Print(err)
		http.Error(writer, fmt.Sprintf("failed to get token: %v", err), http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(writer, "no id_token in token response", http.StatusInternalServerError)
		return
	}

	idToken, err := authClient.verifier.Verify(request.Context(), rawIDToken)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Failed to verify ID token"), http.StatusInternalServerError)
		log.Print(err)
		return
	}

	var claims json.RawMessage
	if claimErr := idToken.Claims(&claims); err != nil {
		log.Print(claimErr)
		http.Error(writer, fmt.Sprintf("Failed to get claims from JWT"), http.StatusInternalServerError)
		return
	}

	jwt := jwtToString(claims, writer)
	validData := jwtChecker(jwt)
	if validData {
		log.Print("about to sendback")
		sendBack(writer, request, jwt, port)
		return
	}
	log.Print(jwt)
	http.Error(writer, fmt.Sprintf("Jwt does not contain necessary data"), http.StatusInternalServerError)
	return

}

func sendBack(writer http.ResponseWriter, request *http.Request, jwt string, port string) {
	/*
	   this will take the jwt and pass it back to the client using the port given earlier and lastly redirect to the clients localhost
	*/
	form := url.Values{}
	form.Add("jwt", jwt)
	url := "http://localhost:" + port + "/local"
	log.Print("going to sendBack to this url: ", url)
	resp, err := http.Post(url, "application/x-www-form-encoded", strings.NewReader(form.Encode()))
	if resp.StatusCode != 200 || err != nil {
		http.Error(writer, "Couldnt post to url: ["+url+"]", http.StatusBadRequest)
		return
	}
	http.Redirect(writer, request, url, http.StatusSeeOther)
}

func localListener(writer http.ResponseWriter, request *http.Request) {
	//this belongs on CLI side but for testing purposes will be here
	request.ParseForm()
	body, _ := ioutil.ReadAll(request.Body)
	log.Print(string(body))
	fmt.Fprint(writer, "received a request")
}

func authClientSetup() serverSideClient {
	var authClient serverSideClient
	authClient.clientID = os.Getenv("CLIENT_ID")
	authClient.clientSecret = os.Getenv("CLIENT_SEC")
	authClient.redirectURI = "http://localhost:3000/callback"
	authClient.client = http.DefaultClient
	contxt := oidc.ClientContext(context.Background(), authClient.client)
	provider, err := oidc.NewProvider(contxt, "https://nauth-test.auth0.com/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err.Error())
	}
	authClient.provider = provider
	authClient.verifier = provider.Verifier(&oidc.Config{ClientID: authClient.clientID})
	return authClient
}

func main() {
	/*
		sets up a new mux. upon a user clicking the link to our server, it will be handled by the handleLogin function.
		When the auth server posts to our server it will be controlled by the callbackHandler. This is also initial setup for
		the struct to contain necessary information
	*/

	authClient := authClientSetup()

	mux := http.NewServeMux()
	mux.HandleFunc("/local", localListener)
	mux.HandleFunc("/callback", authClient.callbackHandler)
	mux.HandleFunc("/login/", authClient.handleCliLogin)
	if err := http.ListenAndServe(":3000", mux); err != nil {
		log.Fatal(err)
	}
}

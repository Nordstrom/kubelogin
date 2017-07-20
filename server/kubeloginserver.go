package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

/*
   struct that contains necessary oauth/oidc information
*/
type authOClient struct {
	clientID     string
	clientSecret string
	redirectURI  string
	verifier     *oidc.IDTokenVerifier
	provider     *oidc.Provider
	client       *http.Client
	usernameSpec string
}

const (
	idTokenField  = "id_token"
	portField     = "port"
	stateField    = "state"
	groupsField   = "groups"
	usernameField = "username"
	authCodeField = "code"
)

/*
   the config for oauth2, scopes contain info we want back from the auth server
*/
func (authClient *authOClient) getOAuth2Config(scopes []string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     authClient.clientID,
		ClientSecret: authClient.clientSecret,
		Endpoint:     authClient.provider.Endpoint(),
		Scopes:       scopes,
		RedirectURL:  authClient.redirectURI,
	}
}

/*
   used to grab the field from the callback request
*/
func getField(request *http.Request, fieldName string) string {

	if request.FormValue(fieldName) != "" {
		log.Print("fieldname: [" + request.FormValue(fieldName) + "]")
		return request.FormValue(fieldName)
	}
	return ""
}

/*
   converts the jwt from bytes to a readable string
*/
func rawMessageToString(claims json.RawMessage) string {

	buff := new(bytes.Buffer)
	json.Indent(buff, []byte(claims), "", "  ")
	jwt, err := buff.ReadString('}')
	if err != nil {
		log.Print(err)
		return err.Error()
	}
	log.Print(jwt)
	return jwt
}

/*
   checks to make sure the jwt contains necessary info to send back to the client
*/
func verifyJWT(jwt, usernameSpec string) bool {

	groups := strings.Contains(jwt, groupsField)
	username := strings.Contains(jwt, usernameField)
	validUsername := strings.Contains(jwt, usernameSpec)
	log.Print(groups, validUsername, username)
	if groups && username && validUsername {
		return true
	}
	return false
}

/*
   handles the get request from the client clicking the link they receive from the CLI
   this will grab the port and sets it as the state for later use
   we set the scopes to be openid, username, and groups so we get a jwt later with the needed info
   we then redirect to the login page with the necessary info.
*/
func (authClient *authOClient) handleCliLogin(writer http.ResponseWriter, request *http.Request) {
	portState := request.FormValue(portField)
	if portState == "" {
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	var scopes = []string{"openid", " https://claims.nordstrom.com/nauth/username ", " https://claims.nordstrom.com/nauth/groups "}
	authCodeURL := authClient.getOAuth2Config(scopes).AuthCodeURL(portState)
	http.Redirect(writer, request, authCodeURL, http.StatusSeeOther)
}

func (authClient *authOClient) doAuthDance(requestContext context.Context, authCode string) (*oidc.IDToken, error) {
	var (
		err   error
		token *oauth2.Token
	)
	nullToken := &oidc.IDToken{}
	oidcClientContext := oidc.ClientContext(requestContext, authClient.client)
	token, err = authClient.getOAuth2Config(nil).Exchange(oidcClientContext, authCode)
	if err != nil {
		log.Print("Failed to exchange token")
		log.Print("Error: " + err.Error() + "\n" + authCode)
		return nullToken, err
	}

	rawIDToken, ok := token.Extra(idTokenField).(string)
	if !ok {
		log.Print("Failed to get the id_token field")
		return nullToken, err
	}
	return authClient.verifier.Verify(requestContext, rawIDToken)
}

/*
   handles the callback from the auth server, exchanges the authcode, clientID, clientSecret for a rawToken which holds an id_token
   field that has the JWT. Upon verification of the jwt, we pull the claims out which is the info that is needed to send back to the client
*/
func (authClient *authOClient) callbackHandler(writer http.ResponseWriter, request *http.Request) {
	authCode := getField(request, authCodeField)
	port := getField(request, stateField)
	if authCode == "" || port == "" {
		log.Print("Error! Need authcode and port. Received: " + authCode + " " + port)
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	idToken, err := authClient.doAuthDance(request.Context(), authCode)
	if err != nil {
		log.Print("Error verifying idToken: " + err.Error())
		http.Error(writer, fmt.Sprintf("Failed to verify ID token"), http.StatusInternalServerError)
		return
	}

	sendBackURL, err := generateSendBackURL(idToken, port, authClient.usernameSpec)
	if err != nil {
		http.Error(writer, "Failed to generate send back url", http.StatusInternalServerError)
	}
	http.Redirect(writer, request, sendBackURL, http.StatusSeeOther)
	return
}

/*
   this will take the jwt and port and generate the url that will be redirected to
*/
func generateSendBackURL(rawJWT *oidc.IDToken, port string, usernameSpec string) (string, error) {
	var claims json.RawMessage
	if claimErr := rawJWT.Claims(&claims); claimErr != nil {
		log.Print("Error getting claims from idToken: " + claimErr.Error())
		return "", claimErr
	}
	jwt := rawMessageToString(claims)
	if !verifyJWT(jwt, usernameSpec) {
		log.Print("Error! Failed to verify jwt: " + jwt)
		return "", errors.New("JWT failed to verify")
	}
	sendBackURL := "http://localhost:" + port + "/client?jwt=" + jwt
	return sendBackURL, nil
}

//sets up the struct for later use
func newAuthClient(clientID string, clientSecret string, redirectURI string, usernameSpec string, provider *oidc.Provider) authOClient {
	var authClient authOClient
	authClient.clientID = clientID
	authClient.clientSecret = clientSecret
	authClient.redirectURI = redirectURI
	authClient.client = http.DefaultClient
	authClient.provider = provider
	authClient.verifier = provider.Verifier(&oidc.Config{ClientID: authClient.clientID})
	authClient.usernameSpec = usernameSpec
	return authClient
}

func healthHandler(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func getMux(authClient authOClient) *http.ServeMux {
	newMux := http.NewServeMux()
	newMux.HandleFunc("/callback", authClient.callbackHandler)
	newMux.HandleFunc("/login", authClient.handleCliLogin)
	newMux.HandleFunc("/health", healthHandler)
	return newMux
}

/*
   sets up a new mux. upon a user clicking the link to our server, it will be handled by the handleLogin function.
   When the auth server posts to our server it will be controlled by the callbackHandler. This is also initial setup for
   the struct to contain necessary information
*/
func main() {
	contxt := oidc.ClientContext(context.Background(), http.DefaultClient)
	provider, err := oidc.NewProvider(contxt, os.Getenv("OIDC_PROVIDER"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err.Error())
	}
	if err := http.ListenAndServe(os.Getenv("LISTEN_PORT"), getMux(newAuthClient(os.Getenv("CLIENT_ID"), os.Getenv("CLIENT_SECRET"), os.Getenv("REDIRECT"), os.Getenv("USERNAME_SPEC"), provider))); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"context"
	"crypto/sha1"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/go-redis/redis"
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
}

const (
	idTokenField  = "id_token"
	portField     = "port"
	stateField    = "state"
	groupsField   = "groups"
	usernameField = "username"
	authCodeField = "code"
	tokenField    = "token"
)

var (
	redisClient *redis.Client
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
		log.Printf("%s: [%s]", fieldName, request.FormValue(fieldName))
		return request.FormValue(fieldName)
	}
	return ""
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
	var scopes = []string{"openid", os.Getenv("GROUPS_CLAIM"), os.Getenv("USER_CLAIM")}
	authCodeURL := authClient.getOAuth2Config(scopes).AuthCodeURL(portState)
	http.Redirect(writer, request, authCodeURL, http.StatusSeeOther)
}

func (authClient *authOClient) doAuthDance(requestContext context.Context, authCode string) (string, error) {
	var (
		err   error
		token *oauth2.Token
	)
	oidcClientContext := oidc.ClientContext(requestContext, authClient.client)
	token, err = authClient.getOAuth2Config(nil).Exchange(oidcClientContext, authCode)
	if err != nil {
		log.Print("Failed to exchange token")
		log.Print("Error: " + err.Error() + "\n" + authCode)
		return "", err
	}

	rawIDToken, ok := token.Extra(idTokenField).(string)
	if !ok {
		log.Print("Failed to get the id_token field")
		return "", err
	}

	return rawIDToken, nil
	//return authClient.verifier.Verify(requestContext, rawIDToken)
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
	jwt, err := authClient.doAuthDance(request.Context(), authCode)
	if err != nil {
		log.Print("Error in auth: " + err.Error())
		http.Error(writer, fmt.Sprintf("Error in auth"), http.StatusInternalServerError)
		return
	}

	sendBackURL, err := generateSendBackURL(jwt, port)
	if err != nil {
		http.Error(writer, "Failed to generate send back url", http.StatusInternalServerError)
		return
	}
	http.Redirect(writer, request, sendBackURL, http.StatusSeeOther)
	return
}

func exchangeHandler(w http.ResponseWriter, r *http.Request) {
	token := getField(r, tokenField)
	jwt, err := redisClient.Get(token).Result()
	if err != nil {
		log.Printf("Error exchanging token for jwt: %v", err)
		http.Error(w, "Failed to find jwt for token "+token, http.StatusUnauthorized)
	}

	w.Write([]byte(jwt))
}

/*
   this will take the jwt and port and generate the url that will be redirected to
*/
func generateSendBackURL(jwt string, port string) (string, error) {
	// Generate sha sum for jwt
	hash := sha1.New()
	hash.Write([]byte(jwt))
	token := hash.Sum(nil)
	stringToken := fmt.Sprintf("%x", token)

	//store jwt in "database" by shasum
	err := redisClient.Set(stringToken, jwt, 10*time.Second).Err()
	if err != nil {
		log.Printf("Error storing token in database: %v", err)
	}

	sendBackURL := "http://localhost:" + port + "/client?token=" + stringToken
	return sendBackURL, nil
}

//sets up the struct for later use
func newAuthClient(clientID string, clientSecret string, redirectURI string, provider *oidc.Provider) authOClient {
	var authClient authOClient
	authClient.clientID = clientID
	authClient.clientSecret = clientSecret
	authClient.redirectURI = redirectURI
	authClient.client = http.DefaultClient
	authClient.provider = provider
	authClient.verifier = provider.Verifier(&oidc.Config{ClientID: authClient.clientID})
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
	newMux.HandleFunc("/exchange", exchangeHandler)
	return newMux
}

func makeRedisClient() error {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	ping, err := redisClient.Ping().Result()
	if err != nil {
		log.Printf("Error pinging Redis database: %v", err)
		return err
	}
	log.Printf("The result from pinging the database is %s", ping)
	return nil
}

/*
   sets up a new mux. upon a user clicking the link to our server, it will be handled by the handleLogin function.
   When the auth server posts to our server it will be controlled by the callbackHandler. This is also initial setup for
   the struct to contain necessary information
*/
func main() {
	if err := makeRedisClient(); err != nil {
		log.Fatalf("Error communicating with redis: %v", err)
	}
	contxt := oidc.ClientContext(context.Background(), http.DefaultClient)
	provider, err := oidc.NewProvider(contxt, os.Getenv("OIDC_PROVIDER_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err.Error())
	}
	listenPort := ":" + os.Getenv("LISTEN_PORT")
	if err := http.ListenAndServe(listenPort, getMux(newAuthClient(os.Getenv("CLIENT_ID"), os.Getenv("CLIENT_SECRET"), os.Getenv("REDIRECT_URL"), provider))); err != nil {
		log.Fatal(err)
	}
}

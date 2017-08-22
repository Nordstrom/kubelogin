package main

import (
	"context"
	"crypto/sha1"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/go-redis/redis"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/oauth2"
)

// struct that contains necessary oauth/oidc information
type oidcClient struct {
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
	redisClient  *redis.Client
	errorCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kubelogin_errors_total",
		Help: "number of times an error occurs",
	})
	successCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kubelogin_logins_total",
		Help: "number of times the server returns a full jwt successfully",
	})
	tokenCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kubelogin_tokens_generated_total",
		Help: "number of times the server generates a token to go into redis",
	})
	serverResponseLatencies = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "kubelogin_server_request_duration_seconds_bucket",
		Help: "tracks the duration of each response handler. classified by the request method",
	},
		[]string{"method"})
)

// the config for oauth2, scopes contain info we want back from the auth server
func (authClient *oidcClient) getOAuth2Config(scopes []string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     authClient.clientID,
		ClientSecret: authClient.clientSecret,
		Endpoint:     authClient.provider.Endpoint(),
		Scopes:       scopes,
		RedirectURL:  authClient.redirectURI,
	}
}

// used to grab the field from the callback request
func getField(request *http.Request, fieldName string) string {

	if request.FormValue(fieldName) != "" {
		log.Printf("%s: [%s]", fieldName, request.FormValue(fieldName))
		return request.FormValue(fieldName)
	}
	return ""
}

// handles the get request from the client clicking the link they receive from the CLI
// redirects to the OIDC providers login page
func (authClient *oidcClient) handleCLILogin(writer http.ResponseWriter, request *http.Request) {
	startTime := time.Now()

	portState := request.FormValue(portField)
	if portState == "" {
		errorCounter.Inc()
		http.Error(writer, "No return port in URL", http.StatusBadRequest)
		return
	}
	var scopes = []string{"openid", os.Getenv("GROUPS_CLAIM"), os.Getenv("USER_CLAIM")}
	authCodeURL := authClient.getOAuth2Config(scopes).AuthCodeURL(portState)

	http.Redirect(writer, request, authCodeURL, http.StatusSeeOther)

	elapsedTime := time.Since(startTime)
	elapsedSec := elapsedTime / time.Second
	serverResponseLatencies.WithLabelValues(request.Method).Observe(float64(elapsedSec))
}

func (authClient *oidcClient) doAuthDance(requestContext context.Context, authCode string) (string, error) {
	var (
		err   error
		token *oauth2.Token
	)

	oidcClientContext := oidc.ClientContext(requestContext, authClient.client)
	token, err = authClient.getOAuth2Config(nil).Exchange(oidcClientContext, authCode)
	if err != nil {
		errorCounter.Inc()
		log.Printf("Failed to exchange token. Error: %v", err)
		return "", err
	}

	rawIDToken, ok := token.Extra(idTokenField).(string)
	if !ok {
		errorCounter.Inc()
		log.Print("Failed to get the id_token field")
		return "", err
	}

	return rawIDToken, nil
}

// handles the callback from the auth server, exchanges the authcode, clientID, clientSecret for a rawToken which holds an id_token
// field that has the JWT. Upon verification of the JWT, we pull the claims out which is the info that is needed to send back to the client
func (authClient *oidcClient) callbackHandler(writer http.ResponseWriter, request *http.Request) {
	startTime := time.Now()

	authCode := getField(request, authCodeField)
	port := getField(request, stateField)
	if authCode == "" || port == "" {
		errorCounter.Inc()
		log.Printf("Error! Need authcode and port. Received this authcode: [%s] | Received this port: [%s]", authCode, port)
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	jwt, err := authClient.doAuthDance(request.Context(), authCode)
	if err != nil {
		errorCounter.Inc()
		log.Print("Error in auth: " + err.Error())
		http.Error(writer, fmt.Sprintf("Error in auth"), http.StatusInternalServerError)
		return
	}

	sendBackURL, err := generateSendBackURL(jwt, port)
	if err != nil {
		errorCounter.Inc()
		http.Error(writer, "Failed to generate send back url", http.StatusInternalServerError)
		return
	}

	http.Redirect(writer, request, sendBackURL, http.StatusSeeOther)

	elapsedTime := time.Since(startTime)
	elapsedSec := elapsedTime / time.Second
	serverResponseLatencies.WithLabelValues(request.Method).Observe(float64(elapsedSec))
}

func exchangeToken(token string) (string, error) {
	jwt, err := redisClient.Get(token).Result()
	if err != nil {
		errorCounter.Inc()
		log.Printf("Error exchanging token for jwt: %v", err)
		return "", err
	}
	return jwt, nil
}

func exchangeHandler(writer http.ResponseWriter, request *http.Request) {
	startTime := time.Now()
	token := getField(request, tokenField)
	jwt, err := exchangeToken(token)
	if err != nil {
		log.Print(err)
		errorCounter.Inc()
		http.Error(writer, "Invalid token", http.StatusUnauthorized)
		return
	}
	successCounter.Inc()

	writer.Write([]byte(jwt))

	elapsedTime := time.Since(startTime)
	elapsedSec := elapsedTime / time.Second
	serverResponseLatencies.WithLabelValues(request.Method).Observe(float64(elapsedSec))
}

func setToken(jwt, token string) error {
	rawTime, err := strconv.Atoi(os.Getenv("REDIS_TTL"))
	if err != nil {
		errorCounter.Inc()
		return err
	}
	ttl := time.Duration(rawTime) * time.Second
	if err := redisClient.Set(token, jwt, ttl).Err(); err != nil {
		errorCounter.Inc()
		log.Printf("Error storing token in database: %v", err)
		return err
	}
	return nil
}

// Generate SHA sum for JWT
func generateToken(jwt string) (string, error) {
	hash := sha1.New()
	hash.Write([]byte(jwt))
	token := hash.Sum(nil)
	tokenCounter.Inc()
	if err := setToken(jwt, fmt.Sprintf("%x", token)); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", token), nil
}

// this will take the JWT and port and generate the URL that will be redirected to
func generateSendBackURL(jwt string, port string) (string, error) {
	stringToken, err := generateToken(jwt)
	if err != nil {
		errorCounter.Inc()
		log.Printf("Error when setting token in database")
		return "", err
	}
	sendBackURL := "http://localhost:" + port + "/client?token=" + stringToken
	return sendBackURL, nil
}

// sets up the struct for later use
func newAuthClient(clientID string, clientSecret string, redirectURI string, provider *oidc.Provider) oidcClient {
	var authClient oidcClient
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

// sets up a new mux with the necessary handlers for the endpoints that are required
func getMux(authClient oidcClient) *http.ServeMux {
	newMux := http.NewServeMux()
	newMux.HandleFunc("/callback", authClient.callbackHandler)
	newMux.HandleFunc("/login", authClient.handleCLILogin)
	newMux.HandleFunc("/health", healthHandler)
	newMux.HandleFunc("/exchange", exchangeHandler)
	newMux.Handle("/metrics", prometheus.Handler())
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
	log.Print(ping)
	return nil
}

// registers the error and success counters with prometheus
func init() {
	prometheus.MustRegister(errorCounter)
	prometheus.MustRegister(successCounter)
	prometheus.MustRegister(tokenCounter)
	prometheus.MustRegister(serverResponseLatencies)
}

// creates our Redis client for communication
// creates an auth client based on the environment variables and provider
func main() {
	if err := makeRedisClient(); err != nil {
		log.Fatalf("Error communicating with Redis: %v", err)
	}
	ctx := oidc.ClientContext(context.Background(), http.DefaultClient)
	provider, err := oidc.NewProvider(ctx, os.Getenv("OIDC_PROVIDER_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err.Error())
	}
	listenPort := ":" + os.Getenv("LISTEN_PORT")
	if err := http.ListenAndServe(listenPort, getMux(newAuthClient(os.Getenv("CLIENT_ID"), os.Getenv("CLIENT_SECRET"), os.Getenv("REDIRECT_URL"), provider))); err != nil {
		log.Fatal(err)
	}
}

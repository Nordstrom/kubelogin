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

type app struct {
	redisValues *redisValues
	authClient  *oidcClient
}

// struct that contains necessary oauth/oidc information
type redisValues struct {
	password   string
	address    string
	timeToLive time.Duration
	client     *redis.Client
}

type oidcClient struct {
	clientID     string
	clientSecret string
	redirectURI  string
	verifier     *oidc.IDTokenVerifier
	provider     *oidc.Provider
	client       *http.Client
	groupsClaim  string
	userClaim    string
}

const (
	idTokenField     = "id_token"
	accessTokenField = "access_token"
	portField        = "port"
	stateField       = "state"
	groupsField      = "groups"
	usernameField    = "username"
	authCodeField    = "code"
	tokenField       = "token"
)

var (
	cliToServerErrorCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kubelogin_cliToServerErrors_total",
		Help: "number of times an error occurs",
	})
	serverToAuthErrorCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kubelogin_ServerToAuthErrors_total",
		Help: "number of times an error occurs",
	})
	cliToServerRequestCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kubelogin_cliToServerRequests_total",
		Help: "number of times the server returns a full jwt successfully",
	})
	serverToAuthRequestCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kubelogin_ServerToAuthRequests_total",
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

func getEnvOrDefault(envVar, defaultVal string) string {
	val := defaultVal
	if os.Getenv(envVar) != "" {
		val = os.Getenv(envVar)
	}
	return val
}

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

// used to grab fields from HTTP requests
func getField(request *http.Request, fieldName string) string {
	if request.FormValue(fieldName) != "" {
		log.Printf("%s: [%s]", fieldName, request.FormValue(fieldName))
		return request.FormValue(fieldName)
	}
	return ""
}

// handles the get request from the client clicking the link they receive from the CLI
// redirects to the OIDC providers login page
func (app *app) handleCLILogin(writer http.ResponseWriter, request *http.Request) {
	startTime := time.Now()
	cliToServerRequestCounter.Inc()
	portState := request.FormValue(portField)
	if portState == "" {
		cliToServerErrorCounter.Inc()
		http.Error(writer, "No return port in URL", http.StatusBadRequest)
		return
	}
	var scopes = []string{"openid", app.authClient.groupsClaim, app.authClient.userClaim}
	authCodeURL := app.authClient.getOAuth2Config(scopes).AuthCodeURL(portState)

	http.Redirect(writer, request, authCodeURL, http.StatusSeeOther)

	elapsedTime := time.Since(startTime)
	elapsedSec := elapsedTime / time.Second
	serverResponseLatencies.WithLabelValues(request.Method).Observe(float64(elapsedSec))
}

func (authClient *oidcClient) initiateAuthorization(requestContext context.Context, authCode string) (string, error) {
	var (
		err   error
		token *oauth2.Token
	)
	serverToAuthRequestCounter.Inc()
	oidcClientContext := oidc.ClientContext(requestContext, authClient.client)
	token, err = authClient.getOAuth2Config(nil).Exchange(oidcClientContext, authCode)
	if err != nil {
		log.Printf("Failed to exchange token. Error: %v", err)
		return "", err
	}

	fieldName := getEnvOrDefault("TOKEN_TYPE", idTokenField)
	log.Printf("Using [%s] as the JWT", fieldName)

	rawIDToken, exists := token.Extra(fieldName).(string)
	if !exists {
		errMsg := fmt.Sprintf("field [%s] not found in token", fieldName)
		log.Printf(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	return rawIDToken, nil
}

// handles the callback from the auth server, exchanges the authcode, clientID, clientSecret for a rawToken which holds an id_token
// field that has the JWT. Upon verification of the JWT, we pull the claims out which is the info that is needed to send back to the client
func (app *app) callbackHandler(writer http.ResponseWriter, request *http.Request) {
	startTime := time.Now()
	serverToAuthRequestCounter.Inc()

	authCode := getField(request, authCodeField)
	port := getField(request, stateField)
	if authCode == "" || port == "" {
		serverToAuthErrorCounter.Inc()
		log.Printf("Error! Need authcode and port. Received this authcode: [%s] | Received this port: [%s]", authCode, port)
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	jwt, err := app.authClient.initiateAuthorization(request.Context(), authCode)
	if err != nil {
		serverToAuthErrorCounter.Inc()
		log.Print("Error in auth: " + err.Error())
		http.Error(writer, fmt.Sprintf("Error in auth"), http.StatusInternalServerError)
		return
	}

	sendBackURL, err := app.redisValues.generateSendBackURL(jwt, port)
	if err != nil {
		cliToServerErrorCounter.Inc()
		http.Error(writer, "Failed to generate send back url", http.StatusInternalServerError)
		return
	}
	http.Redirect(writer, request, sendBackURL, http.StatusSeeOther)

	elapsedTime := time.Since(startTime)
	elapsedSec := elapsedTime / time.Second
	serverResponseLatencies.WithLabelValues(request.Method).Observe(float64(elapsedSec))
}

func (rv *redisValues) fetchJWTForToken(token string) (string, error) {
	jwt, err := rv.client.Get(token).Result()
	if err != nil {
		return "", err
	}
	return jwt, nil
}

func (app *app) exchangeHandler(writer http.ResponseWriter, request *http.Request) {
	cliToServerRequestCounter.Inc()
	startTime := time.Now()
	token := getField(request, tokenField)
	jwt, err := app.redisValues.fetchJWTForToken(token)
	if err != nil {
		cliToServerErrorCounter.Inc()
		log.Printf("Error exchanging token for JWT: %v", err)
		http.Error(writer, "Invalid token", http.StatusUnauthorized)
		return
	}

	_, e := writer.Write([]byte(jwt))
	if e != nil {
		cliToServerErrorCounter.Inc()
		log.Printf("unable to write jwt token: %v ", e)
		http.Error(writer, "unable to send token", http.StatusInternalServerError)
		return
	}

	elapsedTime := time.Since(startTime)
	elapsedSec := elapsedTime / time.Second
	serverResponseLatencies.WithLabelValues(request.Method).Observe(float64(elapsedSec))
}

func (rv *redisValues) setToken(jwt, token string) error {
	if err := rv.client.Set(token, jwt, rv.timeToLive).Err(); err != nil {
		log.Printf("Error storing token in database: %v", err)
		return err
	}
	return nil
}

// Generate SHA sum for JWT
func (rv *redisValues) generateToken(jwt string) (string, error) {
	hash := sha1.New()
	_, e := hash.Write([]byte(jwt))
	if e != nil {
		log.Printf("error hashing jwt: %v ", e)
		return "", e
	}
	token := hash.Sum(nil)
	tokenCounter.Inc()
	if err := rv.setToken(jwt, fmt.Sprintf("%x", token)); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", token), nil
}

// this will take the JWT and port and generate the URL that will be redirected to
func (rv *redisValues) generateSendBackURL(jwt string, port string) (string, error) {
	stringToken, err := rv.generateToken(jwt)
	if err != nil {
		log.Printf("Error when setting token in database")
		return "", err
	}
	sendBackURL := "http://localhost:" + port + "/exchange/client?token=" + stringToken
	return sendBackURL, nil
}

// sets up the struct for later use
func newAuthClient(clientID string, clientSecret string, redirectURI string, provider *oidc.Provider, groupsClaim string, userClaim string) *oidcClient {
	return &oidcClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		client:       http.DefaultClient,
		provider:     provider,
		verifier:     provider.Verifier(&oidc.Config{ClientID: clientID}),
		groupsClaim:  groupsClaim,
		userClaim:    userClaim,
	}
}

func healthHandler(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusOK)
}

func defaultHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(writer, "<!doctype html><html><head><title>Welcome to Kubelogin</title></head><body><h1>Kubelogin</h1><p>For kubelogin to work appropriately, there are a few things you'll need to setup on your own machine/environment! <br>These specs can be found on the github page <a href=https://github.com/Nordstrom/kubelogin/tree/master>here!</a> <br></p><h2>Kubelogin CLI</h2><p>Download the appropriate client from here <a href=\"https://github.com/nordstrom/kubelogin/releases\">https://github.com/nordstrom/kubelogin/releases</a>.</p></body></html>")
}

//creates a mux with handlers for desired endpoints
func getMux(app app, downloadDir string) *http.ServeMux {
	newMux := http.NewServeMux()
	fs := http.FileServer(http.Dir(downloadDir))
	newMux.HandleFunc("/", defaultHandler)
	newMux.HandleFunc("/callback", app.callbackHandler)
	newMux.Handle("/download/", http.StripPrefix("/download", fs))
	newMux.HandleFunc("/login", app.handleCLILogin)
	newMux.HandleFunc("/health", healthHandler)
	newMux.HandleFunc("/exchange", app.exchangeHandler)
	newMux.Handle("/metrics", prometheus.Handler())
	return newMux
}

func setRedisValues(redisAddress string, redisPassword string, redisTTL time.Duration) *redisValues {
	return &redisValues{
		address:    redisAddress,
		password:   redisPassword,
		timeToLive: redisTTL,
	}
}

func setAppMemberFields(rv *redisValues, oidcClient *oidcClient) app {
	return app{
		redisValues: rv,
		authClient:  oidcClient,
	}
}

func (rv *redisValues) makeRedisClient() error {
	rv.client = redis.NewClient(&redis.Options{
		Addr:     rv.address,
		Password: rv.password,
		DB:       0,
	})
	ping, err := rv.client.Ping().Result()
	if err != nil {
		log.Printf("Error pinging Redis database: %v", err)
		return err
	}
	log.Print(ping)
	return nil
}

// registers the error and success counters with prometheus
func init() {
	prometheus.MustRegister(cliToServerErrorCounter)
	prometheus.MustRegister(cliToServerRequestCounter)
	prometheus.MustRegister(serverToAuthErrorCounter)
	prometheus.MustRegister(serverToAuthRequestCounter)
	prometheus.MustRegister(serverResponseLatencies)
	prometheus.MustRegister(tokenCounter)
}

// creates our Redis client for communication
// creates an auth client based on the environment variables and provider
func main() {
	if os.Getenv("REDIS_ADDR") == "" {
		log.Fatal("REDIS_ADDR not set! Is this variable configured in the deployment?")
	}
	if os.Getenv("REDIS_PASSWORD") == "" {
		log.Fatal("REDIS_PASSWORD not set! This should be supplied as a secret in Kubernetes")
	}
	if os.Getenv("CLIENT_ID") == "" {
		log.Fatal("CLIENT_ID not set!")
	}
	if os.Getenv("CLIENT_SECRET") == "" {
		log.Fatal("CLIENT_SECRET not set!")
	}
	if os.Getenv("REDIRECT_URL") == "" {
		log.Fatal("REDIRECT_URL not set!")
	}
	if os.Getenv("HTTPS_CERT_PATH") == "" {
		log.Fatal("HTTPS_CERT_PATH not set!")
	}
	if os.Getenv("HTTPS_KEY_PATH") == "" {
		log.Fatal("HTTPS_KEY_PATH not set!")
	}

	ctx := oidc.ClientContext(context.Background(), http.DefaultClient)
	provider, err := oidc.NewProvider(ctx, os.Getenv("OIDC_PROVIDER_URL"))
	if err != nil {
		log.Fatalf("error: %v\n", err.Error())
	}
	listenPortInt, err := strconv.Atoi(os.Getenv("LISTEN_PORT"))
	if err != nil {
		log.Fatalf("Error parsing port to listen on. Error: %v", err)
	}
	if listenPortInt < 0 || listenPortInt > 65536 {
		log.Fatalf("LISTEN_PORT contains an invalid port. Port given: %v", listenPortInt)
	}
	listenPort := ":" + os.Getenv("LISTEN_PORT")
	downloadDir := os.Getenv("DOWNLOAD_DIR")
	if downloadDir == "" {
		downloadDir = "/download"
	}
	groupsClaim := os.Getenv("GROUPS_CLAIM")
	if groupsClaim == "" {
		groupsClaim = "groups"
	}
	userClaim := os.Getenv("USER_CLAIM")
	if userClaim == "" {
		userClaim = "email"
	}
	ttl := os.Getenv("REDIS_TTL")
	if ttl == "" {
		ttl = "10s"
	}
	redisTTL, err := time.ParseDuration(ttl)
	if err != nil {
		log.Fatal("Failed to parse the duration of the Redis TTL, please check that a valid value was set. e.g. 10s or 1m10s")
	}
	rv := setRedisValues(os.Getenv("REDIS_ADDR"), os.Getenv("REDIS_PASSWORD"), redisTTL)
	oidcClient := newAuthClient(os.Getenv("CLIENT_ID"), os.Getenv("CLIENT_SECRET"), os.Getenv("REDIRECT_URL"), provider, groupsClaim, userClaim)
	app := setAppMemberFields(rv, oidcClient)
	if err := app.redisValues.makeRedisClient(); err != nil {
		log.Fatalf("Error communicating with Redis: %v", err)
	}
	mux := getMux(app, downloadDir)
	crt := os.Getenv("HTTPS_CERT_PATH")
	key := os.Getenv("HTTPS_KEY_PATH")
	if err := http.ListenAndServeTLS(listenPort, crt, key, mux); err != nil {
		log.Fatalf("Failed to listen on port: %s | Error: %v", listenPort, err)
	}
}

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path"
	"runtime"
	"strings"
	"time"
)

var (
	testTest               = false
	aliasFlag              string
	userFlag               string
	kubeloginServerBaseURL string
	version                string
	doneChannel            chan bool
	usageMessage           = `Kubelogin Usage:

  One time login:
    kubelogin login --server-url=https://kubelogin.example.com --kubectl-user=user

  Configure an alias (shortcut):
    kubelogin config --alias=example --server-url=https://kubelogin.example.com --kubectl-user=example_oidc

  Use an alias:
    kubelogin login example`
)

//AliasConfig contains the structure of what's in the config file
type AliasConfig struct {
	Alias       string `yaml:"alias"`
	BaseURL     string `yaml:"server-url"`
	KubectlUser string `yaml:"kubectl-user"`
}

// fortest checks at runtime if tests are running, if not we must have version.
func fortest() {
	if !testTest { // testTest is set in main_test.go
		panic("Kubelogin version is not set.") // Makefile must inject version string
	} else {
		version = "testing"
	}
}

func findFreePort() (string, error) {
	server, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}
	defer server.Close() // nolint: errcheck
	hostString := server.Addr().String()
	_, portString, err := net.SplitHostPort(hostString)
	if err != nil {
		return "", err
	}
	return portString, nil
}

func createMux(app app) *http.ServeMux {
	newMux := http.NewServeMux()
	newMux.HandleFunc("/exchange/", app.tokenHandler)
	newMux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(""))
		if err != nil {
			log.Printf("Unable to write favicon: %v ", err)
		}
		return
	})
	return newMux
}

func generateURLAndListenForServerResponse(app app) {
	loginURL, portNum, err := app.generateAuthURL()
	if err != nil {
		log.Fatal(err.Error())
	}
	doneChannel = make(chan bool)
	go func() {
		l, err := net.Listen("tcp", ":"+portNum)
		if err != nil {
			fmt.Printf("Error listening on port: %s. Error: %v\n", portNum, err)
			os.Exit(1)
		}
		if runtime.GOOS == "darwin" {
			// On OS X, run the `open` CLI to use the default browser to open the login URL.
			fmt.Printf("Opening %s ...\n", loginURL)
			err := exec.Command("/usr/bin/open", loginURL).Run()
			if err != nil {
				fmt.Printf("Error opening; please open the URL manually: %s \n", loginURL)
			}
		} else {
			fmt.Printf("Follow this URL to log into auth provider: %s\n", loginURL)
		}
		if err = http.Serve(l, createMux(app)); err != nil {
			fmt.Printf("Error listening on port: %s. Error: %v\n", portNum, err)
			os.Exit(1)
		}
	}()
	<-doneChannel
	fmt.Println("You are now logged in! Enjoy kubectl-ing!")
	time.Sleep(1 * time.Second)
}

func setFlags(command *flag.FlagSet, loginCmd bool) {
	if !loginCmd {
		command.StringVar(&aliasFlag, "alias", "default", "alias name in the config file, used for an easy login")
	}
	command.StringVar(&userFlag, "kubectl-user", "kubelogin_user", "in kubectl config, username used to store credentials")
	command.StringVar(&kubeloginServerBaseURL, "server-url", "", "base URL of the kubelogin server, ex: https://kubelogin.example.com")
}

func login(app app) {
	loginCommmand := flag.NewFlagSet("login", flag.ExitOnError)
	setFlags(loginCommmand, true)
	if !strings.HasPrefix(os.Args[2], "--") {
		//use alias to extract needed information
		if err := app.getConfigSettings(os.Args[2]); err != nil {
			log.Fatal(err)
		}
		generateURLAndListenForServerResponse(app)
	} else {
		_ = loginCommmand.Parse(os.Args[2:])
		if loginCommmand.Parsed() {
			if kubeloginServerBaseURL == "" {
				log.Fatal("--server-url must be set!")
			}
			app.kubectlUser = userFlag
			app.kubeloginServer = kubeloginServerBaseURL
			generateURLAndListenForServerResponse(app)
		}
	}
}

func config(app app) {
	configCommand := flag.NewFlagSet("config", flag.ExitOnError)
	setFlags(configCommand, false)
	_ = configCommand.Parse(os.Args[2:])
	if configCommand.Parsed() {
		if kubeloginServerBaseURL == "" {
			log.Fatal("--server-url must be set!")
		}
		verifiedServerURL, err := url.ParseRequestURI(kubeloginServerBaseURL)
		if err != nil {
			log.Fatalf("Invalid URL given: %v | Err: %v", kubeloginServerBaseURL, err)
		}

		if err := app.configureFile(aliasFlag, verifiedServerURL, userFlag); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}
}

func main() {
	var app app

	fortest() // Check if test are run

	user, err := user.Current()
	if err != nil {
		log.Fatalf("Could not determine current user of this system. Err: %v", err)
	}

	if os.Args[1] == "version" {
		fmt.Println(version)
		os.Exit(0)
	}

	app.filenameWithPath = path.Join(user.HomeDir, "/.kubeloginrc.yaml")

	if len(os.Args) < 3 {
		fmt.Println(usageMessage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "login":
		login(app)
	case "config":
		config(app)
	default:
		fmt.Println(usageMessage)
		os.Exit(1)
	}
}

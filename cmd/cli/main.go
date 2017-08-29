package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

var (
	aliasFlag   string
	userFlag    string
	serverFlag  string
	doneChannel chan bool
)

func findFreePort() (string, error) {
	server, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}
	defer server.Close()
	hostString := server.Addr().String()
	_, portString, err := net.SplitHostPort(hostString)
	if err != nil {
		return "", err
	}
	return portString, nil
}

func makeExchange(token string) error {
	url := fmt.Sprintf("https://%s/exchange?token=%s", serverFlag, token)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Unable to create request. %s", err)
		return err
	}
	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Unable to make request. %s", err)
		return err
	}

	defer res.Body.Close()
	jwt, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Unable to read response body. %s", err)
		return err
	}
	if err := configureKubectl(string(jwt)); err != nil {
		log.Printf("Error when setting credentials: %v", err)
		return err
	}
	return nil
}

func localHandler(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	if err := makeExchange(token); err != nil {
		log.Fatalf("Could not exchange token for jwt %v", err)
	}
	fmt.Fprint(w, "You are now logged in! You can close me  :)")
	doneChannel <- true
}

func configureKubectl(jwt string) error {
	configCmd := exec.Command("kubectl", "config", "set-credentials", userFlag, "--token="+jwt)
	if err := configCmd.Run(); err != nil {
		return err
	}
	return nil
}

func generateAuthURL() (string, string, error) {
	portNum, err := findFreePort()
	if err != nil {
		log.Print("err, could not find an open port")
		return "", "", err
	}

	serverURL := fmt.Sprintf("https://%s/login?port=%s", serverFlag, portNum)

	return serverURL, portNum, nil
}

func createMux() *http.ServeMux {
	newMux := http.NewServeMux()
	newMux.HandleFunc("/", localHandler)
	return newMux
}

func beginInteraction() {
	authURL, portNum, err := generateAuthURL()
	if err != nil {
		log.Fatal(err.Error())
	}
	doneChannel = make(chan bool)
	go func() {
		log.Print("\nFollow this url if you want to live!: ", authURL)
		if err := http.ListenAndServe(":"+portNum, createMux()); err != nil {
			log.Fatalf("Error listening on port: %s. Error: %v", portNum, err)
		}
	}()
	<-doneChannel
	log.Print("You are now logged in! Enjoy kubectl-ing!")
}

func loginFlags(login *flag.FlagSet) {
	login.StringVar(&userFlag, "kubectl-user", "kubelogin_user", "username used in kube config")
	login.StringVar(&serverFlag, "server", "", "cluster id used in conjuction with host name")
}
func configFlags(config *flag.FlagSet) {
	config.StringVar(&aliasFlag, "alias", "default", "alias name in the config file, used for as an easy login")
	config.StringVar(&userFlag, "kubectl_user", "kubelogin_user", "username used in kube config")
	config.StringVar(&serverFlag, "server", "", "hostname of the server url, paths added in other functions")
}
func main() {
	loginCommmand := flag.NewFlagSet("login", flag.ExitOnError)
	loginFlags(loginCommmand)
	configCommand := flag.NewFlagSet("config", flag.ExitOnError)
	configFlags(configCommand)
	if len(os.Args) < 3 {
		log.Fatal("Must supply login or config command with flags/alias")
	}
	switch os.Args[1] {
	case "login":
		if !(strings.Contains(os.Args[2], "--") || strings.Contains(os.Args[2], "-")) {
			log.Fatal("Assume alias case and run through flow")
			beginInteraction()
		} else {
			loginCommmand.Parse(os.Args[2:])
			if loginCommmand.Parsed() {
				if serverFlag == "" {
					log.Fatal("--server must be set!")
				}
				beginInteraction()
			}
		}
	case "config":
		configCommand.Parse(os.Args[2:])
		if configCommand.Parsed() {
			if serverFlag == "" {
				log.Fatal("--server must be set!")
			}
			//configureRcFile
		}
	default:
		log.Fatal("Correct usage: kublogin COMMAND FLAGS | valid commands are login or config")
	}
}

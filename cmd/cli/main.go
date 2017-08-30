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
	"os/user"
	"strings"

	"gopkg.in/yaml.v2"
)

var (
	aliasFlag   string
	userFlag    string
	serverFlag  string
	doneChannel chan bool
)

//Config file format for extracting and writing the config file
type AliasConfig struct {
	Alias       string `yaml:"alias"`
	BaseURL     string `yaml:"base-url"`
	KubectlUser string `yaml:"kubectl-user"`
}

type Config struct {
	Aliases []*AliasConfig `yaml:"aliases"`
}

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
	url := fmt.Sprintf("%s/exchange?token=%s", serverFlag, token)
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

	loginURL := fmt.Sprintf("%s/login?port=%s", serverFlag, portNum)

	return loginURL, portNum, nil
}

func createMux() *http.ServeMux {
	newMux := http.NewServeMux()
	newMux.HandleFunc("/", localHandler)
	return newMux
}

func beginInteraction() {
	loginURL, portNum, err := generateAuthURL()
	if err != nil {
		log.Fatal(err.Error())
	}
	doneChannel = make(chan bool)
	go func() {
		log.Print("\nFollow this url if you want to live!: ", loginURL)
		if err := http.ListenAndServe(":"+portNum, createMux()); err != nil {
			log.Fatalf("Error listening on port: %s. Error: %v", portNum, err)
		}
	}()
	<-doneChannel
	log.Print("You are now logged in! Enjoy kubectl-ing!")
}

func setFlags(command *flag.FlagSet, loginCmd bool) {
	if !loginCmd {
		command.StringVar(&aliasFlag, "alias", "default", "alias name in the config file, used for an easy login")
	}
	command.StringVar(&userFlag, "kubectl_user", "kubelogin_user", "username used in kube config")
	command.StringVar(&serverFlag, "server", "", "hostname of the server url, correct paths added in other functions")
}
func getConfigSettings(alias string) {
	user, err := user.Current()
	if err != nil {
		log.Fatalf("Could not determine current user of this system. Err: %v", err)
	}
	filenameWithPath := fmt.Sprintf("%s/.kubeloginrc.yaml", user.HomeDir)
	yamlFile, err := ioutil.ReadFile(filenameWithPath)
	if err != nil {
		log.Fatalf("Couldn't read config file! Err: %v", err)
	}
	var config Config
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		log.Fatalf("Error unmarshaling yaml file! Err: %v", err)
	}
	for _, aliases := range config.Aliases {
		if alias == aliases.Alias {
			userFlag = aliases.KubectlUser
			serverFlag = aliases.BaseURL
			return
		}
	}
	log.Fatal("Could not find specified alias, check spelling or use config command to create an entry")
}

func configureFile() {
	var config Config
	var aliasConfig AliasConfig
	user, err := user.Current()
	if err != nil {
		log.Fatalf("Could not determine current user of this system. Err: %v", err)
	}
	filenameWithPath := fmt.Sprintf("%s/.kubeloginrc.yaml", user.HomeDir)
	yamlFile, err := ioutil.ReadFile(filenameWithPath)
	if err != nil {
		log.Print("Couldn't find config file in root directory. Creating config file...")
		createCmd := exec.Command("touch", filenameWithPath)
		if err := createCmd.Run(); err != nil {
			log.Fatalf("Error creating file in root directory! %v", err)
		}
		log.Print("Config file created, setting config values...")
		config.Aliases = make([]*AliasConfig, 0)
		aliasConfig.BaseURL = serverFlag
		aliasConfig.Alias = aliasFlag
		aliasConfig.KubectlUser = userFlag
		config.Aliases = append(config.Aliases, &aliasConfig)
		marshaledYaml, err := yaml.Marshal(config)
		if err != nil {
			log.Fatalf("Error configuring file: %v", err)
		}
		if err := ioutil.WriteFile(filenameWithPath, marshaledYaml, 0600); err != nil {
			log.Fatalf("Error writing to kubeloginrc file: %v", err)
		}
		log.Print("File configured")
	}
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		log.Fatalf("Error unmarshaling yaml file! Err: %v", err)
	}
	for _, aliases := range config.Aliases {
		if aliasFlag == aliases.Alias {
			aliases.KubectlUser = userFlag
			aliases.BaseURL = serverFlag
			marshaledYaml, err := yaml.Marshal(config)
			log.Printf(string(marshaledYaml))
			if err != nil {
				log.Fatalf("Error marshaling yaml: %v", err)
			}
			if err := ioutil.WriteFile(filenameWithPath, marshaledYaml, 0600); err != nil {
				log.Fatalf("Error writing to kubeloginrc file: %v", err)
			}
			log.Print("File configured")
			os.Exit(0)
		}
	}
	var newAliasConfig AliasConfig
	newAliasConfig.BaseURL = serverFlag
	newAliasConfig.Alias = aliasFlag
	newAliasConfig.KubectlUser = userFlag
	config.Aliases = append(config.Aliases, &newAliasConfig)
	marshaledYaml, err := yaml.Marshal(config)
	if err != nil {
		log.Fatalf("Error configuring file: %v", err)
	}
	if err := ioutil.WriteFile(filenameWithPath, marshaledYaml, 0600); err != nil {
		log.Fatalf("Error writing to kubeloginrc file: %v", err)
	}
	log.Print("File configured")

}

func main() {
	loginCommmand := flag.NewFlagSet("login", flag.ExitOnError)
	setFlags(loginCommmand, true)
	configCommand := flag.NewFlagSet("config", flag.ExitOnError)
	setFlags(configCommand, false)
	if len(os.Args) < 3 {
		log.Fatal("Must supply login or config command with flags/alias")
	}
	switch os.Args[1] {
	case "login":
		if !(strings.Contains(os.Args[2], "--") || strings.Contains(os.Args[2], "-")) {
			//use alias to extract needed information
			getConfigSettings(os.Args[2])
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
			configureFile()
		}
	default:
		log.Fatal("Correct usage: kublogin COMMAND FLAGS | valid commands are login or config")
	}
}

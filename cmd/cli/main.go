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

	"github.com/pkg/errors"

	"gopkg.in/yaml.v2"
)

var (
	aliasFlag        string
	userFlag         string
	serverFlag       string
	doneChannel      chan bool
	filenameWithPath string
)

//AliasConfig contains the structure of what's in the config file
type AliasConfig struct {
	Alias       string `yaml:"alias"`
	BaseURL     string `yaml:"base-url"`
	KubectlUser string `yaml:"kubectl-user"`
}

//Config struct to wrap the Aliases with their specific values inside of
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
func getConfigSettings(alias string) error {
	yamlFile, err := ioutil.ReadFile(filenameWithPath)
	if err != nil {
		return errors.Wrap(err, "failed to read config file for login use")
	}
	var config Config
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		return errors.Wrap(err, "failed to unmarshal yaml file for login use")
	}

	index, ok := config.aliasSearch(alias)
	if !ok {
		return errors.Wrap(err, "Could not find specified alias, check spelling or use config command to create an entry")
	}
	userFlag = config.Aliases[index].KubectlUser
	serverFlag = config.Aliases[index].BaseURL
	return nil
}

func (config *Config) aliasSearch(alias string) (int, bool) {
	for index, aliases := range config.Aliases {
		if alias == aliases.Alias {
			return index, true
		}
	}
	return -1, false
}

func (config *Config) createConfig(aliasConfig AliasConfig) error {
	log.Print("Couldn't find config file in root directory. Creating config file...")
	createCmd := exec.Command("touch", filenameWithPath)
	if err := createCmd.Run(); err != nil {
		return errors.Wrap(err, "failed to create file in root directory")
	}
	log.Print("Config file created, setting config values...")
	config.Aliases = make([]*AliasConfig, 0)
	aliasConfig.BaseURL = serverFlag
	aliasConfig.Alias = aliasFlag
	aliasConfig.KubectlUser = userFlag
	config.Aliases = append(config.Aliases, &aliasConfig)
	marshaledYaml, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "failed to marshal yaml when creating config file")
	}
	if err := ioutil.WriteFile(filenameWithPath, marshaledYaml, 0600); err != nil {
		return errors.Wrap(err, "Error writing to kubeloginrc file after creation")
	}
	log.Printf(string(marshaledYaml))
	log.Print("File configured")
	return nil
}

func (config *Config) newAliasConfig() error {
	var newAliasConfig AliasConfig
	newAliasConfig.BaseURL = serverFlag
	newAliasConfig.Alias = aliasFlag
	newAliasConfig.KubectlUser = userFlag
	config.Aliases = append(config.Aliases, &newAliasConfig)
	marshaledYaml, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "failed to marshal new alias yaml")
	}
	if err := ioutil.WriteFile(filenameWithPath, marshaledYaml, 0600); err != nil {
		return errors.Wrap(err, "failed to write to kubeloginrc file with the new alias")
	}
	log.Printf(string(marshaledYaml))
	log.Print("New Alias configured")
	return nil
}

func (config *Config) updateAlias(index int) error {
	config.Aliases[index].KubectlUser = userFlag
	config.Aliases[index].BaseURL = serverFlag
	marshaledYaml, err := yaml.Marshal(config)
	log.Printf(string(marshaledYaml))
	if err != nil {
		return errors.Wrap(err, "Error marshaling yaml")
	}
	if err := ioutil.WriteFile(filenameWithPath, marshaledYaml, 0600); err != nil {
		return errors.Wrap(err, "Error writing to kubeloginrc file")
	}
	log.Print("Alias updated")
	return nil
}

func configureFile() error {
	var config Config
	var aliasConfig AliasConfig
	yamlFile, err := ioutil.ReadFile(filenameWithPath)
	if err != nil {
		if err := config.createConfig(aliasConfig); err != nil {
			return err
		}
		return nil
	}
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		return errors.Wrap(err, "failed to unmarshal yaml file")
	}
	index, ok := config.aliasSearch(aliasFlag)
	if !ok {
		if err := config.newAliasConfig(); err != nil {
			return err
		}
		return nil
	}
	if err := config.updateAlias(index); err != nil {
		return err
	}
	return nil
}

func main() {
	loginCommmand := flag.NewFlagSet("login", flag.ExitOnError)
	setFlags(loginCommmand, true)
	configCommand := flag.NewFlagSet("config", flag.ExitOnError)
	setFlags(configCommand, false)
	user, err := user.Current()
	if err != nil {
		log.Fatalf("Could not determine current user of this system. Err: %v", err)
	}
	filenameWithPath = fmt.Sprintf("%s/.kubeloginrc.yaml", user.HomeDir)
	if len(os.Args) < 3 {
		log.Fatal("Must supply login or config command with flags/alias")
	}
	switch os.Args[1] {
	case "login":
		if !(strings.Contains(os.Args[2], "--") || strings.Contains(os.Args[2], "-")) {
			//use alias to extract needed information
			if err := getConfigSettings(os.Args[2]); err != nil {
				log.Fatal(err)
			}
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
			if err := configureFile(); err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		}
	default:
		log.Fatal("Correct usage: kublogin COMMAND FLAGS | valid commands are login or config")
	}
}

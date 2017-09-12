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
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"

	"gopkg.in/yaml.v2"
)

var (
	aliasFlag              string
	userFlag               string
	kubeloginServerBaseURL string
	doneChannel            chan bool
	filenameWithPath       string
)

//AliasConfig contains the structure of what's in the config file
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
	url := fmt.Sprintf("%s/exchange?token=%s", kubeloginServerBaseURL, token)
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
	if res.StatusCode != http.StatusOK {
		log.Fatalf("Failed to retrieve token from server. Please try again or contact your administrator")
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

func tokenHandler(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	if err := makeExchange(token); err != nil {
		log.Fatalf("Could not exchange token for jwt %v", err)
	}
	fmt.Fprint(w, "You are now logged in! You can close this window")
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

	loginURL := fmt.Sprintf("%s/login?port=%s", kubeloginServerBaseURL, portNum)

	return loginURL, portNum, nil
}

func createMux() *http.ServeMux {
	newMux := http.NewServeMux()
	newMux.HandleFunc("/exchange/", tokenHandler)
	newMux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(""))
		return
	})
	return newMux
}

func generateURLAndListenForServerResponse() {
	loginURL, portNum, err := generateAuthURL()
	if err != nil {
		log.Fatal(err.Error())
	}
	doneChannel = make(chan bool)
	go func() {
		log.Print("Follow this URL to log into auth provider: ", loginURL)
		if err := http.ListenAndServe(":"+portNum, createMux()); err != nil {
			log.Fatalf("Error listening on port: %s. Error: %v", portNum, err)
		}
	}()
	<-doneChannel
	log.Print("You are now logged in! Enjoy kubectl-ing!")
	time.Sleep(1 * time.Second)
}

func setFlags(command *flag.FlagSet, loginCmd bool) {
	if !loginCmd {
		command.StringVar(&aliasFlag, "alias", "default", "alias name in the config file, used for an easy login")
	}
	command.StringVar(&userFlag, "kubectlUser", "kubelogin_user", "username used in kubectl config")
	command.StringVar(&kubeloginServerBaseURL, "server", "", "base URL of the server, correct paths added in other functions")
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

	aliasConfig, ok := config.aliasSearch(alias)
	if !ok {
		return errors.New("Could not find specified alias, check spelling or use the config verb to create an alias")
	}
	userFlag = aliasConfig.KubectlUser
	kubeloginServerBaseURL = aliasConfig.BaseURL
	return nil
}

func (config *Config) aliasSearch(alias string) (*AliasConfig, bool) {
	for index, aliases := range config.Aliases {
		if alias == aliases.Alias {
			return config.Aliases[index], true
		}
	}
	return nil, false
}

func (config *Config) createConfig(aliasConfig AliasConfig) error {
	log.Print("Couldn't find config file in root directory. Creating config file...")
	createCmd := exec.Command("touch", filenameWithPath)
	if err := createCmd.Run(); err != nil {
		return errors.Wrap(err, "failed to create file in root directory")
	}
	log.Print("Config file created, setting config values...")
	config.Aliases = make([]*AliasConfig, 0)
	aliasConfig.BaseURL = kubeloginServerBaseURL
	aliasConfig.Alias = aliasFlag
	aliasConfig.KubectlUser = userFlag
	config.appendAlias(aliasConfig)
	if err := config.writeToFile(); err != nil {
		log.Fatal(err)
	}
	log.Print("File configured")
	return nil
}

func (config *Config) newAliasConfig(kubeloginrcAlias, loginServerURL, kubectlUser string) AliasConfig {
	newConfig := AliasConfig{
		BaseURL:     loginServerURL,
		Alias:       kubeloginrcAlias,
		KubectlUser: kubectlUser,
	}
	return newConfig
}

func (config *Config) appendAlias(aliasConfig AliasConfig) {
	config.Aliases = append(config.Aliases, &aliasConfig)
}

func (config *Config) writeToFile() error {
	marshaledYaml, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "failed to marshal alias yaml")
	}
	if err := ioutil.WriteFile(filenameWithPath, marshaledYaml, 0600); err != nil {
		return errors.Wrap(err, "failed to write to kubeloginrc file with the alias")
	}
	log.Printf(string(marshaledYaml))
	return nil
}

func (config *Config) updateAlias(aliasConfig *AliasConfig) error {
	aliasConfig.KubectlUser = userFlag
	aliasConfig.BaseURL = kubeloginServerBaseURL
	if err := config.writeToFile(); err != nil {
		log.Fatal(err)
	}
	log.Print("Alias updated")
	return nil
}

func configureFile(kubeloginrcAlias, loginServerURL, kubectlUser string) error {
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
	foundAliasConfig, ok := config.aliasSearch(aliasFlag)
	if !ok {
		newConfig := config.newAliasConfig(kubeloginrcAlias, loginServerURL, kubectlUser)
		config.appendAlias(newConfig)
		if err := config.writeToFile(); err != nil {
			log.Fatal(err)
		}
		log.Print("New Alias configured")
		return nil
	}
	if err := config.updateAlias(foundAliasConfig); err != nil {
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
	filenameWithPath = path.Join(user.HomeDir, "/.kubeloginrc.yaml")
	if len(os.Args) < 3 {
		usageMessage := `Kubelogin Usage:
        kubelogin config --server=server --alias=alias --kubectlUser=user
        kubelogin login ALIAS
        kubelogin login --server=baseURL --kubectlUser=user`
		log.Fatal(usageMessage)
	}
	switch os.Args[1] {
	case "login":
		if !(strings.Contains(os.Args[2], "--") || strings.Contains(os.Args[2], "-")) {
			//use alias to extract needed information
			if err := getConfigSettings(os.Args[2]); err != nil {
				log.Fatal(err)
			}
			generateURLAndListenForServerResponse()
		} else {
			loginCommmand.Parse(os.Args[2:])
			if loginCommmand.Parsed() {
				if kubeloginServerBaseURL == "" {
					log.Fatal("--server must be set!")
				}
				generateURLAndListenForServerResponse()
			}
		}
	case "config":
		configCommand.Parse(os.Args[2:])
		if configCommand.Parsed() {
			if kubeloginServerBaseURL == "" {
				log.Fatal("--server must be set!")
			}
			if err := configureFile(aliasFlag, kubeloginServerBaseURL, userFlag); err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		}
	default:
		log.Fatal("Kubelogin Usage: \n\n kubelogin config --server=server --alias=alias --kubectl_user=user \n\n kubelogin login alias \n kubelogin login --server=server --kubectl_user=user")
	}
}

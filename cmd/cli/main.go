package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"

	"gopkg.in/yaml.v2"
)

type app struct {
	filenameWithPath string
	kubectlUser      string
	kubeloginAlias   string
	kubeloginServer  string
}

var (
	aliasFlag              string
	userFlag               string
	kubeloginServerBaseURL string
	doneChannel            chan bool
	usageMessage           = `Kubelogin Usage:
    kubelogin config --server=server --alias=alias --kubectlUser=user
    kubelogin login ALIAS
    kubelogin login --server=baseURL --kubectlUser=user`
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

func (app *app) makeExchange(token string) error {
	url := fmt.Sprintf("%s/exchange?token=%s", app.kubeloginServer, token)
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
	if err := app.configureKubectl(string(jwt)); err != nil {
		log.Printf("Error when setting credentials: %v", err)
		return err
	}
	return nil
}

func (app *app) tokenHandler(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	if err := app.makeExchange(token); err != nil {
		log.Fatalf("Could not exchange token for jwt %v", err)
	}
	fmt.Fprint(w, "You are now logged in! You can close this window")
	doneChannel <- true
}

func (app *app) configureKubectl(jwt string) error {
	configCmd := exec.Command("kubectl", "config", "set-credentials", app.kubectlUser, "--token="+jwt)
	if err := configCmd.Run(); err != nil {
		return err
	}
	return nil
}

func (app *app) generateAuthURL() (string, string, error) {
	portNum, err := findFreePort()
	if err != nil {
		log.Print("err, could not find an open port")
		return "", "", err
	}

	loginURL := fmt.Sprintf("%s/login?port=%s", app.kubeloginServer, portNum)

	return loginURL, portNum, nil
}

func createMux(app app) *http.ServeMux {
	newMux := http.NewServeMux()
	newMux.HandleFunc("/exchange/", app.tokenHandler)
	newMux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(""))
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
		fmt.Printf("Follow this URL to log into auth provider: %s\n", loginURL)
		if err := http.ListenAndServe(":"+portNum, createMux(app)); err != nil {
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
	command.StringVar(&userFlag, "kubectlUser", "kubelogin_user", "username used in kubectl config")
	command.StringVar(&kubeloginServerBaseURL, "server", "", "base URL of the server, correct paths added in other functions")
}
func (app *app) getConfigSettings(alias string) error {
	yamlFile, err := ioutil.ReadFile(app.filenameWithPath)
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
	app.kubectlUser = aliasConfig.KubectlUser
	app.kubeloginServer = aliasConfig.BaseURL
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

func (config *Config) createConfig(onDiskFile string, aliasConfig AliasConfig) error {
	log.Print("Couldn't find config file in root directory. Creating config file...")
	createCmd := exec.Command("touch", onDiskFile)
	if err := createCmd.Run(); err != nil {
		return errors.Wrap(err, "failed to create file in root directory")
	}
	log.Print("Config file created, setting config values...")
	config.Aliases = make([]*AliasConfig, 0)
	config.appendAlias(aliasConfig)
	if err := config.writeToFile(onDiskFile); err != nil {
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

func (config *Config) writeToFile(onDiskFile string) error {
	marshaledYaml, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "failed to marshal alias yaml")
	}
	if err := ioutil.WriteFile(onDiskFile, marshaledYaml, 0600); err != nil {
		return errors.Wrap(err, "failed to write to kubeloginrc file with the alias")
	}
	log.Printf(string(marshaledYaml))
	return nil
}

func (config *Config) updateAlias(aliasConfig *AliasConfig, loginServerURL *url.URL, onDiskFile string) error {
	aliasConfig.KubectlUser = userFlag
	aliasConfig.BaseURL = loginServerURL.String()
	if err := config.writeToFile(onDiskFile); err != nil {
		log.Fatal(err)
	}
	log.Print("Alias updated")
	return nil
}

func (app *app) configureFile(kubeloginrcAlias string, loginServerURL *url.URL, kubectlUser string) error {
	var config Config
	aliasConfig := config.newAliasConfig(kubeloginrcAlias, loginServerURL.String(), kubectlUser)
	yamlFile, err := ioutil.ReadFile(app.filenameWithPath)
	if err != nil {
		if err := config.createConfig(app.filenameWithPath, aliasConfig); err != nil {
			return err
		}
		return nil
	}
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		return errors.Wrap(err, "failed to unmarshal yaml file")
	}
	foundAliasConfig, ok := config.aliasSearch(aliasFlag)
	if !ok {
		newConfig := config.newAliasConfig(kubeloginrcAlias, loginServerURL.String(), kubectlUser)
		config.appendAlias(newConfig)
		if err := config.writeToFile(app.filenameWithPath); err != nil {
			log.Fatal(err)
		}
		log.Print("New Alias configured")
		return nil
	}
	if err := config.updateAlias(foundAliasConfig, loginServerURL, app.filenameWithPath); err != nil {
		return err
	}
	return nil
}

func main() {
	var app app
	loginCommmand := flag.NewFlagSet("login", flag.ExitOnError)
	setFlags(loginCommmand, true)
	configCommand := flag.NewFlagSet("config", flag.ExitOnError)
	setFlags(configCommand, false)
	user, err := user.Current()
	if err != nil {
		log.Fatalf("Could not determine current user of this system. Err: %v", err)
	}
	app.filenameWithPath = path.Join(user.HomeDir, "/.kubeloginrc.yaml")
	if len(os.Args) < 3 {
		fmt.Println(usageMessage)
		os.Exit(1)
	}
	switch os.Args[1] {
	case "login":
		if !(strings.Contains(os.Args[2], "--") || strings.Contains(os.Args[2], "-")) {
			//use alias to extract needed information
			if err := app.getConfigSettings(os.Args[2]); err != nil {
				log.Fatal(err)
			}
			generateURLAndListenForServerResponse(app)
		} else {
			loginCommmand.Parse(os.Args[2:])
			if loginCommmand.Parsed() {
				if kubeloginServerBaseURL == "" {
					log.Fatal("--server must be set!")
				}
				app.kubectlUser = userFlag
				app.kubeloginServer = kubeloginServerBaseURL
				generateURLAndListenForServerResponse(app)
			}
		}
	case "config":
		configCommand.Parse(os.Args[2:])
		if configCommand.Parsed() {
			if kubeloginServerBaseURL == "" {
				log.Fatal("--server must be set!")
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
	default:
		fmt.Println(usageMessage)
		os.Exit(1)
	}
}

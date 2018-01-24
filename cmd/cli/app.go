package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os/exec"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type app struct {
	filenameWithPath string
	kubectlUser      string
	kubeloginAlias   string
	kubeloginServer  string
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
		log.Fatalf("Failed to retrieve token from kubelogin server. Please try again or contact your administrator")
	}
	defer res.Body.Close() // nolint: errcheck
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
	return configCmd.Run()
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

func (app *app) configureFile(kubeloginrcAlias string, loginServerURL *url.URL, kubectlUser string) error {
	var config Config
	aliasConfig := config.newAliasConfig(kubeloginrcAlias, loginServerURL.String(), kubectlUser)
	yamlFile, err := ioutil.ReadFile(app.filenameWithPath)
	if err != nil {
		return config.createConfig(app.filenameWithPath, aliasConfig) // Either error or nil value
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

	return config.updateAlias(foundAliasConfig, loginServerURL, app.filenameWithPath) // Either error or nil value
}

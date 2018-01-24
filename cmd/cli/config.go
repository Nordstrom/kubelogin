package main

import (
	"io/ioutil"
	"log"
	"net/url"
	"os"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Config contains the array of aliases (AliasConfig)
type Config struct {
	Aliases []*AliasConfig `yaml:"aliases"`
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
	_, e := os.Stat(onDiskFile) // Does config file exist?
	if os.IsNotExist(e) {       // Create file
		fh, err := os.Create(onDiskFile)
		if err != nil {
			return errors.Wrap(err, "failed to create file in root directory")
		}
		_ = fh.Close()
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

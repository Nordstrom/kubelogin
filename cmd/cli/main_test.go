package main

import (
	"fmt"
	"log"
	"net/http"
	"os/user"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFindFreePort(t *testing.T) {
	Convey("findFreePort", t, func() {
		Convey("should find a free port and return a port as a string if there is no error", func() {
			port, _ := findFreePort()
			So(port, ShouldNotEqual, nil)
		})
	})
}
func TestMakeExchange(t *testing.T) {
	Convey("makeExchange", t, func() {
		Convey("should return an error if the hostFlag is not set or incorrect", func() {
			err := makeExchange("hoopla")
			So(err, ShouldNotEqual, nil)
		})
		Convey("should return an error if the token can't be found", func() {
			serverFlag = "www.google.com"
			err := makeExchange("hoopla")
			So(err, ShouldNotEqual, nil)
		})
	})
}

func TestGenerateAuthURL(t *testing.T) {
	Convey("generateAuthURL", t, func() {
		Convey("should return a url with a port based on the findFreePort function", func() {
			url, _, _ := generateAuthURL()
			So(url, ShouldNotEqual, nil)
		})
	})
}

func TestCreateMux(t *testing.T) {
	Convey("createMux", t, func() {
		Convey("should return a new mux", func() {
			testMux := createMux()
			newMux := http.NewServeMux()
			So(testMux, ShouldHaveSameTypeAs, newMux)
		})
	})
}

func TestConfigureKubectl(t *testing.T) {
	Convey("configureKubectl", t, func() {
		userFlag = "auth_user"
		Convey("should return nil upon setting the token correctly", func() {
			err := configureKubectl("hoopla")
			So(err, ShouldEqual, nil)
		})
		Convey("should return an error when running the command with no user defined", func() {
			userFlag = ""
			err := configureKubectl("hoopla")
			So(err, ShouldNotEqual, nil)
		})
	})
}

func TestConfigureFile(t *testing.T) {
	Convey("configureFile", t, func() {
		user, err := user.Current()
		if err != nil {
			log.Fatalf("Could not determine current user of this system. Err: %v", err)
		}
		filenameWithPath = fmt.Sprintf("%s/.test.yaml", user.HomeDir)
		Convey("should return nil if a file was able to be configured", func() {
			err := configureFile()
			So(err, ShouldEqual, nil)
		})
		Convey("should return an err if a file failed to be configured", func() {
			filenameWithPath = ""
			err := configureFile()
			So(err, ShouldNotEqual, nil)
		})
	})
}

func TestGetConfigSettings(t *testing.T) {
	Convey("getConfigSettings", t, func() {
		user, err := user.Current()
		if err != nil {
			log.Fatalf("Could not determine current user of this system. Err: %v", err)
		}
		filenameWithPath = fmt.Sprintf("%s/.test.yaml", user.HomeDir)
		Convey("should return nil upon finding an existing alias", func() {
			err := getConfigSettings("prod")
			So(err, ShouldEqual, nil)
		})
		Convey("should return an error if no alias is found", func() {
			err := getConfigSettings("fail")
			So(err, ShouldNotEqual, nil)
		})
		Convey("should return an error if the file is not found", func() {
			filenameWithPath = ""
			err := getConfigSettings("fail")
			So(err, ShouldNotEqual, nil)
		})
	})
}
func TestCreateConfig(t *testing.T) {
	Convey("getConfigSettings", t, func() {
		user, err := user.Current()
		if err != nil {
			log.Fatalf("Could not determine current user of this system. Err: %v", err)
		}
		filenameWithPath = fmt.Sprintf("%s/.test.yaml", user.HomeDir)
		var config Config
		var aliasConfig AliasConfig
		Convey("should return nil upon creating the config file", func() {
			err := config.createConfig(aliasConfig)
			So(err, ShouldEqual, nil)
		})
	})
}

func TestNewAliasConfig(t *testing.T) {
	Convey("newAliasConfig", t, func() {
		user, err := user.Current()
		if err != nil {
			log.Fatalf("Could not determine current user of this system. Err: %v", err)
		}
		filenameWithPath = fmt.Sprintf("%s/.test.yaml", user.HomeDir)
		var config Config
		Convey("should return nil upon putting in a new entry into the config file", func() {
			aliasFlag = "test"
			serverFlag = "testServer"
			err := config.newAliasConfig()
			So(err, ShouldEqual, nil)
		})
	})
}
func TestUpdateAlias(t *testing.T) {
	Convey("updateAlias", t, func() {
		user, err := user.Current()
		if err != nil {
			log.Fatalf("Could not determine current user of this system. Err: %v", err)
		}
		filenameWithPath = fmt.Sprintf("%s/.test.yaml", user.HomeDir)
		var config Config
		var newAliasConfig AliasConfig
		newAliasConfig.BaseURL = "testServer"
		newAliasConfig.Alias = "prod"
		newAliasConfig.KubectlUser = "testuser"
		config.Aliases = append(config.Aliases, &newAliasConfig)
		Convey("should return nil upon updating an entry in the config file", func() {
			aliasFlag = "prod"
			userFlag = "test"
			err := config.updateAlias(0)
			So(err, ShouldEqual, nil)
		})
	})
}

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os/user"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	yaml "gopkg.in/yaml.v2"
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
		var app app
		Convey("should return an error if the hostFlag is not set or incorrect", func() {
			err := app.makeExchange("hoopla")
			So(err, ShouldNotEqual, nil)
		})
		Convey("should return an error if the token can't be found", func() {
			kubeloginServerBaseURL = "www.google.com"
			err := app.makeExchange("hoopla")
			So(err, ShouldNotEqual, nil)
		})
	})
}

func TestGenerateAuthURL(t *testing.T) {
	Convey("generateAuthURL", t, func() {
		var app app
		Convey("should return a url with a port based on the findFreePort function", func() {
			url, _, _ := app.generateAuthURL()
			So(url, ShouldNotEqual, nil)
		})
	})
}

func TestCreateMux(t *testing.T) {
	Convey("createMux", t, func() {
		var app app
		Convey("should return a new mux", func() {
			testMux := createMux(app)
			newMux := http.NewServeMux()
			So(testMux, ShouldHaveSameTypeAs, newMux)
		})
	})
}

func TestConfigureKubectl(t *testing.T) {
	Convey("configureKubectl", t, func() {
		userFlag = "auth_user"
		var app app
		app.kubectlUser = "test"
		Convey("should return nil upon setting the token correctly", func() {
			err := app.configureKubectl("hoopla")
			So(err, ShouldEqual, nil)
		})
		Convey("should return an error when running the command with no user defined", func() {
			app.kubectlUser = ""
			err := app.configureKubectl("hoopla")
			So(err, ShouldNotEqual, nil)
		})
	})
}

func TestEditToken(t *testing.T) {
	Convey("editToken", t, func() {
		var app app
		app.kubectlUser = "nonprod_oidc"
		token := "fancyToken"
		Convey("should return nil upon setting the token correctly", func() {
			kyaml, err := constructYaml()
			if err != nil {
				t.Error(err)
			}
			var u k8User
			y := editToken(kyaml, app, token)
			ok, u := findUserStruct(y, app.kubectlUser)
			if !ok {
				u.User["token"] = ""
			}
			So(u.User["token"], ShouldEqual, "fancyToken")
		})
		Convey("should construct a new user with token if user does not exist", func() {
			kyaml, err := constructYaml()
			if err != nil {
				t.Error(err)
			}
			var u k8User
			app.kubectlUser = "doesNotExist"
			y := editToken(kyaml, app, token)
			ok, u := findUserStruct(y, app.kubectlUser)
			if !ok {
				u.User["token"] = ""
			}
			So(u.User["token"], ShouldEqual, "fancyToken")
		})
	})
}

func TestConfigureFile(t *testing.T) {
	Convey("configureFile", t, func() {
		var app app
		user, err := user.Current()
		if err != nil {
			log.Fatalf("Could not determine current user of this system. Err: %v", err)
		}
		app.filenameWithPath = fmt.Sprintf("%s/.test.yaml", user.HomeDir)
		fakeURL, _ := url.Parse("bar")
		Convey("should return nil if a file was able to be configured", func() {
			err := app.configureFile("foo", fakeURL, "foobar")
			So(err, ShouldEqual, nil)
		})
		Convey("should return an err if a file failed to be configured", func() {
			app.filenameWithPath = ""
			err := app.configureFile("foo", fakeURL, "foobar")
			So(err, ShouldNotEqual, nil)
		})
	})
}

func TestGetConfigSettings(t *testing.T) {
	Convey("getConfigSettings", t, func() {
		var app app
		user, err := user.Current()
		if err != nil {
			log.Fatalf("Could not determine current user of this system. Err: %v", err)
		}
		app.filenameWithPath = fmt.Sprintf("%s/.test.yaml", user.HomeDir)
		Convey("should return nil upon finding an existing alias", func() {
			err := app.getConfigSettings("test")
			So(err, ShouldEqual, nil)
		})
		Convey("should return an error if no alias is found", func() {
			err := app.getConfigSettings("fail")
			So(err, ShouldNotEqual, nil)
		})
		Convey("should return an error if the file is not found", func() {
			app.filenameWithPath = ""
			err := app.getConfigSettings("fail")
			So(err, ShouldNotEqual, nil)
		})
	})
}

func TestCreateConfig(t *testing.T) {
	Convey("getConfigSettings", t, func() {
		var app app
		user, err := user.Current()
		if err != nil {
			log.Fatalf("Could not determine current user of this system. Err: %v", err)
		}
		app.filenameWithPath = fmt.Sprintf("%s/.test.yaml", user.HomeDir)
		var config Config
		var aliasConfig AliasConfig
		Convey("should return nil upon creating the config file", func() {
			err := config.createConfig(app.filenameWithPath, aliasConfig)
			So(err, ShouldEqual, nil)
		})
	})
}

func TestNewAliasConfig(t *testing.T) {
	Convey("newAliasConfig", t, func() {
		var app app
		user, err := user.Current()
		if err != nil {
			log.Fatalf("Could not determine current user of this system. Err: %v", err)
		}
		app.filenameWithPath = fmt.Sprintf("%s/.test.yaml", user.HomeDir)
		var config Config
		Convey("should return nil upon putting in a new entry into the config file", func() {
			aliasFlag = "test"
			kubeloginServerBaseURL = "testServer"
			newConfig := config.newAliasConfig("foo", "bar", "foobar")
			So(newConfig, ShouldNotBeEmpty)
		})
	})
}

func TestUpdateAlias(t *testing.T) {
	Convey("updateAlias", t, func() {
		var app app
		user, err := user.Current()
		if err != nil {
			log.Fatalf("Could not determine current user of this system. Err: %v", err)
		}
		app.filenameWithPath = fmt.Sprintf("%s/.test.yaml", user.HomeDir)
		var config Config
		var newAliasConfig AliasConfig
		newAliasConfig.BaseURL = "bar"
		newAliasConfig.Alias = "test"
		newAliasConfig.KubectlUser = "testuser"
		config.Aliases = append(config.Aliases, &newAliasConfig)
		fakeURL, _ := url.Parse("bar")
		Convey("should return nil upon updating an entry in the config file", func() {
			aliasFlag = "test"
			userFlag = "test"
			err := config.updateAlias(&newAliasConfig, fakeURL, app.filenameWithPath)
			So(err, ShouldEqual, nil)
		})
	})
}

// Must search for User struct because slices have no guaranteed order
func findUserStruct(y kubeYAML, p string) (bool, k8User) {
	for _, v := range y.Users {
		if v.Name == p {
			return true, v
		}
	}
	return false, k8User{}
}

// Attempting to unmarshal a text string as YAML failed due to illegal characters
func constructYaml() (kubeYAML, error) {
	var kyaml kubeYAML
	tokenKube, err := ioutil.ReadFile("testdata.yml")
	if err != nil {
		return kyaml, err
	}
	err = yaml.Unmarshal(tokenKube, &kyaml)
	return kyaml, err
}

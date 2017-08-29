package main

import (
	"log"
	"net/http"
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

func TestConfigureFile(t *testing.T) {
	Convey("configureFile", t, func() {
		Convey("should return nil if the command executes correctly", func() {
			err := configureKubectl("hoopla")
			log.Print(err)
			So(err.Error(), ShouldEqual, "exit status 1")
		})
	})
}

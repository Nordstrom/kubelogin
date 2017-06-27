package main

import(
    "testing"
    . "github.com/smartystreets/goconvey/convey"
)

func TestSomething(t *testing.T) {
    Convey("my application", t, func () {
        Convey("it does this", func ()  {
            So(1, ShouldEqual, 0)
        })
        Convey("it does that", func ()  {

        })
    })
}

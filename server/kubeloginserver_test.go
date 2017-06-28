package main

import(
    "net/url"
    "fmt"
    "io/ioutil"
    "strings"
    "net/http/httptest"
    "testing"
    "net/http"
    . "github.com/smartystreets/goconvey/convey"
)
func handler(w http.ResponseWriter, r *http.Request) {
	/*
	   dictates where the request should go
	*/
	if r.URL.Path != "/" {
		http.Error(w, "404 not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		getHandler(w, r)
	case "POST":
		postHandler(w, r)
	default:
		fmt.Fprintf(w, "Only GET and POST methods are supported")
	}
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hoopla")
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	/*
	   this is only for the specific example i found online. once i know what each method needs to do this will be changed
	*/
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "Hello, POST method. Parseform() err: %v", err)
		return
	}
	switch r.FormValue("post_from") {
	case "web":
		fmt.Fprintf(w, "Post from website! r.PostFrom = %v\n", r.PostForm)
		s := r.FormValue("key")
		fmt.Fprintf(w, "key = %s, len = %v\n", s, r.PostForm)

	case "client":
		fmt.Fprintf(w, "Post from client! r.PostForm = %v\n", r.PostForm)
	default:
		fmt.Fprint(w, "Unkown Post source :( \n")
	}
}
func TestSpecs(t *testing.T) {
    Convey("Kubelogin Server Tests", t, func () {
        testServer := httptest.NewServer(http.HandlerFunc(handler))
        response,_ :=http.Get(testServer.URL)
        Convey("The server should get a 200 response", func ()  {
            So(response.StatusCode, ShouldEqual, 200)
        })
        Convey("The postHandler shouldn't be able to know who the source is", func ()  {
            resp,_ := http.Post(testServer.URL, "application/x-www-form-urlencoded", strings.NewReader("hello world"))
            b,_:=ioutil.ReadAll(resp.Body)
            So(string(b), ShouldEqual, "Unkown Post source :( \n")
        })
        Convey("The postHandler should know that the source is a client", func ()  {
            v:=url.Values{}
            v.Set("post_from", "client")
            s := v.Encode()
            req, _ := http.NewRequest("POST", testServer.URL, strings.NewReader(s))
        	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
        	c := &http.Client{}
        	resp, _ := c.Do(req)
        	data, _ := ioutil.ReadAll(resp.Body)
            So(string(data), ShouldEqual, "Post from client! r.PostForm = map[post_from:[client]]\n",)
        })
        Convey("The get request should be hoopla", func ()  {
            response, _:=http.Get(testServer.URL)
            body,_:=ioutil.ReadAll(response.Body)
            So(string(body), ShouldEqual, "hoopla")
        })
    })
}

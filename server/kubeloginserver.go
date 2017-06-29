package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func cliPostHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	mappedItems := r.PostForm
	id := strings.Join(mappedItems["clientID"], "")
	port := strings.Join(mappedItems["portNum"], "")
	if id == "" || port == "" {
		fmt.Fprint(w, "400 Bad request")
		//may be better to do a log.Fatal() for this error
		return
	}
	//will need to verify the Id based on some predetermined location that it's saved in. this will determine if it proceeds or returns a 404
	realID := verifyID(id)
	if !realID {
		fmt.Fprint(w, "404 Not found")
		return
	}
	idport := (id + "," + port)
	//current way of proving that we can get the id and the port num to the server
	fmt.Fprint(w, idport)
}

func authPostHandler(w http.ResponseWriter, r *http.Request) {
	//need infor for auth server
	fmt.Fprint(w, "good news everyone")
}

func authPostJwtHandler(w http.ResponseWriter, r *http.Request) {
	//need info for auth server
	fmt.Fprint(w, "good news everyone")
}

func postToAuthHandler(id string, secret string, authCode string) error {
	//need info for auth server
	fmt.Print("send to the auth server")
	return nil
}

func postTokenToCliHandler(jwtToken string) error {
	//will be easier to implement with a base cli created to communicate with this server
	fmt.Print("send back to the client")
	return nil
}

func verifyID(id string) bool {
	//this will eventually grab the id from wherever it's stored and return true or false if it's valid or not
	if id == "myclient" {
		return true
	}
	return false
}

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

func main() {
	/*
	   sets up a new mux. the default handler at root is handler and if there's an, log it
	*/
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	if err := http.ListenAndServe(":8000", mux); err != nil {
		log.Fatal(err)
	}
}

var htmlStr = `
html stuff
`

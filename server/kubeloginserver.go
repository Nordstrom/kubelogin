package main

import (
	"fmt"
	"log"
	"net/http"
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

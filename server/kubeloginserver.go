package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
)

func responseHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello world")
}

func cliGetHandler(w http.ResponseWriter, r *http.Request) {

	u, _ := url.Parse(r.URL.String())
	mappedItems, _ := url.ParseQuery(u.RawQuery)
	id := mappedItems["clientID"][0]
	port := mappedItems["port"][0]
	if id == "" || port == "" {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		//may be better to do a log.Fatal() for this error
		return
	}
	//will need to verify the Id based on some predetermined location that it's saved in. this will determine if it proceeds or returns a 404
	idport := (id + "," + port)
	//current way of proving that we can get the id and the port num to the server
	fmt.Fprint(w, idport)
}

func authPostHandler(w http.ResponseWriter, r *http.Request) {
	//need info for auth server
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

func main() {
	/*
	   sets up a new mux. the default handler at root is handler and if there's an, log it
	*/
	mux := http.NewServeMux()
	mux.HandleFunc("/login", cliGetHandler)
	if err := http.ListenAndServe(":8000", mux); err != nil {
		log.Fatal(err)
	}
}

var htmlStr = `
html stuff
`

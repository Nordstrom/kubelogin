package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
)

func responseHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello world")
}

var (
	clusterName string
	port        string
)

func cliGetRedirectHandler(w http.ResponseWriter, r *http.Request) {
	redirectURL := "https://nauth-test.auth0.com/login?client=" + os.Getenv("CLIENT_ID")
	http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)

}

func cliGetHandler(w http.ResponseWriter, r *http.Request) {
	ipAddr := r.RemoteAddr
	link, _ := url.Parse(r.URL.String())
	mappedItems, _ := url.ParseQuery(link.RawQuery)
	clusterName = mappedItems["clusterID"][0]
	port = mappedItems["port"][0]
	if clusterName == "" || port == "" {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		//may be better to do a log.Fatal() for this error
		return
	}
	//will need to verify the Id based on some predetermined location that it's saved in. this will determine if it proceeds or returns a 404
	namePortIP := (clusterName + "," + port + "," + ipAddr)
	//current way of proving that we can get the id and the port num to the server
	fmt.Fprint(w, namePortIP)
	//log.Print(os.Getenv("CLIENT_ID"))
	//redirectURL := "https://nauth-test.auth0.com/login?client=" + os.Getenv("CLIENT_ID")
	//log.Print(redirectURL)
	//http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
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
	//need info on how auth server handles post requests
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
			   sets up a new mux. upon a user clicking the link to our server, it will be handled by the cliGetHandler.
		       When the auth server posts to our server it should be controlled by the authPostHandler.
	*/
	mux := http.NewServeMux()
	//mux.HandleFunc("/authcode/", authPostHandler)
	mux.HandleFunc("/login/", cliGetHandler)
	if err := http.ListenAndServe(":8000", mux); err != nil {
		log.Fatal(err)
	}
}

var htmlStr = `
html stuff
`

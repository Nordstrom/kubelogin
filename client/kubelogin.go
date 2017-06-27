package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
)

func main() {
	/*
	   called upon executing kubelogin. This will pick apart the flags passed in
	   through the commandline. Uses: --cluster and --server or -cluster and -server
	*/

	clusterName := flag.String("cluster", "cluster", "cluster name to be connected to")
	url := flag.String("server", "url", "url for the server you're sending to")
	flag.Parse()

	conn, err := net.Dial("tcp", *url)
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}
	fmt.Fprintf(conn, *clusterName+"\n")
	message, _ := bufio.NewReader(conn).ReadString('\n')
	fmt.Print("Message received from server: " + message)

}

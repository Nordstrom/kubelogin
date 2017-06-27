package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	/*
	   Currently listening on localhost. This needs to be changed upon deployment.
	   If it cannot listen to the server given it will print an error but should it close
	   the listener? Currently runs forever unless a ctrl-c interrupt is given
	*/

	fmt.Println("launching server...")
	ln, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		fmt.Println("Error:", err.Error())
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			/*
			   if there is an error accepting the connection, close it.
			   but this should keep the server running. previous code had the
			   server quitting upon an error accepting a connection
			*/
			fmt.Println("Error:", err.Error())
			conn.Close()
		}
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	message, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		/*
		   if there is an error reading from the client,
		   log the error to the server then write the error back to the client
		   and close the connection
		*/
		fmt.Println("Error reading:", err.Error())
		conn.Write([]byte("Error reading: " + err.Error() + "\n"))
		conn.Close()
	}
	fmt.Println("Message received from client: ", (message))
	authorizedMessage := authorize(message)
	conn.Write([]byte(authorizedMessage + "\n"))
	conn.Close()
}

func authorize(clusterName string) string {
	//talk with authorization server TBI
	return clusterName
}

package backend

import (
	"fmt"
	"net"
)

func SetupServer() {
	listener, err := net.Listen("tcp", "0.0.0.0:8080")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println("Error:", err)
		}
	}(listener)

	fmt.Println("Server is listening on port 8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Println("Error:", err)
		}
	}(conn)

	fmt.Println("Client connected")
	fmt.Println(conn.RemoteAddr().String())
}
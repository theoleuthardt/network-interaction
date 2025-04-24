package backend

import (
	"fmt"
	"net"
	"network-interaction/utils"
	_ "network-interaction/utils"
)

func SetupServer() {
	listener, err := net.Listen("tcp", "0.0.0.0:56867")

	if err != nil {
		utils.LogError(err.Error())
		return
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			utils.LogError(err.Error())
		}
	}(listener)

	utils.LogInfo("Server is listening on port 56867")

	for {
		conn, err := listener.Accept()
		if err != nil {
			utils.LogError(err.Error())
			continue
		}

		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			utils.LogError(err.Error())
		}
	}(conn)

	fmt.Println("Client connected")
	fmt.Println(conn.RemoteAddr().String())
}

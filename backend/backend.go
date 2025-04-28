package backend

import (
	"fmt"
	"net"
	"network-interaction/utils"
	_ "network-interaction/utils"
	"os"
	"strconv"
	"time"
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

	// Start the async function to write queue length into our silly file
	go writeQueueLength()

	for {
		conn, err := listener.Accept()
		if err != nil {
			utils.LogError(err.Error())
			continue
		}

		go handleClient(conn)
	}
}

var (
	queue uint = 0
)

func handleClient(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			utils.LogError(err.Error())
		}
	}(conn)
	queue++
	utils.LogInfo("Queue length: " + fmt.Sprint(queue))
}

// writeQueueLength asynchronously writes the current queue length to a file every 0.5 seconds
func writeQueueLength() {
	ticker := time.NewTicker(100 * time.Millisecond) //every 0.1 seconds!
	defer ticker.Stop()

	for range ticker.C {
		file, err := os.OpenFile("queue_length.txt", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			utils.LogError(err.Error())
			continue
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				utils.LogError(err.Error())
			}
		}(file)

		_, err = file.WriteString(strconv.Itoa(int(queue)))
		if err != nil {
			utils.LogError(err.Error())
		}
	}
}

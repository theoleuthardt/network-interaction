package backend

import (
	"net"
	"network-interaction/utils"
	_ "network-interaction/utils"
	"os"
	"strconv"
	"strings"
)

// SetupServer initializes the server, listens for TCP connections, and handles incoming client requests.
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

	// Initialize the files to reset them ^^ silly
	writeToFile("fast_send.txt", &fastQueue)
	writeToFile("dynamic_send.txt", &dynamicQueue)
	writeToFile("slow_send.txt", &slowQueue)

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
	fastQueue    uint = 0
	dynamicQueue uint = 0
	slowQueue    uint = 0
)

// handleClient parses incoming TCP packets, extracts the message as a string, and passes it to the handleMessage function.
func handleClient(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			utils.LogError(err.Error())
		}
	}(conn)

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		utils.LogError(err.Error())
		return
	}

	message := string(buffer[:n])
	handleMessage(message)
}

// handleMessage receives the tcp packages contents as a string and sorts it out accordingly
func handleMessage(message string) {
	// Determine which queue and file to write based on the incoming messages content
	if strings.HasPrefix(message, "fast") {
		fastQueue++
		writeToFile("fast_send.txt", &fastQueue)
	} else if strings.HasPrefix(message, "dynamic") {
		dynamicQueue++
		writeToFile("dynamic_send.txt", &dynamicQueue)
	} else if strings.HasPrefix(message, "slow") {
		slowQueue++
		writeToFile("slow_send.txt", &slowQueue)
	} else {
		//Uncalled-for message? Stop texting me weirdo...
		utils.LogInfo("Unknown message type: " + message)
	}
}

// writeToFile writes the queue length of the given variable to its designated file, creating the file if it doesn't exist.
func writeToFile(fileName string, queue *uint) {
	file, err := os.OpenFile(fileName, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		utils.LogError(err.Error())
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			utils.LogError(err.Error())
		}
	}(file)

	_, err = file.WriteString(strconv.Itoa(int(*queue)))
	if err != nil {
		utils.LogError(err.Error())
	}
}

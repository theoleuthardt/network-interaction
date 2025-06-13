package backend

import (
	"encoding/json"
	"fmt"
	"net"
	"network-interaction/utils"
	"strings"
	"time"
)

var (
	fastQueue    uint = 50
	dynamicQueue uint = 50
	slowQueue    uint = 50

	serverPort  int
	peerAddress string
)

type QueueState struct {
	FastQueue    uint `json:"fast"`
	DynamicQueue uint `json:"dynamic"`
	SlowQueue    uint `json:"slow"`
}

// SetupServer starts the server and peer discovery
func SetupServer(messageChan chan string) {
	// Find available port for server
	serverPort = utils.FindAvailablePort(50500, 50600)
	utils.LogInfo(fmt.Sprintf("Starting server on port %d", serverPort))

	// Start TCP server
	go startTCPServer(messageChan)

	// Start peer discovery
	func() {
		portRange := make([]int, 0, 100)
		for port := 50500; port < 50600; port++ {
			portRange = append(portRange, port)
		}
		peerAddress = utils.DiscoverPeers(serverPort, portRange, func() string { return peerAddress })
		utils.LogInfo("Peer discovery complete, connected to: " + peerAddress)
	}()

	// Start message senders
	utils.StartMessageSenders(func() string { return peerAddress })

	// Start queue management
	sendQueueState(messageChan)
	go countdownQueues()
	go sendQueueStatePeriodically(messageChan, 100*time.Millisecond)
}

// Start TCP server
func startTCPServer(messageChan chan string) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
	if err != nil {
		utils.LogError("Failed to start server: " + err.Error())
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn, messageChan)
	}
}

// Handle incoming TCP connections
func handleConnection(conn net.Conn, messageChan chan string) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return
	}

	message := string(buffer[:n])

	// Handle discovery requests
	if strings.HasPrefix(message, "DISCOVER ") {
		if peerAddress != "" {
			return // Already connected
		}

		conn.Write([]byte("PEER_RESPONSE"))

		// Extract port and set peer address
		if port := utils.ExtractPortFromDiscoverMessage(message); port != "" {
			ip := utils.GetIPFromRemoteAddr(conn.RemoteAddr().String())
			peerAddress = net.JoinHostPort(ip, port)
			utils.LogInfo("Accepted peer: " + peerAddress)
		}
		return
	}

	// Update queues based on message content
	if strings.Contains(message, "fast") {
		fastQueue++
	} else if strings.Contains(message, "dynamic") {
		dynamicQueue++
	} else if strings.Contains(message, "slow") {
		slowQueue++
	}
}

// Countdown queue values every second
func countdownQueues() {
	for {
		time.Sleep(1 * time.Second)

		if fastQueue > 100 {
			fastQueue = 0
		}
		if dynamicQueue > 100 {
			dynamicQueue = 0
		}
		if slowQueue > 100 {
			slowQueue = 0
		}

		if fastQueue > 0 {
			fastQueue--
		}
		if dynamicQueue > 0 {
			dynamicQueue--
		}
		if slowQueue > 0 {
			slowQueue--
		}
	}
}

// Send queue state every x seconds
func sendQueueStatePeriodically(messageChan chan string, interval time.Duration) {
	for {
		time.Sleep(interval)
		sendQueueState(messageChan)
	}
}

// Send queue state to frontend
func sendQueueState(messageChan chan string) {
	state := QueueState{
		FastQueue:    fastQueue,
		DynamicQueue: dynamicQueue,
		SlowQueue:    slowQueue,
	}

	data, err := json.Marshal(state)
	if err != nil {
		return
	}

	messageChan <- string(data)
}

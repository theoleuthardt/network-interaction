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
	connected   = false
	// A list of all discovered potential peers
	discoveredPeers []string
)

type QueueState struct {
	FastQueue       uint     `json:"fast"`
	DynamicQueue    uint     `json:"dynamic"`
	SlowQueue       uint     `json:"slow"`
	Connected       bool     `json:"connected"`
	DiscoveredPeers []string `json:"discovered_peers"`
}

// SetupServer starts the server and peer discovery
func SetupServer(messageChan chan string, sgnChan chan string) {
	// Find available port for server
	serverPort = utils.FindAvailablePort(50500, 50600)
	utils.LogInfo(fmt.Sprintf("Starting server on port %d", serverPort))

	// Start TCP server
	go startTCPServer()
	// Start queue management
	go sendQueueStatePeriodically(messageChan, 100*time.Millisecond)
	// Listen to the messages from the frontend
	go func() {
		for data := range sgnChan {
			if utils.TryToConnectToPeer(data, serverPort) {
				peerAddress = data
				connected = true
			}
		}
	}()

	// Start peer discovery
	go func() {
		for {
			for connected {
				discoveredPeers = []string{}
				time.Sleep(500 * time.Millisecond)
			}

			portRange := make([]int, 0, 100)
			for port := 50500; port < 50600; port++ {
				portRange = append(portRange, port)
			}
			discoveredPeers = utils.DiscoverPeers(serverPort, portRange)
			utils.LogInfo(fmt.Sprintf("Discovered peers: %v", discoveredPeers))

			// Wait 2.5 seconds until next discovery (or until connected)
			for i := 0; i < 10; i++ {
				if connected {
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()

	go countdownQueues()
	go monitorConnection()
	utils.StartMessageSenders(func() string { return peerAddress })
}

func monitorConnection() {
	for {
		time.Sleep(1 * time.Second)

		if peerAddress != "" {
			// Try to connect to peer to check if it's still alive
			conn, err := net.DialTimeout("tcp", peerAddress, 2*time.Second)
			if err != nil {
				if connected {
					utils.LogInfo("Peer connection lost: " + peerAddress)
					connected = false
					peerAddress = ""
				}
			} else {
				conn.Close()
				if !connected {
					utils.LogInfo("Peer connection restored: " + peerAddress)
					connected = true
				}
			}
		}
	}
}

// Start TCP server
func startTCPServer() {
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
		go handleConnection(conn)
	}
}

// Handle incoming TCP connections
func handleConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return
	}

	message := string(buffer[:n])
	// Handle discovery requests
	if strings.HasPrefix(message, "DISCOVER_SYN") {
		if peerAddress != "" {
			return // Already connected
		}
		conn.Write([]byte("DISCOVER_ACK"))
		return
	}

	if strings.HasPrefix(message, "CONNECT_REQ") {
		if peerAddress != "" {
			return // Already connected
		}

		parts := strings.Split(message, ":")
		address := parts[1]
		port := parts[2]
		peerAddress = fmt.Sprintf("%s:%s", address, port)
		connected = true

		conn.Write([]byte("CONNECT_OK"))
		utils.LogInfo("Connected to peer: " + peerAddress)
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
		if !connected {
			fastQueue = 50
			dynamicQueue = 50
			slowQueue = 50
			continue
		}

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
		sendDataToFrontend(messageChan)
	}
}

// Send queue state to frontend
func sendDataToFrontend(messageChan chan string) {
	state := QueueState{
		FastQueue:       fastQueue,
		DynamicQueue:    dynamicQueue,
		SlowQueue:       slowQueue,
		Connected:       connected,
		DiscoveredPeers: discoveredPeers,
	}

	data, err := json.Marshal(state)
	if err != nil {
		return
	}

	messageChan <- string(data)
}

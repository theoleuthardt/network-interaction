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

	serverPort      int
	peerAddress     string
	connected       = false
	discoveredPeers []string
)

// QueueState represents the current state of all message queues and network connections.
// Used for communication between backend and frontend via JSON serialization.
type QueueState struct {
	FastQueue       uint     `json:"fast"`
	DynamicQueue    uint     `json:"dynamic"`
	SlowQueue       uint     `json:"slow"`
	Connected       bool     `json:"connected"`
	DiscoveredPeers []string `json:"discovered_peers"`
}

// SetupServer initializes and starts the backend server with peer discovery.
// Launches TCP server, queue state broadcaster, peer discovery, and frontend signal listener.
// The messageChan is used to send queue state updates to the frontend.
// The sgnChan receives connection and disconnection signals from the frontend.
func SetupServer(messageChan chan string, sgnChan chan string) {
	serverPort = utils.FindAvailablePort(50500, 50600)
	utils.LogInfo(fmt.Sprintf("Starting server on port %d", serverPort))

	go startTCPServer()
	go sendQueueStatePeriodically(messageChan, 100*time.Millisecond)

	go func() {
		for data := range sgnChan {
			if data == "disconnect" {
				utils.LogInfo("Disconnecting from peer: " + peerAddress)
				if peerAddress != "" {
					conn, err := net.DialTimeout("tcp", peerAddress, 2*time.Second)
					if err == nil {
						conn.Write([]byte("DISCONNECT"))
						conn.Close()
					}
				}
				peerAddress = ""
				connected = false
			} else if utils.TryToConnectToPeer(data, serverPort) {
				peerAddress = data
				connected = true
			}
		}
	}()

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

// monitorConnection continuously checks the health of the peer connection.
// Attempts to connect to the peer every second to verify it's still alive.
// Automatically updates connection status when peer becomes unreachable or restored.
func monitorConnection() {
	for {
		time.Sleep(1 * time.Second)

		if peerAddress != "" {
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

// startTCPServer starts a TCP listener on the server port.
// Accepts incoming connections and spawns a goroutine to handle each one.
// Runs indefinitely until the program terminates.
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

// handleConnection processes incoming TCP connection requests.
// Handles discovery requests (DISCOVER_SYN), connection requests (CONNECT_REQ),
// disconnect signals (DISCONNECT), and message queue updates.
func handleConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return
	}

	message := string(buffer[:n])

	if strings.HasPrefix(message, "DISCOVER_SYN") {
		if peerAddress != "" {
			return
		}
		conn.Write([]byte("DISCOVER_ACK"))
		return
	}

	if strings.HasPrefix(message, "CONNECT_REQ") {
		if peerAddress != "" {
			return
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

	if strings.HasPrefix(message, "DISCONNECT") {
		utils.LogInfo("Peer disconnected: " + peerAddress)
		peerAddress = ""
		connected = false
		return
	}
	if strings.Contains(message, "fast") {
		fastQueue++
	} else if strings.Contains(message, "dynamic") {
		dynamicQueue++
	} else if strings.Contains(message, "slow") {
		slowQueue++
	}
}

// countdownQueues decrements all queue values every second when connected.
// Resets queues to 50 when not connected, and to 0 when they exceed 100.
// Runs indefinitely in a goroutine.
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

// sendQueueStatePeriodically broadcasts queue state updates at regular intervals.
// Sends JSON-encoded QueueState to the frontend via messageChan.
// The interval parameter determines how frequently updates are sent.
func sendQueueStatePeriodically(messageChan chan string, interval time.Duration) {
	for {
		time.Sleep(interval)
		sendDataToFrontend(messageChan)
	}
}

// sendDataToFrontend serializes and sends current queue state to the frontend.
// Creates a QueueState struct with current values and sends it as JSON through messageChan.
// Silently ignores marshaling errors.
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

package backend

import (
	"encoding/json"
	"fmt"
	"math/rand"
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
	serverPort = findPort()

	utils.LogInfo(fmt.Sprintf("Starting server on port %d", serverPort))

	// Start TCP server in its own goroutine
	go startTCPServer(messageChan)

	// Start peer discovery and communication
	discoverPeers()

	utils.LogInfo("Peer discovery complete, connected to: " + peerAddress)

	// Send messages in regular intervals
	go shitFuck()

	// Send initial state and start queue countdown
	sendQueueState(messageChan)
	go countdownQueues(messageChan)
	go sendQueueStatePeriodically(messageChan, time.Duration(100)) //Every 100ms
}

// Find an available port
func findPort() int {
	for port := 50500; port < 50600; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			listener.Close()
			return port
		}
	}
	return 50600
}

// Discover other peers by scanning TCP ports
func discoverPeers() {
	for {
		if peerAddress != "" {
			utils.LogInfo("Already connected to peer: " + peerAddress)
			return
		}
		for port := 50500; port < 50600; port++ {
			// Skip our own port
			if port == serverPort {
				continue
			}

			// Try localhost first (most common case)
			if tryConnectToPeer("127.0.0.1", port) {
				return
			}
		}
		utils.LogInfo("No peers found, retrying in 10 seconds...")
		time.Sleep(10 * time.Second)
	}
}

// Try to connect to a potential peer
func tryConnectToPeer(ip string, port int) bool {
	address := net.JoinHostPort(ip, fmt.Sprintf("%d", port))

	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Send discovery message
	_, err = conn.Write([]byte("DISCOVER " + fmt.Sprintf("%d", serverPort)))
	if err != nil {
		return false
	}

	// Read response
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		return false
	}

	response := string(buffer[:n])
	if response == "PEER_RESPONSE" {
		addPeer(address)
		return true
	}

	return false
}

// Add peer address (only one allowed)
func addPeer(address string) {
	peerAddress = address
	utils.LogInfo("Found peer: " + address)
}

// Start TCP server in its own goroutine
func startTCPServer(messageChan chan string) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
	if err != nil {
		utils.LogError("Failed to start server: " + err.Error())
		return
	}
	defer listener.Close()

	// Accept connections
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
		//ignore if peerAdress is already set
		if peerAddress != "" {
			return
		}

		conn.Write([]byte("PEER_RESPONSE"))

		// If we don't have a peer yet, save this address with the provided port
		// Extract port from the DISCOVER message
		parts := strings.Split(message, " ")
		if len(parts) >= 2 {
			port := parts[1]
			// Get the IP from the remote address and combine with the provided port
			remoteAddr := conn.RemoteAddr().String()
			ip := strings.Split(remoteAddr, ":")[0]
			peerAddress = net.JoinHostPort(ip, port)
			utils.LogInfo("Accepted peer: " + peerAddress)
		}

		return
	}

	// Regular message handling - update queues based on message
	if strings.Contains(message, "fast") {
		fastQueue++
	} else if strings.Contains(message, "dynamic") {
		dynamicQueue++
	} else if strings.Contains(message, "slow") {
		slowQueue++
	}
}

// Countdown queue values every second
func countdownQueues(messageChan chan string) {
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

// Send a message to the connected peer
func sendMessageToPeer(message string) {
	if peerAddress == "" {
		return
	}

	conn, err := net.DialTimeout("tcp", peerAddress, 1*time.Second)
	if err != nil {
		utils.LogError("Failed to connect to peer: " + err.Error())
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte(message))
	if err != nil {
		utils.LogError("Failed to send message to peer: " + err.Error())
	}
}

func shitFuck() {
	// Send fast messages with variation around 200ms base interval
	go func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		for {
			// Base interval 200ms with ±50ms variation (150ms to 250ms)
			baseInterval := 200 * time.Millisecond
			variation := time.Duration(r.Intn(201)-100) * time.Millisecond // -50ms to +50ms
			interval := baseInterval + variation

			time.Sleep(interval)
			if peerAddress != "" {
				sendMessageToPeer("fast message")
			}
		}
	}()

	// Send dynamic messages with configurable chance of short vs long intervals
	go func() {
		// Seed the random number generator
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		for {
			shortIntervalChance := 0.8

			var interval time.Duration
			if r.Float64() < shortIntervalChance {
				interval = 50 * time.Millisecond // Short interval
			} else {
				interval = 3000 * time.Millisecond // Long interval
			}

			time.Sleep(interval)
			if peerAddress != "" {
				sendMessageToPeer("dynamic message")
			}
		}
	}()

	// Send slow messages with variation around 1 second base interval
	go func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		for {
			// Base interval 1000ms with ±200ms variation (800ms to 1200ms)
			baseInterval := 1000 * time.Millisecond
			variation := time.Duration(r.Intn(401)-200) * time.Millisecond // -200ms to +200ms
			interval := baseInterval + variation

			time.Sleep(interval)
			if peerAddress != "" {
				sendMessageToPeer("slow message")
			}
		}
	}()
}

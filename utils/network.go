package utils

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

// GetLocalIP returns the local network interface IP address
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// FindAvailablePort finds an available TCP port in the given range
func FindAvailablePort(startPort, endPort int) int {
	for port := startPort; port < endPort; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			listener.Close()
			return port
		}
	}
	return endPort
}

// IsPeerDiscoverable attempts to connect to a peer and send discovery message
func IsPeerDiscoverable(peer string) bool {
	conn, err := net.DialTimeout("tcp", peer, 1*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Send discovery message
	_, err = conn.Write([]byte("DISCOVER_SYN"))
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

	return string(buffer[:n]) == "DISCOVER_ACK"
}

// ScanSubnetForPeer scans all IPs in a /24 subnet for a peer on given port
func ScanSubnetForPeer(localIP string, port int) []string {
	var (
		discoveredPeers []string
		ip              = net.ParseIP(localIP).To4()
		baseIP          = fmt.Sprintf("%d.%d.%d", ip[0], ip[1], ip[2])
		mu              sync.Mutex
		wg              sync.WaitGroup
	)

	for i := 1; i <= 254; i++ {
		wg.Add(1)
		go func(hostNum int) {
			defer wg.Done()

			targetIP := fmt.Sprintf("%s.%d", baseIP, hostNum)
			if targetIP == localIP {
				return
			}

			peer := net.JoinHostPort(targetIP, fmt.Sprintf("%d", port))
			if IsPeerDiscoverable(peer) {
				mu.Lock()
				discoveredPeers = append(discoveredPeers, peer)
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	return discoveredPeers
}

// DiscoverPeers scans for peers on localhost and subnet
func DiscoverPeers(serverPort int, portRange []int) []string {
	var (
		discoveredPeers []string
		localIP         = GetLocalIP()
		mu              sync.Mutex
		wg              sync.WaitGroup
	)

	for _, port := range portRange {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()

			if p == serverPort {
				return
			}

			peer := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", p))
			if IsPeerDiscoverable(peer) {
				mu.Lock()
				discoveredPeers = append(discoveredPeers, peer)
				mu.Unlock()
			}

			if localIP == "" {
				LogWarning("Local IP not found, skipping subnet scan")
				return
			}

			discoveredPeersFromSubnet := ScanSubnetForPeer(localIP, p)
			mu.Lock()
			discoveredPeers = append(discoveredPeers, discoveredPeersFromSubnet...)
			mu.Unlock()
		}(port)
	}

	wg.Wait() // wait for all goroutines
	return discoveredPeers
}

func TryToConnectToPeer(peer string, myPort int) bool {
	conn, err := net.DialTimeout("tcp", peer, 1*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Send discovery message
	_, err = conn.Write([]byte("CONNECT_REQ:" + GetLocalIP() + ":" + strconv.Itoa(myPort)))
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

	return string(buffer[:n]) == "CONNECT_OK"
}

package utils

import (
	"fmt"
	"net"
	"strings"
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

// TryConnectToPeer attempts to connect to a peer and send discovery message
func TryConnectToPeer(ip string, port, serverPort int) (bool, string) {
	address := net.JoinHostPort(ip, fmt.Sprintf("%d", port))

	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		return false, ""
	}
	defer conn.Close()

	// Send discovery message
	_, err = conn.Write([]byte("DISCOVER " + fmt.Sprintf("%d", serverPort)))
	if err != nil {
		return false, ""
	}

	// Read response
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		return false, ""
	}

	if string(buffer[:n]) == "PEER_RESPONSE" {
		return true, address
	}
	return false, ""
}

// ScanSubnetForPeer scans all IPs in a /24 subnet for a peer on given port
func ScanSubnetForPeer(localIP string, port, serverPort int) (bool, string) {
	ip := net.ParseIP(localIP)
	if ip == nil {
		return false, ""
	}

	ipv4 := ip.To4()
	if ipv4 == nil {
		return false, ""
	}

	baseIP := fmt.Sprintf("%d.%d.%d", ipv4[0], ipv4[1], ipv4[2])
	found := make(chan struct {
		success bool
		address string
	}, 254)

	// Scan all IPs concurrently
	for i := 1; i <= 254; i++ {
		go func(hostNum int) {
			targetIP := fmt.Sprintf("%s.%d", baseIP, hostNum)
			if targetIP != localIP {
				success, addr := TryConnectToPeer(targetIP, port, serverPort)
				found <- struct {
					success bool
					address string
				}{success, addr}
			} else {
				found <- struct {
					success bool
					address string
				}{false, ""}
			}
		}(i)
	}

	// Check results
	for i := 0; i < 254; i++ {
		result := <-found
		if result.success {
			return true, result.address
		}
	}
	return false, ""
}

// DiscoverPeers scans for peers on localhost and subnet
func DiscoverPeers(serverPort int, portRange []int, getPeerAddress func() string) string {
	for {
		localIP := GetLocalIP()
		found := make(chan struct {
			success bool
			address string
		}, len(portRange))

		// Scan all ports concurrently
		for _, port := range portRange {
			if port == serverPort {
				continue
			}

			go func(p int) {
				// Try localhost first
				if success, addr := TryConnectToPeer("127.0.0.1", p, serverPort); success {
					found <- struct {
						success bool
						address string
					}{true, addr}
					return
				}
				// Try subnet if localhost fails
				if localIP != "" {
					success, addr := ScanSubnetForPeer(localIP, p, serverPort)
					found <- struct {
						success bool
						address string
					}{success, addr}
					return
				}
				found <- struct {
					success bool
					address string
				}{false, ""}
			}(port)
		}

		// Check if any port found a peer
		for i := 0; i < len(portRange)-1; i++ { // minus our own port
			result := <-found
			if result.success {
				return result.address
			}
		}

		LogInfo("No peers found, retrying in 10 seconds...")

		// Wait 10 seconds but check for existing peer every 500ms
		for i := 0; i < 20; i++ { // 20 * 500ms = 10 seconds
			time.Sleep(500 * time.Millisecond)

			// Check if someone already connected to us
			if existingPeer := getPeerAddress(); existingPeer != "" {
				LogInfo("Already connected to peer: " + existingPeer)
				return existingPeer
			}
		}
	}
}

// ExtractPortFromDiscoverMessage extracts port from "DISCOVER <port>" message
func ExtractPortFromDiscoverMessage(message string) string {
	parts := strings.Split(message, " ")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// GetIPFromRemoteAddr extracts IP from remote address string
func GetIPFromRemoteAddr(remoteAddr string) string {
	return strings.Split(remoteAddr, ":")[0]
}

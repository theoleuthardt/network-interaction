package utils

import (
	"math/rand"
	"net"
	"time"
)

// SendMessageToPeer sends a message to the specified peer address
func SendMessageToPeer(peerAddress, message string) {
	if peerAddress == "" {
		return
	}

	conn, err := net.DialTimeout("tcp", peerAddress, 1*time.Second)
	if err != nil {
		LogError("Failed to connect to peer: " + err.Error())
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte(message))
	if err != nil {
		LogError("Failed to send message to peer: " + err.Error())
	}
}

// StartMessageSenders starts all message sending goroutines
func StartMessageSenders(getPeerAddress func() string) {
	// Fast messages - ~200ms with ±50ms variation
	go func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		for {
			baseInterval := 200 * time.Millisecond
			variation := time.Duration(r.Intn(101)-50) * time.Millisecond // ±50ms
			time.Sleep(baseInterval + variation)

			if addr := getPeerAddress(); addr != "" {
				SendMessageToPeer(addr, "fast message")
			}
		}
	}()

	// Dynamic messages - 80% short (50ms), 20% long (3s)
	go func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		for {
			var interval time.Duration
			if r.Float64() < 0.8 {
				interval = 50 * time.Millisecond
			} else {
				interval = 3000 * time.Millisecond
			}
			time.Sleep(interval)

			if addr := getPeerAddress(); addr != "" {
				SendMessageToPeer(addr, "dynamic message")
			}
		}
	}()

	// Slow messages - ~1s with ±200ms variation
	go func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		for {
			baseInterval := 1000 * time.Millisecond
			variation := time.Duration(r.Intn(401)-200) * time.Millisecond // ±200ms
			time.Sleep(baseInterval + variation)

			if addr := getPeerAddress(); addr != "" {
				SendMessageToPeer(addr, "slow message")
			}
		}
	}()
}

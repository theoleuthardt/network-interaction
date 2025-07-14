package main

import (
	"network-interaction/backend"
	"network-interaction/frontend"
)

func main() {
	messageChan := make(chan string)

	go backend.SetupServer(messageChan)
	frontend.SetupGUI(messageChan)
}

package main

import (
	"network-interaction/backend"
	"network-interaction/frontend"
)

func main() {
	messageChan := make(chan string)

	signalChan := make(chan string)

	go backend.SetupServer(messageChan, signalChan)
	frontend.SetupGUI(messageChan, signalChan)
}

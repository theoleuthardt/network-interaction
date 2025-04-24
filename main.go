package main

import (
	"network-interaction/backend"
	"network-interaction/frontend"
)

func main() {
	go backend.SetupServer()
	frontend.SetupGUI()
}

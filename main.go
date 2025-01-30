package main

import (
	"network-interaction/backend"
	"network-interaction/frontend"
)

func main() {
	frontend.SetupGUI()
	backend.SetupServer()
}
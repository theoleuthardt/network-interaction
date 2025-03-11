package main

import (
	"network-interaction/backend"
	"network-interaction/frontend"
	"network-interaction/utils"
)

func main() {
	utils.LogError("Test")
	go backend.SetupServer()
	frontend.SetupGUI()
}

package main

import (
	"network-interaction/backend"
	"network-interaction/frontend"
)

func main() {
	go frontend.SetupGUI()
	go backend.SetupServer()

	select {}
}
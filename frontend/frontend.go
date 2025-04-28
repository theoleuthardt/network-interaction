package frontend

import (
	"fmt"
	"log"
	"network-interaction/utils"
	"os"
	"strconv"
	"time"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/widget/material"
)

var (
	queueLength uint
)

func SetupGUI() {
	go func() {
		window := new(app.Window)
		window.Option(app.Title("Network Interaction"))
		err := run(window)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(window *app.Window) error {
	theme := material.NewTheme()
	var ops op.Ops

	// Start the ticker to update the queue length every 0.1 seconds
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// Start a goroutine to continuously increment the queue length
	go incrementQueueLength(window)

	for {
		// Event loop for Gio window
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			// Lock mutex to safely read the queueLength
			queueLabel := material.Body1(theme, fmt.Sprintf("Queue Length: %d", queueLength))

			queueLabel.Alignment = text.Middle
			queueLabel.Layout(gtx)

			e.Frame(gtx.Ops)
		}
	}
}

// incrementQueueLength increments the queue length every 100ms
func incrementQueueLength(window *app.Window) {
	for {
		// Read the file content, ignore errors for simplicity
		fileData, err := os.ReadFile("queue_length.txt")
		if err == nil {
			// Convert the file content to an integer and update the queueLength
			if newQueueLength, err := strconv.Atoi(string(fileData)); err == nil {
				queueLength = uint(newQueueLength)
				utils.LogInfo(fmt.Sprintf("[FRONTEND] QueueLength: %d", queueLength))
			}
		}

		// Invalidate the window to trigger a redraw
		window.Invalidate()

		// Sleep for 100ms before the next increment
		time.Sleep(100 * time.Millisecond)
	}
}

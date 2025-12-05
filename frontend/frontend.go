package frontend

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"os"
	"sync"
	_ "time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/layout"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"network-interaction/utils"
)

var (
	fastSend, dynamicSend, slowSend uint
	darkMode                        = true
	connected                       bool
	DiscoveredPeers                 []string
	windowMutex                     sync.RWMutex
	messageChannel                  chan string
	signalChan                      chan string
	shutdownChan                    chan bool

	fastLabel, dynamicLabel, slowLabel *widget.Label
	peersButtonsContainer              *fyne.Container
	peersButtonsMap                    = map[string]*widget.Button{}
	fastBar, dynamicBar, slowBar       *widget.ProgressBar
	connectionLED                      *fyne.Container
	ledCircle                          *canvas.Circle
	darkModeButton                     *widget.Button
	disconnectButton                   *widget.Button
	window                             fyne.App
	mainWindow                         fyne.Window
)

// QueueState represents the backend's message queue state and network information.
// It contains queue lengths, connection status, and a list of discovered peers.
type QueueState struct {
	FastQueue       uint     `json:"fast"`
	DynamicQueue    uint     `json:"dynamic"`
	SlowQueue       uint     `json:"slow"`
	Connected       bool     `json:"connected"`
	DiscoveredPeers []string `json:"discovered_peers"`
}

// SetupGUI initializes and starts the graphical user interface.
// It accepts message and signal channels for communication with the backend,
// sets up panic recovery, and launches the Fyne application.
func SetupGUI(msgChan chan string, sgnChan chan string) {
	messageChannel = msgChan
	signalChan = sgnChan

	shutdownChan = make(chan bool, 1)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in GUI: %v", r)
			select {
			case shutdownChan <- true:
			default:
			}
		}
	}()
	runFyneApp()
}

// runFyneApp creates and configures the main application window.
// It sets up the theme, window size, initializes UI elements, and starts
// the message handler goroutine before showing the window.
func runFyneApp() {
	os.Setenv("FYNE_SCALE", "1.3")
	window = app.New()
	window.SetIcon(theme.ComputerIcon())

	if darkMode {
		window.Settings().SetTheme(theme.DarkTheme())
	} else {
		window.Settings().SetTheme(theme.LightTheme())
	}

	mainWindow = window.NewWindow("Network Interaction")
	mainWindow.SetFixedSize(false)
	mainWindow.Resize(fyne.NewSize(700, 600))

	initializeUIElements()
	content := createLayout()
	mainWindow.SetContent(content)

	go messageHandler()
	mainWindow.ShowAndRun()
}

// initializeUIElements creates and configures all UI components including
// labels, progress bars, the connection LED indicator, and the dark mode button.
func initializeUIElements() {
	fastLabel = widget.NewLabel("Length: 0")
	dynamicLabel = widget.NewLabel("Length: 0")
	slowLabel = widget.NewLabel("Length: 0")
	peersButtonsContainer = container.New(layout.NewGridWrapLayout(fyne.NewSize(200, 40)))

	fastBar = widget.NewProgressBar()
	fastBar.SetValue(0)
	fastBar.TextFormatter = func() string { return "" }

	dynamicBar = widget.NewProgressBar()
	dynamicBar.SetValue(0)
	dynamicBar.TextFormatter = func() string { return "" }

	slowBar = widget.NewProgressBar()
	slowBar.SetValue(0)
	slowBar.TextFormatter = func() string { return "" }

	ledCircle = canvas.NewCircle(color.RGBA{R: 255, A: 255})
	ledCircle.Resize(fyne.NewSize(18, 18))
	ledCircle.Move(fyne.NewPos(5, 8))
	connectionLED = container.NewWithoutLayout(ledCircle)
	connectionLED.Resize(fyne.NewSize(30, 30))

	darkModeButton = widget.NewButton("🌙", toggleDarkMode)
	if !darkMode {
		darkModeButton.SetText("☀️")
	}

	disconnectButton = widget.NewButton("Disconnect", func() {
		signalChan <- "disconnect"
	})
	disconnectButton.Importance = widget.DangerImportance
	disconnectButton.Hide()
}

// updateLEDColor updates the connection status LED indicator.
// Sets the LED to green when connected, red when disconnected.
func updateLEDColor(connected bool) {
	if connected {
		ledCircle.FillColor = color.RGBA{G: 255, A: 255}
	} else {
		ledCircle.FillColor = color.RGBA{R: 255, A: 255}
	}
	ledCircle.Refresh()
}

// createLayout constructs the main window layout with header, queue visualizations,
// and the discovered peers section. Returns a padded container with all UI elements.
func createLayout() *fyne.Container {
	title := widget.NewLabel("Buffer Queue Visualisation")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Resize(fyne.NewSize(200, 50))

	sizedLEDContainer := container.New(layout.NewGridWrapLayout(fyne.NewSize(30, 30)), connectionLED)

	headerRight := container.NewHBox(
		widget.NewLabel("Connection Status:"),
		sizedLEDContainer,
		disconnectButton,
		widget.NewSeparator(),
		darkModeButton,
	)

	headerLeft := container.NewHBox(title)
	header := container.NewBorder(nil, nil, headerLeft, headerRight)
	queuesContainer := container.NewHBox(
		layout.NewSpacer(),
		container.NewVBox(
			widget.NewLabel("Fast Queue"),
			fastLabel,
			fastBar,
		),
		layout.NewSpacer(),
		container.NewVBox(
			widget.NewLabel("Dynamic Queue"),
			dynamicLabel,
			dynamicBar,
		),
		layout.NewSpacer(),
		container.NewVBox(
			widget.NewLabel("Slow Queue"),
			slowLabel,
			slowBar,
		),
		layout.NewSpacer(),
	)

	peersTitle := widget.NewLabel("Discovered Peers")
	peersTitle.TextStyle = fyne.TextStyle{Bold: true}

	peersScroll := container.NewVScroll(peersButtonsContainer)
	peersScroll.SetMinSize(fyne.NewSize(0, 150))

	peersCard := container.NewBorder(
		container.NewVBox(peersTitle, widget.NewSeparator()),
		nil, nil, nil,
		peersScroll,
	)

	main := container.NewVBox(
		header,
		widget.NewSeparator(),
		widget.NewLabel(""),
		queuesContainer,
		widget.NewSeparator(),
		peersCard,
	)

	return container.NewPadded(main)
}

// messageHandler listens for queue state updates from the backend channel.
// Deserializes JSON messages and updates the UI on the main thread using fyne.Do.
// Includes panic recovery to prevent crashes from malformed messages.
func messageHandler() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from message handler panic: %v", r)
		}
	}()

	for msg := range messageChannel {
		var state QueueState
		err := json.Unmarshal([]byte(msg), &state)
		if err != nil {
			utils.LogError("Failed to parse queue state: " + err.Error())
			continue
		}

		fyne.Do(func() {
			updateUI(state)
		})
	}
}

// updateUI synchronizes the UI with the provided queue state from the backend.
// Updates queue lengths, progress bars, connection status, and manages the dynamic
// peer button list. Also refreshes the discovery window if open.
func updateUI(state QueueState) {
	windowMutex.Lock()
	defer windowMutex.Unlock()

	fastSend = state.FastQueue
	dynamicSend = state.DynamicQueue
	slowSend = state.SlowQueue
	connected = state.Connected
	DiscoveredPeers = state.DiscoveredPeers

	fastLabel.SetText(fmt.Sprintf("Length: %d", fastSend))
	dynamicLabel.SetText(fmt.Sprintf("Length: %d", dynamicSend))
	slowLabel.SetText(fmt.Sprintf("Length: %d", slowSend))

	maxValue := float64(100)
	fastBar.SetValue(float64(fastSend) / maxValue)
	dynamicBar.SetValue(float64(dynamicSend) / maxValue)
	slowBar.SetValue(float64(slowSend) / maxValue)

	connected = state.Connected
	updateLEDColor(connected)

	if connected {
		disconnectButton.Show()
	} else {
		disconnectButton.Hide()
	}

	currentPeers := map[string]struct{}{}
	for _, peer := range DiscoveredPeers {
		currentPeers[peer] = struct{}{}
		if _, exists := peersButtonsMap[peer]; !exists {
			peerCopy := peer
			btn := widget.NewButtonWithIcon("  "+peerCopy, theme.ComputerIcon(), func() {
				sendConnectSignalToBackend(peerCopy)
			})
			btn.Importance = widget.HighImportance
			peersButtonsMap[peer] = btn
			peersButtonsContainer.Add(btn)
		}
	}

	for peer, btn := range peersButtonsMap {
		if _, stillExists := currentPeers[peer]; !stillExists {
			peersButtonsContainer.Remove(btn)
			delete(peersButtonsMap, peer)
		}
	}

	peersButtonsContainer.Refresh()
}

// toggleDarkMode switches between dark and light themes.
// Updates the application theme and the dark mode button icon.
func toggleDarkMode() {
	darkMode = !darkMode

	if darkMode {
		window.Settings().SetTheme(theme.DarkTheme())
		darkModeButton.SetText("🌙")
	} else {
		window.Settings().SetTheme(theme.LightTheme())
		darkModeButton.SetText("☀️")
	}
}

// sendConnectSignalToBackend sends a peer connection request to the backend.
// The address parameter should be in "IP:PORT" format.
func sendConnectSignalToBackend(address string) {
	signalChan <- address
}

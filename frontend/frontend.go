package frontend

import (
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/layout"
	"image/color"
	"log"
	"os"
	"sync"
	_ "time"

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
	windowMutex                     sync.RWMutex
	messageChannel                  chan string
	shutdownChan                    chan bool

	fastLabel, dynamicLabel, slowLabel *widget.Label
	fastBar, dynamicBar, slowBar       *widget.ProgressBar
	connectionLED                      *fyne.Container
	ledCircle                          *canvas.Circle
	darkModeButton                     *widget.Button
	window                             fyne.App
	mainWindow                         fyne.Window
)

// QueueState represents the state of all queues received from backend
type QueueState struct {
	FastQueue    uint `json:"fast"`
	DynamicQueue uint `json:"dynamic"`
	SlowQueue    uint `json:"slow"`
	Connected    bool `json:"connected"`
}

func SetupGUI(msgChan chan string) {
	messageChannel = msgChan
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
	mainWindow.SetFixedSize(true)
	mainWindow.Resize(fyne.NewSize(600, 300))

	initializeUIElements()
	content := createLayout()
	mainWindow.SetContent(content)

	go messageHandler()
	mainWindow.ShowAndRun()
}

func initializeUIElements() {
	fastLabel = widget.NewLabel("Length: 0")
	dynamicLabel = widget.NewLabel("Length: 0")
	slowLabel = widget.NewLabel("Length: 0")

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
}

func updateLEDColor(connected bool) {
	if connected {
		ledCircle.FillColor = color.RGBA{G: 255, A: 255}
	} else {
		ledCircle.FillColor = color.RGBA{R: 255, A: 255}
	}
	ledCircle.Refresh()
}

func createLayout() *fyne.Container {
	title := widget.NewLabel("Buffer Queue Visualisation")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Resize(fyne.NewSize(200, 50))

	sizedLEDContainer := container.New(layout.NewGridWrapLayout(fyne.NewSize(30, 30)), connectionLED)

	headerRight := container.NewHBox(
		widget.NewLabel("Connection Status:"),
		sizedLEDContainer,
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

	main := container.NewVBox(
		header,
		widget.NewSeparator(),
		widget.NewLabel(""),
		queuesContainer,
	)

	return container.NewPadded(main)
}

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

func updateUI(state QueueState) {
	windowMutex.Lock()
	defer windowMutex.Unlock()

	fastSend = state.FastQueue
	dynamicSend = state.DynamicQueue
	slowSend = state.SlowQueue
	connected = state.Connected

	fastLabel.SetText(fmt.Sprintf("Length: %d", fastSend))
	dynamicLabel.SetText(fmt.Sprintf("Length: %d", dynamicSend))
	slowLabel.SetText(fmt.Sprintf("Length: %d", slowSend))

	maxValue := float64(100)
	fastBar.SetValue(float64(fastSend) / maxValue)
	dynamicBar.SetValue(float64(dynamicSend) / maxValue)
	slowBar.SetValue(float64(slowSend) / maxValue)

	connected = state.Connected
	updateLEDColor(connected)
}

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

package frontend

import (
	"encoding/json"
	"fmt"
	"gioui.org/f32"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"runtime"
	"sync"
	"time"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"network-interaction/utils"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var (
	fastSend, dynamicSend, slowSend uint
	darkMode                        = true
	darkModeButton                  widget.Clickable
	messageChannel                  chan string
	connected                       bool
	windowMutex                     sync.RWMutex
	shutdownChan                    chan bool
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

	if runtime.GOOS == "darwin" {
		os.Setenv("GOOS", "darwin")
		os.Setenv("GODEBUG", "gioui.debug=1")
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in GUI: %v", r)
				select {
				case shutdownChan <- true:
				default:
				}
			}
		}()

		mainWindow := new(app.Window)
		mainWindow.Option(app.Title("Network Interaction"))
		mainWindow.Option(app.Size(unit.Dp(600), unit.Dp(300)))
		mainWindow.Option(app.MinSize(unit.Dp(600), unit.Dp(300)))
		mainWindow.Option(app.MaxSize(unit.Dp(800), unit.Dp(350)))
		mainWindow.Option(app.Decorated(true))

		err := run(mainWindow)
		if err != nil {
			log.Printf("GUI error: %v", err)
		}

		select {
		case shutdownChan <- true:
		default:
		}

		os.Exit(0)
	}()
	app.Main()
}

func run(window *app.Window) error {
	theme := getTheme()
	var ops op.Ops

	updateChan := make(chan QueueState, 10)

	go func() {
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

			select {
			case updateChan <- state:
			default:
				log.Println("Update channel full, skipping update")
			}
		}
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from UI update panic: %v", r)
			}
		}()

		for state := range updateChan {
			windowMutex.Lock()
			fastSend = state.FastQueue
			dynamicSend = state.DynamicQueue
			slowSend = state.SlowQueue
			connected = state.Connected
			windowMutex.Unlock()

			if window != nil {
				window.Invalidate()
			}
		}
	}()

	eventTimeout := time.NewTimer(time.Second * 30)
	defer eventTimeout.Stop()

	for {
		select {
		case <-eventTimeout.C:
			eventTimeout.Reset(time.Second * 30)

		default:
			switch event := window.Event().(type) {
			case app.DestroyEvent:
				log.Println("Destroy event received")
				close(updateChan)
				return event.Err

			case app.FrameEvent:
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("Recovered from frame event panic: %v", r)
						}
					}()

					context := app.NewContext(&ops, event)

					windowMutex.RLock()
					if darkModeButton.Clicked(context) {
						darkMode = !darkMode
						theme = getTheme()
					}
					windowMutex.RUnlock()

					layoutApp(context, theme)
					event.Frame(context.Ops)
				}()
			}
		}
	}
}

func getTheme() *material.Theme {
	theme := material.NewTheme()
	if darkMode {
		theme.Palette.Bg = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
		theme.Palette.Fg = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		theme.Palette.ContrastBg = color.NRGBA{R: 40, G: 40, B: 40, A: 255}
		theme.Palette.ContrastFg = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	} else {
		theme.Palette.Bg = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		theme.Palette.Fg = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
		theme.Palette.ContrastBg = color.NRGBA{R: 240, G: 240, B: 240, A: 255}
		theme.Palette.ContrastFg = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	}
	return theme
}

func layoutApp(context layout.Context, theme *material.Theme) layout.Dimensions {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from layout panic: %v", r)
		}
	}()

	paint.Fill(context.Ops, theme.Palette.Bg)

	return layout.UniformInset(unit.Dp(20)).Layout(context, func(context layout.Context) layout.Dimensions {
		return layout.Stack{Alignment: layout.N}.Layout(context,
			layout.Expanded(func(context layout.Context) layout.Dimensions {
				paint.Fill(context.Ops, theme.Palette.Bg)
				return layout.Dimensions{Size: context.Constraints.Min}
			}),
			layout.Stacked(func(context layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis:      layout.Vertical,
					Alignment: layout.Start,
					Spacing:   layout.SpaceStart,
				}.Layout(context,
					layout.Rigid(func(context layout.Context) layout.Dimensions {
						return layout.Flex{
							Axis:      layout.Horizontal,
							Alignment: layout.Start,
						}.Layout(context,
							layout.Rigid(func(context layout.Context) layout.Dimensions {
								title := material.H4(theme, "Network Interaction")
								return title.Layout(context)
							}),
							layout.Flexed(1, func(context layout.Context) layout.Dimensions {
								return layout.Dimensions{Size: context.Constraints.Min}
							}),
							layout.Rigid(func(context layout.Context) layout.Dimensions {
								windowMutex.RLock()
								conn := connected
								windowMutex.RUnlock()
								return drawConnectionLED(context, conn)
							}),
							layout.Rigid(func(context layout.Context) layout.Dimensions {
								var iconData []byte
								if darkMode {
									iconData = icons.ImageBrightness2
								} else {
									iconData = icons.ImageBrightness7
								}
								icon, _ := widget.NewIcon(iconData)
								btn := material.IconButton(theme, &darkModeButton, icon, "Toggle Dark Mode")
								btn.Size = unit.Dp(20)
								btn.Background = theme.Palette.ContrastBg
								btn.Inset = layout.UniformInset(unit.Dp(12))
								return btn.Layout(context)
							}),
						)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(context layout.Context) layout.Dimensions {
						windowMutex.RLock()
						fast, dynamic, slow := fastSend, dynamicSend, slowSend
						windowMutex.RUnlock()

						return layout.Flex{
							Axis:      layout.Horizontal,
							Alignment: layout.Start,
						}.Layout(context,
							drawVerticalQueueBar(theme, "Fast Send", fast),
							layout.Rigid(layout.Spacer{Width: unit.Dp(20)}.Layout),
							drawVerticalQueueBar(theme, "Dynamic Send", dynamic),
							layout.Rigid(layout.Spacer{Width: unit.Dp(20)}.Layout),
							drawVerticalQueueBar(theme, "Slow Send", slow),
						)
					}),
				)
			}),
		)
	})
}

func drawVerticalQueueBar(th *material.Theme, label string, value uint) layout.FlexChild {
	return layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: layout.Middle,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Body1(th, label)
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Body2(th, fmt.Sprintf("Length: %d", value))
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return drawVerticalBar(gtx, value)
			}),
		)
	})
}

func drawVerticalBar(gtx layout.Context, value uint) layout.Dimensions {
	barWidth := unit.Dp(40)
	barHeight := unit.Dp(150)
	maxValue := uint(100)

	actualWidth := gtx.Dp(barWidth)
	actualHeight := gtx.Dp(barHeight)

	fillHeight := float32(actualHeight) * float32(value) / float32(maxValue)
	if fillHeight > float32(actualHeight) {
		fillHeight = float32(actualHeight)
	}

	bgColor := color.NRGBA{R: 100, G: 100, B: 100, A: 255}
	if !darkMode {
		bgColor = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
	}

	roundRadius := gtx.Dp(unit.Dp(8))
	bgRect := clip.RRect{
		Rect: image.Rectangle{
			Max: image.Pt(actualWidth, actualHeight),
		},
		SE: roundRadius, NE: roundRadius, NW: roundRadius, SW: roundRadius,
	}.Op(gtx.Ops)
	paint.FillShape(gtx.Ops, bgColor, bgRect)

	if fillHeight > 0 {
		var fillColor color.NRGBA
		if value <= maxValue/3 {
			fillColor = color.NRGBA{R: 34, G: 197, B: 94, A: 255}
		} else if value <= maxValue*2/3 {
			fillColor = color.NRGBA{R: 251, G: 191, B: 36, A: 255}
		} else {
			fillColor = color.NRGBA{R: 239, G: 68, B: 68, A: 255}
		}

		fillRect := clip.RRect{
			Rect: image.Rectangle{
				Min: image.Pt(0, actualHeight-int(fillHeight)),
				Max: image.Pt(actualWidth, actualHeight),
			},
			SE: roundRadius, NE: roundRadius, NW: roundRadius, SW: roundRadius,
		}.Op(gtx.Ops)
		paint.FillShape(gtx.Ops, fillColor, fillRect)
	}

	return layout.Dimensions{Size: image.Pt(actualWidth, actualHeight)}
}

func drawConnectionLED(gtx layout.Context, isConnected bool) layout.Dimensions {
	size := gtx.Dp(unit.Dp(24))

	ledColor := color.NRGBA{R: 239, G: 68, B: 68, A: 255}
	if isConnected {
		ledColor = color.NRGBA{R: 34, G: 197, B: 94, A: 255}
	}

	var path clip.Path
	path.Begin(gtx.Ops)
	radius := float32(size) / 2
	center := f32.Point{X: radius, Y: radius}
	path.Arc(center, f32.Point{X: radius, Y: 0}, 2*math.Pi)
	path.Close()
	circle := clip.Outline{Path: path.End()}.Op()
	paint.FillShape(gtx.Ops, ledColor, circle)

	return layout.Dimensions{
		Size: image.Pt(size, size),
	}
}

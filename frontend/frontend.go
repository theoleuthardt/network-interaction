package frontend

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"

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
	extraWindowButton               widget.Clickable
	mainWindow                      *app.Window
	extraWindow                     *app.Window
	messageChannel                  chan string
)

// QueueState represents the state of all queues received from backend
type QueueState struct {
	FastQueue    uint `json:"fast"`
	DynamicQueue uint `json:"dynamic"`
	SlowQueue    uint `json:"slow"`
}

func SetupGUI(msgChan chan string) {
	messageChannel = msgChan

	go func() {
		mainWindow := new(app.Window)
		mainWindow.Option(app.Title("Network Interaction"))
		mainWindow.Option(app.Size(unit.Dp(600), unit.Dp(300)))
		mainWindow.Option(app.MinSize(unit.Dp(600), unit.Dp(300)))
		mainWindow.Option(app.MaxSize(unit.Dp(800), unit.Dp(350)))
		err := run(mainWindow)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(window *app.Window) error {
	theme := getTheme()
	var ops op.Ops

	go func() {
		for msg := range messageChannel {
			// Parse the message as JSON
			var state QueueState
			err := json.Unmarshal([]byte(msg), &state)
			if err != nil {
				utils.LogError("Failed to parse queue state: " + err.Error())
				continue
			}

			// Update the UI values
			fastSend = state.FastQueue
			dynamicSend = state.DynamicQueue
			slowSend = state.SlowQueue

			window.Invalidate()
		}
	}()

	for {
		switch event := window.Event().(type) {
		case app.DestroyEvent:
			return event.Err
		case app.FrameEvent:
			context := app.NewContext(&ops, event)

			if darkModeButton.Clicked(context) {
				darkMode = !darkMode
				theme = getTheme()
				if extraWindow != nil {
					extraWindow.Invalidate()
				}

			}

			if extraWindowButton.Clicked(context) {
				openExtraWindow()
			}

			layoutApp(context, theme)
			event.Frame(context.Ops)
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

func openExtraWindow() {
	if extraWindow != nil {
		return
	}

	go func() {
		extraWindow = new(app.Window)
		extraWindow.Option(app.Title("Network Discovery"))
		extraWindow.Option(app.Size(unit.Dp(400), unit.Dp(300)))
		extraWindow.Option(app.MinSize(unit.Dp(400), unit.Dp(300)))
		extraWindow.Option(app.MaxSize(unit.Dp(800), unit.Dp(600)))
		err := runExtraWindow(extraWindow)
		if err != nil {
			log.Printf("Extra window error: %v", err)
		}
		extraWindow = nil
	}()
}

func runExtraWindow(window *app.Window) error {
	var ops op.Ops
	var localDarkModeButton widget.Clickable

	for {
		switch event := window.Event().(type) {
		case app.DestroyEvent:
			return event.Err
		case app.FrameEvent:
			context := app.NewContext(&ops, event)
			theme := getTheme()

			if localDarkModeButton.Clicked(context) {
				darkMode = !darkMode
				theme = getTheme()
				if mainWindow != nil {
					mainWindow.Invalidate()
				}

			}

			layoutExtraWindow(context, theme)
			event.Frame(context.Ops)

		}
	}
}

func layoutExtraWindow(context layout.Context, theme *material.Theme) layout.Dimensions {
	paint.Fill(context.Ops, theme.Palette.Bg)

	return layout.UniformInset(unit.Dp(20)).Layout(context, func(context layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis: layout.Vertical,
		}.Layout(context,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis: layout.Horizontal,
				}.Layout(gtx,
					layout.Rigid(material.H5(theme, "Network Discovery Manager").Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Dimensions{Size: gtx.Constraints.Min}
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			layout.Rigid(material.Body1(theme, "You can connect and send tcp packages here!").Layout),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(material.Body2(theme, "TCP is so cool!").Layout),
		)
	})
}

func layoutApp(context layout.Context, theme *material.Theme) layout.Dimensions {
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
							layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
							layout.Rigid(func(context layout.Context) layout.Dimensions {
								icon, _ := widget.NewIcon(icons.HardwareDeviceHub)
								btn := material.IconButton(theme, &extraWindowButton, icon, "Open Extra Window")
								btn.Size = unit.Dp(20)
								btn.Background = theme.Palette.ContrastBg
								btn.Inset = layout.UniformInset(unit.Dp(12))
								return btn.Layout(context)
							}),
						)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(context layout.Context) layout.Dimensions {
						return layout.Flex{
							Axis:      layout.Horizontal,
							Alignment: layout.Start,
						}.Layout(context,
							drawVerticalQueueBar(theme, "Fast Send", fastSend),
							layout.Rigid(layout.Spacer{Width: unit.Dp(20)}.Layout),
							drawVerticalQueueBar(theme, "Dynamic Send", dynamicSend),
							layout.Rigid(layout.Spacer{Width: unit.Dp(20)}.Layout),
							drawVerticalQueueBar(theme, "Slow Send", slowSend),
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

package frontend

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"network-interaction/utils"
	"os"
	"strconv"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

var (
	fastSend, dynamicSend, slowSend uint
)

func SetupGUI() {
	go func() {
		window := new(app.Window)
		window.Option(app.Title("network-interaction"))
		err := run(window)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(window *app.Window) error {
	th := material.NewTheme()
	var ops op.Ops

	go func() {
		for {
			fastSend = readUint("fast_send.txt")
			dynamicSend = readUint("dynamic_send.txt")
			slowSend = readUint("slow_send.txt")
			window.Invalidate()
			time.Sleep(500 * time.Millisecond)
		}
	}()

	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			layoutApp(gtx, th)
			e.Frame(gtx.Ops)
		}
	}
}

func layoutApp(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		inset := layout.UniformInset(unit.Dp(16))
		return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis:    layout.Vertical,
				Spacing: layout.SpaceBetween,
			}.Layout(gtx,
				layout.Rigid(material.H6(th, "Network Buffer Queue Lengths").Layout),
				drawQueueBar(gtx, th, "Fast Send", fastSend),
				drawQueueBar(gtx, th, "Dynamic Send", dynamicSend),
				drawQueueBar(gtx, th, "Slow Send", slowSend),
			)
		})
	})
}

func drawQueueBar(gtx layout.Context, th *material.Theme, label string, value uint) layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis: layout.Vertical,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis: layout.Horizontal,
				}.Layout(gtx,
					layout.Rigid(material.Body1(th, label).Layout),
					layout.Flexed(1, layout.Spacer{Width: unit.Dp(1)}.Layout),
					layout.Rigid(material.Body1(th, fmt.Sprintf("Length: %d", value)).Layout),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				barHeight := unit.Dp(24)
				maxValue := uint(20)
				barWidth := float32(gtx.Constraints.Max.X)
				filled := barWidth * float32(value) / float32(maxValue)

				// Background bar
				bgRect := clip.Rect{
					Max: image.Pt(int(barWidth), gtx.Dp(barHeight)),
				}.Op()
				paint.FillShape(gtx.Ops, color.NRGBA{R: 230, G: 230, B: 230, A: 255}, bgRect)

				// Filled portion
				fgRect := clip.Rect{
					Max: image.Pt(int(filled), gtx.Dp(barHeight)),
				}.Op()
				paint.FillShape(gtx.Ops, color.NRGBA{R: 80, G: 120, B: 255, A: 255}, fgRect)

				return layout.Dimensions{Size: image.Pt(int(barWidth), gtx.Dp(barHeight))}
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		)
	})
}

func readUint(filename string) uint {
	data, err := os.ReadFile(filename)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(string(data))
	if err != nil {
		return 0
	}
	utils.LogInfo(fmt.Sprintf("[FRONTEND] %s: %d", filename, n))
	return uint(n)
}

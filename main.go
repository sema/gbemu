package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"time"

	"github.com/alecthomas/kong"
	"github.com/sema/gbemu/pkg/emulator"

	wde "github.com/skelterjohn/go.wde"
	_ "github.com/skelterjohn/go.wde/cocoa"
)

var shadeToColor = []color.RGBA{
	color.RGBA{R: 155, G: 188, B: 15, A: 255}, // "white"
	color.RGBA{R: 139, G: 172, B: 15, A: 255},
	color.RGBA{R: 48, G: 98, B: 48, A: 255},
	color.RGBA{R: 15, G: 56, B: 15, A: 255}, // "black"
}

type runCmd struct {
	BootROM string `help:"Use boot ROM" type:"path"`

	Path string `arg name:"path" help:"Path to ROM" type:"path"`
}

type sprite struct {
}

func (s sprite) ColorModel() color.Model {
	return color.GrayModel
}

func (s sprite) Bounds() image.Rectangle {
	return image.Rectangle{
		Min: image.Point{
			X: 0,
			Y: 0,
		},
		Max: image.Point{
			X: 1,
			Y: 1,
		},
	}
}

func (s sprite) At(xx, y int) color.Color {
	return color.Black
}

func (r *runCmd) Run() error {
	ctx := context.Background()
	e := emulator.New()

	go func() {
		if err := e.Run(ctx, r.Path, r.BootROM); err != nil {
			log.Panicln(err)
		}
	}()

	go func() {
		frames := 0
		ticker := time.Tick(time.Second)

		w, err := wde.NewWindow(512, 512)
		if err != nil {
			log.Panicln(err)
		}

		// TODO lock screen to 512x512 as large screens are slow to render.
		// Need to improve render performance.
		w.LockSize(true)
		w.Show()

		events := w.EventChan()

		for {
			select {

			case <-ticker:
				w.SetTitle(fmt.Sprintf("gbemu | FPS: %d", frames))
				frames = 0

			case event := <-events:
				switch v := event.(type) {
				case wde.CloseEvent:
					log.Panicln("stop") // TODO implement proper stop
				case wde.KeyTypedEvent:
					switch v.Key {
					case wde.KeyEscape:
						log.Panicln("stop") // TODO implement proper stop
					}
				}

			case frame := <-e.FrameChan:
				// scale original buffer to fill window
				scale := int(math.Min(float64(w.Screen().Bounds().Max.X/160), float64(w.Screen().Bounds().Max.Y/144)))

				screenWidth := 160 * scale
				screenHeight := 144 * scale

				centerX := w.Screen().Bounds().Max.X / 2
				centerY := w.Screen().Bounds().Max.Y / 2

				minX := centerX - screenWidth/2
				minY := centerY - screenHeight/2
				maxX := centerX + screenWidth/2
				maxY := centerY + screenHeight/2
				screenSize := image.Rect(minX, minY, maxX, maxY)

				buffer := image.NewRGBA(screenSize)

				for y, row := range frame {
					for x, shade := range row {
						for ys := minY + y*scale; ys < minY+y*scale+scale; ys++ {
							for xs := minX + x*scale; xs < minX+x*scale+scale; xs++ {
								buffer.Set(xs, ys, shadeToColor[shade])
							}
						}
					}
				}

				w.Screen().CopyRGBA(buffer, screenSize)
				w.FlushImage(screenSize)

				frames++
			}
		}

	}()

	wde.Run()

	return nil
}

var root struct {
	Run runCmd `cmd help:"run ROM"`
}

func main() {
	cli := kong.Parse(&root)
	err := cli.Run()
	cli.FatalIfErrorf(err)
}

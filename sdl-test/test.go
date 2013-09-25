package main

import (
	"fmt"
	"github.com/scottferg/Go-SDL2/mixer"
	"github.com/scottferg/Go-SDL2/sdl"
	"log"
	"math"
)

type Point struct {
	x int
	y int
}

func (a Point) add(b Point) Point { return Point{a.x + b.x, a.y + b.y} }

func (a Point) sub(b Point) Point { return Point{a.x - b.x, a.y - b.y} }

func (a Point) length() float64 { return math.Sqrt(float64(a.x*a.x + a.y*a.y)) }

func (a Point) mul(b float64) Point {
	return Point{int(float64(a.x) * b), int(float64(a.y) * b)}
}

func worm(in <-chan Point, out chan<- Point, draw chan<- Point) {

	t := Point{0, 0}

	for {
		p := (<-in).sub(t)

		if p.length() > 48 {
			t = t.add(p.mul(0.1))
		}

		draw <- t
		out <- t
	}
}

func main() {
	if sdl.Init(sdl.INIT_EVERYTHING) != 0 {
		log.Fatal(sdl.GetError())
	}

	if mixer.OpenAudio(mixer.DEFAULT_FREQUENCY, mixer.DEFAULT_FORMAT,
		mixer.DEFAULT_CHANNELS, 4096) != 0 {
		log.Fatal(sdl.GetError())
	}

	var window = sdl.CreateWindow("SDL2 Sample", sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, 640, 480, sdl.WINDOW_SHOWN)

	if window == nil {
		log.Println("nil window")
		log.Fatal(sdl.GetError())
	}

	rend := sdl.CreateRenderer(window, -1,
		sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)

	if rend == nil {
		log.Println("nil rend")
		log.Fatal(sdl.GetError())
	}

	window.SetTitle("First SDL2 Window")

	image := sdl.Load("./test.png")

	if image == nil {
		log.Println("nil image")
		log.Fatal(sdl.GetError())
	}

	window.SetIcon(image)

	tex := sdl.CreateTextureFromSurface(rend, image)

	running := true

	worm_in := make(chan Point)
	draw := make(chan Point, 64)

	var out chan Point
	var in chan Point

	out = worm_in

	in = out
	out = make(chan Point)
	go worm(in, out, draw)

	// ticker := time.NewTicker(time.Second / 50) // 50 Hz

	rend.SetDrawColor(sdl.Color{0x30, 0x20, 0x19, 0xFF, 0x00})
	rend.FillRect(nil)
	rend.Copy(tex, nil, nil)
	rend.Present()

	window.ShowSimpleMessageBox(sdl.MESSAGEBOX_INFORMATION, "Test Message", "SDL2 supports message boxes!")

	for running {
		select {
		/*
					case <-ticker.C:
						rend.SetDrawColor(0x30, 0x20, 0x19, 0xFF)
						rend.FillRect(nil)
			            if sdl.GetError() != "" {
			                log.Fatalf(sdl.GetError())
			            }

					loop:
						for {
							select {
							case p := <-draw:
								rend.Clear()
								rend.Copy(tex, &sdl.Rect{int16(p.x), int16(p.y), 0, 0}, nil)

							case <-out:
							default:
								break loop
							}
						}

						var p Point
						sdl.GetMouseState(&p.x, &p.y)
						worm_in <- p

						rend.Present()
		*/
		case _event := <-sdl.Events:
			switch e := _event.(type) {
			case sdl.QuitEvent:
				running = false
			case sdl.KeyboardEvent:
				println("")
				println(e.Keysym.Sym, ": ", sdl.GetKeyName(sdl.Key(e.Keysym.Sym)))

				if e.Keysym.Sym == sdl.K_ESCAPE {
					running = false
				}

				fmt.Printf("%04x ", e.Type)

				for i := 0; i < len(e.Pad0); i++ {
					fmt.Printf("%02x ", e.Pad0[i])
				}
				println()

				fmt.Printf("Type: %02x State: %02x Pad: %02x\n", e.Type, e.State, e.Pad0[0])
				fmt.Printf("Scancode: %02x Sym: %08x Mod: %04x Unicode: %04x\n", e.Keysym.Scancode, e.Keysym.Sym, e.Keysym.Mod, e.Keysym.Unicode)

			case sdl.MouseButtonEvent:
				if e.Type == sdl.MOUSEBUTTONDOWN {
					println("Click:", e.X, e.Y)
					in = out
					out = make(chan Point)
					go worm(in, out, draw)
				}
			}
		}
	}

	image.Free()
	tex.Destroy()
	rend.Destroy()
	window.Destroy()
	sdl.Quit()
}

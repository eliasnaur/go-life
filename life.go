package main

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"os"
	"time"

	"gioui.org/ui"
	"gioui.org/ui/app"
	"gioui.org/ui/draw"
	"gioui.org/ui/f32"
	"gioui.org/ui/key"
	"gioui.org/ui/layout"
	"gioui.org/ui/measure"
	"gioui.org/ui/pointer"
	"gioui.org/ui/text"
	"github.com/theclapp/go-life/gesture"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
)

type (
	Pos struct {
		x, y int
	}

	Universe struct {
		cells      map[Pos]bool
		gen        int
		minX, minY int
		maxX, maxY int
	}

	Window struct {
		w       *app.Window
		u       *Universe
		scale   int
		regular *sfnt.Font
		faces   measure.Faces

		scrollXY         gesture.ScrollXY
		scrollX, scrollY int
	}
)

func main() {
	go func() {
		w := &Window{scale: 10}
		w.u = NewUniverse()
		// w.u.RPentomino()
		w.u.Random(100, 100, 333)

		w.w = app.NewWindow(&app.WindowOptions{
			Width:  ui.Dp(800),
			Height: ui.Dp(600),
			Title:  "Gio Life",
		})

		w.regular, _ = sfnt.Parse(goregular.TTF)

		ops := &ui.Ops{}

		interval := 100 * time.Millisecond
		genTimer := time.NewTicker(interval)
		for {
			select {
			case <-genTimer.C:
				w.u = w.u.NextGen()
				w.w.Invalidate()
			case e := <-w.w.Events():
				switch e := e.(type) {
				case key.EditEvent:
					switch e.Text {
					case "q":
						os.Exit(0)
					case "-":
						if w.scale > 1 {
							w.scale--
						}
					case "+", "=":
						w.scale++
					// faster
					case ">", ".":
						genTimer.Stop()
						interval /= 2
						genTimer = time.NewTicker(interval)
					// slower
					case "<", ",":
						genTimer.Stop()
						interval *= 2
						genTimer = time.NewTicker(interval)
					default:
						// fmt.Printf("key name: %s\n", e.Text)
					}
				case app.DrawEvent:
					ops.Reset()
					w.Layout(e, ops)
					w.w.Draw(ops)
				default:
					// fmt.Printf("event %T %+v\n", e, e)
				}
			}
		}
	}()
	app.Main()
}

func NewUniverse() *Universe {
	return &Universe{
		cells: make(map[Pos]bool),
	}
}

func (u *Universe) RPentomino() {
	//  xx
	// xx
	//  x
	for _, rec := range []Pos{
		{1, 2}, {2, 2},
		{0, 1}, {1, 1},
		{1, 0},
	} {
		u.Set(rec)
	}
}

func (u *Universe) Set(p Pos) {
	u.cells[p] = true
}

func (u *Universe) IsSet(p Pos) int {
	if u.cells[p] {
		return 1
	}
	return 0
}

func (u *Universe) NextGen() *Universe {
	next := *u
	next.cells = make(map[Pos]bool)
	next.gen = u.gen + 1
	checked := make(map[Pos]bool)
	for pos := range u.cells {
		// Track min/max dimensions
		next.minX = min(pos.x, next.minX)
		next.minY = min(pos.y, next.minY)
		next.maxX = max(pos.x, next.maxX)
		next.maxY = max(pos.y, next.maxY)

		// Will this cell be alive in the next generation?
		if n := u.NumNeighbors(pos); n == 2 || n == 3 {
			next.Set(pos)
		}

		// Will any of this cell's neighbors be alive in the next generation?
		xm1, xp1 := pos.x-1, pos.x+1
		ym1, yp1 := pos.y-1, pos.y+1
		for _, neighbor := range []Pos{
			{xm1, ym1}, {pos.x, ym1}, {xp1, ym1},
			{xm1, pos.y}, {xp1, pos.y},
			{xm1, yp1}, {pos.x, yp1}, {xp1, yp1}} {
			if checked[neighbor] {
				continue
			}
			if u.NumNeighbors(neighbor) == 3 {
				next.Set(neighbor)
			}
			checked[neighbor] = true
		}
	}
	return &next
}

// Aborts at 4
func (u *Universe) NumNeighbors(pos Pos) int {
	xm1, xp1 := pos.x-1, pos.x+1
	ym1, yp1 := pos.y-1, pos.y+1
	n := u.IsSet(Pos{xm1, ym1}) + u.IsSet(Pos{pos.x, ym1}) + u.IsSet(Pos{xp1, ym1}) +
		u.IsSet(Pos{xm1, pos.y})
	if n == 4 {
		return n
	}
	for _, neighbor := range []Pos{
		{xp1, pos.y},
		{xm1, yp1}, {pos.x, yp1}, {xp1, yp1}} {
		if u.cells[neighbor] {
			n++
			if n == 4 {
				return n
			}
		}
	}
	return n
}

func min(n1, n2 int) int {
	if n1 < n2 {
		return n1
	}
	return n2
}

func max(n1, n2 int) int {
	if n1 > n2 {
		return n1
	}
	return n2
}

func (w *Window) Layout(e app.DrawEvent, ops *ui.Ops) {
	cfg := &e.Config
	cs := layout.RigidConstraints(e.Size)
	w.faces.Reset(cfg)

	scrollX, scrollY := w.scrollXY.ScrollXY(cfg, w.w.Queue())
	if scrollX != 0 || scrollY != 0 {
		// fmt.Printf("window layout: scrollX: %d, scrollY: %d\n", scrollX, scrollY)
		w.scrollX -= scrollX
		w.scrollY -= scrollY
	}
	r := image.Rectangle{Max: e.Size}
	pointer.RectAreaOp{Rect: r}.Add(ops)
	w.scrollXY.Add(ops)

	draw.ColorOp{
		Color: color.RGBA{A: 0x80, R: 0xff, B: 0xff, G: 0xff},
	}.Add(ops)
	lbl := text.Label{
		Face: w.faces.For(w.regular, ui.Sp(13)),
		Text: fmt.Sprintf("Gen: %d", w.u.gen),
	}
	lbl.Layout(ops, cs)

	// // If the farthest cell is beyond the window edge, reduce the scale, if
	// // possible.
	// if w.scale > 2 &&
	// 	(xOffset+(w.u.maxX-w.u.minX)*w.scale > e.Size.X ||
	// 		yOffset+(w.u.maxY-w.u.minY)*w.scale > e.Size.Y) {
	// 	w.scale--
	// }

	xOffset, yOffset := 0, 14
	// ui.TransformOp{}.Offset(f32.Point{
	// 	X: float32(xOffset + w.scale*-w.u.minX + 10),
	// 	Y: float32(yOffset + w.scale*-w.u.minY + 10),
	// }).Add(ops)
	ui.TransformOp{}.Offset(f32.Point{
		X: float32(xOffset + w.scrollX),
		Y: float32(yOffset + w.scrollY),
	}).Add(ops)
	draw.ColorOp{Color: color.RGBA{A: 0xff, G: 0xff}}.Add(ops)
	for pos := range w.u.cells {
		draw.DrawOp{
			Rect: f32.Rectangle{
				Min: f32.Point{float32(w.scale * pos.x), float32(w.scale * pos.y)},
				Max: f32.Point{float32(w.scale * (pos.x + 1)), float32(w.scale * (pos.y + 1))},
			},
		}.Add(ops)
	}

}

func (u *Universe) Random(x, y, density int) {
	for px := 0; px < x; px++ {
		for py := 0; py < y; py++ {
			if rand.Intn(1000) < density {
				u.Set(Pos{px, py})
			}
		}
	}
}

func (w *Window) label(ops *ui.Ops, cs layout.Constraints, txt string) (pressed bool) {
	draw.ColorOp{
		Color: color.RGBA{A: 0x80, R: 0xff, B: 0xff, G: 0xff},
	}.Add(ops)
	lbl := text.Label{
		Face: w.faces.For(w.regular, ui.Sp(13)),
		Text: txt,
	}
	lbl.Layout(ops, cs)
	return false
}

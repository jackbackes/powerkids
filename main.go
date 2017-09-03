package main

import (
	"fmt"
	"time"
	"math"
	"math/rand"
	"github.com/faiface/pixel"
        "github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
	"image"
	"image/color"
	"os"
	_ "image/png"
	"strconv"
	"encoding/csv"
	"io"
	"github.com/pkg/errors"
)

type animState int

const (
	south animState = iota
	east
	west
	north
)

func loadPicture(path string) (pixel.Picture, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return pixel.PictureDataFromImage(img), nil
}

func loadAnimationSheet(sheetPath, descPath string, frameWidth float64) (sheet pixel.Picture, anims map[string][]pixel.Rect, err error) {
	// total hack, nicely format the error at the end, so I don't have to type it every time
	defer func() {
		if err != nil {
			err = errors.Wrap(err, "error loading animation sheet")
		}
	}()

	// open and load the spritesheet
	sheetFile, err := os.Open(sheetPath)
	if err != nil {
		return nil, nil, err
	}
	defer sheetFile.Close()
	sheetImg, _, err := image.Decode(sheetFile)
	if err != nil {
		return nil, nil, err
	}
	sheet = pixel.PictureDataFromImage(sheetImg)

	// create a slice of frames inside the spritesheet
	var frames [][]pixel.Rect
	for y := sheet.Bounds().Max.Y-frameWidth; y >= frameWidth; y -= frameWidth {
		var row []pixel.Rect
		for x := 0.0; x+frameWidth <= sheet.Bounds().Max.X; x += frameWidth {
			row = append(row, pixel.R(
				x,
				y,
				x+frameWidth,
				y+frameWidth,
			))
		}
		frames = append(frames, row)
	}

	descFile, err := os.Open(descPath)
	if err != nil {
		return nil, nil, err
	}
	defer descFile.Close()

	anims = make(map[string][]pixel.Rect)

	// load the animation information, name and interval inside the spritesheet
	desc := csv.NewReader(descFile)
	for {
		anim, err := desc.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		name := anim[0]
		row, _ := strconv.Atoi(anim[1])
		start, _ := strconv.Atoi(anim[2])
		end, _ := strconv.Atoi(anim[3])
		fmt.Printf("%v %v %v\n", row, start, end)
		anims[name] = frames[row][start : end+1]
	}

	return sheet, anims, nil
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}
type sarahPhys struct {

	rect   pixel.Rect
	vel    pixel.Vec
	ground bool
	dir animState
}

func (gp *sarahPhys) update(dt float64, ctrl pixel.Vec) {
	gp.vel = ctrl
	gp.rect = gp.rect.Moved(ctrl)
	switch {
	case gp.vel.X > 0:
		gp.dir = east
	case gp.vel.X < 0:
		gp.dir = west
	case gp.vel.Y < 0:
		gp.dir = south
	case gp.vel.Y > 0:
		gp.dir = north
	}
}

type sarahAnimation struct {
	sheet pixel.Picture
	anims map[string][]pixel.Rect
	rate  float64

	state   animState
	counter float64
	dir     float64

	frame pixel.Rect

	sprite *pixel.Sprite
}

func (sa *sarahAnimation) update(dt float64, phys *sarahPhys) {
	sa.counter += dt

	// determine the new animation state
	var newState animState
	switch {
	default:
		newState = phys.dir
	}

	// reset the time counter if the state changed
	if sa.state != newState {
		fmt.Printf("oldState: %v, newState: %v\n", sa.state, newState)
		sa.state = newState
		sa.counter = 0
	}

	// determine the correct animation frame
	switch sa.state {
	case south:
		sa.frame = sa.anims["South"][0]
	case east:
		sa.frame = sa.anims["East"][0]
	case north:
		sa.frame = sa.anims["North"][0]
	case west:
		sa.frame = sa.anims["West"][0]
	}
	if sa.counter == 0 {
		fmt.Printf("Frame: %v\n", sa.frame)
	}
}

func (sa *sarahAnimation) draw(t pixel.Target, phys *sarahPhys) {
	if sa.sprite == nil {
		sa.sprite = pixel.NewSprite(nil, pixel.Rect{})
	}
	// draw the correct frame with the correct position and direction
	sa.sprite.Set(sa.sheet, sa.frame)
	sa.sprite.Draw(t, pixel.IM.Moved(phys.rect.Center()))
}

type goal struct {
	pos    pixel.Vec
	radius float64
	step   float64

	counter float64
	cols    [5]pixel.RGBA
}

func (g *goal) update(dt float64) {
	g.counter += dt
	for g.counter > g.step {
		g.counter -= g.step
		for i := len(g.cols) - 2; i >= 0; i-- {
			g.cols[i+1] = g.cols[i]
		}
		g.cols[0] = randomNiceColor()
	}
}

func randomNiceColor() pixel.RGBA {
again:
	r := rand.Float64()
	g := rand.Float64()
	b := rand.Float64()
	len := math.Sqrt(r*r + g*g + b*b)
	if len == 0 {
		goto again
	}
	return pixel.RGB(r/len, g/len, b/len)
}

func run() {
	cfg := pixelgl.WindowConfig{
		Title: "PowerKids!",
		Bounds: pixel.R(0,0,1024,768),
	}
	win, err := pixelgl.NewWindow(cfg)
	handleErr(err)

	sarahSheet, sarahAnims, err := loadAnimationSheet("static/LPC_Sara/SaraFullSheet.png", "static/LPC_Sara/SaraAnimations.csv", 64)
	handleErr(err)

	phys := &sarahPhys{
		rect:      pixel.R(-32, -32, 32, 32),
	}

	sarahAnim := &sarahAnimation{
		sheet: sarahSheet,
		anims: sarahAnims,
		rate:  1.0 / 10,
		dir:   +1,
	}

	canvas := pixelgl.NewCanvas(win.Bounds())


	camPos := pixel.ZV
	last := time.Now()
	for !win.Closed() {
		win.Clear(colornames.Green)
		dt := time.Since(last).Seconds()
		last = time.Now()

		// lerp the camera position towards the gopher
		camPos = pixel.Lerp(camPos, phys.rect.Center(), 1-math.Pow(1.0/128, dt))
		cam := pixel.IM.Moved(camPos.Scaled(-1))
		canvas.SetMatrix(cam)

		// slow motion with tab
		if win.Pressed(pixelgl.KeyTab) {
			dt /= 8
		}

		// restart the level on pressing enter
		if win.JustPressed(pixelgl.KeyEnter) {
			phys.rect = phys.rect.Moved(phys.rect.Center().Scaled(-1))
			phys.vel = pixel.ZV
		}

		// control the hero with keys
		ctrl := pixel.ZV
		ctrl.X = 0
		ctrl.Y = 0
		if win.Pressed(pixelgl.KeyLeft) {
			ctrl.X = -1
		}
		if win.Pressed(pixelgl.KeyRight) {
			ctrl.X = 1
		}
		if win.Pressed(pixelgl.KeyUp) {
			ctrl.Y = 1
		}
		if win.Pressed(pixelgl.KeyDown) {
			ctrl.Y = -1
		}

		canvas.Clear(color.Transparent)

		// draw the map
		castle, err := NewMapFromText(
			"maps/castle-one/castleMap.txt",
			"static/castle2.png",
			"static/castle2.csv",
			32)
		handleErr(err)

		mat := pixel.IM.Scaled(pixel.ZV,
			math.Min(
				win.Bounds().W()/canvas.Bounds().W(),
				win.Bounds().H()/canvas.Bounds().H(),
			),
		).Moved(win.Bounds().Center())
		castle.Draw(win, mat)

		// update the physics and animation
		phys.update(dt, ctrl)
		sarahAnim.update(dt, phys)

		// draw the scene to the canvas using IMDraw
		sarahAnim.draw(win, phys)
		// stretch the canvas to the window
		win.SetMatrix(mat)
		win.Update()
	}
}







// THE STORY GOES HERE
// th pawrkids wr so busy. they were farming.
















func main() {
	pixelgl.Run(run)
}


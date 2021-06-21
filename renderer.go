package main

import (
	"fmt"
	"github.com/tfriedel6/canvas"
	"github.com/tfriedel6/canvas/backend/softwarebackend"
	_ "image/jpeg"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
)

const Width = 1920
const Height = 1080
const EntryWidth = 0.3 * Width
const EntryHeight = 64

const AvatarSize = 32

const MiddleX = Width / 2
const EntryFontSize = 36
const FontFile = "font.ttf"

var Colours = [...]string{"#FFD700", "#dbdbdb", "#cd7f32"}
var backgrounds []*canvas.Image
var currentBg int
var can *canvas.Canvas

// Initialize the canvas and load images from the bg folder
func initialize() {
	backend := softwarebackend.New(1920, 1080)
	can = canvas.New(backend)

	backgroundFiles, _ := ioutil.ReadDir("bg")
	for _, file := range backgroundFiles {
		img, err := can.LoadImage("bg/" + file.Name())
		if err != nil {
			panic(err)
		}

		backgrounds = append(backgrounds, img)
	}
}

// Gets the next background in the file name array or 0 if reached the end
func getCurrentBackground() *canvas.Image {
	if currentBg == len(backgrounds) { // Loop back
		currentBg = 0
	}

	bg := backgrounds[currentBg]
	currentBg++

	return bg
}

// Render the canvas with the values of the entries specified
func render(entries [3]LeaderboardEntry) {
	bg := getCurrentBackground()
	can.DrawImage(bg, 0, 0, Width, Height) // Render current background

	const headerY = 100.0
	renderCenteredModule("#1e3a8a", headerY, 750, 120) // Render top header

	can.SetFillStyle("#FFF") // All info text should be white

	// Header main text
	const headerFontSize = 48.0
	const headerTextY = headerY + headerFontSize + 20.0
	renderCenteredText(headerFontSize, "Veckans topplista", headerTextY)

	// Header reset text
	const resetFontSize = 14.0
	renderCenteredText(resetFontSize, "Återställs varje söndag", headerTextY+resetFontSize+15)

	// Render leaderboard entries
	can.SetFont("font.ttf", EntryFontSize)
	entryY := 300.0
	for i, entry := range entries {
		renderLeaderboardEntry(i, entryY, entry.Name, int(entry.XP))
		entryY += 100
	}

	// IP Text
	renderCenteredModuleWithRadius("#3b82f6", 600, 650, 50, 5)
	can.SetFillStyle("#FFF")
	const ipTextFontSize = 24.0
	renderCenteredText(ipTextFontSize, "Vill du vara med och tävla? Anslut till minarty.fun!", entryY+(50/2)+ipTextFontSize/2)

	// Save canvas data to PNG
	img := can.GetImageData(0, 0, Width, Height)
	f, _ := os.OpenFile(ImageFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	_ = png.Encode(f, img)
}

func renderCenteredText(fontSize float64, text string, y float64) {
	can.SetFont(FontFile, fontSize)
	can.FillText(text, MiddleX-can.MeasureText(text).Width/2, y)
}

// Renders a entry in the leaderboard
func renderLeaderboardEntry(index int, y float64, name string, value int) {
	x := renderCenteredModule(Colours[index], y, EntryWidth, EntryHeight)
	drawImageFromUrl("https://minotar.net/avatar/"+name+"/32.png", x+20, y+(AvatarSize/2))

	text := fmt.Sprintf("%s - %d XP", name, value)

	can.SetFillStyle("#000000")
	can.FillText(text, MiddleX-can.MeasureText(text).Width/2, y+(EntryHeight/2)+EntryFontSize/2)
}

// Fetches an image from URL and draws it at the specified coordinates
func drawImageFromUrl(url string, x float64, y float64) {
	data := getImg(url)
	if data == nil {
		return
	}

	img, _ := can.LoadImage(data)
	can.DrawImage(img, x, y)
}

// Fetches image bytes from an URL
func getImg(url string) []byte {
	resp, _ := http.Get(url)
	if resp == nil || resp.Body == nil {
		return nil
	}

	slice, _ := ioutil.ReadAll(resp.Body)
	return slice
}

// Renders a horizontally centered rect with default 20 as corner radius
func renderCenteredModule(colour string, y float64, width float64, height float64) float64 {
	return renderCenteredModuleWithRadius(colour, y, width, height, 20)
}

// Renders a horizontally centered rect with specific corner radius
func renderCenteredModuleWithRadius(colour string, y float64, width float64, height float64, r float64) float64 {
	can.SetFillStyle(colour)
	x := (1920 / 2) - (width / 2)
	roundRect(x, y, width, height, r)
	return x
}

// https://stackoverflow.com/a/7838871
func roundRect(x float64, y float64, w float64, h float64, r float64) {
	if w < 2*r {
		r = w / 2
	}

	if h < 2*r {
		r = h / 2
	}

	can.BeginPath()
	can.MoveTo(x+r, y)
	can.ArcTo(x+w, y, x+w, y+h, r)
	can.ArcTo(x+w, y+h, x, y+h, r)
	can.ArcTo(x, y+h, x, y, r)
	can.ArcTo(x, y, x+w, y, r)
	can.ClosePath()
	can.Fill()
}

package main

import (
	"os"
	"encoding/csv"
	"strconv"
	"io"
	"bufio"
        "github.com/faiface/pixel"
        "github.com/faiface/pixel/pixelgl"
	"github.com/pkg/errors"
)

func mapSizeFinder(mapFile io.Reader) (x int, y int, err error) {
	buf := bufio.NewScanner(mapFile)
	x = 0
	y = 0
	for buf.Scan() {
		mapRow := buf.Text()
		if len(mapRow) > x {
			x = len(mapRow)
		}
		y += 1
	}
	if err = buf.Err(); err != nil {
		return 0, 0, err
	}
	return x, y, nil
}


func NewMapFromText(
	mapFilePath string,
	spriteSheetPath string,
	spriteMapPath string,
	frameWidth float64,
) (mapCanvas *pixelgl.Canvas, err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, "error loading map from text")
		}
	}()


	// first, get the spritesheet
	spriteSheet, err := loadPicture(spriteSheetPath)
	if err != nil {
		return nil, err
	}

	// slice the spritesheet into frames
        var frames [][]pixel.Rect
        for y := spriteSheet.Bounds().Max.Y-frameWidth; y >= frameWidth; y -= frameWidth {
                var row []pixel.Rect
                for x := 0.0; x+frameWidth <= spriteSheet.Bounds().Max.X; x += frameWidth {
                        row = append(row, pixel.R(
                                x,
                                y,
                                x+frameWidth,
                                y+frameWidth,
                        ))
                }
                frames = append(frames, row)
        }

	// get the csv map
	spriteMap, err := os.Open(spriteMapPath)
	if err != nil {
		return nil, err
	}
	defer spriteMap.Close()

	// map the csv map to the spritesheet to load location information
	mapTiles := make(map[string][]pixel.Rect)
	spriteMapper := csv.NewReader(spriteMap)
	for {
		mapTile, err := spriteMapper.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		name := mapTile[0]
		row, _ := strconv.Atoi(mapTile[1])
		start, _ := strconv.Atoi(mapTile[2])
		end, _ := strconv.Atoi(mapTile[3])
		mapTiles[name] = frames[row][start : end+1]
	}

	// get the mapFile
	mapFile, err := os.Open(mapFilePath)
	if err != nil {
		return nil, err
	}

	mapCharNameMap := make(map[rune]string)
	mapCharNameMap['W'] = "CastleMiddle"
	mapCharNameMap['C'] = "CastleCross"
	mapCharNameMap['D'] = "CastleEmpty"
	mapCharNameMap['O'] = "CastleWindow"
	mapCharNameMap[' '] = "CastleEmpty"

	mapWidth, mapHeight, err := mapSizeFinder(mapFile)
	if err != nil {
		return nil, err
	}

	// map bounds
	mapBounds := pixel.R(0,0,float64(mapWidth*32),float64(mapHeight*32))
	// create map canvas
	mapCanvas = pixelgl.NewCanvas(mapBounds)
	// scan the map file and add it to a canvas
	mapScanner := bufio.NewScanner(mapFile)
	cursorX := float64(0)
	cursorY := float64(mapHeight * 32)
	for mapScanner.Scan() {
		rowText := mapScanner.Text()
		for _, char := range rowText {
			tileName := mapCharNameMap[char]
			tileFrame := mapTiles[tileName][0]
			tile := pixel.NewSprite(spriteSheet, tileFrame)
			tile.Draw(mapCanvas, pixel.IM.Moved(pixel.Vec{cursorX,cursorY}))
			cursorX += 32
		}
		cursorX = 0
		cursorY -= 32
	}
	return mapCanvas, nil
}

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/util"
)



//to run:  go test -run=Bench -bench BenchmarkGol
func BenchmarkGol(b *testing.B) {
	// Disable all program output apart from benchmark results
	os.Stdout = nil
	var wg sync.WaitGroup
	test := gol.Params{ImageWidth: 512, ImageHeight: 512, Turns: 1000}
	for threads := 1; threads <= 16; threads++ {
		test.Threads = threads
		testName := fmt.Sprintf("%dx%dx%d-%d", test.ImageWidth, test.ImageHeight, test.Turns, test.Threads)
		b.Run(testName, func(b *testing.B) {
			for i := 0; i<b.N; i++ {
				wg.Add(1)
				events := make(chan gol.Event)
				go func() {
					defer wg.Done()
					gol.Run(test, events, nil)
				}()
				var cells []util.Cell
				for event := range events {
					switch e := event.(type) {
					case gol.FinalTurnComplete:
						e.Alive = cells
					}
				}
			}
		})
		wg.Wait()
	}
}

// TestGol tests 16x16, 64x64 and 512x512 images on 0, 1 and 100 turns using 1-16 worker threads.
func TestGol(t *testing.T) {
	tests := []gol.Params{
		{ImageWidth: 16, ImageHeight: 16},
		{ImageWidth: 64, ImageHeight: 64},
		{ImageWidth: 512, ImageHeight: 512},
	}
	for _, p := range tests {
		for _, turns := range []int{0, 1, 100} {
			p.Turns = turns
			expectedAlive := readAliveCells(
				"check/images/"+fmt.Sprintf("%vx%vx%v.pgm", p.ImageWidth, p.ImageHeight, turns),
				p.ImageWidth,
				p.ImageHeight,
			)
			for threads := 1; threads <= 16; threads++ {
				p.Threads = threads
				testName := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
				t.Run(testName, func(t *testing.T) {
					events := make(chan gol.Event)
					go gol.Run(p, events, nil)
					var cells []util.Cell
					for event := range events {
						switch e := event.(type) {
						case gol.FinalTurnComplete:
							cells = e.Alive
						}
					}
					assertEqualBoard(t, cells, expectedAlive, p)
				})
			}
		}
	}
}

func boardFail(t *testing.T, given, expected []util.Cell, p gol.Params) bool {
	errorString := fmt.Sprintf("-----------------\n\n  FAILED TEST\n  %vx%v\n  %d Workers\n  %d Turns\n", p.ImageWidth, p.ImageHeight, p.Threads, p.Turns)
	if p.ImageWidth == 16 && p.ImageHeight == 16 {
		errorString = errorString + util.AliveCellsToString(given, expected, p.ImageWidth, p.ImageHeight)
	}
	t.Error(errorString)
	return false
}

func assertEqualBoard(t *testing.T, given, expected []util.Cell, p gol.Params) bool {
	givenLen := len(given)
	expectedLen := len(expected)

	if givenLen != expectedLen {
		return boardFail(t, given, expected, p)
	}

	visited := make([]bool, expectedLen)
	for i := 0; i < givenLen; i++ {
		element := given[i]
		found := false
		for j := 0; j < expectedLen; j++ {
			if visited[j] {
				continue
			}
			if expected[j] == element {
				visited[j] = true
				found = true
				break
			}
		}
		if !found {
			return boardFail(t, given, expected, p)
		}
	}

	return true
}

func readAliveCells(path string, width, height int) []util.Cell {
	data, ioError := ioutil.ReadFile(path)
	util.Check(ioError)

	fields := strings.Fields(string(data))

	//fmt.Println("fields [0]: ",fields[0])
	//fmt.Println("fields [1]: ",fields[1])
	//fmt.Println("fields [2]: ",fields[2])
	//fmt.Println("fields [3]: ",fields[3])
	//fmt.Println("fields [4]: ",fields[4])

	if fields[0] != "P5" {
		panic("Not a pgm file")
	}

	imageWidth, _ := strconv.Atoi(fields[1])
	if imageWidth != width {
		panic("Incorrect width")
	}

	imageHeight, _ := strconv.Atoi(fields[2])
	if imageHeight != height {
		panic("Incorrect height")
	}

	maxval, _ := strconv.Atoi(fields[3])
	if maxval != 255 {
		panic("Incorrect maxval/bit depth")
	}

	image := []byte(fields[4])

	var cells []util.Cell
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cell := image[0]
			if cell != 0 {
				cells = append(cells, util.Cell{
					X: x,
					Y: y,
				})
			}
			image = image[1:]
		}
	}
	return cells
}
package gol

import (
	"fmt"
	"strconv"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}



func makeImmutableMatrix(matrix [][]uint8) func(y, x int) uint8 {
	return func(y, x int) uint8 {
		return matrix[y][x]
	}
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	alive := make([]util.Cell,0)
	height := p.ImageHeight
	width := p.ImageWidth

	for i:=0; i<height; i++ {
		for z:=0; z<width; z++ {
			if world[i][z]==255 {
				var x util.Cell
				x.X = z
				x.Y = i
				alive = append(alive, x)
			}
		}
	}
	return alive
}

func mod(a, b int) int {
	return (a % b + b) % b
}
func checkSurrounding(i, z, dimension int, neww [][]byte) int {
	x := 0
	if neww[mod(i-1,dimension)][z] == 255 {x++}
	if neww[mod(i+1,dimension)][z] == 255 {x++}
	if neww[i][mod(z+1,dimension)] == 255 {x++}
	if neww[i][mod(z-1,dimension)] == 255 {x++}
	if neww[mod(i-1,dimension)][mod(z+1,dimension)] == 255 {x++}
	if neww[mod(i-1,dimension)][mod(z-1,dimension)] == 255 {x++}
	if neww[mod(i+1,dimension)][mod(z+1,dimension)] == 255 {x++}
	if neww[mod(i+1,dimension)][mod(z-1,dimension)] == 255 {x++}
	return x
}

func calculateNextState(h int, w int, world [][]byte) [][]byte {
	neww := make([][]byte, h)
	for i := range neww {
		neww[i] = make([]byte, w)
		copy(neww[i], world[i][:])
	}

	for i:=0; i<h; i++ {
		for z:=0; z<w; z++ {
			alive := checkSurrounding(i,z,h,world)
			if world[i][z] == 0 && alive==3 {neww[i][z] = 255
			} else {
				if world[i][z] == 255 && (alive<2 || alive>3) {neww[i][z] = 0}
			}
		}
	}
	return neww
}

func gameOfLife(sy, ey, sx, ex int, initialWorld [][]byte, p Params) [][]byte {
	world := initialWorld
	for i := 0; i < p.Turns; i++ {
		world = calculateNextState(p.ImageHeight, p.ImageWidth, world)
	}
	return world
}
func worker(startY, endY, startX, endX int, initial [][]byte, out chan<- [][]uint8, p Params) {
	theMatrix := gameOfLife(startY,endY, startX, endX, initial, p)
	out <- theMatrix
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	filename := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight)
	c.ioCommand <- ioInput
	c.ioFilename <- filename


	// TODO: Create a 2D slice to store the world.
	inital := make([][]byte, p.ImageHeight)
	for i := range inital {
		inital[i] = make([]byte, p.ImageWidth)
	}
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			byte := <- c.ioInput
			inital[y][x] = byte
		}
	}

	fmt.Println("distributor initialised world")
	turn := 0

	var finalData [][]uint8


	//create channels for each worker thread
	chanz := make([]chan [][]uint8, p.Threads)
	for i:=0; i<p.Threads; i++ {
		chanz[i] = make(chan [][]uint8)
	}


	x := p.ImageHeight/p.Threads
	start := 0
	end := x

	if p.Threads == 1 {
		go worker(0,p.ImageHeight,0,p.ImageWidth,inital,chanz[0], p)
	} else {
		for i:=1; i<=p.Threads; i++ {
			go worker(start,end,0,p.ImageWidth,inital,chanz[i-1],p)
			start = start + x
			if i==p.Threads-1 {
				end = p.ImageHeight
			} else {
				end = end + x
			}
		}
	}
	for i:=0; i<p.Threads; i++ {
		y := <- chanz[i]
		close(chanz[i])
		finalData = append(finalData, y...)
	}


	// TODO: Execute all turns of the Game of Life.
	alive := calculateAliveCells(p, finalData)
	fmt.Println("turns executed")



	// TODO: Report the final state using FinalTurnCompleteEvent.
	final := FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          alive,
	}
	c.events <- final
	fmt.Println("final state sent")


	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

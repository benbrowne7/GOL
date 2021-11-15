package gol

import (
	"fmt"
	"strconv"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

var wg sync.WaitGroup

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}


func nAlive(p Params, world [][]byte) int {
	c := 0
	for i:= 0; i<p.ImageHeight; i++ {
		for z:= 0; z<p.ImageWidth; z++ {
			if world[i][z] == 255 {
				c++
			}
		}
	}
	return c
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	alive := make([]util.Cell,0)

	for i:=0; i<p.ImageHeight; i++ {
		for z:=0; z<p.ImageWidth; z++ {
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

func ticka(everytwo chan bool, nalive chan int, count chan int) {
	ticker := time.NewTicker(2 * time.Second)
	for _ = range ticker.C {
		everytwo <- true
		go bufferget(nalive, count)
	}
}
func bufferget(nalive chan int, count chan int) {
	x := <- nalive
	fmt.Println("in bufferget")
	//for i := range nalive {
	//	x = x + i
	//	fmt.Println("adding up alive")
	//}
	count <- x
	fmt.Println("sent value to count:", x)

}
func aliveSender(count chan int, turn *int, c distributorChannels) {
	for {
		fmt.Println("aliveSender waiting...")
		x := <- count
		fmt.Println("alivesender recieved to x")
		aliveEvent := AliveCellsCount{
			CompletedTurns: *turn,
			CellsCount:     x,
		}
		c.events <- aliveEvent
	}
}

func turnCounter(test chan bool, turn *int, c distributorChannels, p Params) {
	var y = float32(p.Threads)
	var x float32 = 0
	for {
		<- test
		x = x + 1/y
		if x==1 {
			*turn++
			c.events <- TurnComplete{CompletedTurns: *turn}
			x = 0
		}
	}
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

func calculateNextState(sy, ey, h int, w int, world [][]byte) [][]byte {
	neww := make([][]byte, h)
	for i := range neww {
		neww[i] = make([]byte, w)
		copy(neww[i], world[i][:])
	}

	for i:=sy; i<ey; i++ {
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

func gameOfLife(sy, ey int, initialWorld [][]byte, p Params, everytwo chan bool, nalive chan int, test, next chan bool) [][]byte {
	world := initialWorld
	select {
	case command := <- everytwo:
		switch command {
		case true:
			x := nAlive(p, world)
			nalive <- x
			fmt.Println("x:", x)
		}
	default:
		test <- true
		world = calculateNextState(sy, ey, p.ImageHeight, p.ImageWidth, world)
	}
	return world
}
func worker(startY, endY int, initial [][]byte, iteration chan<- [][]byte, p Params, everytwo chan bool, nalive chan int, test, next chan bool) {
	theMatrix := gameOfLife(startY,endY, initial, p, everytwo, nalive, test, next)
	iteration <- theMatrix[startY:endY][0:]
	fmt.Println("work sent to iteration")
}


func controller(ratio int, p Params, iteration, chanz []chan [][]uint8, world [][]byte, nalive chan int, everytwo, test, next chan bool) [][]byte {
	start := 0
	end := ratio
	temp := make(chan [][]byte)
	go iterationMaker(iteration, temp)
	if p.Threads == 1 {
		go worker(0,p.ImageHeight,world,iteration[0], p, everytwo, nalive, test, next)
	} else {
		for i:=1; i<=p.Threads; i++ {
			go worker(start,end,world,iteration[i-1],p, everytwo, nalive, test, next)
			start = start + ratio
			if i==p.Threads-1 {
				end = p.ImageHeight
			} else {
				end = end + ratio
			}
		}
	}
	x := <- temp
	return x
}

func iterationMaker(iteration []chan [][]byte, temp chan [][]byte) {
	var world [][]byte
	for x := range iteration {
		y := <- iteration[x]
		world = append(world, y...)
	}
	temp <- world
	fmt.Println("iteration made and sent")
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
	iteration := make([]chan [][]uint8, p.Threads)
	for i:=0; i<p.Threads; i++ {
		iteration[i] = make(chan [][]uint8)
	}


	x := p.ImageHeight/p.Threads


	//create chan for sending n. alive
	nalive := make(chan int, p.Threads)
	everytwo := make(chan bool)
	count := make(chan int)
	test := make(chan bool)
	next := make(chan bool)

	go ticka(everytwo, nalive, count)
	go aliveSender(count, &turn, c)
	go turnCounter(test, &turn, c, p)
	fmt.Println("aliveSender+ticker routines started")

	for i:=0; i<p.Turns; i++ {
		inital = controller(x, p, iteration, chanz, inital, nalive, everytwo, test, next)
	}
	finalData = inital


	//for i:=0; i<p.Threads; i++ {
	//	y := <- chanz[i]
	//	close(chanz[i])
	//	finalData = append(finalData, y...)
	//}


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
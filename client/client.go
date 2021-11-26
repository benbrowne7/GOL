package main

import (
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

func handleError(err error) {
	fmt.Println("errar")
	log.Fatal(err)
}

func calculateAliveCells(p gol.Params, world [][]byte) []util.Cell {
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

func main() {
	server := flag.String("server","127.0.0.1:8030","IP:port string to connect to as server")
	flag.Parse()
	fmt.Println("Server: ", *server)
	client, err := rpc.Dial("tcp", *server)
	if err != nil {
		fmt.Println("accept error")
		handleError(err)
	}
	defer client.Close()

	p := gol.Params{
		Turns:       500,
		Threads:     1,
		ImageWidth:  64,
		ImageHeight: 64,
	}
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if y%2 == 0 {
				world[y][x] = 255
			}
		}
	}
	iteration := make([]chan [][]byte, p.Threads)
	for i:=0; i<p.Threads; i++ {
		iteration[i] = make(chan [][]byte)
	}
	turn := 0
	ratio := p.ImageHeight/p.Threads

	for i:=0; i<p.Turns; i++ {
		request := stubs.Request{
			World:     world,
			P:         p,
			Ratio:     ratio,
			Iteration: iteration,
			Turn:      turn,}

		response := new(stubs.Response)
		err = client.Call(stubs.GameOfLife, request, response)
		if err != nil {
			fmt.Println("client.call error")
		}
		world = response.World
		turn++
	}
	//finalData := world
	//alive := calculateAliveCells(p, finalData)
	//final := gol.FinalTurnComplete{
	//	CompletedTurns: turn,
	//	Alive:          alive,
	//}



}

package stubs

import (
	"uk.ac.bris.cs/gameoflife/gol"
)

var GameOfLife = "GameOfLife.Process"



type Response struct {
	World [][]byte
}

type Request struct {
	World     [][]byte
	P         gol.Params
	Ratio     int
	Iteration []chan [][]byte
	Turn      int
}




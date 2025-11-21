package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type Grid struct {
	Width  int
	Height int
}

type Position struct {
	X, Y int
}

type Tetromino struct {
	Type     int
	Shape    [][]bool
	Color    rl.Color
	Rotation int
}

type Velocity struct {
	FallSpeed   float32
	Accumulator float32
}

type LockedPiece struct {
	Color rl.Color
}

type GameState struct {
	Score        int
	Level        int
	LinesCleared int
	GameOver     bool
	SpawnTimer   float32
	LockDelay    float32
	NextPieces   []int
}

type InputState struct {
	MoveLeftTime  float32
	MoveRightTime float32
	DownTime      float32
	RepeatDelay   float32
	RepeatRate    float32
}

type CollisionMap struct {
	OccupiedCells map[[2]int]bool
}

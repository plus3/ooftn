package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/plus3/ooftn/ecs"
)

func main() {
	rl.InitWindow(500, 700, "Tetris - ECS Example")
	rl.SetTargetFPS(60)
	defer rl.CloseWindow()

	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Grid](registry)
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[Tetromino](registry)
	ecs.RegisterComponent[Velocity](registry)
	ecs.RegisterComponent[LockedPiece](registry)
	ecs.RegisterComponent[GameState](registry)
	ecs.RegisterComponent[InputState](registry)
	ecs.RegisterComponent[CollisionMap](registry)

	storage := ecs.NewStorage(registry)

	initGame(storage)

	scheduler := ecs.NewScheduler(storage)
	scheduler.Register(&CollisionSystem{})
	scheduler.Register(&InputSystem{})
	scheduler.Register(&GravitySystem{})
	scheduler.Register(&LockSystem{})
	scheduler.Register(&LineClearSystem{})
	scheduler.Register(&SpawnSystem{})
	scheduler.Register(&RenderSystem{})

	lastTime := rl.GetTime()

	for !rl.WindowShouldClose() {
		currentTime := rl.GetTime()
		deltaTime := currentTime - lastTime
		lastTime = currentTime

		if rl.IsKeyPressed(rl.KeyR) {
			resetGame(storage)
		}

		scheduler.Once(deltaTime)
	}
}

func initGame(storage *ecs.Storage) {
	ecs.NewSingleton[Grid](storage, Grid{
		Width:  GridWidth,
		Height: GridHeight,
	})

	ecs.NewSingleton[GameState](storage, GameState{
		Score:        0,
		Level:        1,
		LinesCleared: 0,
		GameOver:     false,
		SpawnTimer:   0,
		LockDelay:    0,
		NextPieces:   []int{},
	})

	ecs.NewSingleton[CollisionMap](storage, CollisionMap{
		OccupiedCells: make(map[[2]int]bool),
	})
}

func resetGame(storage *ecs.Storage) {
	// Reset singleton game state
	var gameState *GameState
	if storage.ReadSingleton(&gameState) {
		gameState.Score = 0
		gameState.Level = 1
		gameState.LinesCleared = 0
		gameState.GameOver = false
		gameState.SpawnTimer = 0
		gameState.LockDelay = 0
		gameState.NextPieces = []int{}
	}

	// Reset collision map
	var collisionMap *CollisionMap
	if storage.ReadSingleton(&collisionMap) {
		collisionMap.OccupiedCells = make(map[[2]int]bool)
	}

	// Delete all active pieces
	activePieceView := ecs.NewView[struct {
		ecs.EntityId
		*Position
		*Tetromino
	}](storage)

	for entity := range activePieceView.Iter() {
		storage.Delete(entity.EntityId)
	}

	// Delete all locked pieces
	lockedView := ecs.NewView[struct {
		ecs.EntityId
		*LockedPiece
	}](storage)

	for entity := range lockedView.Iter() {
		storage.Delete(entity.EntityId)
	}
}

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
	storage.Spawn(Grid{
		Width:  GridWidth,
		Height: GridHeight,
	})

	storage.Spawn(GameState{
		Score:        0,
		Level:        1,
		LinesCleared: 0,
		GameOver:     false,
		SpawnTimer:   0,
		LockDelay:    0,
		NextPieces:   []int{},
	})

	storage.Spawn(CollisionMap{
		OccupiedCells: make(map[[2]int]bool),
	})
}

func resetGame(storage *ecs.Storage) {
	view := ecs.NewView[struct {
		ecs.EntityId
		*GameState
	}](storage)

	for entity := range view.Iter() {
		entity.GameState.Score = 0
		entity.GameState.Level = 1
		entity.GameState.LinesCleared = 0
		entity.GameState.GameOver = false
		entity.GameState.SpawnTimer = 0
		entity.GameState.LockDelay = 0
		entity.GameState.NextPieces = []int{}
	}

	activePieceView := ecs.NewView[struct {
		ecs.EntityId
		*Position
		*Tetromino
	}](storage)

	for entity := range activePieceView.Iter() {
		storage.Delete(entity.EntityId)
	}

	lockedView := ecs.NewView[struct {
		ecs.EntityId
		*LockedPiece
	}](storage)

	for entity := range lockedView.Iter() {
		storage.Delete(entity.EntityId)
	}
}

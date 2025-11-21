package main

import (
	"fmt"
	"math/rand/v2"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/plus3/ooftn/ecs"
)

const (
	GridWidth  = 10
	GridHeight = 20
	CellSize   = 30
)

var tetrominoShapes = [][][]bool{
	{ // I
		{false, false, false, false},
		{true, true, true, true},
		{false, false, false, false},
		{false, false, false, false},
	},
	{ // O
		{false, false, false, false},
		{false, true, true, false},
		{false, true, true, false},
		{false, false, false, false},
	},
	{ // T
		{false, false, false, false},
		{false, true, false, false},
		{true, true, true, false},
		{false, false, false, false},
	},
	{ // S
		{false, false, false, false},
		{false, true, true, false},
		{true, true, false, false},
		{false, false, false, false},
	},
	{ // Z
		{false, false, false, false},
		{true, true, false, false},
		{false, true, true, false},
		{false, false, false, false},
	},
	{ // J
		{false, false, false, false},
		{true, false, false, false},
		{true, true, true, false},
		{false, false, false, false},
	},
	{ // L
		{false, false, false, false},
		{false, false, true, false},
		{true, true, true, false},
		{false, false, false, false},
	},
}

var tetrominoColors = []rl.Color{
	rl.SkyBlue,
	rl.Gold,
	rl.Violet,
	rl.Lime,
	rl.Pink,
	rl.Blue,
	rl.Orange,
}

func rotateShape(shape [][]bool, clockwise bool) [][]bool {
	size := len(shape)
	rotated := make([][]bool, size)
	for i := range rotated {
		rotated[i] = make([]bool, size)
	}

	for i := range size {
		for j := range size {
			if clockwise {
				rotated[j][size-1-i] = shape[i][j]
			} else {
				rotated[size-1-j][i] = shape[i][j]
			}
		}
	}

	return rotated
}

func checkCollision(shape [][]bool, pos Position, grid *Grid, collisionMap *CollisionMap) bool {
	for i := range shape {
		for j, value := range shape[i] {
			if !value {
				continue
			}

			x := pos.X + j
			y := pos.Y + i

			if x < 0 || x >= grid.Width || y >= grid.Height {
				return true
			}

			if y >= 0 && collisionMap.OccupiedCells[[2]int{x, y}] {
				return true
			}
		}
	}

	return false
}

type CollisionSystem struct {
	LockedPieces ecs.Query[struct {
		*Position
		*LockedPiece
	}]
	CollisionMap ecs.Singleton[CollisionMap]
}

func (s *CollisionSystem) Execute(frame *ecs.UpdateFrame) {
	collisionMap := s.CollisionMap.Get()
	if collisionMap == nil {
		return
	}

	collisionMap.OccupiedCells = make(map[[2]int]bool)
	for piece := range s.LockedPieces.Iter() {
		collisionMap.OccupiedCells[[2]int{piece.Position.X, piece.Position.Y}] = true
	}
}

type InputSystem struct {
	ActivePiece ecs.Query[struct {
		*Position
		*Tetromino
		*InputState
	}]
	Grid         ecs.Singleton[Grid]
	CollisionMap ecs.Singleton[CollisionMap]
}

func (s *InputSystem) Execute(frame *ecs.UpdateFrame) {
	grid := s.Grid.Get()
	if grid == nil {
		return
	}

	collisionMap := s.CollisionMap.Get()
	if collisionMap == nil {
		return
	}

	for entity := range s.ActivePiece.Iter() {
		dt := float32(frame.DeltaTime)

		if rl.IsKeyPressed(rl.KeyLeft) {
			entity.InputState.MoveLeftTime = 0
			newPos := Position{X: entity.Position.X - 1, Y: entity.Position.Y}
			if !checkCollision(entity.Tetromino.Shape, newPos, grid, collisionMap) {
				entity.Position.X--
			}
		} else if rl.IsKeyDown(rl.KeyLeft) {
			entity.InputState.MoveLeftTime += dt
			if entity.InputState.MoveLeftTime > entity.InputState.RepeatDelay {
				entity.InputState.MoveLeftTime -= entity.InputState.RepeatRate
				newPos := Position{X: entity.Position.X - 1, Y: entity.Position.Y}
				if !checkCollision(entity.Tetromino.Shape, newPos, grid, collisionMap) {
					entity.Position.X--
				}
			}
		} else {
			entity.InputState.MoveLeftTime = 0
		}

		if rl.IsKeyPressed(rl.KeyRight) {
			entity.InputState.MoveRightTime = 0
			newPos := Position{X: entity.Position.X + 1, Y: entity.Position.Y}
			if !checkCollision(entity.Tetromino.Shape, newPos, grid, collisionMap) {
				entity.Position.X++
			}
		} else if rl.IsKeyDown(rl.KeyRight) {
			entity.InputState.MoveRightTime += dt
			if entity.InputState.MoveRightTime > entity.InputState.RepeatDelay {
				entity.InputState.MoveRightTime -= entity.InputState.RepeatRate
				newPos := Position{X: entity.Position.X + 1, Y: entity.Position.Y}
				if !checkCollision(entity.Tetromino.Shape, newPos, grid, collisionMap) {
					entity.Position.X++
				}
			}
		} else {
			entity.InputState.MoveRightTime = 0
		}

		if rl.IsKeyDown(rl.KeyDown) {
			entity.InputState.DownTime += dt
			if entity.InputState.DownTime > 0.05 {
				entity.InputState.DownTime = 0
				newPos := Position{X: entity.Position.X, Y: entity.Position.Y + 1}
				if !checkCollision(entity.Tetromino.Shape, newPos, grid, collisionMap) {
					entity.Position.Y++
				}
			}
		} else {
			entity.InputState.DownTime = 0
		}

		if rl.IsKeyPressed(rl.KeyUp) || rl.IsKeyPressed(rl.KeyZ) {
			rotated := rotateShape(entity.Tetromino.Shape, true)
			if !checkCollision(rotated, *entity.Position, grid, collisionMap) {
				entity.Tetromino.Shape = rotated
				entity.Tetromino.Rotation = (entity.Tetromino.Rotation + 1) % 4
			}
		}

		if rl.IsKeyPressed(rl.KeyX) {
			rotated := rotateShape(entity.Tetromino.Shape, false)
			if !checkCollision(rotated, *entity.Position, grid, collisionMap) {
				entity.Tetromino.Shape = rotated
				entity.Tetromino.Rotation = (entity.Tetromino.Rotation + 3) % 4
			}
		}

		if rl.IsKeyPressed(rl.KeySpace) {
			for {
				newPos := Position{X: entity.Position.X, Y: entity.Position.Y + 1}
				if checkCollision(entity.Tetromino.Shape, newPos, grid, collisionMap) {
					break
				}
				entity.Position.Y++
			}
		}
	}
}

type GravitySystem struct {
	ActivePiece ecs.Query[struct {
		*Position
		*Velocity
		*Tetromino
	}]
	Grid         ecs.Singleton[Grid]
	GameState    ecs.Singleton[GameState]
	CollisionMap ecs.Singleton[CollisionMap]
}

func (s *GravitySystem) Execute(frame *ecs.UpdateFrame) {
	grid := s.Grid.Get()
	if grid == nil {
		return
	}

	gameState := s.GameState.Get()
	if gameState == nil || gameState.GameOver {
		return
	}

	collisionMap := s.CollisionMap.Get()
	if collisionMap == nil {
		return
	}

	for entity := range s.ActivePiece.Iter() {
		entity.Velocity.Accumulator += float32(frame.DeltaTime)

		newPos := Position{X: entity.Position.X, Y: entity.Position.Y + 1}
		isGrounded := checkCollision(entity.Tetromino.Shape, newPos, grid, collisionMap)

		if isGrounded {
			gameState.LockDelay += float32(frame.DeltaTime)
		} else {
			gameState.LockDelay = 0
		}

		if entity.Velocity.Accumulator >= 1.0/entity.Velocity.FallSpeed {
			entity.Velocity.Accumulator = 0

			if !isGrounded {
				entity.Position.Y++
			}
		}
	}
}

type LockSystem struct {
	ActivePiece ecs.Query[struct {
		ecs.EntityId
		*Position
		*Tetromino
	}]
	Grid      ecs.Singleton[Grid]
	GameState ecs.Singleton[GameState]
}

func (s *LockSystem) Execute(frame *ecs.UpdateFrame) {
	grid := s.Grid.Get()
	if grid == nil {
		return
	}

	gameState := s.GameState.Get()
	if gameState == nil || gameState.GameOver {
		return
	}

	for entity := range s.ActivePiece.Iter() {
		if gameState.LockDelay >= 0.5 {
			for i := 0; i < len(entity.Tetromino.Shape); i++ {
				for j := 0; j < len(entity.Tetromino.Shape[i]); j++ {
					if entity.Tetromino.Shape[i][j] {
						x := entity.Position.X + j
						y := entity.Position.Y + i

						if y >= 0 && y < grid.Height && x >= 0 && x < grid.Width {
							frame.Commands.Spawn(
								Position{X: x, Y: y},
								LockedPiece{Color: entity.Tetromino.Color},
							)
						}
					}
				}
			}

			frame.Commands.Delete(entity.EntityId)
			gameState.LockDelay = 0
			gameState.SpawnTimer = 0.1
		}
	}
}

type LineClearSystem struct {
	Grid         ecs.Singleton[Grid]
	GameState    ecs.Singleton[GameState]
	LockedPieces ecs.Query[struct {
		ecs.EntityId
		*Position
		*LockedPiece
	}]
}

func (s *LineClearSystem) Execute(frame *ecs.UpdateFrame) {
	grid := s.Grid.Get()
	if grid == nil {
		return
	}

	gameState := s.GameState.Get()
	if gameState == nil {
		return
	}

	rowCounts := make(map[int]int)
	for y := 0; y < grid.Height; y++ {
		rowCounts[y] = 0
	}

	for entity := range s.LockedPieces.Iter() {
		rowCounts[entity.Position.Y]++
	}

	completedLines := []int{}
	for y := 0; y < grid.Height; y++ {
		if rowCounts[y] == grid.Width {
			completedLines = append(completedLines, y)
		}
	}

	if len(completedLines) == 0 {
		return
	}

	completedLinesSet := make(map[int]bool)
	for _, y := range completedLines {
		completedLinesSet[y] = true
	}

	for _, y := range completedLines {
		for entity := range s.LockedPieces.Iter() {
			if entity.Position.Y == y {
				frame.Commands.Delete(entity.EntityId)
			}
		}
	}

	for entity := range s.LockedPieces.Iter() {
		linesBelow := 0
		for _, clearedY := range completedLines {
			if clearedY > entity.Position.Y {
				linesBelow++
			}
		}
		entity.Position.Y += linesBelow
	}

	gameState.LinesCleared += len(completedLines)
	gameState.Score += len(completedLines) * 100
	gameState.Level = gameState.LinesCleared/10 + 1
}

type SpawnSystem struct {
	ActivePiece ecs.Query[struct {
		*Tetromino
	}]
	Grid         ecs.Singleton[Grid]
	GameState    ecs.Singleton[GameState]
	CollisionMap ecs.Singleton[CollisionMap]
}

func (s *SpawnSystem) Execute(frame *ecs.UpdateFrame) {
	hasActivePiece := false
	for range s.ActivePiece.Iter() {
		hasActivePiece = true
		break
	}

	if hasActivePiece {
		return
	}

	grid := s.Grid.Get()
	if grid == nil {
		return
	}

	gameState := s.GameState.Get()
	if gameState == nil || gameState.GameOver {
		return
	}

	collisionMap := s.CollisionMap.Get()
	if collisionMap == nil {
		return
	}

	gameState.SpawnTimer += float32(frame.DeltaTime)
	if gameState.SpawnTimer < 0.1 {
		return
	}

	if len(gameState.NextPieces) == 0 {
		bag := []int{0, 1, 2, 3, 4, 5, 6}
		rand.Shuffle(len(bag), func(i, j int) {
			bag[i], bag[j] = bag[j], bag[i]
		})
		gameState.NextPieces = bag
	}

	pieceType := gameState.NextPieces[0]
	gameState.NextPieces = gameState.NextPieces[1:]

	shape := make([][]bool, len(tetrominoShapes[pieceType]))
	for i := range shape {
		shape[i] = make([]bool, len(tetrominoShapes[pieceType][i]))
		copy(shape[i], tetrominoShapes[pieceType][i])
	}

	pos := Position{X: 3, Y: 0}
	tetromino := Tetromino{
		Type:     pieceType,
		Shape:    shape,
		Color:    tetrominoColors[pieceType],
		Rotation: 0,
	}

	if checkCollision(shape, pos, grid, collisionMap) {
		gameState.GameOver = true
		return
	}

	fallSpeed := float32(1.0 + float32(gameState.Level)*0.1)

	frame.Commands.Spawn(
		pos,
		tetromino,
		Velocity{FallSpeed: fallSpeed, Accumulator: 0},
		InputState{RepeatDelay: 0.2, RepeatRate: 0.05},
	)
}

type RenderSystem struct {
	Grid         ecs.Singleton[Grid]
	GameState    ecs.Singleton[GameState]
	CollisionMap ecs.Singleton[CollisionMap]
	ActivePiece  ecs.Query[struct {
		*Position
		*Tetromino
	}]
	LockedPieces ecs.Query[struct {
		*Position
		*LockedPiece
	}]
}

func (s *RenderSystem) Execute(frame *ecs.UpdateFrame) {
	rl.BeginDrawing()
	rl.ClearBackground(rl.Black)

	offsetX := int32(50)
	offsetY := int32(50)

	rl.DrawRectangleLines(offsetX-2, offsetY-2, GridWidth*CellSize+4, GridHeight*CellSize+4, rl.Gray)

	grid := s.Grid.Get()

	for entity := range s.LockedPieces.Iter() {
		x := offsetX + int32(entity.Position.X*CellSize)
		y := offsetY + int32(entity.Position.Y*CellSize)
		rl.DrawRectangle(x, y, CellSize, CellSize, entity.LockedPiece.Color)
		rl.DrawRectangleLines(x, y, CellSize, CellSize, rl.Black)
	}

	collisionMap := s.CollisionMap.Get()

	for entity := range s.ActivePiece.Iter() {
		if grid != nil && collisionMap != nil {
			ghostY := entity.Position.Y
			for {
				testPos := Position{X: entity.Position.X, Y: ghostY + 1}
				if checkCollision(entity.Tetromino.Shape, testPos, grid, collisionMap) {
					break
				}
				ghostY++
			}

			ghostColor := rl.NewColor(255, 255, 255, 80)
			for i := 0; i < len(entity.Tetromino.Shape); i++ {
				for j := 0; j < len(entity.Tetromino.Shape[i]); j++ {
					if entity.Tetromino.Shape[i][j] {
						x := offsetX + int32((entity.Position.X+j)*CellSize)
						y := offsetY + int32((ghostY+i)*CellSize)
						rl.DrawRectangle(x, y, CellSize, CellSize, ghostColor)
					}
				}
			}
		}

		for i := 0; i < len(entity.Tetromino.Shape); i++ {
			for j := 0; j < len(entity.Tetromino.Shape[i]); j++ {
				if entity.Tetromino.Shape[i][j] {
					x := offsetX + int32((entity.Position.X+j)*CellSize)
					y := offsetY + int32((entity.Position.Y+i)*CellSize)
					rl.DrawRectangle(x, y, CellSize, CellSize, entity.Tetromino.Color)
					rl.DrawRectangleLines(x, y, CellSize, CellSize, rl.Black)
				}
			}
		}
	}

	gameState := s.GameState.Get()
	if gameState != nil {
		textX := offsetX + GridWidth*CellSize + 20
		rl.DrawText("SCORE", textX, offsetY, 20, rl.White)
		rl.DrawText(fmt.Sprintf("%d", gameState.Score), textX, offsetY+25, 20, rl.White)

		rl.DrawText("LEVEL", textX, offsetY+60, 20, rl.White)
		rl.DrawText(fmt.Sprintf("%d", gameState.Level), textX, offsetY+85, 20, rl.White)

		rl.DrawText("LINES", textX, offsetY+120, 20, rl.White)
		rl.DrawText(fmt.Sprintf("%d", gameState.LinesCleared), textX, offsetY+145, 20, rl.White)

		if gameState.GameOver {
			rl.DrawText("GAME OVER", offsetX+20, offsetY+GridHeight*CellSize/2-10, 30, rl.Red)
			rl.DrawText("Press R to restart", offsetX+10, offsetY+GridHeight*CellSize/2+30, 20, rl.White)
		}
	}

	rl.EndDrawing()
}

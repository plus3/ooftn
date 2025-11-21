package ecs_test

import (
	"context"
	"fmt"
	"time"

	"github.com/plus3/ooftn/ecs"
)

type Transform struct {
	X, Y float32
}

type Speed struct {
	DX, DY float32
}

type Hitpoints struct {
	Current, Max int
}

type PhysicsSystem struct {
	Entities ecs.Query[struct {
		*Transform
		*Speed
	}]
}

func (s *PhysicsSystem) Execute(frame *ecs.UpdateFrame) {
	for entity := range s.Entities.Iter() {
		entity.Transform.X += entity.Speed.DX * float32(frame.DeltaTime)
		entity.Transform.Y += entity.Speed.DY * float32(frame.DeltaTime)
	}
}

type HealingSystem struct {
	Entities  ecs.Query[struct{ *Hitpoints }]
	RegenRate float32
}

func (s *HealingSystem) Execute(frame *ecs.UpdateFrame) {
	for entity := range s.Entities.Iter() {
		if entity.Hitpoints.Current < entity.Hitpoints.Max {
			entity.Hitpoints.Current += int(s.RegenRate * float32(frame.DeltaTime))
			if entity.Hitpoints.Current > entity.Hitpoints.Max {
				entity.Hitpoints.Current = entity.Hitpoints.Max
			}
		}
	}
}

// ExampleScheduler demonstrates building a game loop with multiple systems.
// The Scheduler manages system execution order, automatically initializes
// Query fields, executes queries each frame, and flushes command buffers.
// Systems are executed in registration order, and all queries are synchronized
// before any system runs.
func ExampleScheduler() {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Transform](registry)
	ecs.RegisterComponent[Speed](registry)
	ecs.RegisterComponent[Hitpoints](registry)
	storage := ecs.NewStorage(registry)

	storage.Spawn(
		Transform{X: 0, Y: 0},
		Speed{DX: 10, DY: 5},
		Hitpoints{Current: 80, Max: 100},
	)
	storage.Spawn(
		Transform{X: 100, Y: 100},
		Speed{DX: -5, DY: -5},
		Hitpoints{Current: 50, Max: 100},
	)

	scheduler := ecs.NewScheduler(storage)
	scheduler.Register(&PhysicsSystem{})
	scheduler.Register(&HealingSystem{RegenRate: 10})

	scheduler.Once(1.0)

	view := ecs.NewView[struct {
		*Transform
		*Hitpoints
	}](storage)

	fmt.Println("After one frame:")
	for item := range view.Iter() {
		fmt.Printf("Position: (%.0f, %.0f), Health: %d/%d\n",
			item.Transform.X, item.Transform.Y,
			item.Hitpoints.Current, item.Hitpoints.Max)
	}

	// Output:
	// After one frame:
	// Position: (10, 5), Health: 90/100
	// Position: (95, 95), Health: 60/100
}

// ExampleScheduler_Run demonstrates running a continuous game loop.
// The Run method blocks and executes all systems at a fixed interval
// until the context is cancelled. This is the typical pattern for
// a real-time game or simulation.
func ExampleScheduler_Run() {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Transform](registry)
	ecs.RegisterComponent[Speed](registry)
	storage := ecs.NewStorage(registry)

	storage.Spawn(Transform{X: 0, Y: 0}, Speed{DX: 1, DY: 1})

	scheduler := ecs.NewScheduler(storage)
	scheduler.Register(&PhysicsSystem{})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	scheduler.Run(ctx, 16*time.Millisecond)

	fmt.Println("Scheduler stopped")
	// Output:
	// Scheduler stopped
}

type GameTime struct {
	TotalFrames int
	TotalTime   float64
}

type TimeTracker struct {
	Entities ecs.Query[struct{ *Transform }]
	GameTime ecs.Singleton[GameTime]
}

func (s *TimeTracker) Execute(frame *ecs.UpdateFrame) {
	gameTime := s.GameTime.Get()
	gameTime.TotalFrames++
	gameTime.TotalTime += frame.DeltaTime
}

type ScoreTracker struct {
	Points int
}

type ScoreSystem struct {
	Entities ecs.Query[struct{ *Transform }]
	Score    ecs.Singleton[ScoreTracker]
}

func (s *ScoreSystem) Execute(frame *ecs.UpdateFrame) {
	count := 0
	for range s.Entities.Iter() {
		count++
	}
	s.Score.Get().Points += count * 10
}

// ExampleScheduler_withSingletons demonstrates using singleton components in systems.
// Singleton fields are automatically initialized by the Scheduler, just like Query fields.
// This provides efficient access to global state without the iteration overhead of queries.
func ExampleScheduler_withSingletons() {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Transform](registry)
	storage := ecs.NewStorage(registry)

	// Initialize singletons
	ecs.NewSingleton[GameTime](storage, GameTime{TotalFrames: 0, TotalTime: 0})
	ecs.NewSingleton[ScoreTracker](storage, ScoreTracker{Points: 0})

	// Spawn entities
	storage.Spawn(Transform{X: 0, Y: 0})
	storage.Spawn(Transform{X: 10, Y: 10})
	storage.Spawn(Transform{X: 20, Y: 20})

	// Create scheduler with systems that use singletons
	scheduler := ecs.NewScheduler(storage)
	scheduler.Register(&TimeTracker{})
	scheduler.Register(&ScoreSystem{})

	// Run for 3 frames
	scheduler.Once(0.016)
	scheduler.Once(0.016)
	scheduler.Once(0.016)

	// Check singleton values
	var gameTime *GameTime
	storage.ReadSingleton(&gameTime)
	fmt.Printf("Frames: %d, Time: %.3f\n", gameTime.TotalFrames, gameTime.TotalTime)

	var score *ScoreTracker
	storage.ReadSingleton(&score)
	fmt.Printf("Score: %d points\n", score.Points)

	// Output:
	// Frames: 3, Time: 0.048
	// Score: 90 points
}

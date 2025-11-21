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

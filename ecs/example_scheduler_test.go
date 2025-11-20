package ecs_test

import (
	"context"
	"fmt"
	"time"

	"github.com/plus3/ooftn/ecs"
)

// Define component types
type Transform struct {
	X, Y float32
}

type Speed struct {
	DX, DY float32
}

type Hitpoints struct {
	Current, Max int
}

// PhysicsSystem updates entity positions based on their speed
type PhysicsSystem struct {
	// Query fields are automatically initialized by the Scheduler
	Entities ecs.Query[struct {
		*Transform
		*Speed
	}]
}

func (s *PhysicsSystem) Execute(frame *ecs.UpdateFrame) {
	// Iterate over all entities with Transform and Speed components
	for entity := range s.Entities.IterValues() {
		entity.Transform.X += entity.Speed.DX * float32(frame.DeltaTime)
		entity.Transform.Y += entity.Speed.DY * float32(frame.DeltaTime)
	}
}

// HealingSystem regenerates hitpoints for entities over time
type HealingSystem struct {
	Entities ecs.Query[struct {
		*Hitpoints
	}]
	RegenRate float32 // Custom state persists between frames
}

func (s *HealingSystem) Execute(frame *ecs.UpdateFrame) {
	for entity := range s.Entities.IterValues() {
		// Regenerate hitpoints if not at max
		if entity.Hitpoints.Current < entity.Hitpoints.Max {
			entity.Hitpoints.Current += int(s.RegenRate * float32(frame.DeltaTime))
			if entity.Hitpoints.Current > entity.Hitpoints.Max {
				entity.Hitpoints.Current = entity.Hitpoints.Max
			}
		}
	}
}

func ExampleScheduler() {
	// Set up component registry and storage
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Transform](registry)
	ecs.RegisterComponent[Speed](registry)
	ecs.RegisterComponent[Hitpoints](registry)
	storage := ecs.NewStorage(registry)

	// Create some entities
	player := storage.Spawn(
		Transform{X: 0, Y: 0},
		Speed{DX: 10, DY: 5},
		Hitpoints{Current: 80, Max: 100},
	)
	enemy := storage.Spawn(
		Transform{X: 100, Y: 100},
		Speed{DX: -5, DY: -5},
		Hitpoints{Current: 50, Max: 100},
	)

	// Create scheduler and register systems
	scheduler := ecs.NewScheduler(storage)

	// Systems execute in registration order
	scheduler.Register(&PhysicsSystem{})
	scheduler.Register(&HealingSystem{RegenRate: 10}) // Regenerate 10 HP per second

	// Run a single frame with 1 second delta time
	scheduler.Once(1.0)

	// Check results
	playerTransform := ecs.ReadComponent[Transform](storage, player)
	playerHP := ecs.ReadComponent[Hitpoints](storage, player)

	fmt.Printf("Player position: (%.0f, %.0f)\n", playerTransform.X, playerTransform.Y)
	fmt.Printf("Player health: %d/%d\n", playerHP.Current, playerHP.Max)

	enemyTransform := ecs.ReadComponent[Transform](storage, enemy)
	enemyHP := ecs.ReadComponent[Hitpoints](storage, enemy)

	fmt.Printf("Enemy position: (%.0f, %.0f)\n", enemyTransform.X, enemyTransform.Y)
	fmt.Printf("Enemy health: %d/%d\n", enemyHP.Current, enemyHP.Max)

	// Output:
	// Player position: (10, 5)
	// Player health: 90/100
	// Enemy position: (95, 95)
	// Enemy health: 60/100
}

func ExampleScheduler_Run() {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Transform](registry)
	ecs.RegisterComponent[Speed](registry)
	storage := ecs.NewStorage(registry)

	// Spawn a moving entity
	storage.Spawn(
		Transform{X: 0, Y: 0},
		Speed{DX: 1, DY: 1},
	)

	scheduler := ecs.NewScheduler(storage)
	scheduler.Register(&PhysicsSystem{})

	// Run continuously with context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Scheduler.Run blocks and executes systems at the specified interval
	// until the context is cancelled
	scheduler.Run(ctx, 16*time.Millisecond) // ~60 FPS

	fmt.Println("Scheduler stopped after context cancellation")
	// Output:
	// Scheduler stopped after context cancellation
}

// SpawnerSystem demonstrates using Commands to spawn entities
type SpawnerSystem struct {
	SpawnCount int
}

func (s *SpawnerSystem) Execute(frame *ecs.UpdateFrame) {
	if s.SpawnCount < 3 {
		// Use Commands to spawn entities during system execution
		// Commands are flushed at the end of the frame
		frame.Commands.Spawn(
			Transform{X: float32(s.SpawnCount * 10), Y: 0},
			Speed{DX: 1, DY: 1},
		)
		s.SpawnCount++
	}
}

func ExampleScheduler_commands() {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Transform](registry)
	ecs.RegisterComponent[Speed](registry)
	storage := ecs.NewStorage(registry)

	scheduler := ecs.NewScheduler(storage)
	spawner := &SpawnerSystem{}
	scheduler.Register(spawner)
	scheduler.Register(&PhysicsSystem{})

	// Run 3 frames, spawning one entity per frame
	for i := 0; i < 3; i++ {
		scheduler.Once(1.0)
	}

	// Count entities
	view := ecs.NewView[struct{ *Transform }](storage)
	count := 0
	for range view.Iter() {
		count++
	}

	fmt.Printf("Spawned %d entities using Commands\n", count)
	// Output:
	// Spawned 3 entities using Commands
}

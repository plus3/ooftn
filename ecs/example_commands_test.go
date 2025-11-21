package ecs_test

import (
	"fmt"

	"github.com/plus3/ooftn/ecs"
)

type CleanupSystem struct {
	Entities ecs.Query[struct {
		Id ecs.EntityId
		*Position
		*Health
	}]
}

func (s *CleanupSystem) Execute(frame *ecs.UpdateFrame) {
	deadCount := 0
	for item := range s.Entities.Iter() {
		if item.Health.Current <= 0 {
			frame.Commands.Delete(item.Id)
			deadCount++
		}
	}
	if deadCount > 0 {
		fmt.Printf("Queued %d dead entities for deletion\n", deadCount)
	}
}

// ExampleCommands demonstrates using command buffers to defer entity mutations.
// Commands are essential when modifying entities during iteration, as directly
// spawning or deleting entities while iterating can invalidate iterators and
// cause crashes. The Scheduler automatically flushes commands at the end of each
// frame, applying all deferred operations safely.
func ExampleCommands() {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[Velocity](registry)
	ecs.RegisterComponent[Health](registry)
	storage := ecs.NewStorage(registry)

	storage.Spawn(Position{X: 0, Y: 0}, Health{Current: 0, Max: 100})
	storage.Spawn(Position{X: 10, Y: 10}, Health{Current: 50, Max: 100})
	storage.Spawn(Position{X: 20, Y: 20}, Health{Current: 100, Max: 100})

	scheduler := ecs.NewScheduler(storage)
	scheduler.Register(&CleanupSystem{})

	scheduler.Once(1.0)

	view := ecs.NewView[struct{ *Position }](storage)
	remaining := 0
	for range view.Iter() {
		remaining++
	}
	fmt.Printf("Remaining entities: %d\n", remaining)

	// Output:
	// Queued 1 dead entities for deletion
	// Remaining entities: 2
}

type ShootTimer struct {
	TimeUntilShot float32
}

type ShootingSystem struct {
	Entities ecs.Query[struct {
		*Position
		*Velocity
		*ShootTimer
	}]
}

func (s *ShootingSystem) Execute(frame *ecs.UpdateFrame) {
	for item := range s.Entities.Iter() {
		if item.ShootTimer.TimeUntilShot <= 0 {
			frame.Commands.Spawn(
				Position{X: item.Position.X, Y: item.Position.Y},
				Velocity{DX: item.Velocity.DX * 2, DY: item.Velocity.DY * 2},
			)
			fmt.Printf("Spawned projectile at (%.0f, %.0f)\n", item.Position.X, item.Position.Y)
			item.ShootTimer.TimeUntilShot = 10
		}
	}
}

// ExampleCommands_spawning shows using commands to spawn entities during iteration.
// This is common for systems that need to create projectiles, particles, or other
// entities based on existing entity state. Commands ensure spawning happens after
// iteration completes, preventing iterator invalidation.
func ExampleCommands_spawning() {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[Velocity](registry)
	ecs.RegisterComponent[ShootTimer](registry)
	storage := ecs.NewStorage(registry)

	storage.Spawn(
		Position{X: 10, Y: 10},
		Velocity{DX: 1, DY: 0},
		ShootTimer{TimeUntilShot: 0},
	)
	storage.Spawn(
		Position{X: 20, Y: 20},
		Velocity{DX: 0, DY: 1},
		ShootTimer{TimeUntilShot: 5},
	)

	scheduler := ecs.NewScheduler(storage)
	scheduler.Register(&ShootingSystem{})

	scheduler.Once(1.0)

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)
	count := 0
	for range view.Iter() {
		count++
	}
	fmt.Printf("Total entities with velocity: %d\n", count)

	// Output:
	// Spawned projectile at (10, 10)
	// Total entities with velocity: 3
}

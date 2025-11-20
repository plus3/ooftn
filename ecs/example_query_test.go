package ecs_test

import (
	"fmt"

	"github.com/plus3/ooftn/ecs"
)

// ExampleQuery demonstrates using queries within systems for high-performance
// iteration. Queries cache matching archetypes and pre-build entity/component
// arrays each frame, making them significantly faster than Views for repeated
// iteration. The Scheduler automatically calls Execute() on queries before
// systems run, keeping the cache synchronized with the current frame's state.
func ExampleQuery() {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[Velocity](registry)
	ecs.RegisterComponent[Health](registry)
	storage := ecs.NewStorage(registry)

	storage.Spawn(Position{X: 0, Y: 0}, Velocity{DX: 1, DY: 0})
	storage.Spawn(Position{X: 10, Y: 10}, Velocity{DX: 0, DY: 1}, Health{Current: 100, Max: 100})
	storage.Spawn(Position{X: 20, Y: 20}, Velocity{DX: -1, DY: -1})

	query := ecs.NewQuery[struct {
		*Position
		*Velocity
	}](storage)

	query.Execute()

	type result struct {
		x, y, newX, newY float32
	}
	results := make([]result, 0)
	for _, item := range query.Iter() {
		newX := item.Position.X + item.Velocity.DX
		newY := item.Position.Y + item.Velocity.DY
		results = append(results, result{item.Position.X, item.Position.Y, newX, newY})
	}

	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].x > results[j].x {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	fmt.Println("Moving entities:")
	for _, r := range results {
		fmt.Printf("Position (%.0f, %.0f) -> (%.0f, %.0f)\n", r.x, r.y, r.newX, r.newY)
	}

	// Output:
	// Moving entities:
	// Position (0, 0) -> (1, 0)
	// Position (10, 10) -> (10, 11)
	// Position (20, 20) -> (19, 19)
}

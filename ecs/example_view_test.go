package ecs_test

import (
	"fmt"

	"github.com/plus3/ooftn/ecs"
)

// ExampleView demonstrates using Views for flexible entity queries and spawning.
// Views provide a way to query entities with specific component combinations and
// optionally spawn new entities. Unlike Queries, Views don't require a Scheduler
// and perform iteration on-demand, making them ideal for one-off queries, tools,
// or situations where you need to query entities outside of a system.
func ExampleView() {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[Velocity](registry)
	ecs.RegisterComponent[Health](registry)
	storage := ecs.NewStorage(registry)

	player := storage.Spawn(
		Position{X: 10, Y: 20},
		Velocity{DX: 1, DY: 0},
		Health{Current: 100, Max: 100},
	)

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	if item := view.Get(player); item != nil {
		fmt.Printf("Player at (%.0f, %.0f) moving (%.0f, %.0f)\n",
			item.Position.X, item.Position.Y, item.Velocity.DX, item.Velocity.DY)
	}

	// Output:
	// Player at (10, 20) moving (1, 0)
}

// ExampleView_Iter shows iterating over all entities matching a view.
// Views automatically match entities across all archetypes that contain
// the required components, making it easy to process entities without
// worrying about their specific archetype layout.
//
// By including an EntityId field in the view struct, you can access each
// entity's ID during iteration. This is useful for storing references,
// deleting entities, or performing operations that require the entity ID.
func ExampleView_Iter() {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[Velocity](registry)
	ecs.RegisterComponent[Health](registry)
	storage := ecs.NewStorage(registry)

	storage.Spawn(Position{X: 0, Y: 0}, Velocity{DX: 1, DY: 0})
	storage.Spawn(Position{X: 10, Y: 10}, Velocity{DX: 0, DY: 1}, Health{Current: 50, Max: 100})
	storage.Spawn(Position{X: 20, Y: 20}, Velocity{DX: -1, DY: -1})
	storage.Spawn(Position{X: 100, Y: 100})

	view := ecs.NewView[struct {
		Id ecs.EntityId
		*Position
		*Velocity
	}](storage)

	type result struct {
		x, y float32
	}
	results := make([]result, 0)
	entityIds := make([]ecs.EntityId, 0)
	for item := range view.Iter() {
		item.Position.X += item.Velocity.DX
		item.Position.Y += item.Velocity.DY
		results = append(results, result{item.Position.X, item.Position.Y})
		entityIds = append(entityIds, item.Id)
	}

	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].x > results[j].x {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	fmt.Println("Entities with position and velocity:")
	for _, r := range results {
		fmt.Printf("New position: (%.0f, %.0f)\n", r.x, r.y)
	}
	fmt.Printf("Total entities with IDs: %d\n", len(entityIds))

	// Output:
	// Entities with position and velocity:
	// New position: (1, 0)
	// New position: (10, 11)
	// New position: (19, 19)
	// Total entities with IDs: 3
}

// ExampleView_optional demonstrates using optional components in views.
// Optional components allow a single view to match entities that may or may not
// have certain components. This is useful for systems that need to handle both
// cases, like rendering entities with optional visual effects or processing
// entities with optional AI components.
func ExampleView_optional() {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[Velocity](registry)
	ecs.RegisterComponent[Health](registry)
	storage := ecs.NewStorage(registry)

	storage.Spawn(Position{X: 10, Y: 10}, Velocity{DX: 1, DY: 0}, Health{Current: 50, Max: 100})
	storage.Spawn(Position{X: 20, Y: 20}, Velocity{DX: 0, DY: 1}, Health{Current: 75, Max: 100})
	storage.Spawn(Position{X: 30, Y: 30}, Velocity{DX: -1, DY: 0})

	view := ecs.NewView[struct {
		Position *Position
		Velocity *Velocity
		Health   *Health `ecs:"optional"`
	}](storage)

	fmt.Println("All moving entities:")
	for item := range view.Iter() {
		if item.Health != nil {
			fmt.Printf("Entity at (%.0f, %.0f) with health %d/%d\n",
				item.Position.X, item.Position.Y, item.Health.Current, item.Health.Max)
		} else {
			fmt.Printf("Invulnerable entity at (%.0f, %.0f)\n", item.Position.X, item.Position.Y)
		}
	}

	// Output:
	// All moving entities:
	// Entity at (10, 10) with health 50/100
	// Entity at (20, 20) with health 75/100
	// Invulnerable entity at (30, 30)
}

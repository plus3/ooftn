package ecs_test

import (
	"fmt"
	"reflect"

	"github.com/plus3/ooftn/ecs"
)

// ExampleStorage demonstrates the basic API for managing entities and components.
// Storage is the core container for all entities and their component data.
// Components are organized by archetype - entities with the same component types
// share the same archetype for efficient memory layout and iteration.
func ExampleStorage() {
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

	pos := ecs.ReadComponent[Position](storage, player)
	fmt.Printf("Player spawned at (%.0f, %.0f)\n", pos.X, pos.Y)

	pos.X = 15
	pos.Y = 25
	fmt.Printf("Player moved to (%.0f, %.0f)\n", pos.X, pos.Y)

	storage.Delete(player)
	fmt.Println("Player deleted")

	// Output:
	// Player spawned at (10, 20)
	// Player moved to (15, 25)
	// Player deleted
}

// ExampleStorage_addRemoveComponents shows how entity archetypes change
// when components are added or removed. When an entity's components change,
// it moves to a different archetype that matches its new component set.
func ExampleStorage_addRemoveComponents() {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[Velocity](registry)
	ecs.RegisterComponent[Health](registry)
	storage := ecs.NewStorage(registry)

	entity := storage.Spawn(Position{X: 0, Y: 0})

	hasVel := storage.HasComponent(entity, reflect.TypeOf(Velocity{}))
	fmt.Printf("Has velocity: %v\n", hasVel)

	entity = storage.AddComponent(entity, Velocity{DX: 5, DY: 3})
	vel := ecs.ReadComponent[Velocity](storage, entity)
	fmt.Printf("Has velocity: %v (%.0f, %.0f)\n", vel != nil, vel.DX, vel.DY)

	entity = storage.AddComponent(entity, Health{Current: 50, Max: 50})
	health := ecs.ReadComponent[Health](storage, entity)
	fmt.Printf("Has health: %v (%d/%d)\n", health != nil, health.Current, health.Max)

	entity = storage.RemoveComponent(entity, reflect.TypeOf(Velocity{}))
	hasVel = storage.HasComponent(entity, reflect.TypeOf(Velocity{}))
	fmt.Printf("Has velocity: %v\n", hasVel)

	// Output:
	// Has velocity: false
	// Has velocity: true (5, 3)
	// Has health: true (50/50)
	// Has velocity: false
}

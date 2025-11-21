# ECS

ECS is a Entity Component System implementation in Go, designed for usability and performance.

## Features

- **Modern API** designed around modern Go features like generics, iterators and weak-refs
- **Good Performance** for the vast majority of use cases (see [cmd/ecs-stress](../cmd/ecs-stress))
- **Simple** to use and understand, no code generation means you can easily understand the source

## Example

```go
package main

import (
	"fmt"
	"reflect"

	"github.com/plus3/ooftn/ecs"
)

// Components can be a combination of named types and structs
type Gravity float32

type Name string

type Position struct {
	X, Y float32
}

type Velocity struct {
	DX, DY float32
}

type Health struct {
	Current int
	Max     int
}

// Systems encapsulate behavior and have easy access to querying components
type GravitySystem struct {
	Gravitating ecs.Query[struct {
		ecs.EntityId
		*Position
		*Velocity
	}]
	Gravity ecs.Singleton[Gravity]
}

func (g *GravitySystem) Execute(frame *ecs.UpdateFrame) {
	// Singletons are guarenteed to exist so we can just fetch it
	gravity := *g.Gravity.Get()

	// Queries are a nice wrapper over views for systems
	for entity := range g.Gravitating.Iter() {
		entity.Position.Y -= float32(gravity)

		if entity.Position.Y < 0 {
			// We can use the command buffer to defer mutations until the end of the frame
			frame.Commands.Delete(entity.EntityId)
		}
	}
}

func main() {
	// We register all component types, this helps a lot with the internal performance of the library
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[Velocity](registry)
	ecs.RegisterComponent[Health](registry)
	ecs.RegisterComponent[Name](registry)
	ecs.RegisterComponent[float64](registry)

	// The storage is where our component data actually lives
	storage := ecs.NewStorage(registry)

	// Finally we can directly spawn an entity using components
	playerId := storage.Spawn(
		Position{X: 10, Y: 20},
		Velocity{DX: 1, DY: 0},
		Name("joe"),
	)
	storage.Spawn(Position{X: 100, Y: 100}, Name("rock"))

	// Reading data via an EntityId is **very** fast using the generics API
	position := ecs.ReadComponent[Position](storage, playerId)

	// We can also read data using reflect.Type's
	velocity := storage.GetComponent(playerId, reflect.TypeFor[Velocity]()).(*Velocity)

	// Components we read are pointers to the underlying data storage, meaning we can easily mutate fields
	position.X += velocity.DX
	position.Y += velocity.DY

	// An EntityId provides extremely fast access to data, but it is not guarenteed to remain stable.
	// Lets mutate the underlying "archetype" of the entity by adjusting which components it has, this
	// will cause the EntityId to change.
	playerId = storage.AddComponent(playerId, Health{Current: 5, Max: 10})

	// Since we can't trust EntityId's to remain stable, we instead use EntityRef's to keep track of
	// entities and reference them across components.
	playerRef := storage.CreateEntityRef(playerId)

	// Add a nonsense component to force the underlying archetype and entityid to change
	newPlayerId := storage.AddComponent(playerId, float64(32))

	// PlayerRef remains updated
	fmt.Printf("%d == %d (old: %d)\n", playerRef.Id, newPlayerId, playerId)

	// When dealing with groups of components, View's give us cachable performance benefits and a
	// clean generics-based API.
	named := ecs.NewView[struct {
		*Position
		*Name

		// Views can select "optional" fields
		Health *Health `ecs:"optional"`
	}](storage)

	// Views can be used to read and mutate individual entity data easily
	playerData := named.Get(newPlayerId)
	playerData.Position.X += 1
	playerData.Position.Y += 1

	// We can also easily use references
	playerData = named.GetRef(playerRef)

	// Views can also be used to "query" multiple entities
	for entity := range named.Iter() {
		fmt.Printf("Entity Name: %s\n", *entity.Name)

		// Optional fields must be nil-checked before use
		if entity.Health != nil {
			fmt.Printf("  health: %d\n", entity.Health.Current)
		}
	}

	// Singletons allow us to store global-state components easily
	ecs.NewSingleton[Gravity](storage, Gravity(8.0))

	// We can read singletons directly from storage
	var gravity *Gravity
	if storage.ReadSingleton(&gravity) {
		fmt.Printf("Gravity is %f\n", *gravity)
	}

	// Finally we can use a scheduler to execute systems
	scheduler := ecs.NewScheduler(storage)
	scheduler.Register(&GravitySystem{})
	scheduler.Once(1)
}
```

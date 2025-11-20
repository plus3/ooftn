package ecs_test

import (
	"fmt"

	"github.com/plus3/ooftn/ecs"
)

// ExampleEntityRef demonstrates using EntityRefs to maintain stable references
// to entities. Unlike EntityIds which can become invalid when entities move
// between archetypes or are deleted, EntityRefs remain valid and automatically
// track entities as they move. This makes them ideal for storing relationships
// between entities in components.
func ExampleEntityRef() {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[Velocity](registry)
	storage := ecs.NewStorage(registry)

	target := storage.Spawn(Position{X: 100, Y: 100})
	targetRef := storage.CreateEntityRef(target)

	targetId, ok := storage.ResolveEntityRef(targetRef)
	if ok {
		targetPos := ecs.ReadComponent[Position](storage, targetId)
		fmt.Printf("Target at (%.0f, %.0f)\n", targetPos.X, targetPos.Y)
	}

	target = storage.AddComponent(target, Velocity{DX: 0, DY: 0})

	targetId, ok = storage.ResolveEntityRef(targetRef)
	if ok {
		targetPos := ecs.ReadComponent[Position](storage, targetId)
		fmt.Printf("Target moved archetypes, still at (%.0f, %.0f)\n", targetPos.X, targetPos.Y)
	}

	storage.Delete(target)
	_, ok = storage.ResolveEntityRef(targetRef)
	fmt.Printf("Target deleted, ref valid: %v\n", ok)

	// Output:
	// Target at (100, 100)
	// Target moved archetypes, still at (100, 100)
	// Target deleted, ref valid: false
}

// ExampleEntityRef_relationshipComponent shows using EntityRefs within components
// to create relationships between entities. This pattern is common for AI targets,
// parent-child hierarchies, or any other entity-to-entity relationships that need
// to survive archetype changes and entity deletions.
func ExampleEntityRef_relationshipComponent() {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[FollowerAI](registry)
	storage := ecs.NewStorage(registry)

	leader := storage.Spawn(Position{X: 50, Y: 50})
	leaderRef := storage.CreateEntityRef(leader)

	storage.Spawn(
		Position{X: 40, Y: 40},
		FollowerAI{Target: leaderRef},
	)
	storage.Spawn(
		Position{X: 60, Y: 40},
		FollowerAI{Target: leaderRef},
	)

	fmt.Println("Followers tracking leader:")
	view := ecs.NewView[struct {
		*Position
		*FollowerAI
	}](storage)

	for _, item := range view.Iter() {
		if targetId, ok := storage.ResolveEntityRef(item.FollowerAI.Target); ok {
			targetPos := ecs.ReadComponent[Position](storage, targetId)
			fmt.Printf("Follower at (%.0f, %.0f) following target at (%.0f, %.0f)\n",
				item.Position.X, item.Position.Y, targetPos.X, targetPos.Y)
		}
	}

	storage.Delete(leader)

	fmt.Println("\nAfter leader deleted:")
	for _, item := range view.Iter() {
		if _, ok := storage.ResolveEntityRef(item.FollowerAI.Target); !ok {
			fmt.Printf("Follower at (%.0f, %.0f) lost its target\n",
				item.Position.X, item.Position.Y)
		}
	}

	// Output:
	// Followers tracking leader:
	// Follower at (40, 40) following target at (50, 50)
	// Follower at (60, 40) following target at (50, 50)
	//
	// After leader deleted:
	// Follower at (40, 40) lost its target
	// Follower at (60, 40) lost its target
}

type FollowerAI struct {
	Target *ecs.EntityRef
}

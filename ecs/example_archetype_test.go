package ecs_test

import (
	"fmt"

	"github.com/plus3/ooftn/ecs"
)

// ExampleArchetype_Compact demonstrates archetype compaction to reclaim memory.
// When entities are deleted, they can leave gaps in the archetype's component arrays.
// Compaction moves entities to fill these gaps, improving memory locality and
// iteration performance. Entity IDs are updated during compaction, but EntityRefs
// remain valid and automatically track the new locations.
func ExampleArchetype_Compact() {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[Health](registry)
	storage := ecs.NewStorage(registry)

	entities := make([]ecs.EntityId, 5)
	for i := range 5 {
		entities[i] = storage.Spawn(
			Position{X: float32(i * 10), Y: 0},
			Health{Current: 100, Max: 100},
		)
	}

	storage.Delete(entities[1])
	storage.Delete(entities[3])

	view := ecs.NewView[struct {
		*Position
		*Health
	}](storage)

	fmt.Println("Before compaction:")
	count := 0
	for range view.Iter() {
		count++
	}
	fmt.Printf("Entities: %d\n", count)

	archetype := storage.GetArchetype(entities[0].ArchetypeId())
	if archetype != nil {
		archetype.Compact()
	}

	fmt.Println("\nAfter compaction:")
	count = 0
	for _, item := range view.Iter() {
		fmt.Printf("Position: (%.0f, %.0f)\n", item.Position.X, item.Position.Y)
		count++
	}
	fmt.Printf("Entities: %d\n", count)

	// Output:
	// Before compaction:
	// Entities: 3
	//
	// After compaction:
	// Position: (0, 0)
	// Position: (20, 0)
	// Position: (40, 0)
	// Entities: 3
}

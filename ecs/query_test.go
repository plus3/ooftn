package ecs_test

import (
	"testing"

	"github.com/plus3/ooftn/ecs"
)

func setupQueryTest() (*ecs.Storage, *ecs.Query[struct {
	*Position
	*Velocity
}]) {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[Velocity](registry)
	ecs.RegisterComponent[Health](registry)

	storage := ecs.NewStorage(registry)

	storage.Spawn(Position{X: 1, Y: 2}, Velocity{DX: 0.5, DY: 0.5})
	storage.Spawn(Position{X: 3, Y: 4}, Velocity{DX: 1.0, DY: 1.0})
	storage.Spawn(Position{X: 5, Y: 6}, Velocity{DX: 1.5, DY: 1.5}, Health{Current: 100, Max: 100})
	storage.Spawn(Position{X: 7, Y: 8})

	query := ecs.NewQuery[struct {
		*Position
		*Velocity
	}](storage)

	return storage, query
}

func TestQuery(t *testing.T) {
	t.Run("iterates correct number of entities", func(t *testing.T) {
		_, query := setupQueryTest()
		count := 0
		for range query.Iter() {
			count++
		}

		if count != 3 {
			t.Errorf("expected 3 entities, got %d", count)
		}
	})

	t.Run("multiple iterations are consistent", func(t *testing.T) {
		_, query := setupQueryTest()
		results1 := make(map[ecs.EntityId]bool)
		for id := range query.Iter() {
			results1[id] = true
		}

		results2 := make(map[ecs.EntityId]bool)
		for id := range query.Iter() {
			results2[id] = true
		}

		if len(results1) != len(results2) {
			t.Error("multiple iterations should return same results")
		}

		for id := range results1 {
			if !results2[id] {
				t.Error("multiple iterations should be consistent")
			}
		}
	})

	t.Run("finds newly spawned entities", func(t *testing.T) {
		storage, query := setupQueryTest()
		initialCount := 0
		for range query.Iter() {
			initialCount++
		}

		storage.Spawn(Position{X: 10, Y: 10}, Velocity{DX: 2.0, DY: 2.0})

		afterSpawnCount := 0
		for range query.Iter() {
			afterSpawnCount++
		}

		if afterSpawnCount != initialCount+1 {
			t.Errorf("expected %d entities after spawn, got %d", initialCount+1, afterSpawnCount)
		}
	})

	t.Run("iter values", func(t *testing.T) {
		_, query := setupQueryTest()
		count := 0
		for item := range query.Values() {
			if item.Position == nil || item.Velocity == nil {
				t.Error("expected non-nil components")
			}
			count++
		}

		if count != 3 {
			t.Errorf("expected 3 entities, got %d", count)
		}
	})
}

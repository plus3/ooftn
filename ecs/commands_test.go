package ecs_test

import (
	"reflect"
	"testing"

	"github.com/plus3/ooftn/ecs"
)

type testSpawnSystem struct {
	executed bool
}

func (s *testSpawnSystem) Execute(frame *ecs.UpdateFrame) {
	s.executed = true
	frame.Commands.Spawn(Position{X: 1, Y: 2}, Velocity{DX: 0.5, DY: 0.5})
	frame.Commands.Spawn(Position{X: 3, Y: 4})
}

type testDeleteSystem struct {
	entityToDelete ecs.EntityId
}

func (s *testDeleteSystem) Execute(frame *ecs.UpdateFrame) {
	frame.Commands.Delete(s.entityToDelete)
}

type testAddSystem struct {
	entity ecs.EntityId
}

func (s *testAddSystem) Execute(frame *ecs.UpdateFrame) {
	frame.Commands.AddComponent(s.entity, Velocity{DX: 5, DY: 10})
}

type testRemoveSystem struct {
	entity ecs.EntityId
}

func (s *testRemoveSystem) Execute(frame *ecs.UpdateFrame) {
	frame.Commands.RemoveComponent(s.entity, reflect.TypeOf(Velocity{}))
}

type testMixedSystem struct {
	entity ecs.EntityId
}

func (s *testMixedSystem) Execute(frame *ecs.UpdateFrame) {
	frame.Commands.Spawn(Position{X: 10, Y: 20})
	frame.Commands.AddComponent(s.entity, Velocity{DX: 1, DY: 1})
	frame.Commands.Delete(s.entity)
	frame.Commands.Spawn(Health{Current: 100, Max: 100})
}

// Systems for cross-system entity mutation tests
type systemRemoveVelocity struct {
	entity ecs.EntityId
}

func (s *systemRemoveVelocity) Execute(frame *ecs.UpdateFrame) {
	frame.Commands.RemoveComponent(s.entity, reflect.TypeOf(Velocity{}))
}

type systemAddHealth struct {
	entity ecs.EntityId
}

func (s *systemAddHealth) Execute(frame *ecs.UpdateFrame) {
	frame.Commands.AddComponent(s.entity, Health{Current: 50, Max: 100})
}

type systemAddVelocity struct {
	entity ecs.EntityId
}

func (s *systemAddVelocity) Execute(frame *ecs.UpdateFrame) {
	frame.Commands.AddComponent(s.entity, Velocity{DX: 1, DY: 2})
}

type systemRemoveHealth struct {
	entity ecs.EntityId
}

func (s *systemRemoveHealth) Execute(frame *ecs.UpdateFrame) {
	frame.Commands.RemoveComponent(s.entity, reflect.TypeOf(Health{}))
}

func TestCommands(t *testing.T) {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[Velocity](registry)
	ecs.RegisterComponent[Health](registry)

	t.Run("spawn entities", func(t *testing.T) {
		storage := ecs.NewStorage(registry)
		scheduler := ecs.NewScheduler(storage)

		system := &testSpawnSystem{}
		scheduler.Register(system)

		view := ecs.NewView[struct{ *Position }](storage)
		count := 0
		for range view.Iter() {
			count++
		}
		if count != 0 {
			t.Error("entities spawned before frame execution")
		}

		scheduler.Once(1.0)

		count = 0
		for range view.Iter() {
			count++
		}
		if count != 2 {
			t.Errorf("expected 2 entities after frame, got %d", count)
		}

		if !system.executed {
			t.Error("system was not executed")
		}
	})

	t.Run("delete entities", func(t *testing.T) {
		storage := ecs.NewStorage(registry)
		e1 := storage.Spawn(Position{X: 1, Y: 2})
		e2 := storage.Spawn(Position{X: 3, Y: 4})

		scheduler := ecs.NewScheduler(storage)
		scheduler.Register(&testDeleteSystem{entityToDelete: e1})

		if storage.GetComponent(e1, reflect.TypeOf(Position{})) == nil {
			t.Error("entity deleted before frame execution")
		}

		scheduler.Once(1.0)

		if storage.GetComponent(e1, reflect.TypeOf(Position{})) != nil {
			t.Error("entity not deleted after frame")
		}
		if storage.GetComponent(e2, reflect.TypeOf(Position{})) == nil {
			t.Error("wrong entity deleted")
		}
	})

	t.Run("add components", func(t *testing.T) {
		storage := ecs.NewStorage(registry)
		entity := storage.Spawn(Position{X: 1, Y: 2})

		scheduler := ecs.NewScheduler(storage)
		scheduler.Register(&testAddSystem{entity: entity})

		scheduler.Once(1.0)

		view := ecs.NewView[struct {
			*Position
			*Velocity
		}](storage)

		found := false
		for item := range view.Iter() {
			if item.Position.X == 1 && item.Position.Y == 2 {
				if item.Velocity.DX == 5 && item.Velocity.DY == 10 {
					found = true
				}
			}
		}

		if !found {
			t.Error("component not added after frame or values incorrect")
		}
	})

	t.Run("remove components", func(t *testing.T) {
		storage := ecs.NewStorage(registry)
		entity := storage.Spawn(Position{X: 1, Y: 2}, Velocity{DX: 5, DY: 10})

		scheduler := ecs.NewScheduler(storage)
		scheduler.Register(&testRemoveSystem{entity: entity})

		scheduler.Once(1.0)

		viewWithVelocity := ecs.NewView[struct {
			*Position
			*Velocity
		}](storage)

		countWithVelocity := 0
		for range viewWithVelocity.Iter() {
			countWithVelocity++
		}

		viewWithoutVelocity := ecs.NewView[struct{ *Position }](storage)
		countWithoutVelocity := 0
		foundCorrectEntity := false
		for item := range viewWithoutVelocity.Iter() {
			countWithoutVelocity++
			if item.Position.X == 1 && item.Position.Y == 2 {
				foundCorrectEntity = true
			}
		}

		if countWithVelocity != 0 {
			t.Error("velocity component not removed")
		}
		if !foundCorrectEntity {
			t.Error("entity with only position not found")
		}
	})

	t.Run("mixed operations", func(t *testing.T) {
		storage := ecs.NewStorage(registry)
		e1 := storage.Spawn(Position{X: 1, Y: 2})

		scheduler := ecs.NewScheduler(storage)
		scheduler.Register(&testMixedSystem{entity: e1})
		scheduler.Once(1.0)

		view := ecs.NewView[struct{ *Position }](storage)
		count := 0
		for range view.Iter() {
			count++
		}
		if count != 1 {
			t.Errorf("expected 1 position entity, got %d", count)
		}
	})

	// Cross-system entity mutation tests - verify entity ID tracking during flush
	t.Run("cross-system remove then add same entity", func(t *testing.T) {
		storage := ecs.NewStorage(registry)
		// Entity starts with Position + Velocity
		entity := storage.Spawn(Position{X: 1, Y: 2}, Velocity{DX: 5, DY: 10})

		scheduler := ecs.NewScheduler(storage)
		// System A removes Velocity, System B adds Health
		scheduler.Register(&systemRemoveVelocity{entity: entity})
		scheduler.Register(&systemAddHealth{entity: entity})
		scheduler.Once(1.0)

		// Entity should have Position + Health (no Velocity)
		viewWithHealth := ecs.NewView[struct {
			*Position
			*Health
		}](storage)
		found := false
		for item := range viewWithHealth.Iter() {
			if item.Position.X == 1 && item.Position.Y == 2 {
				if item.Health.Current == 50 && item.Health.Max == 100 {
					found = true
				}
			}
		}
		if !found {
			t.Error("entity should have Position + Health after cross-system mutations")
		}

		// Verify Velocity was removed
		viewWithVelocity := ecs.NewView[struct {
			*Position
			*Velocity
		}](storage)
		for range viewWithVelocity.Iter() {
			t.Error("entity should not have Velocity after RemoveComponent")
		}
	})

	t.Run("cross-system multiple adds same entity", func(t *testing.T) {
		storage := ecs.NewStorage(registry)
		// Entity starts with only Position
		entity := storage.Spawn(Position{X: 3, Y: 4})

		scheduler := ecs.NewScheduler(storage)
		// System A adds Velocity, System B adds Health
		scheduler.Register(&systemAddVelocity{entity: entity})
		scheduler.Register(&systemAddHealth{entity: entity})
		scheduler.Once(1.0)

		// Entity should have Position + Velocity + Health
		viewAll := ecs.NewView[struct {
			*Position
			*Velocity
			*Health
		}](storage)
		found := false
		for item := range viewAll.Iter() {
			if item.Position.X == 3 && item.Position.Y == 4 {
				if item.Velocity.DX == 1 && item.Velocity.DY == 2 {
					if item.Health.Current == 50 && item.Health.Max == 100 {
						found = true
					}
				}
			}
		}
		if !found {
			t.Error("entity should have all three components after cross-system adds")
		}
	})

	t.Run("cross-system chained mutations same entity", func(t *testing.T) {
		storage := ecs.NewStorage(registry)
		// Entity starts with Position + Velocity + Health
		entity := storage.Spawn(Position{X: 5, Y: 6}, Velocity{DX: 1, DY: 1}, Health{Current: 100, Max: 100})

		scheduler := ecs.NewScheduler(storage)
		// System A removes Velocity, System B removes Health
		// This tests that two removes on the same entity both work correctly
		// (entity moves archetype twice: Pos+Vel+Health -> Pos+Health -> Pos)
		scheduler.Register(&systemRemoveVelocity{entity: entity})
		scheduler.Register(&systemRemoveHealth{entity: entity})
		scheduler.Once(1.0)

		// Entity should have only Position
		viewOnlyPosition := ecs.NewView[struct{ *Position }](storage)
		foundPosition := false
		for item := range viewOnlyPosition.Iter() {
			if item.Position.X == 5 && item.Position.Y == 6 {
				foundPosition = true
			}
		}
		if !foundPosition {
			t.Error("entity should still have Position after chained mutations")
		}

		// Verify no Velocity
		viewWithVelocity := ecs.NewView[struct {
			*Position
			*Velocity
		}](storage)
		for range viewWithVelocity.Iter() {
			t.Error("entity should not have Velocity")
		}

		// Verify no Health
		viewWithHealth := ecs.NewView[struct {
			*Position
			*Health
		}](storage)
		for range viewWithHealth.Iter() {
			t.Error("entity should not have Health")
		}
	})

	t.Run("cross-system mutation after delete is ignored", func(t *testing.T) {
		storage := ecs.NewStorage(registry)
		entity := storage.Spawn(Position{X: 7, Y: 8})

		scheduler := ecs.NewScheduler(storage)
		// System A deletes entity, System B tries to add Health
		scheduler.Register(&testDeleteSystem{entityToDelete: entity})
		scheduler.Register(&systemAddHealth{entity: entity})
		scheduler.Once(1.0)

		// Entity should be deleted, no crash should occur
		view := ecs.NewView[struct{ *Position }](storage)
		for item := range view.Iter() {
			if item.Position.X == 7 && item.Position.Y == 8 {
				t.Error("entity should have been deleted")
			}
		}

		// Also verify no Health entities were created
		viewHealth := ecs.NewView[struct{ *Health }](storage)
		count := 0
		for range viewHealth.Iter() {
			count++
		}
		if count > 0 {
			t.Error("no Health-only entities should exist")
		}
	})
}

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
}

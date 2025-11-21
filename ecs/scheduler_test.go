package ecs_test

import (
	"context"
	"testing"
	"time"

	"github.com/plus3/ooftn/ecs"
)

type MovementSystem struct {
	Entities ecs.Query[struct {
		*Position
		*Velocity
	}]
	ExecuteCount int
}

func (s *MovementSystem) Execute(frame *ecs.UpdateFrame) {
	s.ExecuteCount++
	for item := range s.Entities.Iter() {
		item.Position.X += item.Velocity.DX * float32(frame.DeltaTime)
		item.Position.Y += item.Velocity.DY * float32(frame.DeltaTime)
	}
}

type HealthSystem struct {
	Entities ecs.Query[struct {
		*Health
	}]
	ExecuteCount int
	TotalHealth  float64
}

func (s *HealthSystem) Execute(frame *ecs.UpdateFrame) {
	s.ExecuteCount++
	s.TotalHealth = 0
	for item := range s.Entities.Iter() {
		s.TotalHealth += float64(item.Health.Current)
	}
}

func TestScheduler(t *testing.T) {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[Velocity](registry)
	ecs.RegisterComponent[Health](registry)

	t.Run("system execution order and query initialization", func(t *testing.T) {
		storage := ecs.NewStorage(registry)
		scheduler := ecs.NewScheduler(storage)

		movement := &MovementSystem{}
		health := &HealthSystem{}

		scheduler.Register(movement)
		scheduler.Register(health)

		storage.Spawn(Position{X: 0, Y: 0}, Velocity{DX: 1, DY: 2})
		storage.Spawn(Health{Current: 100, Max: 100})

		scheduler.Once(1.0)

		if movement.ExecuteCount != 1 {
			t.Errorf("expected MovementSystem to execute once, got %d", movement.ExecuteCount)
		}

		if health.ExecuteCount != 1 {
			t.Errorf("expected HealthSystem to execute once, got %d", health.ExecuteCount)
		}

		scheduler.Once(1.0)

		if movement.ExecuteCount != 2 {
			t.Errorf("expected MovementSystem to execute twice, got %d", movement.ExecuteCount)
		}

		if health.ExecuteCount != 2 {
			t.Errorf("expected HealthSystem to execute twice, got %d", health.ExecuteCount)
		}
	})

	t.Run("custom state persistence", func(t *testing.T) {
		storage := ecs.NewStorage(registry)
		scheduler := ecs.NewScheduler(storage)

		storage.Spawn(Health{Current: 50, Max: 100})
		storage.Spawn(Health{Current: 75, Max: 100})

		health := &HealthSystem{}
		scheduler.Register(health)

		scheduler.Once(1.0)

		if health.TotalHealth != 125.0 {
			t.Errorf("expected TotalHealth=125.0, got %f", health.TotalHealth)
		}

		storage.Spawn(Health{Current: 25, Max: 100})

		scheduler.Once(1.0)

		if health.TotalHealth != 150.0 {
			t.Errorf("expected TotalHealth=150.0, got %f", health.TotalHealth)
		}
	})

	t.Run("context cancellation in run", func(t *testing.T) {
		storage := ecs.NewStorage(registry)
		scheduler := ecs.NewScheduler(storage)

		movement := &MovementSystem{}
		scheduler.Register(movement)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan bool)
		go func() {
			scheduler.Run(ctx, 1*time.Millisecond)
			done <- true
		}()

		time.Sleep(10 * time.Millisecond)
		cancel()

		select {
		case <-done:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("scheduler did not stop after context cancellation")
		}

		if movement.ExecuteCount == 0 {
			t.Error("expected system to execute at least once")
		}
	})

	t.Run("delta time calculation", func(t *testing.T) {
		storage := ecs.NewStorage(registry)
		scheduler := ecs.NewScheduler(storage)

		storage.Spawn(Position{X: 0, Y: 0}, Velocity{DX: 10, DY: 20})

		movement := &MovementSystem{}
		scheduler.Register(movement)

		scheduler.Once(0.5)

		found := false
		for item := range movement.Entities.Iter() {
			if item.Position.X == 5.0 && item.Position.Y == 10.0 {
				found = true
			}
		}

		if !found {
			t.Error("expected position to be updated with delta time")
		}
	})

	t.Run("commands integration", func(t *testing.T) {
		storage := ecs.NewStorage(registry)
		scheduler := ecs.NewScheduler(storage)

		spawnSystem := &testSpawnSystem{}
		scheduler.Register(spawnSystem)

		scheduler.Once(1.0)

		if !spawnSystem.executed {
			t.Error("expected spawn system to execute")
		}

		movement := &MovementSystem{}
		scheduler.Register(movement)
		scheduler.Once(1.0)

		count := 0
		for range movement.Entities.Iter() {
			count++
		}

		if count == 0 {
			t.Error("expected spawned entity to be visible after command flush")
		}
	})
}

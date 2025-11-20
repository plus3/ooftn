package ecs_test

import (
	"reflect"
	"testing"

	"github.com/plus3/ooftn/ecs"
)

func BenchmarkSpawn(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.Spawn(Position{X: 1.0, Y: 2.0}, Velocity{DX: 0.5, DY: 0.5})
	}
}

func BenchmarkSpawnWithMultipleComponents(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.Spawn(
			Position{X: 1.0, Y: 2.0},
			Velocity{DX: 0.5, DY: 0.5},
			Health{Current: 100, Max: 100},
			Name{Value: "Entity"},
		)
	}
}

func BenchmarkDelete(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	ids := make([]ecs.EntityId, b.N)
	for i := 0; i < b.N; i++ {
		ids[i] = storage.Spawn(Position{X: 1.0, Y: 2.0}, Velocity{DX: 0.5, DY: 0.5})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.Delete(ids[i])
	}
}

func BenchmarkGetComponent(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	id := storage.Spawn(Position{X: 1.0, Y: 2.0}, Velocity{DX: 0.5, DY: 0.5})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ecs.ReadComponent[Position](storage, id)
	}
}

func BenchmarkAddComponent(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	ids := make([]ecs.EntityId, b.N)
	for i := 0; i < b.N; i++ {
		ids[i] = storage.Spawn(Position{X: 1.0, Y: 2.0})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.AddComponent(ids[i], Velocity{DX: 0.5, DY: 0.5})
	}
}

func BenchmarkRemoveComponent(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	ids := make([]ecs.EntityId, b.N)
	for i := 0; i < b.N; i++ {
		ids[i] = storage.Spawn(Position{X: 1.0, Y: 2.0}, Velocity{DX: 0.5, DY: 0.5})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.RemoveComponent(ids[i], reflect.TypeOf(Velocity{}))
	}
}

func BenchmarkEntityRef(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	id := storage.Spawn(Position{X: 1.0, Y: 2.0})
	ref := storage.CreateEntityRef(id)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.ResolveEntityRef(ref)
	}
}

func BenchmarkViewFill(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	type PosVel struct {
		*Position
		*Velocity
	}

	view := ecs.NewView[PosVel](storage)
	id := storage.Spawn(Position{X: 1.0, Y: 2.0}, Velocity{DX: 0.5, DY: 0.5})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var pv PosVel
		view.Fill(id, &pv)
	}
}

func BenchmarkViewGet(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	type PosVel struct {
		*Position
		*Velocity
	}

	view := ecs.NewView[PosVel](storage)
	id := storage.Spawn(Position{X: 1.0, Y: 2.0}, Velocity{DX: 0.5, DY: 0.5})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = view.Get(id)
	}
}

func BenchmarkViewIter(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	type PosVel struct {
		*Position
		*Velocity
	}

	for i := 0; i < 1000; i++ {
		storage.Spawn(Position{X: float32(i), Y: float32(i)}, Velocity{DX: 0.5, DY: 0.5})
	}

	view := ecs.NewView[PosVel](storage)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pv := range view.Iter() {
			_ = pv
		}
	}
}

func BenchmarkViewIterLarge(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	type PosVel struct {
		*Position
		*Velocity
	}

	for i := 0; i < 10000; i++ {
		storage.Spawn(Position{X: float32(i), Y: float32(i)}, Velocity{DX: 0.5, DY: 0.5})
	}

	view := ecs.NewView[PosVel](storage)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pv := range view.Iter() {
			_ = pv
		}
	}
}

func BenchmarkViewSpawn(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	type PosVel struct {
		*Position
		*Velocity
	}

	view := ecs.NewView[PosVel](storage)
	pos := Position{X: 1.0, Y: 2.0}
	vel := Velocity{DX: 0.5, DY: 0.5}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		view.Spawn(PosVel{Position: &pos, Velocity: &vel})
	}
}

func BenchmarkArchetypeCompact(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	archetype := storage.GetArchetype(Position{}, Velocity{})
	if archetype == nil {
		storage.Spawn(Position{X: 1.0, Y: 2.0}, Velocity{DX: 0.5, DY: 0.5})
		archetype = storage.GetArchetype(Position{}, Velocity{})
	}

	for i := 0; i < 1000; i++ {
		id := storage.Spawn(Position{X: float32(i), Y: float32(i)}, Velocity{DX: 0.5, DY: 0.5})
		if i%3 == 0 {
			storage.Delete(id)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		archetype.Compact()
	}
}

func BenchmarkMixedOperations(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	type PosVel struct {
		*Position
		*Velocity
	}

	view := ecs.NewView[PosVel](storage)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := storage.Spawn(Position{X: 1.0, Y: 2.0}, Velocity{DX: 0.5, DY: 0.5})
		_ = ecs.ReadComponent[Position](storage, id)
		id = storage.AddComponent(id, Health{Current: 100, Max: 100})
		_ = view.Get(id)
		storage.Delete(id)
	}
}

func BenchmarkQueryIter(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	type PosVel struct {
		*Position
		*Velocity
	}

	for i := 0; i < 1000; i++ {
		storage.Spawn(Position{X: float32(i), Y: float32(i)}, Velocity{DX: 0.5, DY: 0.5})
	}

	query := ecs.NewQuery[PosVel](storage)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.Execute()
		for _, pv := range query.Iter() {
			_ = pv
		}
	}
}

func BenchmarkQueryIterLarge(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	type PosVel struct {
		*Position
		*Velocity
	}

	for i := 0; i < 10000; i++ {
		storage.Spawn(Position{X: float32(i), Y: float32(i)}, Velocity{DX: 0.5, DY: 0.5})
	}

	query := ecs.NewQuery[PosVel](storage)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.Execute()
		for _, pv := range query.Iter() {
			_ = pv
		}
	}
}

type benchMovementSystem struct {
	Entities ecs.Query[struct {
		*Position
		*Velocity
	}]
}

func (s *benchMovementSystem) Execute(frame *ecs.UpdateFrame) {
	for item := range s.Entities.Values() {
		item.Position.X += item.Velocity.DX * float32(frame.DeltaTime)
		item.Position.Y += item.Velocity.DY * float32(frame.DeltaTime)
	}
}

type benchHealthSystem struct {
	Entities ecs.Query[struct {
		*Health
	}]
}

func (s *benchHealthSystem) Execute(frame *ecs.UpdateFrame) {
	for item := range s.Entities.Values() {
		if item.Health.Current < item.Health.Max {
			item.Health.Current += int(1.0 * float32(frame.DeltaTime))
		}
	}
}

func BenchmarkSchedulerOnce(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	for i := 0; i < 1000; i++ {
		storage.Spawn(Position{X: float32(i), Y: float32(i)}, Velocity{DX: 0.5, DY: 0.5})
	}

	scheduler := ecs.NewScheduler(storage)
	scheduler.Register(&benchMovementSystem{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scheduler.Once(0.016)
	}
}

func BenchmarkSchedulerMultipleSystems(b *testing.B) {
	registry := newTestRegistry()
	storage := ecs.NewStorage(registry)

	for i := 0; i < 1000; i++ {
		storage.Spawn(Position{X: float32(i), Y: float32(i)}, Velocity{DX: 0.5, DY: 0.5}, Health{Current: 50, Max: 100})
	}

	scheduler := ecs.NewScheduler(storage)
	scheduler.Register(&benchMovementSystem{})
	scheduler.Register(&benchHealthSystem{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scheduler.Once(0.016)
	}
}

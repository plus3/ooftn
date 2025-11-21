package ecs

import (
	"testing"
	"time"
)

func TestStorageStats(t *testing.T) {
	registry := NewComponentRegistry()
	RegisterComponent[int](registry)
	RegisterComponent[string](registry)
	RegisterComponent[float64](registry)

	storage := NewStorage(registry)

	stats := storage.CollectStats()
	if stats.ArchetypeCount != 0 {
		t.Errorf("expected 0 archetypes, got %d", stats.ArchetypeCount)
	}
	if stats.TotalEntityCount != 0 {
		t.Errorf("expected 0 entities, got %d", stats.TotalEntityCount)
	}
	if stats.SingletonCount != 0 {
		t.Errorf("expected 0 singletons, got %d", stats.SingletonCount)
	}

	storage.Spawn(42, "hello")
	storage.Spawn(100, "world")
	storage.Spawn(200.0, "test")

	NewSingleton[float64](storage, 3.14)
	NewSingleton[string](storage, "singleton")

	stats = storage.CollectStats()

	if stats.ArchetypeCount != 2 {
		t.Errorf("expected 2 archetypes, got %d", stats.ArchetypeCount)
	}

	if stats.TotalEntityCount != 3 {
		t.Errorf("expected 3 entities, got %d", stats.TotalEntityCount)
	}

	if stats.SingletonCount != 2 {
		t.Errorf("expected 2 singletons, got %d", stats.SingletonCount)
	}

	if len(stats.ArchetypeBreakdown) != 2 {
		t.Errorf("expected 2 archetype breakdown entries, got %d", len(stats.ArchetypeBreakdown))
	}

	if len(stats.SingletonTypes) != 2 {
		t.Errorf("expected 2 singleton types, got %d", len(stats.SingletonTypes))
	}

	foundIntString := false
	foundFloat64String := false
	for _, arch := range stats.ArchetypeBreakdown {
		if arch.EntityCount == 2 {
			foundIntString = true
		}
		if arch.EntityCount == 1 {
			foundFloat64String = true
		}
	}

	if !foundIntString || !foundFloat64String {
		t.Errorf("archetype breakdown incorrect: %+v", stats.ArchetypeBreakdown)
	}
}

type TestSystem struct {
	executeCount int
	sleepDur     time.Duration
}

func (s *TestSystem) Execute(frame *UpdateFrame) {
	s.executeCount++
	if s.sleepDur > 0 {
		time.Sleep(s.sleepDur)
	}
}

func TestSchedulerStats(t *testing.T) {
	registry := NewComponentRegistry()
	storage := NewStorage(registry)
	scheduler := NewScheduler(storage)

	stats := scheduler.GetStats()
	if stats.SystemCount != 0 {
		t.Errorf("expected 0 systems, got %d", stats.SystemCount)
	}
	if stats.TotalExecutions != 0 {
		t.Errorf("expected 0 total executions, got %d", stats.TotalExecutions)
	}

	sys1 := &TestSystem{sleepDur: 1 * time.Millisecond}
	sys2 := &TestSystem{sleepDur: 2 * time.Millisecond}
	scheduler.Register(sys1)
	scheduler.Register(sys2)

	stats = scheduler.GetStats()
	if stats.SystemCount != 2 {
		t.Errorf("expected 2 systems, got %d", stats.SystemCount)
	}

	scheduler.Once(0.016)
	scheduler.Once(0.016)
	scheduler.Once(0.016)

	stats = scheduler.GetStats()

	if stats.TotalExecutions != 6 {
		t.Errorf("expected 6 total executions (2 systems * 3 runs), got %d", stats.TotalExecutions)
	}

	if len(stats.Systems) != 2 {
		t.Errorf("expected 2 system stats, got %d", len(stats.Systems))
	}

	for _, sysStats := range stats.Systems {
		if sysStats.Name != "TestSystem" {
			t.Errorf("expected system name 'TestSystem', got '%s'", sysStats.Name)
		}

		if sysStats.ExecutionCount != 3 {
			t.Errorf("expected 3 executions, got %d", sysStats.ExecutionCount)
		}

		if sysStats.MinDuration == 0 {
			t.Errorf("expected non-zero min duration")
		}

		if sysStats.MaxDuration == 0 {
			t.Errorf("expected non-zero max duration")
		}

		if sysStats.AvgDuration == 0 {
			t.Errorf("expected non-zero avg duration")
		}

		if sysStats.LastDuration == 0 {
			t.Errorf("expected non-zero last duration")
		}

		if sysStats.TotalDuration == 0 {
			t.Errorf("expected non-zero total duration")
		}

		if sysStats.MinDuration > sysStats.AvgDuration {
			t.Errorf("min duration (%v) should be <= avg duration (%v)", sysStats.MinDuration, sysStats.AvgDuration)
		}

		if sysStats.AvgDuration > sysStats.MaxDuration {
			t.Errorf("avg duration (%v) should be <= max duration (%v)", sysStats.AvgDuration, sysStats.MaxDuration)
		}
	}

	if sys1.executeCount != 3 {
		t.Errorf("expected sys1 to execute 3 times, got %d", sys1.executeCount)
	}

	if sys2.executeCount != 3 {
		t.Errorf("expected sys2 to execute 3 times, got %d", sys2.executeCount)
	}
}

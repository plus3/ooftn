package ecs

import (
	"context"
	"reflect"
	"strings"
	"time"
)

// SchedulerStats provides statistics about scheduler execution.
type SchedulerStats struct {
	SystemCount     int
	TotalExecutions int64
	Systems         []SystemStats
}

// SystemStats provides execution statistics for a single system.
type SystemStats struct {
	Name           string
	ExecutionCount int64
	MinDuration    time.Duration
	MaxDuration    time.Duration
	AvgDuration    time.Duration
	LastDuration   time.Duration
	TotalDuration  time.Duration
}

type systemStatsInternal struct {
	name           string
	executionCount int64
	minDuration    time.Duration
	maxDuration    time.Duration
	totalDuration  time.Duration
	lastDuration   time.Duration
}

// Scheduler manages and executes systems in order.
type Scheduler struct {
	storage     *Storage
	systems     []System
	systemStats []*systemStatsInternal
}

// NewScheduler creates a new scheduler for the given storage.
func NewScheduler(storage *Storage) *Scheduler {
	return &Scheduler{
		storage: storage,
		systems: make([]System, 0),
	}
}

// Register adds a system to the scheduler and initializes its Query fields.
func (s *Scheduler) Register(system System) {
	s.initializeQueries(system)
	s.systems = append(s.systems, system)

	systemType := reflect.TypeOf(system)
	if systemType.Kind() == reflect.Ptr {
		systemType = systemType.Elem()
	}
	systemName := systemType.Name()

	s.systemStats = append(s.systemStats, &systemStatsInternal{
		name:        systemName,
		minDuration: time.Duration(1<<63 - 1),
	})
}

func (s *Scheduler) initializeQueries(system System) {
	systemValue := reflect.ValueOf(system)
	if systemValue.Kind() == reflect.Ptr {
		systemValue = systemValue.Elem()
	}

	if systemValue.Kind() != reflect.Struct {
		return
	}

	systemType := systemValue.Type()

	for i := 0; i < systemValue.NumField(); i++ {
		field := systemValue.Field(i)
		fieldType := systemType.Field(i)

		if !field.CanSet() {
			continue
		}

		if field.Kind() != reflect.Struct {
			continue
		}

		typeName := field.Type().Name()

		// Initialize Query fields
		if strings.HasPrefix(typeName, "Query[") {
			initMethod := field.Addr().MethodByName("Init")
			if !initMethod.IsValid() {
				panic("Init method not found on Query field: " + fieldType.Name)
			}

			initMethod.Call([]reflect.Value{
				reflect.ValueOf(s.storage),
			})
			continue
		}

		// Initialize Singleton fields
		if strings.HasPrefix(typeName, "Singleton[") {
			initMethod := field.Addr().MethodByName("Init")
			if !initMethod.IsValid() {
				panic("Init method not found on Singleton field: " + fieldType.Name)
			}

			initMethod.Call([]reflect.Value{
				reflect.ValueOf(s.storage),
			})
			continue
		}
	}
}

// Once executes all registered systems once with the given delta time.
func (s *Scheduler) Once(dt float64) {
	frame := newUpdateFrame(dt, s.storage)

	for i, system := range s.systems {
		start := time.Now()
		system.Execute(frame)
		duration := time.Since(start)

		stats := s.systemStats[i]
		stats.executionCount++
		stats.lastDuration = duration
		stats.totalDuration += duration

		if duration < stats.minDuration {
			stats.minDuration = duration
		}
		if duration > stats.maxDuration {
			stats.maxDuration = duration
		}
	}

	frame.Commands.Flush(s.storage)
}

// Run executes all systems repeatedly at the given interval until the context is cancelled.
func (s *Scheduler) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	lastTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			dt := now.Sub(lastTime).Seconds()
			lastTime = now
			s.Once(dt)
		}
	}
}

// GetStats returns statistics about system execution.
func (s *Scheduler) GetStats() *SchedulerStats {
	stats := &SchedulerStats{
		SystemCount: len(s.systems),
		Systems:     make([]SystemStats, len(s.systemStats)),
	}

	var totalExecs int64
	for i, internal := range s.systemStats {
		avgDuration := time.Duration(0)
		if internal.executionCount > 0 {
			avgDuration = internal.totalDuration / time.Duration(internal.executionCount)
		}

		stats.Systems[i] = SystemStats{
			Name:           internal.name,
			ExecutionCount: internal.executionCount,
			MinDuration:    internal.minDuration,
			MaxDuration:    internal.maxDuration,
			AvgDuration:    avgDuration,
			LastDuration:   internal.lastDuration,
			TotalDuration:  internal.totalDuration,
		}
		totalExecs += internal.executionCount
	}

	stats.TotalExecutions = totalExecs
	return stats
}

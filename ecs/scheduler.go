package ecs

import (
	"context"
	"reflect"
	"strings"
	"time"
)

// Scheduler manages and executes systems in order.
type Scheduler struct {
	storage *Storage
	systems []System
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

		if !strings.HasPrefix(typeName, "Query[") {
			continue
		}

		initMethod := field.Addr().MethodByName("Init")
		if !initMethod.IsValid() {
			panic("Init method not found on Query field: " + fieldType.Name)
		}

		initMethod.Call([]reflect.Value{
			reflect.ValueOf(s.storage),
		})
	}
}

// Once executes all registered systems once with the given delta time.
func (s *Scheduler) Once(dt float64) {
	frame := newUpdateFrame(dt, s.storage)

	s.executeQueries()

	for _, system := range s.systems {
		system.Execute(frame)
	}

	frame.Commands.Flush(s.storage)

	s.invalidateQueries()
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

func (s *Scheduler) executeQueries() {
	for _, system := range s.systems {
		systemValue := reflect.ValueOf(system)
		if systemValue.Kind() == reflect.Ptr {
			systemValue = systemValue.Elem()
		}

		if systemValue.Kind() != reflect.Struct {
			continue
		}

		for i := 0; i < systemValue.NumField(); i++ {
			field := systemValue.Field(i)

			if field.Kind() != reflect.Struct {
				continue
			}

			typeName := field.Type().Name()
			if !strings.HasPrefix(typeName, "Query[") {
				continue
			}

			executeMethod := field.Addr().MethodByName("Execute")
			if executeMethod.IsValid() {
				executeMethod.Call(nil)
			}
		}
	}
}

func (s *Scheduler) invalidateQueries() {
	for _, system := range s.systems {
		systemValue := reflect.ValueOf(system)
		if systemValue.Kind() == reflect.Ptr {
			systemValue = systemValue.Elem()
		}

		if systemValue.Kind() != reflect.Struct {
			continue
		}

		for i := 0; i < systemValue.NumField(); i++ {
			field := systemValue.Field(i)

			if field.Kind() != reflect.Struct {
				continue
			}

			typeName := field.Type().Name()
			if !strings.HasPrefix(typeName, "Query[") {
				continue
			}

			invalidateMethod := field.Addr().MethodByName("invalidateCache")
			if invalidateMethod.IsValid() {
				invalidateMethod.Call(nil)
			}
		}
	}
}

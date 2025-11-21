package ecs

import (
	"reflect"
	"unsafe"
)

// Singleton provides efficient access to a single component instance
// that is not associated with any entity. Use this for global game state,
// configuration, or other singleton data.
type Singleton[T any] struct {
	storage       *Storage
	componentPtr  unsafe.Pointer
	componentType reflect.Type
}

// NewSingleton creates a new Singleton accessor for the given storage.
// If initializer is provided and the singleton doesn't exist in storage,
// it will be created with the initializer value. Otherwise, a zero value is used.
// This guarantees the singleton exists in storage after the call.
func NewSingleton[T any](storage *Storage, initializer ...T) *Singleton[T] {
	var zero T
	componentType := reflect.TypeOf(zero)

	// Check if singleton already exists
	entry := storage.getSingletonEntry(componentType)
	if entry == nil {
		// Create the singleton with initializer or zero value
		var value T
		if len(initializer) > 0 {
			value = initializer[0]
		}
		storage.AddSingleton(value)
		entry = storage.getSingletonEntry(componentType)
	}

	return &Singleton[T]{
		storage:       storage,
		componentPtr:  entry.dataPtr,
		componentType: componentType,
	}
}

// Init initializes the Singleton with a storage reference.
// This is called automatically by the Scheduler during system registration.
// It ensures the singleton exists in storage, creating it with a zero value if needed.
func (s *Singleton[T]) Init(storage *Storage) {
	var zero T
	s.storage = storage
	s.componentType = reflect.TypeOf(zero)

	// Ensure singleton exists in storage
	entry := storage.getSingletonEntry(s.componentType)
	if entry == nil {
		// Create singleton with zero value
		storage.AddSingleton(zero)
		entry = storage.getSingletonEntry(s.componentType)
	}

	s.componentPtr = entry.dataPtr
}

// Get returns a pointer to the singleton component.
// The singleton is guaranteed to exist (it's created automatically if needed).
func (s *Singleton[T]) Get() *T {
	return (*T)(s.componentPtr)
}

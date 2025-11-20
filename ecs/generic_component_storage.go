package ecs

import (
	"iter"
	"reflect"
)

// ComponentRegistry manages component type registration for an ECS instance.
// Each Storage instance has its own ComponentRegistry, allowing multiple
// independent ECS systems to coexist without interference.
type ComponentRegistry struct {
	factories map[reflect.Type]func() iComponentStorage
}

// NewComponentRegistry creates a new component registry.
func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		factories: make(map[reflect.Type]func() iComponentStorage),
	}
}

// RegisterComponent registers a new component type with the given registry.
// This must be called for each component type before it can be used.
func RegisterComponent[T any](r *ComponentRegistry) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	r.factories[t] = func() iComponentStorage {
		return &genericComponentStorage[T]{
			nextIndex: 0,
		}
	}
}

// getFactory returns the factory function for a given component type.
// Returns nil if the type is not registered.
func (r *ComponentRegistry) getFactory(t reflect.Type) func() iComponentStorage {
	return r.factories[t]
}

const (
	genericBlockSize = 64
)

// genericComponentStorage is a generic implementation of iComponentStorage.
// It stores components of a specific type `T` in blocks.
type genericComponentStorage[T any] struct {
	blocks    [][genericBlockSize]T
	filled    [][genericBlockSize]bool
	freeSlots []int
	nextIndex int
}

// Append adds a component to storage and returns its index.
func (cs *genericComponentStorage[T]) Append(item any) int {
	var concreteItem T
	if ptr, ok := item.(*T); ok {
		concreteItem = *ptr
	} else if val, ok := item.(T); ok {
		concreteItem = val
	} else {
		return -1 // Invalid type
	}

	if len(cs.freeSlots) > 0 {
		index := cs.freeSlots[len(cs.freeSlots)-1]
		cs.freeSlots = cs.freeSlots[:len(cs.freeSlots)-1]

		blockIdx := index / genericBlockSize
		slotIdx := index % genericBlockSize

		cs.blocks[blockIdx][slotIdx] = concreteItem
		cs.filled[blockIdx][slotIdx] = true
		return index
	}

	index := cs.nextIndex
	cs.nextIndex++

	blockIdx := index / genericBlockSize
	slotIdx := index % genericBlockSize

	if blockIdx >= len(cs.blocks) {
		cs.blocks = append(cs.blocks, [genericBlockSize]T{})
		cs.filled = append(cs.filled, [genericBlockSize]bool{})
	}

	cs.blocks[blockIdx][slotIdx] = concreteItem
	cs.filled[blockIdx][slotIdx] = true
	return index
}

// Get returns a pointer to the component at the given index.
func (cs *genericComponentStorage[T]) Get(index int) any {
	if index < 0 {
		return nil
	}

	blockIdx := index / genericBlockSize
	slotIdx := index % genericBlockSize

	if blockIdx >= len(cs.blocks) {
		return nil
	}

	if !cs.filled[blockIdx][slotIdx] {
		return nil
	}

	return &cs.blocks[blockIdx][slotIdx]
}

// Delete marks a component slot as empty.
func (cs *genericComponentStorage[T]) Delete(index int) {
	if index < 0 {
		return
	}

	blockIdx := index / genericBlockSize
	slotIdx := index % genericBlockSize

	if blockIdx >= len(cs.blocks) {
		return
	}

	if cs.filled[blockIdx][slotIdx] {
		cs.filled[blockIdx][slotIdx] = false
		var zero T
		cs.blocks[blockIdx][slotIdx] = zero // Zero out the value
		cs.freeSlots = append(cs.freeSlots, index)
	}
}

// Has checks if a component exists at the given index.
func (cs *genericComponentStorage[T]) Has(index int) bool {
	if index < 0 {
		return false
	}

	blockIdx := index / genericBlockSize
	slotIdx := index % genericBlockSize

	if blockIdx >= len(cs.blocks) {
		return false
	}

	return cs.filled[blockIdx][slotIdx]
}

// Compact reorganizes component storage to remove empty slots.
func (cs *genericComponentStorage[T]) Compact() map[int]int {
	indexMap := make(map[int]int)
	writePos := 0

	totalComponents := cs.nextIndex - len(cs.freeSlots)
	if cs.nextIndex == 0 || totalComponents == 0 {
		// Reset to a single block if empty
		cs.blocks = make([][genericBlockSize]T, 1)
		cs.filled = make([][genericBlockSize]bool, 1)
		cs.freeSlots = nil
		cs.nextIndex = 0
		return indexMap
	}

	numNewBlocks := (totalComponents + genericBlockSize - 1) / genericBlockSize
	newBlocks := make([][genericBlockSize]T, numNewBlocks)
	newFilled := make([][genericBlockSize]bool, numNewBlocks)

	for readIdx := 0; readIdx < cs.nextIndex; readIdx++ {
		readBlockIdx := readIdx / genericBlockSize
		readSlotIdx := readIdx % genericBlockSize

		if cs.filled[readBlockIdx][readSlotIdx] {
			oldIndex := readIdx
			indexMap[oldIndex] = writePos

			writeBlockIdx := writePos / genericBlockSize
			writeSlotIdx := writePos % genericBlockSize

			newBlocks[writeBlockIdx][writeSlotIdx] = cs.blocks[readBlockIdx][readSlotIdx]
			newFilled[writeBlockIdx][writeSlotIdx] = true

			writePos++
		}
	}

	cs.blocks = newBlocks
	cs.filled = newFilled
	cs.freeSlots = nil
	cs.nextIndex = writePos

	return indexMap
}

func (cs *genericComponentStorage[T]) Iter() iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; i < cs.nextIndex; i++ {
			blockIdx := i / genericBlockSize
			slotIdx := i % genericBlockSize

			if blockIdx >= len(cs.filled) {
				continue
			}

			if cs.filled[blockIdx][slotIdx] {
				if !yield(i) {
					return
				}
			}
		}
	}
}

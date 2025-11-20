package ecs

import (
	"reflect"
	"slices"
	"weak"

	"github.com/kamstrup/intmap"
)

type byTypeName []reflect.Type

func (a byTypeName) Len() int           { return len(a) }
func (a byTypeName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byTypeName) Less(i, j int) bool { return a[i].String() < a[j].String() }

// Archetype represents a unique combination of component types
type Archetype struct {
	id       uint32
	types    []reflect.Type
	storages []iComponentStorage
	refs     *intmap.Map[EntityId, weak.Pointer[EntityRef]]
}

// NewArchetype creates a new archetype with the given ID and sorted component types
func NewArchetype(id uint32, types []reflect.Type, registry *ComponentRegistry) *Archetype {
	a := &Archetype{
		id:       id,
		types:    types,
		storages: make([]iComponentStorage, len(types)),
		refs:     intmap.New[EntityId, weak.Pointer[EntityRef]](256),
	}

	// Initialize storage for each component type
	for idx, typ := range types {
		factory := registry.getFactory(typ)
		if factory == nil {
			panic("component type " + typ.String() + " not registered")
		}
		a.storages[idx] = factory()
	}

	return a
}

// Spawn creates a new entity in this archetype with the given components
// Returns the storage position as the entity index
func (a *Archetype) Spawn(components []any) uint32 {
	var storagePos int
	for _, comp := range components {
		compType := reflect.TypeOf(comp)
		if compType.Kind() == reflect.Ptr {
			compType = compType.Elem()
		}

		for idx, typ := range a.types {
			if typ == compType {
				storagePos = a.storages[idx].Append(comp)
			}
		}
	}

	return uint32(storagePos)
}

// GetComponent returns the component of the given type for the entity at entityIndex
// The entityIndex is the storage position directly
func (a *Archetype) GetComponent(entityIndex uint32, compType reflect.Type) any {
	var idx int = -1
	for i, typ := range a.types {
		if typ == compType {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil
	}

	return a.storages[idx].Get(int(entityIndex))
}

// Delete marks an entity's components as deleted
// Indices remain stable - the slot is simply marked as empty
func (a *Archetype) Delete(entityIndex uint32) {
	entityId := NewEntityId(a.id, entityIndex)

	weakPtr, ok := a.refs.Get(entityId)
	if ok {
		// Update the EntityRef to mark it as deleted
		if ref := weakPtr.Value(); ref != nil {
			ref.Id = 0
			ref.Archetype = nil
		}
		a.refs.Del(entityId)
	}

	for _, storage := range a.storages {
		storage.Delete(int(entityIndex))
	}
}

// HasComponent checks if this archetype has the given component type
func (a *Archetype) HasComponent(compType reflect.Type) bool {
	return slices.Contains(a.types, compType)
}

// ID returns the archetype's unique identifier
func (a *Archetype) ID() uint32 {
	return a.id
}

// Types returns the sorted component types for this archetype
func (a *Archetype) Types() []reflect.Type {
	return a.types
}

// Compact reorganizes all component storage to eliminate empty slots and reduce fragmentation
// EntityRefs remain valid and are automatically updated to point to the new indices
func (a *Archetype) Compact() {
	if len(a.storages) == 0 {
		return
	}

	// Compact the first storage and use it as the canonical index mapping
	indexMap := a.storages[0].Compact()
	for i := 1; i < len(a.storages); i++ {
		a.storages[i].Compact()
	}

	// Update EntityRefs to point to new indices and clean up dead weak pointers
	// First, update all the refs and collect the mappings
	updatedRefs := make(map[EntityId]weak.Pointer[EntityRef])
	for oldIdx, newIdx := range indexMap {
		oldEntityId := NewEntityId(a.id, uint32(oldIdx))
		weakPtr, ok := a.refs.Get(oldEntityId)
		if ok {
			if ref := weakPtr.Value(); ref != nil {
				// Update the EntityRef's Id to point to the new index
				ref.Id = NewEntityId(a.id, uint32(newIdx))
				updatedRefs[NewEntityId(a.id, uint32(newIdx))] = weakPtr
			}
			// Mark old entry for deletion (whether weak pointer is alive or dead)
		}
	}

	// Clear all old entries from refs map
	a.refs.Clear()

	// Add back only the updated entries
	for newEntityId, weakPtr := range updatedRefs {
		a.refs.Put(newEntityId, weakPtr)
	}
}

// Iter returns an iterator over all valid EntityIds in this archetype
func (a *Archetype) Iter() func(yield func(EntityId) bool) {
	return func(yield func(EntityId) bool) {
		if len(a.storages) == 0 {
			return
		}

		for index := range a.storages[0].Iter() {
			entityId := NewEntityId(a.id, uint32(index))
			if !yield(entityId) {
				return
			}
		}
	}
}

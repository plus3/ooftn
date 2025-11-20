package ecs

import (
	"reflect"
	"sort"
	"unsafe"
	"weak"
)

// Storage is the main ECS storage interface
type Storage struct {
	archetypes map[uint32]*Archetype
	registry   *ComponentRegistry
}

// NewStorage creates a new ECS storage system with the given component registry
func NewStorage(registry *ComponentRegistry) *Storage {
	return &Storage{
		archetypes: make(map[uint32]*Archetype),
		registry:   registry,
	}
}

func (s *Storage) CreateEntityRef(id EntityId) *EntityRef {
	archetype := s.archetypes[id.ArchetypeId()]
	if archetype == nil {
		return nil
	}

	// Check if we already have a ref for this entity
	if weakPtr, ok := archetype.refs.Get(id); ok {
		if ref := weakPtr.Value(); ref != nil {
			return ref
		}
		// Weak pointer is dead, remove it
		archetype.refs.Del(id)
	}

	// Create new EntityRef
	ref := &EntityRef{
		Id:        id,
		Archetype: archetype,
	}

	// Store weak pointer in archetype
	weakPtr := weak.Make(ref)
	archetype.refs.Put(id, weakPtr)

	return ref
}

func (s *Storage) ResolveEntityRef(ref *EntityRef) (EntityId, bool) {
	if ref == nil {
		return 0, false
	}
	// Check if the ref has been invalidated (Id == 0 means deleted)
	if ref.Id == 0 {
		return 0, false
	}
	return ref.Id, true
}

func (s *Storage) InvalidateEntityRef(ref *EntityRef) bool {
	if ref == nil || ref.Id == 0 {
		return false
	}

	// Mark the ref as deleted
	archetype := s.archetypes[ref.Id.ArchetypeId()]
	if archetype != nil {
		archetype.refs.Del(ref.Id)
	}

	ref.Id = 0
	ref.Archetype = nil
	return true
}

// GetArchetype returns an archetype storage (if one exists)
func (s *Storage) GetArchetype(components ...any) *Archetype {
	types := extractComponentTypes(components)
	archetypeId := hashTypesToUint32(types)
	return s.archetypes[archetypeId]
}

// GetArchetypeByTypes returns an archetype storage (if one exists) based on reflect.Type
func (s *Storage) GetArchetypeByTypes(types []reflect.Type) *Archetype {
	sort.Sort(byTypeName(types))
	archetypeId := hashTypesToUint32(types)
	return s.archetypes[archetypeId]
}

// Spawn creates a new entity with the provided components
func (s *Storage) Spawn(components ...any) EntityId {
	if len(components) == 0 {
		panic("cannot spawn entity without components")
	}

	types := extractComponentTypes(components)
	archetypeId := hashTypesToUint32(types)

	archetype, exists := s.archetypes[archetypeId]
	if !exists {
		archetype = NewArchetype(archetypeId, types, s.registry)
		s.archetypes[archetypeId] = archetype
	}

	entityIndex := archetype.Spawn(components)
	return NewEntityId(archetypeId, entityIndex)
}

// Delete removes all data related to the entity ID
func (s *Storage) Delete(id EntityId) {
	archetypeId := id.ArchetypeId()
	entityIndex := id.Index()

	archetype, ok := s.archetypes[archetypeId]
	if !ok {
		return
	}

	archetype.Delete(entityIndex)
}

func (s *Storage) AddComponent(id EntityId, component any) EntityId {
	oldArchetype := s.archetypes[id.ArchetypeId()]

	compType := reflect.TypeOf(component)
	if compType.Kind() == reflect.Ptr {
		compType = compType.Elem()
	}

	newTypes := make([]reflect.Type, 0, len(oldArchetype.types)+1)
	newTypes = append(newTypes, oldArchetype.types...)
	newTypes = append(newTypes, compType)
	sort.Sort(byTypeName(newTypes))

	newArchetypeId := hashTypesToUint32(newTypes)
	newArchetype, exists := s.archetypes[newArchetypeId]
	if !exists {
		newArchetype = NewArchetype(newArchetypeId, newTypes, s.registry)
		s.archetypes[newArchetypeId] = newArchetype
	}

	// Get the weak pointer if it exists
	weakPtr, hasRef := oldArchetype.refs.Get(id)

	components := make([]any, 0, len(newTypes))
	for _, typ := range newTypes {
		if typ == compType {
			components = append(components, component)
		} else {
			comp := oldArchetype.GetComponent(id.Index(), typ)
			components = append(components, comp)
		}
	}

	newIndex := newArchetype.Spawn(components)
	newId := NewEntityId(newArchetypeId, newIndex)

	// Update EntityRef if it exists
	if hasRef {
		if ref := weakPtr.Value(); ref != nil {
			ref.Id = newId
			ref.Archetype = newArchetype
		}
		oldArchetype.refs.Del(id)
		newArchetype.refs.Put(newId, weakPtr)
	}

	oldArchetype.Delete(id.Index())
	return newId
}

func (s *Storage) RemoveComponent(id EntityId, compType reflect.Type) EntityId {
	oldArchetype := s.archetypes[id.ArchetypeId()]

	newTypes := make([]reflect.Type, 0, len(oldArchetype.types)-1)
	for _, typ := range oldArchetype.types {
		if typ != compType {
			newTypes = append(newTypes, typ)
		}
	}

	weakPtr, hasRef := oldArchetype.refs.Get(id)

	if len(newTypes) == 0 {
		// Entity has no components left, delete it
		if hasRef {
			if ref := weakPtr.Value(); ref != nil {
				ref.Id = 0
				ref.Archetype = nil
			}
			oldArchetype.refs.Del(id)
		}
		oldArchetype.Delete(id.Index())
		return 0
	}

	newArchetypeId := hashTypesToUint32(newTypes)
	newArchetype, exists := s.archetypes[newArchetypeId]
	if !exists {
		newArchetype = NewArchetype(newArchetypeId, newTypes, s.registry)
		s.archetypes[newArchetypeId] = newArchetype
	}

	components := make([]any, 0, len(newTypes))
	for _, typ := range newTypes {
		comp := oldArchetype.GetComponent(id.Index(), typ)
		components = append(components, comp)
	}

	newIndex := newArchetype.Spawn(components)
	newId := NewEntityId(newArchetypeId, newIndex)

	// Update EntityRef if it exists
	if hasRef {
		if ref := weakPtr.Value(); ref != nil {
			ref.Id = newId
			ref.Archetype = newArchetype
		}
		oldArchetype.refs.Del(id)
		newArchetype.refs.Put(newId, weakPtr)
	}

	oldArchetype.Delete(id.Index())
	return newId
}

// GetComponent returns the component for the given entity ID and component type
func (s *Storage) GetComponent(id EntityId, compType reflect.Type) any {
	archetypeId := id.ArchetypeId()
	entityIndex := id.Index()

	archetype, ok := s.archetypes[archetypeId]
	if !ok {
		return nil
	}

	return archetype.GetComponent(entityIndex, compType)
}

// HasComponent checks if an entity has a specific component type
func (s *Storage) HasComponent(id EntityId, compType reflect.Type) bool {
	archetypeId := id.ArchetypeId()
	archetype, ok := s.archetypes[archetypeId]
	if !ok {
		return false
	}
	return archetype.HasComponent(compType)
}

// extractComponentTypes extracts and sorts component types from a slice of components
func extractComponentTypes(components []any) []reflect.Type {
	types := make([]reflect.Type, 0, len(components))
	for _, comp := range components {
		compType := reflect.TypeOf(comp)

		// If it's a pointer, get the underlying type
		if compType.Kind() == reflect.Ptr {
			compType = compType.Elem()
		}

		// Components can be structs or primitives (int, string, etc.)
		// But not pointers, maps, channels, or functions (those aren't value types)
		if compType.Kind() == reflect.Ptr || compType.Kind() == reflect.Map ||
			compType.Kind() == reflect.Chan || compType.Kind() == reflect.Func {
			panic("components cannot be pointers, maps, channels, or functions")
		}

		types = append(types, compType)
	}
	sort.Sort(byTypeName(types))
	return types
}

func typeId(t reflect.Type) int {
	ptr := (*iface)(unsafe.Pointer(&t)).data
	return int(uintptr(ptr))
}

// hashTypesToUint32 generates a uint32 hash for a sorted slice of types
func hashTypesToUint32(types []reflect.Type) uint32 {
	var h uint32 = 2166136261     // FNV-1a 32-bit offset basis
	const prime uint32 = 16777619 // FNV-1a 32-bit prime

	for _, t := range types {
		// Use the type's pointer as a unique identifier
		ptr := (*iface)(unsafe.Pointer(&t)).data
		val := uint32(uintptr(ptr))

		// Mix in all 4 bytes if on 64-bit system
		if unsafe.Sizeof(uintptr(0)) == 8 {
			val ^= uint32(uintptr(ptr) >> 32)
		}

		h ^= val
		h *= prime
	}

	return h
}

type ComponentReader interface {
	GetComponent(EntityId, reflect.Type) any
}

func ReadComponent[T any](reader ComponentReader, entityId EntityId) *T {
	return reader.GetComponent(entityId, reflect.TypeFor[T]()).(*T)
}

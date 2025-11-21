package ecs

import (
	"reflect"
	"sort"
	"unsafe"
	"weak"
)

// StorageStats provides statistics about ECS storage state.
type StorageStats struct {
	ArchetypeCount     int
	TotalEntityCount   int
	ArchetypeBreakdown []ArchetypeStats
	SingletonCount     int
	SingletonTypes     []string
	TotalStorageSlots  int
	EmptyStorageSlots  int
	StorageUtilization float32
}

// ArchetypeStats provides statistics for a single archetype.
type ArchetypeStats struct {
	ID             uint32
	ComponentTypes []string
	EntityCount    int
}

// singletonEntry holds data for a singleton component
type singletonEntry struct {
	componentType reflect.Type
	dataPtr       unsafe.Pointer
}

// Storage is the main ECS storage interface
type Storage struct {
	archetypes map[uint32]*Archetype
	registry   *ComponentRegistry
	singletons map[reflect.Type]*singletonEntry
}

// NewStorage creates a new ECS storage system with the given component registry
func NewStorage(registry *ComponentRegistry) *Storage {
	return &Storage{
		archetypes: make(map[uint32]*Archetype),
		registry:   registry,
		singletons: make(map[reflect.Type]*singletonEntry),
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

// GetArchetypes returns all archetypes in storage
func (s *Storage) GetArchetypes() map[uint32]*Archetype {
	return s.archetypes
}

// GetArchetypeById returns an archetype by its ID
func (s *Storage) GetArchetypeById(id uint32) *Archetype {
	return s.archetypes[id]
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

// AddSingleton adds or updates a singleton component in storage.
// Singleton components are not associated with any entity and provide
// efficient global state access. Returns a pointer to the stored component.
func (s *Storage) AddSingleton(component any) unsafe.Pointer {
	// Get the actual value if component is a pointer
	val := reflect.ValueOf(component)
	componentType := val.Type()
	if val.Kind() == reflect.Ptr {
		componentType = componentType.Elem()
		val = val.Elem()
	}

	// Allocate memory for the component and copy the data
	dataPtr := unsafe.Pointer(reflect.New(componentType).Pointer())
	reflect.NewAt(componentType, dataPtr).Elem().Set(val)

	// Store or update the singleton entry
	s.singletons[componentType] = &singletonEntry{
		componentType: componentType,
		dataPtr:       dataPtr,
	}

	return dataPtr
}

// GetSingleton returns a pointer to a singleton component, or nil if it doesn't exist.
func (s *Storage) GetSingleton(componentType reflect.Type) any {
	entry := s.singletons[componentType]
	if entry == nil {
		return nil
	}
	return reflect.NewAt(componentType, entry.dataPtr).Interface()
}

// ReadSingleton reads a singleton component into the provided pointer.
// The ptr parameter must be a pointer to a pointer (e.g., &gameState where gameState is *GameState).
// Returns true if the singleton exists and was successfully read, false otherwise.
//
// Example usage:
//
//	var gameState *GameState
//	if storage.ReadSingleton(&gameState) {
//	    // use gameState here
//	}
func (s *Storage) ReadSingleton(ptr any) bool {
	ptrVal := reflect.ValueOf(ptr)
	if ptrVal.Kind() != reflect.Ptr {
		panic("ReadSingleton: argument must be a pointer to a pointer")
	}

	targetVal := ptrVal.Elem()
	if targetVal.Kind() != reflect.Ptr {
		panic("ReadSingleton: argument must be a pointer to a pointer")
	}

	// Get the component type (not the pointer type)
	componentType := targetVal.Type().Elem()

	entry := s.singletons[componentType]
	if entry == nil {
		return false
	}

	// Set the pointer to point to the singleton data
	targetVal.Set(reflect.NewAt(componentType, entry.dataPtr))
	return true
}

// getSingletonEntry returns the singleton entry for internal use
func (s *Storage) getSingletonEntry(componentType reflect.Type) *singletonEntry {
	return s.singletons[componentType]
}

// CollectStats gathers statistics about the current storage state.
func (s *Storage) CollectStats() *StorageStats {
	stats := &StorageStats{
		ArchetypeCount:     len(s.archetypes),
		ArchetypeBreakdown: make([]ArchetypeStats, 0, len(s.archetypes)),
		SingletonTypes:     make([]string, 0, len(s.singletons)),
	}

	totalEntities := 0
	totalSlots := 0
	emptySlots := 0

	for _, archetype := range s.archetypes {
		entityCount := 0
		if len(archetype.storages) > 0 {
			for idx := range archetype.storages[0].Iter() {
				entityCount++
				_ = idx
			}
		}

		componentTypes := make([]string, len(archetype.types))
		for i, t := range archetype.types {
			componentTypes[i] = t.String()
		}

		stats.ArchetypeBreakdown = append(stats.ArchetypeBreakdown, ArchetypeStats{
			ID:             archetype.id,
			ComponentTypes: componentTypes,
			EntityCount:    entityCount,
		})

		totalEntities += entityCount
	}

	stats.TotalEntityCount = totalEntities
	stats.SingletonCount = len(s.singletons)

	for t := range s.singletons {
		stats.SingletonTypes = append(stats.SingletonTypes, t.String())
	}

	stats.TotalStorageSlots = totalSlots
	stats.EmptyStorageSlots = emptySlots
	if totalSlots > 0 {
		stats.StorageUtilization = float32(totalSlots-emptySlots) / float32(totalSlots)
	}

	return stats
}

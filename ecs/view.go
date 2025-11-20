package ecs

import (
	"iter"
	"reflect"
	"unsafe"
)

// View represents a query for entities with a specific combination of components
// The type T should be a struct with embedded pointer fields for each component type
// Named fields can be marked as optional using the `ecs:"optional"` struct tag
type View[T any] struct {
	storage     *Storage
	types       []reflect.Type
	optional    []bool
	fieldOffset []uintptr

	cachedArchetypeId   *uint32
	cachedSortedTypes   []reflect.Type
	cachedSortedIndices []int
	cachedRequiredCount int
	cachedArchetype     *Archetype

	storageIndicesCache map[uint32][]int
}

// NewView creates a new view for the given struct type
// The struct T should have embedded or named fields that are pointers to component types
// Embedded fields are always required
// Named fields can be marked as optional using the `ecs:"optional"` struct tag
func NewView[T any](storage *Storage) *View[T] {
	var zero T
	structType := reflect.TypeOf(zero)

	if structType.Kind() != reflect.Struct {
		panic("View type parameter must be a struct")
	}

	types := make([]reflect.Type, 0, structType.NumField())
	optional := make([]bool, 0, structType.NumField())
	fieldOffset := make([]uintptr, 0, structType.NumField())

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldType := field.Type

		if fieldType.Kind() != reflect.Ptr {
			panic("View struct fields must be pointer types")
		}

		componentType := fieldType.Elem()
		types = append(types, componentType)
		fieldOffset = append(fieldOffset, field.Offset)

		// Parse struct tag to check if component is optional
		// Embedded fields (field.Anonymous) are always required
		isOptional := false
		if !field.Anonymous {
			tag := field.Tag.Get("ecs")
			if tag != "" {
				if tag == "optional" {
					isOptional = true
				} else {
					panic("invalid ecs tag value: \"" + tag + "\" (only \"optional\" is supported)")
				}
			}
		}
		optional = append(optional, isOptional)
	}

	requiredCount := 0
	for _, opt := range optional {
		if !opt {
			requiredCount++
		}
	}

	sortedIndices := make([]int, len(types))
	for i := range sortedIndices {
		sortedIndices[i] = i
	}

	for i := range sortedIndices {
		for j := i + 1; j < len(sortedIndices); j++ {
			if types[sortedIndices[i]].String() > types[sortedIndices[j]].String() {
				sortedIndices[i], sortedIndices[j] = sortedIndices[j], sortedIndices[i]
			}
		}
	}

	sortedTypes := make([]reflect.Type, len(types))
	for i, idx := range sortedIndices {
		sortedTypes[i] = types[idx]
	}

	return &View[T]{
		storage:             storage,
		types:               types,
		optional:            optional,
		fieldOffset:         fieldOffset,
		cachedSortedIndices: sortedIndices,
		cachedSortedTypes:   sortedTypes,
		cachedRequiredCount: requiredCount,
		storageIndicesCache: make(map[uint32][]int),
	}
}

// Fill populates the provided struct pointer with component data for the given entity
// Returns false if the entity is missing any required components
// Optional components are set to nil if not present
func (v *View[T]) Fill(id EntityId, ptr *T) bool {
	archetypeId := id.ArchetypeId()
	archetype, ok := v.storage.archetypes[archetypeId]
	if !ok {
		return false
	}

	storageIndices, ok := v.storageIndicesCache[archetypeId]
	if !ok {
		storageIndices = v.buildStorageIndices(archetype)
		v.storageIndicesCache[archetypeId] = storageIndices
	}

	structPtr := unsafe.Pointer(ptr)
	entityIndex := int(id.Index())

	for i := 0; i < len(v.types); i++ {
		fieldPtr := unsafe.Pointer(uintptr(structPtr) + v.fieldOffset[i])

		storageIdx := storageIndices[i]
		if storageIdx == -1 {
			if !v.optional[i] {
				return false
			}
			*(*unsafe.Pointer)(fieldPtr) = nil
			continue
		}

		component := archetype.storages[storageIdx].Get(entityIndex)
		if component == nil {
			if !v.optional[i] {
				return false
			}
			*(*unsafe.Pointer)(fieldPtr) = nil
		} else {
			componentPtr := (*iface)(unsafe.Pointer(&component)).data
			*(*unsafe.Pointer)(fieldPtr) = componentPtr
		}
	}

	return true
}

// Get returns a populated view struct for the given entity, or nil if the entity
// doesn't have all the required components
func (v *View[T]) Get(id EntityId) *T {
	var result T
	if !v.Fill(id, &result) {
		return nil
	}
	return &result
}

// GetRef returns a populated view struct for the given entity ref, or nil if invalid
func (v *View[T]) GetRef(ref *EntityRef) *T {
	entityId, ok := v.storage.ResolveEntityRef(ref)
	if !ok {
		return nil
	}

	var result T
	if !v.Fill(entityId, &result) {
		return nil
	}
	return &result
}

// matchesArchetype checks if an archetype contains all the required component types for this view
// Optional components are not checked - they may or may not be present
func (v *View[T]) matchesArchetype(archetype *Archetype) bool {
	for i, requiredType := range v.types {
		// Skip optional components
		if v.optional[i] {
			continue
		}
		// Required component must be present
		if !archetype.HasComponent(requiredType) {
			return false
		}
	}
	return true
}

func (v *View[T]) buildStorageIndices(archetype *Archetype) []int {
	storageIndices := make([]int, len(v.types))
	for i, componentType := range v.types {
		storageIndices[i] = -1
		for idx, archetypeType := range archetype.types {
			if archetypeType == componentType {
				storageIndices[i] = idx
				break
			}
		}
	}
	return storageIndices
}

func (v *View[T]) populateResult(resultPtr unsafe.Pointer, archetype *Archetype, entityIndex int, storageIndices []int) bool {
	for i, storageIdx := range storageIndices {
		fieldPtr := unsafe.Pointer(uintptr(resultPtr) + v.fieldOffset[i])

		if storageIdx == -1 {
			if v.optional[i] {
				*(*unsafe.Pointer)(fieldPtr) = nil
				continue
			}
			return false
		}

		component := archetype.storages[storageIdx].Get(entityIndex)
		if component == nil {
			if v.optional[i] {
				*(*unsafe.Pointer)(fieldPtr) = nil
				continue
			}
			return false
		}

		componentPtr := (*iface)(unsafe.Pointer(&component)).data
		*(*unsafe.Pointer)(fieldPtr) = componentPtr
	}
	return true
}

// Iter returns an iterator over all entities that have all the required components for this view
// The iterator yields (EntityId, T) pairs where T is the populated view struct
// Optional components are set to nil if not present
func (v *View[T]) Iter() iter.Seq2[EntityId, T] {
	return func(yield func(EntityId, T) bool) {
		for archetypeId, archetype := range v.storage.archetypes {
			if !v.matchesArchetype(archetype) {
				continue
			}

			if len(archetype.storages) == 0 {
				continue
			}

			storageIndices, ok := v.storageIndicesCache[archetypeId]
			if !ok {
				storageIndices = v.buildStorageIndices(archetype)
				v.storageIndicesCache[archetypeId] = storageIndices
			}

			firstStorage := archetype.storages[0]

			var result T
			resultPtr := unsafe.Pointer(&result)

			for entityIndex := range firstStorage.Iter() {
				if !v.populateResult(resultPtr, archetype, entityIndex, storageIndices) {
					continue
				}

				entityId := NewEntityId(archetypeId, uint32(entityIndex))
				if !yield(entityId, result) {
					return
				}
			}
		}
	}
}

// Values returns an iterator over just the view structs (without entity IDs)
// This is useful when you only care about the component data, not which entity it belongs to
func (v *View[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, value := range v.Iter() {
			if !yield(value) {
				return
			}
		}
	}
}

// Spawn creates a new entity with components extracted from the view struct
func (v *View[T]) Spawn(data T) EntityId {
	structPtr := unsafe.Pointer(&data)

	allRequired := true
	componentCount := 0
	for i := 0; i < len(v.types); i++ {
		fieldPtr := unsafe.Pointer(uintptr(structPtr) + v.fieldOffset[i])
		componentPtr := *(*unsafe.Pointer)(fieldPtr)

		if componentPtr == nil {
			if !v.optional[i] {
				panic("required component is nil in View.Spawn")
			}
			allRequired = false
		} else {
			componentCount++
		}
	}

	if componentCount == 0 {
		panic("cannot spawn entity without components")
	}

	if allRequired && v.cachedArchetype != nil {
		components := make([]any, len(v.cachedSortedIndices))
		for i, idx := range v.cachedSortedIndices {
			fieldPtr := unsafe.Pointer(uintptr(structPtr) + v.fieldOffset[idx])
			componentPtr := *(*unsafe.Pointer)(fieldPtr)
			componentType := v.types[idx]
			component := reflect.NewAt(componentType, componentPtr).Elem().Interface()
			components[i] = component
		}

		entityIndex := v.cachedArchetype.Spawn(components)
		return NewEntityId(*v.cachedArchetypeId, entityIndex)
	}

	components := make([]any, 0, componentCount)
	componentIndices := make([]int, 0, componentCount)
	for i := 0; i < len(v.types); i++ {
		fieldPtr := unsafe.Pointer(uintptr(structPtr) + v.fieldOffset[i])
		componentPtr := *(*unsafe.Pointer)(fieldPtr)

		if componentPtr == nil {
			continue
		}

		componentType := v.types[i]
		component := reflect.NewAt(componentType, componentPtr).Elem().Interface()
		components = append(components, component)
		componentIndices = append(componentIndices, i)
	}

	sortedComponents := make([]any, len(components))
	sortedTypes := make([]reflect.Type, len(components))

	sortIdx := 0
	for _, idx := range v.cachedSortedIndices {
		for j, compIdx := range componentIndices {
			if compIdx == idx {
				sortedComponents[sortIdx] = components[j]
				sortedTypes[sortIdx] = v.types[idx]
				sortIdx++
				break
			}
		}
	}

	archetypeId := hashTypesToUint32(sortedTypes)

	if allRequired {
		v.cachedArchetypeId = &archetypeId
	}

	archetype, exists := v.storage.archetypes[archetypeId]
	if !exists {
		archetype = NewArchetype(archetypeId, sortedTypes, v.storage.registry)
		v.storage.archetypes[archetypeId] = archetype
	}

	if allRequired {
		v.cachedArchetype = archetype
	}

	entityIndex := archetype.Spawn(sortedComponents)
	return NewEntityId(archetypeId, entityIndex)
}

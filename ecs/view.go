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

	// Cache for the archetype ID that matches all required (non-optional) components
	// This is computed once and reused for Spawn operations
	cachedArchetypeId *uint32
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

	return &View[T]{
		storage:     storage,
		types:       types,
		optional:    optional,
		fieldOffset: fieldOffset,
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

	// Use unsafe.Pointer to directly access the struct's memory
	// This avoids reflection overhead in the hot path
	structPtr := unsafe.Pointer(ptr)

	for i := 0; i < len(v.types); i++ {
		componentType := v.types[i]
		component := archetype.GetComponent(id.Index(), componentType)

		// Calculate the address of the field using the pre-computed offset
		fieldPtr := unsafe.Pointer(uintptr(structPtr) + v.fieldOffset[i])

		if component == nil {
			// If this is a required component, fail
			if !v.optional[i] {
				return false
			}
			// Optional component is missing, set field to nil
			// For pointer fields, nil is represented as a zero pointer
			*(*unsafe.Pointer)(fieldPtr) = nil
		} else {
			// Component found, set the field to point to the component
			// We need to extract the pointer from the interface{}
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

			storageIndices := v.buildStorageIndices(archetype)
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

	components := make([]any, 0, len(v.types))
	componentTypes := make([]reflect.Type, 0, len(v.types))
	for i := 0; i < len(v.types); i++ {
		fieldPtr := unsafe.Pointer(uintptr(structPtr) + v.fieldOffset[i])

		componentPtr := *(*unsafe.Pointer)(fieldPtr)

		if componentPtr == nil {
			if !v.optional[i] {
				panic("required component is nil in View.Spawn")
			}
			continue
		}

		componentType := v.types[i]
		component := reflect.NewAt(componentType, componentPtr).Elem().Interface()
		components = append(components, component)
		componentTypes = append(componentTypes, componentType)
	}

	if len(components) == 0 {
		panic("cannot spawn entity without components")
	}

	sortedIndices := make([]int, len(componentTypes))
	for i := range sortedIndices {
		sortedIndices[i] = i
	}

	for i := range sortedIndices {
		for j := i + 1; j < len(sortedIndices); j++ {
			if componentTypes[sortedIndices[i]].String() > componentTypes[sortedIndices[j]].String() {
				sortedIndices[i], sortedIndices[j] = sortedIndices[j], sortedIndices[i]
			}
		}
	}

	sortedComponents := make([]any, len(components))
	sortedTypes := make([]reflect.Type, len(componentTypes))
	for i, idx := range sortedIndices {
		sortedComponents[i] = components[idx]
		sortedTypes[i] = componentTypes[idx]
	}

	var archetypeId uint32
	if v.cachedArchetypeId != nil && len(sortedTypes) == len(v.requiredTypes()) {
		archetypeId = *v.cachedArchetypeId
	} else {
		archetypeId = hashTypesToUint32(sortedTypes)
		if len(sortedTypes) == len(v.requiredTypes()) {
			v.cachedArchetypeId = &archetypeId
		}
	}

	archetype, exists := v.storage.archetypes[archetypeId]
	if !exists {
		archetype = NewArchetype(archetypeId, sortedTypes, v.storage.registry)
		v.storage.archetypes[archetypeId] = archetype
	}

	entityIndex := archetype.Spawn(sortedComponents)
	return NewEntityId(archetypeId, entityIndex)
}

// requiredTypes returns a slice of only the required (non-optional) component types
func (v *View[T]) requiredTypes() []reflect.Type {
	required := make([]reflect.Type, 0, len(v.types))
	for i, typ := range v.types {
		if !v.optional[i] {
			required = append(required, typ)
		}
	}
	return required
}

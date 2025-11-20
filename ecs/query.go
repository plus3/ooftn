package ecs

import (
	"iter"
	"unsafe"
)

// Query wraps a View with caching optimizations for repeated iteration.
// Queries cache matching archetypes and pre-build entity/component arrays per frame.
type Query[T any] struct {
	view               *View[T]
	storage            *Storage
	cachedArchetypes   []*Archetype
	lastArchetypeCount int

	cachedEntities   []EntityId
	cachedComponents []T
	cacheValid       bool
}

// NewQuery creates a new Query with archetype-level caching.
func NewQuery[T any](storage *Storage) *Query[T] {
	return &Query[T]{
		view:               NewView[T](storage),
		storage:            storage,
		lastArchetypeCount: -1,
	}
}

// Init initializes or re-initializes the Query with a storage.
// Called by the Scheduler during system registration.
func (q *Query[T]) Init(storage *Storage) {
	q.view = NewView[T](storage)
	q.storage = storage
	q.lastArchetypeCount = -1
	q.cacheValid = false
}

func (q *Query[T]) iterArchetype(archetype *Archetype) iter.Seq2[EntityId, T] {
	return func(yield func(EntityId, T) bool) {
		if len(archetype.storages) == 0 {
			return
		}

		storageIndices := q.view.buildStorageIndices(archetype)
		firstStorage := archetype.storages[0]

		var result T
		resultPtr := unsafe.Pointer(&result)

		for entityIndex := range firstStorage.Iter() {
			if !q.view.populateResult(resultPtr, archetype, entityIndex, storageIndices) {
				continue
			}

			entityId := NewEntityId(archetype.id, uint32(entityIndex))
			if !yield(entityId, result) {
				return
			}
		}
	}
}

// Execute builds the entity and component caches for this frame.
// Called automatically by the Scheduler before systems run.
func (q *Query[T]) Execute() {
	q.invalidateIfNeeded()
	q.ensureArchetypeCache()

	q.cachedEntities = q.cachedEntities[:0]
	q.cachedComponents = q.cachedComponents[:0]

	for _, archetype := range q.cachedArchetypes {
		for id, item := range q.iterArchetype(archetype) {
			q.cachedEntities = append(q.cachedEntities, id)
			q.cachedComponents = append(q.cachedComponents, item)
		}
	}

	q.cacheValid = true
}

func (q *Query[T]) invalidateCache() {
	q.cacheValid = false
}

func (q *Query[T]) invalidateIfNeeded() {
	currentCount := len(q.storage.archetypes)
	if currentCount != q.lastArchetypeCount {
		q.cachedArchetypes = nil
		q.lastArchetypeCount = currentCount
	}
}

func (q *Query[T]) ensureArchetypeCache() {
	if q.cachedArchetypes != nil {
		return
	}

	q.cachedArchetypes = make([]*Archetype, 0)
	for _, archetype := range q.storage.archetypes {
		if q.view.matchesArchetype(archetype) {
			q.cachedArchetypes = append(q.cachedArchetypes, archetype)
		}
	}
}

// Iter returns an iterator over entity IDs and component data.
// Panics if Execute() has not been called this frame.
func (q *Query[T]) Iter() iter.Seq2[EntityId, T] {
	if !q.cacheValid {
		panic("Query.Iter() called before Query.Execute()")
	}

	return func(yield func(EntityId, T) bool) {
		for i := range q.cachedEntities {
			if !yield(q.cachedEntities[i], q.cachedComponents[i]) {
				return
			}
		}
	}
}

// Values returns an iterator over component data only.
// Panics if Execute() has not been called this frame.
func (q *Query[T]) Values() iter.Seq[T] {
	if !q.cacheValid {
		panic("Query.Values() called before Query.Execute()")
	}

	return func(yield func(T) bool) {
		for i := range q.cachedComponents {
			if !yield(q.cachedComponents[i]) {
				return
			}
		}
	}
}

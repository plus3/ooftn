package ecs

import (
	"iter"
	"unsafe"
)

// Query wraps a View with caching optimizations for repeated iteration.
// Queries cache matching archetypes to avoid re-calculating this on every run.
type Query[T any] struct {
	view               *View[T]
	storage            *Storage
	cachedArchetypes   []*Archetype
	lastArchetypeCount int
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
func (q *Query[T]) Iter() iter.Seq2[EntityId, T] {
	return func(yield func(EntityId, T) bool) {
		q.invalidateIfNeeded()
		q.ensureArchetypeCache()

		for _, archetype := range q.cachedArchetypes {
			for id, item := range q.iterArchetype(archetype) {
				if !yield(id, item) {
					return
				}
			}
		}
	}
}

// Values returns an iterator over component data only.
func (q *Query[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		q.invalidateIfNeeded()
		q.ensureArchetypeCache()

		for _, archetype := range q.cachedArchetypes {
			for _, item := range q.iterArchetype(archetype) {
				if !yield(item) {
					return
				}
			}
		}
	}
}

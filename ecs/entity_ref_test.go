package ecs_test

import (
	"reflect"
	"testing"

	"github.com/plus3/ooftn/ecs"
	"github.com/stretchr/testify/assert"
)

func TestEntityRefBasicLifecycle(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id := storage.Spawn(&Position{X: 1.0, Y: 2.0})
	ref := storage.CreateEntityRef(id)

	assert.NotNil(t, ref)
	assert.Equal(t, id, ref.Id)
	assert.NotNil(t, ref.Archetype)

	resolved, ok := storage.ResolveEntityRef(ref)
	assert.True(t, ok)
	assert.Equal(t, id, resolved)

	pos := storage.GetComponent(resolved, reflect.TypeOf(Position{})).(*Position)
	assert.Equal(t, float32(1.0), pos.X)
	assert.Equal(t, float32(2.0), pos.Y)

	ok = storage.InvalidateEntityRef(ref)
	assert.True(t, ok)

	_, ok = storage.ResolveEntityRef(ref)
	assert.False(t, ok)
}

func TestEntityRefStability(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id1 := storage.Spawn(&Position{X: 1.0, Y: 1.0})
	id2 := storage.Spawn(&Position{X: 2.0, Y: 2.0})
	id3 := storage.Spawn(&Position{X: 3.0, Y: 3.0})

	ref1 := storage.CreateEntityRef(id1)
	ref2 := storage.CreateEntityRef(id2)
	ref3 := storage.CreateEntityRef(id3)

	storage.InvalidateEntityRef(ref2)

	resolved1, ok1 := storage.ResolveEntityRef(ref1)
	resolved3, ok3 := storage.ResolveEntityRef(ref3)

	assert.True(t, ok1)
	assert.True(t, ok3)
	assert.Equal(t, id1, resolved1)
	assert.Equal(t, id3, resolved3)

	_, ok2 := storage.ResolveEntityRef(ref2)
	assert.False(t, ok2)
}

func TestEntityRefIdempotency(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id := storage.Spawn(&Position{X: 5.0, Y: 10.0})

	ref1 := storage.CreateEntityRef(id)
	ref2 := storage.CreateEntityRef(id)

	// Should return the same EntityRef pointer
	assert.Same(t, ref1, ref2)
}

func TestEntityRefMultipleInvalidations(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id := storage.Spawn(&Position{X: 1.0, Y: 1.0})
	ref := storage.CreateEntityRef(id)

	ok := storage.InvalidateEntityRef(ref)
	assert.True(t, ok)

	ok = storage.InvalidateEntityRef(ref)
	assert.False(t, ok)

	_, resolved := storage.ResolveEntityRef(ref)
	assert.False(t, resolved)
}

func TestEntityRefInvalidBeforeCreate(t *testing.T) {
	storage := ecs.NewStorage(newTestRegistry())

	_, ok := storage.ResolveEntityRef(nil)
	assert.False(t, ok)

	ok = storage.InvalidateEntityRef(nil)
	assert.False(t, ok)
}

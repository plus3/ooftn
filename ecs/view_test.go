package ecs_test

import (
	"reflect"
	"testing"

	"github.com/plus3/ooftn/ecs"
	"github.com/stretchr/testify/assert"
)

func TestView(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	entityId := storage.Spawn(&Position{
		X: 1,
		Y: 2,
	}, Temperature(32))

	view := ecs.NewView[struct {
		*Position
		*Temperature
	}](storage)

	item := view.Get(entityId)
	assert.NotNil(t, item)
	assert.Equal(t, Temperature(32), *item.Temperature)
	assert.Equal(t, float32(1), item.Position.X)
	assert.Equal(t, float32(2), item.Position.Y)
}

func TestViewMultipleComponents(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	entityId := storage.Spawn(
		&Position{X: 10, Y: 20},
		&Velocity{DX: 1.5, DY: 2.5},
		&Name{Value: "Test Entity"},
	)

	view := ecs.NewView[struct {
		*Position
		*Velocity
		*Name
	}](storage)

	item := view.Get(entityId)
	assert.NotNil(t, item)
	assert.Equal(t, float32(10), item.Position.X)
	assert.Equal(t, float32(20), item.Position.Y)
	assert.Equal(t, float32(1.5), item.Velocity.DX)
	assert.Equal(t, float32(2.5), item.Velocity.DY)
	assert.Equal(t, "Test Entity", item.Name.Value)
}

func TestViewMissingComponent(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	// Entity only has Position, not Velocity
	entityId := storage.Spawn(&Position{X: 5, Y: 10})

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	// Should return nil because entity is missing Velocity
	item := view.Get(entityId)
	assert.Nil(t, item)
}

func TestViewFill(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	entityId := storage.Spawn(&Position{X: 3, Y: 4}, &Health{Current: 50, Max: 100})

	view := ecs.NewView[struct {
		*Position
		*Health
	}](storage)

	var result struct {
		*Position
		*Health
	}

	// Fill should return true and populate the struct
	ok := view.Fill(entityId, &result)
	assert.True(t, ok)
	assert.NotNil(t, result.Position)
	assert.NotNil(t, result.Health)
	assert.Equal(t, float32(3), result.Position.X)
	assert.Equal(t, 50, result.Health.Current)
}

func TestViewFillMissingComponent(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	entityId := storage.Spawn(&Position{X: 1, Y: 2})

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	var result struct {
		*Position
		*Velocity
	}

	// Fill should return false because Velocity is missing
	ok := view.Fill(entityId, &result)
	assert.False(t, ok)
}

func TestViewComponentMutation(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	entityId := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0, DY: 0})

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	item := view.Get(entityId)
	assert.NotNil(t, item)

	// Mutate the components through the view
	item.Position.X = 100
	item.Position.Y = 200
	item.Velocity.DX = 5
	item.Velocity.DY = 10

	// Verify mutations are persisted in storage
	pos := storage.GetComponent(entityId, reflect.TypeOf(Position{})).(*Position)
	assert.Equal(t, float32(100), pos.X)
	assert.Equal(t, float32(200), pos.Y)

	vel := storage.GetComponent(entityId, reflect.TypeOf(Velocity{})).(*Velocity)
	assert.Equal(t, float32(5), vel.DX)
	assert.Equal(t, float32(10), vel.DY)
}

func TestViewWithPrimitiveComponents(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	entityId := storage.Spawn(&Position{X: 7, Y: 8}, Score(1000))

	view := ecs.NewView[struct {
		*Position
		*Score
	}](storage)

	item := view.Get(entityId)
	assert.NotNil(t, item)
	assert.Equal(t, float32(7), item.Position.X)
	assert.Equal(t, Score(1000), *item.Score)

	// Mutate the primitive
	*item.Score = 2000

	// Verify mutation persisted
	score := storage.GetComponent(entityId, reflect.TypeOf(Score(0))).(*Score)
	assert.Equal(t, Score(2000), *score)
}

func TestViewInvalidEntityId(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	fakeId := ecs.NewEntityId(9999, 9999)

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	item := view.Get(fakeId)
	assert.Nil(t, item)
}

func TestViewMultipleEntities(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	// Create multiple entities with same components
	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1})
	id2 := storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2})
	id3 := storage.Spawn(&Position{X: 3, Y: 3}, &Velocity{DX: 0.3, DY: 0.3})

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	// Verify each entity can be queried correctly
	item1 := view.Get(id1)
	assert.NotNil(t, item1)
	assert.Equal(t, float32(1), item1.Position.X)
	assert.Equal(t, float32(0.1), item1.Velocity.DX)

	item2 := view.Get(id2)
	assert.NotNil(t, item2)
	assert.Equal(t, float32(2), item2.Position.X)
	assert.Equal(t, float32(0.2), item2.Velocity.DX)

	item3 := view.Get(id3)
	assert.NotNil(t, item3)
	assert.Equal(t, float32(3), item3.Position.X)
	assert.Equal(t, float32(0.3), item3.Velocity.DX)
}

func TestViewSubset(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	// Entity has more components than the view requires
	entityId := storage.Spawn(
		&Position{X: 5, Y: 5},
		&Velocity{DX: 1, DY: 1},
		&Name{Value: "Extra"},
		&Health{Current: 100, Max: 100},
	)

	// View only asks for a subset
	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	item := view.Get(entityId)
	assert.NotNil(t, item)
	assert.Equal(t, float32(5), item.Position.X)
	assert.Equal(t, float32(1), item.Velocity.DX)
}

func TestViewIter(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	// Spawn entities with Position and Velocity
	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1})
	id2 := storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2})
	id3 := storage.Spawn(&Position{X: 3, Y: 3}, &Velocity{DX: 0.3, DY: 0.3})

	// Spawn an entity with only Position (should not be included)
	storage.Spawn(&Position{X: 99, Y: 99})

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	// Collect all entities from the iterator
	entities := make(map[ecs.EntityId]struct {
		*Position
		*Velocity
	})

	for id, item := range view.Iter() {
		entities[id] = item
	}

	// Should have exactly 3 entities
	assert.Equal(t, 3, len(entities))

	// Verify each entity is present with correct data
	assert.Contains(t, entities, id1)
	assert.Equal(t, float32(1), entities[id1].Position.X)
	assert.Equal(t, float32(0.1), entities[id1].Velocity.DX)

	assert.Contains(t, entities, id2)
	assert.Equal(t, float32(2), entities[id2].Position.X)
	assert.Equal(t, float32(0.2), entities[id2].Velocity.DX)

	assert.Contains(t, entities, id3)
	assert.Equal(t, float32(3), entities[id3].Position.X)
	assert.Equal(t, float32(0.3), entities[id3].Velocity.DX)
}

func TestViewIterEmpty(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	count := 0
	for range view.Iter() {
		count++
	}

	assert.Equal(t, 0, count)
}

func TestViewIterMultipleArchetypes(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	// Create entities with different archetype combinations
	// Archetype 1: Position + Velocity
	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1})
	id2 := storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2})

	// Archetype 2: Position + Velocity + Name
	id3 := storage.Spawn(&Position{X: 3, Y: 3}, &Velocity{DX: 0.3, DY: 0.3}, &Name{Value: "Entity3"})
	id4 := storage.Spawn(&Position{X: 4, Y: 4}, &Velocity{DX: 0.4, DY: 0.4}, &Name{Value: "Entity4"})

	// Archetype 3: Position only (should not match)
	storage.Spawn(&Position{X: 99, Y: 99})

	// Archetype 4: Velocity only (should not match)
	storage.Spawn(&Velocity{DX: 99, DY: 99})

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	// Collect all entities
	entities := make(map[ecs.EntityId]bool)
	for id := range view.Iter() {
		entities[id] = true
	}

	// Should match entities from both archetypes 1 and 2
	assert.Equal(t, 4, len(entities))
	assert.True(t, entities[id1])
	assert.True(t, entities[id2])
	assert.True(t, entities[id3])
	assert.True(t, entities[id4])
}

func TestViewValues(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	storage.Spawn(&Position{X: 1, Y: 10}, &Velocity{DX: 0.1, DY: 1.0})
	storage.Spawn(&Position{X: 2, Y: 20}, &Velocity{DX: 0.2, DY: 2.0})
	storage.Spawn(&Position{X: 3, Y: 30}, &Velocity{DX: 0.3, DY: 3.0})

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	// Collect X values
	xValues := make([]float32, 0)
	for item := range view.Values() {
		xValues = append(xValues, item.Position.X)
	}

	assert.Equal(t, 3, len(xValues))
	assert.Contains(t, xValues, float32(1))
	assert.Contains(t, xValues, float32(2))
	assert.Contains(t, xValues, float32(3))
}

func TestViewIterMutation(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0, DY: 0})
	id2 := storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0, DY: 0})
	id3 := storage.Spawn(&Position{X: 3, Y: 3}, &Velocity{DX: 0, DY: 0})

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	// Mutate all entities through the iterator
	for _, item := range view.Iter() {
		item.Velocity.DX = item.Position.X * 10
		item.Velocity.DY = item.Position.Y * 10
	}

	// Verify mutations persisted
	vel1 := storage.GetComponent(id1, reflect.TypeOf(Velocity{})).(*Velocity)
	assert.Equal(t, float32(10), vel1.DX)
	assert.Equal(t, float32(10), vel1.DY)

	vel2 := storage.GetComponent(id2, reflect.TypeOf(Velocity{})).(*Velocity)
	assert.Equal(t, float32(20), vel2.DX)
	assert.Equal(t, float32(20), vel2.DY)

	vel3 := storage.GetComponent(id3, reflect.TypeOf(Velocity{})).(*Velocity)
	assert.Equal(t, float32(30), vel3.DX)
	assert.Equal(t, float32(30), vel3.DY)
}

func TestViewIterEarlyBreak(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1})
	storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2})
	storage.Spawn(&Position{X: 3, Y: 3}, &Velocity{DX: 0.3, DY: 0.3})
	storage.Spawn(&Position{X: 4, Y: 4}, &Velocity{DX: 0.4, DY: 0.4})
	storage.Spawn(&Position{X: 5, Y: 5}, &Velocity{DX: 0.5, DY: 0.5})

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	// Break after processing 2 entities
	count := 0
	for range view.Iter() {
		count++
		if count == 2 {
			break
		}
	}

	assert.Equal(t, 2, count)
}

func TestViewIterWithDeletedEntities(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1})
	id2 := storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2})
	id3 := storage.Spawn(&Position{X: 3, Y: 3}, &Velocity{DX: 0.3, DY: 0.3})
	id4 := storage.Spawn(&Position{X: 4, Y: 4}, &Velocity{DX: 0.4, DY: 0.4})

	// Delete the middle entity
	storage.Delete(id2)

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	// Collect all entities
	entities := make(map[ecs.EntityId]bool)
	for id := range view.Iter() {
		entities[id] = true
	}

	// Should have 3 entities (id2 deleted)
	assert.Equal(t, 3, len(entities))
	assert.True(t, entities[id1])
	assert.False(t, entities[id2]) // Deleted, should not be present
	assert.True(t, entities[id3])
	assert.True(t, entities[id4])
}

func TestViewIterLargeDataset(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	const numEntities = 1000

	// Spawn many entities
	for i := 0; i < numEntities; i++ {
		storage.Spawn(
			&Position{X: float32(i), Y: float32(i * 2)},
			&Velocity{DX: float32(i) * 0.1, DY: float32(i) * 0.2},
		)
	}

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	// Count and verify
	count := 0
	sum := float32(0)
	for _, item := range view.Iter() {
		count++
		sum += item.Position.X
	}

	assert.Equal(t, numEntities, count)
	// Sum of 0 to 999 is 499500
	assert.Equal(t, float32(499500), sum)
}

func TestViewIterWithPrimitives(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	storage.Spawn(&Position{X: 1, Y: 1}, Score(100))
	storage.Spawn(&Position{X: 2, Y: 2}, Score(200))
	storage.Spawn(&Position{X: 3, Y: 3}, Score(300))

	view := ecs.NewView[struct {
		*Position
		*Score
	}](storage)

	totalScore := Score(0)
	for _, item := range view.Iter() {
		totalScore += *item.Score
	}

	assert.Equal(t, Score(600), totalScore)
}

// Tests for optional component support

func TestViewOptionalComponent(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	// Entity with both components
	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1})
	// Entity with only Position (Velocity optional)
	id2 := storage.Spawn(&Position{X: 2, Y: 2})

	view := ecs.NewView[struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}](storage)

	// Get entity with both components
	item1 := view.Get(id1)
	assert.NotNil(t, item1)
	assert.NotNil(t, item1.Position)
	assert.NotNil(t, item1.Velocity)
	assert.Equal(t, float32(1), item1.Position.X)
	assert.Equal(t, float32(0.1), item1.Velocity.DX)

	// Get entity with only required component
	item2 := view.Get(id2)
	assert.NotNil(t, item2)
	assert.NotNil(t, item2.Position)
	assert.Nil(t, item2.Velocity) // Optional component is nil
	assert.Equal(t, float32(2), item2.Position.X)
}

func TestViewOptionalIterMixedArchetypes(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	// Archetype 1: Position + Velocity
	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1})
	id2 := storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2})

	// Archetype 2: Position only
	id3 := storage.Spawn(&Position{X: 3, Y: 3})
	id4 := storage.Spawn(&Position{X: 4, Y: 4})

	// Archetype 3: Position + Velocity + Health
	id5 := storage.Spawn(&Position{X: 5, Y: 5}, &Velocity{DX: 0.5, DY: 0.5}, &Health{Current: 100, Max: 100})

	view := ecs.NewView[struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}](storage)

	entities := make(map[ecs.EntityId]bool)
	velocityCount := 0

	for id, item := range view.Iter() {
		entities[id] = true
		assert.NotNil(t, item.Position)

		if item.Velocity != nil {
			velocityCount++
		}
	}

	// All 5 entities should match (Position is required, Velocity is optional)
	assert.Equal(t, 5, len(entities))
	assert.True(t, entities[id1])
	assert.True(t, entities[id2])
	assert.True(t, entities[id3])
	assert.True(t, entities[id4])
	assert.True(t, entities[id5])

	// Only 3 entities have Velocity
	assert.Equal(t, 3, velocityCount)
}

func TestViewMultipleOptionalComponents(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	// All components
	storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1}, &Health{Current: 100, Max: 100})
	// Position + Velocity
	storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2})
	// Position + Health
	storage.Spawn(&Position{X: 3, Y: 3}, &Health{Current: 50, Max: 100})
	// Position only
	storage.Spawn(&Position{X: 4, Y: 4})

	view := ecs.NewView[struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
		Health   *Health   `ecs:"optional"`
	}](storage)

	count := 0
	for item := range view.Values() {
		count++
		assert.NotNil(t, item.Position)
		// Velocity and Health may or may not be present
	}

	assert.Equal(t, 4, count)
}

func TestViewOptionalMutation(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 1, DY: 1})
	id2 := storage.Spawn(&Position{X: 2, Y: 2})

	view := ecs.NewView[struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}](storage)

	// Mutate through iterator
	for item := range view.Values() {
		if item.Velocity != nil {
			item.Velocity.DX *= 2
			item.Velocity.DY *= 2
		}
	}

	// Verify mutations
	vel1 := storage.GetComponent(id1, reflect.TypeOf(Velocity{})).(*Velocity)
	assert.Equal(t, float32(2), vel1.DX)
	assert.Equal(t, float32(2), vel1.DY)

	// id2 has no Velocity, so nothing to check
	vel2 := storage.GetComponent(id2, reflect.TypeOf(Velocity{}))
	assert.Nil(t, vel2)
}

func TestViewAllOptional(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	storage.Spawn(&Velocity{DX: 1, DY: 1})
	storage.Spawn(&Health{Current: 100, Max: 100})
	storage.Spawn(&Velocity{DX: 2, DY: 2}, &Health{Current: 50, Max: 100})

	// Both components optional - matches all entities
	view := ecs.NewView[struct {
		Velocity *Velocity `ecs:"optional"`
		Health   *Health   `ecs:"optional"`
	}](storage)

	count := 0
	for item := range view.Values() {
		count++
		// At least one should be present (otherwise entity wouldn't exist)
		assert.True(t, item.Velocity != nil || item.Health != nil)
	}

	assert.Equal(t, 3, count)
}

func TestViewFillWithOptional(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id1 := storage.Spawn(&Position{X: 10, Y: 20}, &Velocity{DX: 1, DY: 2})
	id2 := storage.Spawn(&Position{X: 30, Y: 40})

	view := ecs.NewView[struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}](storage)

	var result1 struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}

	ok := view.Fill(id1, &result1)
	assert.True(t, ok)
	assert.NotNil(t, result1.Position)
	assert.NotNil(t, result1.Velocity)

	var result2 struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}

	ok = view.Fill(id2, &result2)
	assert.True(t, ok)
	assert.NotNil(t, result2.Position)
	assert.Nil(t, result2.Velocity)
}

func TestViewEmbeddedAndOptionalMixed(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1}, &Health{Current: 100, Max: 100})
	id2 := storage.Spawn(&Position{X: 2, Y: 2}, &Health{Current: 50, Max: 100})

	// Mix embedded (required) and named (optional) fields
	view := ecs.NewView[struct {
		*Position           // embedded: always required
		Velocity  *Velocity `ecs:"optional"` // named: optional
		*Health             // embedded: always required
	}](storage)

	// id1 has all components
	item1 := view.Get(id1)
	assert.NotNil(t, item1)
	assert.NotNil(t, item1.Position)
	assert.NotNil(t, item1.Velocity)
	assert.NotNil(t, item1.Health)

	// id2 missing Velocity (optional), should still match
	item2 := view.Get(id2)
	assert.NotNil(t, item2)
	assert.NotNil(t, item2.Position)
	assert.Nil(t, item2.Velocity)
	assert.NotNil(t, item2.Health)
}

func TestViewInvalidTag(t *testing.T) {

	defer func() {
		r := recover()
		assert.NotNil(t, r)
		assert.Contains(t, r.(string), "invalid ecs tag value")
	}()

	storage := ecs.NewStorage(newTestRegistry())

	// This should panic due to invalid tag
	_ = ecs.NewView[struct {
		Position *Position
		Velocity *Velocity `ecs:"invalid"`
	}](storage)
}

func TestViewOptionalWithDeletedEntities(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1})
	id2 := storage.Spawn(&Position{X: 2, Y: 2})
	id3 := storage.Spawn(&Position{X: 3, Y: 3}, &Velocity{DX: 0.3, DY: 0.3})

	// Delete entity with optional component
	storage.Delete(id1)

	view := ecs.NewView[struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}](storage)

	entities := make(map[ecs.EntityId]bool)
	for id := range view.Iter() {
		entities[id] = true
	}

	// Should have 2 entities (id1 deleted)
	assert.Equal(t, 2, len(entities))
	assert.False(t, entities[id1])
	assert.True(t, entities[id2])
	assert.True(t, entities[id3])
}

func TestViewOptionalDoesNotAffectRequiredMatching(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	// Entity missing required component (Health)
	id1 := storage.Spawn(&Position{X: 1, Y: 1})
	// Entity with all components
	id2 := storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2}, &Health{Current: 100, Max: 100})

	view := ecs.NewView[struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
		Health   *Health   // required
	}](storage)

	entities := make(map[ecs.EntityId]bool)
	for id := range view.Iter() {
		entities[id] = true
	}

	// Only id2 should match (id1 missing required Health)
	assert.Equal(t, 1, len(entities))
	assert.False(t, entities[id1])
	assert.True(t, entities[id2])
}

// Tests for View.Spawn functionality

func TestViewSpawn(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	// Spawn an entity using the view
	entityId := view.Spawn(struct {
		*Position
		*Velocity
	}{
		Position: &Position{X: 10, Y: 20},
		Velocity: &Velocity{DX: 1.5, DY: 2.5},
	})

	// Verify the entity was created correctly
	item := view.Get(entityId)
	assert.NotNil(t, item)
	assert.Equal(t, float32(10), item.Position.X)
	assert.Equal(t, float32(20), item.Position.Y)
	assert.Equal(t, float32(1.5), item.Velocity.DX)
	assert.Equal(t, float32(2.5), item.Velocity.DY)
}

func TestViewSpawnMultipleEntities(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	// Spawn multiple entities
	id1 := view.Spawn(struct {
		*Position
		*Velocity
	}{
		Position: &Position{X: 1, Y: 1},
		Velocity: &Velocity{DX: 0.1, DY: 0.1},
	})

	id2 := view.Spawn(struct {
		*Position
		*Velocity
	}{
		Position: &Position{X: 2, Y: 2},
		Velocity: &Velocity{DX: 0.2, DY: 0.2},
	})

	id3 := view.Spawn(struct {
		*Position
		*Velocity
	}{
		Position: &Position{X: 3, Y: 3},
		Velocity: &Velocity{DX: 0.3, DY: 0.3},
	})

	// Verify all entities exist and are correct
	item1 := view.Get(id1)
	assert.NotNil(t, item1)
	assert.Equal(t, float32(1), item1.Position.X)

	item2 := view.Get(id2)
	assert.NotNil(t, item2)
	assert.Equal(t, float32(2), item2.Position.X)

	item3 := view.Get(id3)
	assert.NotNil(t, item3)
	assert.Equal(t, float32(3), item3.Position.X)
}

func TestViewSpawnWithPrimitives(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	view := ecs.NewView[struct {
		*Position
		*Score
	}](storage)

	score := Score(1000)
	entityId := view.Spawn(struct {
		*Position
		*Score
	}{
		Position: &Position{X: 5, Y: 10},
		Score:    &score,
	})

	item := view.Get(entityId)
	assert.NotNil(t, item)
	assert.Equal(t, float32(5), item.Position.X)
	assert.Equal(t, Score(1000), *item.Score)

	// Verify mutation works
	*item.Score = 2000
	item2 := view.Get(entityId)
	assert.Equal(t, Score(2000), *item2.Score)
}

func TestViewSpawnWithOptionalComponentsAllPresent(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	view := ecs.NewView[struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}](storage)

	entityId := view.Spawn(struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}{
		Position: &Position{X: 10, Y: 20},
		Velocity: &Velocity{DX: 1, DY: 2},
	})

	item := view.Get(entityId)
	assert.NotNil(t, item)
	assert.NotNil(t, item.Position)
	assert.NotNil(t, item.Velocity)
	assert.Equal(t, float32(10), item.Position.X)
	assert.Equal(t, float32(1), item.Velocity.DX)
}

func TestViewSpawnWithOptionalComponentsNil(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	view := ecs.NewView[struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}](storage)

	// Spawn with optional component set to nil
	entityId := view.Spawn(struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}{
		Position: &Position{X: 10, Y: 20},
		Velocity: nil,
	})

	// The entity should only have Position component
	item := view.Get(entityId)
	assert.NotNil(t, item)
	assert.NotNil(t, item.Position)
	assert.Nil(t, item.Velocity)
	assert.Equal(t, float32(10), item.Position.X)
}

func TestViewSpawnMixedOptional(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	view := ecs.NewView[struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
		Health   *Health   `ecs:"optional"`
	}](storage)

	// Spawn with only Position and Health
	entityId := view.Spawn(struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
		Health   *Health   `ecs:"optional"`
	}{
		Position: &Position{X: 5, Y: 5},
		Velocity: nil,
		Health:   &Health{Current: 100, Max: 100},
	})

	item := view.Get(entityId)
	assert.NotNil(t, item)
	assert.NotNil(t, item.Position)
	assert.Nil(t, item.Velocity)
	assert.NotNil(t, item.Health)
	assert.Equal(t, 100, item.Health.Current)
}

func TestViewSpawnNilRequiredComponentPanics(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	view := ecs.NewView[struct {
		Position *Position
		Velocity *Velocity
	}](storage)

	defer func() {
		r := recover()
		assert.NotNil(t, r)
		assert.Contains(t, r.(string), "required component is nil")
	}()

	// This should panic because Velocity is required but nil
	view.Spawn(struct {
		Position *Position
		Velocity *Velocity
	}{
		Position: &Position{X: 10, Y: 20},
		Velocity: nil,
	})
}

func TestViewSpawnArchetypeCaching(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	// Spawn first entity (should cache archetype ID)
	id1 := view.Spawn(struct {
		*Position
		*Velocity
	}{
		Position: &Position{X: 1, Y: 1},
		Velocity: &Velocity{DX: 0.1, DY: 0.1},
	})

	// Spawn second entity (should reuse cached archetype ID)
	id2 := view.Spawn(struct {
		*Position
		*Velocity
	}{
		Position: &Position{X: 2, Y: 2},
		Velocity: &Velocity{DX: 0.2, DY: 0.2},
	})

	// Both entities should be in the same archetype
	assert.Equal(t, id1.ArchetypeId(), id2.ArchetypeId())

	// Verify both entities exist with correct data
	item1 := view.Get(id1)
	assert.NotNil(t, item1)
	assert.Equal(t, float32(1), item1.Position.X)

	item2 := view.Get(id2)
	assert.NotNil(t, item2)
	assert.Equal(t, float32(2), item2.Position.X)
}

func TestViewSpawnCompatibleWithStorageSpawn(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	// Spawn using view
	viewId := view.Spawn(struct {
		*Position
		*Velocity
	}{
		Position: &Position{X: 1, Y: 1},
		Velocity: &Velocity{DX: 0.1, DY: 0.1},
	})

	// Spawn using storage with same components
	storageId := storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2})

	// Both should be in the same archetype
	assert.Equal(t, viewId.ArchetypeId(), storageId.ArchetypeId())

	// Both should be retrievable via the view
	viewItem := view.Get(viewId)
	assert.NotNil(t, viewItem)
	assert.Equal(t, float32(1), viewItem.Position.X)

	storageItem := view.Get(storageId)
	assert.NotNil(t, storageItem)
	assert.Equal(t, float32(2), storageItem.Position.X)
}

func TestViewSpawnManyComponents(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	view := ecs.NewView[struct {
		*Position
		*Velocity
		*Health
		*Name
	}](storage)

	entityId := view.Spawn(struct {
		*Position
		*Velocity
		*Health
		*Name
	}{
		Position: &Position{X: 10, Y: 20},
		Velocity: &Velocity{DX: 1, DY: 2},
		Health:   &Health{Current: 80, Max: 100},
		Name:     &Name{Value: "TestEntity"},
	})

	item := view.Get(entityId)
	assert.NotNil(t, item)
	assert.Equal(t, float32(10), item.Position.X)
	assert.Equal(t, float32(1), item.Velocity.DX)
	assert.Equal(t, 80, item.Health.Current)
	assert.Equal(t, "TestEntity", item.Name.Value)
}

func TestViewSpawnIterateSpawnedEntities(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	// Spawn multiple entities
	for i := 0; i < 10; i++ {
		view.Spawn(struct {
			*Position
			*Velocity
		}{
			Position: &Position{X: float32(i), Y: float32(i * 2)},
			Velocity: &Velocity{DX: float32(i) * 0.1, DY: float32(i) * 0.2},
		})
	}

	// Iterate and verify
	count := 0
	for _, item := range view.Iter() {
		count++
		assert.NotNil(t, item.Position)
		assert.NotNil(t, item.Velocity)
	}

	assert.Equal(t, 10, count)
}

func TestViewSpawnMutateAfterSpawn(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	entityId := view.Spawn(struct {
		*Position
		*Velocity
	}{
		Position: &Position{X: 1, Y: 1},
		Velocity: &Velocity{DX: 0, DY: 0},
	})

	// Get and mutate
	item := view.Get(entityId)
	assert.NotNil(t, item)
	item.Position.X = 100
	item.Velocity.DX = 50

	// Verify mutations persisted
	item2 := view.Get(entityId)
	assert.Equal(t, float32(100), item2.Position.X)
	assert.Equal(t, float32(50), item2.Velocity.DX)
}

func TestViewWithPointerComponents(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	enemy := &Name{Value: "Boss"}
	id := storage.Spawn(&Position{X: 5.0, Y: 10.0}, &Target{Enemy: enemy})

	view := ecs.NewView[struct {
		*Position
		*Target
	}](storage)

	item := view.Get(id)
	assert.NotNil(t, item)
	assert.Equal(t, float32(5.0), item.Position.X)
	assert.NotNil(t, item.Target.Enemy)
	assert.Equal(t, "Boss", item.Target.Enemy.Value)
}

func TestViewIterWithPointerComponents(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	target1 := &Position{X: 100.0, Y: 200.0}
	target2 := &Position{X: 300.0, Y: 400.0}

	storage.Spawn(&Position{X: 1.0, Y: 1.0}, &AIPointer{Target: target1})
	storage.Spawn(&Position{X: 2.0, Y: 2.0}, &AIPointer{Target: target2})

	view := ecs.NewView[struct {
		*Position
		*AIPointer
	}](storage)

	count := 0
	for _, item := range view.Iter() {
		assert.NotNil(t, item.AIPointer.Target)
		count++
	}
	assert.Equal(t, 2, count)
}

func TestViewWithSliceComponent(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	items := []string{"sword", "shield"}
	id := storage.Spawn(&Position{X: 1.0, Y: 1.0}, &Inventory{Items: items})

	view := ecs.NewView[struct {
		*Position
		*Inventory
	}](storage)

	item := view.Get(id)
	assert.NotNil(t, item)
	assert.Equal(t, 2, len(item.Inventory.Items))
	assert.Equal(t, "sword", item.Inventory.Items[0])
}

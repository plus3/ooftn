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
	entityId := storage.Spawn(&Position{X: 5, Y: 10})

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

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

	item.Position.X = 100
	item.Position.Y = 200
	item.Velocity.DX = 5
	item.Velocity.DY = 10

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

	*item.Score = 2000

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

	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1})
	id2 := storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2})
	id3 := storage.Spawn(&Position{X: 3, Y: 3}, &Velocity{DX: 0.3, DY: 0.3})

	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

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

	entityId := storage.Spawn(
		&Position{X: 5, Y: 5},
		&Velocity{DX: 1, DY: 1},
		&Name{Value: "Extra"},
		&Health{Current: 100, Max: 100},
	)

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

	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1})
	id2 := storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2})
	id3 := storage.Spawn(&Position{X: 3, Y: 3}, &Velocity{DX: 0.3, DY: 0.3})

	storage.Spawn(&Position{X: 99, Y: 99})

	view := ecs.NewView[struct {
		Id ecs.EntityId
		*Position
		*Velocity
	}](storage)

	entities := make(map[ecs.EntityId]struct {
		Id ecs.EntityId
		*Position
		*Velocity
	})

	for item := range view.Iter() {
		entities[item.Id] = item
	}

	assert.Equal(t, 3, len(entities))

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

	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1})
	id2 := storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2})

	id3 := storage.Spawn(&Position{X: 3, Y: 3}, &Velocity{DX: 0.3, DY: 0.3}, &Name{Value: "Entity3"})
	id4 := storage.Spawn(&Position{X: 4, Y: 4}, &Velocity{DX: 0.4, DY: 0.4}, &Name{Value: "Entity4"})

	storage.Spawn(&Position{X: 99, Y: 99})

	storage.Spawn(&Velocity{DX: 99, DY: 99})

	view := ecs.NewView[struct {
		Id ecs.EntityId
		*Position
		*Velocity
	}](storage)

	entities := make(map[ecs.EntityId]bool)
	for item := range view.Iter() {
		entities[item.Id] = true
	}

	assert.Equal(t, 4, len(entities))
	assert.True(t, entities[id1])
	assert.True(t, entities[id2])
	assert.True(t, entities[id3])
	assert.True(t, entities[id4])
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

	for item := range view.Iter() {
		item.Velocity.DX = item.Position.X * 10
		item.Velocity.DY = item.Position.Y * 10
	}

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

	storage.Delete(id2)

	view := ecs.NewView[struct {
		Id ecs.EntityId
		*Position
		*Velocity
	}](storage)

	entities := make(map[ecs.EntityId]bool)
	for item := range view.Iter() {
		entities[item.Id] = true
	}

	assert.Equal(t, 3, len(entities))
	assert.True(t, entities[id1])
	assert.False(t, entities[id2])
	assert.True(t, entities[id3])
	assert.True(t, entities[id4])
}

func TestViewIterLargeDataset(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	const numEntities = 1000

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

	count := 0
	sum := float32(0)
	for item := range view.Iter() {
		count++
		sum += item.Position.X
	}

	assert.Equal(t, numEntities, count)
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
	for item := range view.Iter() {
		totalScore += *item.Score
	}

	assert.Equal(t, Score(600), totalScore)
}

func TestViewOptionalComponent(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1})
	id2 := storage.Spawn(&Position{X: 2, Y: 2})

	view := ecs.NewView[struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}](storage)

	item1 := view.Get(id1)
	assert.NotNil(t, item1)
	assert.NotNil(t, item1.Position)
	assert.NotNil(t, item1.Velocity)
	assert.Equal(t, float32(1), item1.Position.X)
	assert.Equal(t, float32(0.1), item1.Velocity.DX)

	item2 := view.Get(id2)
	assert.NotNil(t, item2)
	assert.NotNil(t, item2.Position)
	assert.Nil(t, item2.Velocity)
	assert.Equal(t, float32(2), item2.Position.X)
}

func TestViewOptionalIterMixedArchetypes(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1})
	id2 := storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2})

	id3 := storage.Spawn(&Position{X: 3, Y: 3})
	id4 := storage.Spawn(&Position{X: 4, Y: 4})

	id5 := storage.Spawn(&Position{X: 5, Y: 5}, &Velocity{DX: 0.5, DY: 0.5}, &Health{Current: 100, Max: 100})

	view := ecs.NewView[struct {
		Id       ecs.EntityId
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}](storage)

	entities := make(map[ecs.EntityId]bool)
	velocityCount := 0

	for item := range view.Iter() {
		entities[item.Id] = true
		assert.NotNil(t, item.Position)

		if item.Velocity != nil {
			velocityCount++
		}
	}

	assert.Equal(t, 5, len(entities))
	assert.True(t, entities[id1])
	assert.True(t, entities[id2])
	assert.True(t, entities[id3])
	assert.True(t, entities[id4])
	assert.True(t, entities[id5])

	assert.Equal(t, 3, velocityCount)
}

func TestViewMultipleOptionalComponents(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1}, &Health{Current: 100, Max: 100})
	storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2})
	storage.Spawn(&Position{X: 3, Y: 3}, &Health{Current: 50, Max: 100})
	storage.Spawn(&Position{X: 4, Y: 4})

	view := ecs.NewView[struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
		Health   *Health   `ecs:"optional"`
	}](storage)

	count := 0
	for item := range view.Iter() {
		count++
		assert.NotNil(t, item.Position)
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

	for item := range view.Iter() {
		if item.Velocity != nil {
			item.Velocity.DX *= 2
			item.Velocity.DY *= 2
		}
	}

	vel1 := storage.GetComponent(id1, reflect.TypeOf(Velocity{})).(*Velocity)
	assert.Equal(t, float32(2), vel1.DX)
	assert.Equal(t, float32(2), vel1.DY)

	vel2 := storage.GetComponent(id2, reflect.TypeOf(Velocity{}))
	assert.Nil(t, vel2)
}

func TestViewAllOptional(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	storage.Spawn(&Velocity{DX: 1, DY: 1})
	storage.Spawn(&Health{Current: 100, Max: 100})
	storage.Spawn(&Velocity{DX: 2, DY: 2}, &Health{Current: 50, Max: 100})

	view := ecs.NewView[struct {
		Velocity *Velocity `ecs:"optional"`
		Health   *Health   `ecs:"optional"`
	}](storage)

	count := 0
	for item := range view.Iter() {
		count++
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

	view := ecs.NewView[struct {
		*Position
		Velocity *Velocity `ecs:"optional"`
		*Health
	}](storage)

	item1 := view.Get(id1)
	assert.NotNil(t, item1)
	assert.NotNil(t, item1.Position)
	assert.NotNil(t, item1.Velocity)
	assert.NotNil(t, item1.Health)

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

	storage.Delete(id1)

	view := ecs.NewView[struct {
		Id       ecs.EntityId
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}](storage)

	entities := make(map[ecs.EntityId]bool)
	for item := range view.Iter() {
		entities[item.Id] = true
	}

	assert.Equal(t, 2, len(entities))
	assert.False(t, entities[id1])
	assert.True(t, entities[id2])
	assert.True(t, entities[id3])
}

func TestViewOptionalDoesNotAffectRequiredMatching(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id1 := storage.Spawn(&Position{X: 1, Y: 1})
	id2 := storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2}, &Health{Current: 100, Max: 100})

	view := ecs.NewView[struct {
		Id       ecs.EntityId
		Position *Position
		Velocity *Velocity `ecs:"optional"`
		Health   *Health
	}](storage)

	entities := make(map[ecs.EntityId]bool)
	for item := range view.Iter() {
		entities[item.Id] = true
	}

	assert.Equal(t, 1, len(entities))
	assert.False(t, entities[id1])
	assert.True(t, entities[id2])
}

func TestViewSpawn(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	view := ecs.NewView[struct {
		*Position
		*Velocity
	}](storage)

	entityId := view.Spawn(struct {
		*Position
		*Velocity
	}{
		Position: &Position{X: 10, Y: 20},
		Velocity: &Velocity{DX: 1.5, DY: 2.5},
	})

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

	entityId := view.Spawn(struct {
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}{
		Position: &Position{X: 10, Y: 20},
		Velocity: nil,
	})

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

	assert.Equal(t, id1.ArchetypeId(), id2.ArchetypeId())

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

	viewId := view.Spawn(struct {
		*Position
		*Velocity
	}{
		Position: &Position{X: 1, Y: 1},
		Velocity: &Velocity{DX: 0.1, DY: 0.1},
	})

	storageId := storage.Spawn(&Position{X: 2, Y: 2}, &Velocity{DX: 0.2, DY: 0.2})

	assert.Equal(t, viewId.ArchetypeId(), storageId.ArchetypeId())

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

	for i := 0; i < 10; i++ {
		view.Spawn(struct {
			*Position
			*Velocity
		}{
			Position: &Position{X: float32(i), Y: float32(i * 2)},
			Velocity: &Velocity{DX: float32(i) * 0.1, DY: float32(i) * 0.2},
		})
	}

	count := 0
	for item := range view.Iter() {
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

	item := view.Get(entityId)
	assert.NotNil(t, item)
	item.Position.X = 100
	item.Velocity.DX = 50

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
	for item := range view.Iter() {
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

func TestViewEntityIdEmbedded(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id1 := storage.Spawn(&Position{X: 10, Y: 20}, &Velocity{DX: 1, DY: 2})
	id2 := storage.Spawn(&Position{X: 30, Y: 40}, &Velocity{DX: 3, DY: 4})

	view := ecs.NewView[struct {
		ecs.EntityId
		*Position
		*Velocity
	}](storage)

	item1 := view.Get(id1)
	assert.NotNil(t, item1)
	assert.Equal(t, id1, item1.EntityId)
	assert.Equal(t, float32(10), item1.Position.X)

	item2 := view.Get(id2)
	assert.NotNil(t, item2)
	assert.Equal(t, id2, item2.EntityId)
	assert.Equal(t, float32(30), item2.Position.X)
}

func TestViewEntityIdNamed(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	entityId := storage.Spawn(&Position{X: 5, Y: 10})

	view := ecs.NewView[struct {
		Id ecs.EntityId
		*Position
	}](storage)

	item := view.Get(entityId)
	assert.NotNil(t, item)
	assert.Equal(t, entityId, item.Id)
	assert.Equal(t, float32(5), item.Position.X)
}

func TestViewEntityIdFill(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	id := storage.Spawn(&Position{X: 7, Y: 8})

	view := ecs.NewView[struct {
		Id ecs.EntityId
		*Position
	}](storage)

	var result struct {
		Id ecs.EntityId
		*Position
	}

	ok := view.Fill(id, &result)
	assert.True(t, ok)
	assert.Equal(t, id, result.Id)
	assert.Equal(t, float32(7), result.Position.X)
}

func TestViewEntityIdWithOptional(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id1 := storage.Spawn(&Position{X: 1, Y: 1}, &Velocity{DX: 0.1, DY: 0.1})
	id2 := storage.Spawn(&Position{X: 2, Y: 2})

	view := ecs.NewView[struct {
		Id       ecs.EntityId
		Position *Position
		Velocity *Velocity `ecs:"optional"`
	}](storage)

	item1 := view.Get(id1)
	assert.NotNil(t, item1)
	assert.Equal(t, id1, item1.Id)
	assert.NotNil(t, item1.Velocity)

	item2 := view.Get(id2)
	assert.NotNil(t, item2)
	assert.Equal(t, id2, item2.Id)
	assert.Nil(t, item2.Velocity)
}

func TestViewSpawnIgnoresEntityId(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	view := ecs.NewView[struct {
		Id ecs.EntityId
		*Position
		*Velocity
	}](storage)

	spawnedId := view.Spawn(struct {
		Id ecs.EntityId
		*Position
		*Velocity
	}{
		Id:       ecs.EntityId(999),
		Position: &Position{X: 5, Y: 5},
		Velocity: &Velocity{DX: 1, DY: 1},
	})

	assert.NotEqual(t, ecs.EntityId(999), spawnedId)

	item := view.Get(spawnedId)
	assert.NotNil(t, item)
	assert.Equal(t, spawnedId, item.Id)
	assert.Equal(t, float32(5), item.Position.X)
}

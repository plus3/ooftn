package ecs_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/plus3/ooftn/ecs"
	"github.com/stretchr/testify/assert"
)

// Test EntityId encoding/decoding
func TestEntityIdEncoding(t *testing.T) {
	archetypeId := uint32(12345)
	index := uint32(67890)

	entityId := ecs.NewEntityId(archetypeId, index)

	assert.Equal(t, archetypeId, entityId.ArchetypeId())
	assert.Equal(t, index, entityId.Index())
}

func TestEntityIdEdgeCases(t *testing.T) {
	tests := []struct {
		archetypeId uint32
		index       uint32
	}{
		{0, 0},
		{0xFFFFFFFF, 0xFFFFFFFF},
		{1, 0},
		{0, 1},
		{0x12345678, 0x9ABCDEF0},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("archetype=%d,index=%d", tt.archetypeId, tt.index), func(t *testing.T) {
			entityId := ecs.NewEntityId(tt.archetypeId, tt.index)
			assert.Equal(t, tt.archetypeId, entityId.ArchetypeId())
			assert.Equal(t, tt.index, entityId.Index())
		})
	}
}

// Test basic storage operations
func TestSpawnEntity(t *testing.T) {
	storage := ecs.NewStorage(newTestRegistry())

	id := storage.Spawn(&Position{X: 1.0, Y: 2.0}, &Velocity{DX: 0.5, DY: 0.5}, Score(32))
	assert.NotEqual(t, ecs.EntityId(0), id)

	// Verify archetype ID is encoded
	assert.Greater(t, id.ArchetypeId(), uint32(0))
}

func TestGetComponent(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id := storage.Spawn(&Position{X: 3.0, Y: 4.0}, Name("Test Entity"))

	// Get Position component
	posComp := storage.GetComponent(id, reflect.TypeOf(Position{}))
	assert.NotNil(t, posComp)
	pos := posComp.(*Position)
	assert.Equal(t, float32(3.0), pos.X)
	assert.Equal(t, float32(4.0), pos.Y)

	// Get Name component
	nameComp := storage.GetComponent(id, reflect.TypeOf(Name("")))
	assert.NotNil(t, nameComp)
	name := nameComp.(*Name)
	assert.Equal(t, Name("Test Entity"), *name)

	// Try to get non-existent component
	velocityComp := storage.GetComponent(id, reflect.TypeOf(Velocity{}))
	assert.Nil(t, velocityComp)
}

func TestDeleteEntity(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id := storage.Spawn(&Position{X: 1.0, Y: 1.0}, &Health{Current: 100, Max: 100})

	// Verify entity exists
	comp := storage.GetComponent(id, reflect.TypeOf(Position{}))
	assert.NotNil(t, comp)

	// Delete entity
	storage.Delete(id)

	// Verify entity is gone
	comp = storage.GetComponent(id, reflect.TypeOf(Position{}))
	assert.Nil(t, comp)
}

func TestMultipleEntitiesSameArchetype(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	// Spawn multiple entities with same component types
	id1 := storage.Spawn(&Position{X: 1.0, Y: 1.0}, &Velocity{DX: 0.1, DY: 0.1})
	id2 := storage.Spawn(&Position{X: 2.0, Y: 2.0}, &Velocity{DX: 0.2, DY: 0.2})
	id3 := storage.Spawn(&Position{X: 3.0, Y: 3.0}, &Velocity{DX: 0.3, DY: 0.3})

	// They should all have the same archetype ID
	assert.Equal(t, id1.ArchetypeId(), id2.ArchetypeId())
	assert.Equal(t, id1.ArchetypeId(), id3.ArchetypeId())

	// But different entity indices
	assert.NotEqual(t, id1.Index(), id2.Index())
	assert.NotEqual(t, id1.Index(), id3.Index())
	assert.NotEqual(t, id2.Index(), id3.Index())

	// Verify components are correct
	pos1 := storage.GetComponent(id1, reflect.TypeOf(Position{})).(*Position)
	pos2 := storage.GetComponent(id2, reflect.TypeOf(Position{})).(*Position)
	pos3 := storage.GetComponent(id3, reflect.TypeOf(Position{})).(*Position)

	assert.Equal(t, float32(1.0), pos1.X)
	assert.Equal(t, float32(2.0), pos2.X)
	assert.Equal(t, float32(3.0), pos3.X)
}

func TestMultipleDifferentArchetypes(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id1 := storage.Spawn(&Position{X: 1.0, Y: 1.0})
	id2 := storage.Spawn(&Position{X: 2.0, Y: 2.0}, &Velocity{DX: 0.1, DY: 0.1})
	id3 := storage.Spawn(&Position{X: 3.0, Y: 3.0}, Name("Entity 3"))
	id4 := storage.Spawn(&Health{Current: 50, Max: 100})

	// All should have different archetype IDs
	assert.NotEqual(t, id1.ArchetypeId(), id2.ArchetypeId())
	assert.NotEqual(t, id1.ArchetypeId(), id3.ArchetypeId())
	assert.NotEqual(t, id1.ArchetypeId(), id4.ArchetypeId())
	assert.NotEqual(t, id2.ArchetypeId(), id3.ArchetypeId())
	assert.NotEqual(t, id2.ArchetypeId(), id4.ArchetypeId())
	assert.NotEqual(t, id3.ArchetypeId(), id4.ArchetypeId())

	// Verify components
	assert.NotNil(t, storage.GetComponent(id1, reflect.TypeOf(Position{})))
	assert.Nil(t, storage.GetComponent(id1, reflect.TypeOf(Velocity{})))

	assert.NotNil(t, storage.GetComponent(id2, reflect.TypeOf(Position{})))
	assert.NotNil(t, storage.GetComponent(id2, reflect.TypeOf(Velocity{})))
	assert.Nil(t, storage.GetComponent(id2, reflect.TypeOf(Name(""))))

	assert.NotNil(t, storage.GetComponent(id4, reflect.TypeOf(Health{})))
	assert.Nil(t, storage.GetComponent(id4, reflect.TypeOf(Position{})))
}

func TestHasComponent(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id := storage.Spawn(&Position{X: 1.0, Y: 1.0}, &Velocity{DX: 0.5, DY: 0.5})

	assert.True(t, storage.HasComponent(id, reflect.TypeOf(Position{})))
	assert.True(t, storage.HasComponent(id, reflect.TypeOf(Velocity{})))
	assert.False(t, storage.HasComponent(id, reflect.TypeOf(Name(""))))
	assert.False(t, storage.HasComponent(id, reflect.TypeOf(Health{})))
}

func TestComponentMutation(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id := storage.Spawn(&Position{X: 1.0, Y: 1.0})

	// Get and mutate component
	pos := storage.GetComponent(id, reflect.TypeOf(Position{})).(*Position)
	pos.X = 10.0
	pos.Y = 20.0

	// Verify mutation persisted
	pos2 := storage.GetComponent(id, reflect.TypeOf(Position{})).(*Position)
	assert.Equal(t, float32(10.0), pos2.X)
	assert.Equal(t, float32(20.0), pos2.Y)
}

func TestDeleteWithStableIndices(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	// Spawn several entities with same archetype
	id1 := storage.Spawn(&Position{X: 1.0, Y: 1.0}, &Velocity{DX: 0.1, DY: 0.1})
	id2 := storage.Spawn(&Position{X: 2.0, Y: 2.0}, &Velocity{DX: 0.2, DY: 0.2})
	id3 := storage.Spawn(&Position{X: 3.0, Y: 3.0}, &Velocity{DX: 0.3, DY: 0.3})
	id4 := storage.Spawn(&Position{X: 4.0, Y: 4.0}, &Velocity{DX: 0.4, DY: 0.4})

	// Delete middle entity
	storage.Delete(id2)

	// Verify id2 is gone
	assert.Nil(t, storage.GetComponent(id2, reflect.TypeOf(Position{})))

	// Verify others still exist with correct data (indices remain stable)
	pos1 := storage.GetComponent(id1, reflect.TypeOf(Position{})).(*Position)
	assert.Equal(t, float32(1.0), pos1.X)

	pos3 := storage.GetComponent(id3, reflect.TypeOf(Position{})).(*Position)
	assert.Equal(t, float32(3.0), pos3.X)

	pos4 := storage.GetComponent(id4, reflect.TypeOf(Position{})).(*Position)
	assert.Equal(t, float32(4.0), pos4.X)

	// Spawn a new entity - it should reuse the deleted slot
	id5 := storage.Spawn(&Position{X: 5.0, Y: 5.0}, &Velocity{DX: 0.5, DY: 0.5})

	// Verify new entity uses same archetype
	assert.Equal(t, id1.ArchetypeId(), id5.ArchetypeId())

	// Verify new entity data
	pos5 := storage.GetComponent(id5, reflect.TypeOf(Position{})).(*Position)
	assert.Equal(t, float32(5.0), pos5.X)
}

func TestLargeNumberOfEntities(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	const numEntities = 10000

	ids := make([]ecs.EntityId, numEntities)
	for i := range numEntities {
		ids[i] = storage.Spawn(
			&Position{X: float32(i), Y: float32(i * 2)},
			&Health{Current: i, Max: i * 10},
		)
	}

	// Verify all entities
	for i, id := range ids {
		pos := storage.GetComponent(id, reflect.TypeOf(Position{})).(*Position)
		assert.Equal(t, float32(i), pos.X)
		assert.Equal(t, float32(i*2), pos.Y)

		health := storage.GetComponent(id, reflect.TypeOf(Health{})).(*Health)
		assert.Equal(t, i, health.Current)
		assert.Equal(t, i*10, health.Max)
	}
}

func TestComponentTypeOrderIndependence(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	// Spawn entities with same components but in different order
	id1 := storage.Spawn(&Position{X: 1.0, Y: 1.0}, &Velocity{DX: 0.1, DY: 0.1}, Name("A"))
	id2 := storage.Spawn(&Velocity{DX: 0.2, DY: 0.2}, Name("B"), &Position{X: 2.0, Y: 2.0})
	id3 := storage.Spawn(Name("C"), &Position{X: 3.0, Y: 3.0}, &Velocity{DX: 0.3, DY: 0.3})

	// All should have the same archetype ID (components are sorted internally)
	assert.Equal(t, id1.ArchetypeId(), id2.ArchetypeId())
	assert.Equal(t, id1.ArchetypeId(), id3.ArchetypeId())

	// Verify components are stored correctly
	pos1 := storage.GetComponent(id1, reflect.TypeOf(Position{})).(*Position)
	pos2 := storage.GetComponent(id2, reflect.TypeOf(Position{})).(*Position)
	pos3 := storage.GetComponent(id3, reflect.TypeOf(Position{})).(*Position)

	assert.Equal(t, float32(1.0), pos1.X)
	assert.Equal(t, float32(2.0), pos2.X)
	assert.Equal(t, float32(3.0), pos3.X)
}

func TestInvalidEntityId(t *testing.T) {
	storage := ecs.NewStorage(newTestRegistry())

	// Try to get component for non-existent entity
	fakeId := ecs.NewEntityId(9999, 9999)
	comp := storage.GetComponent(fakeId, reflect.TypeOf(Position{}))
	assert.Nil(t, comp)

	// Delete non-existent entity (should not panic)
	storage.Delete(fakeId)
}

func TestPrimitiveComponents(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	// Test with custom primitive types (non-pointer)
	id := storage.Spawn(Score(1337), Tag("player"), Temperature(98.6))

	// Verify we can get the components back
	scoreComp := storage.GetComponent(id, reflect.TypeOf(Score(0)))
	assert.NotNil(t, scoreComp)
	score := scoreComp.(*Score)
	assert.Equal(t, Score(1337), *score)

	tagComp := storage.GetComponent(id, reflect.TypeOf(Tag("")))
	assert.NotNil(t, tagComp)
	tag := tagComp.(*Tag)
	assert.Equal(t, Tag("player"), *tag)

	tempComp := storage.GetComponent(id, reflect.TypeOf(Temperature(0)))
	assert.NotNil(t, tempComp)
	temp := tempComp.(*Temperature)
	assert.Equal(t, Temperature(98.6), *temp)
}

func TestMixedStructAndPrimitiveComponents(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	// Mix struct pointers and primitive values
	id := storage.Spawn(&Position{X: 10, Y: 20}, Score(100), Name("test"))

	// Verify all components
	pos := storage.GetComponent(id, reflect.TypeOf(Position{})).(*Position)
	assert.Equal(t, float32(10), pos.X)

	score := storage.GetComponent(id, reflect.TypeOf(Score(0))).(*Score)
	assert.Equal(t, Score(100), *score)

	name := storage.GetComponent(id, reflect.TypeOf(Name(""))).(*Name)
	assert.Equal(t, Name("test"), *name)
}

func TestPrimitiveMutation(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id := storage.Spawn(Score(100))

	// Get and mutate the component
	score := storage.GetComponent(id, reflect.TypeOf(Score(0))).(*Score)
	*score = 500

	// Verify mutation persisted
	score2 := storage.GetComponent(id, reflect.TypeOf(Score(0))).(*Score)
	assert.Equal(t, Score(500), *score2)
}

func TestBuiltinPrimitives(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	// Test with built-in types (not custom types)
	id := storage.Spawn(int32(42), float64(3.14), string("hello"))

	// Verify we can get them back
	intComp := storage.GetComponent(id, reflect.TypeOf(int32(0))).(*int32)
	assert.Equal(t, int32(42), *intComp)

	floatComp := storage.GetComponent(id, reflect.TypeOf(float64(0))).(*float64)
	assert.Equal(t, 3.14, *floatComp)

	strComp := storage.GetComponent(id, reflect.TypeOf(string(""))).(*string)
	assert.Equal(t, "hello", *strComp)
}

func TestComponentReader(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())
	id := storage.Spawn(TestA("A"), TestB("B"))

	testA := ecs.ReadComponent[TestA](storage, id)
	assert.Equal(t, *testA, TestA("A"))

	testB := ecs.ReadComponent[TestB](storage, id)
	assert.Equal(t, *testB, TestB("B"))
}

func TestGetArchetype(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id := storage.Spawn(TestA("A"), TestB("B"))

	arch1 := storage.GetArchetype(TestA("A"), TestB("B"))
	arch2 := storage.GetArchetypeByTypes([]reflect.Type{reflect.TypeFor[TestA](), reflect.TypeFor[TestB]()})
	assert.Equal(t, arch1, arch2)

	assert.Equal(t, *arch1.GetComponent(id.Index(), reflect.TypeFor[TestA]()).(*TestA), TestA("A"))
}

func TestAddComponent(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id := storage.Spawn(&Position{X: 1.0, Y: 2.0})
	ref := storage.CreateEntityRef(id)

	assert.True(t, storage.HasComponent(id, reflect.TypeOf(Position{})))
	assert.False(t, storage.HasComponent(id, reflect.TypeOf(Velocity{})))

	storage.AddComponent(id, &Velocity{DX: 0.5, DY: 0.5})

	newId, ok := storage.ResolveEntityRef(ref)
	assert.True(t, ok)

	assert.True(t, storage.HasComponent(newId, reflect.TypeOf(Position{})))
	assert.True(t, storage.HasComponent(newId, reflect.TypeOf(Velocity{})))

	pos := storage.GetComponent(newId, reflect.TypeOf(Position{})).(*Position)
	assert.Equal(t, float32(1.0), pos.X)
	assert.Equal(t, float32(2.0), pos.Y)

	vel := storage.GetComponent(newId, reflect.TypeOf(Velocity{})).(*Velocity)
	assert.Equal(t, float32(0.5), vel.DX)
	assert.Equal(t, float32(0.5), vel.DY)
}

func TestAddComponentWithEntityRef(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id := storage.Spawn(&Position{X: 10.0, Y: 20.0})
	ref := storage.CreateEntityRef(id)

	storage.AddComponent(id, &Velocity{DX: 1.0, DY: 2.0})

	resolvedId, ok := storage.ResolveEntityRef(ref)
	assert.True(t, ok)

	pos := storage.GetComponent(resolvedId, reflect.TypeOf(Position{})).(*Position)
	assert.Equal(t, float32(10.0), pos.X)

	vel := storage.GetComponent(resolvedId, reflect.TypeOf(Velocity{})).(*Velocity)
	assert.Equal(t, float32(1.0), vel.DX)
}

func TestRemoveComponent(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id := storage.Spawn(&Position{X: 1.0, Y: 2.0}, &Velocity{DX: 0.5, DY: 0.5})
	ref := storage.CreateEntityRef(id)

	assert.True(t, storage.HasComponent(id, reflect.TypeOf(Position{})))
	assert.True(t, storage.HasComponent(id, reflect.TypeOf(Velocity{})))

	storage.RemoveComponent(id, reflect.TypeOf(Velocity{}))

	newId, ok := storage.ResolveEntityRef(ref)
	assert.True(t, ok)

	assert.True(t, storage.HasComponent(newId, reflect.TypeOf(Position{})))
	assert.False(t, storage.HasComponent(newId, reflect.TypeOf(Velocity{})))

	pos := storage.GetComponent(newId, reflect.TypeOf(Position{})).(*Position)
	assert.Equal(t, float32(1.0), pos.X)
	assert.Equal(t, float32(2.0), pos.Y)
}

func TestRemoveComponentWithEntityRef(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id := storage.Spawn(&Position{X: 5.0, Y: 10.0}, &Velocity{DX: 1.0, DY: 1.0}, Name("test"))
	ref := storage.CreateEntityRef(id)

	storage.RemoveComponent(id, reflect.TypeOf(Velocity{}))

	resolvedId, ok := storage.ResolveEntityRef(ref)
	assert.True(t, ok)

	pos := storage.GetComponent(resolvedId, reflect.TypeOf(Position{})).(*Position)
	assert.Equal(t, float32(5.0), pos.X)

	name := storage.GetComponent(resolvedId, reflect.TypeOf(Name(""))).(*Name)
	assert.Equal(t, Name("test"), *name)

	vel := storage.GetComponent(resolvedId, reflect.TypeOf(Velocity{}))
	assert.Nil(t, vel)
}

func TestRemoveLastComponent(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	id := storage.Spawn(&Position{X: 1.0, Y: 2.0})
	ref := storage.CreateEntityRef(id)

	storage.RemoveComponent(id, reflect.TypeOf(Position{}))

	_, ok := storage.ResolveEntityRef(ref)
	assert.False(t, ok)

	comp := storage.GetComponent(id, reflect.TypeOf(Position{}))
	assert.Nil(t, comp)
}

func TestPointerComponent(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	target := &Position{X: 10.0, Y: 20.0}

	id := storage.Spawn(&AIPointer{Target: target})

	ai := storage.GetComponent(id, reflect.TypeOf(AIPointer{})).(*AIPointer)
	assert.NotNil(t, ai)
	assert.NotNil(t, ai.Target)
	assert.Equal(t, float32(10.0), ai.Target.X)
	assert.Equal(t, float32(20.0), ai.Target.Y)

	ai.Target.X = 100.0
	assert.Equal(t, float32(100.0), target.X)
}

func TestSliceComponent(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	items := []string{"sword", "shield", "potion"}
	id := storage.Spawn(&Inventory{Items: items})

	inv := storage.GetComponent(id, reflect.TypeOf(Inventory{})).(*Inventory)
	assert.NotNil(t, inv)
	assert.Equal(t, 3, len(inv.Items))
	assert.Equal(t, "sword", inv.Items[0])

	inv.Items = append(inv.Items, "armor")
	assert.Equal(t, 4, len(inv.Items))
}

func TestMapComponent(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	attrs := map[string]int{"strength": 10, "dexterity": 15}
	id := storage.Spawn(&Stats{Attributes: attrs})

	stats := storage.GetComponent(id, reflect.TypeOf(Stats{})).(*Stats)
	assert.NotNil(t, stats)
	assert.Equal(t, 10, stats.Attributes["strength"])
	assert.Equal(t, 15, stats.Attributes["dexterity"])

	stats.Attributes["wisdom"] = 12
	assert.Equal(t, 3, len(stats.Attributes))
}

func TestMixedPointerAndValueComponents(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	enemy := ptr(Name("Dragon"))
	id := storage.Spawn(&Position{X: 1.0, Y: 2.0}, &Target{Enemy: enemy})

	pos := storage.GetComponent(id, reflect.TypeOf(Position{})).(*Position)
	assert.Equal(t, float32(1.0), pos.X)

	target := storage.GetComponent(id, reflect.TypeOf(Target{})).(*Target)
	assert.NotNil(t, target.Enemy)
	assert.Equal(t, Name("Dragon"), *target.Enemy)
}

func TestPointerComponentWithEntityRef(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	next := &Position{X: 5.0, Y: 10.0}
	id := storage.Spawn(&Link{Next: next})
	ref := storage.CreateEntityRef(id)

	storage.AddComponent(id, &Velocity{DX: 1.0, DY: 1.0})

	resolvedId, ok := storage.ResolveEntityRef(ref)
	assert.True(t, ok)

	link := storage.GetComponent(resolvedId, reflect.TypeOf(Link{})).(*Link)
	assert.NotNil(t, link.Next)
	assert.Equal(t, float32(5.0), link.Next.X)
}

func TestNestedPointerComponent(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	inner1 := &Inner{Value: 42}
	inner2 := &Inner{Value: 99}
	id := storage.Spawn(&Outer{
		Data: inner1,
		List: []*Inner{inner1, inner2},
	})

	outer := storage.GetComponent(id, reflect.TypeOf(Outer{})).(*Outer)
	assert.NotNil(t, outer)
	assert.Equal(t, 42, outer.Data.Value)
	assert.Equal(t, 2, len(outer.List))
	assert.Equal(t, 99, outer.List[1].Value)
}

func TestPointerComponentDeletion(t *testing.T) {

	storage := ecs.NewStorage(newTestRegistry())

	ref := &Position{X: 1.0, Y: 2.0}
	id := storage.Spawn(&RefComponent{Ref: ref})

	comp := storage.GetComponent(id, reflect.TypeOf(RefComponent{})).(*RefComponent)
	assert.NotNil(t, comp.Ref)

	storage.Delete(id)

	comp2 := storage.GetComponent(id, reflect.TypeOf(RefComponent{}))
	assert.Nil(t, comp2)
}

func TestArchetypeCompact(t *testing.T) {
	storage := ecs.NewStorage(newTestRegistry())

	ids := make([]ecs.EntityId, 100)
	for i := range 100 {
		ids[i] = storage.Spawn(Position{X: float32(i), Y: float32(i)}, Velocity{DX: 1.0, DY: 1.0})
	}

	for i := 0; i < 100; i += 2 {
		storage.Delete(ids[i])
	}

	archetype := storage.GetArchetype(Position{}, Velocity{})
	assert.NotNil(t, archetype)

	archetype.Compact()

	count := 0
	for range archetype.Iter() {
		count += 1
	}
	assert.Equal(t, count, 50)
}

func TestArchetypeCompactWithEntityRefs(t *testing.T) {
	storage := ecs.NewStorage(newTestRegistry())

	type EntityData struct {
		id  ecs.EntityId
		ref *ecs.EntityRef
		x   float32
		y   float32
	}

	entities := make([]EntityData, 100)
	for i := range 100 {
		id := storage.Spawn(Position{X: float32(i), Y: float32(i)}, Velocity{DX: 1.0, DY: 1.0})
		ref := storage.CreateEntityRef(id)
		entities[i] = EntityData{id: id, ref: ref, x: float32(i), y: float32(i)}
	}

	for i := 0; i < 100; i += 2 {
		storage.Delete(entities[i].id)
	}

	archetype := storage.GetArchetype(Position{}, Velocity{})
	assert.NotNil(t, archetype)

	archetype.Compact()

	for i := 1; i < 100; i += 2 {
		resolvedId, ok := storage.ResolveEntityRef(entities[i].ref)
		assert.True(t, ok, fmt.Sprintf("EntityRef %d should still be valid after compaction", i))

		pos := storage.GetComponent(resolvedId, reflect.TypeOf(Position{})).(*Position)
		assert.NotNil(t, pos)
		assert.Equal(t, entities[i].x, pos.X)
		assert.Equal(t, entities[i].y, pos.Y)

		vel := storage.GetComponent(resolvedId, reflect.TypeOf(Velocity{})).(*Velocity)
		assert.NotNil(t, vel)
		assert.Equal(t, float32(1.0), vel.DX)
		assert.Equal(t, float32(1.0), vel.DY)
	}

	for i := 0; i < 100; i += 2 {
		_, ok := storage.ResolveEntityRef(entities[i].ref)
		assert.False(t, ok, fmt.Sprintf("Deleted EntityRef %d should be invalid", i))
	}
}

func TestArchetypeCompactMultipleTimes(t *testing.T) {
	storage := ecs.NewStorage(newTestRegistry())

	refs := make([]*ecs.EntityRef, 0)

	for i := range 50 {
		id := storage.Spawn(Position{X: float32(i), Y: float32(i)}, Velocity{DX: 1.0, DY: 1.0})
		refs = append(refs, storage.CreateEntityRef(id))
	}

	archetype := storage.GetArchetype(Position{}, Velocity{})

	for i := range 25 {
		id, _ := storage.ResolveEntityRef(refs[i])
		storage.Delete(id)
	}
	archetype.Compact()

	for i := 50; i < 75; i++ {
		id := storage.Spawn(Position{X: float32(i), Y: float32(i)}, Velocity{DX: 1.0, DY: 1.0})
		refs = append(refs, storage.CreateEntityRef(id))
	}

	for i := 25; i < 50; i++ {
		id, _ := storage.ResolveEntityRef(refs[i])
		storage.Delete(id)
	}
	archetype.Compact()

	for i := 50; i < 75; i++ {
		id, ok := storage.ResolveEntityRef(refs[i])
		assert.True(t, ok)

		pos := storage.GetComponent(id, reflect.TypeOf(Position{})).(*Position)
		assert.Equal(t, float32(i), pos.X)
	}
}

func TestCompactEmptyArchetype(t *testing.T) {
	storage := ecs.NewStorage(newTestRegistry())

	for i := range 10 {
		id := storage.Spawn(Position{X: float32(i), Y: float32(i)}, Velocity{DX: 1.0, DY: 1.0})
		storage.Delete(id)
	}

	archetype := storage.GetArchetype(Position{}, Velocity{})
	assert.NotNil(t, archetype)

	archetype.Compact()
}

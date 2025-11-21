package ecs

import "reflect"

// Commands provides a buffer for deferred ECS operations that are executed at the end of a frame.
// This prevents structural changes to the ECS storage during system execution.
type Commands struct {
	spawns  []spawnCommand
	deletes []EntityId
	adds    []addComponentCommand
	removes []removeComponentCommand
	defers  []deferCommand
}

func newCommands() *Commands {
	return &Commands{}
}

type deferCommand struct {
	fn func()
}

type spawnCommand struct {
	components []any
}

type addComponentCommand struct {
	entity    EntityId
	component any
}

type removeComponentCommand struct {
	entity   EntityId
	compType reflect.Type
}

// Defer queues a function execution operation.
func (c *Commands) Defer(fn func()) {
	c.defers = append(c.defers, deferCommand{fn: fn})
}

// Spawn queues an entity spawn operation with the given components.
func (c *Commands) Spawn(components ...any) {
	c.spawns = append(c.spawns, spawnCommand{components: components})
}

// Delete queues an entity deletion operation.
func (c *Commands) Delete(entity EntityId) {
	c.deletes = append(c.deletes, entity)
}

// AddComponent queues a component addition operation.
func (c *Commands) AddComponent(entity EntityId, component any) {
	c.adds = append(c.adds, addComponentCommand{
		entity:    entity,
		component: component,
	})
}

// RemoveComponent queues a component removal operation.
func (c *Commands) RemoveComponent(entity EntityId, compType reflect.Type) {
	c.removes = append(c.removes, removeComponentCommand{
		entity:   entity,
		compType: compType,
	})
}

// Flush flushes all commands to the provided storage, reseting the buffer state
func (c *Commands) Flush(storage *Storage) {
	deletedEntities := make(map[EntityId]bool)

	for _, cmd := range c.deletes {
		storage.Delete(cmd)
		deletedEntities[cmd] = true
	}

	for _, cmd := range c.removes {
		if !deletedEntities[cmd.entity] {
			storage.RemoveComponent(cmd.entity, cmd.compType)
		}
	}

	for _, cmd := range c.adds {
		if !deletedEntities[cmd.entity] {
			storage.AddComponent(cmd.entity, cmd.component)
		}
	}

	for _, cmd := range c.spawns {
		storage.Spawn(cmd.components...)
	}

	for _, df := range c.defers {
		df.fn()
	}

	c.spawns = c.spawns[:0]
	c.deletes = c.deletes[:0]
	c.adds = c.adds[:0]
	c.removes = c.removes[:0]
	c.defers = c.defers[:0]
}

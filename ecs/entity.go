package ecs

// EntityId encodes both the archetype ID (upper 32 bits) and the entity index (lower 32 bits)
type EntityId uint64

// NewEntityId creates an EntityId from an archetype ID and entity index
func NewEntityId(archetypeId uint32, index uint32) EntityId {
	return EntityId(uint64(archetypeId)<<32 | uint64(index))
}

// ArchetypeId extracts the archetype ID from the entity ID
func (e EntityId) ArchetypeId() uint32 {
	return uint32(e >> 32)
}

// Index extracts the entity index from the entity ID
func (e EntityId) Index() uint32 {
	return uint32(e & 0xFFFFFFFF)
}

// EntityRef is a stable reference to an entity
type EntityRef struct {
	Id        EntityId
	Archetype *Archetype
}

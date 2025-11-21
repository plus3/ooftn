# ECS Internals

## EntityId

We use EntityIds to encode two very important pieces of information that together can be used to quickly look up data for an entity. Since EntityId is a `uint64`, we use two `uint32`s to encode this information: one for the ArchetypeId and one for the StorageIndex. The ArchetypeId allows us to very quickly identify which archetype this entity lives in, but at the cost of losing stability when entities move archetypes. The StorageIndex encodes the exact index at which this entity's component data lives within the internal component storage. This allows us very easy access to an entity's component data at the cost of losing stability when the underlying entity data is moved.

### Archetype Migration

When an entity's underlying archetype changes, e.g., when we add or remove components from an entity, the underlying storage must be moved. This invalidates **all** pointers to the entity's components, and also the entity's existing EntityId. Only an `EntityRef` that was previously created for this entity can be used to access the new data storage.

### Storage Compaction

The only way for an entity's StorageIndex to change is if the archetype's storage is compacted. This is only ever done manually by users, and it can cause the underlying component data for entities to move, thus invalidating the previous EntityId.

package ecs

// System represents a behavior that operates on entities with specific components.
// User-defined systems should implement this interface and can include Query fields
// for accessing entities, as well as custom state fields that persist between frames.
type System interface {
	Execute(frame *UpdateFrame)
}

package ecs_test

import "github.com/plus3/ooftn/ecs"

// Common test component types
type Position struct {
	X, Y float32
}

type Velocity struct {
	DX, DY float32
}

type Name struct {
	Value string
}

type Health struct {
	Current int
	Max     int
}

type PlayerController struct{}

type AI struct {
	State int
}

// Custom primitive types for testing non-pointer components
type Score int32
type Tag string
type Temperature float64

type TestA string
type TestB string

type AIPointer struct {
	Target *Position
}
type Inventory struct {
	Items []string
}
type Stats struct {
	Attributes map[string]int
}
type Target struct {
	Enemy *Name
}
type Link struct {
	Next *Position
}
type Inner struct {
	Value int
}
type Outer struct {
	Data *Inner
	List []*Inner
}
type RefComponent struct {
	Ref *Position
}

func newTestRegistry() *ecs.ComponentRegistry {
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[Velocity](registry)
	ecs.RegisterComponent[Name](registry)
	ecs.RegisterComponent[Health](registry)
	ecs.RegisterComponent[PlayerController](registry)
	ecs.RegisterComponent[AI](registry)
	ecs.RegisterComponent[Score](registry)
	ecs.RegisterComponent[Tag](registry)
	ecs.RegisterComponent[Temperature](registry)
	ecs.RegisterComponent[TestA](registry)
	ecs.RegisterComponent[TestB](registry)
	ecs.RegisterComponent[int32](registry)
	ecs.RegisterComponent[float64](registry)
	ecs.RegisterComponent[string](registry)
	ecs.RegisterComponent[AIPointer](registry)
	ecs.RegisterComponent[Inventory](registry)
	ecs.RegisterComponent[Stats](registry)
	ecs.RegisterComponent[Target](registry)
	ecs.RegisterComponent[Link](registry)
	ecs.RegisterComponent[Inner](registry)
	ecs.RegisterComponent[Outer](registry)
	ecs.RegisterComponent[RefComponent](registry)
	return registry
}

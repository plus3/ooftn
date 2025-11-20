package ecs

import "unsafe"

// iface represents the internal memory layout of an interface{}.
type iface struct {
	typ  unsafe.Pointer
	data unsafe.Pointer
}

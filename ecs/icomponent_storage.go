package ecs

import "iter"

// iComponentStorage is an interface for a type-erased component storage.
type iComponentStorage interface {
	Append(item any) int
	Delete(index int)
	Get(index int) any
	Has(index int) bool
	Compact() map[int]int
	Iter() iter.Seq[int]
}

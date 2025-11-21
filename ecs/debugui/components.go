package debugui

import (
	"github.com/plus3/ooftn/ecs"
)

type EntityBrowserComponent struct {
	cache              *EntityBrowserCache
	selectedEntityId   ecs.EntityId
	filterText         string
	filterArchetypeId  *uint32
	maxEntitiesPerPage int
	currentPage        int
}

type ComponentInspectorComponent struct {
	selectedEntityId ecs.EntityId
}

type ArchetypeViewerComponent struct {
	cache          *ArchetypeViewerCache
	selectedArchId *uint32
	sortColumn     int
	sortAscending  bool
}

type PerformanceStatsComponent struct {
	historyFrames int
	frameHistory  []float32
	frameIndex    int
}

type QueryDebuggerComponent struct {
	selectedComponentTypes map[string]bool
	cache                  *QueryDebuggerCache
}

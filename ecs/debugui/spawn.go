package debugui

import "github.com/plus3/ooftn/ecs"

func SpawnDebugUI(storage *ecs.Storage) {
	storage.Spawn(NewEntityBrowserComponent(100))
	storage.Spawn(NewComponentInspectorComponent())
	storage.Spawn(NewArchetypeViewerComponent())
	storage.Spawn(NewPerformanceStatsComponent(120))
	storage.Spawn(NewQueryDebuggerComponent())
}

func RegisterDebugUIComponents(registry *ecs.ComponentRegistry) {
	ecs.RegisterComponent[EntityBrowserComponent](registry)
	ecs.RegisterComponent[ComponentInspectorComponent](registry)
	ecs.RegisterComponent[ArchetypeViewerComponent](registry)
	ecs.RegisterComponent[PerformanceStatsComponent](registry)
	ecs.RegisterComponent[QueryDebuggerComponent](registry)
	ecs.RegisterComponent[FrameTimer](registry)
}

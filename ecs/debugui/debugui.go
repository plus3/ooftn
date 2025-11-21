// Package debugui provides immediate-mode GUI integration for ECS applications using Dear ImGui.
// It manages ImGui rendering and input state through ECS components and systems.
package debugui

import (
	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/plus3/ooftn/ecs"
)

// ImguiItem is a component that holds a Dear ImGui render function.
// Attach this to entities that should render ImGui widgets each frame.
type ImguiItem struct {
	Render func()
}

// ImguiInputState tracks Dear ImGui's input capture state as a singleton component.
// Use this to determine if ImGui is consuming mouse or keyboard input.
type ImguiInputState struct {
	WantCaptureMouse    bool
	WantCaptureKeyboard bool
}

// ImguiSystem queries all ImguiItem components and defers their render functions.
// It also updates the ImguiInputState singleton with current input capture state.
type ImguiSystem struct {
	Items               ecs.Query[struct{ *ImguiItem }]
	InputState          ecs.Singleton[ImguiInputState]
	EntityBrowsers      ecs.Query[struct{ *EntityBrowserComponent }]
	ComponentInspectors ecs.Query[struct{ *ComponentInspectorComponent }]
	ArchetypeViewers    ecs.Query[struct{ *ArchetypeViewerComponent }]
	PerformanceStats    ecs.Query[struct{ *PerformanceStatsComponent }]
	QueryDebuggers      ecs.Query[struct{ *QueryDebuggerComponent }]
	FrameTimer          ecs.Singleton[FrameTimer]
}

// Execute updates input state and queues all ImGui render functions for execution.
func (i *ImguiSystem) Execute(frame *ecs.UpdateFrame) {
	state := i.InputState.Get()
	state.WantCaptureMouse = imgui.CurrentIO().WantCaptureMouse()
	state.WantCaptureKeyboard = imgui.CurrentIO().WantCaptureKeyboard()

	var selectedEntityId ecs.EntityId
	var filterArchetypeId *uint32

	for browser := range i.EntityBrowsers.Iter() {
		frame.Commands.Defer(func() {
			browser.Render(frame.Storage)
			selectedEntityId = browser.GetSelectedEntity()
			if browser.filterArchetypeId != nil {
				filterArchetypeId = browser.filterArchetypeId
			}
		})
	}

	for viewer := range i.ArchetypeViewers.Iter() {
		frame.Commands.Defer(func() {
			clickedArchId := viewer.Render(frame.Storage)
			if clickedArchId != nil {
				filterArchetypeId = clickedArchId
			}
		})
	}

	for inspector := range i.ComponentInspectors.Iter() {
		frame.Commands.Defer(func() {
			inspector.Render(frame.Storage, selectedEntityId)
		})
	}

	deltaTime := float32(0.016)
	timer := i.FrameTimer.Get()
	if timer != nil {
		deltaTime = timer.GetDeltaTime()
	}

	for stats := range i.PerformanceStats.Iter() {
		frame.Commands.Defer(func() {
			stats.Render(frame.Storage, deltaTime)
		})
	}

	for debugger := range i.QueryDebuggers.Iter() {
		frame.Commands.Defer(func() {
			debugger.Render(frame.Storage)
		})
	}

	for item := range i.Items.Iter() {
		frame.Commands.Defer(item.Render)
	}

	if filterArchetypeId != nil {
		for browser := range i.EntityBrowsers.Iter() {
			browser.filterArchetypeId = filterArchetypeId
		}
	}
}

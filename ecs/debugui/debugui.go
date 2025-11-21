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
	Items      ecs.Query[struct{ *ImguiItem }]
	InputState ecs.Singleton[ImguiInputState]
}

// Execute updates input state and queues all ImGui render functions for execution.
func (i *ImguiSystem) Execute(frame *ecs.UpdateFrame) {
	state := i.InputState.Get()
	state.WantCaptureMouse = imgui.CurrentIO().WantCaptureMouse()
	state.WantCaptureKeyboard = imgui.CurrentIO().WantCaptureKeyboard()

	for item := range i.Items.Iter() {
		frame.Commands.Defer(item.Render)
	}
}

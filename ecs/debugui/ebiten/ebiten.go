// Package ebiten provides Dear ImGui backend integration for the Ebiten game engine.
package ebiten

import (
	ebitenbackend "github.com/AllenDang/cimgui-go/backend/ebiten-backend"
)

// ImguiBackend wraps the Ebiten-specific Dear ImGui backend implementation.
// Use this to integrate Dear ImGui rendering into Ebiten game loops.
type ImguiBackend struct {
	*ebitenbackend.EbitenBackend
}

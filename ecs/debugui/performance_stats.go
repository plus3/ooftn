package debugui

import (
	"fmt"
	"time"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/plus3/ooftn/ecs"
)

func NewPerformanceStatsComponent(historyFrames int) PerformanceStatsComponent {
	return PerformanceStatsComponent{
		historyFrames: historyFrames,
		frameHistory:  make([]float32, historyFrames),
		frameIndex:    0,
	}
}

func (ps *PerformanceStatsComponent) Render(storage *ecs.Storage, deltaTime float32) {
	if !imgui.BeginV("Performance Stats", nil, imgui.WindowFlagsNone) {
		imgui.End()
		return
	}

	ps.frameHistory[ps.frameIndex] = deltaTime * 1000.0
	ps.frameIndex = (ps.frameIndex + 1) % ps.historyFrames

	stats := storage.CollectStats()

	imgui.Text(fmt.Sprintf("Total Entities: %d", stats.TotalEntityCount))
	imgui.Text(fmt.Sprintf("Archetypes: %d", stats.ArchetypeCount))
	imgui.Text(fmt.Sprintf("Singletons: %d", stats.SingletonCount))

	var avgFrameTime float32
	for _, ft := range ps.frameHistory {
		avgFrameTime += ft
	}
	avgFrameTime /= float32(ps.historyFrames)

	imgui.Text(fmt.Sprintf("Avg Frame Time: %.2f ms (%.0f FPS)", avgFrameTime, 1000.0/avgFrameTime))

	imgui.Separator()
	imgui.Text("Frame Time Graph (ms)")
	imgui.PlotLinesFloatPtr("##frametime", &ps.frameHistory[0], int32(len(ps.frameHistory)))

	if imgui.TreeNodeStr("Archetype Details") {
		const tableFlags = imgui.TableFlagsBorders | imgui.TableFlagsRowBg
		if imgui.BeginTableV("ArchStatsTable", 3, tableFlags, imgui.NewVec2(0, 0), 0) {
			imgui.TableSetupColumn("Archetype ID")
			imgui.TableSetupColumn("Components")
			imgui.TableSetupColumn("Entity Count")
			imgui.TableHeadersRow()

			for _, arch := range stats.ArchetypeBreakdown {
				imgui.TableNextRow()
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("0x%X", arch.ID))
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("%d", len(arch.ComponentTypes)))
				imgui.TableNextColumn()
				imgui.Text(fmt.Sprintf("%d", arch.EntityCount))
			}

			imgui.EndTable()
		}
		imgui.TreePop()
	}

	if imgui.TreeNodeStr("Singleton Details") {
		for _, singletonType := range stats.SingletonTypes {
			imgui.BulletText(singletonType)
		}
		imgui.TreePop()
	}

	imgui.End()
}

type FrameTimer struct {
	lastFrameTime time.Time
}

func NewFrameTimer() *FrameTimer {
	return &FrameTimer{
		lastFrameTime: time.Now(),
	}
}

func (ft *FrameTimer) GetDeltaTime() float32 {
	now := time.Now()
	delta := float32(now.Sub(ft.lastFrameTime).Seconds())
	ft.lastFrameTime = now
	return delta
}

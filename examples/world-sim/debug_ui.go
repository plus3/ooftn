package main

import (
	"fmt"
	"sort"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/AllenDang/cimgui-go/implot"
	"github.com/plus3/ooftn/ecs"
	"github.com/plus3/ooftn/ecs/debugui"
)

const fpsHistorySize = 100

type PerformanceChart struct {
	FPSSamples []float32
	Offset     int
}

func NewPerformanceChart() *PerformanceChart {
	return &PerformanceChart{
		FPSSamples: make([]float32, fpsHistorySize),
		Offset:     0,
	}
}

func spawnECSDebugWindow(storage *ecs.Storage) {
	storage.Spawn(debugui.ImguiItem{
		Render: func() {
			var perf *PerformanceMetrics
			if !storage.ReadSingleton(&perf) {
				return
			}

			imgui.SetNextWindowPosV(imgui.NewVec2(10, 10), imgui.CondOnce, imgui.NewVec2(0, 0))
			imgui.SetNextWindowSizeV(imgui.NewVec2(300, 250), imgui.CondOnce)

			if imgui.BeginV("ECS Debug", nil, 0) {
				imgui.Text(fmt.Sprintf("Avg FPS: %.1f", perf.AvgFPS))
				imgui.Text(fmt.Sprintf("Avg Frame Time: %.2f ms", perf.AvgFrameTime))
				imgui.Text(fmt.Sprintf("Min/Max: %.2f / %.2f ms", perf.MinFrameTime, perf.MaxFrameTime))
				imgui.Separator()
				imgui.Text(fmt.Sprintf("Update Time: %.2f ms", perf.UpdateTime*1000))
				imgui.Text(fmt.Sprintf("Render Time: %.2f ms", perf.RenderTime*1000))
				imgui.Separator()
				imgui.Text(fmt.Sprintf("Entity Count: %d", perf.EntityCount))
				imgui.Text(fmt.Sprintf("Archetype Count: %d", perf.ArchetypeCount))

				imgui.End()
			}
		},
	})
}

func spawnSimulationStatsWindow(storage *ecs.Storage) {
	storage.Spawn(debugui.ImguiItem{
		Render: func() {
			var sim *SimulationMetrics
			var gameTime *GameTime
			var worldConfig *WorldConfig

			if !storage.ReadSingleton(&sim) {
				return
			}
			storage.ReadSingleton(&gameTime)
			storage.ReadSingleton(&worldConfig)

			imgui.SetNextWindowPosV(imgui.NewVec2(10, 270), imgui.CondOnce, imgui.NewVec2(0, 0))
			imgui.SetNextWindowSizeV(imgui.NewVec2(300, 220), imgui.CondOnce)

			if imgui.BeginV("Simulation Stats", nil, 0) {
				if gameTime != nil {
					imgui.Text(fmt.Sprintf("Day: %d", gameTime.CurrentDay))
					imgui.Text(fmt.Sprintf("Time: %.1f", gameTime.Elapsed))
					imgui.Separator()
				}

				imgui.Text(fmt.Sprintf("Total Population: %d", sim.TotalPopulation))
				imgui.Text(fmt.Sprintf("Active Tasks: %d", sim.ActiveTasks))
				imgui.Text(fmt.Sprintf("Colonies: %d", sim.ColonyCount))
				imgui.Separator()
				imgui.Text(fmt.Sprintf("Resources: %d nodes", sim.ResourceCount))
				imgui.Text(fmt.Sprintf("Total Resources: %d", sim.TotalResources))

				if worldConfig != nil {
					imgui.Separator()
					imgui.Text(fmt.Sprintf("World Size: %dx%d", worldConfig.Width, worldConfig.Height))
					imgui.Text(fmt.Sprintf("Seed: %d", worldConfig.Seed))
				}

				imgui.End()
			}
		},
	})
}

func spawnColonyInfoWindow(storage *ecs.Storage) {
	storage.Spawn(debugui.ImguiItem{
		Render: func() {
			imgui.SetNextWindowPosV(imgui.NewVec2(320, 10), imgui.CondOnce, imgui.NewVec2(0, 0))
			imgui.SetNextWindowSizeV(imgui.NewVec2(350, 400), imgui.CondOnce)

			if imgui.BeginV("Colony Info", nil, 0) {
				colonies := ecs.NewView[struct {
					ecs.EntityId
					*Colony
					*ColonyResources
					*ColonyTraits
				}](storage)

				for colony := range colonies.Iter() {
					color := imgui.NewVec4(
						float32(colony.Colony.Color[0])/255.0,
						float32(colony.Colony.Color[1])/255.0,
						float32(colony.Colony.Color[2])/255.0,
						1.0,
					)

					imgui.PushStyleColorVec4(imgui.ColText, color)
					imgui.Text(fmt.Sprintf("■ Colony %d", colony.EntityId&0xFFFFFFFF))
					imgui.PopStyleColor()

					imgui.Indent()
					imgui.Text(fmt.Sprintf("Population: %d", colony.Colony.Population))
					imgui.Text(fmt.Sprintf("Food: %d | Wood: %d | Stone: %d",
						colony.ColonyResources.Food,
						colony.ColonyResources.Wood,
						colony.ColonyResources.Stone))

					imgui.Text("Traits:")
					imgui.Indent()
					imgui.Text(fmt.Sprintf("Aggression: %.2f", colony.ColonyTraits.Aggression))
					imgui.Text(fmt.Sprintf("Expansion: %.2f", colony.ColonyTraits.Expansion))
					imgui.Text(fmt.Sprintf("Industry: %.2f", colony.ColonyTraits.Industry))
					imgui.Text(fmt.Sprintf("Reproduction: %.2f", colony.ColonyTraits.Reproduction))
					imgui.Unindent()

					colonists := ecs.NewView[struct {
						*ColonyMember
						*Role
					}](storage)

					roleCount := make(map[RoleType]int)
					for colonist := range colonists.Iter() {
						if colonist.ColonyMember.ColonyRef != nil {
							colonyId, valid := storage.ResolveEntityRef(colonist.ColonyMember.ColonyRef)
							if valid && colonyId == colony.EntityId {
								roleCount[colonist.Role.Type]++
							}
						}
					}

					imgui.Text("Roles:")
					imgui.Indent()
					imgui.Text(fmt.Sprintf("Gatherers: %d", roleCount[RoleGatherer]))
					imgui.Text(fmt.Sprintf("Builders: %d", roleCount[RoleBuilder]))
					imgui.Text(fmt.Sprintf("Farmers: %d", roleCount[RoleFarmer]))
					imgui.Text(fmt.Sprintf("Fighters: %d", roleCount[RoleFighter]))
					imgui.Text(fmt.Sprintf("Idle: %d", roleCount[RoleIdle]))
					imgui.Unindent()

					imgui.Unindent()
					imgui.Separator()
				}

				imgui.End()
			}
		},
	})
}

func spawnSystemPerformanceWindow(storage *ecs.Storage, scheduler *ecs.Scheduler) {
	storage.Spawn(debugui.ImguiItem{
		Render: func() {
			stats := scheduler.GetStats()

			imgui.SetNextWindowPosV(imgui.NewVec2(680, 10), imgui.CondOnce, imgui.NewVec2(0, 0))
			imgui.SetNextWindowSizeV(imgui.NewVec2(400, 400), imgui.CondOnce)

			if imgui.BeginV("System Performance", nil, 0) {
				imgui.Text(fmt.Sprintf("System Count: %d", stats.SystemCount))
				imgui.Separator()

				const tableFlags = imgui.TableFlagsBorders | imgui.TableFlagsRowBg | imgui.TableFlagsSortable | imgui.TableFlagsSizingFixedFit
				if imgui.BeginTableV("Systems", 4, tableFlags, imgui.NewVec2(0, 0), 0) {
					imgui.TableSetupColumn("Name")
					imgui.TableSetupColumn("Avg (ms)")
					imgui.TableSetupColumn("Min (ms)")
					imgui.TableSetupColumn("Max (ms)")
					imgui.TableHeadersRow()

					systems := stats.Systems
					if sortSpecs := imgui.TableGetSortSpecs(); sortSpecs.SpecsCount() > 0 {
						sort.Slice(systems, func(i, j int) bool {
							spec := sortSpecs.Specs()
							left := systems[i]
							right := systems[j]

							var less bool
							switch spec.ColumnIndex() {
							case 0: // Name
								less = left.Name < right.Name
							case 1: // Avg (ms)
								less = left.AvgDuration < right.AvgDuration
							case 2: // Min (ms)
								less = left.MinDuration < right.MinDuration
							case 3: // Max (ms)
								less = left.MaxDuration < right.MaxDuration
							}

							if spec.SortDirection() == imgui.SortDirectionDescending {
								return !less
							}
							return less
						})
					}

					for _, sys := range systems {
						imgui.TableNextRow()

						imgui.TableNextColumn()
						imgui.Text(sys.Name)

						imgui.TableNextColumn()
						imgui.Text(fmt.Sprintf("%.3f", float64(sys.AvgDuration.Microseconds())/1000.0))

						imgui.TableNextColumn()
						imgui.Text(fmt.Sprintf("%.3f", float64(sys.MinDuration.Microseconds())/1000.0))

						imgui.TableNextColumn()
						imgui.Text(fmt.Sprintf("%.3f", float64(sys.MaxDuration.Microseconds())/1000.0))
					}
					imgui.EndTable()
				}

				imgui.End()
			}
		},
	})
}

func spawnPerformanceChartWindow(storage *ecs.Storage) {
	storage.Spawn(debugui.ImguiItem{
		Render: func() {
			var perf *PerformanceMetrics
			var chartData *PerformanceChart

			if !storage.ReadSingleton(&perf) {
				return
			}
			if !storage.ReadSingleton(&chartData) {
				return
			}

			// Add current values to history
			chartData.FPSSamples[chartData.Offset] = float32(perf.AvgFPS)
			chartData.Offset = (chartData.Offset + 1) % fpsHistorySize

			plotSamples := make([]float32, fpsHistorySize)
			copy(plotSamples, chartData.FPSSamples[chartData.Offset:])
			copy(plotSamples[fpsHistorySize-chartData.Offset:], chartData.FPSSamples[:chartData.Offset])

			imgui.SetNextWindowPosV(imgui.NewVec2(10, 500), imgui.CondOnce, imgui.NewVec2(0, 0))
			imgui.SetNextWindowSizeV(imgui.NewVec2(400, 200), imgui.CondOnce)

			if imgui.BeginV("Performance Charts", nil, 0) {
				if implot.BeginPlotV("FPS", imgui.NewVec2(-1, 0), 0) {
					implot.SetupAxesV("Frame", "FPS", 0, implot.AxisFlagsAutoFit)
					implot.PlotLineFloatPtrInt("FPS", &plotSamples[0], int32(len(plotSamples)))
					implot.EndPlot()
				}
				imgui.End()
			}
		},
	})
}

func initDebugUI(storage *ecs.Storage, scheduler *ecs.Scheduler) {
	storage.AddSingleton(NewPerformanceChart())

	spawnECSDebugWindow(storage)
	spawnSimulationStatsWindow(storage)
	spawnColonyInfoWindow(storage)
	spawnSystemPerformanceWindow(storage, scheduler)
	spawnPerformanceChartWindow(storage)
}

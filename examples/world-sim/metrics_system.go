package main

import (
	"time"

	"github.com/plus3/ooftn/ecs"
)

type MetricsSystem struct {
	Performance ecs.Singleton[PerformanceMetrics]
	Simulation  ecs.Singleton[SimulationMetrics]

	Colonists ecs.Query[struct {
		ecs.EntityId
		*ColonyMember
		*Task
	}]
	Colonies  ecs.Query[struct{ *Colony }]
	Resources ecs.Query[struct{ *Resource }]
	Dead      ecs.Query[struct{ *Dead }]

	lastTime          time.Time
	storageStatsCache *ecs.StorageStats
}

func (m *MetricsSystem) Execute(frame *ecs.UpdateFrame) {
	now := time.Now()
	if !m.lastTime.IsZero() {
		frameTime := float32(now.Sub(m.lastTime).Seconds())
		perf := m.Performance.Get()

		perf.FrameTime = frameTime
		if frameTime > 0 {
			perf.FPS = 1.0 / frameTime
		}

		if len(perf.LastFrameSamples) >= 60 {
			perf.LastFrameSamples = perf.LastFrameSamples[1:]
		}
		perf.LastFrameSamples = append(perf.LastFrameSamples, frameTime*1000)

		sum := float32(0)
		min := float32(999999)
		max := float32(0)
		for _, sample := range perf.LastFrameSamples {
			sum += sample
			if sample < min {
				min = sample
			}
			if sample > max {
				max = sample
			}
		}
		if len(perf.LastFrameSamples) > 0 {
			perf.AvgFrameTime = sum / float32(len(perf.LastFrameSamples))
			perf.AvgFPS = 1000.0 / perf.AvgFrameTime
			perf.MinFrameTime = min
			perf.MaxFrameTime = max
		}

		if m.storageStatsCache == nil || perf.FrameTime > 0.1 {
			m.storageStatsCache = frame.Storage.CollectStats()
		}

		perf.EntityCount = m.storageStatsCache.TotalEntityCount
		perf.ArchetypeCount = m.storageStatsCache.ArchetypeCount
	}
	m.lastTime = now

	colonistCount := 0
	activeTasks := 0
	for colonist := range m.Colonists.Iter() {
		colonistCount++
		if colonist.Task.Type != TaskIdle {
			activeTasks++
		}
	}

	colonyCount := 0
	for range m.Colonies.Iter() {
		colonyCount++
	}

	resourceCount := 0
	totalResources := 0
	for resource := range m.Resources.Iter() {
		resourceCount++
		totalResources += resource.Resource.Amount
	}

	deadCount := 0
	for range m.Dead.Iter() {
		deadCount++
	}

	sim := m.Simulation.Get()
	sim.TotalPopulation = colonistCount
	sim.ActiveTasks = activeTasks
	sim.ColonyCount = colonyCount
	sim.ResourceCount = resourceCount
	sim.TotalResources = totalResources
	sim.DeadCount = deadCount
}

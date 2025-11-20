package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/plus3/ooftn/ecs"
)

// These will be updated by the generator's flags
const (
	componentCount = 250
	systemCount    = 50
)

func main() {
	duration := flag.Duration("duration", 10*time.Second, "The total duration the test should run for.")
	entityCount := flag.Int("entities", 10000, "The initial number of entities to create.")
	gcPauseMetrics := flag.Bool("gc-pause-metrics", false, "Enable detailed GC pause metrics in the report.")
	flag.Parse()

	log.Println("Starting ECS stress test...")

	// 1. Setup Registry, Storage, and Scheduler
	registry := ecs.NewComponentRegistry()
	RegisterAllGeneratedComponents(registry)
	storage := ecs.NewStorage(registry)
	scheduler := ecs.NewScheduler(storage)
	RegisterAllGeneratedSystems(scheduler)

	// 2. Populate Storage with initial entities
	log.Printf("Populating storage with %d entities...\n", *entityCount)
	for i := 0; i < *entityCount; i++ {
		// Spawn an entity with 1 to 5 random components
		numComponents := rand.Intn(5) + 1
		SpawnRandomEntity(storage, numComponents)
	}
	log.Println("Population complete.")

	// 3. Run the simulation loop
	report := &Report{
		Duration:       *duration,
		Entities:       *entityCount,
		Components:     componentCount,
		Systems:        systemCount,
		GCPauseMetrics: *gcPauseMetrics,
		UpdateTime: Stats{
			Samples: make([]time.Duration, 0),
		},
	}

	runtime.ReadMemStats(&report.MemStatsStart)

	log.Printf("Running simulation for %s...\n", *duration)
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	startTime := time.Now()
	var totalUpdates int64
	lastFrameTime := time.Now()

Loop:
	for {
		select {
		case <-ctx.Done():
			break Loop
		default:
			deltaTime := time.Since(lastFrameTime)
			lastFrameTime = time.Now()

			updateStart := time.Now()
			scheduler.Once(float64(deltaTime) / float64(time.Second))
			updateDuration := time.Since(updateStart)

			report.UpdateTime.Samples = append(report.UpdateTime.Samples, updateDuration)
			totalUpdates++
		}
	}

	report.TotalTime = time.Since(startTime)
	report.TotalUpdates = totalUpdates
	report.UpdateTime.Finalize()
	runtime.ReadMemStats(&report.MemStatsEnd)

	log.Println("Simulation finished.")

	// 4. Generate Report to Console
	fmt.Println("\n\n--- Stress Test Report ---")
	if err := report.Generate(os.Stdout); err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}
	fmt.Println("--- End of Report ---")

	log.Println("Stress test complete.")
}

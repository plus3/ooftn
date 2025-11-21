package ecs_test

import (
	"fmt"

	"github.com/plus3/ooftn/ecs"
)

type GameConfig struct {
	MaxPlayers int
	Difficulty string
}

type GameScore struct {
	Points int
	Level  int
}

// ExampleNewSingleton demonstrates creating and accessing singleton components.
// Singletons are global components not associated with any entity, useful for
// game state, configuration, or other application-wide data.
func ExampleNewSingleton() {
	registry := ecs.NewComponentRegistry()
	storage := ecs.NewStorage(registry)

	// Create singleton with initializer
	config := ecs.NewSingleton[GameConfig](storage, GameConfig{
		MaxPlayers: 4,
		Difficulty: "Normal",
	})

	fmt.Printf("Config: %d players, %s difficulty\n", config.Get().MaxPlayers, config.Get().Difficulty)

	// Modify the singleton
	config.Get().Difficulty = "Hard"
	fmt.Printf("Updated difficulty: %s\n", config.Get().Difficulty)

	// Create another reference to the same singleton
	sameConfig := ecs.NewSingleton[GameConfig](storage)
	fmt.Printf("Same config: %s difficulty\n", sameConfig.Get().Difficulty)

	// Output:
	// Config: 4 players, Normal difficulty
	// Updated difficulty: Hard
	// Same config: Hard difficulty
}

// ExampleSingleton_multipleReferences shows that multiple Singleton instances
// reference the same underlying data.
func ExampleSingleton_multipleReferences() {
	registry := ecs.NewComponentRegistry()
	storage := ecs.NewStorage(registry)

	// Create first singleton reference
	score1 := ecs.NewSingleton[GameScore](storage, GameScore{Points: 0, Level: 1})
	fmt.Printf("Score1: %d points, Level %d\n", score1.Get().Points, score1.Get().Level)

	// Modify via first reference
	score1.Get().Points = 100
	score1.Get().Level = 2

	// Create second reference to same singleton
	score2 := ecs.NewSingleton[GameScore](storage)
	fmt.Printf("Score2: %d points, Level %d\n", score2.Get().Points, score2.Get().Level)

	// Both references point to the same data
	score2.Get().Points = 250
	fmt.Printf("Score1 after Score2 update: %d points\n", score1.Get().Points)

	// Output:
	// Score1: 0 points, Level 1
	// Score2: 100 points, Level 2
	// Score1 after Score2 update: 250 points
}

// ExampleStorage_ReadSingleton demonstrates the ReadSingleton API for
// convenient singleton access outside of systems.
func ExampleStorage_ReadSingleton() {
	registry := ecs.NewComponentRegistry()
	storage := ecs.NewStorage(registry)

	// Create singleton
	ecs.NewSingleton[GameConfig](storage, GameConfig{
		MaxPlayers: 8,
		Difficulty: "Expert",
	})

	// Read singleton using pointer pattern
	var config *GameConfig
	if storage.ReadSingleton(&config) {
		fmt.Printf("Game: %d players, %s mode\n", config.MaxPlayers, config.Difficulty)
	}

	// Try reading non-existent singleton
	var score *GameScore
	if storage.ReadSingleton(&score) {
		fmt.Println("Score exists")
	} else {
		fmt.Println("Score not found")
	}

	// Output:
	// Game: 8 players, Expert mode
	// Score not found
}

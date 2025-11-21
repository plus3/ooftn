package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/plus3/ooftn/ecs"
	debugui_ebiten "github.com/plus3/ooftn/ecs/debugui/ebiten"
)

type Position struct {
	X, Y float32
}

type GridPosition struct {
	X, Y int
}

type Velocity struct {
	DX, DY float32
}

type Sprite struct {
	Color [3]uint8
	Scale float32
	Shape ShapeType
}

type ShapeType int

const (
	ShapeCircle ShapeType = iota
	ShapeSquare
	ShapeTriangle
)

type Name string

type ColonyMember struct {
	ColonyRef *ecs.EntityRef
}

type Colony struct {
	Name       string
	Color      [3]uint8
	Population int
	Territory  [][2]int
}

type ColonyResources struct {
	Food  int
	Wood  int
	Stone int
}

type ColonyTraits struct {
	Aggression   float32
	Expansion    float32
	Industry     float32
	Reproduction float32
}

type Role struct {
	Type  RoleType
	Skill float32
}

type RoleType int

const (
	RoleGatherer RoleType = iota
	RoleBuilder
	RoleFarmer
	RoleFighter
	RoleBreeder
	RoleIdle
)

type Stats struct {
	Health    int
	MaxHealth int
	Hunger    int
	MaxHunger int
	Age       float32
	Speed     float32
}

type Inventory struct {
	Food  int
	Wood  int
	Stone int
}

type Task struct {
	Type      TaskType
	Target    *ecs.EntityRef
	Progress  float32
	Duration  float32
	TargetPos [2]int
}

type TaskType int

const (
	TaskGather TaskType = iota
	TaskBuild
	TaskEat
	TaskReproduce
	TaskWander
	TaskReturn
	TaskIdle
)

type Path struct {
	Waypoints  [][2]int
	CurrentIdx int
}

type Resource struct {
	Type         ResourceType
	Amount       int
	MaxAmount    int
	RegrowthRate float32
	RegrowthTime float32
}

type ResourceType int

const (
	ResourceTree ResourceType = iota
	ResourceRock
	ResourceBerryBush
)

type Structure struct {
	Type          StructureType
	BuildProgress float32
	Built         bool
	Owner         *ecs.EntityRef
}

type StructureType int

const (
	StructureHouse StructureType = iota
	StructureStorage
	StructureFarm
)

type Producer struct {
	ResourceType ResourceType
	Rate         float32
	Accumulator  float32
}

type Lifespan struct {
	BirthTime float32
	MaxAge    float32
}

type Fertile struct {
	LastBirth float32
	Cooldown  float32
}

type Dead struct{}

type Combat struct {
	AttackPower int
	AttackSpeed float32
	AttackTimer float32
	Target      *ecs.EntityRef
}

type WorldConfig struct {
	Width    int
	Height   int
	CellSize int
	Seed     int64
}

type GameTime struct {
	Elapsed    float32
	DayLength  float32
	CurrentDay int
}

type Camera struct {
	X       float32
	Y       float32
	Zoom    float32
	ScreenW int
	ScreenH int
}

type InputState struct {
	LastMouseX    int
	LastMouseY    int
	Dragging      bool
	DragStartX    float32
	DragStartY    float32
	PrevMouseLeft bool
}

type SpatialGrid struct {
	CellSize int
	Cells    map[[2]int][]ecs.EntityId
}

type RenderLayer int

const (
	LayerTerrain RenderLayer = iota
	LayerResources
	LayerStructures
	LayerColonists
	LayerUI
)

type PerformanceMetrics struct {
	FPS              float32
	FrameTime        float32
	UpdateTime       float32
	RenderTime       float32
	EntityCount      int
	ArchetypeCount   int
	LastFrameSamples []float32
	AvgFPS           float32
	AvgFrameTime     float32
	MinFrameTime     float32
	MaxFrameTime     float32
}

type SimulationMetrics struct {
	TotalPopulation int
	TotalResources  int
	ActiveTasks     int
	ColonyCount     int
	ResourceCount   int
	DeadCount       int
}

type Screen struct {
	*ebiten.Image
}

type Game struct {
	Storage         *ecs.Storage
	Scheduler       *ecs.Scheduler
	RenderScheduler *ecs.Scheduler
	RenderSystem    *RenderSystem
	ImguiBackend    *ecs.Singleton[debugui_ebiten.ImguiBackend]
	Screen          *ecs.Singleton[Screen]
}

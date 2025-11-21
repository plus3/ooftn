package main

import (
	"log"
	"math/rand/v2"
	"net/http"
	_ "net/http/pprof"

	ebitenbackend "github.com/AllenDang/cimgui-go/backend/ebiten-backend"
	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/AllenDang/cimgui-go/implot"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/plus3/ooftn/ecs"
	"github.com/plus3/ooftn/ecs/debugui"
	debugui_ebiten "github.com/plus3/ooftn/ecs/debugui/ebiten"
)

const (
	ScreenWidth  = 1280
	ScreenHeight = 720
	WorldWidth   = 300
	WorldHeight  = 300
	CellSize     = 16
)

var pastelColors = [][3]uint8{
	{255, 179, 186},
	{179, 229, 252},
	{255, 223, 186},
	{186, 255, 201},
	{255, 200, 221},
	{186, 225, 255},
	{255, 255, 186},
	{217, 186, 255},
}

func main() {
	// Start pprof HTTP server for profiling
	go func() {
		log.Println("Starting pprof server on http://localhost:6060/debug/pprof/")
		log.Println("  CPU profile: http://localhost:6060/debug/pprof/profile?seconds=30")
		log.Println("  Heap profile: http://localhost:6060/debug/pprof/heap")
		log.Println("  Goroutine profile: http://localhost:6060/debug/pprof/goroutine")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Printf("pprof server error: %v", err)
		}
	}()

	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("World Simulator - ECS Example")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	imguiBackend := ebitenbackend.NewEbitenBackend()
	imguiBackend.CreateWindow("backseas", ScreenWidth, ScreenHeight)
	imgui.CurrentIO().SetIniFilename("")
	implot.CreateContext()
	defer implot.DestroyContext()

	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[GridPosition](registry)
	ecs.RegisterComponent[PreviousGridPosition](registry)
	ecs.RegisterComponent[Velocity](registry)
	ecs.RegisterComponent[Sprite](registry)
	ecs.RegisterComponent[Name](registry)
	ecs.RegisterComponent[ColonyMember](registry)
	ecs.RegisterComponent[Colony](registry)
	ecs.RegisterComponent[ColonyResources](registry)
	ecs.RegisterComponent[ColonyTraits](registry)
	ecs.RegisterComponent[Role](registry)
	ecs.RegisterComponent[Stats](registry)
	ecs.RegisterComponent[Inventory](registry)
	ecs.RegisterComponent[Task](registry)
	ecs.RegisterComponent[Path](registry)
	ecs.RegisterComponent[Resource](registry)
	ecs.RegisterComponent[Structure](registry)
	ecs.RegisterComponent[Producer](registry)
	ecs.RegisterComponent[Lifespan](registry)
	ecs.RegisterComponent[Fertile](registry)
	ecs.RegisterComponent[Dead](registry)
	ecs.RegisterComponent[PendingDeaths](registry)
	ecs.RegisterComponent[Combat](registry)

	ecs.RegisterComponent[debugui_ebiten.ImguiBackend](registry)
	ecs.RegisterComponent[debugui.ImguiItem](registry)
	ecs.RegisterComponent[debugui.ImguiInputState](registry)
	ecs.RegisterComponent[PerformanceMetrics](registry)
	ecs.RegisterComponent[SimulationMetrics](registry)
	ecs.RegisterComponent[PerformanceChart](registry)
	ecs.RegisterComponent[PauseState](registry)

	storage := ecs.NewStorage(registry)

	debugui.RegisterDebugUIComponents(registry)
	debugui.SpawnDebugUI(storage)

	ecs.NewSingleton[debugui_ebiten.ImguiBackend](storage, debugui_ebiten.ImguiBackend{
		EbitenBackend: imguiBackend,
	})

	ecs.NewSingleton[WorldConfig](storage, WorldConfig{
		Width:    WorldWidth,
		Height:   WorldHeight,
		CellSize: CellSize,
		Seed:     rand.Int64(),
	})

	ecs.NewSingleton[GameTime](storage, GameTime{
		Elapsed:    0,
		DayLength:  60,
		CurrentDay: 0,
	})

	ecs.NewSingleton[Camera](storage, Camera{
		X:       float32(WorldWidth) / 2,
		Y:       float32(WorldHeight) / 2,
		Zoom:    2.0,
		ScreenW: ScreenWidth,
		ScreenH: ScreenHeight,
	})

	ecs.NewSingleton[InputState](storage, InputState{})
	ecs.NewSingleton[SpatialGrid](storage, SpatialGrid{
		CellSize: 10,
		Cells:    make(map[[2]int][]ecs.EntityId),
	})
	ecs.NewSingleton[FighterGrid](storage, FighterGrid{
		CellSize: 10,
		Cells:    make(map[[2]int][]ecs.EntityId),
	})

	ecs.NewSingleton[PendingDeaths](storage, PendingDeaths{
		pending: make(map[ecs.EntityId]bool),
	})

	ecs.NewSingleton[PerformanceMetrics](storage, PerformanceMetrics{
		LastFrameSamples: make([]float32, 0, 60),
	})
	ecs.NewSingleton[SimulationMetrics](storage, SimulationMetrics{})
	ecs.NewSingleton[PauseState](storage, PauseState{
		Paused:        false,
		StepRequested: false,
	})

	initWorld(storage)

	scheduler := ecs.NewScheduler(storage)
	scheduler.Register(&PauseControlSystem{})
	scheduler.Register(&ClearPendingDeathsSystem{})
	scheduler.Register(&MetricsSystem{})
	scheduler.Register(&debugui.ImguiSystem{})
	scheduler.Register(&TimeSystem{})
	scheduler.Register(&ColonyManagementSystem{})
	scheduler.Register(&MovementSystem{})
	scheduler.Register(&SpatialGridSystem{})
	scheduler.Register(&FighterGridSystem{})
	scheduler.Register(&TaskAssignmentSystem{})
	scheduler.Register(&WorkSystem{})
	scheduler.Register(&HungerSystem{})
	scheduler.Register(&ReproductionSystem{})
	scheduler.Register(&CombatSystem{})
	scheduler.Register(&LifespanSystem{})
	scheduler.Register(&ResourceRegrowthSystem{})
	scheduler.Register(&DeathSystem{})
	scheduler.Register(&CameraControlSystem{})

	renderScheduler := ecs.NewScheduler(storage)
	renderSystem := &RenderSystem{}
	renderScheduler.Register(renderSystem)

	initDebugUI(storage, scheduler)

	game := &Game{
		Storage:         storage,
		Scheduler:       scheduler,
		RenderScheduler: renderScheduler,
		RenderSystem:    renderSystem,
		ImguiBackend:    ecs.NewSingleton[debugui_ebiten.ImguiBackend](storage),
		Screen:          ecs.NewSingleton[Screen](storage),
	}

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyQ) || ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	var perf *PerformanceMetrics
	g.Storage.ReadSingleton(&perf)

	// Check if we're advancing time while paused - if so, run multiple ticks per frame
	var pauseState *PauseState
	g.Storage.ReadSingleton(&pauseState)

	ticksPerFrame := 1
	if pauseState != nil && pauseState.TimeToAdvance > 0 {
		// Run 10 ticks per frame when advancing time (makes it 10x faster)
		ticksPerFrame = 10
	}

	g.ImguiBackend.Get().BeginFrame()

	for i := 0; i < ticksPerFrame; i++ {
		// Only render UI on the last tick
		if pauseState != nil {
			pauseState.SkipUIRender = (i < ticksPerFrame-1)
		}
		g.Scheduler.Once(1.0 / 60.0)
	}

	if perf != nil {
		// Calculate actual update time from scheduler stats
		stats := g.Scheduler.GetStats()
		totalUpdateTime := float32(0)
		for _, sys := range stats.Systems {
			totalUpdateTime += float32(sys.AvgDuration.Seconds())
		}
		perf.UpdateTime = totalUpdateTime
	}

	g.ImguiBackend.Get().EndFrame()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	var camera *Camera
	if g.Storage.ReadSingleton(&camera) {
		camera.ScreenW = screen.Bounds().Dx()
		camera.ScreenH = screen.Bounds().Dy()
	}

	g.Screen.Get().Image = screen

	g.RenderScheduler.Once(0)

	var perf *PerformanceMetrics
	if g.Storage.ReadSingleton(&perf) {
		// Calculate actual render time from render scheduler stats
		stats := g.RenderScheduler.GetStats()
		totalRenderTime := float32(0)
		for _, sys := range stats.Systems {
			totalRenderTime += float32(sys.AvgDuration.Seconds())
		}
		perf.RenderTime = totalRenderTime
	}

	g.ImguiBackend.Get().Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.ImguiBackend.Get().Layout(outsideWidth, outsideHeight)
	return outsideWidth, outsideHeight
}

func initWorld(storage *ecs.Storage) {
	spawnResources(storage)

	colonyPositions := [][2]int{
		{50, 50},
		{250, 50},
		{50, 250},
		{250, 250},
	}

	for i, pos := range colonyPositions {
		color := pastelColors[i%len(pastelColors)]
		spawnColony(storage, pos[0], pos[1], color)
	}
}

func spawnResources(storage *ecs.Storage) {
	for i := 0; i < 1800; i++ {
		x := rand.IntN(WorldWidth)
		y := rand.IntN(WorldHeight)

		var resType ResourceType
		var color [3]uint8
		var amount int

		roll := rand.Float32()
		if roll < 0.5 {
			resType = ResourceTree
			color = [3]uint8{144, 238, 144}
			amount = 20
		} else if roll < 0.8 {
			resType = ResourceRock
			color = [3]uint8{169, 169, 169}
			amount = 15
		} else {
			resType = ResourceBerryBush
			color = [3]uint8{255, 182, 193}
			amount = 10
		}

		storage.Spawn(
			Position{X: float32(x), Y: float32(y)},
			GridPosition{X: x, Y: y},
			Sprite{
				Color: color,
				Scale: 0.6,
				Shape: ShapeSquare,
			},
			Resource{
				Type:         resType,
				Amount:       amount,
				MaxAmount:    amount,
				RegrowthRate: 0.01,
			},
		)
	}
}

func spawnColony(storage *ecs.Storage, x, y int, color [3]uint8) ecs.EntityId {
	colonyId := storage.Spawn(
		Position{X: float32(x), Y: float32(y)},
		GridPosition{X: x, Y: y},
		Colony{
			Name:       "Colony",
			Color:      color,
			Population: 0,
			Territory:  [][2]int{},
		},
		ColonyResources{
			Food:  50,
			Wood:  20,
			Stone: 10,
		},
		ColonyTraits{
			Aggression:   rand.Float32()*0.5 + 0.1,
			Expansion:    rand.Float32()*0.5 + 0.3,
			Industry:     rand.Float32()*0.5 + 0.3,
			Reproduction: rand.Float32()*0.3 + 0.1,
		},
	)

	for i := 0; i < 5; i++ {
		spawnColonistDirect(storage, colonyId, x+rand.IntN(5)-2, y+rand.IntN(5)-2)
	}

	return colonyId
}

func spawnColonistDirect(storage *ecs.Storage, colonyId ecs.EntityId, x, y int) {
	colonyRef := storage.CreateEntityRef(colonyId)

	var colonyColor [3]uint8
	if colony := ecs.ReadComponent[Colony](storage, colonyId); colony != nil {
		colonyColor = colony.Color
	}

	cellSize := 10
	storage.Spawn(
		Position{X: float32(x), Y: float32(y)},
		GridPosition{X: x, Y: y},
		PreviousGridPosition{X: x, Y: y, CellX: x / cellSize, CellY: y / cellSize},
		Sprite{
			Color: colonyColor,
			Scale: 0.4,
			Shape: ShapeCircle,
		},
		ColonyMember{ColonyRef: colonyRef},
		Role{Type: RoleIdle, Skill: 0.5},
		Stats{
			Health:    100,
			MaxHealth: 100,
			Hunger:    0,
			MaxHunger: 100,
			Age:       0,
			Speed:     10.0,
		},
		Inventory{},
		Task{Type: TaskIdle},
		Lifespan{
			BirthTime: 0,
			MaxAge:    300,
		},
		Fertile{
			LastBirth: 0,
			Cooldown:  30,
		},
		Combat{
			AttackPower: 10,
			AttackSpeed: 0.5,
			AttackTimer: 0,
		},
	)
}

func spawnColonist(frame *ecs.UpdateFrame, colonyId ecs.EntityId, x, y int) {
	colonyRef := frame.Storage.CreateEntityRef(colonyId)

	var colonyColor [3]uint8
	if colony := ecs.ReadComponent[Colony](frame.Storage, colonyId); colony != nil {
		colonyColor = colony.Color
	}

	cellSize := 10
	frame.Commands.Spawn(
		Position{X: float32(x), Y: float32(y)},
		GridPosition{X: x, Y: y},
		PreviousGridPosition{X: x, Y: y, CellX: x / cellSize, CellY: y / cellSize},
		Sprite{
			Color: colonyColor,
			Scale: 0.4,
			Shape: ShapeCircle,
		},
		ColonyMember{ColonyRef: colonyRef},
		Role{Type: RoleIdle, Skill: 0.5},
		Stats{
			Health:    100,
			MaxHealth: 100,
			Hunger:    0,
			MaxHunger: 100,
			Age:       0,
			Speed:     10.0,
		},
		Inventory{},
		Task{Type: TaskIdle},
		Lifespan{
			BirthTime: 0,
			MaxAge:    300,
		},
		Fertile{
			LastBirth: 0,
			Cooldown:  30,
		},
		Combat{
			AttackPower: 10,
			AttackSpeed: 0.5,
			AttackTimer: 0,
		},
	)
}

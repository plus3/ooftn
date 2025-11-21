package main

import (
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/plus3/ooftn/ecs"
)

const (
	ScreenWidth  = 1280
	ScreenHeight = 720
	WorldWidth   = 100
	WorldHeight  = 100
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
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("World Simulator - ECS Example")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[Position](registry)
	ecs.RegisterComponent[GridPosition](registry)
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
	ecs.RegisterComponent[Combat](registry)

	storage := ecs.NewStorage(registry)

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

	initWorld(storage)

	scheduler := ecs.NewScheduler(storage)
	scheduler.Register(&TimeSystem{})
	scheduler.Register(&ColonyManagementSystem{})
	scheduler.Register(&TaskAssignmentSystem{})
	scheduler.Register(&MovementSystem{})
	scheduler.Register(&WorkSystem{})
	scheduler.Register(&HungerSystem{})
	scheduler.Register(&ReproductionSystem{})
	scheduler.Register(&CombatSystem{})
	scheduler.Register(&LifespanSystem{})
	scheduler.Register(&ResourceRegrowthSystem{})
	scheduler.Register(&DeathSystem{})
	scheduler.Register(&CameraControlSystem{})

	renderSystem := &RenderSystem{}
	scheduler.Register(renderSystem)

	game := &Game{
		Storage:      storage,
		Scheduler:    scheduler,
		RenderSystem: renderSystem,
	}

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyQ) || ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	g.Scheduler.Once(1.0 / 60.0)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	var camera *Camera
	if g.Storage.ReadSingleton(&camera) {
		camera.ScreenW = screen.Bounds().Dx()
		camera.ScreenH = screen.Bounds().Dy()
	}

	g.RenderSystem.screen = screen
	g.Scheduler.Once(0)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func initWorld(storage *ecs.Storage) {
	spawnResources(storage)

	colonyPositions := [][2]int{
		{20, 20},
		{80, 20},
		{20, 80},
		{80, 80},
	}

	for i, pos := range colonyPositions {
		color := pastelColors[i%len(pastelColors)]
		spawnColony(storage, pos[0], pos[1], color)
	}
}

func spawnResources(storage *ecs.Storage) {
	for i := 0; i < 200; i++ {
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

	storage.Spawn(
		Position{X: float32(x), Y: float32(y)},
		GridPosition{X: x, Y: y},
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

	frame.Commands.Spawn(
		Position{X: float32(x), Y: float32(y)},
		GridPosition{X: x, Y: y},
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

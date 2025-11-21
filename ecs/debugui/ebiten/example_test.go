package ebiten_test

import (
	ebitenbackend "github.com/AllenDang/cimgui-go/backend/ebiten-backend"
	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/plus3/ooftn/ecs"
	"github.com/plus3/ooftn/ecs/debugui"
	debugui_ebiten "github.com/plus3/ooftn/ecs/debugui/ebiten"
)

// Game implements ebiten.Game and integrates the ECS with ImGui rendering.
type Game struct {
	storage      *ecs.Storage
	scheduler    *ecs.Scheduler
	imguiBackend *ecs.Singleton[debugui_ebiten.ImguiBackend]
}

func (g *Game) Update() error {
	// Begin ImGui frame before executing systems
	g.imguiBackend.Get().BeginFrame()

	// Execute all ECS systems (including ImguiSystem)
	g.scheduler.Once(1.0 / 60.0)

	// End ImGui frame after systems complete
	g.imguiBackend.Get().EndFrame()

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw game content to screen
	// ...

	// Draw ImGui overlay on top
	g.imguiBackend.Get().Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.imguiBackend.Get().Layout(outsideWidth, outsideHeight)
	return outsideWidth, outsideHeight
}

func Example() {
	// Create Ebiten window and ImGui backend
	imguiBackend := ebitenbackend.NewEbitenBackend()
	imguiBackend.CreateWindow("ECS ImGui Example", 1280, 720)
	imgui.CurrentIO().SetIniFilename("") // Disable imgui.ini

	// Set up ECS component registry
	registry := ecs.NewComponentRegistry()
	ecs.RegisterComponent[debugui_ebiten.ImguiBackend](registry)
	ecs.RegisterComponent[debugui.ImguiItem](registry)
	ecs.RegisterComponent[debugui.ImguiInputState](registry)

	// Create ECS storage
	storage := ecs.NewStorage(registry)

	// Register ImGui backend as a singleton
	ecs.NewSingleton[debugui_ebiten.ImguiBackend](storage, debugui_ebiten.ImguiBackend{
		EbitenBackend: imguiBackend,
	})

	// Spawn entities with ImGui render functions
	storage.Spawn(debugui.ImguiItem{
		Render: func() {
			imgui.Begin("Debug Window")
			imgui.Text("Hello from ECS!")
			imgui.End()
		},
	})

	// Create scheduler and register ImguiSystem
	scheduler := ecs.NewScheduler(storage)
	scheduler.Register(&debugui.ImguiSystem{})

	// Create game instance
	game := &Game{
		storage:      storage,
		scheduler:    scheduler,
		imguiBackend: ecs.NewSingleton[debugui_ebiten.ImguiBackend](storage),
	}

	// Run the game
	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}

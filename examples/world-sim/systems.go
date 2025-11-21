package main

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/plus3/ooftn/ecs"
	"github.com/plus3/ooftn/ecs/debugui"
)

type TimeSystem struct {
	GameTime ecs.Singleton[GameTime]
}

func (s *TimeSystem) Execute(frame *ecs.UpdateFrame) {
	time := s.GameTime.Get()
	time.Elapsed += float32(frame.DeltaTime)

	newDay := int(time.Elapsed / time.DayLength)
	if newDay > time.CurrentDay {
		time.CurrentDay = newDay
	}
}

type ColonyManagementSystem struct {
	Colonies ecs.Query[struct {
		ecs.EntityId
		*Colony
		*ColonyResources
		*ColonyTraits
	}]
	Colonists ecs.Query[struct {
		ecs.EntityId
		*ColonyMember
		*Role
		*Task
	}]
}

func (s *ColonyManagementSystem) Execute(frame *ecs.UpdateFrame) {
	for colony := range s.Colonies.Iter() {
		population := 0
		roleCount := make(map[RoleType]int)

		for colonist := range s.Colonists.Iter() {
			if colonist.ColonyMember.ColonyRef != nil {
				colonyId, valid := frame.Storage.ResolveEntityRef(colonist.ColonyMember.ColonyRef)
				if valid && colonyId == colony.EntityId {
					population++
					roleCount[colonist.Role.Type]++
				}
			}
		}

		colony.Colony.Population = population

		if population < 3 {
			continue
		}

		gatherers := roleCount[RoleGatherer]
		builders := roleCount[RoleBuilder]
		farmers := roleCount[RoleFarmer]

		if gatherers < population/3 {
			s.reassignRole(frame, colony.EntityId, RoleGatherer)
		}
		if colony.ColonyResources.Wood > 20 && builders < 2 {
			s.reassignRole(frame, colony.EntityId, RoleBuilder)
		}
		if farmers < 1 && colony.ColonyResources.Food < 20 {
			s.reassignRole(frame, colony.EntityId, RoleFarmer)
		}
	}
}

func (s *ColonyManagementSystem) reassignRole(frame *ecs.UpdateFrame, colonyId ecs.EntityId, newRole RoleType) {
	for colonist := range s.Colonists.Iter() {
		if colonist.ColonyMember.ColonyRef != nil {
			id, valid := frame.Storage.ResolveEntityRef(colonist.ColonyMember.ColonyRef)
			if valid && id == colonyId && colonist.Role.Type == RoleIdle {
				colonist.Role.Type = newRole
				colonist.Task.Type = TaskIdle
				colonist.Task.Target = nil
				break
			}
		}
	}
}

type TaskAssignmentSystem struct {
	Colonists ecs.Query[struct {
		ecs.EntityId
		*ColonyMember
		*Role
		*Task
		*GridPosition
	}]
	Resources ecs.Query[struct {
		ecs.EntityId
		*GridPosition
		*Resource
	}]
	Structures ecs.Query[struct {
		ecs.EntityId
		*GridPosition
		*Structure
	}]
	WorldConfig ecs.Singleton[WorldConfig]
}

func (s *TaskAssignmentSystem) Execute(frame *ecs.UpdateFrame) {
	for colonist := range s.Colonists.Iter() {
		if colonist.Task.Type != TaskIdle {
			continue
		}

		switch colonist.Role.Type {
		case RoleGatherer:
			s.assignGatherTask(frame, colonist)
		case RoleBuilder:
			s.assignBuildTask(frame, colonist)
		case RoleFarmer:
			s.assignFarmTask(frame, colonist)
		default:
			colonist.Task.Type = TaskWander
			colonist.Task.Duration = rand.Float32()*3 + 2
			config := s.WorldConfig.Get()
			colonist.Task.TargetPos = [2]int{
				rand.IntN(config.Width),
				rand.IntN(config.Height),
			}
		}
	}
}

func (s *TaskAssignmentSystem) assignGatherTask(frame *ecs.UpdateFrame, colonist struct {
	ecs.EntityId
	*ColonyMember
	*Role
	*Task
	*GridPosition
}) {
	var closestResource ecs.EntityId
	closestDist := float32(999999)

	for resource := range s.Resources.Iter() {
		if resource.Resource.Amount <= 0 {
			continue
		}

		dx := float32(resource.GridPosition.X - colonist.GridPosition.X)
		dy := float32(resource.GridPosition.Y - colonist.GridPosition.Y)
		dist := dx*dx + dy*dy

		if dist < closestDist {
			closestDist = dist
			closestResource = resource.EntityId
		}
	}

	if closestResource != 0 {
		colonist.Task.Type = TaskGather
		colonist.Task.Target = frame.Storage.CreateEntityRef(closestResource)
		colonist.Task.Duration = 2.0
		colonist.Task.Progress = 0

		if resource := ecs.ReadComponent[GridPosition](frame.Storage, closestResource); resource != nil {
			colonist.Task.TargetPos = [2]int{resource.X, resource.Y}
		}
	}
}

func (s *TaskAssignmentSystem) assignBuildTask(frame *ecs.UpdateFrame, colonist struct {
	ecs.EntityId
	*ColonyMember
	*Role
	*Task
	*GridPosition
}) {
	for structure := range s.Structures.Iter() {
		if structure.Structure.Built {
			continue
		}

		colonist.Task.Type = TaskBuild
		colonist.Task.Target = frame.Storage.CreateEntityRef(structure.EntityId)
		colonist.Task.Duration = 5.0
		colonist.Task.Progress = 0
		colonist.Task.TargetPos = [2]int{structure.GridPosition.X, structure.GridPosition.Y}
		return
	}
}

func (s *TaskAssignmentSystem) assignFarmTask(frame *ecs.UpdateFrame, colonist struct {
	ecs.EntityId
	*ColonyMember
	*Role
	*Task
	*GridPosition
}) {
	colonist.Task.Type = TaskWander
	colonist.Task.Duration = rand.Float32()*2 + 1
	config := s.WorldConfig.Get()
	colonist.Task.TargetPos = [2]int{
		rand.IntN(config.Width),
		rand.IntN(config.Height),
	}
}

type MovementSystem struct {
	Moving ecs.Query[struct {
		*Position
		*GridPosition
		*Task
		*Stats
		Path *Path `ecs:"optional"`
	}]
}

func (s *MovementSystem) Execute(frame *ecs.UpdateFrame) {
	for entity := range s.Moving.Iter() {
		if entity.Task.Type == TaskIdle {
			continue
		}

		targetX := float32(entity.Task.TargetPos[0])
		targetY := float32(entity.Task.TargetPos[1])

		dx := targetX - entity.Position.X
		dy := targetY - entity.Position.Y
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))

		if dist < 0.1 {
			entity.GridPosition.X = entity.Task.TargetPos[0]
			entity.GridPosition.Y = entity.Task.TargetPos[1]
			continue
		}

		speed := entity.Stats.Speed * float32(frame.DeltaTime)
		if dist < speed {
			entity.Position.X = targetX
			entity.Position.Y = targetY
		} else {
			entity.Position.X += (dx / dist) * speed
			entity.Position.Y += (dy / dist) * speed
		}

		entity.GridPosition.X = int(entity.Position.X)
		entity.GridPosition.Y = int(entity.Position.Y)
	}
}

type WorkSystem struct {
	Workers ecs.Query[struct {
		ecs.EntityId
		*Task
		*GridPosition
		*Inventory
		*ColonyMember
	}]
	Resources ecs.Query[struct {
		ecs.EntityId
		*GridPosition
		*Resource
	}]
	Structures ecs.Query[struct {
		ecs.EntityId
		*GridPosition
		*Structure
	}]
	Colonies ecs.Query[struct {
		ecs.EntityId
		*ColonyResources
	}]
}

func (s *WorkSystem) Execute(frame *ecs.UpdateFrame) {
	for worker := range s.Workers.Iter() {
		if worker.Task.Type == TaskIdle || worker.Task.Type == TaskWander {
			continue
		}

		atTarget := worker.GridPosition.X == worker.Task.TargetPos[0] &&
			worker.GridPosition.Y == worker.Task.TargetPos[1]

		if !atTarget {
			continue
		}

		switch worker.Task.Type {
		case TaskGather:
			s.processGathering(frame, worker)
		case TaskBuild:
			s.processBuilding(frame, worker)
		case TaskReturn:
			s.processReturn(frame, worker)
		}
	}
}

func (s *WorkSystem) processGathering(frame *ecs.UpdateFrame, worker struct {
	ecs.EntityId
	*Task
	*GridPosition
	*Inventory
	*ColonyMember
}) {
	worker.Task.Progress += float32(frame.DeltaTime)

	if worker.Task.Progress >= worker.Task.Duration {
		if worker.Task.Target != nil {
			resourceId, valid := frame.Storage.ResolveEntityRef(worker.Task.Target)
			if valid {
				for resource := range s.Resources.Iter() {
					if resource.EntityId == resourceId && resource.Resource.Amount > 0 {
						amount := 5
						if amount > resource.Resource.Amount {
							amount = resource.Resource.Amount
						}
						resource.Resource.Amount -= amount

						switch resource.Resource.Type {
						case ResourceTree:
							worker.Inventory.Wood += amount
						case ResourceRock:
							worker.Inventory.Stone += amount
						case ResourceBerryBush:
							worker.Inventory.Food += amount
						}

						break
					}
				}
			}
		}

		worker.Task.Type = TaskReturn
		worker.Task.Target = worker.ColonyMember.ColonyRef
		worker.Task.Progress = 0
		worker.Task.Duration = 0

		if worker.ColonyMember.ColonyRef != nil {
			colonyId, valid := frame.Storage.ResolveEntityRef(worker.ColonyMember.ColonyRef)
			if valid {
				for colony := range s.Colonies.Iter() {
					if colony.EntityId == colonyId {
						if colonyPos := ecs.ReadComponent[GridPosition](frame.Storage, colonyId); colonyPos != nil {
							worker.Task.TargetPos = [2]int{colonyPos.X, colonyPos.Y}
						}
						break
					}
				}
			}
		}
	}
}

func (s *WorkSystem) processBuilding(frame *ecs.UpdateFrame, worker struct {
	ecs.EntityId
	*Task
	*GridPosition
	*Inventory
	*ColonyMember
}) {
	if worker.Task.Target == nil {
		worker.Task.Type = TaskIdle
		return
	}

	structureId, valid := frame.Storage.ResolveEntityRef(worker.Task.Target)
	if !valid {
		worker.Task.Type = TaskIdle
		return
	}

	for structure := range s.Structures.Iter() {
		if structure.EntityId == structureId {
			structure.Structure.BuildProgress += float32(frame.DeltaTime) / worker.Task.Duration

			if structure.Structure.BuildProgress >= 1.0 {
				structure.Structure.Built = true
				structure.Structure.BuildProgress = 1.0
				worker.Task.Type = TaskIdle
			}
			break
		}
	}
}

func (s *WorkSystem) processReturn(frame *ecs.UpdateFrame, worker struct {
	ecs.EntityId
	*Task
	*GridPosition
	*Inventory
	*ColonyMember
}) {
	if worker.ColonyMember.ColonyRef != nil {
		colonyId, valid := frame.Storage.ResolveEntityRef(worker.ColonyMember.ColonyRef)
		if valid {
			for colony := range s.Colonies.Iter() {
				if colony.EntityId == colonyId {
					colony.ColonyResources.Food += worker.Inventory.Food
					colony.ColonyResources.Wood += worker.Inventory.Wood
					colony.ColonyResources.Stone += worker.Inventory.Stone

					worker.Inventory.Food = 0
					worker.Inventory.Wood = 0
					worker.Inventory.Stone = 0

					worker.Task.Type = TaskIdle
					break
				}
			}
		}
	}
}

type HungerSystem struct {
	Living ecs.Query[struct {
		ecs.EntityId
		*Stats
		*ColonyMember
	}]
	Colonies ecs.Query[struct {
		ecs.EntityId
		*ColonyResources
	}]
}

func (s *HungerSystem) Execute(frame *ecs.UpdateFrame) {
	for entity := range s.Living.Iter() {
		entity.Stats.Hunger += int(float32(frame.DeltaTime) * 2)

		if entity.Stats.Hunger >= entity.Stats.MaxHunger {
			if entity.ColonyMember.ColonyRef != nil {
				colonyId, valid := frame.Storage.ResolveEntityRef(entity.ColonyMember.ColonyRef)
				if valid {
					for colony := range s.Colonies.Iter() {
						if colony.EntityId == colonyId && colony.ColonyResources.Food > 0 {
							colony.ColonyResources.Food--
							entity.Stats.Hunger = 0
							break
						}
					}
				}
			}

			if entity.Stats.Hunger >= entity.Stats.MaxHunger {
				entity.Stats.Health -= 1
				if entity.Stats.Health <= 0 {
					frame.Commands.AddComponent(entity.EntityId, Dead{})
				}
			}
		}
	}
}

type ReproductionSystem struct {
	FertileColonists ecs.Query[struct {
		ecs.EntityId
		*ColonyMember
		*Fertile
		*GridPosition
		*Stats
	}]
	Colonies ecs.Query[struct {
		ecs.EntityId
		*Colony
		*ColonyTraits
	}]
	GameTime ecs.Singleton[GameTime]
}

func (s *ReproductionSystem) Execute(frame *ecs.UpdateFrame) {
	time := s.GameTime.Get()

	for colonist := range s.FertileColonists.Iter() {
		if time.Elapsed-colonist.Fertile.LastBirth < colonist.Fertile.Cooldown {
			continue
		}

		if colonist.Stats.Health < colonist.Stats.MaxHealth/2 {
			continue
		}

		if colonist.ColonyMember.ColonyRef == nil {
			continue
		}

		colonyId, valid := frame.Storage.ResolveEntityRef(colonist.ColonyMember.ColonyRef)
		if !valid {
			continue
		}

		for colony := range s.Colonies.Iter() {
			if colony.EntityId == colonyId {
				if rand.Float32() < colony.ColonyTraits.Reproduction*float32(frame.DeltaTime) {
					spawnColonist(frame, colony.EntityId, colonist.GridPosition.X, colonist.GridPosition.Y)
					colonist.Fertile.LastBirth = time.Elapsed
				}
				break
			}
		}
	}
}

type CombatSystem struct {
	Fighters ecs.Query[struct {
		ecs.EntityId
		*Combat
		*GridPosition
		*Stats
		*ColonyMember
	}]
}

func (s *CombatSystem) Execute(frame *ecs.UpdateFrame) {
	fighters := make([]struct {
		ecs.EntityId
		*Combat
		*GridPosition
		*Stats
		*ColonyMember
	}, 0)

	for fighter := range s.Fighters.Iter() {
		fighters = append(fighters, fighter)
	}

	for i := range fighters {
		for j := i + 1; j < len(fighters); j++ {
			f1 := fighters[i]
			f2 := fighters[j]

			col1, valid1 := frame.Storage.ResolveEntityRef(f1.ColonyMember.ColonyRef)
			col2, valid2 := frame.Storage.ResolveEntityRef(f2.ColonyMember.ColonyRef)

			if !valid1 || !valid2 || col1 == col2 {
				continue
			}

			dx := f1.GridPosition.X - f2.GridPosition.X
			dy := f1.GridPosition.Y - f2.GridPosition.Y
			distSq := dx*dx + dy*dy

			if distSq <= 4 {
				f1.Combat.AttackTimer += float32(frame.DeltaTime)
				f2.Combat.AttackTimer += float32(frame.DeltaTime)

				if f1.Combat.AttackTimer >= 1.0/f1.Combat.AttackSpeed {
					f2.Stats.Health -= f1.Combat.AttackPower
					f1.Combat.AttackTimer = 0
					if f2.Stats.Health <= 0 {
						frame.Commands.AddComponent(f2.EntityId, Dead{})
					}
				}

				if f2.Combat.AttackTimer >= 1.0/f2.Combat.AttackSpeed {
					f1.Stats.Health -= f2.Combat.AttackPower
					f2.Combat.AttackTimer = 0
					if f1.Stats.Health <= 0 {
						frame.Commands.AddComponent(f1.EntityId, Dead{})
					}
				}
			}
		}
	}
}

type LifespanSystem struct {
	Aging ecs.Query[struct {
		ecs.EntityId
		*Lifespan
		*Stats
	}]
	GameTime ecs.Singleton[GameTime]
}

func (s *LifespanSystem) Execute(frame *ecs.UpdateFrame) {
	time := s.GameTime.Get()

	for entity := range s.Aging.Iter() {
		age := time.Elapsed - entity.Lifespan.BirthTime
		if age >= entity.Lifespan.MaxAge {
			frame.Commands.AddComponent(entity.EntityId, Dead{})
		}
	}
}

type DeathSystem struct {
	Dead ecs.Query[struct {
		ecs.EntityId
		*Dead
	}]
}

func (s *DeathSystem) Execute(frame *ecs.UpdateFrame) {
	for entity := range s.Dead.Iter() {
		frame.Commands.Delete(entity.EntityId)
	}
}

type ResourceRegrowthSystem struct {
	Resources ecs.Query[struct {
		*Resource
	}]
}

func (s *ResourceRegrowthSystem) Execute(frame *ecs.UpdateFrame) {
	for resource := range s.Resources.Iter() {
		if resource.Resource.Amount < resource.Resource.MaxAmount {
			resource.Resource.RegrowthTime += float32(frame.DeltaTime)
			if resource.Resource.RegrowthTime >= 1.0/resource.Resource.RegrowthRate {
				resource.Resource.Amount++
				resource.Resource.RegrowthTime = 0
			}
		}
	}
}

type CameraControlSystem struct {
	Camera          ecs.Singleton[Camera]
	InputState      ecs.Singleton[InputState]
	ImguiInputState ecs.Singleton[debugui.ImguiInputState]
}

func (s *CameraControlSystem) Execute(frame *ecs.UpdateFrame) {
	camera := s.Camera.Get()
	input := s.InputState.Get()

	imguiInput := s.ImguiInputState.Get()

	if imguiInput.WantCaptureMouse {
		// reset input state?
		return
	}

	mx, my := ebiten.CursorPosition()
	mouseLeft := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	if mouseLeft && !input.PrevMouseLeft {
		input.Dragging = true
		input.DragStartX = camera.X
		input.DragStartY = camera.Y
		input.LastMouseX = mx
		input.LastMouseY = my
	}

	if !mouseLeft {
		input.Dragging = false
	}

	if input.Dragging {
		dx := float32(mx - input.LastMouseX)
		dy := float32(my - input.LastMouseY)
		camera.X = input.DragStartX - dx/(camera.Zoom*CellSize)
		camera.Y = input.DragStartY - dy/(camera.Zoom*CellSize)
	}

	input.PrevMouseLeft = mouseLeft

	_, dy := ebiten.Wheel()
	if dy != 0 {
		oldZoom := camera.Zoom
		camera.Zoom += float32(dy) * 0.2
		if camera.Zoom < 0.5 {
			camera.Zoom = 0.5
		}
		if camera.Zoom > 4.0 {
			camera.Zoom = 4.0
		}

		mouseWorldX := camera.X + float32(mx)/(oldZoom*CellSize)
		mouseWorldY := camera.Y + float32(my)/(oldZoom*CellSize)
		camera.X = mouseWorldX - float32(mx)/(camera.Zoom*CellSize)
		camera.Y = mouseWorldY - float32(my)/(camera.Zoom*CellSize)
	}
}

type RenderSystem struct {
	Camera      ecs.Singleton[Camera]
	WorldConfig ecs.Singleton[WorldConfig]

	Resources ecs.Query[struct {
		*Position
		*Sprite
		*Resource
	}]
	Structures ecs.Query[struct {
		*Position
		*Sprite
		*Structure
	}]
	Colonists ecs.Query[struct {
		*Position
		*Sprite
		*Stats
	}]
	Colonies ecs.Query[struct {
		ecs.EntityId
		*Colony
		*GridPosition
		*ColonyResources
	}]

	screen *ebiten.Image
}

func (s *RenderSystem) Execute(frame *ecs.UpdateFrame) {
	if frame.DeltaTime > 0 || s.screen == nil {
		return
	}

	camera := s.Camera.Get()
	config := s.WorldConfig.Get()

	s.screen.Fill(color.RGBA{245, 245, 240, 255})

	cellSize := float32(config.CellSize)

	for x := 0; x < config.Width; x++ {
		for y := 0; y < config.Height; y++ {
			wx := float32(x) - camera.X
			wy := float32(y) - camera.Y
			sx := wx * camera.Zoom * cellSize
			sy := wy * camera.Zoom * cellSize

			if sx < -cellSize || sy < -cellSize || sx > float32(camera.ScreenW) || sy > float32(camera.ScreenH) {
				continue
			}

			gridColor := color.RGBA{230, 230, 225, 255}
			vector.DrawFilledRect(s.screen, sx, sy, cellSize*camera.Zoom, cellSize*camera.Zoom, gridColor, false)
			vector.StrokeRect(s.screen, sx, sy, cellSize*camera.Zoom, cellSize*camera.Zoom, 1, color.RGBA{220, 220, 215, 255}, false)
		}
	}

	for resource := range s.Resources.Iter() {
		if resource.Resource.Amount <= 0 {
			continue
		}
		s.renderEntity(resource.Position, resource.Sprite, camera, cellSize)
	}

	for structure := range s.Structures.Iter() {
		s.renderEntity(structure.Position, structure.Sprite, camera, cellSize)
	}

	for colonist := range s.Colonists.Iter() {
		s.renderEntity(colonist.Position, colonist.Sprite, camera, cellSize)

		wx := colonist.Position.X - camera.X
		wy := colonist.Position.Y - camera.Y
		sx := wx * camera.Zoom * cellSize
		sy := wy * camera.Zoom * cellSize

		healthPct := float32(colonist.Stats.Health) / float32(colonist.Stats.MaxHealth)
		barWidth := cellSize * camera.Zoom * colonist.Sprite.Scale
		barHeight := float32(2)

		vector.DrawFilledRect(s.screen, sx-barWidth/2, sy-barHeight-5, barWidth, barHeight, color.RGBA{100, 100, 100, 255}, false)
		vector.DrawFilledRect(s.screen, sx-barWidth/2, sy-barHeight-5, barWidth*healthPct, barHeight, color.RGBA{100, 200, 100, 255}, false)
	}

	for colony := range s.Colonies.Iter() {
		wx := float32(colony.GridPosition.X) - camera.X
		wy := float32(colony.GridPosition.Y) - camera.Y
		sx := wx * camera.Zoom * cellSize
		sy := wy * camera.Zoom * cellSize

		c := color.RGBA{colony.Colony.Color[0], colony.Colony.Color[1], colony.Colony.Color[2], 100}
		radius := float32(8) * camera.Zoom
		vector.DrawFilledCircle(s.screen, sx, sy, radius, c, false)
	}
}

func (s *RenderSystem) renderEntity(pos *Position, sprite *Sprite, camera *Camera, cellSize float32) {
	wx := pos.X - camera.X
	wy := pos.Y - camera.Y
	sx := wx * camera.Zoom * cellSize
	sy := wy * camera.Zoom * cellSize

	if sx < -cellSize || sy < -cellSize || sx > float32(camera.ScreenW) || sy > float32(camera.ScreenH) {
		return
	}

	c := color.RGBA{sprite.Color[0], sprite.Color[1], sprite.Color[2], 255}
	size := cellSize * camera.Zoom * sprite.Scale

	switch sprite.Shape {
	case ShapeCircle:
		vector.DrawFilledCircle(s.screen, sx, sy, size/2, c, false)
	case ShapeSquare:
		vector.DrawFilledRect(s.screen, sx-size/2, sy-size/2, size, size, c, false)
	case ShapeTriangle:
		vector.DrawFilledRect(s.screen, sx-size/2, sy-size/2, size, size, c, false)
	}
}

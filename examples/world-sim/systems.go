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

type PauseControlSystem struct {
	PauseState ecs.Singleton[PauseState]
}

func (s *PauseControlSystem) Execute(frame *ecs.UpdateFrame) {
	pauseState := s.PauseState.Get()

	// Handle time-based advancement
	if pauseState.TimeToAdvance > 0 {
		pauseState.TimeAdvanced += float32(frame.DeltaTime)

		// Check if we've advanced enough time
		if pauseState.TimeAdvanced >= pauseState.TimeToAdvance {
			pauseState.TimeToAdvance = 0
			pauseState.TimeAdvanced = 0
			pauseState.FramesToAdvance = 0
		}
	} else if pauseState.FramesToAdvance > 0 {
		// Handle frame-based advancement
		pauseState.FramesToAdvance--
		if pauseState.FramesToAdvance == 0 {
			pauseState.TimeAdvanced = 0
		}
	}

	// Reset step request after it's been processed
	if pauseState.StepRequested {
		pauseState.StepRequested = false
	}
}

func (s *PauseControlSystem) ShouldRunSimulation() bool {
	pauseState := s.PauseState.Get()
	return !pauseState.Paused || pauseState.StepRequested || pauseState.FramesToAdvance > 0 || pauseState.TimeToAdvance > 0
}

type ClearPendingDeathsSystem struct {
	PendingDeaths ecs.Singleton[PendingDeaths]
	PauseState    ecs.Singleton[PauseState]
}

func (s *ClearPendingDeathsSystem) Execute(frame *ecs.UpdateFrame) {
	pauseState := s.PauseState.Get()
	if pauseState.Paused && !pauseState.StepRequested && pauseState.TimeToAdvance == 0 {
		return
	}
	clear(s.PendingDeaths.Get().pending)
}

type TimeSystem struct {
	GameTime   ecs.Singleton[GameTime]
	PauseState ecs.Singleton[PauseState]
}

func (s *TimeSystem) Execute(frame *ecs.UpdateFrame) {
	pauseState := s.PauseState.Get()
	if pauseState.Paused && !pauseState.StepRequested && pauseState.TimeToAdvance == 0 {
		return
	}

	time := s.GameTime.Get()
	time.Elapsed += float32(frame.DeltaTime)

	newDay := int(time.Elapsed / time.DayLength)
	if newDay > time.CurrentDay {
		time.CurrentDay = newDay
	}
}

type SpatialGridSystem struct {
	Grid       ecs.Singleton[SpatialGrid]
	PauseState ecs.Singleton[PauseState]
	Entities   ecs.Query[struct {
		ecs.EntityId
		*GridPosition
	}]
}

func (s *SpatialGridSystem) Execute(frame *ecs.UpdateFrame) {
	pauseState := s.PauseState.Get()
	if pauseState.Paused && !pauseState.StepRequested && pauseState.TimeToAdvance == 0 {
		return
	}

	grid := s.Grid.Get()
	clear(grid.Cells)

	// Populate the grid
	for entity := range s.Entities.Iter() {
		cellX := entity.GridPosition.X / grid.CellSize
		cellY := entity.GridPosition.Y / grid.CellSize
		cellKey := [2]int{cellX, cellY}
		grid.Cells[cellKey] = append(grid.Cells[cellKey], entity.EntityId)
	}
}

type ColonyManagementSystem struct {
	PauseState ecs.Singleton[PauseState]
	Colonies   ecs.Query[struct {
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
	pauseState := s.PauseState.Get()
	if pauseState.Paused && !pauseState.StepRequested && pauseState.TimeToAdvance == 0 {
		return
	}

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
	PauseState ecs.Singleton[PauseState]
	Colonists  ecs.Query[struct {
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
	pauseState := s.PauseState.Get()
	if pauseState.Paused && !pauseState.StepRequested && pauseState.TimeToAdvance == 0 {
		return
	}

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
	PauseState ecs.Singleton[PauseState]
	Moving     ecs.Query[struct {
		*Position
		*GridPosition
		*Task
		*Stats
		Path *Path `ecs:"optional"`
	}]
}

func (s *MovementSystem) Execute(frame *ecs.UpdateFrame) {
	pauseState := s.PauseState.Get()
	if pauseState.Paused && !pauseState.StepRequested && pauseState.TimeToAdvance == 0 {
		return
	}

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
	PauseState ecs.Singleton[PauseState]
	Workers    ecs.Query[struct {
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
	pauseState := s.PauseState.Get()
	if pauseState.Paused && !pauseState.StepRequested && pauseState.TimeToAdvance == 0 {
		return
	}

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
	PauseState ecs.Singleton[PauseState]
	Living     ecs.Query[struct {
		ecs.EntityId
		*Stats
		*ColonyMember
	}]
	Colonies ecs.Query[struct {
		ecs.EntityId
		*ColonyResources
	}]
	PendingDeaths ecs.Singleton[PendingDeaths]
}

func (s *HungerSystem) Execute(frame *ecs.UpdateFrame) {
	pauseState := s.PauseState.Get()
	if pauseState.Paused && !pauseState.StepRequested && pauseState.TimeToAdvance == 0 {
		return
	}

	pending := s.PendingDeaths.Get().pending
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
					if !pending[entity.EntityId] {
						frame.Commands.AddComponent(entity.EntityId, Dead{})
						pending[entity.EntityId] = true
					}
				}
			}
		}
	}
}

type ReproductionSystem struct {
	PauseState       ecs.Singleton[PauseState]
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
	pauseState := s.PauseState.Get()
	if pauseState.Paused && !pauseState.StepRequested && pauseState.TimeToAdvance == 0 {
		return
	}

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

// FighterGridSystem maintains a spatial grid containing only fighters
type FighterGridSystem struct {
	PauseState ecs.Singleton[PauseState]
	Fighters   ecs.Query[struct {
		ecs.EntityId
		*GridPosition
		*Combat
	}]
	FighterGrid ecs.Singleton[FighterGrid]
}

func (s *FighterGridSystem) Execute(frame *ecs.UpdateFrame) {
	pauseState := s.PauseState.Get()
	if pauseState.Paused && !pauseState.StepRequested && pauseState.TimeToAdvance == 0 {
		return
	}

	grid := s.FighterGrid.Get()

	// Clear and rebuild - reuse slice capacity to avoid allocations
	for k := range grid.Cells {
		grid.Cells[k] = grid.Cells[k][:0]
	}

	// Rebuild with only fighters
	for fighter := range s.Fighters.Iter() {
		cellX := fighter.GridPosition.X / grid.CellSize
		cellY := fighter.GridPosition.Y / grid.CellSize
		cellKey := [2]int{cellX, cellY}

		grid.Cells[cellKey] = append(grid.Cells[cellKey], fighter.EntityId)
	}
}

type CombatSystem struct {
	PauseState ecs.Singleton[PauseState]
	Camera     ecs.Singleton[Camera]
	Fighters   ecs.Query[struct {
		ecs.EntityId
		*Combat
		*GridPosition
		*Stats
		*ColonyMember
	}]
	FighterGrid   ecs.Singleton[FighterGrid]
	PendingDeaths ecs.Singleton[PendingDeaths]

	// Cache for fast lookups
	fighterCache map[ecs.EntityId]*combatData

	// Rate limiting: rotate through fighters over multiple frames
	frameCounter uint64
	rotationRate int // Check 1/Nth of fighters per frame
}

// combatData holds cached component data for fast lookup
type combatData struct {
	combat       *Combat
	gridPos      *GridPosition
	stats        *Stats
	colonyMember *ColonyMember
	colonyId     ecs.EntityId // Cached resolved colony ID
	hasColony    bool         // Whether colony ref is valid
}

func (s *CombatSystem) Execute(frame *ecs.UpdateFrame) {
	pauseState := s.PauseState.Get()
	if pauseState.Paused && !pauseState.StepRequested && pauseState.TimeToAdvance == 0 {
		return
	}

	camera := s.Camera.Get()
	grid := s.FighterGrid.Get()
	pending := s.PendingDeaths.Get().pending

	// Initialize rotation rate (check 1/5 of off-screen fighters per frame)
	// This means off-screen combat updates at 12fps when game runs at 60fps
	if s.rotationRate == 0 {
		s.rotationRate = 5
	}
	s.frameCounter++
	currentBucket := s.frameCounter % uint64(s.rotationRate)

	// Combat LOD: skip detailed combat when zoomed out
	skipDetailedCombat := camera.Zoom < 0.5

	// Calculate visible area with some padding for combat range
	const combatPadding = 100.0
	screenWidth := float64(1920.0 / camera.Zoom)
	screenHeight := float64(1080.0 / camera.Zoom)
	minVisibleX := float64(camera.X) - screenWidth/2 - combatPadding
	maxVisibleX := float64(camera.X) + screenWidth/2 + combatPadding
	minVisibleY := float64(camera.Y) - screenHeight/2 - combatPadding
	maxVisibleY := float64(camera.Y) + screenHeight/2 + combatPadding

	// Build cache using query (no reflection!)
	if s.fighterCache == nil {
		s.fighterCache = make(map[ecs.EntityId]*combatData, 1000)
	}
	clear(s.fighterCache)

	for fighter := range s.Fighters.Iter() {
		// Resolve colony ref once and cache it
		colonyId, hasColony := frame.Storage.ResolveEntityRef(fighter.ColonyMember.ColonyRef)

		s.fighterCache[fighter.EntityId] = &combatData{
			combat:       fighter.Combat,
			gridPos:      fighter.GridPosition,
			stats:        fighter.Stats,
			colonyMember: fighter.ColonyMember,
			colonyId:     colonyId,
			hasColony:    hasColony,
		}
	}

	// Use cache for combat checks
	// Get cached colony info for f1 once before inner loops
	fighterIndex := uint64(0)
	for f1 := range s.Fighters.Iter() {
		// Rate limiting: only check fighters in current bucket
		// Fighters on screen get checked every frame (priority)
		f1X := float64(f1.GridPosition.X)
		f1Y := float64(f1.GridPosition.Y)
		isOnScreen := f1X >= minVisibleX && f1X <= maxVisibleX && f1Y >= minVisibleY && f1Y <= maxVisibleY

		if !isOnScreen {
			// Off-screen fighters: rotate through buckets
			if fighterIndex%uint64(s.rotationRate) != currentBucket {
				fighterIndex++
				continue
			}
		}
		fighterIndex++

		if pending[f1.EntityId] {
			continue
		}

		f1Data := s.fighterCache[f1.EntityId]
		if !f1Data.hasColony {
			continue // Can't fight without a colony
		}

		// Combat LOD: skip fighters when zoomed out
		if skipDetailedCombat {
			continue
		}

		// Cache f1 data to avoid repeated field access
		// (f1X, f1Y already calculated above for rate limiting)
		f1PosX := f1.GridPosition.X
		f1PosY := f1.GridPosition.Y
		f1EntityId := f1.EntityId
		cellX := f1PosX / grid.CellSize
		cellY := f1PosY / grid.CellSize

		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				cellKey := [2]int{cellX + dx, cellY + dy}
				entitiesInCell := grid.Cells[cellKey]

				for _, entityId := range entitiesInCell {
					if f1EntityId >= entityId {
						continue
					}

					// Check pending without map lookup in hot path
					if pending[entityId] {
						continue
					}

					// Use cache instead of GetComponent (no reflection!)
					f2Data, exists := s.fighterCache[entityId]
					if !exists {
						continue // Not in cache (shouldn't happen, but be defensive)
					}

					// Use cached colony IDs instead of resolving refs
					if !f2Data.hasColony || f1Data.colonyId == f2Data.colonyId {
						continue // Same colony or no colony
					}

					// Early distance check before accessing more data
					// Use cheaper absolute value checks first
					dx := f1PosX - f2Data.gridPos.X
					if dx < 0 {
						dx = -dx
					}
					if dx > 2 {
						continue // Too far in X direction
					}

					dy := f1PosY - f2Data.gridPos.Y
					if dy < 0 {
						dy = -dy
					}
					if dy > 2 {
						continue // Too far in Y direction
					}

					// Only compute squared distance if within bounding box
					distSq := dx*dx + dy*dy
					if distSq <= 4 {
						// Now access combat data only if in range
						f2Combat := f2Data.combat
						f2Stats := f2Data.stats
						f1.Combat.AttackTimer += float32(frame.DeltaTime)
						f2Combat.AttackTimer += float32(frame.DeltaTime)

						if f1.Combat.AttackTimer >= 1.0/f1.Combat.AttackSpeed {
							f2Stats.Health -= f1.Combat.AttackPower
							f1.Combat.AttackTimer = 0
							if f2Stats.Health <= 0 && !pending[entityId] {
								frame.Commands.AddComponent(entityId, Dead{})
								pending[entityId] = true
							}
						}

						if f2Combat.AttackTimer >= 1.0/f2Combat.AttackSpeed {
							f1.Stats.Health -= f2Combat.AttackPower
							f2Combat.AttackTimer = 0
							if f1.Stats.Health <= 0 && !pending[f1.EntityId] {
								frame.Commands.AddComponent(f1.EntityId, Dead{})
								pending[f1.EntityId] = true
							}
						}
					}
				}
			}
		}
	}
}

type LifespanSystem struct {
	PauseState ecs.Singleton[PauseState]
	Aging      ecs.Query[struct {
		ecs.EntityId
		*Lifespan
		*Stats
	}]
	GameTime      ecs.Singleton[GameTime]
	PendingDeaths ecs.Singleton[PendingDeaths]
}

func (s *LifespanSystem) Execute(frame *ecs.UpdateFrame) {
	pauseState := s.PauseState.Get()
	if pauseState.Paused && !pauseState.StepRequested && pauseState.TimeToAdvance == 0 {
		return
	}

	time := s.GameTime.Get()
	pending := s.PendingDeaths.Get().pending

	for entity := range s.Aging.Iter() {
		age := time.Elapsed - entity.Lifespan.BirthTime
		if age >= entity.Lifespan.MaxAge {
			if !pending[entity.EntityId] {
				frame.Commands.AddComponent(entity.EntityId, Dead{})
				pending[entity.EntityId] = true
			}
		}
	}
}

type DeathSystem struct {
	PauseState ecs.Singleton[PauseState]
	Dead       ecs.Query[struct {
		ecs.EntityId
		*Dead
	}]
}

func (s *DeathSystem) Execute(frame *ecs.UpdateFrame) {
	pauseState := s.PauseState.Get()
	if pauseState.Paused && !pauseState.StepRequested && pauseState.TimeToAdvance == 0 {
		return
	}

	for entity := range s.Dead.Iter() {
		frame.Commands.Delete(entity.EntityId)
	}
}

type ResourceRegrowthSystem struct {
	PauseState ecs.Singleton[PauseState]
	Resources  ecs.Query[struct {
		*Resource
	}]
}

func (s *ResourceRegrowthSystem) Execute(frame *ecs.UpdateFrame) {
	pauseState := s.PauseState.Get()
	if pauseState.Paused && !pauseState.StepRequested && pauseState.TimeToAdvance == 0 {
		return
	}

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
	Grid        ecs.Singleton[SpatialGrid]

	Colonies ecs.Query[struct {
		ecs.EntityId
		*Colony
		*GridPosition
		*ColonyResources
	}]
	Screen ecs.Singleton[Screen]

	// Queries for efficient component access (no reflection)
	RenderableEntities ecs.Query[struct {
		ecs.EntityId
		*Position
		*Sprite
	}]

	RenderableResources ecs.Query[struct {
		ecs.EntityId
		*Position
		*Sprite
		*Resource
	}]

	ColonistsWithHealth ecs.Query[struct {
		ecs.EntityId
		*Position
		*Sprite
		*Stats
		*ColonyMember
	}]

	tileCache      *ebiten.Image
	lastCameraX    float32
	lastCameraY    float32
	lastCameraZoom float32
	tileCacheValid bool

	// Cache for fast spatial lookups
	entityCache map[ecs.EntityId]*renderData
}

// renderData holds cached component data for fast lookup
type renderData struct {
	pos          *Position
	sprite       *Sprite
	resource     *Resource     // nil if not a resource
	stats        *Stats        // nil if not a colonist
	colonyMember *ColonyMember // nil if not a colonist
}

func (s *RenderSystem) Execute(frame *ecs.UpdateFrame) {
	camera := s.Camera.Get()
	config := s.WorldConfig.Get()
	grid := s.Grid.Get()
	screen := s.Screen.Get().Image

	screen.Fill(color.RGBA{245, 245, 240, 255})

	cellSize := float32(config.CellSize)

	cameraMovedSignificantly := false
	if !s.tileCacheValid || s.lastCameraZoom != camera.Zoom {
		cameraMovedSignificantly = true
	} else {
		dx := camera.X - s.lastCameraX
		dy := camera.Y - s.lastCameraY
		threshold := float32(5.0)
		if dx*dx+dy*dy > threshold*threshold {
			cameraMovedSignificantly = true
		}
	}

	if cameraMovedSignificantly {
		s.renderTiles(camera, config, cellSize)
		s.lastCameraX = camera.X
		s.lastCameraY = camera.Y
		s.lastCameraZoom = camera.Zoom
		s.tileCacheValid = true
	}

	if s.tileCache != nil {
		opts := &ebiten.DrawImageOptions{}
		offsetX := (camera.X - s.lastCameraX) * camera.Zoom * cellSize
		offsetY := (camera.Y - s.lastCameraY) * camera.Zoom * cellSize
		opts.GeoM.Translate(float64(-offsetX-100), float64(-offsetY-100))
		screen.DrawImage(s.tileCache, opts)
	}

	// Build cache using queries (no reflection!)
	if s.entityCache == nil {
		s.entityCache = make(map[ecs.EntityId]*renderData, 1000)
	}
	clear(s.entityCache)

	// Cache colonists with health bars
	for colonist := range s.ColonistsWithHealth.Iter() {
		s.entityCache[colonist.EntityId] = &renderData{
			pos:          colonist.Position,
			sprite:       colonist.Sprite,
			stats:        colonist.Stats,
			colonyMember: colonist.ColonyMember,
		}
	}

	// Cache resources
	for resource := range s.RenderableResources.Iter() {
		s.entityCache[resource.EntityId] = &renderData{
			pos:      resource.Position,
			sprite:   resource.Sprite,
			resource: resource.Resource,
		}
	}

	// Cache other renderable entities (won't overwrite colonists/resources due to ECS archetype separation)
	for entity := range s.RenderableEntities.Iter() {
		if _, exists := s.entityCache[entity.EntityId]; !exists {
			s.entityCache[entity.EntityId] = &renderData{
				pos:    entity.Position,
				sprite: entity.Sprite,
			}
		}
	}

	// Use spatial grid for culling, but lookup cached data
	minWorldX := camera.X - 20
	maxWorldX := camera.X + float32(camera.ScreenW)/(cellSize*camera.Zoom) + 20
	minWorldY := camera.Y - 20
	maxWorldY := camera.Y + float32(camera.ScreenH)/(cellSize*camera.Zoom) + 20

	minCellX := int(minWorldX) / grid.CellSize
	maxCellX := int(maxWorldX) / grid.CellSize
	minCellY := int(minWorldY) / grid.CellSize
	maxCellY := int(maxWorldY) / grid.CellSize

	// LOD: Skip rendering individual entities when zoomed out too far
	// At low zoom levels, individual entities are tiny (< 2 pixels) and not visible anyway
	skipDetailedRendering := camera.Zoom < 0.8

	if !skipDetailedRendering {
		for x := minCellX; x <= maxCellX; x++ {
			for y := minCellY; y <= maxCellY; y++ {
				cellKey := [2]int{x, y}
				if entitiesInCell, ok := grid.Cells[cellKey]; ok {
					for _, entityId := range entitiesInCell {
						data, exists := s.entityCache[entityId]
						if !exists {
							continue
						}

						// Skip resources with 0 amount
						if data.resource != nil && data.resource.Amount <= 0 {
							continue
						}

						// Render the entity
						s.renderEntity(screen, data.pos, data.sprite, camera, cellSize)

						// Render health bar for colonists (skip at medium-low zoom)
						if data.stats != nil && data.colonyMember != nil && camera.Zoom > 1.2 {
							s.renderHealthBar(screen, data.pos, data.sprite, data.stats, camera, cellSize)
						}
					}
				}
			}
		}
	}

	// Render colony markers
	for colony := range s.Colonies.Iter() {
		wx := float32(colony.GridPosition.X) - camera.X
		wy := float32(colony.GridPosition.Y) - camera.Y
		sx := wx * camera.Zoom * cellSize
		sy := wy * camera.Zoom * cellSize

		c := color.RGBA{colony.Colony.Color[0], colony.Colony.Color[1], colony.Colony.Color[2], 100}
		radius := float32(8) * camera.Zoom
		vector.DrawFilledCircle(screen, sx, sy, radius, c, false)
	}
}

func (s *RenderSystem) renderTiles(camera *Camera, config *WorldConfig, cellSize float32) {
	cacheW := camera.ScreenW + 200
	cacheH := camera.ScreenH + 200

	if s.tileCache == nil || s.tileCache.Bounds().Dx() != cacheW || s.tileCache.Bounds().Dy() != cacheH {
		s.tileCache = ebiten.NewImage(cacheW, cacheH)
	}
	s.tileCache.Clear()

	minX := int(camera.X) - 20
	maxX := int(camera.X) + camera.ScreenW/int(cellSize*camera.Zoom) + 20
	minY := int(camera.Y) - 20
	maxY := int(camera.Y) + camera.ScreenH/int(cellSize*camera.Zoom) + 20

	if minX < 0 {
		minX = 0
	}
	if maxX > config.Width {
		maxX = config.Width
	}
	if minY < 0 {
		minY = 0
	}
	if maxY > config.Height {
		maxY = config.Height
	}

	for x := minX; x < maxX; x++ {
		for y := minY; y < maxY; y++ {
			wx := float32(x) - camera.X
			wy := float32(y) - camera.Y
			sx := wx*camera.Zoom*cellSize + 100
			sy := wy*camera.Zoom*cellSize + 100

			gridColor := color.RGBA{230, 230, 225, 255}
			vector.DrawFilledRect(s.tileCache, sx, sy, cellSize*camera.Zoom, cellSize*camera.Zoom, gridColor, false)

			if camera.Zoom > 1.0 {
				vector.StrokeRect(s.tileCache, sx, sy, cellSize*camera.Zoom, cellSize*camera.Zoom, 1, color.RGBA{220, 220, 215, 255}, false)
			}
		}
	}
}

func (s *RenderSystem) renderEntity(screen *ebiten.Image, pos *Position, sprite *Sprite, camera *Camera, cellSize float32) {
	wx := pos.X - camera.X
	wy := pos.Y - camera.Y
	sx := wx * camera.Zoom * cellSize
	sy := wy * camera.Zoom * cellSize

	margin := cellSize * 2
	if sx < -margin || sy < -margin || sx > float32(camera.ScreenW)+margin || sy > float32(camera.ScreenH)+margin {
		return
	}

	c := color.RGBA{sprite.Color[0], sprite.Color[1], sprite.Color[2], 255}
	size := cellSize * camera.Zoom * sprite.Scale

	switch sprite.Shape {
	case ShapeCircle:
		vector.DrawFilledCircle(screen, sx, sy, size/2, c, false)
	case ShapeSquare:
		vector.DrawFilledRect(screen, sx-size/2, sy-size/2, size, size, c, false)
	case ShapeTriangle:
		vector.DrawFilledRect(screen, sx-size/2, sy-size/2, size, size, c, false)
	}
}

func (s *RenderSystem) renderHealthBar(screen *ebiten.Image, pos *Position, sprite *Sprite, stats *Stats, camera *Camera, cellSize float32) {
	wx := pos.X - camera.X
	wy := pos.Y - camera.Y
	sx := wx * camera.Zoom * cellSize
	sy := wy * camera.Zoom * cellSize

	healthPct := float32(stats.Health) / float32(stats.MaxHealth)
	barWidth := cellSize * camera.Zoom * sprite.Scale
	barHeight := float32(2)

	// Background bar
	vector.DrawFilledRect(screen, sx-barWidth/2, sy-barHeight-5, barWidth, barHeight, color.RGBA{100, 100, 100, 255}, false)
	// Health bar
	vector.DrawFilledRect(screen, sx-barWidth/2, sy-barHeight-5, barWidth*healthPct, barHeight, color.RGBA{100, 200, 100, 255}, false)
}

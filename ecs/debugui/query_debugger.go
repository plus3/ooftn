package debugui

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/plus3/ooftn/ecs"
)

type QueryDebuggerCache struct {
	componentTypes     []string
	lastArchetypeCount int
}

func NewQueryDebuggerComponent() QueryDebuggerComponent {
	return QueryDebuggerComponent{
		selectedComponentTypes: make(map[string]bool),
		cache: &QueryDebuggerCache{
			lastArchetypeCount: -1,
		},
	}
}

func (qd *QueryDebuggerComponent) Render(storage *ecs.Storage) {
	if !imgui.BeginV("Query Debugger", nil, imgui.WindowFlagsNone) {
		imgui.End()
		return
	}

	qd.rebuildCacheIfNeeded(storage)

	imgui.Text("Select Component Types:")
	imgui.Separator()

	if imgui.Button("Clear All") {
		qd.selectedComponentTypes = make(map[string]bool)
	}

	for _, compType := range qd.cache.componentTypes {
		selected := qd.selectedComponentTypes[compType]
		if imgui.Checkbox(compType, &selected) {
			if selected {
				qd.selectedComponentTypes[compType] = true
			} else {
				delete(qd.selectedComponentTypes, compType)
			}
		}
	}

	imgui.Separator()

	selectedTypes := make([]reflect.Type, 0)
	typeMap := make(map[string]reflect.Type)

	for _, archetype := range storage.GetArchetypes() {
		for _, t := range archetype.Types() {
			typeMap[t.String()] = t
		}
	}

	for typeName := range qd.selectedComponentTypes {
		if t, ok := typeMap[typeName]; ok {
			selectedTypes = append(selectedTypes, t)
		}
	}

	if len(selectedTypes) == 0 {
		imgui.Text("No component types selected")
		imgui.End()
		return
	}

	matchingArchetypes := qd.findMatchingArchetypes(storage, selectedTypes)
	totalEntities := 0
	for _, arch := range matchingArchetypes {
		for range arch.Iter() {
			totalEntities++
		}
	}

	imgui.Text(fmt.Sprintf("Matching Archetypes: %d", len(matchingArchetypes)))
	imgui.Text(fmt.Sprintf("Matching Entities: %d", totalEntities))

	if imgui.TreeNodeStr("Archetype Details") {
		const tableFlags = imgui.TableFlagsBorders | imgui.TableFlagsRowBg
		if imgui.BeginTableV("QueryArchTable", 3, tableFlags, imgui.NewVec2(0, 0), 0) {
			imgui.TableSetupColumn("Archetype ID")
			imgui.TableSetupColumn("All Components")
			imgui.TableSetupColumn("Entity Count")
			imgui.TableHeadersRow()

			for _, arch := range matchingArchetypes {
				imgui.TableNextRow()

				imgui.TableSetColumnIndex(0)
				imgui.Text(fmt.Sprintf("0x%X", arch.ID()))

				imgui.TableSetColumnIndex(1)
				componentNames := make([]string, len(arch.Types()))
				for i, t := range arch.Types() {
					componentNames[i] = t.String()
				}
				imgui.Text(fmt.Sprintf("%v", componentNames))

				imgui.TableSetColumnIndex(2)
				entityCount := 0
				for range arch.Iter() {
					entityCount++
				}
				imgui.Text(fmt.Sprintf("%d", entityCount))
			}

			imgui.EndTable()
		}
		imgui.TreePop()
	}

	imgui.End()
}

func (qd *QueryDebuggerComponent) rebuildCacheIfNeeded(storage *ecs.Storage) {
	currentArchetypeCount := len(storage.GetArchetypes())
	if qd.cache.lastArchetypeCount != currentArchetypeCount {
		qd.cache.componentTypes = nil
		qd.cache.lastArchetypeCount = currentArchetypeCount
	}

	if qd.cache.componentTypes == nil {
		qd.rebuildCache(storage)
	}
}

func (qd *QueryDebuggerComponent) rebuildCache(storage *ecs.Storage) {
	typeMap := make(map[string]bool)

	for _, archetype := range storage.GetArchetypes() {
		for _, t := range archetype.Types() {
			typeMap[t.String()] = true
		}
	}

	qd.cache.componentTypes = make([]string, 0, len(typeMap))
	for typeName := range typeMap {
		qd.cache.componentTypes = append(qd.cache.componentTypes, typeName)
	}

	sort.Strings(qd.cache.componentTypes)
}

func (qd *QueryDebuggerComponent) findMatchingArchetypes(storage *ecs.Storage, requiredTypes []reflect.Type) []*ecs.Archetype {
	matching := make([]*ecs.Archetype, 0)

	for _, archetype := range storage.GetArchetypes() {
		if qd.archetypeHasAllTypes(archetype, requiredTypes) {
			matching = append(matching, archetype)
		}
	}

	return matching
}

func (qd *QueryDebuggerComponent) archetypeHasAllTypes(archetype *ecs.Archetype, requiredTypes []reflect.Type) bool {
	archetypeTypes := archetype.Types()
	typeMap := make(map[reflect.Type]bool)
	for _, t := range archetypeTypes {
		typeMap[t] = true
	}

	for _, required := range requiredTypes {
		if !typeMap[required] {
			return false
		}
	}

	return true
}

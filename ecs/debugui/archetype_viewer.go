package debugui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/plus3/ooftn/ecs"
)

type ArchetypeInfo struct {
	ID             uint32
	ComponentTypes []string
	EntityCount    int
	ComponentCount int
}

type ArchetypeViewerCache struct {
	archetypes         []ArchetypeInfo
	lastArchetypeCount int
	sortColumn         int
	sortAscending      bool
}

func NewArchetypeViewerComponent() ArchetypeViewerComponent {
	return ArchetypeViewerComponent{
		cache: &ArchetypeViewerCache{
			sortColumn:    3,
			sortAscending: false,
		},
		sortColumn:    3,
		sortAscending: false,
	}
}

func (av *ArchetypeViewerComponent) Render(storage *ecs.Storage) *uint32 {
	if !imgui.BeginV("Archetype Viewer", nil, imgui.WindowFlagsNone) {
		imgui.End()
		return nil
	}

	av.rebuildCacheIfNeeded(storage)

	maxEntityCount := 0
	for _, arch := range av.cache.archetypes {
		if arch.EntityCount > maxEntityCount {
			maxEntityCount = arch.EntityCount
		}
	}

	const tableFlags = imgui.TableFlagsBorders | imgui.TableFlagsRowBg | imgui.TableFlagsSortable | imgui.TableFlagsScrollY
	if imgui.BeginTableV("ArchetypeTable", 4, tableFlags, imgui.NewVec2(0, 0), 0) {
		imgui.TableSetupColumn("Archetype ID")
		imgui.TableSetupColumn("Components")
		imgui.TableSetupColumn("Comp Count")
		imgui.TableSetupColumn("Entity Count")
		imgui.TableHeadersRow()

		sortSpecs := imgui.TableGetSortSpecs()
		if sortSpecs.SpecsDirty() && sortSpecs.SpecsCount() > 0 {
			spec := sortSpecs.Specs()
			av.cache.sortColumn = int(spec.ColumnIndex())
			av.cache.sortAscending = spec.SortDirection() == imgui.SortDirectionAscending
			av.sortColumn = av.cache.sortColumn
			av.sortAscending = av.cache.sortAscending
			av.sortArchetypes()
			sortSpecs.SetSpecsDirty(false)
		}

		var clickedArchId *uint32

		for _, arch := range av.cache.archetypes {
			imgui.TableNextRow()

			imgui.TableNextColumn()
			isSelected := av.selectedArchId != nil && *av.selectedArchId == arch.ID
			if imgui.SelectableBoolV(fmt.Sprintf("0x%X", arch.ID), isSelected, imgui.SelectableFlagsSpanAllColumns, imgui.NewVec2(0, 0)) {
				archIdCopy := arch.ID
				clickedArchId = &archIdCopy
				av.selectedArchId = &archIdCopy
			}

			imgui.TableNextColumn()
			imgui.Text(strings.Join(arch.ComponentTypes, ", "))

			imgui.TableNextColumn()
			imgui.Text(fmt.Sprintf("%d", arch.ComponentCount))

			imgui.TableNextColumn()
			imgui.Text(fmt.Sprintf("%d", arch.EntityCount))

			if maxEntityCount > 0 {
				barWidth := float32(arch.EntityCount) / float32(maxEntityCount) * 80.0
				imgui.SameLine()
				drawList := imgui.WindowDrawList()
				pos := imgui.CursorScreenPos()
				color := imgui.ColorU32Vec4(imgui.NewVec4(0.2, 0.6, 0.8, 0.6))
				drawList.AddRectFilled(pos, imgui.NewVec2(pos.X+barWidth, pos.Y+10), color)
			}
		}

		imgui.EndTable()

		imgui.End()
		return clickedArchId
	}

	imgui.End()
	return nil
}

func (av *ArchetypeViewerComponent) rebuildCacheIfNeeded(storage *ecs.Storage) {
	currentArchetypeCount := len(storage.GetArchetypes())
	if av.cache.lastArchetypeCount != currentArchetypeCount {
		av.cache.archetypes = nil
		av.cache.lastArchetypeCount = currentArchetypeCount
	}

	if av.cache.archetypes == nil {
		av.rebuildCache(storage)
	} else {
		av.updateEntityCounts(storage)
	}
}

func (av *ArchetypeViewerComponent) rebuildCache(storage *ecs.Storage) {
	av.cache.archetypes = make([]ArchetypeInfo, 0, len(storage.GetArchetypes()))

	for _, archetype := range storage.GetArchetypes() {
		componentTypes := make([]string, len(archetype.Types()))
		for i, t := range archetype.Types() {
			componentTypes[i] = t.String()
		}

		entityCount := 0
		for range archetype.Iter() {
			entityCount++
		}

		av.cache.archetypes = append(av.cache.archetypes, ArchetypeInfo{
			ID:             archetype.ID(),
			ComponentTypes: componentTypes,
			EntityCount:    entityCount,
			ComponentCount: len(componentTypes),
		})
	}

	av.sortArchetypes()
}

func (av *ArchetypeViewerComponent) updateEntityCounts(storage *ecs.Storage) {
	archetypeMap := make(map[uint32]*ecs.Archetype)
	for _, archetype := range storage.GetArchetypes() {
		archetypeMap[archetype.ID()] = archetype
	}

	for i := range av.cache.archetypes {
		archetype, ok := archetypeMap[av.cache.archetypes[i].ID]
		if !ok {
			continue
		}

		entityCount := 0
		for range archetype.Iter() {
			entityCount++
		}
		av.cache.archetypes[i].EntityCount = entityCount
	}

	if av.sortColumn == 3 {
		av.sortArchetypes()
	}
}

func (av *ArchetypeViewerComponent) sortArchetypes() {
	sort.Slice(av.cache.archetypes, func(i, j int) bool {
		a, b := av.cache.archetypes[i], av.cache.archetypes[j]
		var less bool

		switch av.cache.sortColumn {
		case 0:
			less = a.ID < b.ID
		case 1:
			less = strings.Join(a.ComponentTypes, ",") < strings.Join(b.ComponentTypes, ",")
		case 2:
			less = a.ComponentCount < b.ComponentCount
		case 3:
			less = a.EntityCount < b.EntityCount
		default:
			less = a.EntityCount < b.EntityCount
		}

		if !av.cache.sortAscending {
			return !less
		}
		return less
	})
}

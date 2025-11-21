package debugui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/plus3/ooftn/ecs"
)

type EntityInfo struct {
	ID             ecs.EntityId
	ArchetypeID    uint32
	ComponentTypes []string
	ComponentCount int
}

type EntityBrowserCache struct {
	entities           []EntityInfo
	lastArchetypeCount int
	sortColumn         int
	sortAscending      bool
}

func NewEntityBrowserComponent(maxEntitiesPerPage int) EntityBrowserComponent {
	return EntityBrowserComponent{
		cache: &EntityBrowserCache{
			sortColumn:    0,
			sortAscending: true,
		},
		maxEntitiesPerPage: maxEntitiesPerPage,
	}
}

func (eb *EntityBrowserComponent) Render(storage *ecs.Storage) {
	if !imgui.BeginV("Entity Browser", nil, imgui.WindowFlagsNone) {
		imgui.End()
		return
	}

	eb.rebuildCacheIfNeeded(storage)

	imgui.InputTextWithHint("##search", "Search...", &eb.filterText, imgui.InputTextFlagsNone, nil)
	imgui.SameLine()
	if imgui.Button("Clear Filter") {
		eb.filterText = ""
		eb.filterArchetypeId = nil
	}

	const tableFlags = imgui.TableFlagsBorders | imgui.TableFlagsRowBg | imgui.TableFlagsSortable | imgui.TableFlagsScrollY
	if imgui.BeginTableV("EntityTable", 4, tableFlags, imgui.NewVec2(0, 0), 0) {
		imgui.TableSetupColumn("Entity ID")
		imgui.TableSetupColumn("Archetype ID")
		imgui.TableSetupColumn("Components")
		imgui.TableSetupColumn("Count")
		imgui.TableHeadersRow()

		sortSpecs := imgui.TableGetSortSpecs()
		if sortSpecs.SpecsDirty() && sortSpecs.SpecsCount() > 0 {
			spec := sortSpecs.Specs()
			eb.cache.sortColumn = int(spec.ColumnIndex())
			eb.cache.sortAscending = spec.SortDirection() == imgui.SortDirectionAscending
			eb.sortEntities()
			sortSpecs.SetSpecsDirty(false)
		}

		filteredEntities := eb.getFilteredEntities()

		startIdx := eb.currentPage * eb.maxEntitiesPerPage
		endIdx := startIdx + eb.maxEntitiesPerPage
		if endIdx > len(filteredEntities) {
			endIdx = len(filteredEntities)
		}

		for i := startIdx; i < endIdx; i++ {
			entity := filteredEntities[i]
			imgui.TableNextRow()

			imgui.TableNextColumn()
			isSelected := eb.selectedEntityId == entity.ID
			if imgui.SelectableBoolV(fmt.Sprintf("%d", entity.ID), isSelected, imgui.SelectableFlagsSpanAllColumns, imgui.NewVec2(0, 0)) {
				eb.selectedEntityId = entity.ID
			}

			imgui.TableNextColumn()
			imgui.Text(fmt.Sprintf("0x%X", entity.ArchetypeID))

			imgui.TableNextColumn()
			imgui.Text(strings.Join(entity.ComponentTypes, ", "))

			imgui.TableNextColumn()
			imgui.Text(fmt.Sprintf("%d", entity.ComponentCount))
		}

		imgui.EndTable()
	}

	filteredEntities := eb.getFilteredEntities()

	if len(filteredEntities) > eb.maxEntitiesPerPage {
		totalPages := (len(filteredEntities) + eb.maxEntitiesPerPage - 1) / eb.maxEntitiesPerPage
		imgui.Text(fmt.Sprintf("Page %d / %d (%d entities)", eb.currentPage+1, totalPages, len(filteredEntities)))
		imgui.SameLine()
		if imgui.Button("Prev") && eb.currentPage > 0 {
			eb.currentPage--
		}
		imgui.SameLine()
		if imgui.Button("Next") && eb.currentPage < totalPages-1 {
			eb.currentPage++
		}
	} else {
		imgui.Text(fmt.Sprintf("Total: %d entities", len(filteredEntities)))
	}

	imgui.End()
}

func (eb *EntityBrowserComponent) rebuildCacheIfNeeded(storage *ecs.Storage) {
	currentArchetypeCount := len(storage.GetArchetypes())
	if eb.cache.lastArchetypeCount != currentArchetypeCount {
		eb.cache.entities = nil
		eb.cache.lastArchetypeCount = currentArchetypeCount
	}

	if eb.cache.entities == nil {
		eb.rebuildCache(storage)
	}
}

func (eb *EntityBrowserComponent) rebuildCache(storage *ecs.Storage) {
	eb.cache.entities = make([]EntityInfo, 0, 1024)

	for _, archetype := range storage.GetArchetypes() {
		componentTypes := make([]string, len(archetype.Types()))
		for i, t := range archetype.Types() {
			componentTypes[i] = t.String()
		}

		for entityId := range archetype.Iter() {
			eb.cache.entities = append(eb.cache.entities, EntityInfo{
				ID:             entityId,
				ArchetypeID:    archetype.ID(),
				ComponentTypes: componentTypes,
				ComponentCount: len(componentTypes),
			})
		}
	}

	eb.sortEntities()
}

func (eb *EntityBrowserComponent) sortEntities() {
	sort.Slice(eb.cache.entities, func(i, j int) bool {
		a, b := eb.cache.entities[i], eb.cache.entities[j]
		var less bool

		switch eb.cache.sortColumn {
		case 0:
			less = a.ID < b.ID
		case 1:
			less = a.ArchetypeID < b.ArchetypeID
		case 2:
			less = strings.Join(a.ComponentTypes, ",") < strings.Join(b.ComponentTypes, ",")
		case 3:
			less = a.ComponentCount < b.ComponentCount
		default:
			less = a.ID < b.ID
		}

		if !eb.cache.sortAscending {
			return !less
		}
		return less
	})
}

func (eb *EntityBrowserComponent) getFilteredEntities() []EntityInfo {
	if eb.filterText == "" && eb.filterArchetypeId == nil {
		return eb.cache.entities
	}

	filtered := make([]EntityInfo, 0, len(eb.cache.entities))
	filterLower := strings.ToLower(eb.filterText)

	for _, entity := range eb.cache.entities {
		if eb.filterArchetypeId != nil && entity.ArchetypeID != *eb.filterArchetypeId {
			continue
		}

		if eb.filterText != "" {
			idStr := fmt.Sprintf("%d", entity.ID)
			archStr := fmt.Sprintf("0x%x", entity.ArchetypeID)
			componentsStr := strings.ToLower(strings.Join(entity.ComponentTypes, " "))

			if !strings.Contains(idStr, filterLower) &&
				!strings.Contains(archStr, filterLower) &&
				!strings.Contains(componentsStr, filterLower) {
				continue
			}
		}

		filtered = append(filtered, entity)
	}

	return filtered
}

func (eb *EntityBrowserComponent) GetSelectedEntity() ecs.EntityId {
	return eb.selectedEntityId
}

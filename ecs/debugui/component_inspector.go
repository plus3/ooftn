package debugui

import (
	"fmt"
	"reflect"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/plus3/ooftn/ecs"
)

func NewComponentInspectorComponent() ComponentInspectorComponent {
	return ComponentInspectorComponent{}
}

func (ci *ComponentInspectorComponent) Render(storage *ecs.Storage, selectedEntityId ecs.EntityId) {
	if !imgui.BeginV("Component Inspector", nil, imgui.WindowFlagsNone) {
		imgui.End()
		return
	}

	ci.selectedEntityId = selectedEntityId

	if ci.selectedEntityId == 0 {
		imgui.Text("No entity selected")
		imgui.End()
		return
	}

	archetypeId := ci.selectedEntityId.ArchetypeId()
	archetype := storage.GetArchetypeById(archetypeId)
	if archetype == nil {
		imgui.Text(fmt.Sprintf("Entity %d not found (invalid archetype)", ci.selectedEntityId))
		imgui.End()
		return
	}

	imgui.Text(fmt.Sprintf("Entity ID: %d", ci.selectedEntityId))
	imgui.Text(fmt.Sprintf("Archetype: 0x%X", archetypeId))
	imgui.Separator()

	for _, compType := range archetype.Types() {
		component := storage.GetComponent(ci.selectedEntityId, compType)
		if component == nil {
			continue
		}

		if imgui.TreeNodeStr(compType.String()) {
			ci.renderComponent(component, compType, storage, ci.selectedEntityId)
			imgui.TreePop()
		}
	}

	imgui.End()
}

func (ci *ComponentInspectorComponent) renderComponent(component any, compType reflect.Type, storage *ecs.Storage, entityId ecs.EntityId) {
	val := reflect.ValueOf(component)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	fields := globalReflectionCache.GetFields(compType)

	for _, field := range fields {
		fieldVal := val.Field(field.Index)
		if field.IsPointer && !fieldVal.IsNil() {
			fieldVal = fieldVal.Elem()
		}

		ci.renderField(field.Name, fieldVal, field, storage, entityId, compType)
	}
}

func (ci *ComponentInspectorComponent) renderField(name string, val reflect.Value, field FieldInfo, storage *ecs.Storage, entityId ecs.EntityId, compType reflect.Type) {
	if !val.IsValid() {
		imgui.Text(fmt.Sprintf("%s: <invalid>", name))
		return
	}

	if field.IsPointer && val.IsNil() {
		imgui.Text(fmt.Sprintf("%s: nil", name))
		return
	}

	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v := int32(val.Int())
		imgui.Text(fmt.Sprintf("%s:", name))
		imgui.SameLine()
		imgui.SetNextItemWidth(150)
		if imgui.InputInt(fmt.Sprintf("##%s", name), &v) {
			ci.updateIntField(storage, entityId, compType, field.Index, int64(v), val.Type())
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v := int32(val.Uint())
		imgui.Text(fmt.Sprintf("%s:", name))
		imgui.SameLine()
		imgui.SetNextItemWidth(150)
		if imgui.InputInt(fmt.Sprintf("##%s", name), &v) {
			if v >= 0 {
				ci.updateUintField(storage, entityId, compType, field.Index, uint64(v), val.Type())
			}
		}

	case reflect.Float32, reflect.Float64:
		v := float32(val.Float())
		imgui.Text(fmt.Sprintf("%s:", name))
		imgui.SameLine()
		imgui.SetNextItemWidth(150)
		if imgui.InputFloat(fmt.Sprintf("##%s", name), &v) {
			ci.updateFloatField(storage, entityId, compType, field.Index, float64(v), val.Type())
		}

	case reflect.Bool:
		v := val.Bool()
		if imgui.Checkbox(name, &v) {
			ci.updateBoolField(storage, entityId, compType, field.Index, v)
		}

	case reflect.String:
		v := val.String()
		imgui.Text(fmt.Sprintf("%s:", name))
		imgui.SameLine()
		imgui.SetNextItemWidth(200)
		if imgui.InputTextWithHint(fmt.Sprintf("##%s", name), "", &v, imgui.InputTextFlagsNone, nil) {
			ci.updateStringField(storage, entityId, compType, field.Index, v)
		}

	case reflect.Struct:
		if imgui.TreeNodeStr(name) {
			nestedFields := globalReflectionCache.GetFields(val.Type())
			for _, nf := range nestedFields {
				nestedVal := val.Field(nf.Index)
				if nf.IsPointer && !nestedVal.IsNil() {
					nestedVal = nestedVal.Elem()
				}
				ci.renderField(nf.Name, nestedVal, nf, storage, entityId, compType)
			}
			imgui.TreePop()
		}

	case reflect.Slice:
		imgui.Text(fmt.Sprintf("%s: [%d items]", name, val.Len()))

	case reflect.Map:
		imgui.Text(fmt.Sprintf("%s: map[%d items]", name, val.Len()))

	default:
		imgui.Text(fmt.Sprintf("%s: %v", name, val.Interface()))
	}
}

func (ci *ComponentInspectorComponent) updateIntField(storage *ecs.Storage, entityId ecs.EntityId, compType reflect.Type, fieldIdx int, value int64, fieldType reflect.Type) {
	component := storage.GetComponent(entityId, compType)
	if component == nil {
		return
	}

	val := reflect.ValueOf(component)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	field := val.Field(fieldIdx)
	if field.CanSet() {
		switch fieldType.Kind() {
		case reflect.Int:
			field.SetInt(value)
		case reflect.Int8:
			field.SetInt(value)
		case reflect.Int16:
			field.SetInt(value)
		case reflect.Int32:
			field.SetInt(value)
		case reflect.Int64:
			field.SetInt(value)
		}
	}
}

func (ci *ComponentInspectorComponent) updateUintField(storage *ecs.Storage, entityId ecs.EntityId, compType reflect.Type, fieldIdx int, value uint64, fieldType reflect.Type) {
	component := storage.GetComponent(entityId, compType)
	if component == nil {
		return
	}

	val := reflect.ValueOf(component)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	field := val.Field(fieldIdx)
	if field.CanSet() {
		field.SetUint(value)
	}
}

func (ci *ComponentInspectorComponent) updateFloatField(storage *ecs.Storage, entityId ecs.EntityId, compType reflect.Type, fieldIdx int, value float64, fieldType reflect.Type) {
	component := storage.GetComponent(entityId, compType)
	if component == nil {
		return
	}

	val := reflect.ValueOf(component)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	field := val.Field(fieldIdx)
	if field.CanSet() {
		field.SetFloat(value)
	}
}

func (ci *ComponentInspectorComponent) updateBoolField(storage *ecs.Storage, entityId ecs.EntityId, compType reflect.Type, fieldIdx int, value bool) {
	component := storage.GetComponent(entityId, compType)
	if component == nil {
		return
	}

	val := reflect.ValueOf(component)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	field := val.Field(fieldIdx)
	if field.CanSet() {
		field.SetBool(value)
	}
}

func (ci *ComponentInspectorComponent) updateStringField(storage *ecs.Storage, entityId ecs.EntityId, compType reflect.Type, fieldIdx int, value string) {
	component := storage.GetComponent(entityId, compType)
	if component == nil {
		return
	}

	val := reflect.ValueOf(component)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	field := val.Field(fieldIdx)
	if field.CanSet() {
		field.SetString(value)
	}
}

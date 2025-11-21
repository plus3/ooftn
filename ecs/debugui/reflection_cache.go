package debugui

import (
	"reflect"
	"sync"
)

type FieldInfo struct {
	Name      string
	Type      reflect.Type
	Index     int
	IsPointer bool
	IsStruct  bool
	IsSlice   bool
	IsMap     bool
}

type ReflectionCache struct {
	mu         sync.RWMutex
	fieldCache map[reflect.Type][]FieldInfo
}

func NewReflectionCache() *ReflectionCache {
	return &ReflectionCache{
		fieldCache: make(map[reflect.Type][]FieldInfo),
	}
}

func (rc *ReflectionCache) GetFields(t reflect.Type) []FieldInfo {
	rc.mu.RLock()
	cached, ok := rc.fieldCache[t]
	rc.mu.RUnlock()
	if ok {
		return cached
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	if cached, ok := rc.fieldCache[t]; ok {
		return cached
	}

	var fields []FieldInfo
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}

			fieldType := field.Type
			isPointer := fieldType.Kind() == reflect.Ptr
			if isPointer {
				fieldType = fieldType.Elem()
			}

			fields = append(fields, FieldInfo{
				Name:      field.Name,
				Type:      fieldType,
				Index:     i,
				IsPointer: isPointer,
				IsStruct:  fieldType.Kind() == reflect.Struct,
				IsSlice:   fieldType.Kind() == reflect.Slice,
				IsMap:     fieldType.Kind() == reflect.Map,
			})
		}
	}

	rc.fieldCache[t] = fields
	return fields
}

var globalReflectionCache = NewReflectionCache()

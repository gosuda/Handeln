package tool

import (
	"fmt"
	"reflect"
	"strings"
)

// Definition represents a tool's metadata and interface.
type Definition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

// GenerateSchema generates a JSON Schema from a Go struct using reflection.
func GenerateSchema(v interface{}) (map[string]interface{}, error) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("GenerateSchema: expected struct, got %v", t.Kind())
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
	}

	required := []string{}
	properties := schema["properties"].(map[string]interface{})

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		name := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				name = parts[0]
			}
		}

		prop, req, err := getTypeSchema(field.Type)
		if err != nil {
			return nil, err
		}

		// Add description from tag if present
		if desc := field.Tag.Get("description"); desc != "" {
			prop["description"] = desc
		}

		properties[name] = prop
		if req {
			required = append(required, name)
		}
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema, nil
}

func getTypeSchema(t reflect.Type) (map[string]interface{}, bool, error) {
	required := true // Default to required unless it's a pointer or has omitempty
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		required = false
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]interface{}{"type": "string"}, required, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]interface{}{"type": "integer"}, required, nil
	case reflect.Float32, reflect.Float64:
		return map[string]interface{}{"type": "number"}, required, nil
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}, required, nil
	case reflect.Slice:
		items, _, err := getTypeSchema(t.Elem())
		if err != nil {
			return nil, false, err
		}
		return map[string]interface{}{
			"type":  "array",
			"items": items,
		}, required, nil
	case reflect.Struct:
		schema := map[string]interface{}{
			"type":       "object",
			"properties": make(map[string]interface{}),
		}
		requiredFields := []string{}
		properties := schema["properties"].(map[string]interface{})

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}
			name := field.Name
			if jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" {
					name = parts[0]
				}
			}

			prop, req, err := getTypeSchema(field.Type)
			if err != nil {
				return nil, false, err
			}
			if desc := field.Tag.Get("description"); desc != "" {
				prop["description"] = desc
			}
			properties[name] = prop
			if req {
				requiredFields = append(requiredFields, name)
			}
		}
		if len(requiredFields) > 0 {
			schema["required"] = requiredFields
		}
		return schema, required, nil
	default:
		return nil, false, fmt.Errorf("unsupported type: %v", t.Kind())
	}
}

// FromStruct creates a tool Definition from a struct name, description and input struct.
func FromStruct(name, description string, input interface{}) (Definition, error) {
	schema, err := GenerateSchema(input)
	if err != nil {
		return Definition{}, err
	}
	return Definition{
		Name:        name,
		Description: description,
		InputSchema: schema,
	}, nil
}

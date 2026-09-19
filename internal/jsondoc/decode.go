package jsondoc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

var jsonUnmarshalerType = reflect.TypeFor[json.Unmarshaler]()

// DecodeObject validates and decodes a JSON object into destination.
// It rejects fields outside the explicit JSON names of each nested struct that
// uses default JSON decoding in destination's schema.
func DecodeObject[T any](data []byte, destination *T) error {
	if destination == nil {
		return errors.New("JSON destination must not be nil")
	}
	if err := Validate(data); err != nil {
		return err
	}
	if err := validateObject(data, reflect.TypeFor[T]()); err != nil {
		return err
	}

	return json.Unmarshal(data, destination)
}

func validateObject(data []byte, destinationType reflect.Type) error {
	fields, err := objectFieldNames(destinationType)
	if err != nil {
		return err
	}
	if err := validateNestedSchemas(destinationType, make(map[reflect.Type]struct{})); err != nil {
		return err
	}
	return validateJSONObject(data, fields, true)
}

func validateNestedSchemas(valueType reflect.Type, seen map[reflect.Type]struct{}) error {
	if implementsJSONUnmarshaler(valueType) {
		return nil
	}

	switch valueType.Kind() {
	case reflect.Pointer, reflect.Array, reflect.Slice, reflect.Map:
		return validateNestedSchemas(valueType.Elem(), seen)
	case reflect.Struct:
		if _, alreadyValidated := seen[valueType]; alreadyValidated {
			return nil
		}
		seen[valueType] = struct{}{}
		fields, err := objectFieldNames(valueType)
		if err != nil {
			return err
		}
		for _, field := range fields {
			if err := validateNestedSchemas(field.Type, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateJSONObject(data []byte, fields map[string]reflect.StructField, requireObject bool) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil || object == nil {
		if requireObject {
			return errors.New("JSON value must be an object")
		}
		return nil
	}

	objectFields := make([]string, 0, len(object))
	for field := range object {
		objectFields = append(objectFields, field)
	}
	sort.Strings(objectFields)
	for _, field := range objectFields {
		structField, known := fields[field]
		if !known {
			return fmt.Errorf("unknown JSON field %q", field)
		}
		if err := validateValue(object[field], structField.Type); err != nil {
			return err
		}
	}
	return nil
}

func validateValue(data json.RawMessage, valueType reflect.Type) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) || implementsJSONUnmarshaler(valueType) {
		return nil
	}

	switch valueType.Kind() {
	case reflect.Pointer:
		return validateValue(data, valueType.Elem())
	case reflect.Struct:
		fields, err := objectFieldNames(valueType)
		if err != nil {
			return err
		}
		return validateJSONObject(data, fields, false)
	case reflect.Array, reflect.Slice:
		var elements []json.RawMessage
		if err := json.Unmarshal(data, &elements); err != nil {
			return nil
		}
		for _, element := range elements {
			if err := validateValue(element, valueType.Elem()); err != nil {
				return err
			}
		}
	case reflect.Map:
		var values map[string]json.RawMessage
		if err := json.Unmarshal(data, &values); err != nil || values == nil {
			return nil
		}
		keys := make([]string, 0, len(values))
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if err := validateValue(values[key], valueType.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}

func objectFieldNames(destinationType reflect.Type) (map[string]reflect.StructField, error) {
	if destinationType.Kind() != reflect.Struct {
		return nil, errors.New("JSON destination must point to a struct")
	}
	if implementsJSONUnmarshaler(destinationType) {
		return nil, errors.New("JSON destination must not implement json.Unmarshaler")
	}

	canonicalFields := make(map[string]reflect.StructField, destinationType.NumField())
	for index := range destinationType.NumField() {
		field := destinationType.Field(index)
		if field.Anonymous {
			return nil, fmt.Errorf("JSON destination field %q must not be embedded", field.Name)
		}
		if !field.IsExported() {
			continue
		}
		tag, exists := field.Tag.Lookup("json")
		name, _, hasOptions := strings.Cut(tag, ",")
		if !exists || !isProductJSONFieldName(name) || hasOptions {
			return nil, fmt.Errorf("JSON destination field %q must have one explicit lowercase-initial ASCII alphanumeric JSON name", field.Name)
		}
		for canonicalField := range canonicalFields {
			if strings.EqualFold(name, canonicalField) {
				return nil, fmt.Errorf("JSON destination fields %q and %q conflict", canonicalField, name)
			}
		}
		canonicalFields[name] = field
	}
	return canonicalFields, nil
}

func implementsJSONUnmarshaler(valueType reflect.Type) bool {
	return valueType.Implements(jsonUnmarshalerType) ||
		valueType.Kind() != reflect.Pointer && reflect.PointerTo(valueType).Implements(jsonUnmarshalerType)
}

func isProductJSONFieldName(name string) bool {
	if name == "" || name[0] < 'a' || name[0] > 'z' {
		return false
	}
	for _, character := range []byte(name[1:]) {
		if character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' {
			continue
		}
		return false
	}
	return true
}

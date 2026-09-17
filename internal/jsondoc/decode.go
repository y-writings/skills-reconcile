package jsondoc

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// StrictObjectSchema marks product-owned schemas for strict top-level JSON objects.
type StrictObjectSchema interface {
	StrictJSONObjectSchema()
}

// DecodeObject decodes one object into a struct whose explicit JSON field names
// define the complete set of accepted top-level fields.
func DecodeObject[T StrictObjectSchema](data []byte, destination *T) error {
	if destination == nil {
		return errors.New("JSON destination must not be nil")
	}
	if err := Validate(data); err != nil {
		return err
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil || object == nil {
		return errors.New("JSON value must be an object")
	}

	canonicalFields, err := objectFieldNames(reflect.TypeFor[T]())
	if err != nil {
		return err
	}
	objectFields := make([]string, 0, len(object))
	for field := range object {
		objectFields = append(objectFields, field)
	}
	sort.Strings(objectFields)
	for _, field := range objectFields {
		if _, known := canonicalFields[field]; !known {
			return fmt.Errorf("unknown JSON field %q", field)
		}
	}

	return json.Unmarshal(data, destination)
}

func objectFieldNames(destinationType reflect.Type) (map[string]struct{}, error) {
	if destinationType.Kind() != reflect.Struct {
		return nil, errors.New("JSON destination must point to a struct")
	}
	jsonUnmarshalerType := reflect.TypeFor[json.Unmarshaler]()
	if destinationType.Implements(jsonUnmarshalerType) || reflect.PointerTo(destinationType).Implements(jsonUnmarshalerType) {
		return nil, errors.New("JSON destination must not implement json.Unmarshaler")
	}

	canonicalFields := make(map[string]struct{}, destinationType.NumField())
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
		canonicalFields[name] = struct{}{}
	}
	return canonicalFields, nil
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

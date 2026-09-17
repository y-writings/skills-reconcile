package jsondoc

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ObjectSchema describes a product-owned JSON object and its canonical field names.
type ObjectSchema interface {
	CanonicalFieldNames() []string
}

// DecodeObject decodes one object into destination, rejects case aliases of
// its canonical fields, and reports other fields in sorted order for caller policy.
func DecodeObject[T ObjectSchema](data []byte, destination *T) ([]string, error) {
	if destination == nil {
		return nil, errors.New("JSON destination must not be nil")
	}
	if err := Validate(data); err != nil {
		return nil, err
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil || object == nil {
		return nil, errors.New("JSON value must be an object")
	}

	canonicalFields := (*destination).CanonicalFieldNames()
	canonical := make(map[string]struct{}, len(canonicalFields))
	for _, field := range canonicalFields {
		canonical[field] = struct{}{}
	}

	unknownFields := make([]string, 0)
	for field := range object {
		if _, known := canonical[field]; known {
			continue
		}
		for _, canonicalField := range canonicalFields {
			if strings.EqualFold(field, canonicalField) {
				return nil, fmt.Errorf("JSON field %q must use canonical spelling %q", field, canonicalField)
			}
		}
		unknownFields = append(unknownFields, field)
	}

	if err := json.Unmarshal(data, destination); err != nil {
		return nil, err
	}
	sort.Strings(unknownFields)
	return unknownFields, nil
}

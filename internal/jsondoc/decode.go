package jsondoc

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// DecodeObject decodes one object into destination, rejects case aliases of
// canonicalFields, and reports other fields in sorted order for caller policy.
func DecodeObject(data []byte, destination any, canonicalFields ...string) ([]string, error) {
	if err := Validate(data); err != nil {
		return nil, err
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil || object == nil {
		return nil, errors.New("JSON value must be an object")
	}

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

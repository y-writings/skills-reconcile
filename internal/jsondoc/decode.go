package jsondoc

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// UnmarshalCanonicalObject validates an object, rejects case aliases of known fields, and decodes it.
func UnmarshalCanonicalObject(data []byte, canonicalFields []string, destination any) error {
	if err := Validate(data); err != nil {
		return err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return errors.New("JSON value must be an object")
	}
	for field := range fields {
		for _, canonical := range canonicalFields {
			if field != canonical && strings.EqualFold(field, canonical) {
				return fmt.Errorf("non-canonical JSON field %q", field)
			}
		}
	}
	return json.Unmarshal(data, destination)
}

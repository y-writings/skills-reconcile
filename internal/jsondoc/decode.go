package jsondoc

import (
	"encoding/json"
	"errors"
)

// DecodeObject validates and decodes a JSON object into destination.
// Field matching, JSON tags, and unknown fields follow encoding/json.
func DecodeObject[T any](data []byte, destination *T) error {
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

	return json.Unmarshal(data, destination)
}

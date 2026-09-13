// Package jsondoc validates and decodes properties shared by JSON document readers.
package jsondoc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

// maxNestingDepth matches encoding/json's nesting limit.
const maxNestingDepth = 10_000

// Validate requires one valid UTF-8 JSON value with unique object field names.
func Validate(data []byte) error {
	if !utf8.Valid(data) {
		return errors.New("invalid UTF-8")
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := scanValue(decoder, 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}

func scanValue(decoder *json.Decoder, depth int) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	if depth >= maxNestingDepth {
		return errors.New("JSON exceeds maximum nesting depth")
	}

	seen := map[string]struct{}{}
	for decoder.More() {
		if delimiter == '{' {
			fieldToken, err := decoder.Token()
			if err != nil {
				return err
			}
			field := fieldToken.(string)
			if _, duplicate := seen[field]; duplicate {
				return fmt.Errorf("duplicate JSON field %q", field)
			}
			seen[field] = struct{}{}
		}
		if err := scanValue(decoder, depth+1); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}

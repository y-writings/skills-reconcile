package jsondoc

import (
	"strings"
	"testing"
)

func TestUnmarshalCanonicalObjectDecodesKnownFieldsAndAllowsUnknownFields(t *testing.T) {
	var destination struct {
		Name string
	}
	if err := UnmarshalCanonicalObject(
		[]byte("{\"name\":\"value\",\"future\":{\"enabled\":true}}"),
		[]string{"name"},
		&destination,
	); err != nil {
		t.Fatal(err)
	}
	if destination.Name != "value" {
		t.Fatalf("decoded name = %q, want value", destination.Name)
	}
}

func TestUnmarshalCanonicalObjectRejectsInvalidObject(t *testing.T) {
	for _, tc := range []struct {
		name, data, wantError string
		destination           any
	}{
		{"non-object", "null", "JSON value must be an object", &struct{}{}},
		{"non-canonical known field", "{\"Name\":\"value\"}", "non-canonical JSON field \"Name\"", &struct {
			Name string
		}{}},
		{"duplicate in unknown field", "{\"future\":{\"x\":1,\"\\u0078\":2}}", "duplicate JSON field \"x\"", &struct{}{}},
		{"destination type mismatch", "{\"count\":\"wrong\"}", "cannot unmarshal string", &struct {
			Count int
		}{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := UnmarshalCanonicalObject([]byte(tc.data), []string{"name", "count"}, tc.destination)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("UnmarshalCanonicalObject() error = %v, want error containing %q", err, tc.wantError)
			}
		})
	}
}

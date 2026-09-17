package jsondoc

import (
	"encoding/json"
	"reflect"
	"testing"
)

type objectFixture struct {
	Version *string `json:"schemaVersion"`
}

type rawObjectFixture struct {
	Value json.RawMessage `json:"value"`
}

type customObjectFixture struct {
	Value string `json:"value"`
}

type invalidTagFixture struct {
	Value string `json:"foo\\bar"`
}

type scalarFixture string

func (objectFixture) StrictJSONObjectSchema() {}

func (rawObjectFixture) StrictJSONObjectSchema() {}

func (customObjectFixture) StrictJSONObjectSchema() {}

func (invalidTagFixture) StrictJSONObjectSchema() {}

func (scalarFixture) StrictJSONObjectSchema() {}

func (*customObjectFixture) UnmarshalJSON([]byte) error {
	return nil
}

func TestDecodeObjectDecodesCanonicalFields(t *testing.T) {
	var got objectFixture
	if err := DecodeObject([]byte(`{"schemaVersion":"v2"}`), &got); err != nil {
		t.Fatalf("DecodeObject() error = %v, want nil", err)
	}
	if got.Version == nil {
		t.Fatal("decoded schemaVersion = nil, want a value")
	}
	if *got.Version != "v2" {
		t.Fatalf("decoded schemaVersion = %q, want %q", *got.Version, "v2")
	}
}

func TestDecodeObjectLeavesPresenceAndNullPolicyToCaller(t *testing.T) {
	for _, data := range []string{`{}`, `{"schemaVersion":null}`} {
		var got objectFixture
		if err := DecodeObject([]byte(data), &got); err != nil || got.Version != nil {
			t.Fatalf("DecodeObject(%s) = (%v, %v), want (nil, nil)", data, got.Version, err)
		}
	}
}

func TestDecodeObjectRejectsNonObjectRoots(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
	}{
		{"null", `null`},
		{"array", `[]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got objectFixture
			if err := DecodeObject([]byte(tc.data), &got); err == nil {
				t.Fatal("DecodeObject() error = nil, want non-object rejection")
			}
		})
	}
}

func TestDecodeObjectRejectsFieldsOutsideCanonicalSet(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
	}{
		{"unrelated field", `{"future":true}`},
		{"alias only", `{"SchemaVersion":"v2"}`},
		{"canonical and alias", `{"schemaVersion":"v2","SCHEMAVERSION":"other"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got objectFixture
			if err := DecodeObject([]byte(tc.data), &got); err == nil {
				t.Fatal("DecodeObject() error = nil, want unknown-field rejection")
			}
		})
	}
}

func TestDecodeObjectUsesDocumentIntegrityValidation(t *testing.T) {
	var got rawObjectFixture
	if err := DecodeObject([]byte(`{"value":{"x":1,"x":2}}`), &got); err == nil {
		t.Fatal("DecodeObject() error = nil, want duplicate-field rejection")
	}
}

func TestDecodeObjectReturnsTypedDecodeErrors(t *testing.T) {
	var got objectFixture
	if err := DecodeObject([]byte(`{"schemaVersion":42}`), &got); err == nil {
		t.Fatal("DecodeObject() error = nil, want field-type rejection")
	}
}

func TestDecodeObjectRejectsInvalidDestinations(t *testing.T) {
	for _, tc := range []struct {
		name   string
		decode func() error
	}{
		{"nil pointer", func() error {
			var destination *objectFixture
			return DecodeObject([]byte(`{}`), destination)
		}},
		{"non-struct", func() error {
			var destination scalarFixture
			return DecodeObject([]byte(`{}`), &destination)
		}},
		{"invalid tag name", func() error {
			var destination invalidTagFixture
			return DecodeObject([]byte(`{"foo\\bar":"decoded"}`), &destination)
		}},
		{"custom unmarshal", func() error {
			var destination customObjectFixture
			return DecodeObject([]byte(`{}`), &destination)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.decode(); err == nil {
				t.Fatal("DecodeObject() error = nil, want invalid-destination rejection")
			}
		})
	}
}

func TestObjectFieldNamesRejectsInvalidSchemas(t *testing.T) {
	for _, tc := range []struct {
		name            string
		destinationType reflect.Type
	}{
		{"missing tag", reflect.TypeFor[struct{ Value string }]()},
		{"case-conflicting tags", reflect.TypeFor[struct {
			Lower string `json:"value"`
			Upper string `json:"vALUE"`
		}]()},
		{"uppercase initial", reflect.TypeFor[struct {
			Value string `json:"Value"`
		}]()},
		{"non-ASCII character", reflect.TypeFor[struct {
			Value string `json:"valué"`
		}]()},
		{"embedded field", reflect.TypeFor[struct{ objectFixture }]()},
		{"empty tag name", reflect.TypeFor[struct {
			Value string `json:""`
		}]()},
		{"ignored field", reflect.TypeFor[struct {
			Value string `json:"-"`
		}]()},
		{"tag option", reflect.TypeFor[struct {
			Value string `json:"value,omitempty"`
		}]()},
		{"duplicate tag", reflect.StructOf([]reflect.StructField{
			{Name: "First", Type: reflect.TypeFor[string](), Tag: `json:"value"`},
			{Name: "Second", Type: reflect.TypeFor[string](), Tag: `json:"value"`},
		})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := objectFieldNames(tc.destinationType); err == nil {
				t.Fatal("objectFieldNames() error = nil, want invalid-schema rejection")
			}
		})
	}
}

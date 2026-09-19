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

type nestedObjectFixture struct {
	Child   nestedFieldFixture            `json:"child"`
	Pointer *nestedFieldFixture           `json:"pointer"`
	Items   []nestedFieldFixture          `json:"items"`
	ByName  map[string]nestedFieldFixture `json:"byName"`
}

type nestedFieldFixture struct {
	Known string `json:"known"`
}

type rawNestedObjectFixture struct {
	Value json.RawMessage `json:"value"`
}

type customNestedObjectFixture struct {
	Value customFieldFixture `json:"value"`
}

type customFieldFixture struct {
	Decoded string
}

type invalidNestedObjectFixture struct {
	Child struct {
		MissingTag string
	} `json:"child"`
}

func (*customObjectFixture) UnmarshalJSON([]byte) error {
	return nil
}

func (field *customFieldFixture) UnmarshalJSON(data []byte) error {
	field.Decoded = string(data)
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

func TestDecodeObjectValidatesNestedObjectSchemas(t *testing.T) {
	var got nestedObjectFixture
	data := []byte(`{"child":{"known":"child"},"pointer":{"known":"pointer"},"items":[{"known":"item"}],"byName":{"arbitrary":{"known":"map value"}}}`)
	if err := DecodeObject(data, &got); err != nil {
		t.Fatalf("DecodeObject() error = %v, want nil", err)
	}
	if got.Child.Known != "child" {
		t.Fatalf("child = %q, want %q", got.Child.Known, "child")
	}
	if got.Pointer == nil || got.Pointer.Known != "pointer" {
		t.Fatalf("pointer = %#v, want known pointer", got.Pointer)
	}
	if len(got.Items) != 1 || got.Items[0].Known != "item" {
		t.Fatalf("items = %#v, want one known item", got.Items)
	}
	if got.ByName["arbitrary"].Known != "map value" {
		t.Fatalf("byName = %#v, want arbitrary key with known value", got.ByName)
	}
}

func TestDecodeObjectRejectsUnknownFieldsInNestedObjectSchemas(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
	}{
		{"direct struct", `{"child":{"future":true}}`},
		{"pointer to struct", `{"pointer":{"future":true}}`},
		{"slice element", `{"items":[{"future":true}]}`},
		{"map value", `{"byName":{"arbitrary":{"future":true}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got nestedObjectFixture
			if err := DecodeObject([]byte(tc.data), &got); err == nil {
				t.Fatal("DecodeObject() error = nil, want nested unknown-field rejection")
			}
		})
	}
}

func TestDecodeObjectLeavesNestedNullPolicyToCaller(t *testing.T) {
	var got nestedObjectFixture
	if err := DecodeObject([]byte(`{"pointer":null}`), &got); err != nil || got.Pointer != nil {
		t.Fatalf("DecodeObject() = (%#v, %v), want (pointer: nil, nil)", got, err)
	}
}

func TestDecodeObjectLeavesRawAndCustomValuesToTheirDecoders(t *testing.T) {
	t.Run("raw message", func(t *testing.T) {
		var got rawNestedObjectFixture
		if err := DecodeObject([]byte(`{"value":{"future":true}}`), &got); err != nil {
			t.Fatalf("DecodeObject() error = %v, want nil", err)
		}
		if string(got.Value) != `{"future":true}` {
			t.Fatalf("raw value = %s, want preserved object", got.Value)
		}
	})

	t.Run("custom unmarshaler", func(t *testing.T) {
		var got customNestedObjectFixture
		if err := DecodeObject([]byte(`{"value":{"future":true}}`), &got); err != nil {
			t.Fatalf("DecodeObject() error = %v, want nil", err)
		}
		if got.Value.Decoded != `{"future":true}` {
			t.Fatalf("custom value = %s, want decoded object", got.Value.Decoded)
		}
	})
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
		{"invalid nested schema", func() error {
			var destination invalidNestedObjectFixture
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

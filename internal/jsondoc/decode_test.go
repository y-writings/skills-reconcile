package jsondoc

import (
	"encoding/json"
	"reflect"
	"testing"
)

type objectFixture struct {
	Directory *string `json:"workspace"`
}

type rawObjectFixture struct {
	Value json.RawMessage `json:"value"`
}

type customObjectFixture struct {
	Value string `json:"value"`
}

func (*customObjectFixture) UnmarshalJSON([]byte) error {
	return nil
}

func TestDecodeObjectDecodesCanonicalFields(t *testing.T) {
	var got objectFixture
	if err := DecodeObject([]byte(`{"workspace":"/workspace"}`), &got); err != nil {
		t.Fatalf("DecodeObject() error = %v, want nil", err)
	}
	if got.Directory == nil {
		t.Fatal("decoded workspace = nil, want a value")
	}
	if *got.Directory != "/workspace" {
		t.Fatalf("decoded workspace = %q, want %q", *got.Directory, "/workspace")
	}
}

func TestDecodeObjectLeavesPresenceAndNullPolicyToCaller(t *testing.T) {
	for _, data := range []string{`{}`, `{"workspace":null}`} {
		var got objectFixture
		if err := DecodeObject([]byte(data), &got); err != nil || got.Directory != nil {
			t.Fatalf("DecodeObject(%s) = (%v, %v), want (nil, nil)", data, got.Directory, err)
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
		{"alias only", `{"Workspace":"/other"}`},
		{"canonical and alias", `{"workspace":"/workspace","WORKSPACE":"/other"}`},
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
	if err := DecodeObject([]byte(`{"workspace":42}`), &got); err == nil {
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
			var destination string
			return DecodeObject([]byte(`{}`), &destination)
		}},
		{"missing tag", func() error {
			var destination struct{ Value string }
			return DecodeObject([]byte(`{}`), &destination)
		}},
		{"case-conflicting tags", func() error {
			var destination struct {
				Lower string `json:"value"`
				Upper string `json:"VALUE"`
			}
			return DecodeObject([]byte(`{}`), &destination)
		}},
		{"embedded field", func() error {
			var destination struct{ objectFixture }
			return DecodeObject([]byte(`{}`), &destination)
		}},
		{"empty tag name", func() error {
			var destination struct {
				Value string `json:""`
			}
			return DecodeObject([]byte(`{}`), &destination)
		}},
		{"ignored field", func() error {
			var destination struct {
				Value string `json:"-"`
			}
			return DecodeObject([]byte(`{}`), &destination)
		}},
		{"tag option", func() error {
			var destination struct {
				Value string `json:"value,omitempty"`
			}
			return DecodeObject([]byte(`{}`), &destination)
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

func TestObjectFieldNamesRejectsDuplicateTags(t *testing.T) {
	destinationType := reflect.StructOf([]reflect.StructField{
		{Name: "First", Type: reflect.TypeFor[string](), Tag: `json:"value"`},
		{Name: "Second", Type: reflect.TypeFor[string](), Tag: `json:"value"`},
	})
	if _, err := objectFieldNames(destinationType); err == nil {
		t.Fatal("objectFieldNames() error = nil, want duplicate-tag rejection")
	}
}

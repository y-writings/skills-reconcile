package jsondoc

import "testing"

type objectFixture struct {
	Version *string            `json:"schemaVersion,omitempty"`
	Child   nestedFieldFixture `json:"child,omitempty"`
}

type nestedFieldFixture struct {
	Known string `json:"known,omitempty"`
}

type recursiveValues []recursiveValues

type recursiveObjectFixture struct {
	Values recursiveValues `json:"values"`
}

type tagOptionFixture struct {
	Count int `json:"count,string"`
}

func TestDecodeObjectDecodesKnownFields(t *testing.T) {
	var got objectFixture
	if err := DecodeObject([]byte(`{"schemaVersion":"v2"}`), &got); err != nil {
		t.Fatalf("DecodeObject() error = %v, want nil", err)
	}
	if got.Version == nil || *got.Version != "v2" {
		t.Fatalf("decoded schemaVersion = %v, want %q", got.Version, "v2")
	}
}

func TestDecodeObjectIgnoresUnknownFields(t *testing.T) {
	var got objectFixture
	data := []byte(`{"schemaVersion":"v2","future":true,"child":{"known":"value","nestedFuture":42}}`)
	if err := DecodeObject(data, &got); err != nil {
		t.Fatalf("DecodeObject() error = %v, want nil", err)
	}
	if got.Version == nil || *got.Version != "v2" {
		t.Fatalf("decoded schemaVersion = %v, want %q", got.Version, "v2")
	}
	if got.Child.Known != "value" {
		t.Fatalf("decoded child.known = %q, want %q", got.Child.Known, "value")
	}
}

func TestDecodeObjectUsesStandardCaseInsensitiveFieldMatching(t *testing.T) {
	var got objectFixture
	if err := DecodeObject([]byte(`{"SCHEMAVERSION":"v2"}`), &got); err != nil {
		t.Fatalf("DecodeObject() error = %v, want nil", err)
	}
	if got.Version == nil || *got.Version != "v2" {
		t.Fatalf("decoded schemaVersion = %v, want %q", got.Version, "v2")
	}
}

func TestDecodeObjectUsesStandardJSONTagOptions(t *testing.T) {
	var got tagOptionFixture
	if err := DecodeObject([]byte(`{"count":"42"}`), &got); err != nil {
		t.Fatalf("DecodeObject() error = %v, want nil", err)
	}
	if got.Count != 42 {
		t.Fatalf("decoded count = %d, want 42", got.Count)
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

func TestDecodeObjectValidatesIgnoredFields(t *testing.T) {
	var got objectFixture
	if err := DecodeObject([]byte(`{"future":{"x":1,"x":2}}`), &got); err == nil {
		t.Fatal("DecodeObject() error = nil, want duplicate-field rejection")
	}
}

func TestDecodeObjectReturnsTypedDecodeErrors(t *testing.T) {
	var got objectFixture
	if err := DecodeObject([]byte(`{"schemaVersion":42}`), &got); err == nil {
		t.Fatal("DecodeObject() error = nil, want field-type rejection")
	}
}

func TestDecodeObjectRejectsNilDestination(t *testing.T) {
	var destination *objectFixture
	if err := DecodeObject([]byte(`{}`), destination); err == nil {
		t.Fatal("DecodeObject() error = nil, want nil-destination rejection")
	}
}

func TestDecodeObjectHandlesRecursiveDestinationTypes(t *testing.T) {
	var got recursiveObjectFixture
	if err := DecodeObject([]byte(`{"values":[]}`), &got); err != nil {
		t.Fatalf("DecodeObject() error = %v, want nil", err)
	}
	if len(got.Values) != 0 {
		t.Fatalf("decoded values = %#v, want empty", got.Values)
	}
}

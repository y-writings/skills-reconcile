package jsondoc

import (
	"slices"
	"testing"
)

type objectFixture struct {
	Workspace *string `json:"workspace"`
}

func (objectFixture) CanonicalFieldNames() []string {
	return []string{"workspace"}
}

func TestDecodeObjectDecodesCanonicalFieldsAndReportsUnknownFields(t *testing.T) {
	var got objectFixture
	unknownFields, err := DecodeObject(
		[]byte(`{"future":true,"workspace":"/workspace","another":42}`),
		&got,
	)
	if err != nil {
		t.Fatalf("DecodeObject() error = %v, want nil", err)
	}
	if got.Workspace == nil {
		t.Fatal("decoded workspace = nil, want a value")
	}
	if *got.Workspace != "/workspace" {
		t.Fatalf("decoded workspace = %q, want %q", *got.Workspace, "/workspace")
	}
	if want := []string{"another", "future"}; !slices.Equal(unknownFields, want) {
		t.Fatalf("unknown fields = %q, want %q", unknownFields, want)
	}
}

func TestDecodeObjectLeavesNullPolicyToCaller(t *testing.T) {
	var got objectFixture
	unknownFields, err := DecodeObject([]byte(`{"workspace":null}`), &got)
	if err != nil || got.Workspace != nil || len(unknownFields) != 0 {
		t.Fatalf("DecodeObject() = (%v, %q, %v), want (nil, [], nil)", got.Workspace, unknownFields, err)
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
			if _, err := DecodeObject([]byte(tc.data), &got); err == nil {
				t.Fatal("DecodeObject() error = nil, want non-object rejection")
			}
		})
	}
}

func TestDecodeObjectRejectsKnownFieldCaseAliases(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
	}{
		{"alias only", `{"Workspace":"/other"}`},
		{"canonical and alias", `{"workspace":"/workspace","WORKSPACE":"/other"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got objectFixture
			if _, err := DecodeObject([]byte(tc.data), &got); err == nil {
				t.Fatal("DecodeObject() error = nil, want case-alias rejection")
			}
		})
	}
}

func TestDecodeObjectUsesDocumentIntegrityValidation(t *testing.T) {
	var got objectFixture
	if _, err := DecodeObject(
		[]byte(`{"workspace":"/first","workspace":"/second"}`),
		&got,
	); err == nil {
		t.Fatal("DecodeObject() error = nil, want duplicate-field rejection")
	}
}

func TestDecodeObjectReturnsTypedDecodeErrors(t *testing.T) {
	var got objectFixture
	if _, err := DecodeObject([]byte(`{"workspace":42}`), &got); err == nil {
		t.Fatal("DecodeObject() error = nil, want field-type rejection")
	}
}

func TestDecodeObjectRejectsNilDestination(t *testing.T) {
	var destination *objectFixture
	if _, err := DecodeObject([]byte(`{}`), destination); err == nil {
		t.Fatal("DecodeObject() error = nil, want nil destination rejection")
	}
}

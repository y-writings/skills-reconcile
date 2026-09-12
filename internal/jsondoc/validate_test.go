package jsondoc

import (
	"strings"
	"testing"
)

func TestValidateAcceptsOneUnambiguousJSONValue(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"null scalar", []byte(`null`)},
		{"surrounding whitespace", []byte(" \n{}\t")},
		{"same field in separate objects", []byte(`{"x":{"x":1},"items":[{"x":2},{"x":3}]}`)},
		{"large number", []byte(`{"number":1e1000}`)},
		{"replacement character", []byte(`{"x":"�"}`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := Validate(tc.data); err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
		})
	}
}

func TestValidateRejectsInvalidJSONDocument(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"empty input", nil},
		{"whitespace only", []byte(" \n\t")},
		{"incomplete value", []byte(`{`)},
		{"trailing value", []byte(`{} []`)},
		{"invalid UTF-8", []byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := Validate(tc.data); err == nil {
				t.Fatal("Validate() error = nil, want rejection")
			}
		})
	}
}

func TestValidateRejectsDuplicateObjectFields(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"top level", []byte(`{"x":1,"x":2}`)},
		{"escaped name", []byte(`{"x":1,"\u0078":2}`)},
		{"nested object", []byte(`{"outer":{"x":1,"x":2}}`)},
		{"object in array", []byte(`[{"x":1,"x":2}]`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := Validate(tc.data); err == nil {
				t.Fatal("Validate() error = nil, want duplicate-field rejection")
			}
		})
	}
}

func TestValidateAppliesJSONNestingLimit(t *testing.T) {
	const standardJSONMaxNestingDepth = 10_000

	for _, tc := range []struct {
		name    string
		data    string
		wantErr bool
	}{
		{"array at limit", strings.Repeat(`[`, standardJSONMaxNestingDepth) + strings.Repeat(`]`, standardJSONMaxNestingDepth), false},
		{"array beyond limit", strings.Repeat(`[`, standardJSONMaxNestingDepth+1) + strings.Repeat(`]`, standardJSONMaxNestingDepth+1), true},
		{"object at limit", strings.Repeat(`{"x":`, standardJSONMaxNestingDepth) + `0` + strings.Repeat(`}`, standardJSONMaxNestingDepth), false},
		{"object beyond limit", strings.Repeat(`{"x":`, standardJSONMaxNestingDepth+1) + `0` + strings.Repeat(`}`, standardJSONMaxNestingDepth+1), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate([]byte(tc.data))
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %t", err, tc.wantErr)
			}
		})
	}
}

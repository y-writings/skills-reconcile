package workspace

import "testing"

func TestDecodeWorkspaceConfig(t *testing.T) {
	for _, tc := range []struct {
		name, data, want string
		found            bool
	}{
		{"missing workspace", " \n{}\t", "", false},
		{"empty workspace", `{"workspace":""}`, "", false},
		{"configured workspace", `{"workspace":"/workspace"}`, "/workspace", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, found, err := decodeWorkspaceConfig([]byte(tc.data))
			if err != nil || got != tc.want || found != tc.found {
				t.Fatalf("decodeWorkspaceConfig() = (%q, %t, %v), want (%q, %t, nil)", got, found, err, tc.want, tc.found)
			}
		})
	}
}

func TestDecodeWorkspaceConfigRejectsInvalidConfig(t *testing.T) {
	for _, tc := range []struct{ name, data string }{
		{"empty input", ""},
		{"malformed JSON", `{`},
		{"null config", `null`},
		{"trailing JSON", `{} {}`},
		{"null workspace", `{"workspace":null}`},
		{"wrong workspace type", `{"workspace":42}`},
		{"unknown field", `{"workpace":"/workspace"}`},
		{"case variant", `{"workspace":"/workspace","Workspace":"/other"}`},
		{"duplicate workspace", `{"workspace":"/first","workspace":"/second"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			workspaceDir, found, err := decodeWorkspaceConfig([]byte(tc.data))
			if err != errInvalidConfig || workspaceDir != "" || found {
				t.Fatalf("decodeWorkspaceConfig() = (%q, %t, %v), want (\"\", false, %v)", workspaceDir, found, err, errInvalidConfig)
			}
		})
	}
}

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
		{"invalid UTF-8", "{\"workspace\":\"/work\xffspace\"}"},
		{"null config", `null`},
		{"null workspace", `{"workspace":null}`},
		{"wrong workspace type", `{"workspace":42}`},
		{"unknown field", `{"workpace":"/workspace"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			workspaceDir, found, err := decodeWorkspaceConfig([]byte(tc.data))
			if err != errInvalidConfig || workspaceDir != "" || found {
				t.Fatalf("decodeWorkspaceConfig() = (%q, %t, %v), want (\"\", false, %v)", workspaceDir, found, err, errInvalidConfig)
			}
		})
	}
}

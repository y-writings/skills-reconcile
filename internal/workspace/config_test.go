package workspace

import "testing"

func TestDecodeWorkspaceConfig(t *testing.T) {
	for _, tc := range []struct {
		name, data, want string
		found            bool
	}{
		{"missing workspace", " \n{}\t", "", false},
		{"empty workspace", `{"workspace":""}`, "", false},
		{"null workspace", `{"workspace":null}`, "", false},
		{"configured workspace", `{"workspace":"/workspace"}`, "/workspace", true},
		{"configured with unknown field", `{"workspace":"/workspace","future":true}`, "/workspace", true},
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
	workspaceDir, found, err := decodeWorkspaceConfig([]byte(`{"workspace":42}`))
	if err != errInvalidConfig || workspaceDir != "" || found {
		t.Fatalf("decodeWorkspaceConfig() = (%q, %t, %v), want (\"\", false, %v)", workspaceDir, found, err, errInvalidConfig)
	}
}

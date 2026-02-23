package modules

import (
	"testing"
)

func TestToJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    string
		wantErr bool
	}{
		{"map", map[string]string{"a": "b"}, `{"a":"b"}`, false},
		{"struct", struct {
			Name string `json:"name"`
		}{Name: "test"}, `{"name":"test"}`, false},
		{"nil", nil, "null", false},
		{"number", 42, "42", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToJSON(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ToJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name  string
		input []interface{}
		want  int
	}{
		{"all strings", []interface{}{"a", "b", "c"}, 3},
		{"mixed types", []interface{}{"a", 42, true, "b"}, 2},
		{"empty", []interface{}{}, 0},
		{"no strings", []interface{}{1, 2, 3}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToStringSlice(tt.input)
			if len(got) != tt.want {
				t.Errorf("len(ToStringSlice()) = %d, want %d", len(got), tt.want)
			}
		})
	}
}

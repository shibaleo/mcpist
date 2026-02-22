package db

import (
	"encoding/json"
	"testing"
)

func TestJSONBValue(t *testing.T) {
	tests := []struct {
		name string
		j    JSONB
		want string
	}{
		{"valid json", JSONB(`{"key":"value"}`), `{"key":"value"}`},
		{"empty", JSONB(nil), "{}"},
		{"empty bytes", JSONB([]byte{}), "{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := tt.j.Value()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if val != tt.want {
				t.Errorf("Value() = %q, want %q", val, tt.want)
			}
		})
	}
}

func TestJSONBScan(t *testing.T) {
	t.Run("from bytes", func(t *testing.T) {
		var j JSONB
		if err := j.Scan([]byte(`{"a":1}`)); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		if string(j) != `{"a":1}` {
			t.Errorf("got %q", string(j))
		}
	})

	t.Run("from string", func(t *testing.T) {
		var j JSONB
		if err := j.Scan(`{"b":2}`); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		if string(j) != `{"b":2}` {
			t.Errorf("got %q", string(j))
		}
	})

	t.Run("from nil", func(t *testing.T) {
		var j JSONB
		if err := j.Scan(nil); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		if string(j) != "{}" {
			t.Errorf("got %q, want %q", string(j), "{}")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		var j JSONB
		if err := j.Scan(42); err == nil {
			t.Error("expected error for unsupported type")
		}
	})
}

func TestJSONBMarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		j    JSONB
		want string
	}{
		{"valid json", JSONB(`{"key":"value"}`), `{"key":"value"}`},
		{"empty", JSONB(nil), "{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.j.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON failed: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("MarshalJSON() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestJSONBUnmarshalJSON(t *testing.T) {
	var j JSONB
	data := `{"x":"y"}`
	if err := json.Unmarshal([]byte(data), &j); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if string(j) != data {
		t.Errorf("got %q, want %q", string(j), data)
	}
}

package broker

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFlexibleTimeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"unix timestamp", `1700000000`, 1700000000, false},
		{"iso 8601 string", `"2024-01-15T12:00:00Z"`, 1705320000, false},
		{"iso 8601 with millis", `"2024-01-15T12:00:00.000Z"`, 1705320000, false},
		{"empty string", `""`, 0, false},
		{"invalid string", `"not-a-date"`, 0, true},
		{"null (JSON null unmarshals as zero int64)", `null`, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ft FlexibleTime
			err := json.Unmarshal([]byte(tt.input), &ft)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if int64(ft) != tt.want {
				t.Errorf("got %d, want %d", int64(ft), tt.want)
			}
		})
	}
}

func TestNeedsRefresh(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name string
		creds *Credentials
		want  bool
	}{
		{
			"expired",
			&Credentials{ExpiresAt: FlexibleTime(now - 100)},
			true,
		},
		{
			"within buffer (expires in 2 min, buffer is 5 min)",
			&Credentials{ExpiresAt: FlexibleTime(now + 120)},
			true,
		},
		{
			"well before buffer (expires in 10 min)",
			&Credentials{ExpiresAt: FlexibleTime(now + 600)},
			false,
		},
		{
			"ExpiresAt is zero (no expiry)",
			&Credentials{ExpiresAt: 0},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsRefresh(tt.creds); got != tt.want {
				t.Errorf("needsRefresh() = %v, want %v", got, tt.want)
			}
		})
	}
}

package modules

import (
	"testing"
)

func TestValidateParams_RequiredFields(t *testing.T) {
	schema := InputSchema{
		Type: "object",
		Properties: map[string]Property{
			"owner": {Type: "string", Description: "Repository owner"},
			"repo":  {Type: "string", Description: "Repository name"},
		},
		Required: []string{"owner", "repo"},
	}

	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name:    "all required present",
			params:  map[string]any{"owner": "octocat", "repo": "hello-world"},
			wantErr: false,
		},
		{
			name:    "missing one required",
			params:  map[string]any{"owner": "octocat"},
			wantErr: true,
			errMsg:  "missing required parameter(s): repo",
		},
		{
			name:    "missing all required",
			params:  map[string]any{},
			wantErr: true,
			errMsg:  "missing required parameter(s): owner, repo",
		},
		{
			name:    "nil params",
			params:  nil,
			wantErr: true,
			errMsg:  "missing required parameter(s): owner, repo",
		},
		{
			name:    "empty string for required field",
			params:  map[string]any{"owner": "", "repo": "hello-world"},
			wantErr: true,
			errMsg:  "missing required parameter(s): owner",
		},
		{
			name:    "nil value for required field",
			params:  map[string]any{"owner": nil, "repo": "hello-world"},
			wantErr: true,
			errMsg:  "missing required parameter(s): owner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateParams(schema, tt.params)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateParams_TypeCheck(t *testing.T) {
	schema := InputSchema{
		Type: "object",
		Properties: map[string]Property{
			"name":     {Type: "string"},
			"count":    {Type: "number"},
			"enabled":  {Type: "boolean"},
			"tags":     {Type: "array"},
			"metadata": {Type: "object"},
		},
	}

	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name:    "all correct types",
			params:  map[string]any{"name": "test", "count": float64(5), "enabled": true, "tags": []interface{}{"a"}, "metadata": map[string]interface{}{"k": "v"}},
			wantErr: false,
		},
		{
			name:    "string where number expected",
			params:  map[string]any{"count": "five"},
			wantErr: true,
			errMsg:  `parameter "count": expected number, got string`,
		},
		{
			name:    "number where string expected",
			params:  map[string]any{"name": float64(42)},
			wantErr: true,
			errMsg:  `parameter "name": expected string, got float64`,
		},
		{
			name:    "string where boolean expected",
			params:  map[string]any{"enabled": "true"},
			wantErr: true,
			errMsg:  `parameter "enabled": expected boolean, got string`,
		},
		{
			name:    "string where array expected",
			params:  map[string]any{"tags": "not-array"},
			wantErr: true,
			errMsg:  `parameter "tags": expected array, got string`,
		},
		{
			name:    "string where object expected",
			params:  map[string]any{"metadata": "not-object"},
			wantErr: true,
			errMsg:  `parameter "metadata": expected object, got string`,
		},
		{
			name:    "extra params not in schema pass through",
			params:  map[string]any{"unknown_field": "whatever"},
			wantErr: false,
		},
		{
			name:    "nil value skips type check",
			params:  map[string]any{"name": nil},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateParams(schema, tt.params)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateParams_NoRequiredNoProperties(t *testing.T) {
	// Schema with no required and no properties (e.g., get_user)
	schema := InputSchema{
		Type:       "object",
		Properties: map[string]Property{},
	}

	result, err := ValidateParams(schema, map[string]any{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Errorf("expected non-nil result")
	}
}

func TestValidateParams_IntegerType(t *testing.T) {
	schema := InputSchema{
		Type: "object",
		Properties: map[string]Property{
			"page": {Type: "integer"},
		},
	}

	// float64 is accepted for "integer" (JSON numbers are always float64)
	_, err := ValidateParams(schema, map[string]any{"page": float64(3)})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// string is rejected for "integer"
	_, err = ValidateParams(schema, map[string]any{"page": "three"})
	if err == nil {
		t.Errorf("expected error for string as integer")
	}
}

func TestFindTool(t *testing.T) {
	tools := []Tool{
		{Name: "get_user", ID: "github:get_user"},
		{Name: "list_repos", ID: "github:list_repos"},
	}

	tool, found := findTool(tools, "list_repos")
	if !found {
		t.Fatal("expected to find list_repos")
	}
	if tool.ID != "github:list_repos" {
		t.Errorf("expected ID github:list_repos, got %s", tool.ID)
	}

	_, found = findTool(tools, "nonexistent")
	if found {
		t.Error("expected not to find nonexistent tool")
	}
}

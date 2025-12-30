package config

import (
	"reflect"
	"testing"
)

func TestValidateModelInList(t *testing.T) {
	v := NewModelValidator()

	tests := []struct {
		name    string
		model   string
		models  []string
		wantErr bool
	}{
		{
			name:    "model exists in list",
			model:   "claude-3-opus",
			models:  []string{"claude-3-opus", "claude-3-sonnet", "gpt-4"},
			wantErr: false,
		},
		{
			name:    "model not in list",
			model:   "gpt-5",
			models:  []string{"claude-3-opus", "claude-3-sonnet", "gpt-4"},
			wantErr: true,
		},
		{
			name:    "empty model name",
			model:   "",
			models:  []string{"claude-3-opus"},
			wantErr: true,
		},
		{
			name:    "model with whitespace matches trimmed",
			model:   "  claude-3-opus  ",
			models:  []string{"claude-3-opus", "claude-3-sonnet"},
			wantErr: false,
		},
		{
			name:    "list item with whitespace matches trimmed model",
			model:   "claude-3-opus",
			models:  []string{"  claude-3-opus  ", "claude-3-sonnet"},
			wantErr: false,
		},
		{
			name:    "empty list",
			model:   "claude-3-opus",
			models:  []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateModelInList(tt.model, tt.models)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateModelInList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateModelsList(t *testing.T) {
	v := NewModelValidator()

	tests := []struct {
		name    string
		models  []string
		wantErr bool
	}{
		{
			name:    "valid non-empty list",
			models:  []string{"claude-3-opus", "claude-3-sonnet"},
			wantErr: false,
		},
		{
			name:    "single model list",
			models:  []string{"claude-3-opus"},
			wantErr: false,
		},
		{
			name:    "empty list",
			models:  []string{},
			wantErr: true,
		},
		{
			name:    "nil list",
			models:  nil,
			wantErr: true,
		},
		{
			name:    "list with only empty strings",
			models:  []string{"", "  ", "   "},
			wantErr: true,
		},
		{
			name:    "list with one valid model among empty strings",
			models:  []string{"", "claude-3-opus", "  "},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateModelsList(tt.models)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateModelsList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeModels(t *testing.T) {
	v := NewModelValidator()

	tests := []struct {
		name   string
		models []string
		want   []string
	}{
		{
			name:   "no changes needed",
			models: []string{"claude-3-opus", "claude-3-sonnet"},
			want:   []string{"claude-3-opus", "claude-3-sonnet"},
		},
		{
			name:   "trim whitespace",
			models: []string{"  claude-3-opus  ", " claude-3-sonnet "},
			want:   []string{"claude-3-opus", "claude-3-sonnet"},
		},
		{
			name:   "remove duplicates",
			models: []string{"claude-3-opus", "claude-3-sonnet", "claude-3-opus"},
			want:   []string{"claude-3-opus", "claude-3-sonnet"},
		},
		{
			name:   "remove duplicates with whitespace",
			models: []string{"claude-3-opus", "  claude-3-opus  ", "claude-3-sonnet"},
			want:   []string{"claude-3-opus", "claude-3-sonnet"},
		},
		{
			name:   "remove empty strings",
			models: []string{"claude-3-opus", "", "claude-3-sonnet", "  "},
			want:   []string{"claude-3-opus", "claude-3-sonnet"},
		},
		{
			name:   "empty input",
			models: []string{},
			want:   []string{},
		},
		{
			name:   "nil input",
			models: nil,
			want:   []string{},
		},
		{
			name:   "all empty strings",
			models: []string{"", "  ", "   "},
			want:   []string{},
		},
		{
			name:   "preserve order",
			models: []string{"gpt-4", "claude-3-opus", "claude-3-sonnet"},
			want:   []string{"gpt-4", "claude-3-opus", "claude-3-sonnet"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.NormalizeModels(tt.models)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NormalizeModels() = %v, want %v", got, tt.want)
			}
		})
	}
}

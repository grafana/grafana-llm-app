package plugin

import (
	"testing"
)

func TestModelFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected Model
		wantErr  bool
	}{
		{
			input:    "default",
			expected: ModelDefault,
			wantErr:  false,
		},
		{
			input:    "gpt-3.5-turbo",
			expected: ModelDefault,
			wantErr:  false,
		},
		{
			input:    "gpt-3.5-turbo-0125",
			expected: ModelDefault,
			wantErr:  false,
		},
		{
			input:    "high-accuracy",
			expected: ModelHighAccuracy,
			wantErr:  false,
		},
		{
			input:    "gpt-4",
			expected: ModelHighAccuracy,
			wantErr:  false,
		},
		{
			input:    "gpt-4-turbo",
			expected: ModelHighAccuracy,
			wantErr:  false,
		},
		{
			input:    "gpt-4-turbo-2024-04-09",
			expected: ModelHighAccuracy,
			wantErr:  false,
		},
		{
			input:    "invalid_model",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ModelFromString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ModelFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ModelFromString() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

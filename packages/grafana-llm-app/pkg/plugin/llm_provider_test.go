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
			input:    "small",
			expected: ModelSmall,
			wantErr:  false,
		},
		{
			input:    "medium",
			expected: ModelMedium,
			wantErr:  false,
		},
		{
			input:    "large",
			expected: ModelLarge,
			wantErr:  false,
		},

		// unknown models
		{
			input:    "invalid_model",
			expected: "",
			wantErr:  true,
		},
		{
			input:    "",
			expected: "",
			wantErr:  true,
		},

		// backwards-compatibility
		{
			input:    "gpt-3.5-turbo",
			expected: ModelSmall,
			wantErr:  false,
		},
		{
			input:    "gpt-3.5-turbo-0125",
			expected: ModelSmall,
			wantErr:  false,
		},
		{
			input:    "gpt-4-turbo",
			expected: ModelMedium,
			wantErr:  false,
		},
		{
			input:    "gpt-4-turbo-2024-04-09",
			expected: ModelMedium,
			wantErr:  false,
		},
		{
			input:    "gpt-4",
			expected: ModelLarge,
			wantErr:  false,
		},
		{
			input:    "gpt-4-32k-0613",
			expected: ModelLarge,
			wantErr:  false,
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

func TestUnmarshalJSON(t *testing.T) {
	tests := []struct {
		input    []byte
		expected Model
		wantErr  bool
	}{
		{
			input:    []byte(`"small"`),
			expected: ModelSmall,
			wantErr:  false,
		},
		{
			input:    []byte(`"medium"`),
			expected: ModelMedium,
			wantErr:  false,
		},
		{
			input:    []byte(`"large"`),
			expected: ModelLarge,
			wantErr:  false,
		},

		// unknown models
		{
			input:    []byte(`"invalid_model"`),
			expected: "",
			wantErr:  true,
		},
		{
			input:    []byte(`""`),
			expected: "",
			wantErr:  true,
		},
		{
			input:    []byte(`null`),
			expected: "",
			wantErr:  true,
		},

		// backwards-compatibility
		{
			input:    []byte(`"gpt-3.5-turbo"`),
			expected: ModelSmall,
			wantErr:  false,
		},
		{
			input:    []byte(`"gpt-3.5-turbo-0125"`),
			expected: ModelSmall,
			wantErr:  false,
		},
		{
			input:    []byte(`"gpt-4-turbo"`),
			expected: ModelMedium,
			wantErr:  false,
		},
		{
			input:    []byte(`"gpt-4-turbo-2024-04-09"`),
			expected: ModelMedium,
			wantErr:  false,
		},
		{
			input:    []byte(`"gpt-4"`),
			expected: ModelLarge,
			wantErr:  false,
		},
		{
			input:    []byte(`"gpt-4-32k-0613"`),
			expected: ModelLarge,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			var m Model
			err := m.UnmarshalJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if m != tt.expected {
				t.Errorf("UnmarshalJSON() = %v, expected %v", m, tt.expected)
			}
		})
	}
}

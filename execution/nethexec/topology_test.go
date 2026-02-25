package nethexec

import (
	"testing"
)

func TestGetExecutionMode_AllModes(t *testing.T) {
	tests := []struct {
		input    string
		expected ExecutionMode
	}{
		{"external-execution", ModeExternalExecution},
		{"external-sequencer", ModeExternalSequencer},
		{"external-execution-compare", ModeExternalExecutionCompare},
		{"external-sequencer-compare", ModeExternalSequencerCompare},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := GetExecutionMode(tc.input)
			if got != tc.expected {
				t.Errorf("GetExecutionMode(%q) = %d, want %d", tc.input, got, tc.expected)
			}
		})
	}
}

func TestGetExecutionMode_Default(t *testing.T) {
	tests := []string{"", "unknown", "internal", "foo"}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got := GetExecutionMode(input)
			if got != ModeInternalOnly {
				t.Errorf("GetExecutionMode(%q) = %d, want ModeInternalOnly (%d)", input, got, ModeInternalOnly)
			}
		})
	}
}

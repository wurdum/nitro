package nethexec

// ExecutionMode controls how the wrapper uses internal vs external execution layer
type ExecutionMode uint8

const (
	ModeInternalOnly      ExecutionMode = iota // Use built-in Geth execution client (default)
	ModeExternalExecution                      // Delegate ExecutionClient to Nethermind, sequencing optionally internal
	ModeExternalSequencer                      // Delegate ExecutionSequencer (and ExecutionClient) to Nethermind
)

// GetExecutionMode reads from config string and returns the execution mode
func GetExecutionMode(modeStr string) ExecutionMode {
	switch modeStr {
	case "external-execution":
		return ModeExternalExecution
	case "external-sequencer":
		return ModeExternalSequencer
	default:
		return ModeInternalOnly
	}
}

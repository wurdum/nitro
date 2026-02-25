package nethexec

// ExecutionMode controls how the wrapper uses internal vs external execution layer
type ExecutionMode uint8

const (
	ModeInternalOnly             ExecutionMode = iota // Use built-in Geth execution client (default)
	ModeExternalExecution                             // Delegate ExecutionClient to Nethermind, sequencing optionally internal
	ModeExternalSequencer                             // Delegate ExecutionSequencer (and ExecutionClient) to Nethermind
	ModeExternalExecutionCompare                      // Run both Geth and Nethermind ExecutionClients, compare results
	ModeExternalSequencerCompare                      // Run both Geth and Nethermind ExecutionSequencers, compare results
)

// GetExecutionMode reads from config string and returns the execution mode
func GetExecutionMode(modeStr string) ExecutionMode {
	switch modeStr {
	case "", "internal":
		return ModeInternalOnly
	case "external-execution":
		return ModeExternalExecution
	case "external-sequencer":
		return ModeExternalSequencer
	case "external-execution-compare":
		return ModeExternalExecutionCompare
	case "external-sequencer-compare":
		return ModeExternalSequencerCompare
	default:
		return ModeInternalOnly
	}
}

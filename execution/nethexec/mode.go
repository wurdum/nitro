package nethexec

// ExecutionMode controls how the wrapper uses internal vs external execution layer
type ExecutionMode uint8

const (
	ModeInternalOnly ExecutionMode = iota // Use built-in Geth execution client (default)
	ModeExternalOnly                      // Use external Nethermind execution client
)

// GetExecutionMode reads from config string and returns the execution mode
func GetExecutionMode(modeStr string) ExecutionMode {
	switch modeStr {
	case "external", "nethermind":
		return ModeExternalOnly
	default:
		return ModeInternalOnly
	}
}

package rpctypes

// TraceOptions is method input, not a node feature toggle. Defaults select
// ordinary opcode output; named call tracing follows its explicit options.
type TraceOptions struct {
	Tracer           string
	OnlyTopCall      bool
	WithLog          bool
	Timeout          string
	DisableStack     bool
	DisableStorage   bool
	EnableMemory     bool
	EnableReturnData bool
	Limit            int
	StateOverrides   StateOverride
	BlockOverrides   *SimulateBlockOverrides
}

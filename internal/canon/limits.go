package canon

// The v1 resource profile is part of canonicalization semantics. These limits
// are deliberately explicit so hostile identity input cannot turn a comparison
// into an unbounded scanner, parser, or reflection walk.
const (
	MaxInputBytes       = 1 << 20       // 1 MiB of untrusted JSON source.
	MaxTokenCount       = 1 << 17       // Includes the terminal EOF token.
	MaxContainerMembers = 1 << 14       // Per array, object, slice, map, or struct.
	MaxTypedNodes       = MaxTokenCount // Parsed values cannot outrun their admitted token stream.
	MaxTypedStringBytes = 1 << 20       // Aggregate UTF-8 bytes in values and names.
)

type traversalBudget struct {
	nodes       int
	stringBytes int
}

func (b *traversalBudget) enter() error {
	b.nodes++
	if b.nodes > MaxTypedNodes {
		return refusal(CodeTypedNodeLimit, UnknownOffset, "value exceeds the typed node ceiling")
	}
	return nil
}

func (b *traversalBudget) addString(value string) error {
	length := len(value)
	if length > MaxTypedStringBytes-b.stringBytes {
		return refusal(CodeTypedByteLimit, UnknownOffset, "value exceeds the aggregate typed string-byte ceiling")
	}
	b.stringBytes += length
	return nil
}

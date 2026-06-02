package resultshape

// Shaper transforms query rows before they are returned through MCP.
// The first implementation is passthrough only.
type Shaper interface {
	ShapeRows(rows []map[string]any) ([]map[string]any, error)
}

// Passthrough returns rows unchanged.
type Passthrough struct{}

func (Passthrough) ShapeRows(rows []map[string]any) ([]map[string]any, error) {
	return rows, nil
}

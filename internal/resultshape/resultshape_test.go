package resultshape

import "testing"

func TestPassthroughShapeRows(t *testing.T) {
	rows := []map[string]any{{"id": 1}}
	out, err := Passthrough{}.ShapeRows(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0]["id"] != 1 {
		t.Fatalf("out = %#v", out)
	}
}

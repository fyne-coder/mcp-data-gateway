package auth

import (
	"encoding/json"
	"fmt"
)

func parseGroupsClaim(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var groups []string
	if err := json.Unmarshal(raw, &groups); err == nil {
		return groups, nil
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil && single != "" {
		return []string{single}, nil
	}
	var mixed []any
	if err := json.Unmarshal(raw, &mixed); err != nil {
		return nil, fmt.Errorf("group claim is not a string or string array")
	}
	out := make([]string, 0, len(mixed))
	for _, item := range mixed {
		s, ok := item.(string)
		if !ok || s == "" {
			return nil, fmt.Errorf("group claim contains a non-string value")
		}
		out = append(out, s)
	}
	return out, nil
}

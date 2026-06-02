package auth

// Authorize returns ErrForbidden when the actor lacks any required group.
func Authorize(actor Actor, requiredGroups []string) error {
	if len(requiredGroups) == 0 {
		// Fail closed if callers bypass config validation.
		return ErrForbidden
	}
	groupSet := make(map[string]struct{}, len(actor.Groups))
	for _, g := range actor.Groups {
		groupSet[g] = struct{}{}
	}
	for _, required := range requiredGroups {
		if _, ok := groupSet[required]; ok {
			return nil
		}
	}
	return ErrForbidden
}

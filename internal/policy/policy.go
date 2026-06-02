package policy

const ToolPostgresSelect = "postgres_select"

type Actor struct {
	Subject string
	Groups  []string
}

type Request struct {
	Actor Actor
	Tool  string
}

type Decision struct {
	Allowed  bool
	ToolPack string
	Reason   string
}

type Engine struct {
	DefaultToolPack string
	GroupToolPacks  map[string][]string
}

// ResolveToolPack returns the first tool pack mapped from the actor's groups.
func (e Engine) ResolveToolPack(actor Actor) (string, bool) {
	for _, group := range actor.Groups {
		for _, pack := range e.GroupToolPacks[group] {
			if pack != "" {
				return pack, true
			}
		}
	}
	return "", false
}

func (e Engine) Decide(req Request) Decision {
	pack, ok := e.ResolveToolPack(req.Actor)
	if !ok {
		return Decision{Allowed: false, Reason: "no group mapped to a tool pack"}
	}
	if req.Tool == "" {
		return Decision{Allowed: false, ToolPack: pack, Reason: "tool is required"}
	}
	if req.Tool != ToolPostgresSelect {
		return Decision{Allowed: false, ToolPack: pack, Reason: "tool not permitted"}
	}
	return Decision{Allowed: true, ToolPack: pack, Reason: "group mapped to tool pack"}
}

package agent

// Registry holds all registered agent adapters.
type Registry struct {
	adapters []Adapter
}

func NewRegistry(adapters ...Adapter) *Registry {
	return &Registry{adapters: adapters}
}

// All returns every registered adapter.
func (r *Registry) All() []Adapter {
	return r.adapters
}

// Detected returns only adapters whose agent is installed on the system.
func (r *Registry) Detected() []Adapter {
	var out []Adapter
	for _, a := range r.adapters {
		if a.Detect() {
			out = append(out, a)
		}
	}
	return out
}

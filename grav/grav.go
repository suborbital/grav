package grav

// Grav represents a Grav message bus instance
type Grav struct {
	GroupName string

	bus *bus
}

// New creates a new Grav instance
func New() *Grav {
	b := &bus{
		pool: &connectionPool{},
	}

	g := &Grav{
		GroupName: "",
		bus:       b,
	}

	return g
}

type bus struct {
	pool *connectionPool
}

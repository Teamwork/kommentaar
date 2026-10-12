package path

type parent struct {
	Name string `json:"name"` // {required}

	// {schema: child.yaml}
	Child child `json:"child"`

	// {required} {schema: child.yaml}
	TaggedChild *child `json:"taggedChild"`
}

type child struct{}

// POST /path
//
// Request body: parent
// Response 200: parent

package path

type parent struct {
	Name string `json:"name"` // {required}

	// {schema: child.yaml}
	Child child `json:"child"`

	// {required} {schema: child.yaml}
	TaggedChild *child `json:"taggedChild"`

	// {schema: id.yaml} {required}
	ID *child `json:"id"`

	// {schema: child.json}
	JSONChild child `json:"jsonChild"`
}

type child struct{}

// POST /path
//
// Request body: parent
// Response 200: parent

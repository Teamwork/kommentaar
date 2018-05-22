// nolint
package example

// Single entity response.
type entityResponse struct {
	// The requested entity.
	Entity entity `json:"entity"`

	// The requested sideloaded data.
	Include map[string]interface{} `json:"include"`
}

// some entity
type entity struct {
	// ID of the entity.
	ID int64 `json:"id"`
	// Name of the entity. {required}
	Name string `json:"name"`
}

// QueryParams documentation.
type QueryParams struct {
	PageOffset int `json:"pageOffset"`
	PageSize   int `json:"pageSize"`
	Page       int `json:"page"`

	// Comma separated list of targets to order by.
	OrderBy string `json:"orderBy"`
	// Direction to order by ascending or descending.
	OrderMode string `json:"orderMode"` // {enum: asc desc}

	// Comma separated list of targets to include.
	Include string `json:"include"`
}

// GET /entities.json
// List of entities paginated.
//
// Query: $ref: QueryParams
// Response 400: $default
func ListEntities() {}

// GET /entities/{id}.json
// Get entity by id
//
// Query: $ref: QueryParams
// Response 200: $ref: entityResponse
// Response 400: $default
func GetEntity() {}

// POST /entities.json
// Create an entity
//
// Request body: $ref: entity
// Response 200: $ref: entityResponse
// Response 400: $default
func PostEntity() {}

// PATCH /entities/{id}.json
// Update an entity
//
// Request body: $ref: entity
// Response 200: $ref: entityResponse
// Response 400: $default
func PatchEntity() {}

// overriding the docs for id
type deletePathParams struct {
	ID int64 `json:"id" path:"id"` // The id to delete
}

// DELETE /entities/{id}.json
// Delete an entity
//
// Path: $ref: deletePathParams
// Response 204: $empty
// Response 400: $default
func DeleteEntity() {}

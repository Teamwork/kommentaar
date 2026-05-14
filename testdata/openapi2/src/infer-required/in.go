package infer_required

// Currency exercises every inference branch in one struct.
type Currency struct {
	// ID is required by default (non-pointer, no omitempty).
	ID int64 `json:"id"`
	// Name is required by default.
	Name string `json:"name"`
	// Code is required by default.
	Code string `json:"code"`
	// Symbol is a pointer, so it is not inferred as required.
	Symbol *string `json:"symbol"`
	// DecimalPoints is a pointer.
	DecimalPoints *int64 `json:"decimalPoints"`
	// Description has omitempty, so it is not required.
	Description string `json:"description,omitempty"`
	// Notes is explicitly marked optional via doc tag. {optional}
	Notes string `json:"notes"`
	// LegacyRequired exercises the explicit doc tag on a pointer field. {required}
	LegacyRequired *string `json:"legacyRequired"`
}

// CurrencyResponse wraps a currency.
type CurrencyResponse struct {
	Currency Currency `json:"currency"`
}

// CurrencyQuery is a query-parameter struct; required inference must not
// apply here (path/query/form params have their own required handling).
type CurrencyQuery struct {
	Code string `json:"code"`
}

// GET /currencies/{id}.json get a currency
//
// Query: CurrencyQuery
// Response 200 (application/json): CurrencyResponse

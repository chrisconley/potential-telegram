package specs

// MeasurementSpec represents a measured quantity with its unit.
//
// Measurements are extracted from event properties during the metering process
// and represent the quantifiable aspect of usage that will be aggregated and
// rated for billing. The quantity is stored as a decimal string to preserve
// precision across language implementations.
type MeasurementSpec struct {
	// Numeric value as a decimal string.
	//
	// Stored as string to preserve arbitrary precision across language boundaries
	// and avoid floating-point representation issues. Must be parseable as a decimal
	// number. Examples: "42", "123.456", "0.001", "1000000.00".
	Quantity string `json:"quantity"`

	// Unit identifier for this measurement.
	//
	// Defines what is being measured and determines how values aggregate and get
	// rated. Units should be descriptive and match your billing model.
	// Examples: "api-calls", "tokens", "gb-hours", "seats", "requests",
	// "compute-minutes". The same unit across different meter records can be
	// aggregated together.
	Unit string `json:"unit"`
}

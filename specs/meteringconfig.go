package specs

// MeteringConfigSpec defines how to transform EventPayload properties into MeterRecords.
//
// One EventPayload can produce multiple MeterRecords (one per measurement extraction).
// All properties not extracted as measurements are passed through as dimensions.
type MeteringConfigSpec struct {
	// List of measurements to extract from the event payload.
	//
	// Each extraction defines which property to extract, what unit to assign,
	// and optionally a filter condition. A single event can produce multiple
	// meter records if multiple extractions are configured. For example, an
	// LLM completion event might extract both "input_tokens" and "output_tokens"
	// as separate measurements with the "tokens" unit.
	Measurements []MeasurementExtractionSpec `json:"measurements"`
}

// MeasurementExtractionSpec defines how to extract a measurement from EventPayload.
//
// Specifies which property contains the numeric value, what unit to assign to it,
// and optionally a filter to conditionally extract the measurement only when certain
// criteria are met.
type MeasurementExtractionSpec struct {
	// The property key in EventPayload.Properties to extract as a measurement.
	//
	// Must exist in the event's properties map and contain a value parseable as
	// a decimal number. Examples: "response_time_ms", "tokens", "bytes_transferred".
	SourceProperty string `json:"sourceProperty"`

	// Unit identifier to assign to the extracted measurement.
	//
	// Determines how this measurement aggregates with others and how it gets rated
	// for billing. Should match your rate card definitions. Examples: "api-calls",
	// "tokens", "gb-hours", "seats".
	Unit string `json:"unit"`

	// Optional filter condition to apply before extracting the measurement.
	//
	// If specified, the measurement is only extracted when the filter matches.
	// This enables conditional metering, such as extracting different units based
	// on dimension values. For example, only extract "premium-requests" when
	// the "tier" property equals "premium". If nil, the measurement is always
	// extracted.
	Filter *FilterSpec `json:"filter,omitempty"`
}

// FilterSpec defines a filter condition on EventPayload properties.
//
// Currently supports only simple equality matching. More complex filter operations
// (inequality, regex, existence checks) can be added as needed.
type FilterSpec struct {
	// The property key in EventPayload.Properties to check.
	//
	// Examples: "region", "tier", "status_code", "model".
	Property string `json:"property"`

	// The exact value the property must equal for the filter to match.
	//
	// Comparison is case-sensitive string equality. Examples: "premium",
	// "us-east-1", "200", "gpt-4".
	Equals string `json:"equals"`
}

// ObservationExtractionSpec defines how to extract an observation from EventPayload.
//
// Specifies which property contains the numeric value, what unit to assign to it,
// and optionally a filter to conditionally extract the observation only when certain
// criteria are met.
//
// Note: This is the new naming aligned with domain terminology. Use this instead of
// MeasurementExtractionSpec for new code.
type ObservationExtractionSpec struct {
	// The property key in EventPayload.Properties to extract as an observation.
	//
	// Must exist in the event's properties map and contain a value parseable as
	// a decimal number. Examples: "response_time_ms", "tokens", "bytes_transferred".
	SourceProperty string `json:"sourceProperty"`

	// Unit identifier to assign to the extracted observation.
	//
	// Determines how this observation aggregates with others and how it gets rated
	// for billing. Should match your rate card definitions. Examples: "api-calls",
	// "tokens", "gb-hours", "seats".
	Unit string `json:"unit"`

	// Optional filter condition to apply before extracting the observation.
	//
	// If specified, the observation is only extracted when the filter matches.
	// This enables conditional metering, such as extracting different units based
	// on dimension values. For example, only extract "premium-requests" when
	// the "tier" property equals "premium". If nil, the observation is always
	// extracted.
	Filter *FilterSpec `json:"filter,omitempty"`
}

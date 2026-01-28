package specs

import (
	"fmt"
	"time"
)

// ObservationSpec represents a point-in-time or time-spanning observation from events.
//
// Observations are raw measurements extracted from event payloads during metering.
// They capture both the measured quantity and its temporal extent. The quantity is
// stored as a decimal string to preserve precision across language implementations.
//
// Temporal Context:
//   - Instant observations: Window.Start == Window.End (observed at a single moment)
//   - Time-spanning observations: Window.Start < Window.End (observed over a period)
//
// Examples:
//   - Instant gauge: "15 seats at 9:47am" → Window: [9:47am, 9:47am]
//   - Time-spanning: "8 compute-hours from 8pm to 4am" → Window: [8pm, 4am]
type ObservationSpec struct {
	// Numeric value as a decimal string.
	//
	// Stored as string to preserve arbitrary precision across language boundaries
	// and avoid floating-point representation issues. Must be parseable as a decimal
	// number. Examples: "42", "123.456", "0.001", "1000000.00".
	Quantity string `json:"quantity"`

	// Unit identifier for this observation.
	//
	// Defines what is being measured. Units should be descriptive and match your
	// metering model. Examples: "seats", "tokens", "compute-hours", "api-calls",
	// "gb-hours". Observations with the same unit can be aggregated together.
	Unit string `json:"unit"`

	// Temporal extent of this observation.
	//
	// For instant observations (gauges, discrete events): Start == End
	// For time-spanning observations (durations): Start < End
	//
	// The window captures when the observation occurred, enabling downstream use
	// cases like proration across billing periods and time-weighted aggregation.
	Window TimeWindowSpec `json:"window"`
}

// AggregateSpec represents a computed aggregation result.
//
// Aggregates are produced by applying aggregation strategies (sum, max,
// time-weighted-avg, etc.) to a collection of observations. Unlike observations,
// aggregates do not include a Window field—temporal context is provided
// by the parent MeterReading.Window instead.
//
// The quantity is stored as a decimal string to preserve precision across
// language implementations.
type AggregateSpec struct {
	// Numeric value as a decimal string.
	//
	// Result of aggregating multiple observations. Stored as string to preserve
	// arbitrary precision. Examples: "1250.50", "99.95", "10000".
	Quantity string `json:"quantity"`

	// Unit identifier matching the source observations.
	//
	// All observations aggregated into this value must share the same unit.
	// Examples: "seats", "tokens", "compute-hours".
	Unit string `json:"unit"`
}

// NewInstantObservation creates an observation at a single point in time.
//
// The resulting observation has Window.Start == Window.End, representing
// an instant measurement (gauge reading, discrete event, etc.).
//
// Examples:
//   - Gauge reading: "15 seats at 9:47am"
//   - API call: "500 tokens at 10:30:15"
func NewInstantObservation(quantity, unit string, instant time.Time) ObservationSpec {
	return ObservationSpec{
		Quantity: quantity,
		Unit:     unit,
		Window: TimeWindowSpec{
			Start: instant,
			End:   instant,
		},
	}
}

// NewSpanObservation creates an observation over a time window.
//
// The resulting observation has Window.Start < Window.End, representing
// a time-spanning measurement (compute duration, active period, etc.).
//
// Returns error if end is not after start.
//
// Examples:
//   - Compute duration: "8 compute-hours from Jan 31 8pm to Feb 1 4am"
//   - Active session: "1 active-user from 9:00am to 9:45am"
func NewSpanObservation(quantity, unit string, start, end time.Time) (ObservationSpec, error) {
	if !end.After(start) {
		return ObservationSpec{}, fmt.Errorf("span observation: end must be after start (start=%v, end=%v)", start, end)
	}
	return ObservationSpec{
		Quantity: quantity,
		Unit:     unit,
		Window: TimeWindowSpec{
			Start: start,
			End:   end,
		},
	}, nil
}

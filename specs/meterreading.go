package specs

import "time"

// TimeWindowSpec represents a half-open time interval [Start, End).
//
// Used to define billing periods and aggregation windows. The start time is
// inclusive and the end time is exclusive, following standard interval notation.
// This ensures adjacent windows don't overlap or have gaps.
type TimeWindowSpec struct {
	// Inclusive start of the time window.
	//
	// Meter records with RecordedAt >= Start are included in this window.
	// Should be in UTC to avoid timezone ambiguity.
	Start time.Time `json:"start"`

	// Exclusive end of the time window.
	//
	// Meter records with RecordedAt < End are included in this window.
	// Should be in UTC to avoid timezone ambiguity. For example, a monthly
	// window might be [2024-01-01T00:00:00Z, 2024-02-01T00:00:00Z).
	End time.Time `json:"end"`
}

// MeterReadingSpec represents an aggregated usage value over a time window.
//
// Meter readings are created by aggregating meter records that share the same
// subject and unit within a specific time window. They represent the billable
// usage for a subject during a billing period.
//
// The aggregation strategy (sum, max, time-weighted-average, etc.) determines
// how individual meter records combine into the final reading.
type MeterReadingSpec struct {
	// Unique identifier for this meter reading.
	//
	// Deterministically generated from the subject, unit, time window, and
	// aggregation type, ensuring idempotent aggregation. Re-aggregating the
	// same records produces the same reading ID.
	ID string `json:"id"`

	// Workspace that owns the meter records being aggregated.
	//
	// All meter records in an aggregation must share the same workspace.
	WorkspaceID string `json:"workspaceID"`

	// Universe the meter records belong to.
	//
	// All meter records in an aggregation must share the same universe.
	// Subject identity is scoped to this universe.
	UniverseID string `json:"universeID"`

	// The entity this aggregated usage is attributed to for billing.
	//
	// All meter records in this reading have the same subject. Format follows
	// the "type:id" convention (e.g., "customer:cust_123"). This is the billing
	// entity for the aggregated usage.
	Subject string `json:"subject"`

	// Time window over which meter records were aggregated.
	//
	// Defines the half-open interval [Window.Start, Window.End) for this reading.
	// Typically corresponds to a billing period (hour, day, month). Meter records
	// with RecordedAt within this window contribute to the aggregation.
	Window TimeWindowSpec `json:"window"`

	// The aggregated measurement result.
	//
	// Contains the computed quantity (as a decimal string) and the unit from the
	// source meter records. All meter records aggregated into this reading must
	// share the same unit. The quantity is the result of applying the aggregation
	// strategy to the individual record quantities.
	Measurement MeasurementSpec `json:"measurement"`

	// Aggregation strategy applied to compute the measurement.
	//
	// Determines how individual meter record quantities combine:
	//   - "sum": Add all quantities (e.g., total API calls, total tokens)
	//   - "max": Maximum quantity in window (e.g., peak concurrent users)
	//   - "min": Minimum quantity in window
	//   - "latest": Most recent quantity by RecordedAt
	//   - "time-weighted-avg": Average weighted by time between records (e.g., seat count)
	Aggregation string `json:"aggregation"`

	// Number of meter records aggregated to produce this reading.
	//
	// Indicates how many individual meter records contributed to the measurement.
	// Useful for debugging and understanding aggregation granularity. A count of
	// zero is possible for time-weighted-avg when carried forward from a previous
	// period.
	RecordCount int `json:"recordCount"`

	// System timestamp when this reading was created.
	//
	// Represents when the aggregation computation occurred, not the business time
	// of the usage. Used for audit trails and versioning.
	CreatedAt time.Time `json:"createdAt"`

	// Latest MeteredAt timestamp from all aggregated meter records.
	//
	// Tracks the most recent metering processing time among the source records.
	// Used for watermarking in incremental aggregation pipelines to determine
	// which records have been processed. Enables exactly-once aggregation semantics.
	MaxMeteredAt time.Time `json:"maxMeteredAt"`
}

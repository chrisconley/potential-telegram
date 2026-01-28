package specs

import "time"

// MeterRecordSpec represents a single metered usage record.
//
// A meter record is created by applying metering configuration to an event payload.
// It extracts observations (quantity + unit + temporal context) and preserves
// dimensional attributes for downstream aggregation and billing.
//
// One event payload produces one meter record containing all observations extracted
// by the metering configuration. Multiple observations from the same event are bundled
// together in the Observations array, ensuring atomic persistence and preserving the
// relationship between related measurements.
type MeterRecordSpec struct {
	// Unique identifier for this meter record.
	//
	// Deterministically generated from the source event ID, ensuring idempotent
	// metering. Replaying the same event with the same metering configuration
	// produces the same record ID.
	//
	// NOTE: During migration from singular Observation to Observations array,
	// ID generation changed from including the measurement unit to using just
	// the event ID. This ensures one event produces one record with one ID.
	ID string `json:"id"`

	// Identifier for the workspace that owns this record.
	//
	// Workspaces represent operational boundaries (regions, divisions, business units)
	// and own their event schemas and metering configurations. The same subject can
	// have usage across multiple workspaces.
	WorkspaceID string `json:"workspaceID"`

	// Identifier for the universe this record belongs to.
	//
	// Universes represent data namespaces (production, test, simulation) and scope
	// subject identity. The same subject string in different universes represents
	// different entities for billing purposes. For example, subject "customer:123"
	// in the "production" universe is distinct from "customer:123" in "test".
	UniverseID string `json:"universeID"`

	// The entity this usage is attributed to for billing purposes.
	//
	// Format convention: "type:id" where type can be customer, organization, team,
	// cohort, or any attribution entity. Examples: "customer:cust_123", "org:acme",
	// "team:engineering". Subject identity is scoped to the universe.
	Subject string `json:"subject"`

	// Business timestamp indicating when the usage was observed.
	//
	// This is the event time from the original event payload, not when the record
	// was processed. For instant observations, this matches Observation.Window.Start
	// (and Window.End). Used for time-based aggregations and billing period assignment.
	// Distinct from MeteredAt which tracks system processing time.
	ObservedAt time.Time `json:"observedAt"`

	// Multiple observations from the same event (preferred).
	//
	// Contains all observations extracted from a single event, ensuring atomic
	// persistence. When one event produces multiple measurements (e.g., LLM API call
	// with input_tokens and output_tokens), all observations are bundled in this array.
	// This provides natural atomicity: saving the record persists all observations or none.
	//
	// Preferred over the singular Observation field. During migration, both fields
	// may be populated for backwards compatibility.
	Observations []ObservationSpec `json:"observations,omitempty"`

	// Single observation (deprecated, for backwards compatibility).
	//
	// DEPRECATED: Use Observations array instead. This field is maintained during
	// migration for backwards compatibility. When Observations is populated, this
	// field typically contains the first observation from the array.
	//
	// Will be removed in a future version once all consumers migrate to Observations.
	Observation ObservationSpec `json:"observation,omitempty"`

	// Additional categorical attributes from the source event.
	//
	// Contains all event properties that were not extracted as measurements,
	// providing context for filtering and segmentation during aggregation.
	// Common examples: region, model, status_code, feature_flag.
	Dimensions map[string]string `json:"dimensions,omitempty"`

	// Identifier of the source event that produced this record.
	//
	// Links back to the original event payload for audit trails and debugging.
	// In the current design, one event produces one record, so this typically
	// has a one-to-one relationship with the record ID.
	SourceEventID string `json:"sourceEventID"`

	// System timestamp indicating when this record was created by the metering process.
	//
	// Used for incremental processing and watermarking in streaming systems.
	// Records can be queried by "give me all records metered since timestamp X"
	// to support exactly-once processing semantics. Distinct from RecordedAt
	// which represents business time.
	MeteredAt time.Time `json:"meteredAt"`
}

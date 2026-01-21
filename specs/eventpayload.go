package specs

import "time"

// EventPayloadSpec represents a raw usage event submitted for metering.
//
// Event payloads are the input boundary of the metering system. They capture
// usage activity with maximum flexibility through an untyped properties map,
// allowing each workspace to define its own event schemas without coordinating
// type changes across systems.
//
// The metering process transforms event payloads into typed meter records by
// applying workspace-specific metering configurations that extract measurements
// from properties and assign units.
type EventPayloadSpec struct {
	// Unique identifier for this event.
	//
	// Used as an idempotency key to ensure the same event is not metered multiple
	// times. Also used to generate deterministic meter record IDs. Should be unique
	// per event within a workspace and universe. Common approaches: UUID, timestamp-based
	// ID, or external system transaction ID.
	ID string `json:"id"`

	// Identifier for the workspace that owns this event's schema.
	//
	// Workspaces represent operational boundaries (regions, divisions, business units)
	// and own their event type definitions and metering configurations. The same event
	// type in different workspaces can have completely different schemas and be metered
	// differently. For example, "api.request" in a US workspace may have different
	// properties than "api.request" in an EU workspace to accommodate regulatory
	// requirements.
	WorkspaceID string `json:"workspaceID"`

	// Identifier for the universe this event belongs to.
	//
	// Universes represent data namespaces (production, test, simulation, legacy systems)
	// and scope subject identity. The same subject string in different universes represents
	// different entities. This enables safe testing with production-like data, what-if
	// scenario analysis, and post-merger system integration without ID collisions.
	// Examples: "production", "test", "staging", "simulation-q4-pricing".
	UniverseID string `json:"universeID"`

	// Classification of this event within the workspace's schema.
	//
	// Event types determine which metering configuration applies and what measurements
	// can be extracted from properties. Common formats: dot-notation namespacing
	// (e.g., "api.request", "llm.completion.streaming") or action-based naming
	// (e.g., "user.signup", "file.uploaded"). Must be defined in the workspace's
	// event catalog.
	Type string `json:"type"`

	// The entity this event's usage should be attributed to for billing purposes.
	//
	// Format convention: "type:id" where type identifies the attribution model.
	// Examples: "customer:cust_123" (individual customer), "org:acme" (organization-wide),
	// "team:engineering" (department allocation), "cohort:enterprise" (customer segment).
	// Subject identity is scoped to the universe - the same subject string in different
	// universes represents distinct billing entities.
	Subject string `json:"subject"`

	// Business timestamp indicating when this event occurred.
	//
	// Represents the actual time of the usage activity, not when the event was submitted
	// or processed. Used for time-based aggregations, billing period assignment, and
	// time-series analysis. Should be in UTC to avoid timezone ambiguity.
	Time time.Time `json:"time"`

	// Untyped event properties as string key-value pairs.
	//
	// Contains all event-specific data in a maximally flexible format. The metering
	// configuration defines which properties to extract as measurements (numeric values)
	// and which to pass through as dimensions (categorical attributes). This design
	// allows each workspace to evolve its event schemas independently without requiring
	// type system changes.
	//
	// Examples:
	//   - API usage: {"endpoint": "/api/v1/users", "status_code": "200", "response_time_ms": "125"}
	//   - LLM completion: {"model": "gpt-4", "input_tokens": "450", "output_tokens": "890", "cached": "true"}
	//   - Storage: {"bucket": "prod-assets", "bytes_stored": "1073741824", "region": "us-east-1"}
	Properties map[string]string `json:"properties,omitempty"`
}

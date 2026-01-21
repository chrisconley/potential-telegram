package internal

import (
	"testing"
)

func TestFlow(t *testing.T) {

	// EventPayload
	// Filter out irrelevant events
	// Transform string properties into to typed Measures+Dimensions
	// Extract one or more Measurements from Measures
	// Aggregate Measurements into a single Measurement

	// EventPayload (untyped properties)
	// Lookup schema (by WorkspaceID+EventType) - filters out events without schema
	// Transform string properties into typed Measures+Dimensions
	// Validate homogeneity (all events same WorkspaceID+Universe+EventType)
	// For each MeteringConfig:
	//   - Filter by dimensions (optional - config.Filter.Matches(dimensions))
	//   - Extract source measure (config.SourceMeasure)
	//   - Assign unit (config.Unit - may differ from source measure)
	//   → produces MeterRecord
	// Group MeterRecords by unit
	// Aggregate each group into single Measurement (sum/max/time-weighted-avg/latest/min)
}

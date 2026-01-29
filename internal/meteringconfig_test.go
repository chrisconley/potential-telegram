package internal

import (
	"metering-spec/specs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers

type eventPayloadOption func(*specs.EventPayloadSpec)

func withPayloadProperties(properties map[string]string) eventPayloadOption {
	return func(s *specs.EventPayloadSpec) { s.Properties = properties }
}

func withPayloadID(id string) eventPayloadOption {
	return func(s *specs.EventPayloadSpec) { s.ID = id }
}

func withPayloadSubject(subject string) eventPayloadOption {
	return func(s *specs.EventPayloadSpec) { s.Subject = subject }
}

// newTestEventPayload creates an EventPayload with the given options.
// ID defaults to "test-event" if not specified.
// WorkspaceID defaults to "workspace-test" if not specified.
// UniverseID defaults to "universe-test" if not specified.
// Type defaults to "test.event" if not specified.
// Subject defaults to "customer:test" if not specified.
// Time defaults to time.Now() if not specified.
// Properties defaults to empty map if not specified.
func newTestEventPayload(opts ...eventPayloadOption) (EventPayload, error) {
	spec := specs.EventPayloadSpec{
		ID:          "test-event",
		WorkspaceID: "workspace-test",
		UniverseID:  "universe-test",
		Type:        "test.event",
		Subject:     "customer:test",
		Time:        time.Now(),
		Properties:  make(map[string]string),
	}

	for _, opt := range opts {
		opt(&spec)
	}

	return NewEventPayload(spec)
}

func TestMeter(t *testing.T) {
	t.Run("meters single property into meter record", func(t *testing.T) {
		// Arrange: Create specs using real API
		payloadSpec := specs.EventPayloadSpec{
			ID:          "event-123",
			WorkspaceID: "workspace-prod",
			UniverseID:  "production",
			Type:        "api.completion",
			Subject:     "customer:acme",
			Time:        time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
			Properties: map[string]string{
				"tokens": "1250",
				"model":  "gpt-4",
				"region": "us-east-1",
			},
		}

		configSpec := specs.MeteringConfigSpec{
			Observations: []specs.ObservationExtractionSpec{
				{
					SourceProperty: "tokens",
					Unit:           "api-tokens",
				},
			},
		}

		// Act: Transform EventPayloadSpec → MeterRecordSpecs
		recordSpecs, err := Meter(payloadSpec, configSpec)

		// Assert: Verify transformation
		require.NoError(t, err)
		require.Len(t, recordSpecs, 1)

		record := recordSpecs[0]
		require.Len(t, record.Observations, 1)
		assert.Equal(t, "1250", record.Observations[0].Quantity)
		assert.Equal(t, "api-tokens", record.Observations[0].Unit)
		assert.Equal(t, "customer:acme", record.Subject)
		assert.Equal(t, "event-123", record.SourceEventID)

		// Verify dimensions: extracted property should not be included
		_, hasTokens := record.Dimensions["tokens"]
		assert.False(t, hasTokens, "extracted property should not be in dimensions")
		assert.Equal(t, "gpt-4", record.Dimensions["model"], "non-extracted properties should be in dimensions")
		assert.Equal(t, "us-east-1", record.Dimensions["region"])
	})

	t.Run("newTestEventPayload creates valid EventPayload by default", func(t *testing.T) {
		// Validate helper works with defaults
		_, err := newTestEventPayload()
		require.NoError(t, err)
	})

	t.Run("with spec interface transforms correctly", func(t *testing.T) {
		// Arrange: Use spec-level interface (primitives only)
		payloadSpec := specs.EventPayloadSpec{
			ID:          "event-spec",
			WorkspaceID: "workspace-test",
			UniverseID:  "universe-test",
			Type:        "test.event",
			Subject:     "customer:test",
			Time:        time.Now(),
			Properties: map[string]string{
				"tokens": "500",
				"model":  "gpt-4",
			},
		}

		configSpec := specs.MeteringConfigSpec{
			Observations: []specs.ObservationExtractionSpec{
				{SourceProperty: "tokens", Unit: "test-tokens"},
			},
		}

		// Act: Call spec-level function
		recordSpecs, err := Meter(payloadSpec, configSpec)

		// Assert: Verify spec-level results
		require.NoError(t, err)
		require.Len(t, recordSpecs, 1)
		require.Len(t, recordSpecs[0].Observations, 1)
		assert.Equal(t, "500", recordSpecs[0].Observations[0].Quantity)
		assert.Equal(t, "test-tokens", recordSpecs[0].Observations[0].Unit)
		assert.Equal(t, "customer:test", recordSpecs[0].Subject)
		assert.Equal(t, "event-spec", recordSpecs[0].SourceEventID)
		assert.Equal(t, "gpt-4", recordSpecs[0].Dimensions["model"])
	})

	t.Run("extracts multiple measurements from single payload", func(t *testing.T) {
		// Arrange: Spec with multiple measurable properties
		payloadSpec := specs.EventPayloadSpec{
			ID:          "event-multi",
			WorkspaceID: "workspace-test",
			UniverseID:  "universe-test",
			Type:        "test.event",
			Subject:     "customer:test",
			Time:        time.Now(),
			Properties: map[string]string{
				"input_tokens":  "1250",
				"output_tokens": "340",
				"model":         "gpt-4",
			},
		}

		configSpec := specs.MeteringConfigSpec{
			Observations: []specs.ObservationExtractionSpec{
				{SourceProperty: "input_tokens", Unit: "input-tokens"},
				{SourceProperty: "output_tokens", Unit: "output-tokens"},
			},
		}

		// Act
		recordSpecs, err := Meter(payloadSpec, configSpec)

		// Assert
		require.NoError(t, err)
		require.Len(t, recordSpecs, 1, "should create one record with bundled observations")

		record := recordSpecs[0]

		// Verify bundled observations (new field)
		require.Len(t, record.Observations, 2, "should have two observations")
		assert.Equal(t, "1250", record.Observations[0].Quantity)
		assert.Equal(t, "input-tokens", record.Observations[0].Unit)
		assert.Equal(t, "340", record.Observations[1].Quantity)
		assert.Equal(t, "output-tokens", record.Observations[1].Unit)

		// Verify dimensions: all extracted properties excluded, only model remains
		assert.Equal(t, "gpt-4", record.Dimensions["model"], "should have non-extracted dimension")
		_, hasInputTokens := record.Dimensions["input_tokens"]
		assert.False(t, hasInputTokens, "should not have extracted dimension")
		_, hasOutputTokens := record.Dimensions["output_tokens"]
		assert.False(t, hasOutputTokens, "should not have extracted dimension")
	})
}

// Tests for new ObservationExtraction types (parallel to MeasurementExtraction)
func TestNewObservationExtraction(t *testing.T) {
	t.Run("creates valid observation extraction from spec", func(t *testing.T) {
		spec := specs.ObservationExtractionSpec{
			SourceProperty: "tokens",
			Unit:           "api-tokens",
		}

		extraction, err := NewObservationExtraction(spec)

		require.NoError(t, err)
		assert.Equal(t, "tokens", extraction.SourceProperty().ToString())
		assert.Equal(t, "api-tokens", extraction.Unit().ToString())
		assert.Nil(t, extraction.Filter())
	})

	t.Run("creates observation extraction with filter", func(t *testing.T) {
		spec := specs.ObservationExtractionSpec{
			SourceProperty: "tokens",
			Unit:           "premium-tokens",
			Filter: &specs.FilterSpec{
				Property: "tier",
				Equals:   "premium",
			},
		}

		extraction, err := NewObservationExtraction(spec)

		require.NoError(t, err)
		assert.Equal(t, "tokens", extraction.SourceProperty().ToString())
		assert.Equal(t, "premium-tokens", extraction.Unit().ToString())
		assert.NotNil(t, extraction.Filter())
		assert.Equal(t, "tier", extraction.Filter().Property().ToString())
		assert.Equal(t, "premium", extraction.Filter().Equals().ToString())
	})

	t.Run("rejects empty source property", func(t *testing.T) {
		spec := specs.ObservationExtractionSpec{
			SourceProperty: "",
			Unit:           "tokens",
		}

		_, err := NewObservationExtraction(spec)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "source property")
	})

	t.Run("rejects empty unit", func(t *testing.T) {
		spec := specs.ObservationExtractionSpec{
			SourceProperty: "tokens",
			Unit:           "",
		}

		_, err := NewObservationExtraction(spec)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unit")
	})

	t.Run("rejects invalid filter", func(t *testing.T) {
		spec := specs.ObservationExtractionSpec{
			SourceProperty: "tokens",
			Unit:           "api-tokens",
			Filter: &specs.FilterSpec{
				Property: "", // Invalid
				Equals:   "premium",
			},
		}

		_, err := NewObservationExtraction(spec)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "filter")
	})
}

func TestObservationExtraction_Matches(t *testing.T) {
	t.Run("matches when no filter", func(t *testing.T) {
		extraction, err := NewObservationExtraction(specs.ObservationExtractionSpec{
			SourceProperty: "tokens",
			Unit:           "api-tokens",
		})
		require.NoError(t, err)

		payload, err := newTestEventPayload(withPayloadProperties(map[string]string{
			"tokens": "1000",
			"tier":   "basic",
		}))
		require.NoError(t, err)

		assert.True(t, extraction.Matches(payload.Properties))
	})

	t.Run("matches when filter condition is met", func(t *testing.T) {
		extraction, err := NewObservationExtraction(specs.ObservationExtractionSpec{
			SourceProperty: "tokens",
			Unit:           "premium-tokens",
			Filter: &specs.FilterSpec{
				Property: "tier",
				Equals:   "premium",
			},
		})
		require.NoError(t, err)

		payload, err := newTestEventPayload(withPayloadProperties(map[string]string{
			"tokens": "1000",
			"tier":   "premium",
		}))
		require.NoError(t, err)

		assert.True(t, extraction.Matches(payload.Properties))
	})

	t.Run("does not match when filter condition is not met", func(t *testing.T) {
		extraction, err := NewObservationExtraction(specs.ObservationExtractionSpec{
			SourceProperty: "tokens",
			Unit:           "premium-tokens",
			Filter: &specs.FilterSpec{
				Property: "tier",
				Equals:   "premium",
			},
		})
		require.NoError(t, err)

		payload, err := newTestEventPayload(withPayloadProperties(map[string]string{
			"tokens": "1000",
			"tier":   "basic",
		}))
		require.NoError(t, err)

		assert.False(t, extraction.Matches(payload.Properties))
	})

	t.Run("does not match when filter property is missing", func(t *testing.T) {
		extraction, err := NewObservationExtraction(specs.ObservationExtractionSpec{
			SourceProperty: "tokens",
			Unit:           "premium-tokens",
			Filter: &specs.FilterSpec{
				Property: "tier",
				Equals:   "premium",
			},
		})
		require.NoError(t, err)

		payload, err := newTestEventPayload(withPayloadProperties(map[string]string{
			"tokens": "1000",
			// tier property missing
		}))
		require.NoError(t, err)

		assert.False(t, extraction.Matches(payload.Properties))
	})
}

func TestNewObservationSourceProperty(t *testing.T) {
	t.Run("creates valid source property", func(t *testing.T) {
		prop, err := NewObservationSourceProperty("tokens")

		require.NoError(t, err)
		assert.Equal(t, "tokens", prop.ToString())
	})

	t.Run("rejects empty value", func(t *testing.T) {
		_, err := NewObservationSourceProperty("")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})
}

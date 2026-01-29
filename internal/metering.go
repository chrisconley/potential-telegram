package internal

import (
	"fmt"
	specs "metering-spec/specs"
)

// Meter implements specs.Meter.
// Converts specs to domain objects, transforms, and converts back to specs.
func Meter(payloadSpec specs.EventPayloadSpec, configSpec specs.MeteringConfigSpec) ([]specs.MeterRecordSpec, error) {
	// Convert specs to domain objects
	payload, err := NewEventPayload(payloadSpec)
	if err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	config, err := NewMeteringConfig(configSpec)
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Transform using domain objects
	records, err := meter(payload, config)
	if err != nil {
		return nil, err
	}

	// Convert domain objects back to specs
	// NOTE: During migration, we return one record per event with bundled observations
	// For backwards compatibility, we still support the old format where meter()
	// returns multiple records (one per measurement)
	if len(records) == 0 {
		return []specs.MeterRecordSpec{}, nil
	}

	// Bundle observations from the same source event
	// Group records by SourceEventID
	recordsByEvent := make(map[string][]MeterRecord)
	for _, record := range records {
		eventID := record.SourceEventID.ToString()
		recordsByEvent[eventID] = append(recordsByEvent[eventID], record)
	}

	// Create one MeterRecordSpec per event with bundled observations
	recordSpecs := make([]specs.MeterRecordSpec, 0, len(recordsByEvent))
	for _, eventRecords := range recordsByEvent {
		// Use first record for common fields
		firstRecord := eventRecords[0]
		observedAt := firstRecord.ObservedAt.ToTime()

		// Bundle all observations from eventRecords
		observations := make([]specs.ObservationSpec, len(eventRecords))
		for i, record := range eventRecords {
			// Each record already has Observations[0] from meter()
			// Copy the ObservationSpec directly
			observations[i] = specs.ObservationSpec{
				Quantity: record.Observations[0].Quantity().String(),
				Unit:     record.Observations[0].Unit().ToString(),
				Window:   record.Observations[0].Window().ToSpec(),
			}
		}

		recordSpec := specs.MeterRecordSpec{
			ID:            firstRecord.SourceEventID.ToString(), // Just event ID, no unit suffix
			WorkspaceID:   firstRecord.WorkspaceID.ToString(),
			UniverseID:    firstRecord.UniverseID.ToString(),
			Subject:       firstRecord.Subject.ToString(),
			ObservedAt:    observedAt,
			Observations:  observations,
			Dimensions:    convertDimensionsToMap(firstRecord.Dimensions),
			SourceEventID: firstRecord.SourceEventID.ToString(),
			MeteredAt:     firstRecord.MeteredAt.ToTime(),
		}

		recordSpecs = append(recordSpecs, recordSpec)
	}

	return recordSpecs, nil
}

// convertDimensionsToMap converts MeterRecordDimensions to map[string]string
func convertDimensionsToMap(dimensions MeterRecordDimensions) map[string]string {
	result := make(map[string]string)
	for _, name := range dimensions.Names() {
		if value, ok := dimensions.Get(name); ok {
			result[name] = value
		}
	}
	return result
}

// meter transforms an EventPayload into MeterRecords by applying the metering configuration.
// This is the private domain-level function that operates on domain objects.
//
// For each measurement extraction in the config:
//  1. Check if filter matches (if filter exists)
//  2. Extract the source property value
//  3. Cast to Decimal
//  4. Attach the configured unit
//  5. Pass through all non-extracted properties as dimensions
//  6. Create a MeterRecord
//
// Returns a slice of MeterRecords (one per matched extraction).
// Returns empty slice if no extractions match (not an error).
func meter(payload EventPayload, config MeteringConfig) ([]MeterRecord, error) {
	// First pass: collect all source properties that will be extracted
	extractedProperties := make(map[string]bool)
	for _, extraction := range config.measurements {
		extractedProperties[extraction.SourceProperty().ToString()] = true
	}

	records := make([]MeterRecord, 0, len(config.measurements))

	for _, extraction := range config.measurements {
		// Check filter first
		if !extraction.Matches(payload.Properties) {
			continue // Skip this extraction
		}

		// Extract source property
		sourceKey := extraction.SourceProperty().ToString()
		sourceValue, exists := payload.Properties.Get(sourceKey)
		if !exists {
			return nil, fmt.Errorf("source property %q not found in payload", sourceKey)
		}

		// Cast to Decimal
		quantity, err := NewDecimal(sourceValue)
		if err != nil {
			return nil, fmt.Errorf("failed to parse property %q value %q as decimal: %w", sourceKey, sourceValue, err)
		}

		// Build dimensions: all properties except those extracted as measurements
		dimensionsMap := make(map[string]string)
		for _, key := range payload.Properties.Keys() {
			if !extractedProperties[key] {
				if value, ok := payload.Properties.Get(key); ok {
					dimensionsMap[key] = value
				}
			}
		}

		// Build MeterRecord
		// TODO: ID generation strategy - for now just concatenate payload.ID + unit
		recordID := payload.ID.ToString() + ":" + extraction.Unit().ToString()
		observedAt := payload.Time.ToTime()

		record, err := NewMeterRecord(specs.MeterRecordSpec{
			ID:          recordID,
			WorkspaceID: payload.WorkspaceID.ToString(),
			UniverseID:  payload.UniverseID.ToString(),
			Subject:     payload.Subject.ToString(),
			ObservedAt:  observedAt,
			Observations: []specs.ObservationSpec{
				specs.NewInstantObservation(
					quantity.String(),
					extraction.Unit().ToString(),
					observedAt,
				),
			},
			Dimensions:    dimensionsMap,
			SourceEventID: payload.ID.ToString(),
			// MeteredAt will default to time.Now() in NewMeterRecord
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create meter record: %w", err)
		}

		records = append(records, record)
	}

	return records, nil
}

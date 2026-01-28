package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	specs "metering-spec/specs"
	"time"
)

// unbundleObservations converts MeterRecordSpecs with bundled observations
// into separate specs (one per observation) for aggregation processing.
//
// This enables backwards compatibility: aggregation can process both old
// format (single Observation) and new format (multiple Observations array).
func unbundleObservations(recordSpecs []specs.MeterRecordSpec) []specs.MeterRecordSpec {
	result := make([]specs.MeterRecordSpec, 0, len(recordSpecs))

	for _, spec := range recordSpecs {
		// Try new field first (Observations array)
		observations := spec.Observations
		if len(observations) == 0 {
			// Fall back to old field (singular Observation) for backwards compatibility
			observations = []specs.ObservationSpec{spec.Observation}
		}

		// Create one spec per observation
		for _, observation := range observations {
			unbundledSpec := specs.MeterRecordSpec{
				ID:            spec.ID,
				WorkspaceID:   spec.WorkspaceID,
				UniverseID:    spec.UniverseID,
				Subject:       spec.Subject,
				ObservedAt:    spec.ObservedAt,
				Observation:   observation,  // Single observation for this unbundled spec
				Dimensions:    spec.Dimensions,
				SourceEventID: spec.SourceEventID,
				MeteredAt:     spec.MeteredAt,
			}
			result = append(result, unbundledSpec)
		}
	}

	return result
}

// Aggregate implements specs.Aggregate.
// Converts specs to domain objects, transforms, and converts back to specs.
func Aggregate(
	recordsInWindowSpec []specs.MeterRecordSpec,
	lastBeforeWindowSpec *specs.MeterRecordSpec,
	configSpec specs.AggregateConfigSpec,
) (specs.MeterReadingSpec, error) {
	// Unbundle observations: convert each MeterRecordSpec with multiple observations
	// into separate records (one per observation) for aggregation processing
	unbundledSpecs := unbundleObservations(recordsInWindowSpec)

	// Convert record specs to domain objects
	recordsInWindow := make([]MeterRecord, len(unbundledSpecs))
	for i, spec := range unbundledSpecs {
		record, err := NewMeterRecord(spec)
		if err != nil {
			return specs.MeterReadingSpec{}, fmt.Errorf("invalid record at index %d: %w", i, err)
		}
		recordsInWindow[i] = record
	}

	// Convert lastBefore spec if provided (unbundle if needed)
	var lastBeforeWindow *MeterRecord
	if lastBeforeWindowSpec != nil {
		// Unbundle observations and use first one (for time-weighted-avg)
		unbundledLast := unbundleObservations([]specs.MeterRecordSpec{*lastBeforeWindowSpec})
		if len(unbundledLast) > 0 {
			record, err := NewMeterRecord(unbundledLast[0])
			if err != nil {
				return specs.MeterReadingSpec{}, fmt.Errorf("invalid lastBeforeWindow: %w", err)
			}
			lastBeforeWindow = &record
		}
	}

	// Convert config spec to domain object
	config, err := NewAggregationConfig(configSpec)
	if err != nil {
		return specs.MeterReadingSpec{}, fmt.Errorf("invalid config: %w", err)
	}

	// Perform aggregation using domain objects
	reading, err := aggregate(recordsInWindow, lastBeforeWindow, config)
	if err != nil {
		return specs.MeterReadingSpec{}, err
	}

	// Convert domain object back to spec
	return specs.MeterReadingSpec{
		ID:           reading.ID.ToString(),
		WorkspaceID:  reading.WorkspaceID.ToString(),
		UniverseID:   reading.UniverseID.ToString(),
		Subject:      reading.Subject.ToString(),
		Window:       configSpec.Window,
		Value: specs.AggregateSpec{
			Quantity: reading.Measurement.Quantity().String(),
			Unit:     reading.Measurement.Unit().ToString(),
		},
		Aggregation:  reading.Aggregation.ToString(),
		RecordCount:  reading.RecordCount.ToInt(),
		CreatedAt:    reading.CreatedAt.ToTime(),
		MaxMeteredAt: reading.MaxMeteredAt.ToTime(),
	}, nil
}

// aggregate transforms MeterRecords into a MeterReading by applying aggregation.
// This is the private domain-level function that operates on domain objects.
func aggregate(
	recordsInWindow []MeterRecord,
	lastBeforeWindow *MeterRecord,
	config AggregationConfig,
) (MeterReading, error) {
	// Determine metadata source (first in-window record, or last-before if no in-window records)
	var metadataSource MeterRecord
	if len(recordsInWindow) > 0 {
		metadataSource = recordsInWindow[0]
	} else if lastBeforeWindow != nil {
		metadataSource = *lastBeforeWindow
	} else {
		return MeterReading{}, fmt.Errorf("cannot create meter reading: no records in window and no prior record")
	}

	// Perform aggregation - each type uses the parameters it needs
	aggregatedMeasurement, recordCount, err := config.Aggregation().Aggregate(recordsInWindow, lastBeforeWindow, config.Window())
	if err != nil {
		return MeterReading{}, fmt.Errorf("failed to aggregate with %s: %w", config.Aggregation().ToString(), err)
	}

	// Compute MaxMeteredAt from all records (for watermarking)
	maxMeteredAt := computeMaxMeteredAt(recordsInWindow, lastBeforeWindow)

	// Build MeterReading
	id := computeMeterReadingID(
		metadataSource.Subject,
		aggregatedMeasurement.Unit(),
		config.Window(),
		config.Aggregation(),
	)

	workspaceID, err := NewMeterReadingWorkspaceID(metadataSource.WorkspaceID.ToString())
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid workspace ID: %w", err)
	}

	universeID, err := NewMeterReadingUniverseID(metadataSource.UniverseID.ToString())
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid universe ID: %w", err)
	}

	recordCountVO, err := NewMeterReadingRecordCount(recordCount)
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid record count: %w", err)
	}

	createdAt, err := NewMeterReadingCreatedAt(time.Now())
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid created at: %w", err)
	}

	maxMeteredAtVO, err := NewMeterReadingMaxMeteredAt(maxMeteredAt)
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid max metered at: %w", err)
	}

	subject, err := NewMeterReadingSubject(metadataSource.Subject.ToString())
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid subject: %w", err)
	}

	return MeterReading{
		ID:           id,
		WorkspaceID:  workspaceID,
		UniverseID:   universeID,
		Subject:      subject,
		Window:       config.Window(),
		Measurement:  aggregatedMeasurement,
		Aggregation:  config.Aggregation(),
		RecordCount:  recordCountVO,
		CreatedAt:    createdAt,
		MaxMeteredAt: maxMeteredAtVO,
	}, nil
}

// computeMaxMeteredAt finds the maximum MeteredAt timestamp from all records.
func computeMaxMeteredAt(recordsInWindow []MeterRecord, lastBeforeWindow *MeterRecord) time.Time {
	var maxMeteredAt time.Time

	// Check all records in window
	for _, record := range recordsInWindow {
		if record.MeteredAt.ToTime().After(maxMeteredAt) {
			maxMeteredAt = record.MeteredAt.ToTime()
		}
	}

	// Check last-before record (used in gauge aggregations)
	if lastBeforeWindow != nil && lastBeforeWindow.MeteredAt.ToTime().After(maxMeteredAt) {
		maxMeteredAt = lastBeforeWindow.MeteredAt.ToTime()
	}

	return maxMeteredAt
}

// computeMeterReadingID generates a deterministic ID from the reading's key fields.
func computeMeterReadingID(
	subject MeterRecordSubject,
	unit MeasurementUnit,
	window TimeWindow,
	aggregation MeterReadingAggregation,
) MeterReadingID {
	input := fmt.Sprintf("%s|%s|%s|%s|%s",
		subject.ToString(),
		unit.ToString(),
		window.Start().ToTime().UTC().Format(time.RFC3339),
		window.End().ToTime().UTC().Format(time.RFC3339),
		aggregation.ToString(),
	)
	hash := sha256.Sum256([]byte(input))
	hashStr := hex.EncodeToString(hash[:16])
	return MeterReadingID{value: hashStr}
}

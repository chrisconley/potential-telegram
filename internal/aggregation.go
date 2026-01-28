package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	specs "metering-spec/specs"
	"time"
)

// Aggregate implements specs.Aggregate.
// Converts specs to domain objects, transforms, and converts back to specs.
func Aggregate(
	recordsInWindowSpec []specs.MeterRecordSpec,
	lastBeforeWindowSpec *specs.MeterRecordSpec,
	configSpec specs.AggregateConfigSpec,
) (specs.MeterReadingSpec, error) {
	// Convert record specs to domain objects
	recordsInWindow := make([]MeterRecord, len(recordsInWindowSpec))
	for i, spec := range recordsInWindowSpec {
		record, err := NewMeterRecord(spec)
		if err != nil {
			return specs.MeterReadingSpec{}, fmt.Errorf("invalid record at index %d: %w", i, err)
		}
		recordsInWindow[i] = record
	}

	// Convert lastBefore spec if provided
	var lastBeforeWindow *MeterRecord
	if lastBeforeWindowSpec != nil {
		record, err := NewMeterRecord(*lastBeforeWindowSpec)
		if err != nil {
			return specs.MeterReadingSpec{}, fmt.Errorf("invalid lastBeforeWindow: %w", err)
		}
		lastBeforeWindow = &record
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
		Value: specs.AggregatedValueSpec{
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

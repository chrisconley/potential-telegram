package internal

import (
	"fmt"
	specs "metering-spec/specs"
	"time"
)

type MeterReading struct {
	ID               MeterReadingID
	WorkspaceID      MeterReadingWorkspaceID
	UniverseID       MeterReadingUniverseID
	Subject          MeterReadingSubject
	Window           TimeWindow
	Value            AggregateValue  // NEW - alongside Measurement
	Measurement      Measurement     // OLD - keep for backwards compat
	Aggregation      MeterReadingAggregation
	RecordCount      MeterReadingRecordCount
	CreatedAt        MeterReadingCreatedAt
	MaxMeteredAt     MeterReadingMaxMeteredAt
}

func NewMeterReading(spec specs.MeterReadingSpec) (MeterReading, error) {
	id, err := NewMeterReadingID(spec.ID)
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid ID: %w", err)
	}

	workspaceID, err := NewMeterReadingWorkspaceID(spec.WorkspaceID)
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid workspace ID: %w", err)
	}

	universeID, err := NewMeterReadingUniverseID(spec.UniverseID)
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid universe ID: %w", err)
	}

	subject, err := NewMeterReadingSubject(spec.Subject)
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid subject: %w", err)
	}

	window, err := NewTimeWindow(spec.Window)
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid window: %w", err)
	}

	quantity, err := NewDecimal(spec.Value.Quantity)
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid quantity: %w", err)
	}

	unit, err := NewUnit(spec.Value.Unit)
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid unit: %w", err)
	}

	// NEW: Value (AggregateValue)
	value := NewAggregateValue(quantity, unit)

	// OLD: Measurement (same data for backwards compat)
	measurement := NewMeasurement(quantity, unit)

	aggregation, err := NewMeterReadingAggregation(spec.Aggregation)
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid aggregation: %w", err)
	}

	recordCount, err := NewMeterReadingRecordCount(spec.RecordCount)
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid record count: %w", err)
	}

	createdAt, err := NewMeterReadingCreatedAt(spec.CreatedAt)
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid created at: %w", err)
	}

	maxMeteredAt, err := NewMeterReadingMaxMeteredAt(spec.MaxMeteredAt)
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid max metered at: %w", err)
	}

	return MeterReading{
		ID:           id,
		WorkspaceID:  workspaceID,
		UniverseID:   universeID,
		Subject:      subject,
		Window:       window,
		Value:        value,       // NEW
		Measurement:  measurement, // OLD
		Aggregation:  aggregation,
		RecordCount:  recordCount,
		CreatedAt:    createdAt,
		MaxMeteredAt: maxMeteredAt,
	}, nil
}

type MeterReadingID struct {
	value string
}

func NewMeterReadingID(value string) (MeterReadingID, error) {
	if value == "" {
		return MeterReadingID{}, fmt.Errorf("ID is required")
	}
	return MeterReadingID{value: value}, nil
}

func (id MeterReadingID) ToString() string {
	return id.value
}

type MeterReadingWorkspaceID struct {
	value string
}

func NewMeterReadingWorkspaceID(value string) (MeterReadingWorkspaceID, error) {
	if value == "" {
		return MeterReadingWorkspaceID{}, fmt.Errorf("workspace ID is required")
	}
	return MeterReadingWorkspaceID{value: value}, nil
}

func (id MeterReadingWorkspaceID) ToString() string {
	return id.value
}

type MeterReadingUniverseID struct {
	value string
}

func NewMeterReadingUniverseID(value string) (MeterReadingUniverseID, error) {
	if value == "" {
		return MeterReadingUniverseID{}, fmt.Errorf("universe ID is required")
	}
	return MeterReadingUniverseID{value: value}, nil
}

func (u MeterReadingUniverseID) ToString() string {
	return u.value
}

type MeterReadingSubject struct {
	value string
}

func NewMeterReadingSubject(value string) (MeterReadingSubject, error) {
	if value == "" {
		return MeterReadingSubject{}, fmt.Errorf("subject is required")
	}
	return MeterReadingSubject{value: value}, nil
}

func (s MeterReadingSubject) ToString() string {
	return s.value
}

type TimeWindow struct {
	start TimeWindowStart
	end   TimeWindowEnd
}

func NewTimeWindow(spec specs.TimeWindowSpec) (TimeWindow, error) {
	start, err := NewTimeWindowStart(spec.Start)
	if err != nil {
		return TimeWindow{}, fmt.Errorf("invalid start: %w", err)
	}

	end, err := NewTimeWindowEnd(spec.End)
	if err != nil {
		return TimeWindow{}, fmt.Errorf("invalid end: %w", err)
	}

	if !spec.Start.Before(spec.End) && !spec.Start.Equal(spec.End) {
		return TimeWindow{}, fmt.Errorf("start must be before or equal to end")
	}

	return TimeWindow{
		start: start,
		end:   end,
	}, nil
}

// NewInstantWindow creates a TimeWindow for an instant observation (Start == End)
func NewInstantWindow(instant time.Time) (TimeWindow, error) {
	return NewTimeWindow(specs.TimeWindowSpec{
		Start: instant,
		End:   instant,
	})
}

// TimeWindowFromSpec creates a TimeWindow from a spec (alias for NewTimeWindow)
func TimeWindowFromSpec(spec specs.TimeWindowSpec) (TimeWindow, error) {
	return NewTimeWindow(spec)
}

func (w TimeWindow) Start() TimeWindowStart {
	return w.start
}

func (w TimeWindow) End() TimeWindowEnd {
	return w.end
}

// IsInstant returns true if this window represents an instant (Start == End)
func (w TimeWindow) IsInstant() bool {
	return w.start.ToTime().Equal(w.end.ToTime())
}

// ToSpec converts TimeWindow to specs.TimeWindowSpec
func (w TimeWindow) ToSpec() specs.TimeWindowSpec {
	return specs.TimeWindowSpec{
		Start: w.start.ToTime(),
		End:   w.end.ToTime(),
	}
}

type TimeWindowStart struct {
	value time.Time
}

func NewTimeWindowStart(value time.Time) (TimeWindowStart, error) {
	if value.IsZero() {
		return TimeWindowStart{}, fmt.Errorf("start is required")
	}
	return TimeWindowStart{value: value}, nil
}

func (t TimeWindowStart) ToTime() time.Time {
	return t.value
}

type TimeWindowEnd struct {
	value time.Time
}

func NewTimeWindowEnd(value time.Time) (TimeWindowEnd, error) {
	if value.IsZero() {
		return TimeWindowEnd{}, fmt.Errorf("end is required")
	}
	return TimeWindowEnd{value: value}, nil
}

func (t TimeWindowEnd) ToTime() time.Time {
	return t.value
}

// AggregateValue represents a computed aggregation result.
// Unlike Observation which includes temporal context, AggregateValue does not
// have a Window field—temporal context is provided by the parent MeterReading.
type AggregateValue struct {
	quantity Decimal
	unit     Unit
}

func NewAggregateValue(quantity Decimal, unit Unit) AggregateValue {
	return AggregateValue{
		quantity: quantity,
		unit:     unit,
	}
}

func (a AggregateValue) Quantity() Decimal {
	return a.quantity
}

func (a AggregateValue) Unit() Unit {
	return a.unit
}

type MeterReadingAggregation struct {
	value string
}

func NewMeterReadingAggregation(value string) (MeterReadingAggregation, error) {
	if value == "" {
		return MeterReadingAggregation{}, fmt.Errorf("aggregation is required")
	}

	// Validate aggregation type
	switch value {
	case "sum", "max", "time-weighted-avg", "latest", "min":
		// Valid
	default:
		return MeterReadingAggregation{}, fmt.Errorf("invalid aggregation type: %q", value)
	}

	return MeterReadingAggregation{value: value}, nil
}

func (a MeterReadingAggregation) ToString() string {
	return a.value
}

func (a MeterReadingAggregation) IsSum() bool {
	return a.value == "sum"
}

func (a MeterReadingAggregation) IsMax() bool {
	return a.value == "max"
}

func (a MeterReadingAggregation) IsTimeWeightedAvg() bool {
	return a.value == "time-weighted-avg"
}

func (a MeterReadingAggregation) IsLatest() bool {
	return a.value == "latest"
}

func (a MeterReadingAggregation) IsMin() bool {
	return a.value == "min"
}

// Aggregate applies this aggregation type to the given records.
// Each aggregation type uses the parameters it needs:
//   - sum/max/min/latest: use recordsInWindow only
//   - time-weighted-avg: uses all parameters
//
// Returns the aggregated value, record count, and any error.
func (a MeterReadingAggregation) Aggregate(
	recordsInWindow []MeterRecord,
	lastBeforeWindow *MeterRecord,
	window TimeWindow,
) (AggregateValue, int, error) {
	switch a.value {
	case "sum":
		value, err := sumRecords(recordsInWindow)
		return value, len(recordsInWindow), err

	case "max":
		value, err := maxRecords(recordsInWindow)
		return value, len(recordsInWindow), err

	case "min":
		value, err := minRecords(recordsInWindow)
		return value, len(recordsInWindow), err

	case "latest":
		value, err := latestRecord(recordsInWindow)
		return value, len(recordsInWindow), err

	case "time-weighted-avg":
		value, err := timeWeightedAvgRecords(recordsInWindow, lastBeforeWindow, window)
		recordCount := len(recordsInWindow)
		if lastBeforeWindow != nil {
			recordCount++ // Count the carry-forward record
		}
		return value, recordCount, err

	default:
		return AggregateValue{}, 0, fmt.Errorf("unsupported aggregation type: %s", a.value)
	}
}

type MeterReadingRecordCount struct {
	value int
}

func NewMeterReadingRecordCount(value int) (MeterReadingRecordCount, error) {
	if value < 0 {
		return MeterReadingRecordCount{}, fmt.Errorf("record count cannot be negative")
	}
	return MeterReadingRecordCount{value: value}, nil
}

func (r MeterReadingRecordCount) ToInt() int {
	return r.value
}

type MeterReadingCreatedAt struct {
	value time.Time
}

func NewMeterReadingCreatedAt(value time.Time) (MeterReadingCreatedAt, error) {
	if value.IsZero() {
		return MeterReadingCreatedAt{}, fmt.Errorf("created at is required")
	}
	return MeterReadingCreatedAt{value: value}, nil
}

func (c MeterReadingCreatedAt) ToTime() time.Time {
	return c.value
}

type MeterReadingMaxMeteredAt struct {
	value time.Time
}

func NewMeterReadingMaxMeteredAt(value time.Time) (MeterReadingMaxMeteredAt, error) {
	if value.IsZero() {
		return MeterReadingMaxMeteredAt{}, fmt.Errorf("max metered at is required")
	}
	return MeterReadingMaxMeteredAt{value: value}, nil
}

func (m MeterReadingMaxMeteredAt) ToTime() time.Time {
	return m.value
}

// sumRecords returns the sum of all record observations.
// Returns error if records is empty or observations are incompatible.
func sumRecords(records []MeterRecord) (AggregateValue, error) {
	if len(records) == 0 {
		return AggregateValue{}, fmt.Errorf("cannot sum empty records")
	}

	// Use first observation from first record
	sum := records[0].Observations[0].Quantity()
	unit := records[0].Observations[0].Unit()

	for _, r := range records[1:] {
		sum = sum.Add(r.Observations[0].Quantity())
	}

	return NewAggregateValue(sum, unit), nil
}

// maxRecords returns the maximum observation from all records.
// Returns error if records is empty.
func maxRecords(records []MeterRecord) (AggregateValue, error) {
	if len(records) == 0 {
		return AggregateValue{}, fmt.Errorf("cannot find max of empty records")
	}

	maxQuantity := records[0].Observations[0].Quantity()
	unit := records[0].Observations[0].Unit()

	for _, r := range records[1:] {
		if r.Observations[0].Quantity().Cmp(maxQuantity) > 0 {
			maxQuantity = r.Observations[0].Quantity()
		}
	}

	return NewAggregateValue(maxQuantity, unit), nil
}

// minRecords returns the minimum observation from all records.
// Returns error if records is empty.
func minRecords(records []MeterRecord) (AggregateValue, error) {
	if len(records) == 0 {
		return AggregateValue{}, fmt.Errorf("cannot find min of empty records")
	}

	minQuantity := records[0].Observations[0].Quantity()
	unit := records[0].Observations[0].Unit()

	for _, r := range records[1:] {
		if r.Observations[0].Quantity().Cmp(minQuantity) < 0 {
			minQuantity = r.Observations[0].Quantity()
		}
	}

	return NewAggregateValue(minQuantity, unit), nil
}

// latestRecord returns the observation from the most recent record by ObservedAt timestamp.
// Returns error if records is empty.
func latestRecord(records []MeterRecord) (AggregateValue, error) {
	if len(records) == 0 {
		return AggregateValue{}, fmt.Errorf("cannot find latest of empty records")
	}

	latest := records[0]
	for _, r := range records[1:] {
		if r.ObservedAt.ToTime().After(latest.ObservedAt.ToTime()) {
			latest = r
		}
	}

	return NewAggregateValue(latest.Observations[0].Quantity(), latest.Observations[0].Unit()), nil
}

// timeWeightedAvgRecords computes the time-weighted average of gauge readings.
// Uses step interpolation: each value holds until the next reading (or window end).
//
// Parameters:
//   - recordsInWindow: Readings within [WindowStart, WindowEnd)
//   - lastBeforeWindow: Last reading before WindowStart (carries forward initial state)
//   - window: Time window for aggregation
//
// Algorithm:
//  1. Combine lastBeforeWindow (if exists) + recordsInWindow
//  2. Sort by RecordedAt timestamp
//  3. For each reading, compute: value × duration_until_next_reading
//  4. Sum weighted values and divide by total window duration
func timeWeightedAvgRecords(
	recordsInWindow []MeterRecord,
	lastBeforeWindow *MeterRecord,
	window TimeWindow,
) (AggregateValue, error) {
	// Combine records (last-before + in-window)
	var allRecords []MeterRecord
	if lastBeforeWindow != nil {
		allRecords = append(allRecords, *lastBeforeWindow)
	}
	allRecords = append(allRecords, recordsInWindow...)

	if len(allRecords) == 0 {
		return AggregateValue{}, fmt.Errorf("cannot compute time-weighted average: no records")
	}

	// Sort by ObservedAt timestamp
	sortedRecords := make([]MeterRecord, len(allRecords))
	copy(sortedRecords, allRecords)
	for i := 0; i < len(sortedRecords); i++ {
		for j := i + 1; j < len(sortedRecords); j++ {
			if sortedRecords[j].ObservedAt.ToTime().Before(sortedRecords[i].ObservedAt.ToTime()) {
				sortedRecords[i], sortedRecords[j] = sortedRecords[j], sortedRecords[i]
			}
		}
	}

	// Compute weighted sum: Σ(value × duration)
	unit := sortedRecords[0].Observations[0].Unit()
	weightedSum, _ := NewDecimal("0")

	windowStart := window.Start().ToTime()
	windowEnd := window.End().ToTime()

	for i, record := range sortedRecords {
		// Determine when this value is valid (from this timestamp until next, or window end)
		validFrom := record.ObservedAt.ToTime()
		if validFrom.Before(windowStart) {
			validFrom = windowStart // Clamp to window start
		}

		validUntil := windowEnd
		if i+1 < len(sortedRecords) {
			nextTimestamp := sortedRecords[i+1].ObservedAt.ToTime()
			if nextTimestamp.Before(windowEnd) {
				validUntil = nextTimestamp
			}
		}

		// Compute duration this value was active within the window
		if validUntil.After(validFrom) {
			durationSeconds := validUntil.Sub(validFrom).Seconds()
			duration, _ := NewDecimal(fmt.Sprintf("%.15f", durationSeconds))

			contribution := record.Observations[0].Quantity().Mul(duration)
			weightedSum = weightedSum.Add(contribution)
		}
	}

	// Divide by total window duration to get average
	totalSeconds := windowEnd.Sub(windowStart).Seconds()
	totalDuration, _ := NewDecimal(fmt.Sprintf("%.15f", totalSeconds))

	avg := weightedSum.Div(totalDuration)

	return NewAggregateValue(avg, unit), nil
}

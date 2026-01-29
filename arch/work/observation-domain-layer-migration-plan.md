# Migration Plan: Complete Observation-Temporal-Context ADR in Domain Layer

**Date:** 2026-01-28
**Status:** Ready for Execution
**Context:** Issue #4 updated specs but not domain layer. Complete the migration to align domain objects with specs.

---

## Problem Summary

**What was done:** Specs updated to use `Observations []ObservationSpec` and `ObservedAt`
**What remains:** Domain layer still uses `Measurement` (singular) and `RecordedAt`

**Root cause:** Issue #4 scoped both atomicity AND observation-temporal-context ADR, but implementation only updated specs.

**Design smell:** Unbundling logic exists to bridge the gap between specs (bundled) and domain objects (singular).

**Goal:** Eliminate unbundling by aligning domain layer with specs.

---

## Discovery Results

### Current State: Domain Layer

**Types still using Measurement:**
- `MeterRecord.Measurement` (internal/meterrecord.go:15)
- `MeterReading.Measurement` (internal/meterreading.go:15)
- `Measurement` type itself (internal/meterrecord.go:216-234)
- `MeasurementUnit` type (internal/meterrecord.go:236-248)

**Types still using RecordedAt:**
- `MeterRecord.RecordedAt` (internal/meterrecord.go:14)
- `MeterRecordRecordedAt` type (internal/meterrecord.go:123-136)

**Aggregation functions operating on Measurement:**
- `sumRecords() (Measurement, error)` (internal/meterreading.go:350)
- `maxRecords() (Measurement, error)` (internal/meterreading.go:367)
- `minRecords() (Measurement, error)` (internal/meterreading.go:384)
- `latestRecord() (Measurement, error)` (internal/meterreading.go:401)
- `timeWeightedAverageRecords() (Measurement, error)` (internal/meterreading.go:433)

**Unbundling workaround:**
- `unbundleObservations()` function (internal/aggregation.go) - should be removed

### Spec State (Already Updated)

**Specs use:**
- `Observations []ObservationSpec` (specs/meterrecord.go:63)
- `ObservedAt time.Time` (specs/meterrecord.go:55)
- `Value AggregateSpec` (specs/meterreading.go)
- Helper constructors: `NewInstantObservation()`, `NewSpanObservation()` (specs/observation.go)

---

## Migration Strategy

**Pattern:** Add, Migrate, Remove (Principle #10)

**Safety:** Every commit compiles and passes `go test ./...`

**Scope:** This is a PLANNED refactoring (multiple packages, signature changes, breaking changes)

---

## Migration Steps

### Phase 1: Add New Types Alongside Old (Backwards Compatible)

**Goal:** Introduce Observation and Aggregate types in domain layer without breaking existing code.

#### Step 1.1: Add Observation type to internal/meterrecord.go

**Action:**
```go
// Add after Measurement type definition

// Observation represents a single observation from an event
type Observation struct {
	quantity Decimal
	unit     Unit
	window   TimeWindow
}

func NewObservation(quantity Decimal, unit Unit, window TimeWindow) Observation {
	return Observation{
		quantity: quantity,
		unit:     unit,
		window:   window,
	}
}

func (o Observation) Quantity() Decimal {
	return o.quantity
}

func (o Observation) Unit() Unit {
	return o.unit
}

func (o Observation) Window() TimeWindow {
	return o.window
}
```

**Safety check:** `go test ./...` (no callers yet, just adding new type)

**Commit:** "Add Observation type alongside Measurement"

---

#### Step 1.2: Add Aggregate type to internal/meterreading.go

**Action:**
```go
// Add to internal/meterreading.go

// Aggregate represents a computed aggregation result
type Aggregate struct {
	quantity Decimal
	unit     Unit
}

func NewAggregate(quantity Decimal, unit Unit) Aggregate {
	return Aggregate{
		quantity: quantity,
		unit:     unit,
	}
}

func (a Aggregate) Quantity() Decimal {
	return a.quantity
}

func (a Aggregate) Unit() Unit {
	return a.unit
}
```

**Safety check:** `go test ./...`

**Commit:** "Add Aggregate type alongside Measurement in MeterReading"

---

#### Step 1.3: Rename MeasurementUnit → Unit

**Rationale:** "MeasurementUnit" perpetuates deprecated naming. Just "Unit" is clearer.

**Action:** Use refactoring tool or search/replace
- `type MeasurementUnit` → `type Unit`
- `NewMeasurementUnit` → `NewUnit`

**Files affected:**
- internal/meterrecord.go
- internal/meteringconfig.go
- internal/aggregation.go
- All callers

**Safety check:** `go test ./...` (pure rename, no semantic change)

**Commit:** "Rename MeasurementUnit → Unit throughout domain layer"

---

#### Step 1.4: Add TimeWindow type to internal package

**Rationale:** Observations need temporal context (`Window` field)

**Action:**
```go
// Add to internal/timewindow.go (new file)

package internal

import (
	"fmt"
	"metering-spec/specs"
	"time"
)

// TimeWindow represents a temporal extent [Start, End)
// For instant observations: Start == End
// For time-spanning observations: Start < End
type TimeWindow struct {
	start time.Time
	end   time.Time
}

func NewTimeWindow(start, end time.Time) (TimeWindow, error) {
	if start.IsZero() || end.IsZero() {
		return TimeWindow{}, fmt.Errorf("start and end are required")
	}
	if end.Before(start) {
		return TimeWindow{}, fmt.Errorf("end must not be before start")
	}
	return TimeWindow{start: start, end: end}, nil
}

func NewInstantWindow(instant time.Time) (TimeWindow, error) {
	return NewTimeWindow(instant, instant)
}

func (w TimeWindow) Start() time.Time {
	return w.start
}

func (w TimeWindow) End() time.Time {
	return w.end
}

func (w TimeWindow) IsInstant() bool {
	return w.start.Equal(w.end)
}

func (w TimeWindow) ToSpec() specs.TimeWindowSpec {
	return specs.TimeWindowSpec{
		Start: w.start,
		End:   w.end,
	}
}

func TimeWindowFromSpec(spec specs.TimeWindowSpec) (TimeWindow, error) {
	return NewTimeWindow(spec.Start, spec.End)
}
```

**Safety check:** `go test ./...`

**Commit:** "Add TimeWindow domain object for observation temporal context"

---

#### Step 1.5: Add Observations field to MeterRecord (alongside Measurement)

**Action:**
```go
// In internal/meterrecord.go

type MeterRecord struct {
	ID            MeterRecordID
	WorkspaceID   MeterRecordWorkspaceID
	UniverseID    MeterRecordUniverseID
	Subject       MeterRecordSubject
	ObservedAt    MeterRecordObservedAt  // NEW - alongside RecordedAt
	RecordedAt    MeterRecordRecordedAt  // OLD - keep for now
	Observations  []Observation          // NEW - alongside Measurement
	Measurement   Measurement            // OLD - keep for now
	Dimensions    MeterRecordDimensions
	SourceEventID MeterRecordSourceEventID
	MeteredAt     MeterRecordMeteredAt
}
```

**Update NewMeterRecord to populate BOTH fields:**
```go
func NewMeterRecord(spec specs.MeterRecordSpec) (MeterRecord, error) {
	// ... existing validation ...

	// Build observations from spec.Observations array
	observations := make([]Observation, len(spec.Observations))
	for i, obsSpec := range spec.Observations {
		quantity, err := NewDecimal(obsSpec.Quantity)
		if err != nil {
			return MeterRecord{}, fmt.Errorf("invalid observation[%d] quantity: %w", i, err)
		}

		unit, err := NewUnit(obsSpec.Unit)
		if err != nil {
			return MeterRecord{}, fmt.Errorf("invalid observation[%d] unit: %w", i, err)
		}

		window, err := TimeWindowFromSpec(obsSpec.Window)
		if err != nil {
			return MeterRecord{}, fmt.Errorf("invalid observation[%d] window: %w", i, err)
		}

		observations[i] = NewObservation(quantity, unit, window)
	}

	// OLD: Extract first observation for backwards compatibility
	var measurement Measurement
	if len(observations) > 0 {
		measurement = NewMeasurement(
			observations[0].Quantity(),
			observations[0].Unit(),
		)
	}

	// NEW: ObservedAt
	observedAt, err := NewMeterRecordObservedAt(spec.ObservedAt)
	if err != nil {
		return MeterRecord{}, fmt.Errorf("invalid observed at: %w", err)
	}

	// OLD: RecordedAt (same value for backwards compat)
	recordedAt, err := NewMeterRecordRecordedAt(spec.ObservedAt)
	if err != nil {
		return MeterRecord{}, fmt.Errorf("invalid recorded at: %w", err)
	}

	return MeterRecord{
		ID:            id,
		WorkspaceID:   workspaceID,
		UniverseID:    universeID,
		Subject:       subject,
		ObservedAt:    observedAt,    // NEW
		RecordedAt:    recordedAt,    // OLD
		Observations:  observations,  // NEW
		Measurement:   measurement,   // OLD
		Dimensions:    dimensions,
		SourceEventID: sourceEventID,
		MeteredAt:     meteredAt,
	}, nil
}
```

**Safety check:** `go test ./...` (both fields populated, no behavior change)

**Commit:** "Add Observations and ObservedAt fields alongside old fields in MeterRecord"

---

#### Step 1.6: Add MeterRecordObservedAt type

**Action:**
```go
// In internal/meterrecord.go

type MeterRecordObservedAt struct {
	value time.Time
}

func NewMeterRecordObservedAt(value time.Time) (MeterRecordObservedAt, error) {
	if value.IsZero() {
		return MeterRecordObservedAt{}, fmt.Errorf("observed at is required")
	}
	return MeterRecordObservedAt{value: value}, nil
}

func (t MeterRecordObservedAt) ToTime() time.Time {
	return t.value
}
```

**Safety check:** `go test ./...`

**Commit:** "Add MeterRecordObservedAt type alongside MeterRecordRecordedAt"

---

#### Step 1.7: Add Value field to MeterReading (alongside Measurement)

**Action:**
```go
// In internal/meterreading.go

type MeterReading struct {
	ID               MeterReadingID
	WorkspaceID      MeterReadingWorkspaceID
	UniverseID       MeterReadingUniverseID
	Subject          MeterReadingSubject
	Window           TimeWindow
	Value            Aggregate    // NEW - alongside Measurement
	Measurement      Measurement  // OLD - keep for now
	Aggregation      MeterReadingAggregation
	RecordCount      MeterReadingRecordCount
	CreatedAt        MeterReadingCreatedAt
	MaxMeteredAt     MeterReadingMaxMeteredAt
}
```

**Update NewMeterReading to populate BOTH fields:**
```go
func NewMeterReading(spec specs.MeterReadingSpec) (MeterReading, error) {
	// ... existing validation ...

	// NEW: Value from AggregateSpec
	valueQuantity, err := NewDecimal(spec.Value.Quantity)
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid value quantity: %w", err)
	}
	valueUnit, err := NewUnit(spec.Value.Unit)
	if err != nil {
		return MeterReading{}, fmt.Errorf("invalid value unit: %w", err)
	}
	value := NewAggregate(valueQuantity, valueUnit)

	// OLD: Measurement (same data for backwards compat)
	measurement := NewMeasurement(valueQuantity, valueUnit)

	return MeterReading{
		// ... other fields ...
		Value:       value,       // NEW
		Measurement: measurement, // OLD
		// ... other fields ...
	}, nil
}
```

**Safety check:** `go test ./...`

**Commit:** "Add Value (Aggregate) field alongside Measurement in MeterReading"

---

### Phase 2: Migrate Consumers to New Fields

**Goal:** Update all code reading from old fields to use new fields instead.

#### Step 2.1: Migrate aggregation functions to return Aggregate

**Current signatures:**
```go
func sumRecords(records []MeterRecord) (Measurement, error)
func maxRecords(records []MeterRecord) (Measurement, error)
func minRecords(records []MeterRecord) (Measurement, error)
func latestRecord(records []MeterRecord) (Measurement, error)
func timeWeightedAverageRecords(...) (Measurement, error)
```

**New signatures:**
```go
func sumRecords(records []MeterRecord) (Aggregate, error)
func maxRecords(records []MeterRecord) (Aggregate, error)
func minRecords(records []MeterRecord) (Aggregate, error)
func latestRecord(records []MeterRecord) (Aggregate, error)
func timeWeightedAverageRecords(...) (Aggregate, error)
```

**Implementation changes:**
- Read from `record.Observations[0]` instead of `record.Measurement`
- Return `NewAggregate()` instead of returning `Measurement`
- Use `record.ObservedAt` instead of `record.RecordedAt` for time-weighted avg

**Files affected:**
- internal/meterreading.go (function bodies)
- internal/meterreading.go (aggregateRecords caller)

**Safety check:** `go test ./...`

**Commit:** "Migrate aggregation functions to use Observations and return Aggregate"

---

#### Step 2.2: Migrate Meter() function to use new fields only

**Current code in internal/metering.go:**
```go
observations[i] = specs.NewInstantObservation(
    record.Measurement.Quantity().String(),
    record.Measurement.Unit().ToString(),
    observedAt,
)
```

**New code:**
```go
// record.Observations already exist - just convert to specs
for i, obs := range record.Observations {
    observations[i] = specs.ObservationSpec{
        Quantity: obs.Quantity().String(),
        Unit:     obs.Unit().ToString(),
        Window:   obs.Window().ToSpec(),
    }
}
```

**Safety check:** `go test ./...`

**Commit:** "Migrate Meter() to use Observations field instead of Measurement"

---

#### Step 2.3: Migrate Aggregate() function to use new fields

**Current code in internal/aggregation.go:**
```go
Value: specs.AggregateSpec{
    Quantity: reading.Measurement.Quantity().String(),
    Unit:     reading.Measurement.Unit().ToString(),
}
```

**New code:**
```go
Value: specs.AggregateSpec{
    Quantity: reading.Value.Quantity().String(),
    Unit:     reading.Value.Unit().ToString(),
}
```

**Safety check:** `go test ./...`

**Commit:** "Migrate Aggregate() to use Value field instead of Measurement"

---

#### Step 2.4: Remove unbundleObservations() function

**Rationale:** No longer needed - domain layer now supports multiple observations natively.

**Action:**
- Delete `unbundleObservations()` function from internal/aggregation.go
- Remove call to `unbundleObservations()` in `Aggregate()` function

**Before:**
```go
func Aggregate(...) {
    recordSpecs = unbundleObservations(recordSpecs)
    // ... rest of function
}
```

**After:**
```go
func Aggregate(...) {
    // No unbundling needed - MeterRecord.Observations already supports multiple
    // ... rest of function
}
```

**Safety check:** `go test ./...`

**Commit:** "Remove unbundleObservations workaround (no longer needed)"

---

### Phase 3: Remove Old Fields (Breaking Change)

**Goal:** Delete deprecated fields once all code migrated to new fields.

#### Step 3.1: Verify zero usage of old fields

**Check:**
```bash
grep -r "\.Measurement\b" internal/ --include="*.go"
grep -r "\.RecordedAt\b" internal/ --include="*.go"
```

**Expected:** Zero matches (or only type definitions)

**If non-zero:** Go back to Phase 2, missed a migration.

---

#### Step 3.2: Remove Measurement field from MeterRecord

**Action:**
```go
type MeterRecord struct {
	ID            MeterRecordID
	WorkspaceID   MeterRecordWorkspaceID
	UniverseID    MeterRecordUniverseID
	Subject       MeterRecordSubject
	ObservedAt    MeterRecordObservedAt
	// RecordedAt    MeterRecordRecordedAt  // REMOVED
	Observations  []Observation
	// Measurement   Measurement            // REMOVED
	Dimensions    MeterRecordDimensions
	SourceEventID MeterRecordSourceEventID
	MeteredAt     MeterRecordMeteredAt
}
```

**Update NewMeterRecord:**
- Remove code that populates `Measurement` field
- Remove code that populates `RecordedAt` field

**Safety check:** `go test ./...` (compiler catches any missed usages)

**Commit:** "Remove Measurement and RecordedAt fields from MeterRecord"

---

#### Step 3.3: Remove Measurement type definition

**Action:**
```go
// DELETE:
// type Measurement struct { ... }
// func NewMeasurement(...) Measurement { ... }
// func (m Measurement) Quantity() Decimal { ... }
// func (m Measurement) Unit() Unit { ... }
```

**Safety check:** `go test ./...`

**Commit:** "Remove Measurement type (replaced by Observation and Aggregate)"

---

#### Step 3.4: Remove MeterRecordRecordedAt type

**Action:**
```go
// DELETE:
// type MeterRecordRecordedAt struct { ... }
// func NewMeterRecordRecordedAt(...) { ... }
// func (t MeterRecordRecordedAt) ToTime() time.Time { ... }
```

**Safety check:** `go test ./...`

**Commit:** "Remove MeterRecordRecordedAt type (replaced by MeterRecordObservedAt)"

---

#### Step 3.5: Remove Measurement field from MeterReading

**Action:**
```go
type MeterReading struct {
	ID               MeterReadingID
	WorkspaceID      MeterReadingWorkspaceID
	UniverseID       MeterReadingUniverseID
	Subject          MeterReadingSubject
	Window           TimeWindow
	Value            Aggregate
	// Measurement      Measurement  // REMOVED
	Aggregation      MeterReadingAggregation
	RecordCount      MeterReadingRecordCount
	CreatedAt        MeterReadingCreatedAt
	MaxMeteredAt     MeterReadingMaxMeteredAt
}
```

**Update NewMeterReading:**
- Remove code that populates `Measurement` field

**Safety check:** `go test ./...`

**Commit:** "Remove Measurement field from MeterReading (replaced by Value)"

---

### Phase 4: Documentation Update

#### Step 4.1: Update arch/work/issue-04-atomicity-migration-plan.md

**Action:**
- Mark all tasks as completed
- Add note: "Domain layer migration completed in observation-domain-layer-migration-plan.md"

**Commit:** "Mark issue #4 migration plan as complete"

---

#### Step 4.2: Update GitHub issue #4

**Action:**
```bash
gh issue comment 4 --body "Domain layer migration completed. All domain objects now use Observation/Aggregate types and ObservedAt field, matching the specs. Unbundling workaround removed."
gh issue close 4
```

**Commit:** N/A (external action)

---

## Design Principles Applied

### Principle #1: Design from First Principles
✓ Observation ≠ Aggregate in domain. Domain layer should mirror this truth, not require conversion.

### Principle #2: Avoid If/Else Blocks
✓ No conditionals checking "is this an observation or aggregate?" - types enforce it.

### Principle #3: Don't Make Decisions Twice
✓ Specs use Observation/Aggregate. Domain uses same. No duplicate decision about what types to use.

### Principle #10: Migrate Incrementally
✓ Add, Migrate, Remove pattern. Every commit compiles and passes tests.

### Principle #11: Fix Root Causes, Not Symptoms
✓ Unbundling was a symptom. Root cause: domain layer not aligned with specs. Fix: align domain layer.

---

## Safety Checks (Before Each Commit)

1. **Full test suite:** `go test ./...` (not partial)
2. **No compilation errors:** All packages compile
3. **Grep verification:** For removals, verify zero callers first
4. **Design question:** "Am I perpetuating deprecated patterns?"

---

## Red Team Analysis

**Risk:** Large migration touching multiple files
**Mitigation:** Add, Migrate, Remove pattern keeps each commit small and safe

**Risk:** Tests might be using old fields
**Mitigation:** Compiler catches field removals. Tests will fail if not migrated.

**Risk:** Aggregation logic complexity
**Mitigation:** Aggregation already works on arrays in specs. Domain just needs to match.

**Risk:** TimeWindow validation in constructors
**Mitigation:** NewTimeWindow validates start < end. NewInstantWindow enforces start == end.

---

## Estimated Commit Count

- Phase 1: 7 commits (add new types)
- Phase 2: 4 commits (migrate consumers)
- Phase 3: 5 commits (remove old types)
- Phase 4: 1 commit (documentation)

**Total: ~17 commits**

Each commit is small (10-100 lines), focused, and independently verifiable.

---

## Success Criteria

- ✅ All domain objects use `Observation` and `Aggregate` types
- ✅ All domain objects use `ObservedAt` (no `RecordedAt`)
- ✅ `Measurement` type completely removed
- ✅ `unbundleObservations()` function removed
- ✅ All tests pass: `go test ./...`
- ✅ Specs and domain layer aligned (no conversion ceremony)

---

## Next Actions

1. Execute Phase 1, Step 1.1
2. Run safety checks after each commit
3. Proceed sequentially through all phases

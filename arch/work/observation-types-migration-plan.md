# Migration Plan: Observation Types Separation

**Status:** In Progress
**Date:** 2026-01-28
**Issue:** #1 - Separate Observation and Aggregation types

---

## Discovery Phase

### Current Usage Analysis

**MeasurementSpec:**
- Defined: `specs/measurement.go`
- Used in: `MeterRecordSpec.Measurement`, `MeterReadingSpec.Measurement`
- Internal usage: `internal/metering.go`, `internal/aggregation.go`
- Test usage: `internal/meterreading_test.go`
- **Total references: 11**

**RecordedAt field:**
- Defined: `MeterRecordSpec.RecordedAt`
- Widely used in internal domain objects and aggregation logic
- Needs rename to `ObservedAt`

### Pattern Identification

Current pattern: Single type (`MeasurementSpec`) used for both:
1. Raw observations in `MeterRecord`
2. Aggregated results in `MeterReading`

Target pattern: Separate types:
1. `ObservationSpec` (with Window) for `MeterRecord`
2. `AggregateSpec` (without Window) for `MeterReading`

### Migration Strategy

Use Add-Migrate-Remove pattern:
- **Add:** New types coexist with old
- **Migrate:** Update consumers one by one
- **Remove:** Delete old type when zero callers

---

## Phase 1: Add (Parallel Implementation)

### Step 1.1: Create new observation types
**File:** `specs/observation.go`
**Action:** Add `ObservationSpec` and `AggregateSpec`
**Commit:** "Add ObservationSpec and AggregateSpec types"

✅ **Safety check:** `go test ./...` passes

### Step 1.2: Add new fields to MeterRecordSpec (parallel)
**File:** `specs/meterrecord.go`
**Action:**
- Add `Observation ObservationSpec` field
- Add `ObservedAt time.Time` field
- Keep existing `Measurement` and `RecordedAt` fields
**Commit:** "Add Observation and ObservedAt fields to MeterRecordSpec"

✅ **Safety check:** `go test ./...` passes

### Step 1.3: Add new field to MeterReadingSpec (parallel)
**File:** `specs/meterreading.go`
**Action:**
- Add `Value AggregateSpec` field
- Keep existing `Measurement` field
**Commit:** "Add Value field to MeterReadingSpec"

✅ **Safety check:** `go test ./...` passes

---

## Phase 2: Migrate (Update Consumers)

### Step 2.1: Update Meter() to populate new fields
**File:** `internal/metering.go`
**Action:** Update `Meter()` to:
- Populate both `Observation` and `Measurement` (dual write)
- Populate both `ObservedAt` and `RecordedAt` (dual write)
- Set Window to `[T, T]` for instant observations
**Commit:** "Migrate Meter() to populate Observation fields"

✅ **Safety check:** `go test ./...` passes

### Step 2.2: Update Aggregate() to read from new fields
**File:** `internal/aggregation.go`
**Action:** Update `Aggregate()` to:
- Read from `Observation` (with fallback to `Measurement`)
- Write to both `Value` and `Measurement` (dual write)
**Commit:** "Migrate Aggregate() to use Observation fields"

✅ **Safety check:** `go test ./...` passes

### Step 2.3: Update internal domain objects
**File:** `internal/meterrecord.go`
**Action:** Update domain object to support both old and new fields
**Commit:** "Update MeterRecord domain object for new fields"

✅ **Safety check:** `go test ./...` passes

### Step 2.4: Update tests to use new fields
**Files:** `internal/meterreading_test.go`, others
**Action:** Update test assertions to check new fields
**Commit:** "Update tests to verify new Observation fields"

✅ **Safety check:** `go test ./...` passes

---

## Phase 3: Remove (Cleanup)

### Step 3.1: Verify zero usage of old fields
**Action:**
```bash
grep -r "\.Measurement" --include="*.go"
grep -r "\.RecordedAt" --include="*.go"
```
**Expected:** Only field definitions remain, no consumers

### Step 3.2: Remove old fields from specs
**Files:** `specs/meterrecord.go`, `specs/meterreading.go`
**Action:**
- Remove `Measurement` field from `MeterRecordSpec`
- Remove `RecordedAt` field from `MeterRecordSpec`
- Remove `Measurement` field from `MeterReadingSpec`
**Commit:** "Remove deprecated Measurement and RecordedAt fields"

✅ **Safety check:** `go test ./...` passes

### Step 3.3: Delete MeasurementSpec type
**File:** `specs/measurement.go`
**Action:** Delete entire file
**Commit:** "Remove MeasurementSpec type (zero callers)"

✅ **Safety check:** `go test ./...` passes

---

## Safety Checks (Every Commit)

1. Run full test suite: `go test ./...`
2. Verify no compiler errors
3. Check for unexpected test failures

## Rollback Strategy

Each commit is independently revertible. If issues arise:
- Revert the most recent commit
- Fix the issue
- Re-apply with fixes

---

## Current Progress

- [x] Discovery phase complete
- [x] Migration plan written
- [ ] Phase 1: Add (in progress)
- [ ] Phase 2: Migrate
- [ ] Phase 3: Remove

# Issue #4: MeterRecord Atomicity Migration Plan

**Date:** 2026-01-28
**Issue:** [#4 Bundle multiple observations in single MeterRecord for atomicity](https://github.com/chrisconley/metering-spec/issues/4)
**Strategy:** Planned Refactoring (Add, Migrate, Remove)

---

## Executive Summary

Refactor MeterRecord to contain multiple observations instead of creating separate MeterRecord objects for each measurement from a single event. This ensures atomic handling of related observations and aligns with the ObservationSpec design.

**Key changes:**
- `MeterRecordSpec.Observation` (singular) → `Observations` (plural array)
- One event produces ONE MeterRecord with multiple ObservationSpec entries
- Natural atomicity: single record save = all observations persisted

---

## Prerequisites

**BLOCKING ISSUE:** The observation-temporal-context migration (issue #1) is incomplete. Test files still reference deprecated fields that no longer exist in the specs:
- `Measurement` → should be `Observation`
- `MeasurementSpec` → should be `ObservationSpec`
- `RecordedAt` → should be `ObservedAt`

**Decision:** Fix issue #1 test failures FIRST, then proceed with issue #4.

**Rationale:**
- Issue #4 builds on issue #1's foundation (ObservationSpec)
- Cannot safely migrate to `Observations []ObservationSpec` while tests are broken
- Clean slate ensures we can validate each step

---

## Phase 0: Complete Issue #1 Migration

### Step 0.1: Fix test files to use new field names

**Files with errors (from diagnostics):**
- `internal/meterreading_test.go`
- `internal/meterrecord_test.go`
- `benchmarks/sizing_calculator_test.go`

**Changes needed:**
```go
// OLD
MeterRecordSpec{
    RecordedAt: ...,
    Measurement: MeasurementSpec{...},
}

// NEW
MeterRecordSpec{
    ObservedAt: ...,
    Observation: ObservationSpec{...},
}
```

**Safety check:** Run `go test ./...` - must pass before proceeding to Phase 1.

**Commit:** "Fix test files to use ObservedAt and ObservationSpec"

---

## Phase 1: Add New Field (Backwards Compatible)

### Step 1.1: Add `Observations` array alongside existing `Observation`

**File:** `specs/meterrecord.go`

```go
type MeterRecordSpec struct {
    ID            string
    WorkspaceID   string
    UniverseID    string
    Subject       string
    ObservedAt    time.Time

    // NEW: Multiple observations from same event (preferred)
    Observations  []ObservationSpec     `json:"observations,omitempty"`

    // OLD: Single observation (deprecated, for backwards compatibility)
    Observation   ObservationSpec       `json:"observation,omitempty"`

    Dimensions    map[string]string
    SourceEventID string
    MeteredAt     time.Time
}
```

**Documentation update:**
- Add deprecation notice to `Observation` field
- Document that `Observations` is preferred
- Note: during migration, both fields may be populated

**Safety check:**
- Run `go build ./...` - must compile
- No behavioral changes yet

**Commit:** "Add Observations array to MeterRecordSpec (backwards compatible)"

---

## Phase 2: Update Producer (Meter Function)

### Step 2.1: Update Meter() to populate both fields

**File:** `internal/meter.go` (or wherever Meter() lives)

**Current behavior:** Returns multiple MeterRecord objects (one per measurement)

**New behavior:** Returns ONE MeterRecord with multiple observations

```go
// OLD approach (pseudo-code)
func Meter(payload EventPayloadSpec, config MeteringConfig) ([]MeterRecordSpec, error) {
    var records []MeterRecordSpec
    for _, measurement := range extractedMeasurements {
        record := MeterRecordSpec{
            ID: hash(payload.ID + measurement.Unit), // Unit-specific ID
            Observation: measurement,
            // ... other fields
        }
        records = append(records, record)
    }
    return records, nil
}

// NEW approach (populate both fields during migration)
func Meter(payload EventPayloadSpec, config MeteringConfig) ([]MeterRecordSpec, error) {
    observations := extractObservations(payload, config)

    if len(observations) == 0 {
        return nil, nil
    }

    record := MeterRecordSpec{
        ID: hash(payload.ID), // Just event ID, no unit suffix
        Observations: observations,  // NEW: all observations
        Observation: observations[0], // OLD: first one for backwards compat
        // ... other fields
    }

    return []MeterRecordSpec{record}, nil // Returns ONE record
}
```

**Key changes:**
- ID generation: remove unit suffix (just use event ID)
- Return single record instead of multiple
- Populate BOTH `Observations` (new) and `Observation` (old)

**Safety check:**
- Run `go test ./...` - all tests must pass
- Verify aggregation still works (reads from `Observation` field)

**Commit:** "Update Meter() to bundle observations in single record"

---

## Phase 3: Migrate Consumers (Aggregation)

### Step 3.1: Update aggregation to read from Observations array

**File:** `internal/aggregate.go` (or similar)

**Current pattern:**
```go
for _, record := range records {
    // Process single observation
    unit := record.Observation.Unit
    accumulators[unit].Add(record.Observation, record.ObservedAt)
}
```

**New pattern:**
```go
for _, record := range records {
    // Try new field first, fall back to old
    observations := record.Observations
    if len(observations) == 0 {
        // Backwards compatibility: read from old field
        observations = []ObservationSpec{record.Observation}
    }

    for _, observation := range observations {
        unit := observation.Unit
        accumulators[unit].Add(observation, record.ObservedAt)
    }
}
```

**Safety check:**
- Run `go test ./...` - must pass
- Verify works with both old and new record formats

**Commit:** "Update aggregation to use Observations array"

### Step 3.2: Migrate other consumers one-by-one

**Discovery needed:** Grep for usage of `record.Observation` to find all consumers

```bash
grep -r "record\.Observation" --include="*.go" | grep -v "Observations"
```

**For each consumer:**
1. Update to read from `Observations` array
2. Add backwards compatibility fallback
3. Run `go test ./...`
4. Commit: "Migrate {package} to use Observations array"

**Files likely affected:**
- Publishing/event bus handlers
- Repository/persistence layer
- Example code
- Documentation examples

---

## Phase 4: Update Tests

### Step 4.1: Update test assertions to use Observations

**Pattern:**
```go
// OLD
assert.Equal(t, "500", record.Observation.Quantity)

// NEW
assert.Len(t, record.Observations, 1)
assert.Equal(t, "500", record.Observations[0].Quantity)
```

**Files to update:**
- All `*_test.go` files that construct or assert on MeterRecordSpec

**Safety check:**
- Run `go test ./...` after each file
- Each commit compiles and passes tests

**Commit per file:** "Update {filename} tests to use Observations"

---

## Phase 5: Remove Deprecated Field

### Step 5.1: Verify zero usage of old field

```bash
# Should return ZERO results (except in this migration doc)
grep -r "\.Observation[^s]" --include="*.go"
```

**If any results:** Go back and migrate those files first.

### Step 5.2: Remove `Observation` field from MeterRecordSpec

**File:** `specs/meterrecord.go`

```go
type MeterRecordSpec struct {
    ID            string
    WorkspaceID   string
    UniverseID    string
    Subject       string
    ObservedAt    time.Time
    Observations  []ObservationSpec     `json:"observations"` // Remove omitempty
    Dimensions    map[string]string
    SourceEventID string
    MeteredAt     time.Time
    // Observation field REMOVED
}
```

**Safety check:**
- Run `go build ./...` - must compile
- If compilation fails, we missed a usage (grep again and fix)
- Run `go test ./...` - must pass

**Commit:** "Remove deprecated Observation field from MeterRecordSpec"

---

## Phase 6: Update Documentation

### Step 6.1: Update inline documentation

**File:** `specs/meterrecord.go`

Remove text: "One event payload can produce multiple meter records when the metering configuration extracts multiple measurements from the same event."

Add text: "One event payload produces one meter record containing all observations extracted by the metering configuration. Multiple observations from the same event are bundled together, ensuring atomic persistence."

### Step 6.2: Update examples

**Files:**
- `docs/examples/basic-api-metering.md`
- `README.md`
- Any other documentation with MeterRecord JSON examples

**Pattern:**
```json
{
  "id": "rec_xyz789",
  "observations": [
    {
      "quantity": "100",
      "unit": "input_tokens",
      "window": {...}
    },
    {
      "quantity": "50",
      "unit": "output_tokens",
      "window": {...}
    }
  ]
}
```

**Commit:** "Update documentation for bundled observations"

---

## Safety Checks (Apply at Every Step)

Before each commit:

1. **Compilation check:** `go build ./...`
2. **Full test suite:** `go test ./...` (not partial!)
3. **Grep verification:** Confirm expected usage patterns
4. **Red team question:** "What breaks if I deploy this commit right now?"

---

## Root Cause Analysis Checklist

When tests fail during migration, ask in this order:

1. **Remove the need** - Can we delete the failing code? (deprecated test?)
2. **Fix root cause** - WHY does this error exist? (wrong abstraction?)
3. **Fix symptom** - Make it compile/pass (update to new field)
4. ❌ **Never create new code** - Don't add files to satisfy imports

---

## Impact Analysis

### Breaking Changes

**Spec changes:**
- `MeterRecordSpec.Observation` field removed (after migration)
- `MeterRecordSpec.Observations` array added
- ID generation: no longer includes unit suffix
- `Meter()` returns 1 record per event (not N records)

**Behavioral changes:**
- Atomicity: all observations persist together or none persist
- Publishing: single event per record (not N events)
- Storage: dimensions stored once (not duplicated per observation)

### Non-Breaking (Preserved)

- Idempotency: deterministic IDs from event ID
- Aggregation logic: still groups by unit
- MeterReading structure: unchanged (still has single `Value`)
- Time-weighted averages: still work with temporal windows

---

## Open Questions

### Q1: What if event has zero measurements?

**Current answer:** Return empty array `[]MeterRecordSpec{}`

**Validation needed:** Confirm this is correct behavior vs returning error.

### Q2: Should we enforce minimum observation count?

**Option A:** Allow empty `Observations` array (validation at persistence)
**Option B:** Require at least one observation (validation at construction)

**Recommendation:** Start with Option A (YAGNI), add validation if needed.

### Q3: Do different observations need different dimensions?

**Current design:** Shared dimensions for all observations in a record

**Example scenario:**
```json
{
  "cpu_ms": 1000,
  "cpu_region": "us-east",
  "memory_mb": 2048,
  "memory_region": "us-west"
}
```

**Current approach:** Both go into shared dimensions
```go
Observations: [
    {Quantity: "1000", Unit: "cpu_ms"},
    {Quantity: "2048", Unit: "memory_mb"}
]
Dimensions: {
    "cpu_region": "us-east",
    "memory_region": "us-west"
}
```

**Decision:** Start with shared dimensions. Can refactor later if validated need emerges.

---

## Success Criteria

Migration is complete when:

- ✅ All tests pass (`go test ./...`)
- ✅ All code compiles (`go build ./...`)
- ✅ Zero usage of deprecated `Observation` field
- ✅ Documentation updated with new examples
- ✅ One event produces one MeterRecord (not multiple)
- ✅ Natural atomicity: single save operation per event

---

## References

- **Issue:** [#4](https://github.com/chrisconley/metering-spec/issues/4)
- **Design analysis:** `design/references/meterrecord-atomicity-analysis.md`
- **ADR:** `design/observation-temporal-context.md`
- **Pattern guide:** `/Users/chris/.claude/skills/refactoring-sequencer/add-migrate-remove-guide.md`
- **Design principles:** Applied from Chris's design principles (first principles, avoid trap doors, users first)

---

## Next Steps

1. **Confirm prerequisites:** Ensure issue #1 migration is complete
2. **Discovery phase:** Grep for all `record.Observation` usage
3. **Get user sign-off:** Review this plan before implementation
4. **Execute Phase 1:** Add new field (backwards compatible)


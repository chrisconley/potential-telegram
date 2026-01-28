# Issue #1: Separate Observation and Aggregation types with temporal context

**GitHub Issue:** https://github.com/chrisconley/potential-telegram/issues/1
**Autonomy Rating:** 85/100
**Confidence Level:** High
**Status:** Ready for implementation
**Dependencies:** None

---

## Summary

Implement the accepted ADR to separate `ObservationSpec` (raw measurements from events) from `AggregatedValueSpec` (computed results), and add temporal context (Window field) to all observations.

**Current Problem:**
- `MeasurementSpec` is used for both raw observations (MeterRecord) and computed aggregations (MeterReading)
- This conflates semantically different domain concepts
- Temporal information lost for downstream use cases like proration

**Proposed Solution:**
- Create `ObservationSpec` with Window field for raw measurements
- Create `AggregatedValueSpec` without Window for computed results
- Use Window convention: `[T, T]` for instant observations, `[T1, T2]` for time-spanning

---

## Detailed Analysis

### What's Required

#### 1. Create New Types (specs/observation.go)

```go
// ObservationSpec - Point-in-time or time-spanning observation from events
type ObservationSpec struct {
    Quantity string         `json:"quantity"`
    Unit     string         `json:"unit"`
    // Temporal extent of this observation
    // For instant: Start == End (observed at that moment)
    // For time-spanning: Start < End (observed over window)
    Window   TimeWindowSpec `json:"window"`
}

// AggregatedValueSpec - Aggregated value computed from observations
type AggregatedValueSpec struct {
    Quantity string `json:"quantity"`
    Unit     string `json:"unit"`
    // No window - temporal context is in parent MeterReading.Window
}
```

#### 2. Update MeterRecordSpec

```go
type MeterRecordSpec struct {
    ID           string          `json:"id"`
    WorkspaceID  string          `json:"workspaceID"`
    UniverseID   string          `json:"universeID"`
    Subject      string          `json:"subject"`

    // Renamed from RecordedAt for clarity
    ObservedAt   time.Time       `json:"observedAt"`

    // Changed from Measurement to Observation
    Observation  ObservationSpec `json:"observation"`

    Dimensions   map[string]string `json:"dimensions,omitempty"`
    SourceEventID string          `json:"sourceEventID"`
    MeteredAt    time.Time       `json:"meteredAt"`
}
```

#### 3. Update MeterReadingSpec

```go
type MeterReadingSpec struct {
    ID           string              `json:"id"`
    WorkspaceID  string              `json:"workspaceID"`
    UniverseID   string              `json:"universeID"`
    Subject      string              `json:"subject"`
    Window       TimeWindowSpec      `json:"window"`

    // Changed from Measurement to Value
    Value        AggregatedValueSpec `json:"value"`

    Aggregation  string              `json:"aggregation"`
    RecordCount  int                 `json:"recordCount"`
    CreatedAt    time.Time           `json:"createdAt"`
    MaxMeteredAt time.Time           `json:"maxMeteredAt"`
}
```

#### 4. Add Helper Constructors

```go
// NewInstantObservation creates an observation at a point in time
func NewInstantObservation(quantity, unit string, instant time.Time) ObservationSpec {
    return ObservationSpec{
        Quantity: quantity,
        Unit:     unit,
        Window: TimeWindowSpec{
            Start: instant,
            End:   instant,
        },
    }
}

// NewSpanObservation creates an observation over a time window
func NewSpanObservation(quantity, unit string, start, end time.Time) (ObservationSpec, error) {
    if !end.After(start) {
        return ObservationSpec{}, fmt.Errorf("end must be after start")
    }
    return ObservationSpec{
        Quantity: quantity,
        Unit:     unit,
        Window: TimeWindowSpec{
            Start: start,
            End:   end,
        },
    }, nil
}
```

#### 5. Migration Strategy (Add-Migrate-Remove)

**Phase 1: Add new types alongside old**
- Add ObservationSpec, AggregatedValueSpec
- Keep MeasurementSpec (mark deprecated)
- Add new fields alongside old in MeterRecord/MeterReading

**Phase 2: Migrate consumers**
- Update Meter() to populate Observation field
- Update Aggregate() to use ObservationSpec
- Update all readers to use new fields

**Phase 3: Remove old types**
- Delete MeasurementSpec
- Remove deprecated fields
- Compiler catches any missed usages

---

## Implementation Plan

### Step 1: Create new type files
- [ ] Create `specs/observation.go`
- [ ] Define ObservationSpec with Window
- [ ] Define AggregatedValueSpec without Window
- [ ] Add helper constructors
- [ ] Add validation methods

### Step 2: Update MeterRecordSpec
- [ ] Add `Observation ObservationSpec` field
- [ ] Add `ObservedAt time.Time` field
- [ ] Mark `Measurement` and `RecordedAt` as deprecated (comments)
- [ ] Update JSON tags with omitempty

### Step 3: Update MeterReadingSpec
- [ ] Add `Value AggregatedValueSpec` field
- [ ] Mark `Measurement` as deprecated
- [ ] Update JSON tags

### Step 4: Update implementations
- [ ] Update Meter() to populate Observation + ObservedAt
- [ ] Update Aggregate() to read from Observation
- [ ] Update Aggregate() to write to Value
- [ ] Maintain backward compatibility (populate both old and new)

### Step 5: Update tests
- [ ] Add tests for instant observations
- [ ] Add tests for time-spanning observations
- [ ] Add tests for window validation
- [ ] Add tests for helper constructors

### Step 6: Update documentation
- [ ] Update examples in docs
- [ ] Update README with new types
- [ ] Add migration guide for users

### Step 7: Remove deprecated fields
- [ ] Delete Measurement from MeterRecordSpec
- [ ] Delete RecordedAt from MeterRecordSpec
- [ ] Delete Measurement from MeterReadingSpec
- [ ] Delete MeasurementSpec type

---

## Autonomy Assessment: 85/100

### Why Not Higher (15 points lost)

1. **Integration testing requirements unclear** (-5 points)
   - What level of test coverage is expected?
   - Should there be integration tests with full pipeline?
   - Migration validation tests not specified

2. **Temporal convention needs validation** (-5 points)
   - `[T, T]` for instants is documented but may have edge cases
   - Need to validate with real data scenarios
   - Helper constructors may need adjustment

3. **Unforeseen integration points** (-5 points)
   - May discover consumers in codebase not documented
   - External integrations might exist
   - Migration coordination with other services

### Why This High (85 points)

1. **Extremely detailed ADR** (+30 points)
   - 507 lines of specification
   - Examples for every scenario
   - Rationale grounded in design principles
   - Consequences clearly documented

2. **Clear migration path** (+25 points)
   - Phased approach (add, migrate, remove)
   - Backward compatibility strategy
   - Helper constructors prevent errors

3. **Type signatures fully specified** (+15 points)
   - Exact struct definitions provided
   - Field types and JSON tags clear
   - No ambiguity in API

4. **Related docs provide context** (+10 points)
   - time-spanning-events.md (442 lines)
   - aggregation-types.md (320 lines)
   - Full domain context available

5. **Design principles applied** (+5 points)
   - Principle #1: Design from first principles (observations ≠ aggregations)
   - Principle #2: Avoid if/else blocks (type system enforces)
   - Principle #11: Fix root causes (not symptoms)

---

## Risks & Unknowns

### Medium Risks
1. **Breaking changes despite migration strategy**
   - Users may have direct struct field access
   - JSON serialization changes could affect external systems
   - Mitigation: Support both old and new for transition period

2. **Window convention confusion**
   - `[T, T]` for instants is a mathematical quirk
   - Developers might forget to set both Start and End equal
   - Mitigation: Helper constructors, validation, clear docs

### Low Risks
1. **Performance overhead**
   - Window adds 48 bytes to each observation (2 × time.Time)
   - Likely negligible given aggregation reduces volume 100:1
   - Can validate with benchmarks (Issue #3)

---

## Related Documents

### Design Documents
- **Primary:** `design/observation-temporal-context.md` (507 lines, Accepted ADR)
- **Supporting:** `design/references/time-spanning-events.md` (442 lines)
- **Related:** `design/aggregation-types.md` (320 lines)

### Current Implementation
- `specs/measurement.go` - Current MeasurementSpec (26 lines)
- `specs/meterrecord.go` - MeterRecordSpec using Measurement (79 lines)
- `specs/meterreading.go` - MeterReadingSpec using Measurement (105 lines)

### Examples from ADR
- Instant gauge observation (seats at 9:47am)
- Time-spanning observation (8 compute-hours from Jan 31 8pm to Feb 1 4am)
- Aggregated value (average 12.32 seats during February)

---

## Estimated Effort

**Total:** 3-5 days for complete implementation

- **Type creation:** 0.5 day (specs, helpers, validation)
- **Spec updates:** 0.5 day (add new fields, deprecate old)
- **Implementation migration:** 1-2 days (Meter, Aggregate functions)
- **Testing:** 1 day (unit tests, integration tests)
- **Documentation:** 0.5 day (examples, migration guide)
- **Cleanup:** 0.5 day (remove deprecated fields)

**Critical path:** Implementation migration (most complex, touches core logic)

---

## Success Criteria

1. ✅ ObservationSpec and AggregatedValueSpec types exist
2. ✅ MeterRecordSpec uses ObservationSpec with Window
3. ✅ MeterReadingSpec uses AggregatedValueSpec
4. ✅ Helper constructors prevent invalid Window states
5. ✅ Meter() populates Observation with correct temporal context
6. ✅ Aggregate() reads Observation, writes Value
7. ✅ Tests cover instant and time-spanning observations
8. ✅ Documentation updated with migration guide
9. ✅ Backward compatibility maintained during transition
10. ✅ No MeasurementSpec references remain in codebase

---

## Questions for User (Optional)

1. **Test coverage:** What level of test coverage is expected? (unit only, or integration?)
2. **Migration timeline:** How long should backward compatibility be maintained?
3. **Breaking changes:** Is a major version bump acceptable, or must maintain full compatibility?
4. **Validation strictness:** Should Window validation be strict (error) or lenient (warning)?

**Note:** These questions are optional - implementation can proceed with reasonable defaults, but answers would increase confidence.

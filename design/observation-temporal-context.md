# ADR: Observation Temporal Context and Type Separation

**Status:** Accepted
**Date:** 2026-01-21
**Context:** Distinguishing observations from aggregations, and determining temporal context requirements

---

## Summary

Observations and aggregated values are fundamentally different domain concepts that should be represented as separate types. All observations require temporal context: instant observations occur at a point in time, while time-spanning observations occur over a window. The `Window` field in observations uses the convention `[T, T]` for instant observations and `[T1, T2]` where T1 < T2 for time-spanning observations.

---

## Context & Problem

### The Challenge

The metering system uses `MeasurementSpec` for both:
1. **MeterRecord observations** - raw measurements from events ("customer had 15 seats at 9:47am")
2. **MeterReading aggregations** - computed results ("customer averaged 12.32 seats during February")

Two design questions emerged:

**Question 1:** Should observations and aggregations use the same type?
- Both have `{Quantity, Unit}` structure
- But semantically different: raw data vs computed results
- Operations differ: can aggregate observations, but can you re-aggregate aggregations?

**Question 2:** Is temporal context essential to observations?
- Analogy: "500" without "tokens" is meaningless
- Is "15 seats" without temporal context equally meaningless?
- What about time-spanning observations like "8 compute-hours from Jan 31 8pm to Feb 1 4am"?

### Key Insight from Time-Spanning Events

From `time-spanning-events.md`, events can represent:
- **Instant observations:** "At T, customer had 15 seats" (point-in-time gauge)
- **Discrete events:** "At T, API call consumed 500 tokens" (instantaneous)
- **Time-spanning activity:** "From T1 to T2, compute ran consuming 8 hours" (activity over period)

Current approach (Pattern 1) converts to composite units:
```json
{
  "timestamp": "2026-02-01T04:00:00Z",
  "properties": {"duration_seconds": "28800"}
}
```

Becomes:
```go
MeterRecord {
    RecordedAt: 2026-02-01T04:00:00Z,  // End time
    Measurement: {Quantity: "8", Unit: "compute-hours"}
}
```

**Problem:** Original time window `[Jan 31 8pm, Feb 1 4am]` is lost. Cannot be reconstructed for downstream use cases like proration across billing periods.

---

## Decision

### 1. Separate Observation and Aggregation Types

```go
// Point-in-time or time-spanning observation from events
type ObservationSpec struct {
    Quantity string
    Unit     string
    // Temporal extent of this observation
    // For instant observations: Start == End (observed at that moment)
    // For time-spanning observations: Start < End (observed over window)
    Window TimeWindowSpec
}

// Aggregated value computed from observations
type AggregatedValueSpec struct {
    Quantity string
    Unit     string
    // No window - temporal context is in MeterReading.Window
}
```

### 2. Observations Always Have Temporal Context (Window)

All observations include a `Window` field representing their temporal extent:

**Instant observation (gauge):**
```go
MeterRecord {
    RecordedAt: 2026-02-15T09:47:00Z,  // When reported
    Observation: {
        Quantity: "15",
        Unit: "seats",
        Window: {
            Start: 2026-02-15T09:47:00Z,
            End:   2026-02-15T09:47:00Z  // Start == End
        }
    }
}
```
**Meaning:** "At 9:47am, customer had 15 seats"

**Time-spanning observation:**
```go
MeterRecord {
    RecordedAt: 2026-02-01T04:00:00Z,  // When reported
    Observation: {
        Quantity: "8",
        Unit: "compute-hours",
        Window: {
            Start: 2026-01-31T20:00:00Z,
            End:   2026-02-01T04:00:00Z
        }
    }
}
```
**Meaning:** "From Jan 31 8pm to Feb 1 4am, compute consumed 8 hours"

### 3. Rename RecordedAt → ObservedAt for Clarity

```go
type MeterRecordSpec struct {
    // When observation was taken (instant) or finalized (span)
    // Often equals Observation.Window.End for time-spanning observations
    ObservedAt time.Time `json:"observedAt"`

    Observation ObservationSpec `json:"observation"`

    // System timestamp: when metering process created this record
    MeteredAt time.Time `json:"meteredAt"`
}
```

**Separation maintained:**
- `ObservedAt` = business time (when observed/reported)
- `MeteredAt` = system time (when processed)
- `Observation.Window` = temporal extent of the observation itself

### 4. Time Context in Parent Types

```go
type MeterRecordSpec struct {
    ObservedAt  time.Time        // When observed/reported
    Observation ObservationSpec  // May have temporal window
    MeteredAt   time.Time        // When processed
}

type MeterReadingSpec struct {
    Window TimeWindowSpec       // Aggregation period (always present)
    Value  AggregatedValueSpec  // Result (no window - it's in parent)
    // ...
}
```

**Rationale:** Window belongs in different places depending on context:
- **MeterRecord:** Window is part of the observation (what was observed)
- **MeterReading:** Window is the aggregation period (over what period we computed)

---

## Rationale

### Principle #1: Design from First Principles

**What are observations in the domain?**
- Raw measurements from events
- "At instant T, customer had 15 seats" (snapshot)
- "From T1 to T2, activity consumed 8 hours" (activity over period)

**What are aggregations?**
- Computed results from observations
- "During February, customer averaged 12.32 seats" (computed average)
- "During February, customer consumed 12,500 tokens" (computed sum)

These are fundamentally different domain concepts:
- **Observations:** Input data, raw measurements
- **Aggregations:** Output data, computed results

Different operations are valid:
- Can aggregate observations → readings
- Cannot meaningfully re-aggregate readings without knowing windows + weights
- Type system should prevent invalid operations

### Principle #2: Avoid If/Else Blocks

**With separate types, no conditionals:**
```go
// Type system enforces valid operations
func Aggregate(
    observations []ObservationSpec,  // Can ONLY take observations
    config AggregateConfig,
) (AggregatedValueSpec, error)      // Returns aggregated value

// Future: hierarchical aggregation (if needed)
func ReAggregate(
    values []AggregatedValueSpec,    // Can ONLY take aggregated values
    config ReAggregateConfig,
) (AggregatedValueSpec, error)
```

**Without separate types, runtime checks everywhere:**
```go
func processValue(m MeasurementSpec) {
    if isObservation(m) {
        // Can aggregate
    } else if isAggregation(m) {
        // Cannot re-aggregate (or need special handling)
    }
}
```

### Principle #3: Don't Decide Twice

**Current approach (Pattern 1):**
1. **Ingestion:** "This usage ends at Feb 1 4am" → set RecordedAt
2. **Downstream consumer:** "Which billing periods does this span?" → cannot answer without window

**With Window in Observation:**
1. **Ingestion:** Capture full temporal context once
2. **Downstream:** Use Window directly for any temporal reasoning

Decision made once, information preserved for all consumers.

### Principle #6: Client Responsibility (Clear Contracts)

**With separate types:**
```go
// Crystal clear what each is
observation := ObservationSpec{...}    // Raw measurement
aggregated := AggregatedValueSpec{...} // Computed result

// Type system prevents confusion
Aggregate([]ObservationSpec{...})      // ✓ Correct
Aggregate([]AggregatedValueSpec{...})  // ✗ Compiler error
```

**With single type:**
```go
// Ambiguous - is this raw or computed?
measurement := MeasurementSpec{...}

// Runtime validation needed
Aggregate(measurements) // Are these all observations? Or mixed?
```

### Principle #11: Fix Root Causes, Not Symptoms

**Root cause:** Conflating two semantically different concepts (observations vs aggregations)

**Symptom:** Would need runtime validation to prevent invalid operations

**Fix:** Model domain truth - observations ≠ aggregations. Use type system to enforce.

### Temporal Context is Essential

**Units analogy extended:**
- "500" → meaningless (no unit)
- "500 tokens" → meaningful with unit
- **"8 compute-hours" → where in time?** Temporal context needed

For billing and analysis:
- **Instant observations:** "at T" (timestamp)
- **Time-spanning observations:** "from T1 to T2" (window)

Both require temporal context, just as both require units.

### Mathematical Convention: [T, T] for Instants

`TimeWindowSpec` uses half-open intervals `[Start, End)`:
- Standard for time ranges (no gaps/overlaps between adjacent periods)
- For spans: `[T1, T2)` where T1 < T2 works naturally
- For instants: `[T, T)` is technically empty, but we use it as convention

**Convention:** `Start == End` represents an instant at that moment.

This is an acceptable engineering trade-off:
- Simpler schema (one field type for all temporal extents)
- Helper constructors prevent errors
- Clear semantics via documentation

Alternative would require separate fields (instant timestamp vs span window), adding schema complexity for minimal gain.

---

## Consequences

### Positive

1. **Type safety:** Compiler prevents invalid operations (aggregating aggregations incorrectly)
2. **Self-documenting:** Code is clear about observations vs results
3. **Preserves information:** Original temporal windows available for all consumers
4. **Clear contracts:** Client knows exactly what to provide
5. **Enables future features:** Can add hierarchical aggregation if needed
6. **Downstream flexibility:** Consumers can prorate, filter, or analyze using full temporal context

### Negative

1. **Migration effort:** Need to rename `MeasurementSpec` → `ObservationSpec` in MeterRecord, create new `AggregatedValueSpec` for MeterReading
2. **Convention required:** `[T, T]` for instants is mathematical quirk (half-open interval technically empty)
3. **Extra field:** `Window` always present even when redundant with `ObservedAt` for instant observations

### Mitigations

1. **Migration:** Use add-migrate-remove pattern (Principle #10)
   - Add new types alongside old
   - Migrate usage one file at a time
   - Remove old type when nothing references it

2. **Convention clarity:** Provide type-safe constructors
   ```go
   func NewInstantObservation(quantity, unit string, instant time.Time) ObservationSpec
   func NewSpanObservation(quantity, unit string, start, end time.Time) (ObservationSpec, error)
   ```

3. **Field redundancy:** Accept as trade-off for uniform schema and clear semantics

---

## Design Principles Applied

### Principle #1: Design from First Principles
✓ Observations ≠ aggregations in the domain. Model truthfully.

### Principle #2: Avoid If/Else Blocks
✓ Type system enforces validity. No conditionals checking "what kind of measurement is this?"

### Principle #3: Don't Decide Twice
✓ Temporal context captured once at ingestion, used directly by all consumers.

### Principle #4: Avoid Trap Doors
✓ Starting with correct types prevents future breaking refactor when we need operations on values.

### Principle #6: Client Responsibility (Clear Contracts)
✓ Types make contracts crystal clear. Can't misuse a value.

### Principle #9: Users First (Engineer for Next 10)
✓ Prevents user errors. Compiler catches mistakes before runtime.
✓ Preserves information users need (temporal windows) for various use cases.

### Principle #11: Fix Root Causes, Not Symptoms
✓ Root issue is conflating concepts. Separating them fixes the design smell.

---

## Scope Clarification

**The metering spec's responsibility:**
- Capture observations faithfully with full temporal context
- Aggregate observations into readings according to configuration

**NOT the spec's responsibility:**
- Billing period assignment
- Proration across periods
- Splitting observations for billing purposes

Those are **downstream consumer concerns**. The spec provides the information (temporal windows); consumers decide how to use it.

---

## Migration Path

### Phase 1: Add New Types (Backwards Compatible)

```go
// Add alongside existing MeasurementSpec
type ObservationSpec struct {
    Quantity string
    Unit     string
    Window   TimeWindowSpec
}

type AggregatedValueSpec struct {
    Quantity string
    Unit     string
}

// MeasurementSpec remains (deprecated)
type MeasurementSpec struct {
    Quantity string
    Unit     string
}
```

### Phase 2: Migrate MeterRecord

```go
type MeterRecordSpec struct {
    ObservedAt  time.Time        // Renamed from RecordedAt
    Observation ObservationSpec  // New field (replaces Measurement)
    // Measurement MeasurementSpec // Deprecated
    MeteredAt   time.Time
}
```

### Phase 3: Migrate MeterReading

```go
type MeterReadingSpec struct {
    Window TimeWindowSpec
    Value  AggregatedValueSpec   // New field (replaces Measurement)
    // Measurement MeasurementSpec // Deprecated
    Aggregation string
    // ...
}
```

### Phase 4: Remove MeasurementSpec

Once all code migrated, remove deprecated `MeasurementSpec` type.

---

## Examples

### Instant Gauge Observation

```go
// Customer seat count at a moment
observation := ObservationSpec{
    Quantity: "15",
    Unit:     "seats",
    Window: TimeWindowSpec{
        Start: 2026-02-15T09:47:00Z,
        End:   2026-02-15T09:47:00Z,  // Instant: Start == End
    },
}

record := MeterRecordSpec{
    ObservedAt:  2026-02-15T09:47:00Z,
    Observation: observation,
    MeteredAt:   2026-02-15T10:00:00Z,
}
```

### Time-Spanning Observation

```go
// Compute session from Jan 31 8pm to Feb 1 4am
observation := ObservationSpec{
    Quantity: "8",
    Unit:     "compute-hours",
    Window: TimeWindowSpec{
        Start: 2026-01-31T20:00:00Z,
        End:   2026-02-01T04:00:00Z,  // Span: Start < End
    },
}

record := MeterRecordSpec{
    ObservedAt:  2026-02-01T04:00:00Z,  // When span ended/reported
    Observation: observation,
    MeteredAt:   2026-02-01T04:00:05Z,
}
```

### Aggregated Value

```go
// February aggregation result
reading := MeterReadingSpec{
    Window: TimeWindowSpec{
        Start: 2026-02-01T00:00:00Z,
        End:   2026-03-01T00:00:00Z,
    },
    Value: AggregatedValueSpec{
        Quantity: "12.32",  // Computed average
        Unit:     "seats",
    },
    Aggregation: "time-weighted-avg",
}
```

---

## Related Decisions

- **aggregation-types.md:** Counter vs gauge aggregation semantics (explicit names)
- **time-spanning-events.md:** Pattern 1 (duration in data) and Pattern 3 (start/end in data)
- **observability-vs-metering.md:** Time-weighted averages and accuracy requirements

---

## References

### Internal Documentation
- `metering-spec/docs/time-spanning-events.md` - Temporal patterns for events
- `metering-spec/docs/aggregation-types.md` - Counter vs gauge operations
- `metering-spec/docs/observability-vs-metering.md` - Industry patterns and differences
- `arch/reference/chris-design-principles.md` - Design principles applied

### Discussion
- Design discussion: 2026-01-21 "Is an observation an observation without knowing over what time window the observation was observing?"

---

## Conclusion

Observations and aggregations are fundamentally different domain concepts that should be represented as separate types (`ObservationSpec` vs `AggregatedValueSpec`). All observations require temporal context represented as a `Window`, using the convention `[T, T]` for instant observations and `[T1, T2]` for time-spanning observations. This design:

- Provides type safety and clear contracts
- Preserves full temporal information for downstream consumers
- Avoids conditionals by modeling domain truth
- Enables future capabilities without breaking changes

The metering spec's responsibility is to faithfully capture observations with their complete temporal context. How consumers use that information (billing periods, proration, etc.) is outside the spec's scope.

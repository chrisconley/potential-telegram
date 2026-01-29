# Ubiquitous Language: Metering Domain Terminology

**Date:** 2026-01-29
**Status:** Finalized
**Purpose:** Define all domain terms explicitly following DDD principles
**Last Updated:** After MeasurementExtraction → ObservationExtraction and AggregateValue → ComputedValue refactorings

---

## Purpose

This document establishes the **ubiquitous language** for the metering system. All team members, code, documentation, and conversations should use these terms consistently with these definitions.

Before making any naming decisions (like MeasurementExtraction → ObservationExtraction), we must first establish what each term means in our domain and why we choose one over another.

---

## Core Principle: From Observability Domain Research

From `design/references/observability-vs-metering.md`:

> **Observability systems (Prometheus, OpenTelemetry) classify metrics into distinct types:**
> - **Counter:** Monotonically increasing value (only goes up)
> - **Gauge:** Point-in-time value that can go up or down
>
> **Key insight:** The *type* determines what *operations* are valid.

This principle guides our terminology: **names should reveal semantics and valid operations**.

---

## 1. Raw Data: What Events Contain

### Event / EventPayload
**Definition:** Raw business activity data arriving at the system boundary.

**Examples:**
- "API call completed with 500 tokens"
- "Compute session ended, ran 8 hours"
- "Customer seat count changed to 15"

**Structure:**
```go
EventPayload {
    ID, WorkspaceID, UniverseID
    Type, Subject, Time
    Properties map[string]string  // Contains usage data
}
```

**Key characteristics:**
- Immutable once received
- Contains usage data in `Properties`
- No semantic meaning until metered

**Related terms:** "Event" is shorthand for EventPayload

---

## 2. Extracted Values: What We Pull From Events

### Observation
**Definition:** A numeric value extracted from an event with temporal context, representing *what was observed*.

**Meaning:** "We observed this quantity of this unit over/at this time window"

**Examples:**
- "At 9:47am, customer had 15 seats" (instant observation)
- "From Jan 31 8pm to Feb 1 4am, compute consumed 8 hours" (time-spanning observation)
- "At time T, API call consumed 500 tokens" (instant observation)

**Structure:**
```go
Observation {
    Quantity Decimal
    Unit     Unit
    Window   TimeWindow  // Temporal extent
}
```

**Key characteristics:**
- Raw data extracted from events
- Always has temporal context (Window)
- Input to aggregation
- Cannot be re-aggregated (raw data)

**From ADR (observation-temporal-context.md):**
> "Observations are raw measurements from events. They are fundamentally different from aggregations."

**Industry mapping:**
- Similar to OpenTelemetry "data point" or "sample"
- But with explicit temporal window (not just timestamp)

---

### Measurement
**Definition:** *[DEPRECATED - NOT USED IN CODEBASE]*

**Previous usage (REMOVED):**
- ~~`MeasurementExtraction`~~ → Renamed to `ObservationExtraction`
- ~~`MeasurementSourceProperty`~~ → Renamed to `ObservationSourceProperty`

**Decision rationale:**
- "Measurement" is ambiguous (could mean process or result)
- "Observation" better captures semantics: "what was observed" (cannot be re-aggregated)
- Aligns with observability terminology (data points, samples)
- Term removed from codebase in favor of more precise domain terms

---

### Metric
**Definition:** *[NOT CURRENTLY USED IN CODEBASE]*

**Industry meaning (OpenTelemetry/Prometheus):**
- A named, typed stream of measurements
- Example: `http_requests_total{method="GET"}` (counter metric)
- Example: `memory_usage_bytes` (gauge metric)

**Potential metering mapping:**
- Could mean: A (Unit, AggregationType) pair defining what to track
- Could mean: The configuration for tracking something
- Currently we use "Unit" instead (e.g., "tokens", "seats")

**Why not used:**
- "Metric" is vague (could mean observation, aggregate, or configuration)
- We use more specific terms: Observation (raw), AggregateValue (computed), Unit (what we're counting)

**Decision:** Avoid "metric" - too ambiguous. Use specific terms.

---

## 3. Computed Values: What We Calculate

### ComputedValue
**Definition:** A computed value resulting from applying an aggregation or transformation to observations, representing *the answer to a computation query*.

**Meaning:** "Using this aggregation function, the result was this quantity of this unit"

**Examples:**
- "Sum aggregation: customer consumed 12,500 tokens"
- "Time-weighted-avg aggregation: customer averaged 12.32 seats"
- "Max aggregation: peak concurrent connections was 47"

**Structure:**
```go
ComputedValue {
    Quantity    Decimal
    Unit        Unit
    Aggregation AggregationType  // Which function produced this value
    // NO Window - temporal context is in parent MeterReading
}
```

**Key characteristics:**
- Computed from observations
- Includes the aggregation type used (essential metadata)
- Output from aggregation
- More general than "aggregate" (allows for future non-aggregation computations)
- *Could* be re-aggregated with proper windowing/weighting (hierarchical aggregation)

**From ADR (observation-temporal-context.md):**
> "Aggregations are computed results from observations. Different operations are valid: can aggregate observations → readings. Type system should prevent invalid operations."

**Naming rationale:**
- "Computed" is more general than "Aggregate" (not all computations are aggregations)
- Explicitly includes aggregation type to make computation strategy transparent
- `ComputedValueSpec` in specs layer

**Previous name:** `AggregateValue` (renamed for clarity and to include aggregation type)

---

## 4. Containers: What Holds These Values

### MeterRecord
**Definition:** The atomic unit of metered usage, containing observations extracted from an event.

**Meaning:** "At ObservedAt time, for this Subject, we observed these values, from this source event"

**Structure:**
```go
MeterRecord {
    ID, WorkspaceID, UniverseID, Subject
    ObservedAt    time.Time       // When observed
    Observations  []Observation   // What was observed (bundled by source event)
    Dimensions    map[string]string
    SourceEventID string
    MeteredAt     time.Time       // When system processed
}
```

**Key characteristics:**
- Result of `Meter()` operation
- One record per source event (but may contain multiple observations)
- Immutable once created
- Input to aggregation

**Naming rationale:**
- "Record" = immutable historical fact
- "Meter" = verb form of "metering"
- Together: "a record of what was metered"

---

### MeterReading
**Definition:** The result of aggregating meter records over a time window, containing one or more computed values.

**Meaning:** "During this window, for this Subject, these are the computed values (one per unit)"

**Structure:**
```go
MeterReading {
    ID, WorkspaceID, UniverseID, Subject
    Window         TimeWindow        // Aggregation period
    ComputedValues []ComputedValue   // Computed results (one per unit)
    Aggregation    AggregationType   // Which function was used
    RecordCount    int               // How many records contributed
    CreatedAt      time.Time         // When computed
    MaxMeteredAt   time.Time         // Watermark for completeness
}
```

**Key characteristics:**
- Result of `Aggregate()` operation
- Can contain multiple computed values (e.g., input-tokens + output-tokens)
- Each ComputedValue has its own quantity, unit, and aggregation type
- Derived data (can be recomputed from records)
- Output of metering system (used for billing)

**Naming rationale:**
- "Reading" = what you get when you "read the meter"
- Like a utility meter reading: "Your usage this month was X"

**Migration note:** Previously had singular `Value` field, now has `ComputedValues` array to support multiple units per reading.

---

## 5. Operations: What Transforms Data

### Meter() / Metering
**Definition:** The process/function that transforms event payloads into meter records by extracting observations.

**Signature:**
```go
func Meter(
    payload EventPayloadSpec,
    config MeteringConfigSpec,
) ([]MeterRecordSpec, error)
```

**Operation:**
1. Apply filters to determine which extractions apply
2. Extract numeric values from event properties
3. Attach units based on configuration
4. Compute temporal windows (instant or span)
5. Create meter records with observations

**Naming rationale:**
- "Meter" as a verb: "to measure and record"
- Parallel to "metronome" (measure time), "metric" (measure)
- Domain action: we "meter" events to track usage

---

### Aggregate() / Aggregation
**Definition:** The process/function that transforms meter records into meter readings by computing aggregate values.

**Signature:**
```go
func Aggregate(
    recordsInWindow []MeterRecordSpec,
    lastBeforeWindow *MeterRecordSpec,
    config AggregateConfigSpec,
) (MeterReadingSpec, error)
```

**Operation:**
1. Unbundle observations from records
2. Apply aggregation function (sum, max, time-weighted-avg, etc.)
3. Compute aggregate value
4. Create meter reading with result

**Naming rationale:**
- "Aggregate" as a verb: "to combine multiple items into a total"
- Mathematical operation: sum, average, max, etc.
- Standard term across databases and analytics

---

### ObservationExtraction
**Definition:** The configuration for how to extract an observation from an event's properties.

**Meaning:** "Extract this property, assign this unit, if this filter matches"

**Structure:**
```go
ObservationExtraction {
    SourceProperty string
    Unit           string
    Filter         *FilterSpec
}
```

**Key characteristics:**
- Configuration (not data)
- Defines which event property to extract
- Specifies what unit to assign
- Optionally filters which events to extract from

**Naming rationale:**
- "Observation" aligns with data type produced (Observation)
- "Extraction" clarifies this is configuration for extraction process
- Clear relationship: ObservationExtraction → produces → Observation

**Previous name:** `MeasurementExtraction` (renamed for alignment with domain terminology)

---

## 6. Types & Classification

### Counter (Aggregation Type)
**Definition:** An aggregation type for discrete events where each event is a countable occurrence.

**Valid aggregations:**
- `sum-events` - Total number/value of events
- `max-event` - Maximum single event value
- `min-event` - Minimum single event value
- `latest-event` - Most recent event value

**Examples:**
- API calls (count them)
- Token consumption (sum them)
- Bytes transferred (sum them)

**From observability-vs-metering.md:**
> "Counter → Event aggregations (sum-events, max-event)"

**Industry context:**
- Prometheus: monotonically increasing cumulative total
- Metering: sum of discrete event values (different!)

---

### Gauge (Aggregation Type)
**Definition:** An aggregation type for state observations where the value represents state at a point in time.

**Valid aggregations:**
- `time-weighted-avg` - True time-weighted average (not arithmetic mean)
- `peak-state` - Maximum state during period
- `min-state` - Minimum state during period
- `final-state` - State at end of period

**Examples:**
- Active seats (average them over time)
- Concurrent connections (find peak)
- Storage usage (time-weighted average)

**From observability-vs-metering.md:**
> "Gauge → State aggregations (time-weighted-avg, peak-state, final-state)"

**Industry context:**
- Prometheus: current value that goes up/down
- Metering: reconstructed state requiring special aggregation (time-weighting)

---

### Unit
**Definition:** The semantic type of what is being metered, determining how observations aggregate.

**Examples:**
- "tokens" (counter - sum them)
- "seats" (gauge - time-weighted average)
- "api-calls" (counter - count them)
- "gb-hours" (counter - sum composite usage)

**Key characteristic:**
- Unit determines which aggregations are valid
- Unit is used for billing (maps to rate card)
- One unit = one time series of observations

---

### Aggregation Type
**Definition:** The specific function/algorithm used to aggregate observations.

**Current types:**
- Counter aggregations: `sum-events`, `max-event`, `min-event`, `latest-event`
- Gauge aggregations: `time-weighted-avg`, `peak-state`, `min-state`, `final-state`

**Key principle from aggregation-types.md:**
> "Names reveal semantics: 'time-weighted-avg' tells you it's for gauges with time-weighting, not just arithmetic mean"

---

## 7. Temporal Concepts

### Window / TimeWindow
**Definition:** The temporal extent over which something was observed or aggregated.

**Structure:**
```go
TimeWindow {
    Start time.Time
    End   time.Time
}
```

**Convention:** Half-open interval `[Start, End)`

**Two contexts:**
1. **In Observation:** When the observation occurred
   - Instant: `[T, T]` (Start == End)
   - Time-spanning: `[T1, T2]` (Start < End)

2. **In MeterReading:** The aggregation period
   - Always a span: `[T1, T2]`

**From ADR:**
> "All observations require temporal context: instant observations occur at a point in time, while time-spanning observations occur over a window."

---

### ObservedAt
**Definition:** Business time - when the observation was taken (instant) or finalized (span).

**Location:** `MeterRecord.ObservedAt`

**Meaning:**
- For instant observations: the moment of observation
- For time-spanning observations: typically equals Window.End

**Distinction from MeteredAt:**
- ObservedAt = domain/business time
- MeteredAt = system/technical time

---

### MeteredAt
**Definition:** System time - when the metering system processed the event and created the record.

**Location:** `MeterRecord.MeteredAt`

**Purpose:**
- Watermarking (knowing when data is complete)
- Debugging and audit trails
- Ordering records when ObservedAt is the same

---

### Instant vs Time-Spanning
**Definition:** The two temporal patterns for observations.

**Instant observation:**
- Represents a snapshot at a point in time
- Window: `[T, T]` (Start == End)
- Example: "At 9:47am, customer had 15 seats"

**Time-spanning observation:**
- Represents activity over a duration
- Window: `[T1, T2]` (Start < End)
- Example: "From Jan 31 8pm to Feb 1 4am, compute consumed 8 hours"

**From time-spanning-events.md:**
> "Events can represent instant observations (point-in-time gauge) or time-spanning activity (activity over period)"

---

## 8. Organizational Concepts

### Subject
**Definition:** The entity being metered - who or what the usage is attributed to.

**Format:** `"type:id"` (e.g., `"customer:cust_123"`)

**Examples:**
- `"customer:cust_abc"` - A customer
- `"organization:org_456"` - An organization
- `"team:team_789"` - A team
- `"workspace:ws_xyz"` - A workspace itself

**Key characteristic:**
- Subject is the billing entity
- All records and readings are *per subject*
- Subject is the primary grouping dimension

---

### Workspace
**Definition:** The highest level of tenant isolation in the system.

**Purpose:**
- Multi-tenancy boundary
- Each workspace has its own:
  - Event schemas
  - Metering configurations
  - Subject namespace
  - Data isolation

**Example:** Each customer of the metering platform is a separate workspace.

---

### Universe
**Definition:** A secondary isolation boundary within a workspace.

**Purpose:**
- Further partitioning within a workspace
- Use cases:
  - Staging vs production
  - Different business units
  - Geographic regions

**Example:** Same workspace might have `universe:"production"` and `universe:"staging"`

---

### Dimensions
**Definition:** Key-value attributes attached to events and records for filtering and grouping.

**Examples:**
- `region: "us-east-1"`
- `tier: "premium"`
- `model: "gpt-4"`
- `status: "success"`

**Key characteristics:**
- Pass through from event properties
- Not extracted as observations (observations come from numeric properties)
- Used for filtering during extraction
- Available for grouping in downstream analytics

**Cardinality warning:** High-cardinality dimensions (like transaction IDs) should be avoided.

---

## 9. Configuration Concepts

### MeteringConfig
**Definition:** Configuration that defines how to transform events into meter records.

**Structure:**
```go
MeteringConfig {
    Measurements []MeasurementExtraction  // [NAME UNDER REVIEW]
}
```

**Purpose:**
- Maps event properties to observations
- Assigns units
- Applies conditional logic (filters)

---

### AggregateConfig / AggregationConfig
**Definition:** Configuration that defines how to aggregate meter records into readings.

**Structure:**
```go
AggregateConfig {
    Window      TimeWindow
    Aggregation AggregationType
}
```

**Purpose:**
- Specifies aggregation period
- Specifies aggregation function

---

## 10. Terminology We Explicitly AVOID

### "Metric"
**Why avoid:** Overloaded term with multiple meanings
- Could mean: observation, aggregate, configuration, or unit
- Industry uses it for time series streams

**What we use instead:**
- Observation (raw value)
- AggregateValue (computed value)
- Unit (what we're measuring)

---

### "Consumption" / "Usage"
**Current status:** "Usage" appears in documentation but not as a core domain type.

**Why not a core term:**
- Too vague (could mean observation, aggregate, or total billing)
- Not precise enough for technical specification

**Where used:**
- User-facing docs: "Track API usage"
- Generic term in comments

**Prefer instead:**
- "Observation" when talking about extracted values
- "AggregateValue" when talking about computed results
- "Usage-based billing" when talking about the business model

---

## 11. Terms Under Review

### MeasurementExtraction → ObservationExtraction?
**Current:** `MeasurementExtraction`
**Proposed:** `ObservationExtraction`

**Question:** Does "Measurement" have intentional semantic distinction?
- Option A: "Measurement" = process, "Observation" = result → Keep as is, document distinction
- Option B: "Measurement" is obsolete term → Rename to align with "Observation"
- Option C: Both are wrong → Choose better term (e.g., "ValueExtraction")

**Action needed:** Review ADRs and decide.

---

### MeasurementSourceProperty → ObservationSourceProperty?
**Current:** `MeasurementSourceProperty`
**Proposed:** `ObservationSourceProperty`

**Dependencies:** Same decision as MeasurementExtraction.

---

## Summary: Core Terms Hierarchy

```
EventPayload (raw data)
    ↓ [Meter() operation with MeteringConfig]
MeterRecord (metered data)
    containing Observations (extracted values with temporal context)
    ↓ [Aggregate() operation with AggregateConfig]
MeterReading (aggregated data)
    containing AggregateValue (computed result)
```

**Key distinction:**
- **Observation** = raw, extracted from events, has temporal window
- **AggregateValue** = computed, calculated from observations, window is in parent

**Configuration:**
- **MeteringConfig** = how to extract observations
- **AggregateConfig** = how to compute aggregates

**Operations:**
- **Meter()** = EventPayload → MeterRecord (extraction)
- **Aggregate()** = MeterRecord → MeterReading (aggregation)

---

## Implementation Summary

**Status:** All naming decisions finalized and implemented in codebase.

### Key Decisions Made:

1. **Observation over Measurement**
   - ✅ Renamed `MeasurementExtraction` → `ObservationExtraction`
   - ✅ Renamed `MeasurementSourceProperty` → `ObservationSourceProperty`
   - Rationale: "Observation" better captures "what was observed" semantics

2. **ComputedValue over AggregateValue**
   - ✅ Renamed `AggregateValue` → `ComputedValue`
   - ✅ Renamed `AggregateSpec` → `ComputedValueSpec`
   - ✅ Added `Aggregation` field to make computation strategy explicit
   - Rationale: More general term, includes aggregation type metadata

3. **ComputedValues Array**
   - ✅ Changed `MeterReading.Value` (singular) → `MeterReading.ComputedValues` (array)
   - Rationale: Supports multiple units per reading (e.g., input-tokens + output-tokens)

### Migration Completed:

- **Phase 1:** Added new types parallel to old ones
- **Phase 2:** Migrated all callers to new API
- **Phase 3:** Removed all deprecated types
- **All tests passing:** internal, examples, benchmarks, specs

### Codebase Status:

- ✅ Observation terminology throughout
- ✅ ComputedValue with explicit aggregation types
- ✅ No deprecated types remaining
- ✅ Consistent naming across all layers

---

## Related Documentation

- `design/observation-temporal-context.md` - ADR defining Observation and ComputedValue types
- `design/references/observability-vs-metering.md` - Industry terminology and counter/gauge semantics
- `design/aggregation-types.md` - Aggregation type naming and semantics
- `docs/tmp/measurement-extraction-naming.md` - Historical: Original naming consideration (resolved)
- Git commits:
  - Phase 1-3: ObservationExtraction migration
  - Phase 1-3: ComputedValue migration

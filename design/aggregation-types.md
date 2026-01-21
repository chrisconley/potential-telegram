# Aggregation Types: Counter vs Gauge Operations

**Date:** 2026-01-21
**Status:** Proposed Design
**Decision:** Use explicit aggregation names that encode meter type semantics

---

## Problem Statement

Generic aggregation names like `max`, `min`, and `latest` are **implementation details that hide semantic differences** between counter and gauge operations.

### Example: "max" Has Different Meanings

**On Counter Data (API tokens consumed):**
- `max` = "largest single API call this month"
- Looks at discrete events: [100, 500, 200] → max = 500
- Semantic: "What was the biggest event?"

**On Gauge Data (concurrent users):**
- `max` = "peak concurrent users this month"
- Looks at state over time: carries forward values between readings
- Semantic: "What was the highest state ever reached?"

**Same operation name, fundamentally different semantics.**

### The Silent Failure Problem

If we add a separate `meterType` field:

```json
{
  "aggregation": "max",
  "meterType": "gauge",
  "window": {...}
}
```

**Problems:**
1. User must remember to set `meterType` correctly
2. Mismatch causes **silent incorrectness** (no error, just wrong results)
3. `aggregation` and `meterType` can conflict
4. Not self-documenting - have to check both fields

---

## Solution: Explicit Aggregation Names

Aggregation names should **encode the semantic operation**, not just the implementation.

### Counter Aggregations

Operations on **discrete events** (don't persist over time):

| Aggregation | Description | Example Use Case |
|-------------|-------------|------------------|
| `sum-events` | Sum all events in window | Total API tokens consumed |
| `max-event` | Largest single event | Biggest API call |
| `min-event` | Smallest single event | Smallest API call |
| `latest-event` | Value of most recent event | Last API call size |

### Gauge Aggregations

Operations on **state that persists over time** (requires timeline reconstruction):

| Aggregation | Description | Example Use Case |
|-------------|-------------|------------------|
| `time-weighted-avg` | Average state over time | Average seats over month |
| `peak-state` | Maximum state reached | Peak concurrent users |
| `min-state` | Minimum state reached | Minimum concurrent users |
| `final-state` | State at end of window | Final seat count |

---

## Benefits

### 1. No Separate meterType Field Needed

**Before (requires two fields):**
```json
{
  "aggregation": "max",
  "meterType": "gauge",
  "window": {...}
}
```

**After (single field):**
```json
{
  "aggregation": "peak-state",
  "window": {...}
}
```

### 2. Self-Documenting

`peak-state` clearly means gauge data. `max-event` clearly means counter data.

No ambiguity, no need to cross-reference another field.

### 3. Type-Safe by Design

Can't accidentally use `time-weighted-avg` on counter data - the name makes it obvious.

Validation is simple:
```go
if isGaugeAggregation(config.Aggregation) && lastBeforeWindow == nil {
    return error("gauge aggregations require lastBeforeWindow")
}
```

### 4. Future-Proof

If counter `max-event` and gauge `peak-state` implementations need to diverge (e.g., different handling of edge cases), they're already separate operations.

### 5. Clear Semantics

User explicitly declares their intent:
- "I want the **peak state**" vs "I want the **max event**"
- No risk of using the wrong operation silently

---

## Implementation Mapping

### Current Implementation

```go
// internal/meterreading.go
const (
    aggregationTypeSum             = "sum"
    aggregationTypeMax             = "max"
    aggregationTypeMin             = "min"
    aggregationTypeLatest          = "latest"
    aggregationTypeTimeWeightedAvg = "time-weighted-avg"
)
```

### Proposed Implementation

```go
// internal/meterreading.go
const (
    // Counter aggregations
    aggregationTypeSumEvents    = "sum-events"
    aggregationTypeMaxEvent     = "max-event"
    aggregationTypeMinEvent     = "min-event"
    aggregationTypeLatestEvent  = "latest-event"

    // Gauge aggregations
    aggregationTypeTimeWeightedAvg = "time-weighted-avg"
    aggregationTypePeakState       = "peak-state"
    aggregationTypeMinState        = "min-state"
    aggregationTypeFinalState      = "final-state"
)

func (a MeterReadingAggregation) IsGaugeAggregation() bool {
    switch a.value {
    case aggregationTypeTimeWeightedAvg,
         aggregationTypePeakState,
         aggregationTypeMinState,
         aggregationTypeFinalState:
        return true
    default:
        return false
    }
}

func (a MeterReadingAggregation) IsCounterAggregation() bool {
    switch a.value {
    case aggregationTypeSumEvents,
         aggregationTypeMaxEvent,
         aggregationTypeMinEvent,
         aggregationTypeLatestEvent:
        return true
    default:
        return false
    }
}
```

### Aggregate Function Validation

```go
func aggregate(
    recordsInWindow []MeterRecord,
    lastBeforeWindow *MeterRecord,
    config AggregationConfig,
) (MeterReading, error) {
    // Validate gauge aggregations have required lastBeforeWindow
    if config.Aggregation().IsGaugeAggregation() && lastBeforeWindow == nil {
        return MeterReading{}, fmt.Errorf(
            "gauge aggregation %q requires lastBeforeWindow for timeline reconstruction",
            config.Aggregation().ToString(),
        )
    }

    // ... rest of implementation
}
```

---

## Example Configurations

### Counter: API Token Usage

```json
{
  "aggregation": "sum-events",
  "window": {
    "start": "2026-02-01T00:00:00Z",
    "end": "2026-03-01T00:00:00Z"
  }
}
```

**Semantics:** Total API tokens consumed in February

### Gauge: Seat Billing

```json
{
  "aggregation": "time-weighted-avg",
  "window": {
    "start": "2026-02-01T00:00:00Z",
    "end": "2026-03-01T00:00:00Z"
  }
}
```

**Semantics:** Average seats over February (properly accounting for state changes)

### Gauge: Peak Capacity

```json
{
  "aggregation": "peak-state",
  "window": {
    "start": "2026-02-01T00:00:00Z",
    "end": "2026-03-01T00:00:00Z"
  }
}
```

**Semantics:** Peak concurrent users reached in February

---

## Migration Path

### Phase 1: Add New Names (Backwards Compatible)

Support both old and new names:
```go
case "max":
    // Deprecated: use "max-event" or "peak-state"
    return maxRecords(recordsInWindow)
case "max-event", "peak-state":
    return maxRecords(recordsInWindow)
```

### Phase 2: Deprecate Generic Names

Log warnings when generic names are used:
```go
case "max":
    log.Warn("aggregation 'max' is deprecated, use 'max-event' or 'peak-state'")
```

### Phase 3: Remove Generic Names

Only support explicit names.

---

## Alternative Considered: Separate meterType Field

```json
{
  "meterType": "gauge",
  "aggregation": "max",
  "window": {...}
}
```

**Rejected because:**
1. Two fields can be inconsistent
2. Not self-documenting - must check both fields
3. Extra validation complexity
4. User can set wrong meterType silently

---

## Design Principles Applied

### Principle #6: Client Responsibility
> "Clear contracts: client owns data quality"

By making the aggregation name explicit (`peak-state` vs `max-event`), we force the client to declare their intent clearly. No ambiguity, no silent failures.

### Principle #9: Users First
> "Engineer for actual user needs"

Users need **correctness** - they want to compute the right metric. Explicit aggregation names prevent mistakes and make the system easier to use correctly.

### Principle #11: Fix Root Causes, Not Symptoms
> "Surface-level fixes perpetuate bad patterns"

Rather than adding validation to catch mismatched `aggregation` + `meterType` fields, we fix the root cause: **the aggregation name should encode the semantic operation**.

---

## References

- Prometheus Metric Types: https://prometheus.io/docs/concepts/metric_types/
- Chris's Design Principles: `arch/reference/chris-design-principles.md`
- Related discussion: Gauge vs counter semantics in aggregation design (2026-01-21)

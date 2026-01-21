# Time-Spanning Events: Patterns and Recommendations

**Date:** 2026-01-20
**Context:** How to handle events that represent time windows or durations rather than point-in-time occurrences

---

## Problem Statement

Many usage metering scenarios involve events that span time rather than occur at a single instant:

- **Sessions**: User logged in from 13:00 to 15:00
- **Reservations**: Resource reserved from Jan 1 to Jan 7
- **Continuous Usage**: GPU running for 3.5 hours
- **Billable Periods**: Service active during specific time window

The question: How do we represent temporal spans in event formats like CloudEvents?

---

## CloudEvents Standard Guidance

CloudEvents v1.0 defines a single `time` attribute as RFC3339 timestamp representing "when the occurrence happened." The specification does not define standard attributes for:
- Duration
- Time ranges
- Start/end timestamps
- Intervals

**Key Insight from EventSourcingDB (2026):**
> "The CloudEvents `time` field is automatically set by systems when an event is stored, but using this technical timestamp for business logic can lead to subtle bugs. Events are frequently recorded after the actual occurrence."

This highlights the distinction between:
- **System time**: When event was recorded (CloudEvents `time`)
- **Business time**: When event actually occurred (domain-specific)

---

## Industry Patterns

### OpenMeter Approach

OpenMeter uses **point-in-time events with duration in data**:

```json
{
  "specversion": "1.0",
  "type": "request",
  "source": "/api/compute",
  "id": "evt-123",
  "time": "2024-01-01T00:00:00.001Z",
  "data": {
    "duration_seconds": 7200
  }
}
```

**Pattern:**
- `time` = when event completed or was reported
- `data.duration_seconds` = span length
- System aggregates into time windows automatically

**Time Windows:**
OpenMeter assigns events to windows:
- Event at `2024-01-01T00:00:00.001Z`
- Assigned to window `[2024-01-01T00:00, 2024-01-01T00:01)`
- Multiple events in same window are aggregated (SUM, MAX, etc.)

---

## Recommended Patterns

### Pattern 1: Point-in-Time with Duration (Simplest)

**Use when:** Events represent "work completed" or "usage that happened"

```json
{
  "specversion": "1.0",
  "type": "com.example.usage.session",
  "source": "/compute/instance-123",
  "id": "session-abc",
  "time": "2026-01-20T15:00:00Z",
  "data": {
    "duration_seconds": 7200,
    "resource_id": "vm-456"
  }
}
```

**Semantics:**
- `time`: When span ended (or when you're reporting it)
- `duration_seconds`: How long the span was
- Start time: Implicit (time - duration)

**Benefits:**
- Standard CloudEvents compliance
- Simple to process
- Works with existing aggregation tooling
- Common industry pattern

**Trade-offs:**
- Loses explicit start time
- Must calculate backwards to determine span boundaries
- Less clear for overlapping spans

---

### Pattern 2: Custom Extension Attributes (Most Explicit)

**Use when:** Temporal boundaries are first-class routing/filtering concerns

```json
{
  "specversion": "1.0",
  "type": "com.example.usage.session",
  "source": "/compute/instance-123",
  "id": "session-abc",
  "time": "2026-01-20T15:00:00Z",
  "starttime": "2026-01-20T13:00:00Z",
  "endtime": "2026-01-20T15:00:00Z",
  "data": {
    "resource_id": "vm-456"
  }
}
```

**Semantics:**
- `time`: Standard CloudEvents timestamp (typically end or report time)
- `starttime`: Custom extension for span start
- `endtime`: Custom extension for span end

**Benefits:**
- Explicit temporal boundaries
- Can query/filter on start vs. end at routing layer
- Clear semantics for overlapping spans
- No need to parse data payload for temporal info

**Trade-offs:**
- Custom extensions (not in CloudEvents standard)
- Requires documentation for consumers
- Some CloudEvents processors may ignore extensions
- Need consensus across producers/consumers

**Extension Definition:**

Per CloudEvents extension guidelines, custom extensions should:
- Use lowercase attribute names
- Be documented with semantics
- Have at least two organizations agreeing (for standardization)

---

### Pattern 3: Temporal Data in Payload (Most Flexible)

**Use when:** Different event types have different temporal semantics

```json
{
  "specversion": "1.0",
  "type": "com.example.usage.session",
  "source": "/compute/instance-123",
  "id": "session-abc",
  "time": "2026-01-20T15:00:00Z",
  "data": {
    "session_start": "2026-01-20T13:00:00Z",
    "session_end": "2026-01-20T15:00:00Z",
    "duration_seconds": 7200,
    "resource_id": "vm-456"
  }
}
```

**Semantics:**
- `time`: When event was recorded/reported (system time)
- `session_start`: Business time when span began
- `session_end`: Business time when span ended
- `duration_seconds`: Explicit or derived

**Benefits:**
- Fully standard CloudEvents (no extensions)
- Complete temporal information
- Domain-specific field names
- Flexible per event type

**Trade-offs:**
- Can't filter on start/end at CloudEvents routing layer
- Must parse `data` to understand temporal boundaries
- Slightly more verbose

---

## Recommendation for Metering Spec

Given our design principles and EventPayload/Event model:

### **For Most Use Cases: Pattern 1 or Pattern 3**

**Pattern 1** if:
- Events represent completed work or usage
- End time + duration is sufficient
- Integration with existing metering systems (OpenMeter-like)

**Pattern 3** if:
- Temporal boundaries are semantically important
- Different event types have different temporal models
- Need explicit start/end for business logic

### **Our Current Design Already Supports Both**

```go
type EventPayload struct {
    TransactionID EventPayloadTransactionID
    WorkspaceID   EventPayloadWorkspaceID
    Universe      EventPayloadUniverse
    EventType     EventPayloadType
    Party         EventPayloadParty
    Timestamp     EventPayloadTimestamp  // Business time: when event occurred
    Properties    EventPayloadProperties // map[string]string
}
```

**Flexibility via Properties:**
- Properties can contain `duration_seconds` (Pattern 1)
- Properties can contain `start_time` + `end_time` (Pattern 3)
- `IngestionConfig` defines which pattern per event type

**Workspace-Specific Schemas:**
Different workspaces can choose different patterns:
- `(workspace-a, "session")` → uses duration pattern
- `(workspace-b, "session")` → uses start/end pattern

**No changes needed to core schema.**

---

## System Time vs. Business Time

Our model already handles the key distinction identified by EventSourcingDB:

**Business Time:**
- `EventPayload.Timestamp` / `Event.Timestamp`
- When the event actually occurred
- Used for metering, aggregation, time windows
- Can be in the past (corrections, backfills)

**System Time:**
- `Event.CreatedAt` / `MeterRecord.MeteredAt` / `MeterReading.CreatedAt`
- When the event was processed/recorded
- Used for watermarking, incremental processing
- Always monotonically increasing

For time-spanning events:
- `Timestamp` can represent start, end, or midpoint (schema-defined)
- `Properties` contain additional temporal details (duration, start, end)
- `CreatedAt` tracks when we processed the event

---

## Examples

### Example 1: Compute Session (Pattern 1)

```json
{
  "transaction_id": "txn-123",
  "workspace_id": "acme-us",
  "universe": "production",
  "event_type": "compute.session",
  "party": "customer:cust-456",
  "timestamp": "2026-01-20T15:00:00Z",
  "properties": {
    "instance_id": "vm-789",
    "instance_type": "n1-standard-4",
    "duration_seconds": "7200"
  }
}
```

**Ingestion Config:**
```go
MeasureProperties: {
  "duration_seconds": {
    Transformer: PositiveDecimal,
    Required: true
  }
}
```

**Result Event:**
```go
Event{
  Timestamp: 2026-01-20T15:00:00Z  // End time
  Measures: {
    "duration_seconds": 7200
  }
}
```

**Metering:**
- Create MeterRecord with unit "compute-hours"
- Value: 7200 / 3600 = 2.0 hours
- MeterRecord.Timestamp: 2026-01-20T15:00:00Z (end time)

**Aggregation:**
- Assign to time window based on end time
- SUM all compute-hours in billing period

---

### Example 2: Reservation (Pattern 3)

```json
{
  "transaction_id": "txn-456",
  "workspace_id": "acme-us",
  "universe": "production",
  "event_type": "reservation.created",
  "party": "customer:cust-789",
  "timestamp": "2026-01-20T10:00:00Z",
  "properties": {
    "reservation_id": "res-abc",
    "resource_id": "gpu-123",
    "start_time": "2026-01-25T00:00:00Z",
    "end_time": "2026-01-30T00:00:00Z"
  }
}
```

**Ingestion Config:**
```go
DimensionProperties: {
  "reservation_id": {Required: true},
  "resource_id": {Required: true},
  "start_time": {Required: true},
  "end_time": {Required: true}
}
```

**Result Event:**
```go
Event{
  Timestamp: 2026-01-20T10:00:00Z  // When reservation was created
  Dimensions: {
    "reservation_id": "res-abc",
    "start_time": "2026-01-25T00:00:00Z",
    "end_time": "2026-01-30T00:00:00Z"
  }
}
```

**Metering:**
- Downstream service expands to daily reservation events
- Or meters on start_time with 5-day duration
- Or charges upfront based on (end_time - start_time)

---

## When to Use Each Pattern

### Use Pattern 1 (Duration) when:
- Events represent completed work
- End time is the natural reporting point
- Integration with OpenMeter-like systems
- Simplicity is preferred

**Examples:**
- API requests (duration_ms)
- Compute sessions (duration_seconds)
- File storage (measured at end of month)

### Use Pattern 3 (Start/End in Data) when:
- Need explicit temporal boundaries
- Start and end have different business meanings
- Overlapping spans are common
- Downstream needs to reason about time windows

**Examples:**
- Reservations (booked from X to Y)
- Subscriptions (active_from to active_until)
- Rate card validity periods
- Promotional pricing windows

### Avoid Pattern 2 (Custom Extensions) unless:
- You control both producers and consumers
- Temporal boundaries need routing/filtering
- You're proposing a standard extension

---

## Open Questions

1. **Aggregation semantics for overlapping spans:**
   - How do we aggregate usage when spans overlap?
   - Should we convert spans to point-in-time charges?
   - Or track active spans per time window?

2. **Partial period handling:**
   - Event spans Jan 15 - Feb 15, billing period is calendar month
   - Do we split into two meter records?
   - Or record full span and let aggregation handle proration?

3. **Timezone handling:**
   - All timestamps UTC (recommended)
   - Or support timezone in properties?
   - How does this interact with time window assignment?

4. **Late-arriving spans:**
   - Session started yesterday, ended today
   - Do we emit one event (at end) or two (start + end)?
   - How does watermarking handle in-progress spans?

---

## References

### External Resources

- [OpenMeter Usage Events](https://openmeter.io/docs/metering/events/usage-events) - Industry example of duration-based metering
- [CloudEvents Specification v1.0](https://github.com/cloudevents/spec/blob/main/cloudevents/spec.md) - Standard event format
- [CloudEvents Extensions README](https://github.com/cloudevents/spec/blob/main/cloudevents/extensions/README.md) - How to create custom extensions
- [OpenTelemetry CloudEvents Spans](https://opentelemetry.io/docs/specs/semconv/cloudevents/cloudevents-spans/) - Semantic conventions for spans
- [EventSourcingDB: Time is of the Essence](https://docs.eventsourcingdb.io/blog/2026/01/12/time-is-of-the-essence/) - System time vs. business time

### Internal Documentation

- `arch/adr/workspace-universe-isolation.md` - Two-dimensional isolation model
- `arch/designs/watermarking-strategy.md` - System time vs. business time handling
- `ingestion/eventpayload.go` - EventPayload schema
- `ingestion/event.go` - Event schema with Timestamp and CreatedAt

---

## Conclusion

**Recommendation:** Use **Pattern 1** (duration in data) as the default for simplicity, but support **Pattern 3** (start/end in data) when temporal boundaries are semantically important.

**Our current design is flexible:** The `Properties map[string]string` approach allows workspaces to choose their temporal model per event type without changing core schemas.

**No changes needed:** EventPayload and Event already support both patterns via property transformation in IngestionConfig.

**Next step:** Document recommended patterns per event type category in workspace schema guidelines.

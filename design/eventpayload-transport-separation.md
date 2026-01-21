# EventPayload: Metering Domain Model vs. Transport Format

**Date:** 2026-01-21
**Status:** Accepted
**Context:** Separating metering domain concerns from event transport concerns

---

## Summary

EventPayload is the **pure metering domain model**, not a transport format or CloudEvents envelope. While EventPayload can be used within CloudEvents, Kafka, or other transports, it remains transport-agnostic.

**Key principle**: Transport concerns (routing, delivery, system timestamps) are separate from metering concerns (business time, measurements, billing attribution).

---

## Problem Statement

EventPayload was initially modeled after CloudEvents, creating ambiguity about its purpose:
- Is it a CloudEvents-specific format?
- Does it require CloudEvents infrastructure?
- How do non-CloudEvents users integrate?
- What happens with other transports (Kafka, OpenTelemetry, HTTP)?

This coupling creates several issues:

### Issue 1: Business Time vs System Time Confusion

From `time-spanning-events.md`:
> "The CloudEvents `time` field is automatically set by systems when an event is stored, but using this technical timestamp for business logic can lead to subtle bugs. Events are frequently recorded after the actual occurrence."

```
CloudEvents.time:    System time (when recorded/sent)
EventPayload.Time:   Business time (when usage occurred)
```

These are fundamentally different concerns. Metering requires business time for accurate aggregation and billing period assignment.

### Issue 2: Field Purpose Confusion

When EventPayload resembles CloudEvents:

```go
// What does each field mean?
ID   string  // CloudEvents.id or metering idempotency key?
Type string  // CloudEvents.type (routing) or metering config selector?
Time time.Time // CloudEvents.time (system) or business occurrence time?
```

This ambiguity leads to incorrect implementations where system timestamps get used for billing calculations.

### Issue 3: Forced Coupling

Users not using CloudEvents are forced to understand CloudEvents concepts:
- User with Kafka: "Why does EventPayload look like CloudEvents?"
- User with HTTP POST: "Do I need CloudEvents to meter usage?"
- User with OpenTelemetry: "How do I map OTel events to this CloudEvents-like structure?"

### Issue 4: Violates Single Responsibility Principle

From `chris-design-principles.md` Principle #7:
> "Each component should have exactly one responsibility. If a piece of code has multiple reasons to change, split it."

If EventPayload is coupled to CloudEvents:
- **Reason to change #1**: Metering domain evolves (new aggregation types, measurement extraction)
- **Reason to change #2**: CloudEvents specification changes
- **Reason to change #3**: Need to support other transports

This violates SRP—EventPayload should change only when the metering domain changes.

---

## Design from First Principles

### What is EventPayload's Core Purpose?

EventPayload represents **business usage data for metering**:
- **What happened**: event type → which meter config to apply
- **When it happened**: business timestamp → billing period assignment
- **To whom**: subject → attribution for billing
- **How much**: properties → measurements to extract
- **Where**: workspace/universe → operational and data boundaries

**NOT**:
- Transport mechanism (CloudEvents, Kafka, NATS)
- Routing information (where to send events)
- Delivery metadata (system timestamps, delivery guarantees)

### Concern Separation

```
┌─────────────────────────────────────────┐
│  Transport Layer                        │
│  (CloudEvents, Kafka, HTTP, OTel)       │
│  - Routing                               │
│  - Delivery guarantees                   │
│  - System timestamps                     │
│  - Serialization format                  │
└─────────────────────────────────────────┘
              │
              │ wraps
              ▼
┌─────────────────────────────────────────┐
│  Metering Domain (EventPayload)         │
│  - Business timestamps                   │
│  - Measurement data                      │
│  - Billing attribution                   │
│  - Workspace/universe isolation          │
└─────────────────────────────────────────┘
```

The transport layer wraps the metering domain, not vice versa.

---

## Decision

**EventPayload is the pure metering domain model, separate from any transport format.**

### What This Means

**1. EventPayload remains transport-agnostic**

```go
// Pure metering domain model
// NOT a CloudEvents envelope
// NOT specific to any transport
type EventPayloadSpec struct {
    // Metering idempotency key (may differ from transport ID)
    ID string

    // Metering operational boundary
    WorkspaceID string

    // Metering data namespace
    UniverseID string

    // Metering configuration selector (NOT transport routing type)
    Type string

    // Billing attribution (who gets charged)
    Subject string

    // BUSINESS TIME: when usage occurred (NOT when event sent)
    Time time.Time

    // Measurements to extract for metering
    Properties map[string]string
}
```

**2. CloudEvents wraps EventPayload in data field**

```json
{
  "specversion": "1.0",
  "id": "msg-uuid-789",                   // Transport: delivery deduplication
  "source": "/api/gateway",               // Transport: routing source
  "type": "com.acme.metering.event",      // Transport: routing type
  "time": "2026-01-21T10:00:05Z",         // SYSTEM TIME: when sent
  "datacontenttype": "application/json",
  "data": {
    "id": "txn-123",                      // Metering: idempotency key
    "workspaceID": "us-east",
    "universeID": "production",
    "type": "api.request",                // Metering: config selector
    "subject": "customer:acme",
    "time": "2026-01-21T10:00:00Z",       // BUSINESS TIME: when occurred
    "properties": {
      "endpoint": "/api/users",
      "duration_ms": "125"
    }
  }
}
```

**Critical observations:**
- `CloudEvents.id` ≠ `EventPayload.ID` (transport vs business concern)
- `CloudEvents.time` ≠ `EventPayload.Time` (system vs business time)
- `CloudEvents.type` ≠ `EventPayload.Type` (routing vs metering config)
- EventPayload lives in `CloudEvents.data` (clean separation)

**3. Other transports wrap similarly**

**Kafka:**
```
Kafka Message:
  key: partition key for ordering
  timestamp: when produced (system time)
  value: {EventPayloadSpec as JSON} ← the metering data
```

**HTTP POST:**
```
POST /api/v1/metering/events
Content-Type: application/json

{EventPayloadSpec as JSON}
```

**OpenTelemetry Span:**
```
Span:
  trace_id, span_id: distributed tracing
  start_time: when span started (system time)
  attributes:
    "metering.event": {EventPayloadSpec as JSON}
```

---

## Design Principles Applied

### Principle #7: Single Responsibility

> "Each component should have exactly one responsibility"

**Before (implicit coupling):**
- EventPayload changes when: metering domain evolves OR transport spec changes
- **Multiple reasons to change** ❌

**After (separation):**
- EventPayload changes when: metering domain evolves
- CloudEvents adapter changes when: CloudEvents spec changes
- **Single reason to change** ✓

### Principle #3: Don't Make Decisions Twice

> "Each decision should have exactly one place where it's made"

**Before (coupled IDs/times/types):**
```go
// Are these the same? Do they have to match? Who wins?
CloudEvents.id   vs EventPayload.ID
CloudEvents.time vs EventPayload.Time
CloudEvents.type vs EventPayload.Type
```

**After (different purposes):**
```go
// Each serves a distinct purpose in its layer
CloudEvents.id:   Transport deduplication
EventPayload.ID:  Metering idempotency

CloudEvents.time: System time (when sent)
EventPayload.Time: Business time (when occurred)

CloudEvents.type: Routing/filtering
EventPayload.Type: Meter config selection
```

No duplicate decisions—each field has one clear purpose.

### Principle #9: Users First (Next 10)

> "Engineer for actual user needs—but consider the next 10 users, not just the first"

**User personas:**

| User | Transport | With Coupling | With Separation |
|------|-----------|---------------|-----------------|
| 1-3 | Kafka/HTTP/custom | Forced to learn CloudEvents ❌ | Direct EventPayload usage ✓ |
| 4-7 | CloudEvents | Easy integration ✓ | Easy integration ✓ |
| 8-10 | OpenTelemetry/EventBridge | Trapped by CloudEvents design ❌ | Adapter for their transport ✓ |

Separation serves users 1-10, not just 4-7.

### Principle #2: Avoid if/else Blocks

**With coupling:**
```go
func processEvent(e Event) {
    if e.IsCloudEvents {
        // Extract one way
    } else if e.IsKafka {
        // Extract differently
    } else if e.IsHTTP {
        // Extract yet another way
    }
}
```

**With separation:**
```go
// Each adapter owns its mapping uniformly
func CloudEventsAdapter(ce CloudEvent) EventPayload {
    return ExtractFromData(ce.Data)
}

func KafkaAdapter(msg KafkaMessage) EventPayload {
    return ParseFromValue(msg.Value)
}

// No conditionals—clean adapter pattern
```

---

## Implementation Guidance

### For Library Developers

**EventPayload is the core API:**
```go
func Meter(payload EventPayloadSpec, config MeteringConfigSpec) ([]MeterRecordSpec, error)
```

Users provide EventPayload however they obtain it (CloudEvents, Kafka, HTTP, etc.)

### For CloudEvents Users

**Wrap EventPayload in CloudEvents.data:**
```go
import "github.com/cloudevents/sdk-go/v2"

func PublishToCloudEvents(payload EventPayloadSpec) error {
    ce := cloudevents.NewEvent()
    ce.SetID(uuid.New().String())              // Transport ID
    ce.SetSource("/api/gateway")               // Routing source
    ce.SetType("com.acme.metering.event")      // Routing type
    ce.SetTime(time.Now())                     // System time
    ce.SetData("application/json", payload)    // Metering data

    // Send via CloudEvents transport
    return cloudEventsClient.Send(context.Background(), ce)
}

func ConsumeFromCloudEvents(ce cloudevents.Event) (EventPayloadSpec, error) {
    var payload EventPayloadSpec
    if err := ce.DataAs(&payload); err != nil {
        return EventPayloadSpec{}, err
    }
    return payload, nil
}
```

**Critical: Always use `EventPayload.Time` for metering, not `CloudEvents.time`**

### For Kafka Users

**Serialize EventPayload as message value:**
```go
func PublishToKafka(payload EventPayloadSpec) error {
    data, _ := json.Marshal(payload)
    msg := &kafka.Message{
        Key:   []byte(payload.Subject),        // Partition key
        Value: data,                            // EventPayload
        Time:  time.Now(),                      // System time
    }
    return kafkaProducer.WriteMessages(context.Background(), msg)
}

func ConsumeFromKafka(msg kafka.Message) (EventPayloadSpec, error) {
    var payload EventPayloadSpec
    if err := json.Unmarshal(msg.Value, &payload); err != nil {
        return EventPayloadSpec{}, err
    }
    return payload, nil
}
```

### For HTTP Users

**POST EventPayload directly:**
```go
func PublishViaHTTP(payload EventPayloadSpec) error {
    data, _ := json.Marshal(payload)
    resp, err := http.Post(
        "https://api.acme.com/metering/events",
        "application/json",
        bytes.NewBuffer(data),
    )
    return err
}
```

---

## Time Semantics: Critical for Correctness

### System Time vs Business Time

From `time-spanning-events.md`:
- **System time**: When event was recorded/sent (CloudEvents.time, Kafka.timestamp)
- **Business time**: When event actually occurred (EventPayload.Time)

**For metering, always use EventPayload.Time:**
```go
// ✓ CORRECT: Use business time for aggregation
meterRecord := MeterRecord{
    Timestamp: eventPayload.Time,  // When usage occurred
    MeteredAt: time.Now(),          // When we processed it
}

// ❌ WRONG: Using system time causes billing errors
meterRecord := MeterRecord{
    Timestamp: cloudEvent.Time,     // When sent, not when occurred!
}
```

**Example of the problem:**
```
Session occurred:     2026-01-20 23:58:00 UTC (December billing period)
Event sent:           2026-01-21 00:02:00 UTC (January billing period)

Using CloudEvents.time:    Bills customer in January (wrong!)
Using EventPayload.Time:   Bills customer in December (correct!)
```

### Late-Arriving Events

EventPayload.Time enables correct handling of late events:
```
Event 1: Time = Jan 1 00:00:00, received Jan 1 00:00:01 → January
Event 2: Time = Dec 31 23:59:00, received Jan 1 00:00:02 → December (late)
```

The metering system assigns events to billing periods based on `EventPayload.Time`, regardless of when they arrive (system time).

---

## Migration from Coupled Design

If you have existing code that treats EventPayload as CloudEvents:

**Step 1: Identify conflated concerns**
```go
// Before: Ambiguous
payload.Time  // Is this business time or system time?

// After: Clear
payload.Time           // Business time (when usage occurred)
transportMsg.Timestamp // System time (when sent/received)
```

**Step 2: Update time usage**
```go
// Before: Using wrong timestamp
record.Timestamp = cloudEvent.Time  // System time!

// After: Using business time
record.Timestamp = eventPayload.Time  // Business time
record.MeteredAt = time.Now()         // System time tracked separately
```

**Step 3: Separate ID purposes**
```go
// Before: Overloaded ID
cloudEvent.ID = payload.ID  // Same ID for transport and business?

// After: Separate concerns
cloudEvent.ID = uuid.New()     // Transport deduplication
// EventPayload.ID in data      // Metering idempotency
```

---

## Consequences

### Benefits

✓ **Clear separation of concerns**: Transport vs metering domain
✓ **Transport flexibility**: Works with any event infrastructure
✓ **Correct time semantics**: Business time distinct from system time
✓ **Single responsibility**: EventPayload changes only when metering domain changes
✓ **User flexibility**: Users choose their transport (CloudEvents, Kafka, HTTP, etc.)

### Trade-offs

- Users need to wrap EventPayload in their chosen transport (one extra step)
- Documentation must clearly explain the separation
- Requires discipline to not conflate system time with business time

### Non-Issues

- **Not more complex**: Separation clarifies existing concepts
- **Not less CloudEvents-friendly**: CloudEvents users wrap EventPayload in `.data` (standard pattern)
- **Not reinventing wheels**: Using transports for transport concerns, metering model for metering

---

## Related Documentation

- `time-spanning-events.md` - System time vs business time (CloudEvents.time vs EventPayload.Time)
- `workspace-universe-isolation.md` - Two-dimensional isolation model for workspaces and universes
- `arch/reference/chris-design-principles.md` - Design principles applied in this decision

---

## Examples

### Example 1: CloudEvents Pipeline

```go
// Producer: Wrap EventPayload in CloudEvents
func recordAPIRequest(endpoint string, durationMs int) {
    payload := EventPayloadSpec{
        ID:          generateTransactionID(),
        WorkspaceID: "us-east",
        UniverseID:  "production",
        Type:        "api.request",
        Subject:     "customer:acme",
        Time:        time.Now(),  // When request happened
        Properties: map[string]string{
            "endpoint":    endpoint,
            "duration_ms": strconv.Itoa(durationMs),
        },
    }

    ce := cloudevents.NewEvent()
    ce.SetID(uuid.New().String())
    ce.SetSource("/api/gateway")
    ce.SetType("com.acme.metering.event")
    ce.SetTime(time.Now())  // When sending
    ce.SetData("application/json", payload)

    cloudEventsClient.Send(ctx, ce)
}

// Consumer: Extract EventPayload from CloudEvents
func handleMeteringEvent(ce cloudevents.Event) {
    var payload EventPayloadSpec
    ce.DataAs(&payload)

    // Use EventPayload.Time (business time), not ce.Time() (system time)
    records, _ := Meter(payload, config)
    for _, record := range records {
        store.Save(record)
    }
}
```

### Example 2: Kafka Pipeline

```go
// Producer: Serialize EventPayload to Kafka
func recordAPIRequest(endpoint string, durationMs int) {
    payload := EventPayloadSpec{
        ID:          generateTransactionID(),
        WorkspaceID: "us-east",
        UniverseID:  "production",
        Type:        "api.request",
        Subject:     "customer:acme",
        Time:        time.Now(),  // When request happened
        Properties: map[string]string{
            "endpoint":    endpoint,
            "duration_ms": strconv.Itoa(durationMs),
        },
    }

    data, _ := json.Marshal(payload)
    kafkaProducer.WriteMessages(ctx, kafka.Message{
        Key:   []byte(payload.Subject),  // Partition by subject
        Value: data,                      // EventPayload
        Time:  time.Now(),                // When sending
    })
}

// Consumer: Deserialize EventPayload from Kafka
func handleKafkaMessage(msg kafka.Message) {
    var payload EventPayloadSpec
    json.Unmarshal(msg.Value, &payload)

    // Use EventPayload.Time (business time), not msg.Time (system time)
    records, _ := Meter(payload, config)
    for _, record := range records {
        store.Save(record)
    }
}
```

### Example 3: HTTP API

```go
// POST directly without transport wrapper
func handleMeteringEvent(w http.ResponseWriter, r *http.Request) {
    var payload EventPayloadSpec
    json.NewDecoder(r.Body).Decode(&payload)

    records, err := Meter(payload, config)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    for _, record := range records {
        store.Save(record)
    }

    w.WriteHeader(http.StatusAccepted)
}
```

---

## Conclusion

EventPayload is the **metering domain model**, not a transport format. This separation:
- Clarifies purpose (metering vs transport)
- Enables transport flexibility (CloudEvents, Kafka, HTTP, OpenTelemetry, etc.)
- Maintains correct time semantics (business time vs system time)
- Follows single responsibility principle (one reason to change)
- Serves the next 10 users, not just the first

**When in doubt**: Metering concerns go in EventPayload. Transport concerns go in the transport wrapper.

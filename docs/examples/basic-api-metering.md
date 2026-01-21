# Basic API Metering Example

This example shows the complete metering pipeline with minimal complexity: one event becomes one meter record, then aggregates into one meter reading.

## Scenario

You're tracking API usage for a customer. Each API request generates an event with response time and endpoint information. You want to:
1. Meter the response time in milliseconds
2. Aggregate total response time per day for billing

## Step 1: Event Payload

Your application publishes a usage event when an API request completes:

```json
{
  "id": "evt_abc123",
  "workspaceID": "acme-prod",
  "universeID": "production",
  "type": "api.request",
  "subject": "customer:acme-corp",
  "time": "2024-01-15T10:30:45Z",
  "properties": {
    "endpoint": "/api/v1/users",
    "response_time_ms": "145",
    "status_code": "200",
    "region": "us-east-1"
  }
}
```

**Key fields:**
- `id` - Unique event identifier (idempotency key)
- `workspaceID` - Operational boundary (your production environment)
- `universeID` - Data namespace (production vs test vs simulation)
- `subject` - Who to bill (customer:acme-corp)
- `time` - When the usage occurred (business time)
- `properties` - Flexible key-value pairs specific to this event type

## Step 2: Metering Configuration

You configure how to extract measurements from the event:

```json
{
  "measurements": [
    {
      "sourceProperty": "response_time_ms",
      "unit": "milliseconds"
    }
  ]
}
```

This says: "Extract the `response_time_ms` property and assign it the unit `milliseconds`."

All other properties (`endpoint`, `status_code`, `region`) become **dimensions** for filtering and grouping.

## Step 3: Meter Record (Output of Metering)

The `Meter` function transforms the event using the config:

```json
{
  "id": "rec_4f8a9b2c",
  "workspaceID": "acme-prod",
  "universeID": "production",
  "subject": "customer:acme-corp",
  "recordedAt": "2024-01-15T10:30:45Z",
  "measurement": {
    "quantity": "145",
    "unit": "milliseconds"
  },
  "dimensions": {
    "endpoint": "/api/v1/users",
    "status_code": "200",
    "region": "us-east-1"
  },
  "sourceEventID": "evt_abc123",
  "meteredAt": "2024-01-15T10:30:46Z"
}
```

**What happened:**
- Extracted `response_time_ms` → `measurement.quantity` with `unit: "milliseconds"`
- Passed through remaining properties as `dimensions`
- Generated deterministic `id` from source event ID + unit
- Added `meteredAt` timestamp for watermarking

**Why this matters:** The meter record is now in a standard format for aggregation and billing, regardless of the original event schema.

## Step 4: Aggregation Configuration

You want to sum all milliseconds over a daily billing period:

```json
{
  "aggregation": "sum",
  "window": {
    "start": "2024-01-15T00:00:00Z",
    "end": "2024-01-16T00:00:00Z"
  }
}
```

This defines:
- **aggregation**: How to combine values (`sum`, `max`, `time-weighted-avg`, etc.)
- **window**: Time range for aggregation (half-open interval [start, end))

## Step 5: Meter Reading (Output of Aggregation)

The `Aggregate` function combines meter records in the window:

```json
{
  "id": "reading_xyz789",
  "workspaceID": "acme-prod",
  "universeID": "production",
  "subject": "customer:acme-corp",
  "window": {
    "start": "2024-01-15T00:00:00Z",
    "end": "2024-01-16T00:00:00Z"
  },
  "measurement": {
    "quantity": "14523",
    "unit": "milliseconds"
  },
  "aggregation": "sum",
  "recordCount": 100,
  "createdAt": "2024-01-16T00:01:23Z",
  "maxMeteredAt": "2024-01-15T23:59:58Z"
}
```

**What happened:**
- Summed 100 meter records (each with `unit: "milliseconds"`)
- Total: 14,523 milliseconds of response time for the day
- `recordCount` shows how many individual events contributed
- `maxMeteredAt` tracks latest metering processing time (for incremental aggregation)

**This is your billable usage**: 14,523 milliseconds for customer:acme-corp on January 15th.

## Complete Flow Diagram

```
EventPayload                    MeterRecord                   MeterReading
────────────                    ───────────                   ────────────
{                               {                             {
  properties: {          →        measurement: {       →        measurement: {
    response_time_ms: "145"         quantity: "145"               quantity: "14523"
  }                                 unit: "milliseconds"          unit: "milliseconds"
}                               }                             }
                                dimensions: {                 aggregation: "sum"
                                  endpoint: "/api/users"      recordCount: 100
                                  region: "us-east-1"       }
                                }
                              }

       ↓                               ↓                              ↓
[Meter function]              [Aggregate function]           [Rating/Billing]
+ MeteringConfig              + AggregateConfig
```

## Key Concepts Illustrated

### Flexible Schemas
The event payload uses untyped `properties`. You can add new properties (e.g., `request_body_size`) without changing the spec—just update your metering config.

### Deterministic IDs
If you replay `evt_abc123` with the same metering config, you get the same meter record ID. This enables idempotency: your implementation can check if this ID already exists before creating a new record, preventing double-billing.

### Separation of Concerns
- **Metering**: Extracts and types measurements from events
- **Aggregation**: Combines measurements over time windows
- **Rating**: (Not shown) Multiplies quantities by rates to get prices

### Dimensional Filtering
The `dimensions` in meter records let you:
- Filter: "Show me usage for endpoint=/api/users"
- Segment: "Aggregate separately per region"
- Debug: "Why was this customer billed?"

## Next Steps

- **Conditional metering**: [Extract different measurements based on dimensions](conditional-metering.md)
- **Time-weighted aggregation**: [Meter seat count with gauges](time-weighted-seats.md)
- **Production patterns**: [High-throughput pipeline with real-time alerts](production-patterns.md)
- **Core concepts**: [Understand Workspace, Universe, and Subject](../concepts.md)

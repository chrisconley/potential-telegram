# High-Throughput Metering Pipeline Example

## What This Demonstrates

This example shows a production-grade metering pipeline with **in-flight** and **post-flight** processing stages, similar to patterns used at companies like Twilio for high-volume usage tracking.

**Key Pattern:** At 10k requests per second, you don't want to store 10,000 individual meter records. Instead, you aggregate them in real-time into coarser time windows, with different downstream consumers operating at different granularities.

## Architecture

```
EventPayload (raw API request event)
    ↓ published to bus
MeteringHandler
    ↓ calls Meter() to extract measurements
    ↓ publishes InFlightMeterRecorded events
    ↓
    ├─→ InFlightAggregator (1-second windows)
    │     ├─ Batches records within each second
    │     ├─ Detects tick change (moving to next second)
    │     ├─ Calls Aggregate() on batch
    │     └─ Publishes InFlightMeterRead events
    │           ↓
    │           └─→ RatingHandler
    │                 └─ Accumulates revenue, alerts on threshold
    │
    └─→ PostFlightAggregator (10-second windows)
          ├─ Batches records within each 10 seconds
          ├─ Detects tick change (moving to next 10-second window)
          ├─ Calls Aggregate() on batch
          └─ Publishes PostFlightMeterRead events
                ↓
                └─→ CustomerBalanceHandler
                      └─ Updates customer balances
```

## Key Concepts

### Event Bus Decoupling
All stages communicate via typed events on a pub/sub bus. This allows:
- Multiple independent subscribers to the same events
- No coupling between pipeline stages
- Easy addition of new downstream consumers

### Stateful Subscribers
Aggregators maintain state across events:
- `currentTick`: Which time window we're currently batching for
- `batch`: Records accumulated in the current window

### Tick Detection & Batching
The streaming pattern:
1. **During tick N**: Records arrive → append to batch
2. **First event of tick N+1 arrives**: Detect tick change → flush tick N batch
3. **Aggregate & publish**: Call `Aggregate()` on batched records → publish reading
4. **Continue**: Start batching for tick N+1

This matches production behavior: aggregate continuously, flush on time boundaries.

### In-Flight vs Post-Flight
- **In-flight** (1-second windows): Fast path for real-time metrics, rating, alerts
- **Post-flight** (10-second windows): Storage optimization, billing, analytics

Both subscribe to the same raw `MeterRecorded` events but operate independently with different window sizes.

## Running the Example

```bash
cd metering-spec/internal/examples
go test -v -run TestHighThroughputMeteringPipeline
```

## Example Output

```
Publishing EventPayloads to bus (simulating high throughput)...
  Published 10 events (second 0)
  Published 20 events (second 1)
💰 Threshold reached: accumulated revenue = $0.020 (threshold: $0.020)
  Published 30 events (second 2)
  Published 40 events (second 3)
  ...
  Published 100 events (second 9)
📊 Customer balance update: 100 requests for window 10:00:00 to 10:00:10
  Published 110 events (second 10)
  ...
  Published 200 events (second 19)
📊 Customer balance update: 100 requests for window 10:00:10 to 10:00:20
  Published 210 events (second 20)
  ...
  Published 300 events (second 29)
📊 Customer balance update: 100 requests for window 10:00:20 to 10:00:30

✓ In-flight: 300 EventPayloads → 300 MeterRecords → 30 1-second readings
✓ Rating: Accumulated revenue = $0.300, threshold reached
✓ Post-flight: 300 MeterRecords → 3 10-second readings
```

**What this shows:**
- 300 events over 30 seconds (10 per second)
- InFlightAggregator batches 10 records → 1 reading per second (30 total)
- RatingHandler tracks revenue in real-time, alerts at $0.020
- PostFlightAggregator batches 100 records → 1 reading per 10 seconds (3 total)
- CustomerBalanceHandler prints updates for each 10-second window
- Final revenue: 300 requests × $0.001 = $0.300

## Production Considerations

### What's Simplified in This Example
1. **Partial batch flushing**: We manually call `Flush()` at end. In production, a timer would periodically flush old batches.
2. **Concurrency**: This is single-threaded. Production would use goroutines with proper locking.
3. **Error handling**: We panic on errors. Production needs graceful degradation.
4. **Persistence**: Aggregated readings would be written to a database.

### What Matches Production
1. **Stateful batching**: Maintaining window state and accumulating records
2. **Tick detection**: Using timestamps to detect window boundaries
3. **Event-driven architecture**: Decoupled stages via pub/sub
4. **Domain logic in subscribers**: `Meter()` and `Aggregate()` called by subscribers, not test code

## Code Structure

**Event Types** (`EventPayloadEvent`, `InFlightMeterRecordedEvent`, etc.)
- Wrappers implementing `infra.Event` interface
- Type-safe event routing via enum

**ConfigRepo Pattern**
- Interface for retrieving configs (metering, aggregation, rating)
- Hardcoded implementation for this example
- In production: backed by database or config service

**Handler Structs**
- `MeteringHandler`: Transforms EventPayloads → MeterRecords
- `InFlightAggregator`: Batches MeterRecords → 1-second MeterReadings
- `RatingHandler`: Computes revenue from readings
- `PostFlightAggregator`: Batches MeterRecords → 10-second MeterReadings
- `CustomerBalanceHandler`: Updates balances from readings

Each handler has dependencies (bus, configRepo) and runtime state (batch, accumulators).

# metering-spec

Data contracts and a Go reference implementation for usage metering — the part of billing that answers "how much did each customer use?"

That question is harder than it sounds. Your API emits events with a bag of properties (`input_tokens`, `output_tokens`, `model`, `region`). Some of those properties are quantities you bill for. Others are dimensions you filter by. Different event types need different extraction rules. Some customers are metered by total usage (sum), others by peak concurrent usage (max), others by average seat count weighted across the billing period (time-weighted average). Your test environment needs to run the same pipeline as production without cross-contaminating billing data. And every number has to be exact — no floating-point drift in financial calculations.

This spec defines the data shapes and pure functions for that pipeline: take raw events in, get billable quantities out.

## How it works

The pipeline has two stages. First, **extract** measurements from raw events. Then, **aggregate** those measurements over billing windows.

**A raw usage event** — your application emits these with whatever properties are relevant:
```json
{
  "id": "evt_abc123",
  "type": "llm.completion",
  "subject": "customer:cust_123",
  "time": "2024-01-15T10:30:00Z",
  "properties": {
    "input_tokens": "1250",
    "output_tokens": "340",
    "model": "gpt-4",
    "region": "us-east"
  }
}
```

**An extraction config** — tells the pipeline which properties are quantities and what units to assign:
```json
{
  "observations": [
    { "sourceProperty": "input_tokens", "unit": "input-tokens" },
    { "sourceProperty": "output_tokens", "unit": "output-tokens" }
  ]
}
```

**A metered record** — the extracted quantities (observations) with remaining properties preserved as dimensions:
```json
{
  "id": "evt_abc123",
  "subject": "customer:cust_123",
  "observations": [
    { "quantity": "1250", "unit": "input-tokens", "window": { "start": "...", "end": "..." } },
    { "quantity": "340", "unit": "output-tokens", "window": { "start": "...", "end": "..." } }
  ],
  "dimensions": { "model": "gpt-4", "region": "us-east" }
}
```

**A billable quantity** — observations aggregated over a billing window:
```json
{
  "subject": "customer:cust_123",
  "computedValues": [
    { "quantity": "125000", "unit": "input-tokens", "aggregation": "sum" }
  ],
  "window": { "start": "2024-01-01T00:00:00Z", "end": "2024-02-01T00:00:00Z" },
  "recordCount": 100
}
```

This is where metering-spec's job ends. The reading says "customer cust_123 used 125,000 input-tokens this month." What that costs is a pricing/rating concern handled elsewhere.

Properties that aren't extracted as observations (`model`, `region`) become **dimensions** — available for filtering and grouping downstream.

## Who this is for

You're building a system where:

- **Customers are billed based on what they use** — API calls, tokens, compute hours, storage, seats, or any countable resource
- **Usage events come from multiple sources** with different schemas, and you need a consistent metering layer
- **Aggregation isn't just "sum"** — you need peak usage (max), time-weighted averages (seat count over a month), or latest-value gauges
- **Billing periods matter** — you need to window usage into hourly, daily, or monthly buckets for invoicing
- **Precision matters** — you're doing financial math and can't tolerate floating-point drift

This spec doesn't handle pricing, invoicing, payments, or revenue recognition. It produces the quantities that those systems consume.

## Running it

```bash
git clone https://github.com/chrisconley/potential-telegram.git
cd potential-telegram
go test ./...
```

The Go reference implementation lives in `internal/`. The two core functions:

```go
import (
    "metering-spec/internal"
    "metering-spec/specs"
)

// Stage 1: Meter an event — extract observations from properties
records, err := internal.Meter(eventPayload, meteringConfig)

// Stage 2: Aggregate records — combine over a billing window
reading, err := internal.Aggregate(records, lastBeforeWindow, aggregateConfig)
```

## Aggregation strategies

| Strategy | Use case | Example |
|----------|----------|---------|
| **sum** | Cumulative usage | Total API calls, tokens consumed |
| **max** | Peak usage | Concurrent connections, queue depth |
| **min** | Minimum in window | Lowest price, minimum inventory |
| **latest** | Current state | Most recent gauge reading |
| **time-weighted-avg** | Average over time | Seat count across a billing month |

Time-weighted average treats each observation as a step function — if a customer had 10 seats for 20 days then 15 seats for 10 days, the average is 11.67, not 12.5.

## Multi-tenant isolation

Events are scoped by two dimensions:

- **Workspace** — operational boundary (US region, EU region, a business unit). Each workspace owns its event schemas and metering configs.
- **Universe** — data namespace (production, test, staging, simulation). The same customer ID in different universes is a different billing entity.

This means you can run test data through the same pipeline as production without cross-contamination, or meter the same customer differently in different regions.

See [workspace-universe-isolation.md](design/workspace-universe-isolation.md) for the full design rationale.

## Conditional metering

Extract different observations based on event properties using filters:

```json
{
  "observations": [
    {
      "sourceProperty": "request_count",
      "unit": "premium-requests",
      "filter": { "property": "tier", "equals": "premium" }
    },
    {
      "sourceProperty": "request_count",
      "unit": "standard-requests",
      "filter": { "property": "tier", "equals": "standard" }
    }
  ]
}
```

Same event type, different units based on customer tier. Downstream, premium and standard requests can be rated at different prices.

## Precision

All quantities are decimal strings (`"123.45"`, not `123.45`). No floating-point anywhere in the spec. The Go implementation uses [`cockroachdb/apd`](https://github.com/cockroachdb/apd) for arbitrary-precision decimal arithmetic.

## Repository structure

```
specs/           Data contracts (language-agnostic Go structs with JSON tags)
internal/        Go reference implementation (Meter, Aggregate)
  examples/      Working end-to-end pipeline example
design/          Architecture decision records
docs/examples/   Walkthrough guides
benchmarks/      Performance tests
```

## Documentation

- **[Basic API Metering](docs/examples/basic-api-metering.md)** — step-by-step walkthrough with JSON examples
- **[Workspace-Universe Isolation](design/workspace-universe-isolation.md)** — why two dimensions, not one
- **[Observation Temporal Context](design/observation-temporal-context.md)** — instant vs. time-spanning observations
- **[Aggregation Types](design/aggregation-types.md)** — design rationale for aggregation strategies

## Status

Pre-1.0. The core pipeline (event → meter → aggregate) works and is tested. The data contracts are actively evolving. Expect breaking changes.

## License

[To be determined]

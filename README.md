# metron

Data contracts and a Go reference implementation for metering — the part of billing that answers "how much did each customer use?"

> [!WARNING]
> Pre-1.0. The core pipeline works and is tested. The data contracts are actively evolving. Expect breaking changes.

That question is harder than it sounds. Your API emits events with a bag of properties — `input_tokens`, `output_tokens`, `model`, `region`. Some are quantities you bill for. Others are dimensions you filter by. Different event types need different extraction rules.

Aggregation is where it gets interesting. Some customers are metered by total usage (sum), others by peak concurrent usage (max), others by time-weighted average seat count. If a customer had 10 seats for 20 days then 15 seats for 10 days, the time-weighted average is 11.67, not 12.5. Every number has to be exact: no floating-point drift in financial calculations.

metron is a spec and a reference implementation, not a platform you deploy. It defines data shapes and pure functions for the metering pipeline — events in, billable quantities out — so you can run it on infrastructure you own, or reimplement it in another language.

## Try it

```bash
git clone https://github.com/chrisconley/metron.git
cd metron
go run ./examples/hello
```

```
customer:acme-corp used 11.67 seats (time-weighted-avg) from 2024-01-01 to 2024-01-31
```

That's the 11.67 from above, end-to-end — two gauge events become one billable reading. The shape:

```go
meteringConfig := specs.MeteringConfigSpec{
    Observations: []specs.ObservationExtractionSpec{
        {SourceProperty: "seats", Unit: "seats"},
    },
}

events := []specs.EventPayloadSpec{
    {Subject: "customer:acme-corp", Time: jan1,  Properties: map[string]string{"seats": "10"}},
    {Subject: "customer:acme-corp", Time: jan21, Properties: map[string]string{"seats": "15"}},
}

// Stage 1 — Meter: each event → records.
var records []specs.MeterRecordSpec
for _, event := range events {
    recs, _ := internal.Meter(event, meteringConfig)
    records = append(records, recs...)
}

// Stage 2 — Aggregate: records → one reading over the billing window.
reading, _ := internal.Aggregate(records, nil, specs.AggregateConfigSpec{
    Aggregation: "time-weighted-avg",
    Window:      specs.TimeWindowSpec{Start: jan1, End: feb1},
})
```

Full source: [`examples/hello/main.go`](examples/hello/main.go). The output line above is asserted by a test, so it can't drift from the code.

Going further:

- [`docs/examples/basic-api-metering.md`](docs/examples/basic-api-metering.md) — step-through walkthrough with JSON examples.
- [`internal/examples/`](internal/examples/) — production-style pipeline: 300 events through an event bus, multiple aggregators at different time scales, rating handler firing threshold alerts. Run it: `go test -v ./internal/examples/ -run TestHighThroughputMeteringPipeline`.

## What the pipeline does

A metering config plus an event stream produces billable readings. The pipeline runs in two stages: **Meter** (per event) extracts quantities and preserves dimensions; **Aggregate** (per window) combines records into a reading.

```jsonc
// In: an event with untyped properties
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

// Out: a reading over a billing window
{
  "subject": "customer:cust_123",
  "computedValues": [
    { "quantity": "125000", "unit": "input-tokens", "aggregation": "sum" }
  ],
  "window": { "start": "2024-01-01T00:00:00Z", "end": "2024-02-01T00:00:00Z" },
  "recordCount": 100
}
```

Properties on events are `map[string]string`, not a typed schema. Different products emit different properties without coordinating schema changes. The metering config — not the event schema — decides what's extracted.

The config separates **quantities** (extracted, aggregated, summed/maxed/averaged) from **dimensions** (preserved as-is, for grouping and filtering downstream). Filters route the same event into different units based on tier, region, or any other property: `request_count` becomes `premium-requests` or `standard-requests` depending on which filter matches.

Each pipeline stage broken out in [`docs/examples/basic-api-metering.md`](docs/examples/basic-api-metering.md).

## Scope

metron handles the **quantity pipeline**: raw events in, billable quantities out. It stops at the boundary where quantities become money.

**In scope:** observation extraction, unit assignment, dimensional filtering, time-windowed aggregation, multi-tenant isolation, exact decimal arithmetic.

**Out of scope:** pricing, rating, tiered rates, overage charges, committed-use discounts, credits, rollover, proration across billing periods, invoicing, payments, revenue recognition.

**metron does NOT:**

- compute prices or apply rate cards
- round numbers (rounding belongs at the pricing layer, where business rules live)
- silently truncate or coerce types
- depend on a database or persistence layer (records and readings are pure data; storage is your choice)

The boundary is intentional. Pricing logic depends on contract-level rules — "this customer gets a volume discount above 100k tokens," "roll unused credits into next month" — that change per customer, per negotiation. Those computations consume metering quantities as input but aren't metering themselves. Conflating the two makes both harder to change independently.

The spec preserves temporal context on observations (instant events vs. time-spanning measurements like compute sessions) so downstream consumers have what they need for proration and period assignment. See [observation-temporal-context.md](design/observation-temporal-context.md).

## Who this is for

You're building a system where:

- customers are billed by what they use — API calls, tokens, compute hours, storage, seats, or any countable resource
- usage events come from multiple sources with different schemas, and you need a consistent metering layer
- aggregation isn't just sum — you need peak (max), time-weighted averages, or latest-value gauges
- billing periods matter — usage windows into hourly, daily, or monthly buckets for invoicing
- precision matters — you're doing financial math and can't tolerate floating-point drift

If your billing is flat-rate-only, you don't need a metering pipeline.

## Why not just...

- **... a metering platform like [Lago](https://github.com/getlago/lago), [OpenMeter](https://github.com/openmeterio/openmeter), or Stripe Billing?** Those are platforms you deploy or subscribe to. metron is a spec and a library you embed. Pick a platform if you want the whole metering+pricing stack as a vendor-managed thing. Pick metron if you want to own the pipeline and only need the quantity-layer logic.
- **... shopspring/decimal or apd directly?** metron uses [cockroachdb/apd](https://github.com/cockroachdb/apd) internally, and you could absolutely build metering yourself on top of any decimal library. metron's value is the data shapes (events, configs, records, readings) and the windowing/aggregation logic, not the arithmetic.
- **... int64 cents?** Fine for currency at a fixed scale. Metering produces values like 11.67 seats or 0.0023 GB-hours — quantities that aren't currency, don't have a fixed scale, and need more than two decimal places.

## Why this design

**Configuration-driven, not code-driven.** Extraction rules, filters, units, and aggregation strategies are all configuration. New products and pricing models update a metering config; engineering doesn't touch the pipeline.

**Pricing-agnostic.** The same pipeline serves usage-based billing (token counts), seat-based pricing (time-weighted averages), hybrid models (both), and flat-rate-with-overage (commit thresholds). The metering layer produces quantities; what those cost is downstream.

**One event, one record.** All observations from a single event bundle into one MeterRecord. Persistence is atomic — all observations save together or none do. No partial event data in the pipeline. See [meterrecord-atomicity-analysis.md](design/references/meterrecord-atomicity-analysis.md).

**No schema coordination.** Event properties are untyped strings. Teams add new properties to their events without coordinating across the metering system. The metering config — not the event schema — decides what gets extracted.

**Replay-safe.** Pure functions with deterministic IDs. Reprocessing the same events with the same configuration produces identical records. No hidden state, no side effects, no order dependence.

**Exact arithmetic.** All quantities are decimal strings (`"123.45"`, not `123.45`). No floating-point anywhere. The Go implementation uses [cockroachdb/apd](https://github.com/cockroachdb/apd) for arbitrary-precision decimal arithmetic.

## Aggregation strategies

| Strategy | Use case | Example |
|----------|----------|---------|
| **sum** | Cumulative usage | Total API calls, tokens consumed |
| **max** | Peak usage | Concurrent connections, queue depth |
| **min** | Minimum in window | Lowest price, minimum inventory |
| **latest** | Current state | Most recent gauge reading |
| **time-weighted-avg** | Average over time | Seat count across a billing month |

Time-weighted average treats each observation as a step function. This matters when the value changes mid-period: 10 seats for 20 days then 15 seats for 10 days averages to 11.67, not 12.5 (which is what a naive mean of the two values would give). See [aggregation-types.md](design/aggregation-types.md).

## Multi-tenant isolation

Events are scoped by two dimensions:

- **Workspace** — operational boundary (US region, EU region, a business unit). Each workspace owns its event schemas and metering configs.
- **Universe** — data namespace (production, test, staging, simulation). The same customer ID in different universes is a different billing entity.

You can run test data through the same pipeline as production without cross-contamination, or meter the same customer differently in different regions. See [workspace-universe-isolation.md](design/workspace-universe-isolation.md).

## Repository structure

The Go reference implementation lives in `internal/`. The portable data contracts — language-agnostic Go structs with JSON tags — live in `specs/`, designed to be re-implemented in any language.

```
specs/           Data contracts (language-agnostic Go structs with JSON tags)
internal/        Go reference implementation (Meter, Aggregate)
  examples/      Production-style pipeline example
examples/        Runnable quick-start example
design/          Architecture decision records
docs/examples/   Walkthrough guides
benchmarks/      Performance tests
```

## Design notes

- [Observation temporal context](design/observation-temporal-context.md) — instant events vs. time-spanning measurements
- [Aggregation types](design/aggregation-types.md) — design rationale for sum / max / min / latest / time-weighted-avg
- [Workspace × universe isolation](design/workspace-universe-isolation.md) — why two dimensions, not one
- [MeterRecord atomicity](design/references/meterrecord-atomicity-analysis.md) — one event, one record

## License

MIT — see [LICENSE](LICENSE).

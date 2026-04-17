# metron

Data contracts and a Go reference implementation for metering — the part of billing that answers "how much did each customer use?"

> Pre-1.0. The core pipeline works and is tested. The data contracts are actively evolving. Expect breaking changes.

That question is harder than it sounds. Your API emits events with a bag of properties (`input_tokens`, `output_tokens`, `model`, `region`). Some are quantities you bill for. Others are dimensions you filter by. Different event types need different extraction rules.

Then aggregation gets interesting. Some customers are metered by total usage (sum), others by peak concurrent usage (max), others by average seat count weighted across the billing period. If a customer had 10 seats for 20 days then 15 seats for 10 days, the time-weighted average is 11.67 — not 12.5. Every number has to be exact: no floating-point drift in financial calculations.

This is a spec and a reference implementation, not a platform you deploy. It defines data shapes and pure functions for the metering pipeline: take raw events in, get billable quantities out. You own the infrastructure.

## Try it

```bash
git clone https://github.com/chrisconley/metron.git
cd metron
go run ./examples/hello
```

```
customer:acme-corp used 11.67 seats (time-weighted-avg) from 2024-01-01 to 2024-01-31
```

That's the 11.67 from above, computed end-to-end: two gauge events (10 seats at Jan 1, 15 seats at Jan 21) become one billable reading for January. The whole program is one file:

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cockroachdb/apd/v3"

	"github.com/chrisconley/metron/internal"
	"github.com/chrisconley/metron/specs"
)

func main() {
	// Extract the "seats" property as an observation with unit "seats".
	meteringConfig := specs.MeteringConfigSpec{
		Observations: []specs.ObservationExtractionSpec{
			{SourceProperty: "seats", Unit: "seats"},
		},
	}

	// Two gauge events: 10 seats at Jan 1, then 15 seats at Jan 21.
	events := []specs.EventPayloadSpec{
		{
			ID: "evt_1", Type: "subscription.gauge",
			WorkspaceID: "acme-prod", UniverseID: "production",
			Subject:    "customer:acme-corp",
			Time:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Properties: map[string]string{"seats": "10"},
		},
		{
			ID: "evt_2", Type: "subscription.gauge",
			WorkspaceID: "acme-prod", UniverseID: "production",
			Subject:    "customer:acme-corp",
			Time:       time.Date(2024, 1, 21, 0, 0, 0, 0, time.UTC),
			Properties: map[string]string{"seats": "15"},
		},
	}

	// Stage 1 — Meter: event → records.
	var records []specs.MeterRecordSpec
	for _, event := range events {
		recs, err := internal.Meter(event, meteringConfig)
		if err != nil {
			log.Fatalf("meter: %v", err)
		}
		records = append(records, recs...)
	}

	// Stage 2 — Aggregate: records → one reading over the billing window.
	reading, err := internal.Aggregate(records, nil, specs.AggregateConfigSpec{
		Aggregation: "time-weighted-avg",
		Window: specs.TimeWindowSpec{
			Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		log.Fatalf("aggregate: %v", err)
	}

	// The underlying quantity is exact (11.666…); round to cents for display.
	value := reading.ComputedValues[0]
	fmt.Printf("%s used %s %s (%s) from %s to %s\n",
		reading.Subject, roundCents(value.Quantity), value.Unit, value.Aggregation,
		reading.Window.Start.Format("2006-01-02"), reading.Window.End.Format("2006-01-02"),
	)
}

func roundCents(s string) string {
	var parsed, rounded apd.Decimal
	parsed.SetString(s)
	apd.BaseContext.WithPrecision(34).Quantize(&rounded, &parsed, -2)
	return rounded.String()
}
```

Source: [`examples/hello/main.go`](examples/hello/main.go). The output line is asserted by a test, so it can't drift from the code.

### Going deeper

- **[Basic API metering walkthrough](docs/examples/basic-api-metering.md)** — step through each pipeline stage with JSON examples.
- **Production-style pipeline** — 300 events over 30 seconds through an event bus, multiple aggregators at different time scales, and a rating handler firing threshold alerts. Source: [`internal/examples/`](internal/examples/). Run it: `go test -v ./internal/examples/ -run TestHighThroughputMeteringPipeline`.

The Go reference implementation lives in `internal/`. The portable data contracts — language-agnostic Go structs with JSON tags — live in `specs/`, designed to be re-implemented in any language.

## Why this design

**No code changes for new billing models.** Extraction rules, filters, units, and aggregation strategies are all configuration. When a new product launches or a pricing model changes, you update a metering config — engineering doesn't touch the pipeline.

**Any pricing model.** Usage-based billing needs token counts. Seat-based pricing needs time-weighted seat averages. Hybrid models need both. Even flat-rate plans with overage clauses need to know when a customer exceeds their commitment. The metering layer produces quantities; it doesn't know or care which pricing model consumes them.

**One event, one record.** All observations extracted from a single event are bundled in one MeterRecord. Persistence is atomic — all observations save together or none do. No partial event data in the pipeline. See [meterrecord-atomicity-analysis.md](design/references/meterrecord-atomicity-analysis.md) for the full design rationale.

**No schema coordination.** Event properties are untyped strings. Teams add new properties to their events without coordinating across the metering system. The metering config — not the event schema — decides what gets extracted.

**Replay-safe.** Pure functions with deterministic IDs. Reprocessing the same events with the same configuration produces identical records. No hidden state, no side effects, no order dependence.

**Exact arithmetic.** All quantities are decimal strings (`"123.45"`, not `123.45`). No floating-point anywhere. The Go implementation uses [`cockroachdb/apd`](https://github.com/cockroachdb/apd) for arbitrary-precision decimal arithmetic.

**Test and production in one pipeline.** Workspace × universe scoping isolates data without separate infrastructure. Run test events through the same pipeline as production without cross-contamination.

## What the pipeline does

Events arrive from your application with a bag of untyped string properties. The pipeline transforms them into billable quantities through a series of configurable steps.

### 1. Accept events with untyped properties

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

Properties are `map[string]string`, not a typed schema. This is deliberate. Different products emit different properties without coordinating schema changes across the metering system. The metering config decides what matters — not the event schema.

### 2. Filter by property values

Not every event produces the same observations. A metering config can include filters that match on property values:

```json
{
  "sourceProperty": "request_count",
  "unit": "premium-requests",
  "filter": { "property": "tier", "equals": "premium" }
}
```

This lets you meter the same event type differently based on customer tier, region, product variant, or any other property — through configuration, not code branches.

### 3. Extract quantities and assign units

The config specifies which properties are quantities and what unit to assign:

```json
{
  "observations": [
    { "sourceProperty": "input_tokens", "unit": "input-tokens" },
    { "sourceProperty": "output_tokens", "unit": "output-tokens" }
  ]
}
```

Each extraction parses the string value into an exact decimal, pairs it with a unit, and timestamps it with the event's time. The same source property can map to different units depending on which filter matched — `request_count` becomes `premium-requests` or `standard-requests` based on the customer's tier.

### 4. Preserve remaining properties as dimensions

Properties not extracted as quantities (`model`, `region` in the example above) become **dimensions** on the metered record — preserved for filtering and grouping downstream ("show me token usage broken down by model and region").

The distinction matters: quantities get aggregated (summed, maxed, averaged). Dimensions get preserved (for grouping, filtering, reporting). You don't have to decide up front which properties you'll want to group by later — everything that isn't a quantity is kept.

### 5. Aggregate over billing windows

Observations are combined over a time window using a configured [aggregation strategy](#aggregation-strategies):

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

This is where metron's job ends. The reading says "customer cust_123 used 125,000 input-tokens this month." What that costs is a pricing/rating concern handled elsewhere.

## Who this is for

You're building a system where:

- **Customers are billed based on what they use** — API calls, tokens, compute hours, storage, seats, or any countable resource
- **Usage events come from multiple sources** with different schemas, and you need a consistent metering layer
- **Aggregation isn't just "sum"** — you need peak usage (max), time-weighted averages (seat count over a month), or latest-value gauges
- **Billing periods matter** — you need to window usage into hourly, daily, or monthly buckets for invoicing
- **Precision matters** — you're doing financial math and can't tolerate floating-point drift

## Scope

Metering-spec handles the **quantity pipeline**: raw events in, billable quantities out. It stops at the boundary where quantities become money.

**In scope:** observation extraction, unit assignment, dimensional filtering, time-windowed aggregation, multi-tenant isolation, exact decimal arithmetic.

**Out of scope:** pricing, rating, tiered rates, overage charges, committed-use discounts, credits, rollover, proration across billing periods, invoicing, payments, revenue recognition.

The boundary is intentional. Pricing logic depends on business rules that change per customer, per contract, per negotiation — "this customer gets a volume discount above 100k tokens" or "roll unused credits into next month." Those computations take metering quantities as *input* but aren't metering themselves. Conflating the two makes both harder to change independently.

The spec does preserve temporal context on observations (both instant events and time-spanning measurements like compute sessions) so that downstream consumers have the information they need for proration and period assignment. See [observation-temporal-context.md](design/observation-temporal-context.md) for the design rationale.

## Aggregation strategies

| Strategy | Use case | Example |
|----------|----------|---------|
| **sum** | Cumulative usage | Total API calls, tokens consumed |
| **max** | Peak usage | Concurrent connections, queue depth |
| **min** | Minimum in window | Lowest price, minimum inventory |
| **latest** | Current state | Most recent gauge reading |
| **time-weighted-avg** | Average over time | Seat count across a billing month |

Time-weighted average treats each observation as a step function. This matters when the value changes mid-period: 10 seats for 20 days then 15 seats for 10 days averages to 11.67, not 12.5 (which is what you'd get from a naive mean of the two values). See [aggregation-types.md](design/aggregation-types.md) for the full design rationale.

## Multi-tenant isolation

Events are scoped by two dimensions:

- **Workspace** — operational boundary (US region, EU region, a business unit). Each workspace owns its event schemas and metering configs.
- **Universe** — data namespace (production, test, staging, simulation). The same customer ID in different universes is a different billing entity.

This means you can run test data through the same pipeline as production without cross-contamination, or meter the same customer differently in different regions. See [workspace-universe-isolation.md](design/workspace-universe-isolation.md) for the design rationale.

## Repository structure

```
specs/           Data contracts (language-agnostic Go structs with JSON tags)
internal/        Go reference implementation (Meter, Aggregate)
  examples/      Production-style pipeline example
examples/        Runnable quick-start example
design/          Architecture decision records
docs/examples/   Walkthrough guides
benchmarks/      Performance tests
```

## Documentation

- **[Basic API Metering](docs/examples/basic-api-metering.md)** — step-by-step walkthrough with JSON examples
- **[Workspace-Universe Isolation](design/workspace-universe-isolation.md)** — why two dimensions, not one
- **[Observation Temporal Context](design/observation-temporal-context.md)** — instant vs. time-spanning observations
- **[Aggregation Types](design/aggregation-types.md)** — design rationale for aggregation strategies

## License

MIT — see [LICENSE](LICENSE).

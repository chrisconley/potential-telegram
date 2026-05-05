# metron

A metering specification for usage-based billing systems, with a Go reference implementation. Defines the pipeline from raw events to billable readings: idempotent metering, time-weighted gauge aggregation, two-dimensional tenant isolation. No pricing engine, no invoicing, no payment orchestration.

[![Go Reference](https://pkg.go.dev/badge/github.com/chrisconley/metron.svg)](https://pkg.go.dev/github.com/chrisconley/metron)

> [!WARNING]
> **Pre-1.0.** The data model and the two core operations (`Meter`, `Aggregate`) are stable and exercised by the test suite, but field names and the wire format may still change before `v1.0.0`.

## The problem

Recurring failures in usage-based billing pipelines:

- **A seat-count gauge gets averaged as a flat arithmetic mean.** A customer held 10 seats for 25 days, then 15 for 5; you bill `(10 + 15) / 2 = 12.5` seat-months. The right number is `(10 × 25 + 15 × 5) / 30 ≈ 10.83`. Prometheus's `avg_over_time` does the wrong one — fine for dashboards, wrong for invoices.
- **Replaying yesterday's metering creates duplicate charges.** A bugfix means re-running the pipeline; the rerun emits new record IDs and the same usage gets billed twice.
- **A gauge that didn't change during the window looks like zero usage.** No "seat count changed" events arrived this month, so the aggregator emits zero instead of carrying state forward from the last reading.
- **Test events hit the production billing pipeline.** A staging load test shares the same Kafka topic and the same `customer:cust_123` subject as production; a workspace boundary or a separate "universe" would have stopped it.
- **Aggregations cross units.** One reading ends up summing `tokens` and `compute-hours` because the aggregator only keyed by subject, not by `(subject, unit)`.

`metron` is the data model and pipeline spec that makes those failures structural rather than recurring. It is the typed boundary between raw events arriving and billable usage being computed — not a billing platform, not a usage-tracking SaaS, not a query engine.

## What's in this repo

`metron` is structured as a **specification plus a reference implementation**:

- **[`specs/`](specs/)** — language-agnostic types using only Go primitives (`string`, `time.Time`, `map[string]string`). This is what you'd port to another language. Two function signatures live here: `Meter` and `Aggregate`.
- **[`internal/`](internal/)** — Go reference implementation. Domain-driven (value objects, deterministic IDs, decimal arithmetic via [`cockroachdb/apd`](https://github.com/cockroachdb/apd)). Use it as a Go library, or as a working example when implementing the spec elsewhere.
- **[`examples/`](examples/)** — runnable end-to-end examples covering counter sum, time-weighted gauge, bundled observations (atomicity), conditional metering, and time-spanning observations. See [`examples/README.md`](examples/README.md) for the full index.
- **[`design/`](design/)** — ADRs and reference material, including the [ubiquitous language](design/references/ubiquitous-language.md) and the [observability-vs-metering](design/references/observability-vs-metering.md) study.

A spec without a reference implementation is hard to verify; a reference implementation without a spec is hard to port. This repo ships both, with the boundary made explicit so you can take only the part you need.

## Scope

**In scope:** the event-to-record-to-reading pipeline; observation extraction with optional filters; pass-through dimensions; counter and gauge aggregations (`sum`, `max`, `min`, `latest`, `time-weighted-avg`); deterministic record and reading IDs for idempotent processing; workspace and universe tenant isolation; arbitrary-precision decimal quantities serialized as strings; watermarking for incremental aggregation.

**Out of scope:** rate cards and pricing; invoicing, dunning, and payment orchestration; tax computation; subscription lifecycle; an HTTP or gRPC service; a persistence layer; a query language. `metron` answers "given these events and this config, what is this subject's usage over this window?" — and stops there.

**Refused even in scope** — choices `metron` deliberately doesn't make:

- **No mutable records.** A `MeterRecord` is an immutable historical fact; corrections are new records, not edits.
- **No implicit unit coercion in aggregations.** Records with different units don't combine. A reading is always per-`(subject, unit, window)`.
- **No floating-point quantities.** Every quantity crosses language and storage boundaries as a decimal string.
- **No sampling, downsampling, or expiration.** Auditable data is kept; that's an observability feature, not a billing one.

## Install

```sh
go get github.com/chrisconley/metron
```

Requires Go 1.25 or later.

## Hello world

Two seat-count readings — 10 seats on Jan 1, 15 seats on Jan 21 — billed as a 30-day time-weighted average:

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/chrisconley/metron/internal"
    "github.com/chrisconley/metron/specs"
)

func main() {
    // Extract the "seats" property as an observation with unit "seats".
    // Any other property (region, plan, etc.) flows through as a dimension.
    meteringConfig := specs.MeteringConfigSpec{
        Observations: []specs.ObservationExtractionSpec{
            {SourceProperty: "seats", Unit: "seats"},
        },
    }

    // Two gauge events: 10 seats on Jan 1, then 15 seats on Jan 21.
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

    // Stage 1 — Meter: event → record.
    var records []specs.MeterRecordSpec
    for _, e := range events {
        recs, err := internal.Meter(e, meteringConfig)
        if err != nil {
            log.Fatal(err)
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
        log.Fatal(err)
    }

    v := reading.ComputedValues[0]
    fmt.Printf("%s: %s %s (%s)\n", reading.Subject, v.Quantity, v.Unit, v.Aggregation)
    // customer:acme-corp: 11.66666666666666666666666666666667 seats (time-weighted-avg)
}
```

The customer held 10 seats for 20 days and 15 seats for 10 days. The exact answer is `(10 × 20 + 15 × 10) / 30 = 11.666…`, which `time-weighted-avg` returns at full decimal precision; rounding to invoice cents happens at the display boundary in your decimal library of choice. A naive arithmetic mean of 10 and 15 would give 12.5 — a 7% over-bill that compounds across customers and months.

The full source — including a small rounding helper — is in [`examples/hello/main.go`](examples/hello/main.go); `go run ./examples/hello` runs it.

## Core idea: two operations, three types

```
EventPayload  ── Meter ──▶  MeterRecord  ── Aggregate ──▶  MeterReading
(raw usage)   (+ config)    (typed usage     (+ config,       (one or more
                            with one or       + window)        ComputedValues
                            many                               over the window)
                            Observations)
```

- **`EventPayload`** is the input boundary. A flexible `map[string]string` carries event-type-specific data: `tokens`, `endpoint`, `region`, whatever you publish.
- **`MeterRecord`** is the typed result of metering. It carries one or more `Observation`s — each a `(quantity, unit, window)` triple where the window is instant `[T, T]` for gauges or spanning `[T1, T2]` for activity over a period — plus the dimensions that weren't extracted as observations.
- **`MeterReading`** is the aggregated result over a billing window. It carries one or more `ComputedValue`s — each a `(quantity, unit, aggregation)` triple — so the strategy that produced the value is part of the value.

`Meter` and `Aggregate` are the only two operations. `Meter` is per-event and stateless. `Aggregate` is per-`(subject, unit, window)` and accepts an optional `lastBeforeWindow` record for time-weighted gauges, so a gauge whose state didn't change inside the window still aggregates correctly. Both produce deterministic IDs from their inputs, so replaying yields the same output.

## Where it fits

TODO

## Why this design

**Observations carry temporal context.** Every observation has a `Window`. Instant for gauge readings (`Start == End`), spanning for activity over a period (`Start < End`). Time-spanning events ("8 compute-hours from 8pm to 4am") and instant gauges ("15 seats at 9:47am") are both first-class without special-casing.

**One event, one record, possibly many observations.** An LLM completion event with `input_tokens` and `output_tokens` produces a single `MeterRecord` with two observations bundled inside it. They persist atomically — either both observations land or neither does — and the record's idempotency key is the source event ID. No partial double-billing.

**Aggregations carry their strategy.** `ComputedValue` is `(quantity, unit, aggregation)`, not just `(quantity, unit)`. A reading that says "12.5 seats" is incomplete; one that says "12.5 seats, time-weighted-avg" is auditable. Hierarchical aggregation needs this; so does dispute resolution.

**Two dimensions of isolation.** `WorkspaceID` is the operational boundary — regions, business units, tenants of your platform. `UniverseID` is the data namespace — production vs staging vs simulation. The same `customer:cust_123` in two universes is two distinct billing entities. The model collapses cleanly when you don't need it (`UniverseID = "production"` always) and gives you a real isolation layer when you do.

**Decimal strings for quantities.** A quantity is `"123.456"`, not `float64`. Floating-point representation drifts across languages, precision is implicit, and a metering pipeline that crosses Go, Python, and SQL needs the same value to be the same value at every hop. The reference implementation uses [`cockroachdb/apd`](https://github.com/cockroachdb/apd) internally and never exposes it.

**Watermarking is a first-class field.** Every `MeterRecord` has a `MeteredAt` system timestamp distinct from the business `ObservedAt`. Aggregations track `MaxMeteredAt`, so a downstream system can ask "give me readings whose source records were all metered before T" and get a stable answer. Late-arriving events show up as new records with newer `MeteredAt`, not as silent edits to old ones.

**Deterministic IDs are computed, not generated.** A `MeterRecord` ID is derived from the source event ID; a `MeterReading` ID is derived from `(subject, unit, window, aggregation)`. Re-running the pipeline produces the same IDs. Idempotent ingestion is "insert if not exists," not "deduplicate after the fact."

## Why not just…?

### Why not Prometheus or OpenTelemetry?

Observability systems are designed for operational data: sample loss is acceptable, retention is finite, and `avg_over_time` is arithmetic mean. Billing data is auditable: every event must be retained, every aggregation must be exact, and a seat-count gauge needs step-interpolated time weighting. Prometheus's docs say it directly:

> "If you need 100% accuracy, such as for per-request billing, Prometheus is not a good choice."

The taxonomy translates well — counter vs gauge, dimensions, cardinality concerns — but the storage and aggregation guarantees don't. Observability for monitoring; a metering spec for billing. The full mapping is in [`design/references/observability-vs-metering.md`](design/references/observability-vs-metering.md).

### Why not [OpenMeter](https://github.com/openmeterio/openmeter), [Lago](https://github.com/getlago/lago), or [Kill Bill](https://github.com/killbill/killbill)?

Those are full billing platforms — metering, pricing, invoicing, subscriptions, dunning, payment orchestration. `metron` is the data-model layer that sits underneath. If you want to operate a billing system, pick a platform. If you're building one — or you have an existing system and you want a portable, language-agnostic shape for the metering layer — `metron` is the size of one component inside that platform.

### Why not roll your own?

Rolling your own metering is the default and a reasonable place to start. After a few iterations, the same pieces show up in every implementation: time-weighted aggregation, deterministic record IDs, watermarking, a way to keep test events from billing customers, a place to put dimensions. `metron` is that converged shape, with the design rationale traced in [`design/`](design/) so you can adapt it instead of re-deriving it.

## How it compares

**[OpenMeter](https://github.com/openmeterio/openmeter)** — Apache-2.0, Kafka-based real-time aggregation, AI/API-billing focus, ships with Stripe sync. The closest spiritual peer; OpenMeter is the deployable system, `metron` is the data model. Pick OpenMeter if you want to run metering as a service today; reach for the metron spec when you need a portable shape that isn't tied to a specific deployment.

**[Lago](https://github.com/getlago/lago)** — open-source usage-based billing platform with subscription management. One layer above; Lago is what you'd build *with* a metering layer like this, plus pricing, invoicing, and payment.

**[Kill Bill](https://github.com/killbill/killbill)** — long-running JVM billing platform. Same shape as Lago; broader scope, more mature in enterprise contexts.

**[CloudEvents](https://github.com/cloudevents/spec)** — event envelope spec. Compatible. CloudEvents tells you how to wrap an event for transport; `metron` tells you how to turn an event into a meter record.

**Prometheus / OpenTelemetry** — observability. See above.

## Concepts at a glance

| Term | What it is |
|---|---|
| `EventPayload` | Raw usage activity at the system boundary. Untyped properties map. |
| `MeterRecord` | Typed result of metering. Carries `Observation`s and pass-through `Dimensions`. |
| `MeterReading` | Aggregated result over a window. Carries one or more `ComputedValue`s. |
| `Observation` | `(quantity, unit, window)`. Window is instant `[T, T]` or spanning `[T1, T2]`. |
| `ComputedValue` | `(quantity, unit, aggregation)`. The aggregation strategy is part of the value. |
| `Subject` | The billing entity, formatted `"type:id"` (e.g. `"customer:cust_123"`). |
| `Workspace` | Operational boundary. Owns event schemas and metering configs. |
| `Universe` | Data namespace within a workspace. Scopes subject identity. |
| `Aggregation` | One of `sum`, `max`, `min`, `latest`, `time-weighted-avg`. |
| `MeteringConfig` | What to extract from each event, with optional filters. |
| `AggregateConfig` | Aggregation function + half-open `[Start, End)` window. |

The full vocabulary, with the design rationale for each term, lives in [`design/references/ubiquitous-language.md`](design/references/ubiquitous-language.md).

## Repository layout

```
specs/                  Language-agnostic spec (primitives only)
  eventpayload.go       EventPayloadSpec
  meterrecord.go        MeterRecordSpec
  meterreading.go       MeterReadingSpec, TimeWindowSpec
  observation.go        ObservationSpec, ComputedValueSpec
  meteringconfig.go     MeteringConfigSpec, ObservationExtractionSpec, FilterSpec
  aggregate.go          AggregateConfigSpec, Aggregate signature
  meter.go              Meter signature

internal/               Go reference implementation
  metering.go           Meter — event → records
  aggregation.go        Aggregate — records → reading
  meterrecord.go        MeterRecord domain object
  meterreading.go       MeterReading domain object
  decimal.go            Decimal value object (apd-backed)
  ...

examples/               Runnable end-to-end examples — see examples/README.md
  hello/                  time-weighted gauge (the README quick-start)
  api-calls/              counter sum
  llm-tokens/             bundled observations, atomicity
  conditional-tier/       conditional metering with filters
  compute-session/        time-spanning observations
benchmarks/             Pipeline benchmarks
docs/                   Walkthroughs, examples, FAQ
design/                 ADRs and reference material
```

## Development

```sh
go test ./...                      # unit + integration tests
go run ./examples/hello            # run the hello-world example
go test ./benchmarks/...           # benchmark suite
```

## Contributing

Issues and PRs welcome. Spec changes should ground in a use case the reference implementation can demonstrate. Reference-implementation changes should keep the `specs/` layer dependency-free of `apd` or any other decimal library — that boundary is what makes the spec portable.

## License

MIT. See [LICENSE](LICENSE).

# Metering Specification

**Experimental prototype** for usage metering and billing aggregation.

This is a work-in-progress exploration of data contracts for metering pipelines. Expect breaking changes as we iterate based on real-world feedback.

## What is this?

This spec defines data contracts for transforming raw usage events into billable meter readings. It provides a complete pipeline from event ingestion through metering to time-windowed aggregation, with built-in support for multi-tenancy, test isolation, and flexible attribution models.

## Why does this exist?

Building usage-based billing systems requires solving the same problems repeatedly:
- **Flexible event schemas** - Different products need different properties without coordinating type changes
- **Multi-tenant isolation** - Regional workspaces with global customers, test/prod separation, post-merger integration
- **Conditional metering** - Extract different measurements based on event properties
- **Time-windowed aggregation** - Sum, max, time-weighted-average over billing periods
- **Precision** - Exact decimal arithmetic for financial calculations

This spec explores a data model for these problems, with a Go reference implementation.

## Quick Example

**Input:** Raw usage event
```json
{
  "id": "evt_abc123",
  "workspaceID": "acme-prod",
  "universeID": "production",
  "type": "api.request",
  "subject": "customer:cust_123",
  "time": "2024-01-15T10:30:00Z",
  "properties": {
    "endpoint": "/api/users",
    "response_time_ms": "145",
    "region": "us-east"
  }
}
```

**Metering config:** Extract milliseconds as a measurement
```json
{
  "measurements": [{
    "sourceProperty": "response_time_ms",
    "unit": "milliseconds"
  }]
}
```

**Output:** Metered usage record
```json
{
  "id": "rec_xyz789",
  "subject": "customer:cust_123",
  "measurement": {
    "quantity": "145",
    "unit": "milliseconds"
  },
  "dimensions": {
    "endpoint": "/api/users",
    "region": "us-east"
  },
  "recordedAt": "2024-01-15T10:30:00Z"
}
```

**Aggregation:** Sum over billing period
```json
{
  "subject": "customer:cust_123",
  "measurement": {
    "quantity": "14523",
    "unit": "milliseconds"
  },
  "aggregation": "sum",
  "window": {
    "start": "2024-01-15T00:00:00Z",
    "end": "2024-01-16T00:00:00Z"
  },
  "recordCount": 100
}
```

## Key Features

### Two-Dimensional Isolation

**Workspace × Universe** model enables:
- Multi-region with global customers (different schemas per region, shared customer identity)
- Test/staging/production separation (isolate test data from production billing)
- What-if scenarios and simulations (parallel universes for pricing experiments)
- Post-merger integration (namespace legacy systems without ID collisions)

See [workspace-universe-isolation.md](design/workspace-universe-isolation.md) for design rationale.

### Flexible Event Schemas

Events use untyped `properties` maps, allowing each workspace to define custom schemas without coordinating type system changes. Metering configs extract typed measurements at processing time.

### Conditional Metering

Extract different measurements based on event properties. For example, meter requests differently based on customer tier or region.

### Aggregation Strategies

- **sum** - Total usage (API calls, tokens consumed)
- **max** - Peak usage (concurrent connections, queue depth)
- **min** - Minimum value in window
- **latest** - Most recent value by timestamp
- **time-weighted-avg** - Average weighted by duration (seat count, resource allocation)

### Language Interoperability

All specs use primitives-only types (strings, numbers, time) with JSON serialization. Includes Go reference implementation in `internal/`.

### Precision and Financial Calculations

Quantities in metering-spec are represented as **decimal strings** (e.g., `"123.45"`) to avoid floating-point precision issues in JSON serialization.

For **billing and financial use cases**, implementations SHOULD:
- Use exact decimal arithmetic (no floating point)
- Implement consistent rounding rules (recommend: banker's rounding / half-to-even)
- Ensure reproducible calculations (same inputs → same outputs)
- Guarantee allocation totals when splitting values (sum of parts = original whole)

**Go implementations:** See [meters/shared/precision](https://github.com/chrisconley/meters/tree/main/shared/precision) (not yet public) for a production-ready reference implementation that provides financial-grade precision.

**Other languages:** Consider decimal libraries appropriate for financial calculations:
- Python: `decimal.Decimal` (stdlib)
- JavaScript: `bignumber.js`, `decimal.js`
- Java: `java.math.BigDecimal`
- Ruby: `BigDecimal` (stdlib)

## Documentation

- **[Basic Example](docs/examples/basic-api-metering.md)** - Simple walkthrough with JSON examples
- **[Architecture](docs/architecture.md)** - Pipeline stages and data flow
- **[API Reference](specs/)** - Inline documentation for all data types
- **[Core Concepts](docs/concepts.md)** - Workspace, Universe, Subject, Measurements, Aggregations
- **[Production Patterns](docs/examples/production-patterns.md)** - High-throughput metering pipeline
- **[Design Rationale](docs/design/)** - Architecture decision records

## Repository Structure

```
metering-spec/
├── specs/           # Primitives-only data contracts (language-agnostic)
│   ├── eventpayload.go
│   ├── meterrecord.go
│   ├── meterreading.go
│   └── ...
├── internal/        # Go reference implementation
│   ├── meter.go     # EventPayload → MeterRecord transformation
│   ├── aggregate.go # MeterRecord → MeterReading aggregation
│   └── examples/    # Working code examples
└── docs/            # Conceptual documentation and guides
```

## Status

**Experimental prototype.** This spec is actively evolving based on design principles and real-world requirements. The core data model is being refined through [Architecture Decision Records](design/) that guide implementation work.

**Current state:**
- [design/](design/) contains working ADRs that define the target design
- [GitHub Issues](https://github.com/chrisconley/potential-telegram/issues) track active implementation work

Expect breaking changes. Nothing is stable yet.

If you're implementing this, please share feedback on what works and what doesn't.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on proposing changes to the spec.

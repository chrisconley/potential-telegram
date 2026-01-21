# Observability vs. Metering: Terminology, Patterns, and Key Differences

**Date:** 2026-01-21
**Context:** Understanding how observability systems (Prometheus, OpenTelemetry) differ from metering/billing systems, and what patterns translate

---

## Problem Statement

Many engineers approach metering from an observability background (Prometheus, Grafana, DataDog, etc.). While these systems share some concepts with metering, they have **fundamentally different requirements** that lead to different architectures and semantics.

Understanding these differences is critical for:
- **Adopting useful terminology** (metric types, semantic conventions)
- **Avoiding wrong assumptions** (e.g., "just use Prometheus for billing")
- **Learning from existing patterns** while respecting the different problem space

---

## The Fundamental Distinction: Auditable vs. Operational Data

Industry consensus divides telemetry data into two categories:

### Auditable Data
- **Loss tolerance:** None. Every record must be captured and retained.
- **Use case:** Financial transactions, billing events, compliance logs
- **Retention:** Complete historical records required (disputes, audits, tax)
- **Accuracy requirement:** 100% - exact, verifiable calculations
- **Examples:** Billing events, transaction logs, replication logs

### Operational Data
- **Loss tolerance:** Some data loss acceptable for cost efficiency
- **Use case:** Infrastructure monitoring, performance insights, alerting
- **Retention:** Can downsample or expire old data
- **Accuracy requirement:** "Good enough" - trends and patterns matter more than exact values
- **Examples:** Infrastructure metrics, observability data, performance monitoring

**From Prometheus documentation:**
> "If you need 100% accuracy, such as for per-request billing, Prometheus is not a good choice as the collected data will likely not be detailed and complete enough."

**Metering systems are auditable. Observability systems are operational.**

---

## Terminology Translation: What Observability Terms Mean

### Metric Types (Prometheus & OpenTelemetry)

Observability systems classify metrics into distinct types based on their semantic behavior:

#### Counter
- **Semantics:** Monotonically increasing value (only goes up, resets to zero on restart)
- **Examples:** Total HTTP requests, total bytes sent, total errors
- **Query pattern:** Use `rate()` or `increase()` to get change over time
- **Mapping to metering:** Counter → Event aggregations (`sum-events`, `max-event`)
- **Key difference:** Observability counters track cumulative totals; metering counters track discrete events

#### Gauge
- **Semantics:** Point-in-time value that can go up or down
- **Examples:** Current memory usage, concurrent connections, queue depth, active seats
- **Query pattern:** Use `avg_over_time()`, `max_over_time()`, `min_over_time()`
- **Mapping to metering:** Gauge → State aggregations (`time-weighted-avg`, `peak-state`, `final-state`)
- **Key difference:** Observability gauges are sampled periodically; metering gauges require state reconstruction

#### Histogram
- **Semantics:** Bucketed observations for distributions (e.g., request latency p95, p99)
- **Storage:** Creates multiple time series with `_bucket`, `_sum`, `_count` suffixes
- **Examples:** Request duration, response size
- **Query pattern:** Use `histogram_quantile()` to compute percentiles
- **Mapping to metering:** Less common; might use for analyzing event value distributions
- **Innovation:** Prometheus v2.40+ has native histograms with dynamic buckets (much more efficient)

#### Summary
- **Semantics:** Similar to histogram but calculates quantiles client-side over sliding windows
- **Examples:** Request latency quantiles
- **Mapping to metering:** Rarely used in billing

### Time Series Database Concepts

#### Time Series
- **Definition:** Stream of timestamped values identified by metric name + labels
- **Example:** `http_requests_total{method="GET", status="200"}` → [(t1, 100), (t2, 150), ...]
- **Mapping to metering:** Similar to MeterRecord stream for a (customer, unit) pair

#### Sample / Data Point
- **Definition:** Individual measurement at a specific timestamp
- **Mapping to metering:** Similar to MeterRecord (but observability samples can be dropped)

#### Labels / Dimensions
- **Definition:** Key-value pairs attached to metrics for filtering and grouping
- **Example:** `{service="api", region="us-east", customer="acme"}`
- **Mapping to metering:** Similar to Event.Dimensions, but observability has cardinality constraints

#### Cardinality
- **Definition:** Number of unique label combination → number of time series
- **Problem:** Cardinality explosion can crash the database
- **Example:** 50 services × 200 pods × 20 endpoints × 5 status codes × 30 histogram buckets = **30 million time series**
- **Mapping to metering:** Same concern! High-cardinality dimensions (transaction IDs) must be handled carefully
- **Solution pattern:** Drop high-cardinality labels, use recording rules to pre-aggregate, or use histograms

### Aggregation Terminology

#### Recording Rules
- **Definition:** Pre-computed aggregations stored as new time series
- **Purpose:** Reduce query cost by pre-aggregating common queries
- **Mapping to metering:** Similar to MeterReading (pre-aggregated from MeterRecords)

#### Rollups / Downsampling
- **Definition:** Reduced-resolution data for long-term storage
- **Example:** Keep 1-minute samples for 7 days, 5-minute samples for 30 days, 1-hour samples forever
- **Mapping to metering:** NOT applicable - metering requires exact historical records

#### Aggregation Operators (PromQL)
- `sum()` - Sum across dimensions
- `avg()` - Average across dimensions
- `max()` / `min()` - Maximum/minimum across dimensions
- `rate()` - Per-second rate of change (for counters)
- `increase()` - Total increase over time range (for counters)
- `avg_over_time()` - Arithmetic mean of samples in time range (NOT time-weighted!)

---

## The Three Pillars of Observability

Modern observability is built on three types of telemetry:

### 1. Metrics
- **What:** Numerical measurements over time
- **Format:** Time series (timestamp + value + labels)
- **Use case:** Alerting, dashboards, performance monitoring
- **Tools:** Prometheus, Grafana, DataDog

### 2. Traces
- **What:** Request flows through distributed systems
- **Format:** Spans (with parent-child relationships) forming traces
- **Use case:** Understanding request paths, latency attribution, dependency mapping
- **Tools:** Jaeger, Zipkin, Tempo
- **Standard:** OpenTelemetry

### 3. Logs
- **What:** Text event records
- **Format:** Structured (JSON) or unstructured text
- **Use case:** Debugging, forensics, compliance
- **Tools:** Elasticsearch, Loki, Splunk

**Emerging 4th Pillar: Profiling**
- Continuous profiling (CPU, memory flamegraphs)
- Tools: Pyroscope, Parca

**Relevance to metering:** Metering is primarily **metrics** (numerical usage over time), with some overlap with **logs** (audit trails).

---

## Time-Weighted Averages: The Critical Difference

This is where observability and metering diverge most sharply.

### The Problem: Irregularly-Sampled Gauge Data

**Scenario:** Seat-based pricing
- Jan 1: Customer has 10 seats
- Feb 16: Customer adds 5 seats (now 15)
- Billing period: Feb 1-28 (28 days)
- **Question:** What's the average seat count for February?

### How Observability Systems Handle This

**Prometheus `avg_over_time()`:**
- Takes all samples in the time range
- Calculates arithmetic mean: `(10 + 15) / 2 = 12.5 seats`
- **Problem:** This assumes both values existed for equal duration, which is wrong

**From documentation:**
> "All values in the specified interval have the same weight in the aggregation even if the values are not equally spaced throughout the interval."

**Result:** `avg_over_time()` is NOT a true time-weighted average.

### How Metering Systems Should Handle This

**True Time-Weighted Average (Step Interpolation):**

Algorithm:
1. Each gauge value holds its state until the next reading
2. Compute: `value × duration_it_persisted`
3. Sum all weighted values
4. Divide by total window duration

**Calculation:**
- 10 seats for 15 days (Feb 1-15)
- 15 seats for 13 days (Feb 16-28)
- Average: `(10 × 15 + 15 × 13) / 28 = 12.32 seats`

**Correct answer: 12.32 seats** (not 12.5)

### Your Implementation

From `metering/meterreading.go:329-425`:

```go
// timeWeightedAvgRecords computes the time-weighted average of gauge readings.
// Uses step interpolation: each value holds until the next reading (or window end).
//
// Algorithm:
//  1. Combine lastBeforeWindow (if exists) + recordsInWindow
//  2. Sort by timestamp
//  3. For each reading, compute: value × duration_until_next_reading
//  4. Sum weighted values and divide by total window duration
```

**Key insight:** The `lastBeforeWindow` parameter carries forward state from before the billing period, which is essential for accurate gauge aggregation.

### Systems That Do Support Time-Weighted Averages

**Azure Data Explorer:**
> "IoT devices sending data commonly emits metric values in an asynchronous way, only upon change, to conserve bandwidth. In that case we need to calculate Time Weighted Average (TWA), taking into consideration the exact timestamp and duration of each value inside the time bin."

**TimescaleDB:**
- Has explicit `time_weight()` function in timescaledb-toolkit
- Designed for irregularly-sampled gauge data

**TigerData article:**
> "Time-weighted averages are a way to get an unbiased average when you are working with irregularly sampled data. Time-weighted averages are essential for accurately analyzing irregularly-sampled time series data where the time between samples varies."

### Why Observability Systems Don't Do This

1. **Performance:** Step interpolation requires sorting + weighted sum. Fine for billing (hundreds of events), terrible for metrics (millions of samples).

2. **Lossy by design:** Prometheus can drop samples during network issues. Time-weighted avg requires complete history.

3. **Different use case:** They care about "what's happening now?" not "what was the exact average over a billing period?"

4. **Workaround exists:** Push samples at regular intervals to approximate with `avg_over_time()` (wastes bandwidth, not auditable).

---

## Architecture Patterns

### Collection Models

#### Pull Model (Prometheus)
- Server actively "scrapes" `/metrics` endpoints at regular intervals (15s-60s)
- Targets discovered via service discovery
- **Pros:** Centralized control, simple client
- **Cons:** Requires network access to all targets, can miss short-lived processes

#### Push Model (StatsD, OpenTelemetry)
- Applications push metrics to collector
- Batching for efficiency
- **Pros:** Works behind firewalls, captures ephemeral processes
- **Cons:** Requires client buffering, backpressure handling

**Metering mapping:** Metering is push-based (events arrive via HTTP/Kafka). Pull makes no sense for billing events.

### Storage Architecture

#### Time Series Database (TSDB)
- **Optimized for:** Append-heavy writes, numeric samples, time-range queries
- **Storage:** Columnar compression, efficient for time-series data
- **Example:** Prometheus TSDB, InfluxDB, TimescaleDB, VictoriaMetrics

**Key characteristics:**
- Write-optimized (not update-heavy)
- Label-indexed queries
- Retention policies (automatic expiration)

**Metering consideration:** Similar append-only pattern, but CANNOT expire data. Immutability + full retention required.

---

## Patterns to Adopt from Observability

### ✅ 1. Metric Type Taxonomy

**Adopt:** Explicit distinction between counters and gauges

**Your implementation:**
- Counter aggregations: `sum-events`, `max-event`, `min-event`, `latest-event`
- Gauge aggregations: `time-weighted-avg`, `peak-state`, `min-state`, `final-state`

**From `metering-spec/docs/aggregation-types.md`:**
- Aggregation names encode semantic operation (not just implementation)
- Self-documenting, type-safe by design
- Mirrors OpenTelemetry's Counter vs UpDownCounter

### ✅ 2. Dimensional/Label Model

**Adopt:** Key-value labels for filtering and grouping

**Mapping:**
- Observability: `http_requests{service="api", region="us-east"}`
- Metering: `Event.Dimensions map[string]string`

**Benefit:** Flexible querying without schema changes

**Caution:** Cardinality management critical in both systems

### ✅ 3. Semantic Conventions (OpenTelemetry Style)

**Adopt:** Standardized naming patterns and metadata

**OpenTelemetry examples:**
- Instrument types: Counter, UpDownCounter, Gauge, Histogram
- Naming: `<namespace>.<entity>.<metric>` (e.g., `http.server.request.duration`)
- Common attributes: `service.name`, `http.method`, `db.system`
- Units in metadata, not names

**Your implementation:**
- Unit system: `precision.Measure[D]` with typed units
- Workspace-specific schemas: `IngestionConfig` per (workspace, event_type)
- Properties → Measures/Dimensions transformation

### ✅ 4. Cardinality Awareness

**Adopt:** Design for high-cardinality reality, but manage it

**Observability lesson:**
- High-cardinality labels (user IDs, transaction IDs) can explode database
- Solution: Aggregate at ingestion, use recording rules, drop labels

**Metering application:**
- MeterRecord has `(customer, unit)` as key dimensions → bounded cardinality
- Event has `Properties map[string]string` → could have high-cardinality values
- `lastBeforeWindow` pattern reduces storage (don't keep full gauge history)

---

## Patterns to Avoid from Observability

### ❌ 1. Approximate Aggregations

**Observability:** Sampling, downsampling, quantile approximations acceptable

**Metering:** Must be exact. Every event counted, no sampling.

### ❌ 2. Data Loss Tolerance

**Observability:** "Some samples dropped during network blip" → fine for dashboards

**Metering:** "Some events dropped" → incorrect invoice → customer dispute or revenue loss → NOT acceptable

### ❌ 3. Eventual Consistency Without Guarantees

**Observability:** Metric eventually shows up in TSDB, might be delayed

**Metering:** Need strong guarantees on when readings are complete (watermarking, idempotency)

### ❌ 4. Missing Idempotency/Deduplication

**Observability:** Duplicate samples less critical (averaged out in queries)

**Metering:** Duplicate events → double-billing → unacceptable. Need idempotent ingestion.

### ❌ 5. Automatic Data Expiration

**Observability:** Retention policies delete old data (cost management)

**Metering:** Historical data required for audits, disputes, compliance. Never expire.

---

## Existing Open Source Metering Solutions

Several open source projects bridge the gap between observability and billing:

### OpenMeter
- **License:** Apache 2.0
- **Architecture:** Built on Apache Kafka for real-time event aggregation
- **Focus:** AI/API/DevOps usage-based billing
- **Approach:** Push events with duration, aggregate into time windows
- **Integration:** Syncs metering data with Stripe for billing
- **Pattern:** Point-in-time events with `duration_seconds` in payload

**Example event:**
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

### Lago
- **License:** Open source (YC-backed)
- **Focus:** Usage-based billing with subscription management
- **Features:** Pay-as-you-go, hybrid pricing, consumption tracking
- **Deployment:** Self-hosted or managed cloud
- **Approach:** Metering API + pricing engine + payment orchestration

### Flexprice
- **License:** Open source, self-hostable
- **Focus:** Usage-based pricing for developers
- **Features:** Realtime metering, credits/top-ups, feature access control
- **Benefit:** No vendor lock-in, run on own infrastructure

### UniBee
- **Focus:** Billing and payment management
- **Features:** Meter & rating engine, webhook-first architecture
- **Pricing models:** Tiered, usage-based, metered, flat-rate

### Key Observation

All these systems provide **higher-level products** (billing platforms) rather than **specifications**.

**Your project's differentiator:** Building a metering specification with domain-driven design, event-driven architecture, and principled abstractions (Measure types, aggregation semantics, etc.).

---

## Semantic Mapping: Observability → Metering

| Observability Concept | Metering Equivalent | Notes |
|-----------------------|---------------------|-------|
| Counter metric | Event aggregation (`sum-events`) | Discrete events, not cumulative totals |
| Gauge metric | State aggregation (`time-weighted-avg`) | Requires state reconstruction |
| Histogram | Rare in billing | Might analyze event value distributions |
| Time series | MeterRecord stream per (customer, unit) | But with 100% retention |
| Sample/data point | MeterRecord | Must never be dropped |
| Labels/dimensions | Event.Dimensions | Cardinality still matters |
| Scrape interval | Event arrival time | Push-based, not pull |
| Recording rule | MeterReading | Pre-aggregated, but auditable |
| Cardinality | Same concept | Design with care |
| `avg_over_time()` | `time-weighted-avg` | Different implementation! |
| PromQL | Query API (future) | TBD in spec |
| Retention policy | Infinite retention | Never expire |

---

## When to Use Observability Systems vs. Metering Systems

### Use Observability (Prometheus, DataDog, etc.) for:
- Infrastructure monitoring
- Application performance monitoring (APM)
- Real-time alerting
- Dashboards for operational insights
- Debugging production issues
- Service health checks

### Use Metering Systems for:
- Usage-based billing
- Subscription seat tracking
- API quota enforcement
- Revenue recognition
- Customer-facing invoices
- Regulatory compliance (SOX, GDPR billing data)

### Can You Use Both?

**Yes, common pattern:**
1. **Metering system** → Single source of truth for billing
2. **Observability system** → Monitoring, alerting, operational dashboards
3. **Integration:** Export billing metrics to observability for monitoring (e.g., "revenue by customer" dashboard)

**DO NOT:** Use Prometheus as your metering database (data loss risk, no auditability)

**DO:** Push aggregated billing metrics to Prometheus for operational monitoring

---

## Key Takeaways

1. **Observability ≠ Metering**
   - Auditable vs. operational data
   - 100% accuracy vs. "good enough"
   - Prometheus explicitly says: not for billing

2. **Time-weighted averages are critical**
   - Observability: `avg_over_time()` is arithmetic mean (not time-weighted)
   - Metering: Must use step interpolation for gauge aggregations
   - Your implementation is correct; most observability systems are not

3. **Adopt useful patterns**
   - Metric type taxonomy (counter vs gauge)
   - Dimensional/label model
   - Semantic conventions
   - Cardinality awareness

4. **Avoid wrong assumptions**
   - Cannot tolerate data loss
   - Cannot expire historical data
   - Cannot use approximate aggregations
   - Must have idempotency/deduplication

5. **Existing OSS metering solutions**
   - OpenMeter, Lago, Flexprice, UniBee exist as products
   - Your spec takes a different approach: principled design, domain-driven patterns

---

## References

### Observability Systems & Documentation

- [Prometheus: Metric types](https://prometheus.io/docs/concepts/metric_types/)
- [Prometheus: Understanding metric types](https://prometheus.io/docs/tutorials/understanding_metric_types/)
- [Prometheus: Query functions](https://prometheus.io/docs/prometheus/latest/querying/functions/)
- [OpenTelemetry: Observability primer](https://opentelemetry.io/docs/concepts/observability-primer/)
- [OpenTelemetry: Semantic Conventions](https://opentelemetry.io/docs/concepts/semantic-conventions/)
- [OpenTelemetry: Metrics semantic conventions](https://opentelemetry.io/docs/specs/semconv/general/metrics/)
- [IBM: Three Pillars of Observability](https://www.ibm.com/think/insights/observability-pillars)
- [Prometheus Cheat Sheet - Aggregation Over Time](https://iximiuz.com/en/posts/prometheus-functions-agg-over-time/)

### Time-Weighted Averages

- [Azure Data Explorer: Time Weighted Average](https://techcommunity.microsoft.com/blog/azuredataexplorer/time-weighted-average-and-value-in-azure-data-explorer/4257933)
- [TimescaleDB: Time-Weighted Averages](https://deepwiki.com/timescale/timescaledb-toolkit/3.3.1-time-weighted-averages)
- [TigerData: What Time-Weighted Averages Are and Why You Should Care](https://www.tigerdata.com/blog/what-time-weighted-averages-are-and-why-you-should-care)

### Cardinality & TSDB Architecture

- [Last9: How to Manage High Cardinality Metrics in Prometheus](https://last9.io/blog/how-to-manage-high-cardinality-metrics-in-prometheus/)
- [Prometheus TSDB Explained: How It Works, Scales & Optimizes](https://www.groundcover.com/learn/observability/prometheus-tsdb)
- [Chronosphere: Aggregating Prometheus Timeseries with M3](https://chronosphere.io/learn/aggregating-millions-of-prometheus-timeseries-with-m3/)

### Metering vs. Observability

- [OpenMeter: Metering - How to Choose the Right Approach](https://openmeter.io/blog/metering-how-to-choose-the-right-approach)

### Open Source Metering Solutions

- [OpenMeter](https://openmeter.io/)
- [OpenMeter GitHub](https://github.com/openmeterio/openmeter)
- [Lago](https://www.getlago.com/)
- [Lago GitHub](https://github.com/getlago/lago)
- [Flexprice GitHub](https://github.com/flexprice/flexprice)
- [UniBee](https://unibee.dev/)
- [TechCrunch: OpenMeter makes it easier for companies to track usage-based billing](https://techcrunch.com/2024/03/12/openmeter-makes-it-easier-for-companies-to-track-usage-based-billing/)

### Internal Documentation

- `metering-spec/docs/aggregation-types.md` - Counter vs gauge aggregation semantics
- `metering/aggregationtype.go` - Aggregation type implementation
- `metering/meterreading.go:329-425` - Time-weighted average implementation
- `arch/reference/chris-design-principles.md` - Design principles applied

---

## Conclusion

**Observability systems teach us valuable patterns** - metric type taxonomy, dimensional models, semantic conventions, and cardinality awareness. However, they are architected for **operational insights, not financial accuracy**.

**Metering requires stronger guarantees**: zero data loss, exact aggregations, complete audit trails, and idempotent processing. Time-weighted averages exemplify this: `avg_over_time()` is fine for dashboards, but wrong for billing.

**Your metering spec benefits from understanding both domains**: adopting observability's clean abstractions while maintaining the rigorous correctness that billing demands.

When building metering systems, **learn from Prometheus, but don't use it as your database**.

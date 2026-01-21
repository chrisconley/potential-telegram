# Event Data Sizing, Performance, and Cost Analysis

**Date:** 2026-01-21
**Context:** Understanding how to calculate event data size, make informed type selection decisions (int64 vs UUID vs string), and evaluate cloud storage costs for metering systems

---

## Problem Statement

When designing a metering system that processes high-throughput events (10k+ req/sec), critical questions arise:

1. **Type Selection**: Should WorkspaceID be `int64`, `UUID`, `string`, or something else?
2. **Size Calculation**: How do we calculate the actual byte cost of different field types?
3. **Performance Impact**: What are the real-world costs at scale with AWS/GCP?
4. **Design Decisions**: How do we make data-driven choices about data types?

This document provides:
- Industry patterns from observability platforms and cloud services
- Technical analysis of data type sizes across different contexts
- Cost calculations with current cloud provider pricing
- Recommendations for type selection in metering systems

---

## Industry Patterns: What Leading Platforms Use

### OpenTelemetry (Industry Standard for Observability)

**All IDs are strings:**
- Resource attributes: `string` key-value pairs
- Trace IDs: String representation of byte sequences
- Span IDs: String representation
- Event names: Strings

**Rationale:**
- Cross-language compatibility (works in Go, Python, Java, JavaScript, etc.)
- Flexibility for different ID generation schemes (UUID, ULID, timestamp-based, Snowflake IDs)
- JSON/protobuf serialization compatibility

**Cardinality Warning:**
> High-cardinality dimensions (like customer IDs, transaction IDs) can create millions of time series and crash observability databases.

**Key Design Principle:**
Strings provide maximum flexibility at the cost of size. Manage cardinality through aggregation, not by restricting types.

**References:**
- [OpenTelemetry Metrics Data Model](https://opentelemetry.io/docs/specs/otel/metrics/data-model/)
- [OpenTelemetry Logs Data Model](https://opentelemetry.io/docs/specs/otel/logs/data-model/)

---

### AWS CloudWatch

**Dimensions are name/value pairs as strings:**
- All dimension names and values: ASCII strings
- Max 30 dimensions per metric
- Each unique dimension combination creates a new metric variation

**Best Practices (from AWS documentation):**
> "Don't overdo it on dimensions. Use fewer dimensions to avoid unnecessary time series and cost optimization."

**Cardinality Impact:**
Each unique combination of dimension values creates a separate time series. This directly affects:
- Storage costs
- Query performance
- Index size

**Example Cardinality Explosion:**
- 50 services × 200 pods × 20 endpoints × 5 status codes = **30 million time series**

**References:**
- [AWS CloudWatch Metrics Concepts](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/cloudwatch_concepts.html)
- [AWS CloudWatch Dimension API](https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_Dimension.html)

---

### Prometheus (Explicit Anti-Pattern for Billing)

**Labels are all string key-value pairs:**
- Metric names: Strings
- Label names: Strings
- Label values: Strings

**Official Guidance on Billing:**
> "If you need 100% accuracy, such as for per-request billing, Prometheus is not a good choice as the collected data will likely not be detailed and complete enough."

**Why This Matters:**
Prometheus (and most observability systems) are designed for **operational insights**, not **financial accuracy**. See `docs/references/observability-vs-metering.md` for detailed comparison.

**Key Takeaway:**
Observability platforms prioritize flexibility (strings) over size optimization because:
1. Aggregation happens at query time, not storage time
2. Data loss is acceptable for operational monitoring
3. Sampling and downsampling reduce storage costs

**References:**
- [Prometheus: When NOT to use it](https://prometheus.io/docs/introduction/faq/#when-should-i-not-use-prometheus)

---

### Usage-Based Billing Platforms

#### Stripe Metered Billing
- **Customer IDs**: Strings (e.g., `"cust_abc123"`)
- **Event tagging**: Real-time events tagged with customer ID (string), timestamp, metadata
- **ID format**: Flexible string format for external system integration

#### OpenMeter, Lago, UniBee
- **Customer IDs**: Strings for maximum flexibility
- **Throughput**: Systems process millions of events per second
- **Accuracy requirement**: 99%+ through deduplication and audit trails
- **Event format**: CloudEvents spec with string IDs

**Common Pattern:**
All billing platforms use **strings for IDs** to:
- Support external system integration (users bring their own IDs)
- Avoid ID collision across merged systems
- Enable human-readable debugging

**References:**
- [Stripe Metered Billing](https://stripe.com/resources/more/what-is-metered-billing-heres-how-this-adaptable-billing-model-works)
- [Usage-Based Billing Implementation 2026](https://www.zenskar.com/blog/usage-based-billing)
- [OpenMeter Architecture](https://openmeter.io/)

---

### DataDog Observability Pipelines

**Performance Benchmarks:**
- **Throughput**: ~1 TB per vCPU per day (12 processors transforming data)
- **Memory**: ≥2 GiB per vCPU minimum
- **Architecture**: X86_64 offers best performance
- **Scaling**: Auto-scaling recommended to handle traffic spikes without data loss

**Key Insight:**
Performance is achieved through **pipeline optimization and aggregation**, not by minimizing individual event size.

**References:**
- [DataDog Best Practices for Scaling Observability Pipelines](https://docs.datadoghq.com/observability_pipelines/best_practices_for_scaling_observability_pipelines/)

---

## Technical Analysis: String Size Across Contexts

Understanding the **actual byte cost** of strings requires analyzing different contexts: in-memory (Go), wire format (JSON/protobuf), and storage (database).

### Go In-Memory Representation

#### String Header Overhead

Every Go string, even `""`, has a **16-byte overhead**:
```go
type stringStruct struct {
    str unsafe.Pointer  // 8 bytes - pointer to underlying byte array
    len int             // 8 bytes - length of string
}
```

**Examples:**
```go
var empty string = ""                                  // 16 bytes (header only)
var short string = "ws_123"                            // 16 + 6 = 22 bytes
var uuid string = "550e8400-e29b-41d4-a716-446655440000"  // 16 + 36 = 52 bytes
```

**Compared to int64:**
```go
var id int64 = 12345  // 8 bytes (no header overhead)
```

**Savings with int64:**
- vs empty string: 16 - 8 = **8 bytes saved**
- vs `"ws_123"`: 22 - 8 = **14 bytes saved**
- vs UUID: 52 - 8 = **44 bytes saved**

#### Heap Allocation Size Classes

Go's allocator uses size classes: 8, 16, 24, 32, 48, 64, 80, 96 bytes...

**Impact:**
- A 20-byte string (16 header + 4 data) rounds up to 24-byte allocation
- A 50-byte string (16 header + 34 data) rounds up to 64-byte allocation

**Memory efficiency depends on string length relative to size class boundaries.**

**References:**
- [How String Works in Golang](https://perennialsky.medium.com/how-string-works-in-golang-7ac4d797164b)
- [[]byte vs string in Go](https://syslog.ravelin.com/byte-vs-string-in-go-d645b67ca7ff)

---

### JSON Wire Format

#### String Encoding
JSON strings require quotes plus escaping for special characters.

**Examples:**
- Empty string: `""` = **2 bytes**
- Single char: `"a"` = **3 bytes**
- UUID: `"550e8400-e29b-41d4-a716-446655440000"` = **38 bytes** (36 chars + 2 quotes)

**Numbers:**
- Integer: `12345` = **5 bytes** (no quotes)
- Large int64: `9223372036854775807` = **19 bytes**

**Field Name Overhead:**
Each field adds: `"fieldName":` = field name length + 3 bytes (quotes + colon)

**Example EventPayload JSON:**
```json
{
  "id": "evt_123",
  "workspaceID": "ws_a1b2c3d4",
  "universeID": "prod",
  "type": "api.request",
  "subject": "customer:cust_abc123",
  "time": "2024-01-01T10:00:00Z",
  "properties": {"endpoint": "/api/users", "tokens": "1500"}
}
```

**Size breakdown:**
- Field names: `"id"`, `"workspaceID"`, etc. = ~80 bytes
- Field values: ~150 bytes
- JSON structure (`{}`, `:`, `,`): ~20 bytes
- **Total: ~250 bytes**

**With int64 WorkspaceID:**
```json
{"workspaceID": 123}      // 17 bytes (field name + colon + number)
```

**With string WorkspaceID:**
```json
{"workspaceID": "ws_123"} // 26 bytes (field name + colon + quoted string)
```

**Difference: 9 bytes per event for this field**

---

### Protocol Buffers Binary Format

**Empty strings are omitted:**
- Empty string field: **0 bytes** (not encoded if field supports presence)
- Default values: **0 bytes** (proto3 default behavior)

**Non-empty string encoding:**
```
tag (1 byte for field numbers 1-15) +
varint_length (1 byte for strings <128 bytes) +
data (UTF-8 bytes)
```

**Examples:**
```protobuf
// Field 1, string value "test" (4 bytes)
// Wire format: 0x0A 0x04 't' 'e' 's' 't' = 6 bytes total

// Field 2, string value "550e8400-e29b-41d4-a716-446655440000" (36 bytes)
// Wire format: 0x12 0x24 <36 bytes> = 38 bytes total
```

**int64 encoding:**
```protobuf
// Field 1, int64 value 12345
// Wire format with varint: 0x08 0xB9 0x60 = 3 bytes total

// Field 1, int64 max value 9223372036854775807
// Wire format: 0x08 + 9 bytes varint = 10 bytes total
```

**Protobuf is significantly more efficient than JSON** for both strings and numbers.

**References:**
- [Protocol Buffers Encoding](https://protobuf.dev/programming-guides/encoding/)
- [How Protobuf Works—The Art of Data Encoding](https://victoriametrics.com/blog/go-protobuf/)

---

### Database Storage

#### PostgreSQL VARCHAR

**Storage format:**
- Short strings (<127 bytes): **1 byte length prefix** + data
- Longer strings (≥127 bytes): **4 byte length prefix** + data

**Examples:**
```sql
CREATE TABLE events (
  workspace_id VARCHAR(36)  -- Max 36 chars for UUID
);
```

**Storage costs:**
- Empty string: 1 byte
- "ws_123" (6 chars): 1 + 6 = **7 bytes**
- UUID (36 chars): 1 + 36 = **37 bytes**

**int64 (BIGINT) storage:**
```sql
CREATE TABLE events (
  workspace_id BIGINT  -- Always 8 bytes
);
```

**Storage: Fixed 8 bytes**

**Key Difference:**
- VARCHAR is variable-length (more efficient for short strings)
- BIGINT is fixed-length (predictable, no length prefix overhead)

#### MySQL VARCHAR

**Length prefix:**
- VARCHAR(255): 1 byte prefix
- VARCHAR(256+): 2 byte prefix

**Storage:**
- Similar to PostgreSQL but with different threshold

#### DynamoDB Item Size

**Item size calculation:**
- Attribute name length (UTF-8 bytes)
- Attribute value:
  - String: UTF-8 bytes
  - Number: Stored as string representation + metadata (~8 bytes)

**Example:**
```json
{
  "WorkspaceID": "ws_123"  // "WorkspaceID" (11 bytes) + "ws_123" (6 bytes) = 17 bytes
}
```

**With int64:**
```json
{
  "WorkspaceID": 123       // "WorkspaceID" (11 bytes) + number (~8 bytes) = 19 bytes
}
```

**Surprising result:** DynamoDB numbers aren't smaller than short strings due to metadata overhead!

---

## Practical Size Calculation: EventPayloadSpec

From `metering-spec/specs/eventpayload.go`:

```go
type EventPayloadSpec struct {
    ID          string            `json:"id"`
    WorkspaceID string            `json:"workspaceID"`
    UniverseID  string            `json:"universeID"`
    Type        string            `json:"type"`
    Subject     string            `json:"subject"`
    Time        time.Time         `json:"time"`
    Properties  map[string]string `json:"properties,omitempty"`
}
```

### Minimal Event (All Empty Strings)

**Go memory:**
```go
EventPayloadSpec{
    ID:          "",  // 16 bytes (header)
    WorkspaceID: "",  // 16 bytes
    UniverseID:  "",  // 16 bytes
    Type:        "",  // 16 bytes
    Subject:     "",  // 16 bytes
    Time:        time.Time{},  // 24 bytes (3 x int64: seconds, nanoseconds, location)
    Properties:  nil,  // 48 bytes (map header: pointer, length, flags)
}
```
**Total: ~152 bytes** (even with all empty strings)

**JSON:**
```json
{"id":"","workspaceID":"","universeID":"","type":"","subject":"","time":"0001-01-01T00:00:00Z"}
```
**Total: ~96 bytes** (field names + quotes + separators)

---

### Realistic Event with UUID WorkspaceID

```go
EventPayloadSpec{
    ID:          "550e8400-e29b-41d4-a716-446655440000",  // 16 + 36 = 52 bytes
    WorkspaceID: "ws_a1b2c3d4",                           // 16 + 11 = 27 bytes
    UniverseID:  "prod",                                  // 16 + 4 = 20 bytes
    Type:        "api.request",                           // 16 + 11 = 27 bytes
    Subject:     "customer:cust_abc123",                  // 16 + 18 = 34 bytes
    Time:        time.Now(),                              // 24 bytes
    Properties: map[string]string{
        "endpoint": "/api/users",  // (16+8) + (16+10) = 50 bytes
        "tokens": "1500",          // (16+6) + (16+4) = 42 bytes
    },  // Map header: 48 bytes + 2 entries (92 bytes) = 140 bytes
}
```

**Go memory: ~326 bytes**
**JSON: ~200 bytes**

---

### Realistic Event with int64 WorkspaceID (Hypothetical)

```go
type EventPayloadSpecWithInt64 struct {
    ID          string
    WorkspaceID int64  // Changed to int64
    UniverseID  string
    Type        string
    Subject     string
    Time        time.Time
    Properties  map[string]string
}

EventPayloadSpecWithInt64{
    WorkspaceID: 12345,  // 8 bytes (no header)
    // ... other fields same as above
}
```

**Go memory: ~307 bytes** (saved 19 bytes: 27 - 8)
**JSON: ~191 bytes** (saved 9 bytes from quoted string to number)

**Savings: ~6% of total event size**

---

## Scale Impact: 10k Events/Second

### Daily Event Volume

**Assumptions:**
- **Throughput**: 10,000 events/second
- **Uptime**: 24/7 (86,400 seconds/day)
- **Daily events**: 10,000 × 86,400 = **864,000,000 events/day** (864 million)

### WorkspaceID Field Size Impact

#### UUID String WorkspaceID (36 characters)
- **Go memory per field**: 16 + 36 = **52 bytes**
- **JSON per field**: 38 bytes (36 chars + 2 quotes)
- **Daily volume (Go memory)**: 864M × 52 bytes = **44.9 GB/day**
- **Monthly volume**: 44.9 × 30 = **~1,347 GB/month** = **1.35 TB/month**

#### Short String WorkspaceID ("ws_12345" = 8 characters)
- **Go memory per field**: 16 + 8 = **24 bytes**
- **Daily volume**: 864M × 24 bytes = **20.7 GB/day**
- **Monthly volume**: **~621 GB/month**

#### int64 WorkspaceID
- **Go memory per field**: **8 bytes**
- **JSON per field**: 5-19 bytes (depending on number)
- **Daily volume (Go memory)**: 864M × 8 bytes = **6.9 GB/day**
- **Monthly volume**: **~207 GB/month**

### Size Difference Summary

| WorkspaceID Type | Bytes/Event | Daily (GB) | Monthly (GB) | vs int64 Difference |
|------------------|-------------|------------|--------------|---------------------|
| UUID (36 chars)  | 52          | 44.9       | 1,347        | +1,140 GB/month     |
| Short string (8) | 24          | 20.7       | 621          | +414 GB/month       |
| int64            | 8           | 6.9        | 207          | Baseline            |

**Key Finding:** UUID vs int64 = **38 GB/day** = **~1,140 GB/month** = **~1.14 TB/month** difference

---

## Cloud Storage Cost Analysis

Using the **1.14 TB/month difference** between UUID strings and int64, let's calculate actual costs with current AWS/GCP pricing.

### 1. Object Storage (Long-term Event Archival)

Best for: Immutable event storage for compliance, auditing, dispute resolution.

#### AWS S3 Standard
- **Price**: $0.023/GB/month (first 50 TB)
- **Monthly cost**: 1,140 GB × $0.023 = **$26.22/month**
- **Annual cost**: **$314.64/year**

#### AWS S3 Intelligent-Tiering (Recommended)
- **Frequent Access tier**: $0.023/GB/month
- **Infrequent Access tier**: $0.0125/GB/month (after 30 days)
- **Archive tier**: $0.004/GB/month (after 90 days)
- **Estimated cost** (after auto-tiering): **$10-15/month**
- **Annual cost**: **$120-180/year**

**Why Intelligent-Tiering:**
Metering events are frequently accessed for billing cycles, then rarely accessed (disputes, audits). Auto-tiering optimizes costs.

#### GCP Cloud Storage Standard
- **Price**: $0.020/GB/month (North America region)
- **Monthly cost**: 1,140 GB × $0.020 = **$22.80/month**
- **Annual cost**: **$273.60/year**

#### GCP Nearline (30-day access SLA)
- **Price**: $0.010/GB/month
- **Retrieval fee**: $0.01/GB
- **Monthly cost**: 1,140 GB × $0.010 = **$11.40/month**
- **Annual cost**: **$136.80/year**

**Verdict for Object Storage:**
**$250-350/year difference** between UUID and int64. Negligible for most businesses.

**References:**
- [AWS S3 Pricing](https://aws.amazon.com/s3/pricing/)
- [AWS S3 Pricing Guide 2026](https://cloudchipr.com/blog/amazon-s3-pricing-explained)
- [GCP Cloud Storage Pricing](https://cloud.google.com/storage/pricing/)

---

### 2. Relational Database Storage (PostgreSQL on RDS)

Best for: Queryable events, complex joins, ACID transactions.

#### AWS RDS General Purpose SSD (gp3)

**Storage pricing:**
- **Price**: $0.115/GB/month (up to 64 TiB)
- **Monthly cost**: 1,140 GB × $0.115 = **$131.10/month**
- **Annual cost**: **$1,573.20/year**

**IOPS pricing (critical for 10k writes/sec):**
- **Baseline**: 3,000 IOPS included with storage
- **Required**: ~10,000 writes/sec = ~10,000 IOPS
- **Additional IOPS**: 7,000 IOPS needed
- **IOPS cost**: $0.02/IOPS/month
- **Monthly IOPS cost**: 7,000 × $0.02 = **$140/month**

**Total monthly cost (storage + IOPS):**
- **Storage**: $131.10/month
- **IOPS**: $140/month
- **Total**: **$271.10/month** = **$3,253.20/year**

**Verdict for RDS:**
**$3,200/year difference** for storage+IOPS. More significant, but still small compared to compute costs.

**Key Insight:**
IOPS costs often exceed storage costs for high-throughput workloads. The UUID vs int64 decision matters more for index size and query performance than raw storage.

**References:**
- [AWS RDS PostgreSQL Pricing](https://aws.amazon.com/rds/postgresql/pricing/)
- [AWS RDS Pricing Breakdown 2026](https://sedai.io/blog/understanding-amazon-rds-costs-pricing)

---

### 3. NoSQL Database (DynamoDB)

Best for: High-throughput writes, flexible schema, serverless scaling.

#### AWS DynamoDB Standard Table

**Storage pricing:**
- **Price**: $0.25/GB/month
- **Monthly storage cost**: 1,140 GB × $0.25 = **$285/month**
- **Annual storage cost**: **$3,420/year**

**Write pricing (the real cost driver):**
- **On-demand writes**: $1.25 per million WRUs
- **Daily events**: 864,000,000 events
- **Assuming 1 KB per event**: 864M WRUs/day
- **Daily write cost**: 864M × $1.25/1M = **$1,080/day**
- **Monthly write cost**: $1,080 × 30 = **$32,400/month**

**Total monthly cost:**
- **Storage**: $285/month
- **Writes**: $32,400/month
- **Total**: **$32,685/month** = **$392,220/year**

**Key Insight:**
The 1,140 GB storage difference ($285/month) is **0.87% of total cost**. Write costs dominate (99.1% of total).

**Verdict for DynamoDB:**
Storage size is **essentially irrelevant**. Optimize for:
1. Write efficiency (batch writes, smaller items)
2. Data modeling (single-table design to reduce queries)
3. Access patterns (avoid expensive scans)

**References:**
- [AWS DynamoDB Pricing](https://aws.amazon.com/dynamodb/pricing/)
- [DynamoDB Pricing Guide](https://www.nops.io/blog/amazon-dynamodb-pricing/)

---

### 4. Time-Series Database (AWS Timestream)

Best for: Time-series queries, automatic data lifecycle management.

#### Timestream for InfluxDB (Memory Store)
- **Price**: $0.036/GB/hour = $25.92/GB/month
- **Monthly cost** (if all hot): 1,140 GB × $25.92 = **$29,548/month** 😱
- **Annual cost**: **$354,576/year**

**Why so expensive?**
Memory store is optimized for high-speed recent data queries. Not intended for long-term storage.

#### Timestream Magnetic Store (Cold Storage)
- **Price**: $0.03/GB/month
- **Monthly cost**: 1,140 GB × $0.03 = **$34.20/month**
- **Annual cost**: **$410.40/year**

**Lifecycle Policy:**
Move data from memory store (expensive, fast) to magnetic store (cheap, slower queries) after 1-30 days.

**Verdict for Timestream:**
Use magnetic store for long-term event storage. UUID vs int64 costs **$410/year** difference.

**References:**
- [AWS Timestream Pricing](https://aws.amazon.com/timestream/pricing/)

---

### Cost Summary Table

| Storage Type | Monthly Cost | Annual Cost | Use Case |
|--------------|--------------|-------------|----------|
| **S3 Standard** | $26.22 | $314.64 | Long-term archival |
| **S3 Intelligent-Tiering** | $10-15 | $120-180 | Auto-optimized archival (recommended) |
| **GCP Standard** | $22.80 | $273.60 | Long-term archival |
| **GCP Nearline** | $11.40 | $136.80 | Infrequent access |
| **RDS PostgreSQL (storage)** | $131.10 | $1,573.20 | Queryable events |
| **RDS PostgreSQL (storage+IOPS)** | $271.10 | $3,253.20 | High-write queryable events |
| **DynamoDB (storage only)** | $285.00 | $3,420.00 | NoSQL events |
| **DynamoDB (storage+writes)** | $32,685.00 | $392,220.00 | High-throughput NoSQL |
| **Timestream Magnetic** | $34.20 | $410.40 | Time-series cold storage |

---

## The Aggregation Game-Changer

**Critical Context from Your Architecture:**

From `metering-spec/internal/examples/inflightpostflight_README.md`:
- Production pattern: **100:1 aggregation ratio**
- 300 events → 30 1-second readings → 3 10-second readings
- **Result**: 100x compression from raw events to final storage

### Recalculating Costs with Aggregation

**Raw event volume:**
- 864M events/day × 52 bytes (UUID WorkspaceID) = 44.9 GB/day

**After 100:1 aggregation:**
- 8.64M aggregated readings/day × 52 bytes = **449 MB/day**

**Monthly difference (UUID vs int64):**
- Raw: 1,140 GB/month
- Aggregated: **11.4 GB/month** (100x smaller)

### Aggregated Storage Costs

#### S3 Standard
- **Monthly cost**: 11.4 GB × $0.023 = **$0.26/month**
- **Annual cost**: **$3.12/year**

#### RDS PostgreSQL (gp3)
- **Storage**: 11.4 GB × $0.115 = **$1.31/month**
- **Annual cost**: **$15.72/year**

#### DynamoDB
- **Storage**: 11.4 GB × $0.25 = **$2.85/month**
- **Annual cost**: **$34.20/year**

**Verdict:**
With aggregation, the UUID vs int64 storage cost difference is **essentially zero** ($3-35/year).

**Key Architectural Insight:**

Your metering system's performance comes from **aggregation**, not per-event optimization. The design correctly prioritizes:
1. ✅ **Flexibility** (strings for cross-system compatibility)
2. ✅ **Aggregation** (100:1 compression in MeterReadings)
3. ✅ **Cardinality management** (bounded by workspaces, not events)

Optimizing individual event size would be **premature optimization** given the aggregation strategy.

---

## Recommendations

### For WorkspaceID: Use `string` Type

**Recommended:**
```go
type WorkspaceID string  // Flexible, max 36 chars (UUID-compatible)
```

**Rationale:**

1. ✅ **Matches industry standard**
   - OpenTelemetry, AWS CloudWatch, DataDog all use strings
   - Billing platforms (Stripe, Lago, OpenMeter) use strings
   - Cross-platform compatibility (JSON, protobuf, SQL, NoSQL all handle strings natively)

2. ✅ **Flexibility for integration**
   - Supports UUID, ULID, Snowflake IDs, external system IDs
   - No schema migration needed when ID format changes
   - Works with external billing systems that provide their own IDs

3. ✅ **Aggregation makes raw size irrelevant**
   - 100:1 compression reduces cost difference to $3-35/year
   - Storage cost is negligible compared to compute/IOPS costs

4. ✅ **Cardinality is bounded**
   - Number of workspaces << number of events
   - Cardinality determined by business growth, not event volume
   - Index size and query performance more important than field size

5. ✅ **Future-proof**
   - Can adopt new ID schemes without breaking changes
   - Supports multi-tenancy with external ID providers
   - Enables merger/acquisition scenarios (avoid ID collisions)

**Cost Impact:**
- **Archival (S3)**: $120-315/year difference
- **Database (RDS)**: $1,500-3,200/year difference
- **High-throughput (DynamoDB)**: 0.87% of total cost
- **After aggregation**: $3-35/year difference

For a metering system, **flexibility is worth $300/year**.

---

### When int64 Makes Sense

**Use int64 WorkspaceID if:**

```go
type WorkspaceID int64  // Sequential or Snowflake ID
```

**Conditions:**
1. ✅ You control all ID generation (no external systems)
2. ✅ You'll never exceed 2^63 (9.2 quintillion) workspaces
3. ✅ You're storing raw, uncompressed events in expensive databases for years
4. ✅ You need optimal index performance for high-cardinality joins
5. ✅ Every byte counts (IoT devices with cellular data costs)

**Benefits:**
- 8 bytes vs 52 bytes (UUID) = **44 bytes saved per event**
- Faster integer comparisons and joins
- Smaller indexes (BTree depth, cache efficiency)
- Predictable size (no varchar overhead)

**Trade-offs:**
- ❌ Cannot support external system IDs (UUIDs, etc.)
- ❌ Schema migration required if ID format changes
- ❌ Less human-readable in logs/debugging
- ❌ Potential ID collision in multi-tenant scenarios

---

### Hybrid Approach: Constrained String

**Alternative:**
```go
type WorkspaceID string  // Max 16 chars, base62 encoded int64
```

**Pattern:**
- Use base62 encoding: `[a-zA-Z0-9]`
- 16 chars can encode up to 62^16 = ~48 bits of entropy
- Example: `"aBcDeFgHiJkLmN"` represents int64 internally

**Benefits:**
- ✅ String flexibility for external systems
- ✅ Shorter than UUID (16 vs 36 bytes)
- ✅ Human-readable (no special chars)
- ✅ Sortable if designed correctly (e.g., ULID format)

**Example formats:**
- ULID: `01ARZ3NDEKTSV4RRFFQ69G5FAV` (26 chars, timestamp + randomness)
- Base62: `7Nrvaqo0yJFU` (12 chars, 64-bit space)
- Nano ID: `V1StGXR8_Z5jdHi6B-myT` (21 chars, URL-safe)

**Cost impact:**
- 16 + 16 = 32 bytes (vs 52 for UUID, vs 8 for int64)
- Saves **20 bytes per event** compared to UUID
- Still more flexible than int64

---

### Monitoring and Optimization

**Establish baselines:**

1. **Event size distribution**
   - Measure p50, p95, p99 event sizes in production
   - Identify outliers (huge Properties maps)
   - Set size limits (e.g., 10 KB max event size)

2. **Cardinality tracking**
   - Monitor unique WorkspaceIDs per day/month
   - Alert on cardinality explosions (bug in ID generation)
   - Track Properties cardinality (high-cardinality dimensions)

3. **Storage growth**
   - Track daily/monthly storage growth rates
   - Compare raw events vs aggregated readings
   - Verify 100:1 aggregation ratio is maintained

4. **Query performance**
   - Monitor index size growth
   - Track query latency by cardinality
   - Optimize hot paths (billing period queries)

**Build a sizing calculator:**

```go
// Package sizing provides event data size calculation utilities
package sizing

type SizeCalculator struct {
    // Configuration for different contexts
}

type SizeBreakdown struct {
    GoMemory       int  // In-memory struct size
    JSONWireFormat int  // Serialized JSON size
    ProtobufFormat int  // Serialized protobuf size
    PostgresStorage int // Database row size estimate
}

func (c *SizeCalculator) EventPayloadSize(e EventPayloadSpec) SizeBreakdown {
    breakdown := SizeBreakdown{}

    // Go memory calculation
    breakdown.GoMemory =
        16 + len(e.ID) +           // String header + data
        16 + len(e.WorkspaceID) +
        16 + len(e.UniverseID) +
        16 + len(e.Type) +
        16 + len(e.Subject) +
        24 +                       // time.Time
        48 + c.mapSize(e.Properties)  // Map header + entries

    // JSON wire format
    breakdown.JSONWireFormat = c.jsonSize(e)

    // Protobuf (if implemented)
    breakdown.ProtobufFormat = c.protobufSize(e)

    // PostgreSQL (1-byte length prefix + data for varchar)
    breakdown.PostgresStorage = c.postgresSize(e)

    return breakdown
}

func (c *SizeCalculator) mapSize(m map[string]string) int {
    if m == nil {
        return 0
    }
    size := 48  // Map header
    for k, v := range m {
        size += (16 + len(k)) + (16 + len(v))  // Each entry
    }
    return size
}
```

**Usage:**
```go
calc := sizing.NewSizeCalculator()
event := EventPayloadSpec{
    WorkspaceID: "ws_123",
    // ... other fields
}
breakdown := calc.EventPayloadSize(event)
fmt.Printf("Go memory: %d bytes, JSON: %d bytes\n",
    breakdown.GoMemory, breakdown.JSONWireFormat)
```

This calculator helps:
- Document size impact of design decisions
- Identify size optimization opportunities
- Validate assumptions about storage costs
- Guide type selection with real data

---

## Key Takeaways

### 1. Industry Standard: Strings Win for IDs

Every major platform (OpenTelemetry, AWS, GCP, DataDog, Stripe) uses **strings for IDs** because:
- Cross-language compatibility
- Flexibility for different ID schemes
- Integration with external systems
- Future-proofing against schema changes

**Metering systems should follow this pattern.**

---

### 2. Context Matters: Size Varies Dramatically

The "size" of a string depends on context:
- **Go memory**: 16-byte header + data
- **JSON**: 2 quotes + data + field name overhead
- **Protobuf**: 1-2 byte tag + varint length + data
- **PostgreSQL**: 1-4 byte length prefix + data

**Design decisions must consider the full stack, not just one layer.**

---

### 3. Aggregation Changes Everything

Your architecture's **100:1 aggregation ratio** means:
- Raw event size is **100x less important** than aggregated reading size
- Storage costs after aggregation: **$3-35/year** difference (UUID vs int64)
- Performance comes from **batching and windowing**, not field size optimization

**Premature optimization of individual events is counterproductive.**

---

### 4. Cloud Costs: Storage Is Cheap, Writes Are Expensive

**DynamoDB example:**
- Storage: $285/month (1.14 TB)
- Writes: $32,400/month (864M events/day)
- **Storage is 0.87% of total cost**

**RDS example:**
- Storage: $131/month
- IOPS: $140/month
- **IOPS costs exceed storage costs**

**Optimize for write efficiency and query patterns, not storage size.**

---

### 5. Cardinality > Field Size

**Index performance** is determined by:
1. Number of unique values (cardinality)
2. Index depth (BTree levels)
3. Cache efficiency

**Field size matters for:**
- Index size (affects cache)
- Scan performance (less data = faster scans)

But **cardinality management** (bounded number of workspaces, aggregation to reduce records) has **10-100x more impact** than saving 20-40 bytes per field.

---

### 6. Flexibility Is Worth the Cost

For a metering system:
- **$300/year** (S3 archival): Negligible
- **$3,200/year** (RDS): Small compared to engineering time
- **0.87% overhead** (DynamoDB): Irrelevant

The ability to:
- Integrate with external systems (Stripe, etc.)
- Support multiple ID formats (UUID, ULID, Snowflake)
- Avoid schema migrations when requirements change
- Enable merger/acquisition scenarios

**is easily worth $300-3,200/year.**

---

## Conclusion

**Use `string` for WorkspaceID and other identifiers** in your metering system.

The industry has converged on strings for IDs because:
1. **Flexibility** enables integration and future-proofing
2. **Aggregation** makes raw event size nearly irrelevant
3. **Cloud storage** is cheap ($300/year difference for archival)
4. **Write costs** dominate over storage costs (100x+ in DynamoDB)
5. **Cardinality management** matters more than field size optimization

Your architecture's **100:1 aggregation strategy** is the right approach. Optimizing individual event size would be premature optimization that sacrifices flexibility for minimal cost savings.

**When building metering systems:**
- Learn from observability platforms (use strings, manage cardinality)
- Don't use observability databases for billing (data loss unacceptable)
- Optimize through aggregation, not field size reduction
- Design for flexibility first, optimize for cost second

The cost analysis shows that **engineering time spent on premature optimization** costs more than the cloud storage savings.

---

## References

### Industry Patterns & Standards
- [OpenTelemetry Metrics Data Model](https://opentelemetry.io/docs/specs/otel/metrics/data-model/)
- [OpenTelemetry Logs Data Model](https://opentelemetry.io/docs/specs/otel/logs/data-model/)
- [AWS CloudWatch Metrics Concepts](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/cloudwatch_concepts.html)
- [AWS CloudWatch Dimension API](https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_Dimension.html)
- [DataDog Best Practices for Scaling Observability Pipelines](https://docs.datadoghq.com/observability_pipelines/best_practices_for_scaling_observability_pipelines/)
- [Stripe Metered Billing](https://stripe.com/resources/more/what-is-metered-billing-heres-how-this-adaptable-billing-model-works)
- [Usage-Based Billing Implementation 2026](https://www.zenskar.com/blog/usage-based-billing)

### Technical Deep Dives
- [How String Works in Golang](https://perennialsky.medium.com/how-string-works-in-golang-7ac4d797164b)
- [[]byte vs string in Go](https://syslog.ravelin.com/byte-vs-string-in-go-d645b67ca7ff)
- [Protocol Buffers Encoding](https://protobuf.dev/programming-guides/encoding/)
- [How Protobuf Works—The Art of Data Encoding](https://victoriametrics.com/blog/go-protobuf/)

### Cloud Pricing
- [AWS S3 Pricing](https://aws.amazon.com/s3/pricing/)
- [AWS S3 Pricing Guide 2026](https://cloudchipr.com/blog/amazon-s3-pricing-explained)
- [AWS RDS PostgreSQL Pricing](https://aws.amazon.com/rds/postgresql/pricing/)
- [AWS RDS Pricing Breakdown 2026](https://sedai.io/blog/understanding-amazon-rds-costs-pricing)
- [GCP Cloud Storage Pricing](https://cloud.google.com/storage/pricing/)
- [Cloud Storage Pricing Comparison 2025](https://www.finout.io/blog/cloud-storage-pricing-comparison)
- [AWS DynamoDB Pricing](https://aws.amazon.com/dynamodb/pricing/)
- [DynamoDB Pricing Guide](https://www.nops.io/blog/amazon-dynamodb-pricing/)
- [AWS Timestream Pricing](https://aws.amazon.com/timestream/pricing/)

### Internal Documentation
- `metering-spec/docs/references/observability-vs-metering.md` - Observability vs metering patterns
- `metering-spec/specs/eventpayload.go` - Event payload specification
- `metering-spec/internal/examples/inflightpostflight_README.md` - Aggregation architecture

---

**Document Maintenance:**
- Update pricing when AWS/GCP rates change (typically annually)
- Revise throughput assumptions as usage patterns evolve
- Add new storage technologies as they emerge (e.g., S3 Express One Zone)
- Incorporate actual production metrics when available

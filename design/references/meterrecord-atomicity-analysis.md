# MeterRecord Atomicity: Single vs Multiple Measurements

**Date:** 2026-01-21
**Status:** Design Decision Pending
**Context:** Analysis of whether one event should produce one MeterRecord with multiple Measurements vs multiple MeterRecords with single Measurements

---

## Problem Statement

Current design: One event with multiple measurements (e.g., LLM API call with input_tokens and output_tokens) produces **multiple MeterRecords**, each containing a single Measurement.

```go
// Current design
Event: {input_tokens: 100, output_tokens: 50}
  ↓
MeterRecord 1: {ID: "evt_1:input_tokens", Measurement: {100, "input_tokens"}}
MeterRecord 2: {ID: "evt_1:output_tokens", Measurement: {50, "output_tokens"}}
```

**Design smell identified:** Are we creating YAGNI complexity by separating measurements that users need to keep together? Or are we correctly optimizing for downstream processing?

---

## The Core Question

**From first principles: What is the atomic unit of metering?**

Two candidates:
1. **The event** - One transaction (e.g., API call) produces one record
2. **The measurement** - Each extracted quantity+unit is independent

This choice affects:
- Data integrity and atomicity
- Query complexity (relationships between measurements)
- Processing architecture (aggregation patterns)
- Risk of partial failures

---

## Analysis Using Chris's Design Principles

### Principle #1: Design from First Principles

**User mental model:**
- User makes **one LLM API call**
- That call consumed 100 input tokens and 50 output tokens
- Question: Is this ONE billable event or TWO?

**Accounting analogy (double-entry bookkeeping):**
```
Journal Entry (atomic transaction):
  - Debit: Office Supplies $100
  - Credit: Cash $100
  → Both postings must succeed or neither succeeds
```

Similarly:
```
LLM API Call (atomic metering event):
  - Input tokens: 100
  - Output tokens: 50
  → Both measurements are part of same transaction
```

**Current design treats them as separate transactions:**
- input_tokens can succeed while output_tokens fails
- Violates user expectation of atomicity

**Bundled design treats them as one transaction:**
- Both measurements persist together or neither does
- Matches user mental model

**Verdict:** From first principles, the atomic unit is the **event**, not individual measurements.

---

### Principle #4: Minimize Complexity, Avoid Trap Doors

**Current design creates a trap door:**

```go
// Easy to implement persistence INCORRECTLY
func PersistRecords(records []MeterRecord) {
    for _, record := range records {
        db.Save(record)  // ❌ No transaction! Partial failures possible
    }
}

// Scenario: Network hiccup between saves
db.Save(record1)  // ✓ input_tokens saved
// [network error]
db.Save(record2)  // ✗ output_tokens lost

// Result: Customer billed for inputs but not outputs
// Data corruption, silent failure, hard to detect
```

**Correct implementation requires transactions:**
```go
// Every implementation must remember to do this
func PersistRecords(records []MeterRecord) {
    tx := db.BeginTransaction()
    for _, record := range records {
        tx.Save(record)
    }
    tx.Commit()  // All or nothing
}
```

**Problems:**
- Easy to forget transactions (pit of failure)
- Requires infrastructure support for transactions
- Extra complexity at every persistence point
- Silent data corruption if done wrong

**Bundled design avoids the trap door:**

```go
// Natural atomicity - single record = single save operation
func PersistRecord(record MeterRecord) {
    db.Save(record)  // ✓ Either all measurements persist or none do
}
```

**Benefits:**
- "Pit of success" - correct implementation is easiest
- Can't accidentally get partial events
- Event integrity preserved by structure
- No special transaction handling needed

**Verdict:** Current design has a significant trap door. Bundled design provides natural safety.

---

### Principle #9: Users First - Engineer for Next 10

**User needs analysis:**

#### User 1: LLM API Provider
**Needs:**
- Atomicity: "Bill for complete API call or nothing"
- Ratio analysis: "What's my average input/output token ratio?"
- Cost per request: "Total cost = input_cost + output_cost"

**Current design:**
- ❌ Atomicity: Can get partial records (input without output)
- ❌ Ratio analysis: Requires JOIN on SourceEventID, assumes both records exist
- ❌ Cost calculation: Must JOIN to ensure both measurements present

**Bundled design:**
- ✓ Atomicity: Single record, both measurements always present
- ✓ Ratio analysis: `record.Measurements[0].Quantity / record.Measurements[1].Quantity`
- ✓ Cost calculation: Iterate measurements in single record

---

#### User 2: Cloud Infrastructure Provider
**Event:** VM usage tick
```json
{
  "cpu_ms": 1000,
  "memory_mb": 2048,
  "disk_iops": 500,
  "network_bytes": 1048576
}
```

**Needs:**
- Resource correlation: "Show VMs with high CPU but low memory"
- Complete snapshots: "Either capture full resource state or nothing"
- Time-series analysis: "How do CPU/memory/disk correlate over time?"

**Current design:**
- ❌ Could have CPU record without memory record (partial snapshot)
- ❌ Correlation requires complex JOINs across 4 separate records
- ❌ Can't distinguish "metric was zero" from "metric failed to record"

**Bundled design:**
- ✓ Complete resource snapshot guaranteed
- ✓ All metrics in single record, trivial correlation
- ✓ Missing measurement = didn't happen vs zero value = explicit

---

#### User 3: Data Quality / Auditing
**Needs:**
- "Find incomplete events" (missing expected measurements)
- "Verify all events are fully metered"
- "Audit trail: prove we billed correctly"

**Current design:**
- ❌ Impossible to distinguish "measurement was zero" from "measurement failed"
- ❌ Finding orphaned measurements requires complex queries
- ❌ "Did we bill correctly?" requires verifying all related records exist

**Bundled design:**
- ✓ Either event exists (complete) or doesn't (incomplete)
- ✓ Each record carries all measurements or none
- ✓ Audit trail: one record = one complete event

**Verdict:** All users (1-10+) benefit from atomicity and co-location. No validated user need for separation.

---

### Principle #2: Avoid if/else Blocks

**Initial concern:** Does bundling require conditionals during aggregation?

**Answer:** No - both designs can use accumulator pattern with map lookups.

**Current design (separate records):**
```go
accumulators := make(map[string]*Aggregator)
for _, record := range records {
    unit := record.Measurement.Unit
    if accumulators[unit] == nil {
        accumulators[unit] = NewAggregator(getConfigForUnit(unit))
    }
    accumulators[unit].Add(record.Measurement)
}
```

**Bundled design:**
```go
accumulators := make(map[string]*Aggregator)
for _, record := range records {
    for _, measurement := range record.Measurements {
        unit := measurement.Unit
        if accumulators[unit] == nil {
            accumulators[unit] = NewAggregator(getConfigForUnit(unit))
        }
        accumulators[unit].Add(measurement)
    }
}
```

**Processing complexity:**
- Current: One loop (N records)
- Bundled: Nested loops (N records × M measurements per record)

**But:** In practice, M is small (2-5 measurements per event), so nested loop is not a concern.

**Verdict:** Both designs can avoid conditionals. Processing complexity roughly equivalent.

---

### Principle #3: Don't Make Decisions Twice

**Question:** Where should "separate by unit" decision happen?

**Current design:**
- Decision made at **metering time** (early separation)
- Aggregation receives pre-separated records by unit
- Optimization: no re-grouping needed

**Bundled design:**
- Decision made at **aggregation time** (late separation)
- Metering preserves event integrity
- Aggregation groups by unit when needed

**Key insight:** These are actually **different concerns:**
1. **Metering concern:** Preserve event integrity (atomicity)
2. **Aggregation concern:** Group by unit for processing (business logic)

Current design conflates these concerns - makes aggregation's decision at metering time, sacrificing integrity.

**Verdict:** Bundled design makes each decision where it belongs.

---

### Principle #7: Single Responsibility

**Current design:**
- MeterRecord responsibility: "Carry one measurement extracted from an event"
- But: Loses semantic of "these measurements came from same event"
- Multiple reasons to change: how we handle that unit, how we link related measurements

**Bundled design:**
- MeterRecord responsibility: "Preserve complete metered event with all measurements"
- Clear semantic: "I am one event with all its measurements"
- One reason to change: how we represent events

**Verdict:** Bundled design has clearer, single responsibility.

---

### Principle #11: Fix Root Causes, Not Symptoms

**Root cause analysis:**

**Why are we separating measurements?**
- Surface answer: "Because aggregation needs them separated by unit"
- Deeper question: "Is that metering's concern or aggregation's concern?"

**The real requirements:**
- **Metering:** Extract measurements from events, preserve integrity
- **Aggregation:** Group measurements by unit, apply aggregation strategies

**Current design:**
- Optimizes for aggregation (pre-separated)
- At expense of metering integrity (loses atomicity)
- Fixes aggregation's problem by breaking metering's correctness

**Bundled design:**
- Metering does its job: preserve integrity
- Aggregation does its job: separate by unit
- Each layer solves its own problem

**Verdict:** Current design optimizes wrong layer. Bundled design addresses root requirements correctly.

---

## Key Insight: Idempotency vs Atomicity

**Current design has idempotency but not atomicity:**

```go
// Deterministic IDs ensure replaying same event produces same records
MeterRecord 1: ID = hash(event_id + "input_tokens")  // Idempotent ✓
MeterRecord 2: ID = hash(event_id + "output_tokens") // Idempotent ✓

// But atomicity not guaranteed
Replay attempt 1:
  → Record 1 saved ✓
  → Record 2 failed ✗  (partial event in database)

Replay attempt 2:
  → Record 1 rejected (duplicate ID) ✓ Idempotency works
  → Record 2 saved ✓
  → Final state: Both records exist ✓

// Eventually consistent but required 2 attempts
// Customer sees incorrect data during window between attempts
```

**Bundled design has both idempotency AND atomicity:**

```go
MeterRecord: ID = hash(event_id)  // Idempotent ✓

Replay attempt 1:
  → Record saved with all measurements ✓  (atomic)

Replay attempt 2:
  → Record rejected (duplicate ID) ✓  (idempotent)

// Immediately consistent, single attempt
```

---

## Evidence from Codebase

### Current Implementation Shows Atomicity Already Broken

From `/Users/chris/workspace/meters/metering-spec/internal/examples/inflightpostflight_test.go:96-107`:

```go
records, err := internal.Meter(payload, config)
if err != nil {
    panic(fmt.Sprintf("Failed to meter payload: %v", err))
}

for _, record := range records {
    h.bus.Publish(InFlightMeterRecordedEvent{Record: record})  // ← Each record published separately
}
```

**Problem:** Publishing each record individually means:
- No transaction boundary around related measurements
- Partial publishing possible (some succeed, some fail)
- Data integrity already violated in current design

**Fix with current design:** Wrap in transaction (but easy to forget - trap door!)

**Fix with bundled design:** Single record = single publish = naturally atomic

---

### Documentation Says "Multiple Records from One Event"

From `/Users/chris/workspace/meters/metering-spec/specs/meterrecord.go:11-12`:

> "One event payload can produce multiple meter records when the metering configuration extracts multiple measurements from the same event."

**This is the current design**, but the question is whether it's the **right** design.

---

## Real-World Comparison: Double-Entry Accounting Systems

Financial accounting provides a proven model for this exact problem:

**One transaction (event) → Multiple entries (measurements)**

### How Accounting Systems Handle It

**Option 1: Separate row per entry (like current design)**
```sql
-- Ledger table
id | transaction_id | account      | debit | credit
1  | txn_123       | cash         | NULL  | 100
2  | txn_123       | supplies     | 100   | NULL
```

**Problems:**
- Can insert row 1 without row 2 (violates double-entry invariant)
- Requires transaction_id FK to maintain relationship
- Finding unbalanced transactions requires GROUP BY + HAVING

**Option 2: Transaction header + line items (like bundled design)**
```sql
-- Transactions table
id      | date       | description
txn_123 | 2024-01-01 | Office supplies

-- Transaction_lines table (child of transaction)
id | transaction_id | account   | debit | credit
1  | txn_123       | cash      | NULL  | 100
2  | txn_123       | supplies  | 100   | NULL
```

**Benefits:**
- FK constraint ensures lines belong to valid transaction
- Deleting transaction cascades to lines (referential integrity)
- Can enforce "transaction has ≥2 lines" at application layer
- Natural query: "Get all lines for transaction" vs complex JOIN

**Industry practice:** Financial systems use Option 2 (header + lines) specifically to maintain transaction integrity.

**Mapping to metering:**
- Transaction = Event = MeterRecord
- Transaction lines = Measurements
- Bundled design matches proven accounting pattern

---

## Alternative Architectures Considered

### Option 1: Keep Current Design, Add Transaction Support

**Approach:** Require all persistence to use transactions

```go
type RecordRepository interface {
    SaveRecordsAtomically(records []MeterRecord) error
}

func (r *PostgresRepo) SaveRecordsAtomically(records []MeterRecord) error {
    tx := r.db.BeginTransaction()
    for _, record := range records {
        if err := tx.Save(record); err != nil {
            tx.Rollback()
            return err
        }
    }
    return tx.Commit()
}
```

**Problems:**
- Every implementation must remember atomicity requirement
- Infrastructure must support transactions (limits database choices)
- Still can't query "give me all measurements for event X" without JOIN
- Doesn't solve correlation/ratio queries (still need JOINs)

**Verdict:** Fixes atomicity but not usability. Extra complexity.

---

### Option 2: Bundled Design with Nested Structure

**Approach:** One MeterRecord contains multiple Measurements

```go
type MeterRecordSpec struct {
    ID            string                 // From event ID only
    WorkspaceID   string
    UniverseID    string
    Subject       string
    RecordedAt    time.Time
    Measurements  []MeasurementSpec      // ← Multiple measurements from same event
    Dimensions    map[string]string      // Same dimensions for all measurements
    SourceEventID string
    MeteredAt     time.Time
}
```

**Benefits:**
- Natural atomicity (one record save = all measurements)
- Event relationships preserved (no JOIN needed)
- Ratio/correlation queries trivial
- Matches user mental model (one event = one record)
- Simpler ID generation (just event ID, no unit suffix)

**Processing:**
```go
// Aggregation still separates by unit, just later in pipeline
for _, record := range records {
    for _, measurement := range record.Measurements {
        aggregators[measurement.Unit].Add(measurement, record.RecordedAt)
    }
}
```

**Tradeoffs:**
- Nested loops (but M measurements per event is small, not a concern)
- MeterReading still has single Measurement (not bundled) - readings aggregate by unit

**Verdict:** Solves atomicity, preserves relationships, minimal processing overhead.

---

### Option 3: Event Sourcing (Keep Raw Events)

**Approach:** Store raw EventPayload, derive MeterRecords on read

```go
// Store
db.SaveEvent(eventPayload)

// Query time
records := Meter(eventPayload, currentConfig)
readings := Aggregate(records, window)
```

**Benefits:**
- Can replay events with new metering configs
- Complete audit trail
- Natural atomicity (one event)

**Problems:**
- Processing cost on every query (can't pre-aggregate)
- Doesn't match spec design (MeterRecord is the domain object)
- Requires EventPayload retention (storage overhead)

**Verdict:** Interesting for auditing but doesn't address core design question.

---

## Recommendation

**Adopt Option 2: Bundled Design (One MeterRecord with Multiple Measurements)**

### Proposed Structure

```go
type MeterRecordSpec struct {
    ID            string                 // hash(event_id) - no unit suffix
    WorkspaceID   string
    UniverseID    string
    Subject       string
    RecordedAt    time.Time
    Measurements  []MeasurementSpec      // All measurements from same event
    Dimensions    map[string]string      // Shared dimensions
    SourceEventID string
    MeteredAt     time.Time
}
```

### Migration Path (Principle #10: Add, Migrate, Remove)

**Step 1: Add new field alongside old**
```go
type MeterRecordSpec struct {
    // OLD (deprecated, keep for now)
    Measurement  MeasurementSpec        `json:"measurement,omitempty"`

    // NEW (preferred)
    Measurements []MeasurementSpec      `json:"measurements,omitempty"`
}
```

**Step 2: Update Meter() to populate both**
```go
// Returns one record with multiple measurements
record := MeterRecordSpec{
    Measurement:  measurements[0],       // OLD: first measurement for backwards compat
    Measurements: measurements,          // NEW: all measurements
}
```

**Step 3: Update consumers to use Measurements**
- Aggregation logic reads from Measurements array
- Each commit compiles and passes tests

**Step 4: Remove old Measurement field**
- Only when all consumers migrated
- Compiler catches any missed usages

---

## Impact Analysis

### Breaking Changes

**Spec changes:**
- `MeterRecordSpec.Measurement` (singular) → `Measurements` (array)
- ID generation: no longer includes unit suffix
- MeterReading.RecordCount semantics (counts events, not individual measurements)

**Implementation changes:**
- `Meter()` function returns 1 record per event (not N records)
- Aggregation must iterate nested measurements
- Repository save operations simpler (one record, no transaction needed)

### Non-Breaking

**Still works:**
- Idempotency (deterministic IDs from event ID)
- Time-weighted averages (lastBeforeWindow pattern)
- Dimensional filtering (dimensions shared by all measurements in record)
- MeterReading structure (still has single Measurement per reading)

---

## Open Questions

### Q1: What if different measurements have different dimensions?

**Example:**
```json
{
  "cpu_ms": 1000,
  "cpu_region": "us-east",
  "memory_mb": 2048,
  "memory_region": "us-west"
}
```

**Current answer:** Metering config determines which properties are extracted vs dimensions. If regions differ, they'd both be dimensions:
```go
Measurements: [{1000, "cpu_ms"}, {2048, "memory_mb"}]
Dimensions: {"cpu_region": "us-east", "memory_region": "us-west"}
```

**Alternative:** Allow dimensions per measurement (more complex)

**Recommendation:** Start with shared dimensions (YAGNI). Can refactor later if validated need emerges.

---

### Q2: How does this affect storage size?

**Current design:**
```
Record 1: {dimensions: {model: "claude", region: "us-east"}, measurement: {...}}
Record 2: {dimensions: {model: "claude", region: "us-east"}, measurement: {...}}
→ Dimensions duplicated across records
```

**Bundled design:**
```
Record: {dimensions: {model: "claude", region: "us-east"}, measurements: [{...}, {...}]}
→ Dimensions stored once
```

**Verdict:** Bundled design actually REDUCES storage (no dimension duplication).

---

### Q3: Does this affect streaming/batching patterns?

**Current pattern (from inflightpostflight_test.go):**
```go
for _, record := range records {
    bus.Publish(MeterRecordEvent{Record: record})
}
```

**Bundled pattern:**
```go
record := MeterRecord{Measurements: measurements}
bus.Publish(MeterRecordEvent{Record: record})  // Single publish
```

**Verdict:** Bundled design simplifies publishing (one event instead of N).

---

## Summary

| Concern | Current (Separate) | Bundled | Winner |
|---------|-------------------|---------|--------|
| **Atomicity** | ❌ Partial events possible | ✓ Natural atomicity | **Bundled** |
| **Data integrity** | ❌ Trap door (easy to forget transactions) | ✓ Pit of success | **Bundled** |
| **User mental model** | ❌ One event → many records (surprising) | ✓ One event → one record | **Bundled** |
| **Relationships** | ❌ Requires JOINs | ✓ Trivial (same record) | **Bundled** |
| **Processing complexity** | ✓ Single loop | ⚠ Nested loop (but M is small) | Slight edge to current |
| **Storage size** | ❌ Dimension duplication | ✓ Shared dimensions | **Bundled** |
| **Publishing** | ❌ N publish operations | ✓ 1 publish operation | **Bundled** |
| **Idempotency** | ✓ Deterministic IDs | ✓ Deterministic IDs | Tie |

**Overall winner:** Bundled design by significant margin.

---

## Next Steps

1. **Validate with users:** Confirm atomicity is actually needed (don't just assume)
2. **Prototype bundled design:** Implement in branch to verify aggregation complexity
3. **Performance test:** Measure nested loop overhead with realistic data
4. **Migration plan:** Design backwards-compatible transition (Add, Migrate, Remove)
5. **Update docs:** Revise MeterRecord spec and examples

---

## References

### Internal
- `/Users/chris/workspace/meters/arch/reference/chris-design-principles.md` - Design principles applied
- `/Users/chris/workspace/meters/metering-spec/specs/meterrecord.go` - Current spec
- `/Users/chris/workspace/meters/metering-spec/internal/metering.go` - Meter() implementation
- `/Users/chris/workspace/meters/metering-spec/internal/aggregation.go` - Aggregate() implementation
- `/Users/chris/workspace/meters/metering-spec/internal/examples/inflightpostflight_test.go` - Publishing pattern

### Industry Patterns
- Double-entry accounting: Transaction header + line items pattern
- Event sourcing: One event = atomic unit
- Database transactions: ACID properties for related data

### Related Discussions
- Original design conversation (2026-01-21)
- Chris's journal entry analogy for atomicity
- Accumulator pattern for avoiding conditionals

---

## Decision Log

**2026-01-21:** Analysis completed, recommendation for bundled design documented. Awaiting validation with real users and prototype implementation.

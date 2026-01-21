# Documentation Roadmap for Metering Spec

This document outlines the documentation strategy to make this a top-tier open source specification on GitHub.

## Current State

✅ **Inline API reference docs** - `specs/` files have Stripe-style field documentation
✅ **Reference implementation** - `internal/` Go code with tests
✅ **Design rationale** - ADRs in `arch/` (workspace-universe-isolation.md)
✅ **Production example** - `internal/examples/inflightpostflight_*` (advanced pattern)

## Missing Documentation (Prioritized)

### Tier 1: Essential - Make It Usable (Week 1)

These are blocking adoption. Without them, users can't get started.

#### 1. README.md (20 minutes, critical)
**Location:** Root of metering-spec/
**Purpose:** Landing page that answers "what is this and why should I care?"

**Must include:**
- 1-2 sentence description
- Problem statement (why does this exist?)
- Quick example (3-5 lines showing the transformation)
- Links to documentation sections
- Installation/usage basics

**Target audience:** Someone discovering this repo for the first time

#### 2. Basic End-to-End Example (30 minutes, critical)
**Location:** `docs/examples/basic-api-metering.md`
**Purpose:** Show the complete flow with minimal complexity

**Must include:**
- Single EventPayload (JSON)
- Single MeteringConfigSpec (JSON)
- Resulting MeterRecord (JSON)
- Single AggregateConfigSpec (JSON)
- Resulting MeterReading (JSON)
- Brief explanation of each step

**Why not use inflightpostflight example?** Too advanced - event bus, stateful batching, 300 events. Week 1 needs simplicity.

**Target audience:** Developer trying to understand "what does this spec do?"

#### 3. Data Flow Diagram (15 minutes, high impact)
**Location:** `docs/architecture.md` or README.md
**Purpose:** Visual explanation of the pipeline

**Should show:**
```
EventPayload → [Meter] → MeterRecord → [Aggregate] → MeterReading
              ↑                        ↑
         MeteringConfig          AggregateConfig
```

With brief explanation of each stage. ASCII art or Mermaid diagram acceptable.

**Target audience:** Architects evaluating the spec

#### 4. LICENSE (5 minutes, legally required)
**Location:** Root of metering-spec/
**Purpose:** Legal clarity for open source usage

**Recommendation:** MIT or Apache 2.0 (most permissive, common for specs)

### Tier 2: High Value - Make It Understandable (Week 2)

These enable developers to use the spec correctly.

#### 5. CONCEPTS.md (60 minutes)
**Location:** `docs/concepts.md`
**Purpose:** Explain core domain concepts in depth

**Must cover:**
- **Workspace vs Universe** - The two-dimensional isolation model
  - Why both exist (operational vs data namespace)
  - When to use each
  - Examples: multi-region, test/prod, post-merger
- **Subject Attribution** - The "type:id" pattern
  - Format convention
  - Scoping to universe
  - Examples: customer:, org:, team:, cohort:
- **Measurements vs Dimensions** - Numeric vs categorical
  - What gets extracted as each
  - How they're used downstream
- **Time Semantics** - Business time vs system time
  - RecordedAt (when usage occurred)
  - MeteredAt (when processed)
  - CreatedAt (when persisted)
  - Why all three exist
- **Aggregation Strategies** - When to use each
  - sum: Total usage (API calls, tokens)
  - max: Peak usage (concurrent users)
  - time-weighted-avg: Gauge values (seat count)
  - latest: Most recent state
  - min: Minimum value

**Target audience:** Developers implementing the spec

#### 6. Design Rationale (30 minutes)
**Location:** `design/` directory
**Purpose:** Explain why decisions were made

**Approach:** Copy/adapt from `arch/`:
- `workspace-universe-isolation.md` (already excellent, just move it)
- `spec-pattern.md` (primitives-only pattern)
- New: `conditional-metering.md` (why filters exist)
- New: `deterministic-ids.md` (why IDs are computed from content)
- New: `time-semantics.md` (why three different timestamps)

**Target audience:** Contributors, implementers questioning design choices

#### 7. GETTING_STARTED.md (45 minutes)
**Location:** `docs/getting-started.md`
**Purpose:** Step-by-step guide from zero to working implementation

**Should walk through:**
1. Define your first event type (what properties to include)
2. Configure metering to extract measurements
3. Send an event payload
4. Query meter records
5. Configure aggregation
6. Query meter readings

**Include:** Code snippets, expected outputs, common pitfalls

**Target audience:** New developers implementing their first integration

### Tier 3: Nice to Have - Make It Collaborative (Week 3+)

These enable community contributions and evolution.

#### 8. CONTRIBUTING.md (30 minutes)
**Location:** Root of metering-spec/
**Purpose:** How to contribute to the spec

**Should cover:**
- How to propose changes (issues, PRs)
- Spec versioning philosophy (semantic versioning)
- How to add examples
- How to update documentation
- Code of conduct reference

#### 9. More Examples (15-30 min each)
**Location:** `docs/examples/`
**Purpose:** Show common usage patterns

**Candidates:**
- `llm-token-metering.md` - LLM tokens with model dimensions
- `time-weighted-seats.md` - Seat-based pricing with gauges
- `conditional-metering.md` - Using filters for tiered pricing
- `multi-workspace.md` - Cross-region metering

**Note:** The inflightpostflight example should be moved to `docs/examples/production-patterns.md` with its README.

#### 10. FAQ.md (45 minutes)
**Location:** `docs/faq.md`
**Purpose:** Answer common questions

**Questions to cover:**
- Why string Properties instead of typed?
  - Answer: Maximum flexibility, workspace-specific schemas
- Why Workspace AND Universe?
  - Answer: Operational vs data namespace (link to concepts.md)
- Can I use this for real-time billing?
  - Answer: Yes, see inflightpostflight example
- How do I handle late-arriving events?
  - Answer: MeteredAt watermarking, incremental aggregation
- What happens if I change a MeteringConfig?
  - Answer: New events use new config, old meter records unchanged
- Why Decimal strings instead of numbers?
  - Answer: Precision preservation across languages

#### 11. Implementation Guides (60-90 min each)
**Location:** `docs/implementing-in-{language}.md`
**Purpose:** Language-specific guidance

**Wait until:** After getting real-world implementation feedback

**Should cover:**
- Type mappings (Go types → target language)
- JSON Schema generation
- Validation strategies
- Precision handling (Decimal libraries)

#### 12. CHANGELOG.md (ongoing)
**Location:** Root of metering-spec/
**Purpose:** Track version history

**Start when:** First versioned release (v0.1.0)

**Should include:**
- Semantic versioning strategy
- Breaking vs non-breaking changes
- Migration guides between versions

### Tier 4: Polish - Make It Professional (Future)

These add credibility and discoverability.

#### 13. Comparison / Positioning (60 minutes)
**Location:** `docs/comparison.md`
**Purpose:** How this differs from alternatives

**Compare against:**
- Prometheus (metrics system) - Different focus (billing vs observability)
- Metronome/Lago/Stripe Billing (billing platforms) - Different layer (data model vs full platform)
- CloudEvents (event format) - Compatible, can be used together
- OpenTelemetry - Different domain (traces/metrics vs billing)

**Target audience:** Decision makers evaluating options

#### 14. Glossary (20 minutes)
**Location:** `docs/glossary.md` or part of CONCEPTS.md
**Purpose:** Define all domain terms

**Terms to define:**
- Workspace, Universe, Subject, EventPayload, MeterRecord, MeterReading
- Measurement, Dimension, Unit, Quantity, Aggregation
- Business time, System time, Watermark

## Recommended Execution Order

### Week 1: Make it usable
1. README.md (landing page with quick example)
2. Basic example (`docs/examples/basic-api-metering.md`)
3. Data flow diagram (in README.md or docs/architecture.md)
4. LICENSE file

### Week 2: Make it understandable
5. CONCEPTS.md (core domain concepts)
6. Move workspace-universe-isolation.md to design/
7. GETTING_STARTED.md (step-by-step guide)

### Week 3: Make it collaborative
8. CONTRIBUTING.md
9. Add 2-3 more examples (LLM tokens, time-weighted seats)
10. Move inflightpostflight to docs/examples/production-patterns.md
11. FAQ.md

### Later: Polish and evolve
12. Implementation guides (after real-world feedback)
13. Comparison doc (when competitors ask)
14. CHANGELOG (as you version)

## Success Metrics

A top-tier open source spec should enable:

1. **5-minute evaluation** - README + quick example = "I understand what this does"
2. **30-minute implementation** - GETTING_STARTED = working code
3. **Deep understanding** - CONCEPTS + design rationale = "I know why it works this way"
4. **Community contribution** - CONTRIBUTING + examples = "I can propose changes"

## Notes on Existing Examples

**inflightpostflight example:**
- Excellent production architecture pattern
- Too complex for Week 1 basic example
- Perfect for Week 3 "production patterns" documentation
- Should be moved to `docs/examples/production-patterns.md` with README
- Demonstrates real-world value beyond toy examples

**What's still needed for Week 1:**
- Simple JSON-only example showing: 1 event → 1 record → 1 reading
- No event bus, no batching, no complex handlers
- Just the data transformation with minimal explanation

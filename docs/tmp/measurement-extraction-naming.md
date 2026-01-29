# Naming Consideration: MeasurementExtraction vs ObservationExtraction

**Date:** 2026-01-28
**Status:** Proposed - Pending Ubiquitous Language Review
**Context:** After completing observation-temporal-context ADR migration

---

## Current State

After migrating domain layer to use Observation/AggregateValue types, we still have "Measurement" terminology in configuration:

**Configuration types (kept):**
- `MeasurementExtraction` - config for extracting from events
- `MeasurementSourceProperty` - which property to extract
- `specs.MeasurementExtractionSpec` - spec for extraction config

**Data types (removed):**
- ~~`Measurement`~~ → Replaced by `Observation` and `AggregateValue`
- ~~`MeterRecord.Measurement`~~ → `MeterRecord.Observations`
- ~~`MeterReading.Measurement`~~ → `MeterReading.Value`

---

## The Confusion

Having "Measurement" in config but "Observation" in data creates terminology mismatch:

```go
// Configuration says "Measurement"
config := MeteringConfig{
    measurements: []MeasurementExtraction{...}
}

// But result is "Observation"
record := MeterRecord{
    Observations: []Observation{...}
}
```

This could confuse developers: "Are we extracting measurements or observations?"

---

## Proposed Rename

**Option A: Align configuration with domain terminology**

```go
// Before
type MeasurementExtraction struct {
    sourceProperty MeasurementSourceProperty
    unit           Unit
    filter         *Filter
}

config.Measurements() []MeasurementExtraction

// After
type ObservationExtraction struct {
    sourceProperty ObservationSourceProperty
    unit           Unit
    filter         *Filter
}

config.Extractions() []ObservationExtraction
```

**Rationale:**
- Consistent terminology: extract observations, get observations
- Aligns with ubiquitous language from ADR
- Clear mental model: "I configure observation extraction, I get observations"

---

## Before Deciding: Review Ubiquitous Language

**IMPORTANT:** Before making this change, we should revisit DDD ubiquitous language principles:

### Questions to Answer:

1. **What do domain experts call this?**
   - Do they say "extract measurements" or "extract observations"?
   - Is there a distinction in their mental model?

2. **What does the observation-temporal-context ADR say?**
   - Does it define "observation" as the extracted thing?
   - Does it use "measurement" anywhere intentionally?

3. **Is there a semantic difference?**
   - Measurement = the act of measuring?
   - Observation = the result of measuring?
   - If so, `MeasurementExtraction` might actually be correct ("how to measure")

4. **Check other domain concepts:**
   - We have "metering" config (not "observing" config)
   - We have "Meter()" function (not "Observe()")
   - Is "measurement" the process, "observation" the result?

5. **Industry terminology:**
   - What do observability tools call this?
   - What do metering systems call this?
   - OpenTelemetry, Prometheus, etc.?

---

## Possible Outcomes

### Outcome 1: Rename to ObservationExtraction
**If:** "Observation" is the ubiquitous term for both config and data
**Impact:** Low - simple rename, maintains consistency

### Outcome 2: Keep MeasurementExtraction
**If:** "Measurement" = process, "Observation" = result (intentional distinction)
**Impact:** None - document the terminology difference
**Example:** "MeasurementExtraction defines HOW to measure, Observation is WHAT was measured"

### Outcome 3: Different name entirely
**If:** Review reveals better domain term (e.g., "MetricExtraction", "ExtractionSpec")
**Impact:** Medium - rename to more accurate term

---

## Action Items

Before deciding on rename:

1. [ ] Review observation-temporal-context.md ADR for terminology definitions
2. [ ] Check if "measurement" vs "observation" distinction is intentional
3. [ ] Review design/references/meterrecord-atomicity-analysis.md for terminology usage
4. [ ] Grep codebase for comments explaining "measurement" vs "observation"
5. [ ] Consider industry standards (OpenTelemetry, etc.)
6. [ ] Document final decision in ADR or design doc

**Only after ubiquitous language review:** Decide whether to rename or document the distinction.

---

## Migration Path (If Renaming)

If we decide to rename, use Add-Migrate-Remove pattern:

1. Add `ObservationExtraction` alongside `MeasurementExtraction`
2. Migrate callers one by one
3. Remove `MeasurementExtraction` when zero callers

Estimated effort: ~30 minutes, 3-4 commits

---

## Related

- `arch/work/observation-domain-layer-migration-plan.md` - Completed migration
- `design/observation-temporal-context.md` - ADR defining Observation type
- `design/references/meterrecord-atomicity-analysis.md` - Uses "measurements" terminology

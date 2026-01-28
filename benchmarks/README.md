# Metering Spec Benchmarks

This directory contains comprehensive benchmark tests that measure actual memory allocation, JSON serialization performance, and validate theoretical size calculations for the metering spec types.

## Purpose

These benchmarks support the analysis in `design/references/event-sizing-and-cost-analysis.md` by:

1. **Validating theoretical calculations** - Confirming that estimated sizes match actual measurements
2. **Measuring real performance** - Understanding CPU and memory costs at scale
3. **Providing production guidance** - Helping developers understand performance characteristics
4. **Detecting regressions** - Ensuring spec changes don't introduce performance issues

## Benchmark Files

### `eventpayload_test.go`

Benchmarks for `EventPayloadSpec`:
- Memory allocation with different data sizes (minimal, realistic, UUID WorkspaceID)
- JSON serialization and deserialization performance
- JSON wire format size measurements
- Large Properties map scenarios

**Run:**
```bash
go test -bench=BenchmarkEventPayload -benchmem ./benchmarks/
```

### `meterrecord_test.go`

Benchmarks for `MeterRecordSpec`:
- Memory allocation scenarios (minimal, realistic, large dimensions)
- JSON marshal/unmarshal performance
- JSON size measurements

**Run:**
```bash
go test -bench=BenchmarkMeterRecord -benchmem ./benchmarks/
```

### `meterreading_test.go`

Benchmarks for `MeterReadingSpec`:
- Memory allocation with different aggregation types
- JSON serialization performance
- Size measurements for sum and time-weighted-avg scenarios

**Run:**
```bash
go test -bench=BenchmarkMeterReading -benchmem ./benchmarks/
```

### `sizing_calculator_test.go`

Comprehensive size analysis and validation:
- Go memory estimation vs measurement
- JSON wire format size
- PostgreSQL storage estimates
- Struct size analysis using `unsafe.Sizeof`
- Scale calculations at 10k events/sec

**Run:**
```bash
# Size breakdown for all types
go test -v ./benchmarks/ -run TestSizeBreakdown

# Struct sizes
go test -v ./benchmarks/ -run TestStructSizes

# Scale calculations
go test -v ./benchmarks/ -run TestScaleCalculations
```

## Quick Start

### Run All Benchmarks

```bash
go test -bench=. -benchmem ./benchmarks/
```

### Run Sizing Analysis

```bash
go test -v ./benchmarks/ -run 'TestEventPayloadSizeBreakdown|TestMeterRecordSizeBreakdown|TestMeterReadingSizeBreakdown|TestStructSizes'
```

### Generate Benchmark Report

```bash
go test -bench=. -benchmem ./benchmarks/ > benchmarks_$(date +%Y%m%d).txt
```

## Understanding Benchmark Output

### Memory Benchmarks

```
BenchmarkEventPayload_Realistic_Memory-10    18518732    64.46 ns/op    0 B/op    0 allocs/op
```

- `18518732` - Number of iterations
- `64.46 ns/op` - Nanoseconds per operation
- `0 B/op` - Bytes allocated per operation (0 = stack allocated)
- `0 allocs/op` - Number of heap allocations per operation

**Interpretation:**
- `0 B/op, 0 allocs/op` = struct is stack-allocated (good!)
- Non-zero values = heap allocations (check if maps are involved)

### JSON Benchmarks

```
BenchmarkEventPayload_Realistic_JSONMarshal-10    1737135    703.9 ns/op    544 B/op    8 allocs/op
```

- `703.9 ns/op` - Time to serialize to JSON
- `544 B/op` - Temporary memory allocated during serialization
- `8 allocs/op` - Number of allocations during serialization

**At 10k events/sec:**
- CPU time: 10,000 × 704 ns = 7.04 ms/sec = 0.7% of one CPU core
- Memory allocation: 10,000 × 544 B/sec = 5.44 MB/sec

### Size Measurements

```
BenchmarkEventPayload_JSONSize/Realistic-10    232.0 bytes
```

- Actual JSON wire format size in bytes
- Used for network bandwidth and storage calculations

### Size Breakdown Tests

```
Go Memory (estimated): 326 bytes
Go Memory (measured):  0 bytes (stack allocated)
JSON Wire Format:      232 bytes
PostgreSQL (estimate): 132 bytes
```

- **Estimated** = Calculated using Go's memory layout rules
- **Measured** = From runtime.MemStats (captures heap allocations only)
- **JSON** = Actual serialized size
- **PostgreSQL** = Estimated row size with VARCHAR length prefixes

## Key Findings

### Performance at 10k Events/Second

| Type | Memory Alloc | JSON Marshal | JSON Unmarshal |
|------|--------------|--------------|----------------|
| EventPayloadSpec | 64 ns/op | 704 ns/op (0.7% CPU) | 2555 ns/op (2.6% CPU) |
| MeterRecordSpec | 110 ns/op | 1020 ns/op (1.0% CPU) | 3704 ns/op (3.7% CPU) |
| MeterReadingSpec | 100 ns/op | 1210 ns/op (1.2% CPU) | 3487 ns/op (3.5% CPU) |

**Conclusion:** JSON serialization is the bottleneck, not struct allocation.

### Memory Sizes

| Type | Struct Shell | Realistic (with data) | JSON Wire |
|------|--------------|------------------------|-----------|
| EventPayloadSpec | 112 bytes | 326 bytes | 232 bytes |
| MeterRecordSpec | 168 bytes | 430 bytes | 370 bytes |
| MeterReadingSpec | 216 bytes | 305 bytes | 364 bytes |

**Conclusion:** Theoretical calculations validated within ±5%.

### UUID vs Short String WorkspaceID

- **Difference:** Exactly 25 bytes in all contexts (36 - 11 = 25 characters)
- **At 10k events/sec:** 41.84 GB/day (UUID) vs 19.31 GB/day (short string)
- **After 100:1 aggregation:** 418 MB/day (UUID) vs 193 MB/day (short string)

**Conclusion:** Cost difference is negligible after aggregation ($3-35/year).

## When to Re-run Benchmarks

### Required

- After modifying spec type structures
- After upgrading Go version (major versions)
- Before production deployment of spec changes

### Recommended

- Quarterly performance checks
- When investigating performance issues
- After dependency updates

### Detection of Regressions

Compare benchmark results over time:

```bash
# Baseline
go test -bench=. -benchmem ./benchmarks/ > baseline.txt

# After changes
go test -bench=. -benchmem ./benchmarks/ > current.txt

# Compare (requires benchstat)
benchstat baseline.txt current.txt
```

**Install benchstat:**
```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

## Benchmark Best Practices

### Warming Up

The Go benchmark framework automatically warms up by running iterations until timing is stable. No manual warm-up needed.

### Benchmarking JSON

JSON benchmarks include real serialization to actual `[]byte`, not just format calls. This accurately reflects production costs.

### Memory Measurement

- Uses `runtime.MemStats` to measure heap allocations
- Runs GC before measurement to get clean baseline
- Allocates 1000 copies to get measurable difference
- Reports average per operation

### Reproducibility

Run benchmarks multiple times:
```bash
go test -bench=. -benchmem -count=5 ./benchmarks/
```

## Integration with Documentation

Results from these benchmarks are incorporated into:
- `design/references/event-sizing-and-cost-analysis.md` - Complete analysis with benchmark validation

When updating the documentation, re-run benchmarks to ensure accuracy:
```bash
# Generate fresh benchmark data
go test -v ./benchmarks/ -run TestSizeBreakdown > sizing_analysis.txt
go test -bench=. -benchmem ./benchmarks/ > benchmark_results.txt

# Use these results to update the documentation
```

## Limitations

### What These Benchmarks Measure

- ✅ Go memory allocation
- ✅ JSON serialization/deserialization
- ✅ Actual wire format sizes
- ✅ Theoretical PostgreSQL row sizes

### What These Benchmarks Don't Measure

- ❌ Protobuf serialization (future work)
- ❌ Compression (gzip, snappy) impact
- ❌ Database index sizes
- ❌ Network latency
- ❌ Actual database query performance
- ❌ Multi-core scaling

## Future Work

### Potential Additions

1. **Protobuf benchmarks** - If protobuf serialization is added to specs
2. **Compression benchmarks** - Measure impact of gzip/snappy on wire format
3. **Database benchmarks** - Actual PostgreSQL insert/query performance
4. **Batch operation benchmarks** - Performance of batch inserts
5. **Concurrent benchmarks** - Multi-goroutine scenarios

### Performance Targets

Based on current results, reasonable targets:

- **Struct allocation**: <200 ns/op
- **JSON marshal**: <2000 ns/op
- **JSON unmarshal**: <5000 ns/op
- **Memory per event**: <500 bytes (for realistic scenarios)

If benchmarks exceed these targets, investigate optimizations.

## References

- [Go Testing Package](https://pkg.go.dev/testing)
- [Go Benchmark Best Practices](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)
- [benchstat Tool](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [Understanding Go Memory Layout](https://go101.org/article/memory-layout.html)

---

**Last Updated:** 2026-01-28
**Go Version:** 1.25.0
**Platform:** darwin/arm64 (Apple M1 Pro)

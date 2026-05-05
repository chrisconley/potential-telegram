# Examples

Each directory is a runnable program with a small `main_test.go` that pins the expected output. Run any one with `go run ./examples/<name>` from the metron root, or run the whole suite with `go test ./examples/...`.

| Example | Demonstrates |
|---|---|
| [`hello/`](hello/) | Time-weighted gauge — seat counts averaged across a billing window. |
| [`api-calls/`](api-calls/) | Counter sum — total tokens consumed in a day. |
| [`llm-tokens/`](llm-tokens/) | Bundled observations — one event with two measurements lands as one record (atomicity). |
| [`conditional-tier/`](conditional-tier/) | Filters — extract only when a dimension matches (tier-aware billing). |
| [`compute-session/`](compute-session/) | Time-spanning observations — `[start, end]` windows for activity over a period. |

## Hello world: time-weighted gauge

[`hello/`](hello/) — Two seat-count readings billed as a 30-day time-weighted average. The customer held 10 seats for 20 days then 15 for 10; `time-weighted-avg` returns `11.666…` at full precision (rounded to 11.67 for display). A naive arithmetic mean of 10 and 15 would give 12.5.

```
customer:acme-corp used 11.67 seats (time-weighted-avg) from 2024-01-01 to 2024-01-31
```

## Counter sum

[`api-calls/`](api-calls/) — Three API calls on the same day, summed for daily token usage. The most common usage-billing case: total quantity consumed.

```
customer:acme consumed 2000 tokens on 2024-01-15 (3 events, sum)
```

## Bundled observations (atomicity)

[`llm-tokens/`](llm-tokens/) — One LLM completion event carries both `input_tokens` and `output_tokens`. The metering config extracts both, and metron returns one `MeterRecord` with two `Observation`s bundled inside it. They persist atomically — either both observations land or neither does — keyed by the source event ID.

```
event completion_42 -> 1 MeterRecord(s)
  record completion_42 contains 2 observations:
    450 input-tokens
    890 output-tokens
```

## Conditional metering (filter)

[`conditional-tier/`](conditional-tier/) — Three API requests with mixed `tier` dimensions; the extraction has a `Filter` that only fires when `tier == "premium"`. Free-tier events drop out; premium events get billed under the `premium-tokens` unit.

```
metered 2/3 events into records (free-tier filtered out)
sum premium-tokens for the day: 1300 (from 2 events)
```

## Time-spanning observations

[`compute-session/`](compute-session/) — Three compute sessions, each with an explicit `[start, end]` window, summed for daily compute-hours. Observations carry their own temporal extent: instant `[T, T]` for gauges, spanning `[T1, T2]` for activity over a period.

The `Meter()` helper extracts instant observations from event properties. For richer event shapes that already carry a `[start, end]` window, construct `MeterRecord`s directly using `specs.NewSpanObservation`.

```
customer:acme used 17 compute-hours across 3 sessions on 2024-01-15
```

## Adding a new example

Use [`hello/`](hello/) as the template: a small `package main` plus a `main_test.go` that runs `go run .` and asserts the printed output. The test catches drift before the table above goes stale.

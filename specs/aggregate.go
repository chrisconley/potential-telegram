package specs

// Aggregate transforms MeterRecords into a MeterReading by applying aggregation over a time window.
//
// Process:
//  1. Apply aggregation type (sum, max, time-weighted-avg, latest, min)
//  2. For gauges (time-weighted-avg): use lastBeforeWindow to carry forward initial state
//  3. Compute aggregated measurement
//  4. Create MeterReading with result
//
// Returns MeterReading containing the aggregated measurement over the window.
// Returns error if no records available or aggregation fails.
//
// This is the spec-level interface using only primitive types.
// See internal.Aggregate for the reference implementation.
type Aggregate func(
	recordsInWindow []MeterRecordSpec,
	lastBeforeWindow *MeterRecordSpec,
	config AggregateConfigSpec,
) (MeterReadingSpec, error)

// AggregateConfigSpec defines how to aggregate meter records into a meter reading.
//
// Specifies the aggregation strategy and the time window over which to aggregate.
// All meter records with the same subject and unit within the window are combined
// according to the specified aggregation type.
type AggregateConfigSpec struct {
	// Aggregation strategy to apply.
	//
	// Determines how individual meter record quantities combine into the final reading:
	//   - "sum": Add all quantities together (e.g., total API calls, total tokens consumed)
	//   - "max": Take the maximum quantity (e.g., peak concurrent connections)
	//   - "min": Take the minimum quantity
	//   - "latest": Use the most recent quantity by RecordedAt timestamp
	//   - "time-weighted-avg": Compute average weighted by duration between records
	//     (e.g., average seat count, treating each record as a step function until the next)
	Aggregation string `json:"aggregation"`

	// Time window for aggregation.
	//
	// Defines the half-open interval [Window.Start, Window.End) for this aggregation.
	// Only meter records with RecordedAt within this window are included. Typically
	// corresponds to a billing period (hour, day, month).
	Window TimeWindowSpec `json:"window"`
}

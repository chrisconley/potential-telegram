// Hello-world for metron.
//
// Meters two gauge events (seat count at two points in time), then aggregates
// them with time-weighted-avg over a 30-day window.
//
//	customer:acme-corp used 11.67 seats (time-weighted-avg) from 2024-01-01 to 2024-01-31
//
// Why 11.67? The customer had 10 seats for 20 days, then 15 seats for 10 days.
// Time-weighted average: (10×20 + 15×10) / 30 = 11.666... → 11.67.
// A naive mean of 10 and 15 would give 12.5 — but that ignores how long
// each value was in effect.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cockroachdb/apd/v3"

	"github.com/chrisconley/metron/internal"
	"github.com/chrisconley/metron/specs"
)

func main() {
	// Extract the "seats" property as an observation with unit "seats".
	// Any other property (region, plan, etc.) would flow through as a dimension.
	meteringConfig := specs.MeteringConfigSpec{
		Observations: []specs.ObservationExtractionSpec{
			{SourceProperty: "seats", Unit: "seats"},
		},
	}

	// Two gauge events: 10 seats at Jan 1, then 15 seats at Jan 21.
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

	// Stage 1 — Meter: event → records.
	var records []specs.MeterRecordSpec
	for _, event := range events {
		recs, err := internal.Meter(event, meteringConfig)
		if err != nil {
			log.Fatalf("meter: %v", err)
		}
		records = append(records, recs...)
	}

	// Stage 2 — Aggregate: records → one reading over the billing window.
	aggregateConfig := specs.AggregateConfigSpec{
		Aggregation: "time-weighted-avg",
		Window: specs.TimeWindowSpec{
			Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		},
	}
	reading, err := internal.Aggregate(records, nil, aggregateConfig)
	if err != nil {
		log.Fatalf("aggregate: %v", err)
	}

	// The underlying quantity is exact (11.666…); round to cents for display.
	value := reading.ComputedValues[0]
	fmt.Printf("%s used %s %s (%s) from %s to %s\n",
		reading.Subject,
		roundCents(value.Quantity),
		value.Unit,
		value.Aggregation,
		reading.Window.Start.Format("2006-01-02"),
		reading.Window.End.Format("2006-01-02"),
	)
}

// roundCents rounds a decimal string to two places using apd (no floats).
func roundCents(s string) string {
	var parsed apd.Decimal
	if _, _, err := parsed.SetString(s); err != nil {
		return s
	}
	var rounded apd.Decimal
	ctx := apd.BaseContext.WithPrecision(34)
	_, _ = ctx.Quantize(&rounded, &parsed, -2)
	return rounded.String()
}

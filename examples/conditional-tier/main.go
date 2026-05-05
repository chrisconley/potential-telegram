// Conditional metering: extract only when a dimension matches.
//
// Three API requests arrive with mixed tier dimensions. The metering config
// has one extraction with a Filter — extract "tokens" as "premium-tokens"
// only when the tier dimension equals "premium". Free-tier events produce no
// observations; premium-tier events do.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/chrisconley/metron/internal"
	"github.com/chrisconley/metron/specs"
)

func main() {
	cfg := specs.MeteringConfigSpec{
		Observations: []specs.ObservationExtractionSpec{
			{
				SourceProperty: "tokens",
				Unit:           "premium-tokens",
				Filter:         &specs.FilterSpec{Property: "tier", Equals: "premium"},
			},
		},
	}

	events := []specs.EventPayloadSpec{
		{
			ID: "req_1", Type: "api.request",
			WorkspaceID: "acme", UniverseID: "production",
			Subject: "customer:acme",
			Time:    time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			Properties: map[string]string{"tokens": "500", "tier": "premium"},
		},
		{
			ID: "req_2", Type: "api.request",
			WorkspaceID: "acme", UniverseID: "production",
			Subject: "customer:acme",
			Time:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			Properties: map[string]string{"tokens": "300", "tier": "free"},
		},
		{
			ID: "req_3", Type: "api.request",
			WorkspaceID: "acme", UniverseID: "production",
			Subject: "customer:acme",
			Time:    time.Date(2024, 1, 15, 16, 0, 0, 0, time.UTC),
			Properties: map[string]string{"tokens": "800", "tier": "premium"},
		},
	}

	var records []specs.MeterRecordSpec
	for _, e := range events {
		recs, err := internal.Meter(e, cfg)
		if err != nil {
			log.Fatal(err)
		}
		records = append(records, recs...)
	}

	fmt.Printf("metered %d/%d events into records (free-tier filtered out)\n",
		len(records), len(events))

	reading, err := internal.Aggregate(records, nil, specs.AggregateConfigSpec{
		Aggregation: "sum",
		Window: specs.TimeWindowSpec{
			Start: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	v := reading.ComputedValues[0]
	fmt.Printf("%s %s for the day: %s (from %d events)\n",
		v.Aggregation, v.Unit, v.Quantity, reading.RecordCount)
}

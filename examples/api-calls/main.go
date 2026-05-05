// Counter sum: total tokens consumed in a day.
//
// Three API request events on the same day. Each carries a "tokens" property
// that gets extracted as an observation; "endpoint" rides through as a
// dimension. Aggregating with sum gives the day's total token usage.
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
			{SourceProperty: "tokens", Unit: "tokens"},
		},
	}

	events := []specs.EventPayloadSpec{
		{
			ID: "req_1", Type: "api.request",
			WorkspaceID: "acme", UniverseID: "production",
			Subject:    "customer:acme",
			Time:       time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			Properties: map[string]string{"tokens": "450", "endpoint": "/chat"},
		},
		{
			ID: "req_2", Type: "api.request",
			WorkspaceID: "acme", UniverseID: "production",
			Subject:    "customer:acme",
			Time:       time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
			Properties: map[string]string{"tokens": "1200", "endpoint": "/chat"},
		},
		{
			ID: "req_3", Type: "api.request",
			WorkspaceID: "acme", UniverseID: "production",
			Subject:    "customer:acme",
			Time:       time.Date(2024, 1, 15, 22, 15, 0, 0, time.UTC),
			Properties: map[string]string{"tokens": "350", "endpoint": "/embed"},
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
	fmt.Printf("%s consumed %s %s on 2024-01-15 (%d events, %s)\n",
		reading.Subject, v.Quantity, v.Unit, reading.RecordCount, v.Aggregation)
}

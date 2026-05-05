// Time-spanning observations: compute sessions with explicit [start, end].
//
// Each compute session ran for some duration, and the billable quantity is the
// session's hour count. The session's temporal extent is the observation's
// own Window — Start < End — distinct from the instant gauge case (Start == End).
//
// The Meter() helper extracts instant observations from event properties; for
// richer event shapes that already carry a [start, end] window, construct
// MeterRecords directly using NewSpanObservation.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/chrisconley/metron/internal"
	"github.com/chrisconley/metron/specs"
)

func main() {
	sessions := []struct {
		id          string
		start, end  time.Time
		computeHrs  string
	}{
		{
			id:         "session_1",
			start:      time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
			end:        time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			computeHrs: "4",
		},
		{
			id:         "session_2",
			start:      time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
			end:        time.Date(2024, 1, 15, 22, 0, 0, 0, time.UTC),
			computeHrs: "8",
		},
		{
			id:         "session_3",
			start:      time.Date(2024, 1, 15, 18, 0, 0, 0, time.UTC),
			end:        time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
			computeHrs: "5",
		},
	}

	now := time.Date(2024, 1, 16, 0, 0, 1, 0, time.UTC)

	var records []specs.MeterRecordSpec
	for _, s := range sessions {
		obs, err := specs.NewSpanObservation(s.computeHrs, "compute-hours", s.start, s.end)
		if err != nil {
			log.Fatal(err)
		}
		records = append(records, specs.MeterRecordSpec{
			ID:            s.id,
			WorkspaceID:   "acme",
			UniverseID:    "production",
			Subject:       "customer:acme",
			ObservedAt:    s.end,
			Observations:  []specs.ObservationSpec{obs},
			SourceEventID: s.id,
			MeteredAt:     now,
		})
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
	fmt.Printf("%s used %s %s across %d sessions on 2024-01-15\n",
		reading.Subject, v.Quantity, v.Unit, reading.RecordCount)
}

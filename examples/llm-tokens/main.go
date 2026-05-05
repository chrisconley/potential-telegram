// Bundled observations: one LLM event, two measurements, one record.
//
// An LLM completion event carries both input_tokens and output_tokens. The
// metering config extracts both, and metron returns one MeterRecord with two
// Observations bundled inside it. They persist atomically — either both
// observations land or neither does — keyed by the source event ID. No
// partial double-billing.
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
			{SourceProperty: "input_tokens", Unit: "input-tokens"},
			{SourceProperty: "output_tokens", Unit: "output-tokens"},
		},
	}

	event := specs.EventPayloadSpec{
		ID: "completion_42", Type: "llm.completion",
		WorkspaceID: "acme", UniverseID: "production",
		Subject: "customer:acme",
		Time:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Properties: map[string]string{
			"input_tokens":  "450",
			"output_tokens": "890",
			"model":         "gpt-4",
		},
	}

	records, err := internal.Meter(event, cfg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("event %s -> %d MeterRecord(s)\n", event.ID, len(records))
	for _, r := range records {
		fmt.Printf("  record %s contains %d observations:\n", r.ID, len(r.Observations))
		for _, obs := range r.Observations {
			fmt.Printf("    %s %s\n", obs.Quantity, obs.Unit)
		}
	}
}

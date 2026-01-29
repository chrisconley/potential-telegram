package benchmarks

import (
	"encoding/json"
	"testing"
	"time"

	"metering-spec/specs"
)

// Benchmark MeterReadingSpec with minimal data
func BenchmarkMeterReading_Minimal_Memory(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = specs.MeterReadingSpec{
			ID:           "",
			WorkspaceID:  "",
			UniverseID:   "",
			Subject:      "",
			Window:       specs.TimeWindowSpec{Start: time.Time{}, End: time.Time{}},
			ComputedValues: []specs.ComputedValueSpec{
				{Quantity: "", Unit: "", Aggregation: "sum"},
			},
			Aggregation:  "",
			RecordCount:  0,
			CreatedAt:    time.Time{},
			MaxMeteredAt: time.Time{},
		}
	}
}

// Benchmark MeterReadingSpec with realistic data
func BenchmarkMeterReading_Realistic_Memory(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = specs.MeterReadingSpec{
			ID:          "mrd_550e8400-e29b-41d4-a716-446655440000",
			WorkspaceID: "ws_a1b2c3d4",
			UniverseID:  "prod",
			Subject:     "customer:cust_abc123",
			Window: specs.TimeWindowSpec{
				Start: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			},
			ComputedValues: []specs.ComputedValueSpec{
				{Quantity: "12500", Unit: "tokens", Aggregation: "sum"},
			},
			Aggregation:  "sum",
			RecordCount:  1250,
			CreatedAt:    time.Now(),
			MaxMeteredAt: time.Now(),
		}
	}
}

// Benchmark MeterReadingSpec with time-weighted average
func BenchmarkMeterReading_TimeWeightedAvg_Memory(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = specs.MeterReadingSpec{
			ID:          "mrd_550e8400-e29b-41d4-a716-446655440000",
			WorkspaceID: "ws_a1b2c3d4",
			UniverseID:  "prod",
			Subject:     "customer:cust_abc123",
			Window: specs.TimeWindowSpec{
				Start: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			},
			ComputedValues: []specs.ComputedValueSpec{
				{Quantity: "12.32", Unit: "seats", Aggregation: "time-weighted-avg"},
			},
			Aggregation:  "time-weighted-avg",
			RecordCount:  156,
			CreatedAt:    time.Now(),
			MaxMeteredAt: time.Now(),
		}
	}
}

// Benchmark JSON serialization of realistic MeterReadingSpec
func BenchmarkMeterReading_Realistic_JSONMarshal(b *testing.B) {
	reading := specs.MeterReadingSpec{
		ID:          "mrd_550e8400-e29b-41d4-a716-446655440000",
		WorkspaceID: "ws_a1b2c3d4",
		UniverseID:  "prod",
		Subject:     "customer:cust_abc123",
		Window: specs.TimeWindowSpec{
			Start: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		ComputedValues: []specs.ComputedValueSpec{
			{Quantity: "12500", Unit: "tokens", Aggregation: "sum"},
		},
		Aggregation:  "sum",
		RecordCount:  1250,
		CreatedAt:    time.Date(2024, 3, 1, 0, 0, 5, 0, time.UTC),
		MaxMeteredAt: time.Date(2024, 2, 28, 23, 59, 59, 0, time.UTC),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(reading)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark JSON deserialization of realistic MeterReadingSpec
func BenchmarkMeterReading_Realistic_JSONUnmarshal(b *testing.B) {
	jsonData := []byte(`{
		"id": "mrd_550e8400-e29b-41d4-a716-446655440000",
		"workspaceID": "ws_a1b2c3d4",
		"universeID": "prod",
		"subject": "customer:cust_abc123",
		"window": {
			"start": "2024-02-01T00:00:00Z",
			"end": "2024-03-01T00:00:00Z"
		},
		"value": {
			"quantity": "12500",
			"unit": "tokens"
		},
		"aggregation": "sum",
		"recordCount": 1250,
		"createdAt": "2024-03-01T00:00:05Z",
		"maxMeteredAt": "2024-02-28T23:59:59Z"
	}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var reading specs.MeterReadingSpec
		err := json.Unmarshal(jsonData, &reading)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark JSON roundtrip
func BenchmarkMeterReading_Realistic_JSONRoundtrip(b *testing.B) {
	reading := specs.MeterReadingSpec{
		ID:          "mrd_550e8400-e29b-41d4-a716-446655440000",
		WorkspaceID: "ws_a1b2c3d4",
		UniverseID:  "prod",
		Subject:     "customer:cust_abc123",
		Window: specs.TimeWindowSpec{
			Start: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		ComputedValues: []specs.ComputedValueSpec{
			{Quantity: "12500", Unit: "tokens", Aggregation: "sum"},
		},
		Aggregation:  "sum",
		RecordCount:  1250,
		CreatedAt:    time.Date(2024, 3, 1, 0, 0, 5, 0, time.UTC),
		MaxMeteredAt: time.Date(2024, 2, 28, 23, 59, 59, 0, time.UTC),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		jsonData, err := json.Marshal(reading)
		if err != nil {
			b.Fatal(err)
		}

		var decoded specs.MeterReadingSpec
		err = json.Unmarshal(jsonData, &decoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Measure actual JSON wire size
func BenchmarkMeterReading_JSONSize(b *testing.B) {
	scenarios := []struct {
		name    string
		reading specs.MeterReadingSpec
	}{
		{
			name: "Minimal",
			reading: specs.MeterReadingSpec{
				ID:           "",
				WorkspaceID:  "",
				UniverseID:   "",
				Subject:      "",
				Window:       specs.TimeWindowSpec{Start: time.Time{}, End: time.Time{}},
				ComputedValues: []specs.ComputedValueSpec{
					{Quantity: "", Unit: "", Aggregation: "sum"},
				},
				Aggregation:  "",
				RecordCount:  0,
				CreatedAt:    time.Time{},
				MaxMeteredAt: time.Time{},
			},
		},
		{
			name: "Realistic_Sum",
			reading: specs.MeterReadingSpec{
				ID:          "mrd_550e8400-e29b-41d4-a716-446655440000",
				WorkspaceID: "ws_a1b2c3d4",
				UniverseID:  "prod",
				Subject:     "customer:cust_abc123",
				Window: specs.TimeWindowSpec{
					Start: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
				},
				ComputedValues: []specs.ComputedValueSpec{
					{Quantity: "12500", Unit: "tokens", Aggregation: "sum"},
				},
				Aggregation:  "sum",
				RecordCount:  1250,
				CreatedAt:    time.Date(2024, 3, 1, 0, 0, 5, 0, time.UTC),
				MaxMeteredAt: time.Date(2024, 2, 28, 23, 59, 59, 0, time.UTC),
			},
		},
		{
			name: "Realistic_TimeWeightedAvg",
			reading: specs.MeterReadingSpec{
				ID:          "mrd_550e8400-e29b-41d4-a716-446655440000",
				WorkspaceID: "ws_a1b2c3d4",
				UniverseID:  "prod",
				Subject:     "customer:cust_abc123",
				Window: specs.TimeWindowSpec{
					Start: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
				},
				ComputedValues: []specs.ComputedValueSpec{
					{Quantity: "12.32", Unit: "seats", Aggregation: "time-weighted-avg"},
				},
				Aggregation:  "time-weighted-avg",
				RecordCount:  156,
				CreatedAt:    time.Date(2024, 3, 1, 0, 0, 5, 0, time.UTC),
				MaxMeteredAt: time.Date(2024, 2, 28, 23, 59, 59, 0, time.UTC),
			},
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			jsonData, err := json.Marshal(scenario.reading)
			if err != nil {
				b.Fatal(err)
			}

			b.ReportMetric(float64(len(jsonData)), "bytes")
			b.Logf("%s JSON size: %d bytes", scenario.name, len(jsonData))
		})
	}
}

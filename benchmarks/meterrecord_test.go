package benchmarks

import (
	"encoding/json"
	"testing"
	"time"

	"metering-spec/specs"
)

// Benchmark MeterRecordSpec with minimal data
func BenchmarkMeterRecord_Minimal_Memory(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = specs.MeterRecordSpec{
			ID:            "",
			WorkspaceID:   "",
			UniverseID:    "",
			Subject:       "",
			RecordedAt:    time.Time{},
			Measurement:   specs.MeasurementSpec{Quantity: "", Unit: ""},
			Dimensions:    nil,
			SourceEventID: "",
			MeteredAt:     time.Time{},
		}
	}
}

// Benchmark MeterRecordSpec with realistic data
func BenchmarkMeterRecord_Realistic_Memory(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = specs.MeterRecordSpec{
			ID:          "mr_550e8400-e29b-41d4-a716-446655440000",
			WorkspaceID: "ws_a1b2c3d4",
			UniverseID:  "prod",
			Subject:     "customer:cust_abc123",
			RecordedAt:  time.Now(),
			Measurement: specs.MeasurementSpec{
				Quantity: "1500",
				Unit:     "tokens",
			},
			Dimensions: map[string]string{
				"model":    "gpt-4",
				"endpoint": "/api/completions",
			},
			SourceEventID: "evt_550e8400-e29b-41d4-a716-446655440000",
			MeteredAt:     time.Now(),
		}
	}
}

// Benchmark MeterRecordSpec with large dimensions
func BenchmarkMeterRecord_LargeDimensions_Memory(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = specs.MeterRecordSpec{
			ID:          "mr_550e8400-e29b-41d4-a716-446655440000",
			WorkspaceID: "ws_a1b2c3d4",
			UniverseID:  "prod",
			Subject:     "customer:cust_abc123",
			RecordedAt:  time.Now(),
			Measurement: specs.MeasurementSpec{
				Quantity: "1500",
				Unit:     "tokens",
			},
			Dimensions: map[string]string{
				"model":            "gpt-4",
				"endpoint":         "/api/completions",
				"status_code":      "200",
				"region":           "us-east-1",
				"cached":           "false",
				"response_time_ms": "245",
				"input_tokens":     "450",
				"output_tokens":    "890",
				"feature_flag":     "new_ui_enabled",
				"team":             "engineering",
			},
			SourceEventID: "evt_550e8400-e29b-41d4-a716-446655440000",
			MeteredAt:     time.Now(),
		}
	}
}

// Benchmark JSON serialization of realistic MeterRecordSpec
func BenchmarkMeterRecord_Realistic_JSONMarshal(b *testing.B) {
	record := specs.MeterRecordSpec{
		ID:          "mr_550e8400-e29b-41d4-a716-446655440000",
		WorkspaceID: "ws_a1b2c3d4",
		UniverseID:  "prod",
		Subject:     "customer:cust_abc123",
		RecordedAt:  time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		Measurement: specs.MeasurementSpec{
			Quantity: "1500",
			Unit:     "tokens",
		},
		Dimensions: map[string]string{
			"model":    "gpt-4",
			"endpoint": "/api/completions",
		},
		SourceEventID: "evt_550e8400-e29b-41d4-a716-446655440000",
		MeteredAt:     time.Date(2024, 1, 1, 10, 0, 1, 0, time.UTC),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(record)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark JSON deserialization of realistic MeterRecordSpec
func BenchmarkMeterRecord_Realistic_JSONUnmarshal(b *testing.B) {
	jsonData := []byte(`{
		"id": "mr_550e8400-e29b-41d4-a716-446655440000",
		"workspaceID": "ws_a1b2c3d4",
		"universeID": "prod",
		"subject": "customer:cust_abc123",
		"recordedAt": "2024-01-01T10:00:00Z",
		"measurement": {
			"quantity": "1500",
			"unit": "tokens"
		},
		"dimensions": {
			"model": "gpt-4",
			"endpoint": "/api/completions"
		},
		"sourceEventID": "evt_550e8400-e29b-41d4-a716-446655440000",
		"meteredAt": "2024-01-01T10:00:01Z"
	}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var record specs.MeterRecordSpec
		err := json.Unmarshal(jsonData, &record)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark JSON roundtrip
func BenchmarkMeterRecord_Realistic_JSONRoundtrip(b *testing.B) {
	record := specs.MeterRecordSpec{
		ID:          "mr_550e8400-e29b-41d4-a716-446655440000",
		WorkspaceID: "ws_a1b2c3d4",
		UniverseID:  "prod",
		Subject:     "customer:cust_abc123",
		RecordedAt:  time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		Measurement: specs.MeasurementSpec{
			Quantity: "1500",
			Unit:     "tokens",
		},
		Dimensions: map[string]string{
			"model":    "gpt-4",
			"endpoint": "/api/completions",
		},
		SourceEventID: "evt_550e8400-e29b-41d4-a716-446655440000",
		MeteredAt:     time.Date(2024, 1, 1, 10, 0, 1, 0, time.UTC),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		jsonData, err := json.Marshal(record)
		if err != nil {
			b.Fatal(err)
		}

		var decoded specs.MeterRecordSpec
		err = json.Unmarshal(jsonData, &decoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Measure actual JSON wire size
func BenchmarkMeterRecord_JSONSize(b *testing.B) {
	scenarios := []struct {
		name   string
		record specs.MeterRecordSpec
	}{
		{
			name: "Minimal",
			record: specs.MeterRecordSpec{
				ID:            "",
				WorkspaceID:   "",
				UniverseID:    "",
				Subject:       "",
				RecordedAt:    time.Time{},
				Measurement:   specs.MeasurementSpec{Quantity: "", Unit: ""},
				Dimensions:    nil,
				SourceEventID: "",
				MeteredAt:     time.Time{},
			},
		},
		{
			name: "Realistic",
			record: specs.MeterRecordSpec{
				ID:          "mr_550e8400-e29b-41d4-a716-446655440000",
				WorkspaceID: "ws_a1b2c3d4",
				UniverseID:  "prod",
				Subject:     "customer:cust_abc123",
				RecordedAt:  time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				Measurement: specs.MeasurementSpec{
					Quantity: "1500",
					Unit:     "tokens",
				},
				Dimensions: map[string]string{
					"model":    "gpt-4",
					"endpoint": "/api/completions",
				},
				SourceEventID: "evt_550e8400-e29b-41d4-a716-446655440000",
				MeteredAt:     time.Date(2024, 1, 1, 10, 0, 1, 0, time.UTC),
			},
		},
		{
			name: "LargeDimensions",
			record: specs.MeterRecordSpec{
				ID:          "mr_550e8400-e29b-41d4-a716-446655440000",
				WorkspaceID: "ws_a1b2c3d4",
				UniverseID:  "prod",
				Subject:     "customer:cust_abc123",
				RecordedAt:  time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				Measurement: specs.MeasurementSpec{
					Quantity: "1500",
					Unit:     "tokens",
				},
				Dimensions: map[string]string{
					"model":            "gpt-4",
					"endpoint":         "/api/completions",
					"status_code":      "200",
					"region":           "us-east-1",
					"cached":           "false",
					"response_time_ms": "245",
					"input_tokens":     "450",
					"output_tokens":    "890",
					"feature_flag":     "new_ui_enabled",
					"team":             "engineering",
				},
				SourceEventID: "evt_550e8400-e29b-41d4-a716-446655440000",
				MeteredAt:     time.Date(2024, 1, 1, 10, 0, 1, 0, time.UTC),
			},
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			jsonData, err := json.Marshal(scenario.record)
			if err != nil {
				b.Fatal(err)
			}

			b.ReportMetric(float64(len(jsonData)), "bytes")
			b.Logf("%s JSON size: %d bytes", scenario.name, len(jsonData))
		})
	}
}

package benchmarks

import (
	"encoding/json"
	"testing"
	"time"

	"metering-spec/specs"
)

// Benchmark EventPayloadSpec with minimal data (empty strings)
func BenchmarkEventPayload_Minimal_Memory(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = specs.EventPayloadSpec{
			ID:          "",
			WorkspaceID: "",
			UniverseID:  "",
			Type:        "",
			Subject:     "",
			Time:        time.Time{},
			Properties:  nil,
		}
	}
}

// Benchmark EventPayloadSpec with realistic data
func BenchmarkEventPayload_Realistic_Memory(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = specs.EventPayloadSpec{
			ID:          "550e8400-e29b-41d4-a716-446655440000",
			WorkspaceID: "ws_a1b2c3d4",
			UniverseID:  "prod",
			Type:        "api.request",
			Subject:     "customer:cust_abc123",
			Time:        time.Now(),
			Properties: map[string]string{
				"endpoint": "/api/users",
				"tokens":   "1500",
			},
		}
	}
}

// Benchmark EventPayloadSpec with UUID WorkspaceID
func BenchmarkEventPayload_UUIDWorkspace_Memory(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = specs.EventPayloadSpec{
			ID:          "550e8400-e29b-41d4-a716-446655440000",
			WorkspaceID: "550e8400-e29b-41d4-a716-446655440001", // Full UUID
			UniverseID:  "prod",
			Type:        "api.request",
			Subject:     "customer:cust_abc123",
			Time:        time.Now(),
			Properties: map[string]string{
				"endpoint": "/api/users",
				"tokens":   "1500",
			},
		}
	}
}

// Benchmark EventPayloadSpec with large Properties map
func BenchmarkEventPayload_LargeProperties_Memory(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = specs.EventPayloadSpec{
			ID:          "550e8400-e29b-41d4-a716-446655440000",
			WorkspaceID: "ws_a1b2c3d4",
			UniverseID:  "prod",
			Type:        "api.request",
			Subject:     "customer:cust_abc123",
			Time:        time.Now(),
			Properties: map[string]string{
				"endpoint":         "/api/users",
				"tokens":           "1500",
				"status_code":      "200",
				"response_time_ms": "125",
				"region":           "us-east-1",
				"model":            "gpt-4",
				"cached":           "true",
				"user_agent":       "Mozilla/5.0",
				"ip_address":       "192.168.1.1",
				"request_id":       "req_xyz789",
			},
		}
	}
}

// Benchmark JSON serialization of minimal EventPayloadSpec
func BenchmarkEventPayload_Minimal_JSONMarshal(b *testing.B) {
	event := specs.EventPayloadSpec{
		ID:          "",
		WorkspaceID: "",
		UniverseID:  "",
		Type:        "",
		Subject:     "",
		Time:        time.Time{},
		Properties:  nil,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark JSON serialization of realistic EventPayloadSpec
func BenchmarkEventPayload_Realistic_JSONMarshal(b *testing.B) {
	event := specs.EventPayloadSpec{
		ID:          "550e8400-e29b-41d4-a716-446655440000",
		WorkspaceID: "ws_a1b2c3d4",
		UniverseID:  "prod",
		Type:        "api.request",
		Subject:     "customer:cust_abc123",
		Time:        time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		Properties: map[string]string{
			"endpoint": "/api/users",
			"tokens":   "1500",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark JSON deserialization of realistic EventPayloadSpec
func BenchmarkEventPayload_Realistic_JSONUnmarshal(b *testing.B) {
	jsonData := []byte(`{
		"id": "550e8400-e29b-41d4-a716-446655440000",
		"workspaceID": "ws_a1b2c3d4",
		"universeID": "prod",
		"type": "api.request",
		"subject": "customer:cust_abc123",
		"time": "2024-01-01T10:00:00Z",
		"properties": {
			"endpoint": "/api/users",
			"tokens": "1500"
		}
	}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var event specs.EventPayloadSpec
		err := json.Unmarshal(jsonData, &event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark JSON roundtrip (marshal + unmarshal)
func BenchmarkEventPayload_Realistic_JSONRoundtrip(b *testing.B) {
	event := specs.EventPayloadSpec{
		ID:          "550e8400-e29b-41d4-a716-446655440000",
		WorkspaceID: "ws_a1b2c3d4",
		UniverseID:  "prod",
		Type:        "api.request",
		Subject:     "customer:cust_abc123",
		Time:        time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		Properties: map[string]string{
			"endpoint": "/api/users",
			"tokens":   "1500",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		jsonData, err := json.Marshal(event)
		if err != nil {
			b.Fatal(err)
		}

		var decoded specs.EventPayloadSpec
		err = json.Unmarshal(jsonData, &decoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Measure actual JSON wire size for different scenarios
func BenchmarkEventPayload_JSONSize(b *testing.B) {
	scenarios := []struct {
		name  string
		event specs.EventPayloadSpec
	}{
		{
			name: "Minimal",
			event: specs.EventPayloadSpec{
				ID:          "",
				WorkspaceID: "",
				UniverseID:  "",
				Type:        "",
				Subject:     "",
				Time:        time.Time{},
				Properties:  nil,
			},
		},
		{
			name: "ShortStrings",
			event: specs.EventPayloadSpec{
				ID:          "evt_123",
				WorkspaceID: "ws_456",
				UniverseID:  "prod",
				Type:        "api",
				Subject:     "cust_789",
				Time:        time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				Properties:  nil,
			},
		},
		{
			name: "UUID_WorkspaceID",
			event: specs.EventPayloadSpec{
				ID:          "550e8400-e29b-41d4-a716-446655440000",
				WorkspaceID: "550e8400-e29b-41d4-a716-446655440001",
				UniverseID:  "prod",
				Type:        "api.request",
				Subject:     "customer:cust_abc123",
				Time:        time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				Properties: map[string]string{
					"endpoint": "/api/users",
					"tokens":   "1500",
				},
			},
		},
		{
			name: "Realistic",
			event: specs.EventPayloadSpec{
				ID:          "550e8400-e29b-41d4-a716-446655440000",
				WorkspaceID: "ws_a1b2c3d4",
				UniverseID:  "prod",
				Type:        "api.request",
				Subject:     "customer:cust_abc123",
				Time:        time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				Properties: map[string]string{
					"endpoint": "/api/users",
					"tokens":   "1500",
				},
			},
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			jsonData, err := json.Marshal(scenario.event)
			if err != nil {
				b.Fatal(err)
			}

			// Report size in custom metric
			b.ReportMetric(float64(len(jsonData)), "bytes")

			// Log size for documentation
			b.Logf("%s JSON size: %d bytes", scenario.name, len(jsonData))
		})
	}
}

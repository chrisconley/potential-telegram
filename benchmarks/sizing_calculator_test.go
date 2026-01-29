package benchmarks

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"metering-spec/specs"
)

// SizeBreakdown provides size estimates for different contexts
type SizeBreakdown struct {
	GoMemoryEstimate   int // Estimated in-memory struct size
	GoMemoryMeasured   int // Measured from runtime.MemStats
	JSONWireFormat     int // Serialized JSON size
	PostgresEstimate   int // Estimated PostgreSQL row size
	AllocationCount    int // Number of heap allocations
	AllocatedBytes     int64 // Total bytes allocated
}

// Calculate EventPayloadSpec size breakdown
func TestEventPayloadSizeBreakdown(t *testing.T) {
	scenarios := []struct {
		name  string
		event specs.EventPayloadSpec
	}{
		{
			name: "Minimal (all empty)",
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
			name: "Short strings",
			event: specs.EventPayloadSpec{
				ID:          "evt_123",
				WorkspaceID: "ws_456",
				UniverseID:  "prod",
				Type:        "api",
				Subject:     "cust_789",
				Time:        time.Now(),
				Properties:  nil,
			},
		},
		{
			name: "Realistic (short WorkspaceID)",
			event: specs.EventPayloadSpec{
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
			},
		},
		{
			name: "UUID WorkspaceID",
			event: specs.EventPayloadSpec{
				ID:          "550e8400-e29b-41d4-a716-446655440000",
				WorkspaceID: "550e8400-e29b-41d4-a716-446655440001",
				UniverseID:  "prod",
				Type:        "api.request",
				Subject:     "customer:cust_abc123",
				Time:        time.Now(),
				Properties: map[string]string{
					"endpoint": "/api/users",
					"tokens":   "1500",
				},
			},
		},
	}

	t.Log("\n=== EventPayloadSpec Size Analysis ===\n")

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			breakdown := calculateSizeBreakdown(scenario.event)

			t.Logf("\n%s:", scenario.name)
			t.Logf("  Go Memory (estimated): %d bytes", breakdown.GoMemoryEstimate)
			t.Logf("  Go Memory (measured):  %d bytes", breakdown.GoMemoryMeasured)
			t.Logf("  JSON Wire Format:      %d bytes", breakdown.JSONWireFormat)
			t.Logf("  PostgreSQL (estimate): %d bytes", breakdown.PostgresEstimate)
			t.Logf("  Allocations:           %d", breakdown.AllocationCount)
			t.Logf("  Total Allocated:       %d bytes", breakdown.AllocatedBytes)
		})
	}
}

// Calculate MeterRecordSpec size breakdown
func TestMeterRecordSizeBreakdown(t *testing.T) {
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
				ObservedAt:    time.Time{},
				Observations:  []specs.ObservationSpec{{Quantity: "", Unit: "", Window: specs.TimeWindowSpec{Start: time.Time{}, End: time.Time{}}}},
				Dimensions:    nil,
				SourceEventID: "",
				MeteredAt:     time.Time{},
			},
		},
		{
			name: "Realistic",
			record: func() specs.MeterRecordSpec {
				observedAt := time.Now()
				return specs.MeterRecordSpec{
					ID:          "mr_550e8400-e29b-41d4-a716-446655440000",
					WorkspaceID: "ws_a1b2c3d4",
					UniverseID:  "prod",
					Subject:     "customer:cust_abc123",
					ObservedAt:  observedAt,
					Observations: []specs.ObservationSpec{{
						Quantity: "1500",
						Unit:     "tokens",
						Window:   specs.TimeWindowSpec{Start: observedAt, End: observedAt},
					}},
					Dimensions: map[string]string{
						"model":    "gpt-4",
						"endpoint": "/api/completions",
					},
					SourceEventID: "evt_550e8400-e29b-41d4-a716-446655440000",
					MeteredAt:     observedAt,
				}
			}(),
		},
	}

	t.Log("\n=== MeterRecordSpec Size Analysis ===\n")

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			breakdown := calculateSizeBreakdown(scenario.record)

			t.Logf("\n%s:", scenario.name)
			t.Logf("  Go Memory (estimated): %d bytes", breakdown.GoMemoryEstimate)
			t.Logf("  Go Memory (measured):  %d bytes", breakdown.GoMemoryMeasured)
			t.Logf("  JSON Wire Format:      %d bytes", breakdown.JSONWireFormat)
			t.Logf("  PostgreSQL (estimate): %d bytes", breakdown.PostgresEstimate)
			t.Logf("  Allocations:           %d", breakdown.AllocationCount)
			t.Logf("  Total Allocated:       %d bytes", breakdown.AllocatedBytes)
		})
	}
}

// Calculate MeterReadingSpec size breakdown
func TestMeterReadingSizeBreakdown(t *testing.T) {
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
			name: "Realistic",
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
				CreatedAt:    time.Now(),
				MaxMeteredAt: time.Now(),
			},
		},
	}

	t.Log("\n=== MeterReadingSpec Size Analysis ===\n")

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			breakdown := calculateSizeBreakdown(scenario.reading)

			t.Logf("\n%s:", scenario.name)
			t.Logf("  Go Memory (estimated): %d bytes", breakdown.GoMemoryEstimate)
			t.Logf("  Go Memory (measured):  %d bytes", breakdown.GoMemoryMeasured)
			t.Logf("  JSON Wire Format:      %d bytes", breakdown.JSONWireFormat)
			t.Logf("  PostgreSQL (estimate): %d bytes", breakdown.PostgresEstimate)
			t.Logf("  Allocations:           %d", breakdown.AllocationCount)
			t.Logf("  Total Allocated:       %d bytes", breakdown.AllocatedBytes)
		})
	}
}

// calculateSizeBreakdown measures and estimates sizes for any value
func calculateSizeBreakdown(v interface{}) SizeBreakdown {
	breakdown := SizeBreakdown{}

	// Measure JSON wire format
	jsonData, err := json.Marshal(v)
	if err == nil {
		breakdown.JSONWireFormat = len(jsonData)
	}

	// Estimate Go memory based on type
	switch val := v.(type) {
	case specs.EventPayloadSpec:
		breakdown.GoMemoryEstimate = estimateEventPayloadSize(val)
		breakdown.PostgresEstimate = estimateEventPayloadPostgresSize(val)
	case specs.MeterRecordSpec:
		breakdown.GoMemoryEstimate = estimateMeterRecordSize(val)
		breakdown.PostgresEstimate = estimateMeterRecordPostgresSize(val)
	case specs.MeterReadingSpec:
		breakdown.GoMemoryEstimate = estimateMeterReadingSize(val)
		breakdown.PostgresEstimate = estimateMeterReadingPostgresSize(val)
	}

	// Measure actual memory allocation
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Allocate multiple copies to get measurable difference
	const iterations = 1000
	slice := make([]interface{}, iterations)
	for i := 0; i < iterations; i++ {
		slice[i] = v
	}

	runtime.ReadMemStats(&m2)

	breakdown.GoMemoryMeasured = int((m2.Alloc - m1.Alloc) / iterations)
	breakdown.AllocationCount = int((m2.Mallocs - m1.Mallocs) / iterations)
	breakdown.AllocatedBytes = int64((m2.TotalAlloc - m1.TotalAlloc) / iterations)

	return breakdown
}

// estimateEventPayloadSize estimates Go memory for EventPayloadSpec
func estimateEventPayloadSize(e specs.EventPayloadSpec) int {
	size := 0

	// String fields (16 bytes header + data)
	size += 16 + len(e.ID)
	size += 16 + len(e.WorkspaceID)
	size += 16 + len(e.UniverseID)
	size += 16 + len(e.Type)
	size += 16 + len(e.Subject)

	// time.Time (3 x int64 = 24 bytes)
	size += 24

	// map[string]string (48 bytes header + entries)
	if e.Properties != nil {
		size += 48
		for k, v := range e.Properties {
			size += (16 + len(k)) + (16 + len(v))
		}
	}

	return size
}

// estimateMeterRecordSize estimates Go memory for MeterRecordSpec
func estimateMeterRecordSize(r specs.MeterRecordSpec) int {
	size := 0

	// String fields
	size += 16 + len(r.ID)
	size += 16 + len(r.WorkspaceID)
	size += 16 + len(r.UniverseID)
	size += 16 + len(r.Subject)
	size += 16 + len(r.SourceEventID)

	// time.Time fields
	size += 24 // ObservedAt
	size += 24 // MeteredAt

	// Observations array
	for _, obs := range r.Observations {
		size += 16 + len(obs.Quantity)
		size += 16 + len(obs.Unit)
		size += 24 // obs.Window.Start
		size += 24 // obs.Window.End
	}

	// Dimensions map
	if r.Dimensions != nil {
		size += 48
		for k, v := range r.Dimensions {
			size += (16 + len(k)) + (16 + len(v))
		}
	}

	return size
}

// estimateMeterReadingSize estimates Go memory for MeterReadingSpec
func estimateMeterReadingSize(r specs.MeterReadingSpec) int {
	size := 0

	// String fields
	size += 16 + len(r.ID)
	size += 16 + len(r.WorkspaceID)
	size += 16 + len(r.UniverseID)
	size += 16 + len(r.Subject)
	size += 16 + len(r.Aggregation)

	// TimeWindowSpec (2 x time.Time)
	size += 24 // Start
	size += 24 // End

	// AggregateSpec
	size += 16 + len(r.ComputedValues[0].Quantity)
	size += 16 + len(r.ComputedValues[0].Unit)

	// int field
	size += 8 // RecordCount

	// time.Time fields
	size += 24 // CreatedAt
	size += 24 // MaxMeteredAt

	return size
}

// PostgreSQL VARCHAR estimates (1 byte length prefix + data for <127 bytes)
func estimateEventPayloadPostgresSize(e specs.EventPayloadSpec) int {
	size := 0

	// VARCHAR fields
	if len(e.ID) > 0 {
		size += 1 + len(e.ID)
	} else {
		size += 1
	}
	if len(e.WorkspaceID) > 0 {
		size += 1 + len(e.WorkspaceID)
	} else {
		size += 1
	}
	if len(e.UniverseID) > 0 {
		size += 1 + len(e.UniverseID)
	} else {
		size += 1
	}
	if len(e.Type) > 0 {
		size += 1 + len(e.Type)
	} else {
		size += 1
	}
	if len(e.Subject) > 0 {
		size += 1 + len(e.Subject)
	} else {
		size += 1
	}

	// TIMESTAMP (8 bytes)
	size += 8

	// JSONB for properties (approximate)
	if e.Properties != nil && len(e.Properties) > 0 {
		propsSize := 1 // JSONB version byte
		for k, v := range e.Properties {
			propsSize += len(k) + len(v) + 4 // overhead
		}
		size += propsSize
	}

	return size
}

func estimateMeterRecordPostgresSize(r specs.MeterRecordSpec) int {
	size := 0

	// VARCHAR fields
	size += 1 + len(r.ID)
	size += 1 + len(r.WorkspaceID)
	size += 1 + len(r.UniverseID)
	size += 1 + len(r.Subject)
	size += 1 + len(r.SourceEventID)
	for _, obs := range r.Observations {
		size += 1 + len(obs.Quantity)
		size += 1 + len(obs.Unit)
	}

	// TIMESTAMPs
	size += 8 // ObservedAt
	size += 8 // MeteredAt
	for range r.Observations {
		size += 8 // Observation.Window.Start
		size += 8 // Observation.Window.End
	}

	// JSONB for dimensions
	if r.Dimensions != nil && len(r.Dimensions) > 0 {
		dimsSize := 1
		for k, v := range r.Dimensions {
			dimsSize += len(k) + len(v) + 4
		}
		size += dimsSize
	}

	return size
}

func estimateMeterReadingPostgresSize(r specs.MeterReadingSpec) int {
	size := 0

	// VARCHAR fields
	size += 1 + len(r.ID)
	size += 1 + len(r.WorkspaceID)
	size += 1 + len(r.UniverseID)
	size += 1 + len(r.Subject)
	size += 1 + len(r.Aggregation)
	size += 1 + len(r.ComputedValues[0].Quantity)
	size += 1 + len(r.ComputedValues[0].Unit)

	// TIMESTAMPs
	size += 8 // Window.Start
	size += 8 // Window.End
	size += 8 // CreatedAt
	size += 8 // MaxMeteredAt

	// INTEGER
	size += 4 // RecordCount

	return size
}

// Test struct sizes using unsafe.Sizeof
func TestStructSizes(t *testing.T) {
	t.Logf("\n=== Struct Sizes (unsafe.Sizeof) ===\n")

	var event specs.EventPayloadSpec
	var record specs.MeterRecordSpec
	var reading specs.MeterReadingSpec
	var observation specs.ObservationSpec
	var computed specs.ComputedValueSpec
	var window specs.TimeWindowSpec

	t.Logf("EventPayloadSpec:  %d bytes", unsafe.Sizeof(event))
	t.Logf("MeterRecordSpec:   %d bytes", unsafe.Sizeof(record))
	t.Logf("MeterReadingSpec:  %d bytes", unsafe.Sizeof(reading))
	t.Logf("ObservationSpec:   %d bytes", unsafe.Sizeof(observation))
	t.Logf("ComputedValueSpec: %d bytes", unsafe.Sizeof(computed))
	t.Logf("TimeWindowSpec:    %d bytes", unsafe.Sizeof(window))
	t.Logf("time.Time:         %d bytes", unsafe.Sizeof(time.Time{}))
	t.Logf("string header:     %d bytes", unsafe.Sizeof(""))
	t.Logf("map header:        %d bytes", unsafe.Sizeof(map[string]string{}))
}

// Calculate costs at scale
func TestScaleCalculations(t *testing.T) {
	const eventsPerSecond = 10000
	const secondsPerDay = 86400
	const eventsPerDay = eventsPerSecond * secondsPerDay
	const daysPerMonth = 30
	const eventsPerMonth = eventsPerDay * daysPerMonth

	scenarios := []struct {
		name             string
		bytesPerEvent    int
		description      string
	}{
		{
			name:             "UUID WorkspaceID",
			bytesPerEvent:    52, // 16 header + 36 chars
			description:      "Full UUID for WorkspaceID",
		},
		{
			name:             "Short String WorkspaceID",
			bytesPerEvent:    24, // 16 header + 8 chars
			description:      "Short string like 'ws_12345' for WorkspaceID",
		},
		{
			name:             "int64 WorkspaceID",
			bytesPerEvent:    8,
			description:      "int64 for WorkspaceID",
		},
	}

	t.Logf("\n=== Scale Impact Analysis ===\n")
	t.Logf("Throughput: %d events/second", eventsPerSecond)
	t.Logf("Daily events: %s", formatNumber(eventsPerDay))
	t.Logf("Monthly events: %s", formatNumber(eventsPerMonth))
	t.Logf("\n")

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			dailyGB := float64(eventsPerDay*scenario.bytesPerEvent) / (1024 * 1024 * 1024)
			monthlyGB := float64(eventsPerMonth*scenario.bytesPerEvent) / (1024 * 1024 * 1024)

			t.Logf("%s (%s):", scenario.name, scenario.description)
			t.Logf("  Bytes per field: %d", scenario.bytesPerEvent)
			t.Logf("  Daily volume:    %.2f GB", dailyGB)
			t.Logf("  Monthly volume:  %.2f GB", monthlyGB)
			t.Logf("")
		})
	}
}

func formatNumber(n int) string {
	return fmt.Sprintf("%d", n)
}

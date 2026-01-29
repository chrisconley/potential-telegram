package internal

import (
	"metering-spec/specs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMeterReading(t *testing.T) {
	t.Run("creates meter reading with all required fields", func(t *testing.T) {
		now := time.Now()
		windowStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		windowEnd := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		spec := specs.MeterReadingSpec{
			ID:          "reading-123",
			WorkspaceID: "workspace-prod",
			UniverseID:  "production",
			Subject:     "customer:acme",
			Window: specs.TimeWindowSpec{
				Start: windowStart,
				End:   windowEnd,
			},
			Value: specs.AggregateSpec{
				Quantity: "1250.50",
				Unit:     "api-tokens",
			},
			Aggregation:  "sum",
			RecordCount:  5,
			CreatedAt:    now,
			MaxMeteredAt: now,
		}

		reading, err := NewMeterReading(spec)

		require.NoError(t, err)
		assert.Equal(t, "reading-123", reading.ID.ToString())
		assert.Equal(t, "workspace-prod", reading.WorkspaceID.ToString())
		assert.Equal(t, "production", reading.UniverseID.ToString())
		assert.Equal(t, "customer:acme", reading.Subject.ToString())
		assert.Equal(t, windowStart, reading.Window.Start().ToTime())
		assert.Equal(t, windowEnd, reading.Window.End().ToTime())
		assert.Equal(t, "1250.50", reading.Value.Quantity().String())
		assert.Equal(t, "api-tokens", reading.Value.Unit().ToString())
		assert.Equal(t, "sum", reading.Aggregation.ToString())
		assert.Equal(t, 5, reading.RecordCount.ToInt())
		assert.Equal(t, now, reading.CreatedAt.ToTime())
		assert.Equal(t, now, reading.MaxMeteredAt.ToTime())
	})

	t.Run("with zero window start returns error", func(t *testing.T) {
		spec := specs.MeterReadingSpec{
			ID:          "reading-123",
			WorkspaceID: "workspace-prod",
			UniverseID:  "production",
			Subject:     "customer:acme",
			Window: specs.TimeWindowSpec{
				Start: time.Time{}, // Zero value
				End:   time.Now(),
			},
			Value: specs.AggregateSpec{
				Quantity: "100",
				Unit:     "tokens",
			},
			Aggregation: "sum",
			RecordCount: 1,
		}

		_, err := NewMeterReading(spec)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid window")
	})

	t.Run("with invalid aggregation returns error", func(t *testing.T) {
		spec := specs.MeterReadingSpec{
			ID:          "reading-123",
			WorkspaceID: "workspace-prod",
			UniverseID:  "production",
			Subject:     "customer:acme",
			Window: specs.TimeWindowSpec{
				Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			Value: specs.AggregateSpec{
				Quantity: "100",
				Unit:     "tokens",
			},
			Aggregation: "invalid-agg",
			RecordCount: 1,
		}

		_, err := NewMeterReading(spec)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid aggregation")
	})

	t.Run("with negative record count returns error", func(t *testing.T) {
		spec := specs.MeterReadingSpec{
			ID:          "reading-123",
			WorkspaceID: "workspace-prod",
			UniverseID:  "production",
			Subject:     "customer:acme",
			Window: specs.TimeWindowSpec{
				Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			Value: specs.AggregateSpec{
				Quantity: "100",
				Unit:     "tokens",
			},
			Aggregation: "sum",
			RecordCount: -1,
		}

		_, err := NewMeterReading(spec)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "record count cannot be negative")
	})
}

func TestMeterReadingAggregation(t *testing.T) {
	t.Run("sum aggregation type checks", func(t *testing.T) {
		agg, err := NewMeterReadingAggregation("sum")
		require.NoError(t, err)

		assert.True(t, agg.IsSum())
		assert.False(t, agg.IsMax())
		assert.False(t, agg.IsTimeWeightedAvg())
		assert.False(t, agg.IsLatest())
		assert.False(t, agg.IsMin())
	})

	t.Run("validates aggregation types", func(t *testing.T) {
		validTypes := []string{"sum", "max", "time-weighted-avg", "latest", "min"}

		for _, aggType := range validTypes {
			_, err := NewMeterReadingAggregation(aggType)
			assert.NoError(t, err, "aggregation type %q should be valid", aggType)
		}
	})

	t.Run("rejects invalid aggregation type", func(t *testing.T) {
		_, err := NewMeterReadingAggregation("invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid aggregation type")
	})
}

func TestTimeWindow(t *testing.T) {
	t.Run("creates valid time window", func(t *testing.T) {
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		window, err := NewTimeWindow(specs.TimeWindowSpec{
			Start: start,
			End:   end,
		})

		require.NoError(t, err)
		assert.Equal(t, start, window.Start().ToTime())
		assert.Equal(t, end, window.End().ToTime())
	})

	t.Run("with start after end returns error", func(t *testing.T) {
		start := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		_, err := NewTimeWindow(specs.TimeWindowSpec{
			Start: start,
			End:   end,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "start must be before or equal to end")
	})

	t.Run("with equal start and end creates instant window", func(t *testing.T) {
		same := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		window, err := NewTimeWindow(specs.TimeWindowSpec{
			Start: same,
			End:   same,
		})

		require.NoError(t, err)
		assert.True(t, window.IsInstant(), "should be instant window when start == end")
		assert.Equal(t, same, window.Start().ToTime())
		assert.Equal(t, same, window.End().ToTime())
	})
}

func TestNewComputedValue(t *testing.T) {
	t.Run("creates computed value with all fields", func(t *testing.T) {
		quantity, err := NewDecimal("1250.50")
		require.NoError(t, err)

		unit, err := NewUnit("api-tokens")
		require.NoError(t, err)

		aggregation, err := NewMeterReadingAggregation("sum")
		require.NoError(t, err)

		computed := NewComputedValue(quantity, unit, aggregation)

		assert.Equal(t, "1250.50", computed.Quantity().String())
		assert.Equal(t, "api-tokens", computed.Unit().ToString())
		assert.Equal(t, "sum", computed.Aggregation().ToString())
	})

	t.Run("creates computed value with different aggregation types", func(t *testing.T) {
		quantity, _ := NewDecimal("100")
		unit, _ := NewUnit("seats")

		aggregations := []string{"sum", "max", "min", "latest", "time-weighted-avg"}

		for _, aggType := range aggregations {
			agg, err := NewMeterReadingAggregation(aggType)
			require.NoError(t, err)

			computed := NewComputedValue(quantity, unit, agg)

			assert.Equal(t, aggType, computed.Aggregation().ToString(),
				"should preserve aggregation type: %s", aggType)
		}
	})
}

func TestComputedValue_ToSpec(t *testing.T) {
	t.Run("converts to spec correctly", func(t *testing.T) {
		quantity, _ := NewDecimal("1250.50")
		unit, _ := NewUnit("api-tokens")
		aggregation, _ := NewMeterReadingAggregation("sum")

		computed := NewComputedValue(quantity, unit, aggregation)
		spec := computed.ToSpec()

		assert.Equal(t, "1250.50", spec.Quantity)
		assert.Equal(t, "api-tokens", spec.Unit)
		assert.Equal(t, "sum", spec.Aggregation)
	})

	t.Run("converts time-weighted-avg correctly", func(t *testing.T) {
		quantity, _ := NewDecimal("42.75")
		unit, _ := NewUnit("seats")
		aggregation, _ := NewMeterReadingAggregation("time-weighted-avg")

		computed := NewComputedValue(quantity, unit, aggregation)
		spec := computed.ToSpec()

		assert.Equal(t, "42.75", spec.Quantity)
		assert.Equal(t, "seats", spec.Unit)
		assert.Equal(t, "time-weighted-avg", spec.Aggregation)
	})
}

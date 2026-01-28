package specs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInstantObservation(t *testing.T) {
	t.Run("creates observation with window start equal to end", func(t *testing.T) {
		instant := time.Date(2024, 2, 15, 9, 47, 0, 0, time.UTC)

		obs := NewInstantObservation("15", "seats", instant)

		assert.Equal(t, "15", obs.Quantity)
		assert.Equal(t, "seats", obs.Unit)
		assert.Equal(t, instant, obs.Window.Start)
		assert.Equal(t, instant, obs.Window.End)
		assert.True(t, obs.Window.Start.Equal(obs.Window.End), "instant observation must have Start == End")
	})

	t.Run("creates observation at different instant", func(t *testing.T) {
		instant := time.Date(2024, 1, 31, 20, 0, 0, 0, time.UTC)

		obs := NewInstantObservation("500", "tokens", instant)

		assert.Equal(t, "500", obs.Quantity)
		assert.Equal(t, "tokens", obs.Unit)
		assert.Equal(t, instant, obs.Window.Start)
		assert.Equal(t, instant, obs.Window.End)
	})
}

func TestNewSpanObservation(t *testing.T) {
	t.Run("creates observation with window start before end", func(t *testing.T) {
		start := time.Date(2024, 1, 31, 20, 0, 0, 0, time.UTC)
		end := time.Date(2024, 2, 1, 4, 0, 0, 0, time.UTC)

		obs, err := NewSpanObservation("8", "compute-hours", start, end)

		require.NoError(t, err)
		assert.Equal(t, "8", obs.Quantity)
		assert.Equal(t, "compute-hours", obs.Unit)
		assert.Equal(t, start, obs.Window.Start)
		assert.Equal(t, end, obs.Window.End)
		assert.True(t, obs.Window.End.After(obs.Window.Start), "span observation must have End > Start")
	})

	t.Run("with end equal to start returns error", func(t *testing.T) {
		same := time.Date(2024, 2, 15, 10, 0, 0, 0, time.UTC)

		_, err := NewSpanObservation("1", "active-users", same, same)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "end must be after start")
	})

	t.Run("with end before start returns error", func(t *testing.T) {
		start := time.Date(2024, 2, 15, 10, 0, 0, 0, time.UTC)
		end := time.Date(2024, 2, 15, 9, 0, 0, 0, time.UTC)

		_, err := NewSpanObservation("1", "active-users", start, end)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "end must be after start")
	})
}

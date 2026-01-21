package infra

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Example event implementations
type TestMeterRecordedEvent struct {
	MeterID string
	Value   float64
}

func (e TestMeterRecordedEvent) EventType() EventType {
	return MeterRecorded
}

type TestMeterReadEvent struct {
	MeterID   string
	Timestamp int64
	Value     float64
}

func (e TestMeterReadEvent) EventType() EventType {
	return MeterRead
}

func TestEventTypeEnum(t *testing.T) {
	t.Run("EventType.String() returns correct values", func(t *testing.T) {
		// Arrange & Act & Assert
		assert.Equal(t, "MeterRecorded", MeterRecorded.String())
		assert.Equal(t, "MeterRead", MeterRead.String())
		assert.Equal(t, "Unknown", EventType(999).String())
	})
}

func TestBusWithEnumEventTypes(t *testing.T) {
	t.Run("can subscribe to and publish events using enum types", func(t *testing.T) {
		// Arrange
		bus := NewBus()
		var receivedEvents []Event

		handler := func(e Event) {
			receivedEvents = append(receivedEvents, e)
		}

		bus.Subscribe(MeterRecorded, handler)
		bus.Subscribe(MeterRead, handler)

		recordedEvent := TestMeterRecordedEvent{MeterID: "meter-123", Value: 42.5}
		readEvent := TestMeterReadEvent{MeterID: "meter-123", Timestamp: 1234567890, Value: 42.5}

		// Act
		bus.Publish(recordedEvent)
		bus.Publish(readEvent)

		// Assert
		assert.Len(t, receivedEvents, 2)
		assert.Equal(t, MeterRecorded, receivedEvents[0].EventType())
		assert.Equal(t, MeterRead, receivedEvents[1].EventType())
	})

	t.Run("handlers only receive events they subscribed to", func(t *testing.T) {
		// Arrange
		bus := NewBus()
		var recordedEvents []Event
		var readEvents []Event

		recordedHandler := func(e Event) {
			recordedEvents = append(recordedEvents, e)
		}

		readHandler := func(e Event) {
			readEvents = append(readEvents, e)
		}

		bus.Subscribe(MeterRecorded, recordedHandler)
		bus.Subscribe(MeterRead, readHandler)

		recordedEvent := TestMeterRecordedEvent{MeterID: "meter-123", Value: 42.5}
		readEvent := TestMeterReadEvent{MeterID: "meter-123", Timestamp: 1234567890, Value: 42.5}

		// Act
		bus.Publish(recordedEvent)
		bus.Publish(readEvent)

		// Assert
		assert.Len(t, recordedEvents, 1)
		assert.Len(t, readEvents, 1)
		assert.Equal(t, MeterRecorded, recordedEvents[0].EventType())
		assert.Equal(t, MeterRead, readEvents[0].EventType())
	})
}

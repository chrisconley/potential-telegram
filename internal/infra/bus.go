package infra

// EventType represents the type of event in the system
type EventType int

const (
	EventPayloadPublished EventType = iota
	MeterRecorded
	InFlightMeterRecorded
	MeterRead
	PostFlightMeterRead
	InFlightMeterRead
)

// String returns the string representation of the EventType
func (et EventType) String() string {
	switch et {
	case EventPayloadPublished:
		return "EventPayloadPublished"
	case MeterRecorded:
		return "MeterRecorded"
	case InFlightMeterRecorded:
		return "InFlightMeterRecorded"
	case MeterRead:
		return "MeterRead"
	case InFlightMeterRead:
		return "InFlightMeterRead"
	case PostFlightMeterRead:
		return "PostFlightMeterRead"
	default:
		return "Unknown"
	}
}

type Event interface{ EventType() EventType }
type Handler func(Event)
type Bus struct{ subs map[EventType][]Handler }

func NewBus() *Bus { return &Bus{subs: map[EventType][]Handler{}} }
func (b *Bus) Publish(e Event) {
	for _, h := range b.subs[e.EventType()] {
		h(e)
	}
}
func (b *Bus) Subscribe(evt EventType, h Handler) { b.subs[evt] = append(b.subs[evt], h) }

package internal

import (
	"fmt"
	specs "metering-spec/specs"
	"time"
)

type EventPayload struct {
	ID          EventPayloadID
	WorkspaceID EventPayloadWorkspaceID
	UniverseID  EventPayloadUniverseID
	Type        EventPayloadType
	Subject     EventPayloadSubject
	Time        EventPayloadTime
	Properties  EventPayloadProperties
}

func NewEventPayload(spec specs.EventPayloadSpec) (EventPayload, error) {
	ID, err := NewEventPayloadID(spec.ID)
	if err != nil {
		return EventPayload{}, fmt.Errorf("invalid ID: %w", err)
	}

	workspaceID, err := NewEventPayloadWorkspaceID(spec.WorkspaceID)
	if err != nil {
		return EventPayload{}, fmt.Errorf("invalid workspace ID: %w", err)
	}

	universeID, err := NewEventPayloadUniverseID(spec.UniverseID)
	if err != nil {
		return EventPayload{}, fmt.Errorf("invalid universe ID: %w", err)
	}

	eventType, err := NewEventPayloadType(spec.Type)
	if err != nil {
		return EventPayload{}, fmt.Errorf("invalid event type: %w", err)
	}

	subject, err := NewEventPayloadSubject(spec.Subject)
	if err != nil {
		return EventPayload{}, fmt.Errorf("invalid subject: %w", err)
	}

	time, err := NewEventPayloadTime(spec.Time)
	if err != nil {
		return EventPayload{}, fmt.Errorf("invalid time: %w", err)
	}

	properties := NewEventPayloadProperties(spec.Properties)

	return EventPayload{
		ID:          ID,
		WorkspaceID: workspaceID,
		UniverseID:  universeID,
		Type:        eventType,
		Subject:     subject,
		Time:        time,
		Properties:  properties,
	}, nil
}

type EventPayloadID struct {
	value string
}

func NewEventPayloadID(value string) (EventPayloadID, error) {
	if value == "" {
		return EventPayloadID{}, fmt.Errorf("ID is required")
	}
	return EventPayloadID{value: value}, nil
}

func (id EventPayloadID) ToString() string {
	return id.value
}

type EventPayloadWorkspaceID struct {
	value string
}

func NewEventPayloadWorkspaceID(value string) (EventPayloadWorkspaceID, error) {
	if value == "" {
		return EventPayloadWorkspaceID{}, fmt.Errorf("workspace ID is required")
	}
	return EventPayloadWorkspaceID{value: value}, nil
}

func (id EventPayloadWorkspaceID) ToString() string {
	return id.value
}

type EventPayloadUniverseID struct {
	value string
}

func NewEventPayloadUniverseID(value string) (EventPayloadUniverseID, error) {
	if value == "" {
		return EventPayloadUniverseID{}, fmt.Errorf("universe ID is required")
	}
	return EventPayloadUniverseID{value: value}, nil
}

func (u EventPayloadUniverseID) ToString() string {
	return u.value
}

type EventPayloadType struct {
	value string
}

func NewEventPayloadType(value string) (EventPayloadType, error) {
	if value == "" {
		return EventPayloadType{}, fmt.Errorf("event type is required")
	}
	return EventPayloadType{value: value}, nil
}

func (et EventPayloadType) ToString() string {
	return et.value
}

type EventPayloadSubject struct {
	value string
}

func NewEventPayloadSubject(value string) (EventPayloadSubject, error) {
	if value == "" {
		return EventPayloadSubject{}, fmt.Errorf("subject is required")
	}
	return EventPayloadSubject{value: value}, nil
}

func (p EventPayloadSubject) ToString() string {
	return p.value
}

type EventPayloadTime struct {
	value time.Time
}

func NewEventPayloadTime(value time.Time) (EventPayloadTime, error) {
	if value.IsZero() {
		return EventPayloadTime{}, fmt.Errorf("time is required")
	}
	return EventPayloadTime{value: value}, nil
}

func (t EventPayloadTime) ToTime() time.Time {
	return t.value
}

type EventPayloadProperties struct {
	values map[string]string
}

func NewEventPayloadProperties(values map[string]string) EventPayloadProperties {
	if values == nil {
		values = make(map[string]string)
	}
	return EventPayloadProperties{values: values}
}

func (p EventPayloadProperties) Get(key string) (string, bool) {
	val, ok := p.values[key]
	return val, ok
}

func (p *EventPayloadProperties) Set(key string, value string) {
	p.values[key] = value
}

func (p EventPayloadProperties) Has(key string) bool {
	_, ok := p.values[key]
	return ok
}

func (p EventPayloadProperties) Keys() []string {
	keys := make([]string, 0, len(p.values))
	for key := range p.values {
		keys = append(keys, key)
	}
	return keys
}

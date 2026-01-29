package internal

import (
	"fmt"
	specs "metering-spec/specs"
	"time"
)

type MeterRecord struct {
	ID            MeterRecordID
	WorkspaceID   MeterRecordWorkspaceID
	UniverseID    MeterRecordUniverseID
	Subject       MeterRecordSubject
	ObservedAt    MeterRecordObservedAt
	Observations  []Observation
	Dimensions    MeterRecordDimensions
	SourceEventID MeterRecordSourceEventID
	MeteredAt     MeterRecordMeteredAt
}

func NewMeterRecord(spec specs.MeterRecordSpec) (MeterRecord, error) {
	id, err := NewMeterRecordID(spec.ID)
	if err != nil {
		return MeterRecord{}, fmt.Errorf("invalid ID: %w", err)
	}

	workspaceID, err := NewMeterRecordWorkspaceID(spec.WorkspaceID)
	if err != nil {
		return MeterRecord{}, fmt.Errorf("invalid workspace ID: %w", err)
	}

	universeID, err := NewMeterRecordUniverseID(spec.UniverseID)
	if err != nil {
		return MeterRecord{}, fmt.Errorf("invalid universe ID: %w", err)
	}

	subject, err := NewMeterRecordSubject(spec.Subject)
	if err != nil {
		return MeterRecord{}, fmt.Errorf("invalid subject: %w", err)
	}

	// Build observations from spec.Observations array
	if len(spec.Observations) == 0 {
		return MeterRecord{}, fmt.Errorf("observations array is empty")
	}

	observations := make([]Observation, len(spec.Observations))
	for i, obsSpec := range spec.Observations {
		quantity, err := NewDecimal(obsSpec.Quantity)
		if err != nil {
			return MeterRecord{}, fmt.Errorf("invalid observation[%d] quantity: %w", i, err)
		}

		unit, err := NewUnit(obsSpec.Unit)
		if err != nil {
			return MeterRecord{}, fmt.Errorf("invalid observation[%d] unit: %w", i, err)
		}

		window, err := TimeWindowFromSpec(obsSpec.Window)
		if err != nil {
			return MeterRecord{}, fmt.Errorf("invalid observation[%d] window: %w", i, err)
		}

		observations[i] = NewObservation(quantity, unit, window)
	}

	observedAt, err := NewMeterRecordObservedAt(spec.ObservedAt)
	if err != nil {
		return MeterRecord{}, fmt.Errorf("invalid observed at: %w", err)
	}

	dimensions := NewMeterRecordDimensions()
	for name, value := range spec.Dimensions {
		dimensions.Set(name, value)
	}

	sourceEventID, err := NewMeterRecordSourceEventID(spec.SourceEventID)
	if err != nil {
		return MeterRecord{}, fmt.Errorf("invalid source event ID: %w", err)
	}

	meteredAt, err := NewMeterRecordMeteredAt(spec.MeteredAt)
	if err != nil {
		return MeterRecord{}, fmt.Errorf("invalid metered at: %w", err)
	}

	return MeterRecord{
		ID:            id,
		WorkspaceID:   workspaceID,
		UniverseID:    universeID,
		Subject:       subject,
		ObservedAt:    observedAt,
		Observations:  observations,
		Dimensions:    dimensions,
		SourceEventID: sourceEventID,
		MeteredAt:     meteredAt,
	}, nil
}

type MeterRecordID struct {
	value string
}

func NewMeterRecordID(value string) (MeterRecordID, error) {
	if value == "" {
		return MeterRecordID{}, fmt.Errorf("ID is required")
	}
	return MeterRecordID{value: value}, nil
}

func (id MeterRecordID) ToString() string {
	return id.value
}

type MeterRecordSubject struct {
	value string
}

func NewMeterRecordSubject(value string) (MeterRecordSubject, error) {
	if value == "" {
		return MeterRecordSubject{}, fmt.Errorf("subject is required")
	}
	return MeterRecordSubject{value: value}, nil
}

func (s MeterRecordSubject) ToString() string {
	return s.value
}

type MeterRecordRecordedAt struct {
	value time.Time
}

func NewMeterRecordRecordedAt(value time.Time) (MeterRecordRecordedAt, error) {
	if value.IsZero() {
		return MeterRecordRecordedAt{}, fmt.Errorf("recorded at is required")
	}
	return MeterRecordRecordedAt{value: value}, nil
}

func (t MeterRecordRecordedAt) ToTime() time.Time {
	return t.value
}

type MeterRecordObservedAt struct {
	value time.Time
}

func NewMeterRecordObservedAt(value time.Time) (MeterRecordObservedAt, error) {
	if value.IsZero() {
		return MeterRecordObservedAt{}, fmt.Errorf("observed at is required")
	}
	return MeterRecordObservedAt{value: value}, nil
}

func (t MeterRecordObservedAt) ToTime() time.Time {
	return t.value
}

type MeterRecordDimensions struct {
	values map[string]string
}

func NewMeterRecordDimensions() MeterRecordDimensions {
	return MeterRecordDimensions{
		values: make(map[string]string),
	}
}

func (d *MeterRecordDimensions) Set(name string, value string) {
	d.values[name] = value
}

func (d MeterRecordDimensions) Get(name string) (string, bool) {
	val, ok := d.values[name]
	return val, ok
}

func (d MeterRecordDimensions) Has(name string) bool {
	_, ok := d.values[name]
	return ok
}

func (d MeterRecordDimensions) Names() []string {
	names := make([]string, 0, len(d.values))
	for name := range d.values {
		names = append(names, name)
	}
	return names
}

type MeterRecordSourceEventID struct {
	value string
}

func NewMeterRecordSourceEventID(value string) (MeterRecordSourceEventID, error) {
	if value == "" {
		return MeterRecordSourceEventID{}, fmt.Errorf("source event ID is required")
	}
	return MeterRecordSourceEventID{value: value}, nil
}

func (id MeterRecordSourceEventID) ToString() string {
	return id.value
}

type MeterRecordWorkspaceID struct {
	value string
}

func NewMeterRecordWorkspaceID(value string) (MeterRecordWorkspaceID, error) {
	if value == "" {
		return MeterRecordWorkspaceID{}, fmt.Errorf("workspace ID is required")
	}
	return MeterRecordWorkspaceID{value: value}, nil
}

func (id MeterRecordWorkspaceID) ToString() string {
	return id.value
}

type MeterRecordUniverseID struct {
	value string
}

func NewMeterRecordUniverseID(value string) (MeterRecordUniverseID, error) {
	if value == "" {
		return MeterRecordUniverseID{}, fmt.Errorf("universe ID is required")
	}
	return MeterRecordUniverseID{value: value}, nil
}

func (u MeterRecordUniverseID) ToString() string {
	return u.value
}

type Unit struct {
	value string
}

func NewUnit(value string) (Unit, error) {
	if value == "" {
		return Unit{}, fmt.Errorf("unit is required")
	}
	return Unit{value: value}, nil
}

func (u Unit) ToString() string {
	return u.value
}

// Observation represents a single observation from an event with temporal context.
// Observations are raw measurements extracted from event payloads.
type Observation struct {
	quantity Decimal
	unit     Unit
	window   TimeWindow
}

func NewObservation(quantity Decimal, unit Unit, window TimeWindow) Observation {
	return Observation{
		quantity: quantity,
		unit:     unit,
		window:   window,
	}
}

func (o Observation) Quantity() Decimal {
	return o.quantity
}

func (o Observation) Unit() Unit {
	return o.unit
}

func (o Observation) Window() TimeWindow {
	return o.window
}

type MeterRecordMeteredAt struct {
	value time.Time
}

func NewMeterRecordMeteredAt(value time.Time) (MeterRecordMeteredAt, error) {
	if value.IsZero() {
		value = time.Now()
	}
	return MeterRecordMeteredAt{value: value}, nil
}

func (m MeterRecordMeteredAt) ToTime() time.Time {
	return m.value
}

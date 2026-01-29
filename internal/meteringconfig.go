package internal

import (
	"fmt"
	specs "metering-spec/specs"
)

type MeteringConfig struct {
	measurements []MeasurementExtraction
}

func NewMeteringConfig(spec specs.MeteringConfigSpec) (MeteringConfig, error) {
	if len(spec.Measurements) == 0 {
		return MeteringConfig{}, fmt.Errorf("at least one measurement extraction is required")
	}

	measurements := make([]MeasurementExtraction, 0, len(spec.Measurements))
	for i, m := range spec.Measurements {
		extraction, err := NewMeasurementExtraction(m)
		if err != nil {
			return MeteringConfig{}, fmt.Errorf("measurement %d: %w", i, err)
		}
		measurements = append(measurements, extraction)
	}

	return MeteringConfig{
		measurements: measurements,
	}, nil
}

func (c MeteringConfig) Measurements() []MeasurementExtraction {
	return c.measurements
}

type MeasurementExtraction struct {
	sourceProperty MeasurementSourceProperty
	unit           Unit
	filter         *Filter
}

func NewMeasurementExtraction(spec specs.MeasurementExtractionSpec) (MeasurementExtraction, error) {
	sourceProperty, err := NewMeasurementSourceProperty(spec.SourceProperty)
	if err != nil {
		return MeasurementExtraction{}, fmt.Errorf("invalid source property: %w", err)
	}

	unit, err := NewUnit(spec.Unit)
	if err != nil {
		return MeasurementExtraction{}, fmt.Errorf("invalid unit: %w", err)
	}

	var filter *Filter
	if spec.Filter != nil {
		f, err := NewFilter(*spec.Filter)
		if err != nil {
			return MeasurementExtraction{}, fmt.Errorf("invalid filter: %w", err)
		}
		filter = &f
	}

	return MeasurementExtraction{
		sourceProperty: sourceProperty,
		unit:           unit,
		filter:         filter,
	}, nil
}

func (m MeasurementExtraction) SourceProperty() MeasurementSourceProperty {
	return m.sourceProperty
}

func (m MeasurementExtraction) Unit() Unit {
	return m.unit
}

func (m MeasurementExtraction) Filter() *Filter {
	return m.filter
}

// Matches returns true if the filter matches the payload properties (or if no filter exists).
func (m MeasurementExtraction) Matches(properties EventPayloadProperties) bool {
	if m.filter == nil {
		return true
	}
	return m.filter.Matches(properties)
}

type MeasurementSourceProperty struct {
	value string
}

func NewMeasurementSourceProperty(value string) (MeasurementSourceProperty, error) {
	if value == "" {
		return MeasurementSourceProperty{}, fmt.Errorf("source property is required")
	}
	return MeasurementSourceProperty{value: value}, nil
}

func (p MeasurementSourceProperty) ToString() string {
	return p.value
}

type Filter struct {
	property FilterProperty
	equals   FilterValue
}

func NewFilter(spec specs.FilterSpec) (Filter, error) {
	property, err := NewFilterProperty(spec.Property)
	if err != nil {
		return Filter{}, fmt.Errorf("invalid property: %w", err)
	}

	equals, err := NewFilterValue(spec.Equals)
	if err != nil {
		return Filter{}, fmt.Errorf("invalid equals: %w", err)
	}

	return Filter{
		property: property,
		equals:   equals,
	}, nil
}

func (f Filter) Property() FilterProperty {
	return f.property
}

func (f Filter) Equals() FilterValue {
	return f.equals
}

// Matches returns true if the filter condition is satisfied by the properties.
func (f Filter) Matches(properties EventPayloadProperties) bool {
	value, exists := properties.Get(f.property.ToString())
	if !exists {
		return false
	}
	return value == f.equals.ToString()
}

type FilterProperty struct {
	value string
}

func NewFilterProperty(value string) (FilterProperty, error) {
	if value == "" {
		return FilterProperty{}, fmt.Errorf("filter property is required")
	}
	return FilterProperty{value: value}, nil
}

func (p FilterProperty) ToString() string {
	return p.value
}

type FilterValue struct {
	value string
}

func NewFilterValue(value string) (FilterValue, error) {
	if value == "" {
		return FilterValue{}, fmt.Errorf("filter value is required")
	}
	return FilterValue{value: value}, nil
}

func (v FilterValue) ToString() string {
	return v.value
}

// ObservationExtraction defines how to extract an observation from an event.
// This is the new naming aligned with domain terminology (Observation for raw extracted values).
type ObservationExtraction struct {
	sourceProperty ObservationSourceProperty
	unit           Unit
	filter         *Filter
}

func NewObservationExtraction(spec specs.ObservationExtractionSpec) (ObservationExtraction, error) {
	sourceProperty, err := NewObservationSourceProperty(spec.SourceProperty)
	if err != nil {
		return ObservationExtraction{}, fmt.Errorf("invalid source property: %w", err)
	}

	unit, err := NewUnit(spec.Unit)
	if err != nil {
		return ObservationExtraction{}, fmt.Errorf("invalid unit: %w", err)
	}

	var filter *Filter
	if spec.Filter != nil {
		f, err := NewFilter(*spec.Filter)
		if err != nil {
			return ObservationExtraction{}, fmt.Errorf("invalid filter: %w", err)
		}
		filter = &f
	}

	return ObservationExtraction{
		sourceProperty: sourceProperty,
		unit:           unit,
		filter:         filter,
	}, nil
}

func (o ObservationExtraction) SourceProperty() ObservationSourceProperty {
	return o.sourceProperty
}

func (o ObservationExtraction) Unit() Unit {
	return o.unit
}

func (o ObservationExtraction) Filter() *Filter {
	return o.filter
}

// Matches returns true if the filter matches the payload properties (or if no filter exists).
func (o ObservationExtraction) Matches(properties EventPayloadProperties) bool {
	if o.filter == nil {
		return true
	}
	return o.filter.Matches(properties)
}

// ObservationSourceProperty identifies which property to extract from an event.
type ObservationSourceProperty struct {
	value string
}

func NewObservationSourceProperty(value string) (ObservationSourceProperty, error) {
	if value == "" {
		return ObservationSourceProperty{}, fmt.Errorf("source property is required")
	}
	return ObservationSourceProperty{value: value}, nil
}

func (p ObservationSourceProperty) ToString() string {
	return p.value
}

package internal

import (
	"fmt"
	specs "metering-spec/specs"
)

type MeteringConfig struct {
	observations []ObservationExtraction
}

func NewMeteringConfig(spec specs.MeteringConfigSpec) (MeteringConfig, error) {
	if len(spec.Observations) == 0 {
		return MeteringConfig{}, fmt.Errorf("at least one observation extraction is required")
	}

	observations := make([]ObservationExtraction, 0, len(spec.Observations))
	for i, o := range spec.Observations {
		extraction, err := NewObservationExtraction(o)
		if err != nil {
			return MeteringConfig{}, fmt.Errorf("observation %d: %w", i, err)
		}
		observations = append(observations, extraction)
	}

	return MeteringConfig{
		observations: observations,
	}, nil
}

func (c MeteringConfig) Observations() []ObservationExtraction {
	return c.observations
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

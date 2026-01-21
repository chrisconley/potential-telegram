package internal

import (
	"fmt"
	specs "metering-spec/specs"
)

type AggregationConfig struct {
	aggregation MeterReadingAggregation
	window      TimeWindow
}

func NewAggregationConfig(spec specs.AggregateConfigSpec) (AggregationConfig, error) {
	aggregation, err := NewMeterReadingAggregation(spec.Aggregation)
	if err != nil {
		return AggregationConfig{}, fmt.Errorf("invalid aggregation: %w", err)
	}

	window, err := NewTimeWindow(spec.Window)
	if err != nil {
		return AggregationConfig{}, fmt.Errorf("invalid window: %w", err)
	}

	return AggregationConfig{
		aggregation: aggregation,
		window:      window,
	}, nil
}

func (c AggregationConfig) Aggregation() MeterReadingAggregation {
	return c.aggregation
}

func (c AggregationConfig) Window() TimeWindow {
	return c.window
}

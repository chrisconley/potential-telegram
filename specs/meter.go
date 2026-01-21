package specs

// Meter transforms an EventPayload into MeterRecords by applying the metering configuration.
//
// For each measurement extraction in the config:
//  1. Check if filter matches (if filter exists)
//  2. Extract the source property value (string)
//  3. Cast string to Decimal
//  4. Attach the configured unit to create Measurement
//  5. Pass through all non-extracted properties as dimensions
//  6. Create a MeterRecord
//
// Returns a slice of MeterRecordSpecs (one per matched extraction).
// Returns empty slice if no extractions match (not an error).
//
// This is the spec-level interface using only primitive types.
// See internal.Meter for the reference implementation.
type Meter func(payload EventPayloadSpec, config MeteringConfigSpec) ([]MeterRecordSpec, error)

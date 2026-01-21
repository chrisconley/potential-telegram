package examples

import (
	"fmt"
	"metering-spec/internal"
	"metering-spec/internal/infra"
	"metering-spec/specs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// === EVENT WRAPPER TYPES ===

type EventPayloadEvent struct {
	Payload specs.EventPayloadSpec
}

func (e EventPayloadEvent) EventType() infra.EventType {
	return infra.EventPayloadPublished
}

type InFlightMeterRecordedEvent struct {
	Record specs.MeterRecordSpec
}

func (e InFlightMeterRecordedEvent) EventType() infra.EventType {
	return infra.InFlightMeterRecorded
}

type InFlightMeterReadEvent struct {
	Reading specs.MeterReadingSpec
}

func (e InFlightMeterReadEvent) EventType() infra.EventType {
	return infra.InFlightMeterRead
}

type PostFlightMeterReadEvent struct {
	Reading specs.MeterReadingSpec
}

func (e PostFlightMeterReadEvent) EventType() infra.EventType {
	return infra.PostFlightMeterRead
}

// === CONFIG REPO ===

type RatingConfig struct {
	PricePerRequest string // Decimal as string
	Threshold       string // Revenue threshold to trigger alert
}

type ConfigRepo interface {
	GetMeteringConfig() specs.MeteringConfigSpec
	GetAggregationConfig(windowDuration time.Duration) specs.AggregateConfigSpec
	GetRatingConfig() RatingConfig
}

type HardcodedConfigRepo struct{}

func (r *HardcodedConfigRepo) GetMeteringConfig() specs.MeteringConfigSpec {
	return specs.MeteringConfigSpec{
		Measurements: []specs.MeasurementExtractionSpec{
			{SourceProperty: "request_count", Unit: "requests"},
		},
	}
}

func (r *HardcodedConfigRepo) GetAggregationConfig(windowDuration time.Duration) specs.AggregateConfigSpec {
	return specs.AggregateConfigSpec{
		Aggregation: "sum",
		Window: specs.TimeWindowSpec{
			Start: time.Time{}, // Will be set when aggregating
			End:   time.Time{},
		},
	}
}

func (r *HardcodedConfigRepo) GetRatingConfig() RatingConfig {
	return RatingConfig{
		PricePerRequest: "0.001", // $0.001 per request
		Threshold:       "0.020", // Alert at $0.02
	}
}

// === HANDLERS ===

type MeteringHandler struct {
	bus        *infra.Bus
	configRepo ConfigRepo
}

func (h *MeteringHandler) Handle(e infra.Event) {
	payload := e.(EventPayloadEvent).Payload
	config := h.configRepo.GetMeteringConfig()

	records, err := internal.Meter(payload, config)
	if err != nil {
		panic(fmt.Sprintf("Failed to meter payload: %v", err))
	}

	for _, record := range records {
		h.bus.Publish(InFlightMeterRecordedEvent{Record: record})
	}
}

type InFlightAggregator struct {
	bus         *infra.Bus
	configRepo  ConfigRepo
	currentTick time.Time
	batch       []specs.MeterRecordSpec
}

func (h *InFlightAggregator) Handle(e infra.Event) {
	record := e.(InFlightMeterRecordedEvent).Record

	// Determine which tick (1-second window) this record belongs to
	recordTick := record.RecordedAt.Truncate(time.Second)

	// Detect tick change - we've moved to a new time window
	if !h.currentTick.IsZero() && recordTick.After(h.currentTick) {
		// Flush the batch from the previous tick
		h.flushBatch()
	}

	// Initialize current tick on first record
	if h.currentTick.IsZero() {
		h.currentTick = recordTick
	}

	// Batch this record with others in the current tick
	h.batch = append(h.batch, record)
}

func (h *InFlightAggregator) flushBatch() {
	if len(h.batch) == 0 {
		return
	}

	// Get aggregation config for 1-second windows
	config := h.configRepo.GetAggregationConfig(time.Second)

	// Set the window to match the current tick
	config.Window = specs.TimeWindowSpec{
		Start: h.currentTick,
		End:   h.currentTick.Add(time.Second),
	}

	// Aggregate all batched records into a single reading
	reading, err := internal.Aggregate(h.batch, nil, config)
	if err != nil {
		panic(fmt.Sprintf("Failed to aggregate batch: %v", err))
	}

	// Publish aggregated reading for downstream consumers
	h.bus.Publish(InFlightMeterReadEvent{Reading: reading})

	// Reset for next tick
	h.batch = nil
	h.currentTick = time.Time{}
}

func (h *InFlightAggregator) Flush() {
	h.flushBatch()
}

type RatingHandler struct {
	configRepo         ConfigRepo
	accumulatedRevenue internal.Decimal
	threshold          internal.Decimal
	thresholdReached   bool
}

func NewRatingHandler(configRepo ConfigRepo) *RatingHandler {
	config := configRepo.GetRatingConfig()
	threshold, err := internal.NewDecimal(config.Threshold)
	if err != nil {
		panic(fmt.Sprintf("Invalid threshold: %v", err))
	}

	return &RatingHandler{
		configRepo:         configRepo,
		accumulatedRevenue: internal.NewDecimalFromInt64(0),
		threshold:          threshold,
		thresholdReached:   false,
	}
}

func (h *RatingHandler) Handle(e infra.Event) {
	reading := e.(InFlightMeterReadEvent).Reading
	config := h.configRepo.GetRatingConfig()

	// Extract quantity from reading
	quantity, err := internal.NewDecimal(reading.Measurement.Quantity)
	if err != nil {
		panic(fmt.Sprintf("Invalid quantity: %v", err))
	}

	// Compute revenue for this reading
	pricePerRequest, err := internal.NewDecimal(config.PricePerRequest)
	if err != nil {
		panic(fmt.Sprintf("Invalid price: %v", err))
	}

	revenue := quantity.Mul(pricePerRequest)
	h.accumulatedRevenue = h.accumulatedRevenue.Add(revenue)

	// Check if threshold reached
	if !h.thresholdReached && h.accumulatedRevenue.Cmp(h.threshold) >= 0 {
		h.thresholdReached = true
		fmt.Printf("💰 Threshold reached: accumulated revenue = $%s (threshold: $%s)\n",
			h.accumulatedRevenue.String(),
			h.threshold.String())
	}
}

type PostFlightAggregator struct {
	bus         *infra.Bus
	configRepo  ConfigRepo
	currentTick time.Time
	batch       []specs.MeterRecordSpec
}

func (h *PostFlightAggregator) Handle(e infra.Event) {
	record := e.(InFlightMeterRecordedEvent).Record

	// Determine which tick (10-second window) this record belongs to
	recordTick := record.RecordedAt.Truncate(10 * time.Second)

	// Detect tick change - we've moved to a new time window
	if !h.currentTick.IsZero() && recordTick.After(h.currentTick) {
		// Flush the batch from the previous tick
		h.flushBatch()
	}

	// Initialize current tick on first record
	if h.currentTick.IsZero() {
		h.currentTick = recordTick
	}

	// Batch this record with others in the current tick
	h.batch = append(h.batch, record)
}

func (h *PostFlightAggregator) flushBatch() {
	if len(h.batch) == 0 {
		return
	}

	// Get aggregation config for 10-second windows
	config := h.configRepo.GetAggregationConfig(10 * time.Second)

	// Set the window to match the current tick
	config.Window = specs.TimeWindowSpec{
		Start: h.currentTick,
		End:   h.currentTick.Add(10 * time.Second),
	}

	// Aggregate all batched records into a single reading
	reading, err := internal.Aggregate(h.batch, nil, config)
	if err != nil {
		panic(fmt.Sprintf("Failed to aggregate batch: %v", err))
	}

	// Publish aggregated reading for downstream consumers
	h.bus.Publish(PostFlightMeterReadEvent{Reading: reading})

	// Reset for next tick
	h.batch = nil
	h.currentTick = time.Time{}
}

func (h *PostFlightAggregator) Flush() {
	h.flushBatch()
}

type CustomerBalanceHandler struct{}

func (h *CustomerBalanceHandler) Handle(e infra.Event) {
	reading := e.(PostFlightMeterReadEvent).Reading
	fmt.Printf("📊 Customer balance update: %s %s for window %s to %s\n",
		reading.Measurement.Quantity,
		reading.Measurement.Unit,
		reading.Window.Start.Format("15:04:05"),
		reading.Window.End.Format("15:04:05"))
}

func TestHighThroughputMeteringPipeline(t *testing.T) {
	t.Log("Testing high-throughput metering pipeline with in-flight and post-flight processing")

	// Setup bus and config repo
	bus := infra.NewBus()
	configRepo := &HardcodedConfigRepo{}

	// === STEP 1: Wire up MeteringHandler ===
	// Receives EventPayloads, transforms to MeterRecords
	meteringHandler := &MeteringHandler{
		bus:        bus,
		configRepo: configRepo,
	}
	bus.Subscribe(infra.EventPayloadPublished, meteringHandler.Handle)

	// Track published MeterRecords for verification
	var publishedRecords []specs.MeterRecordSpec
	bus.Subscribe(infra.InFlightMeterRecorded, func(e infra.Event) {
		record := e.(InFlightMeterRecordedEvent).Record
		publishedRecords = append(publishedRecords, record)
	})

	// === STEP 2: Wire up InFlightAggregator ===
	// Receives MeterRecords, batches by 1-second windows, publishes aggregated readings
	inFlightAgg := &InFlightAggregator{
		bus:        bus,
		configRepo: configRepo,
	}
	bus.Subscribe(infra.InFlightMeterRecorded, inFlightAgg.Handle)

	// Track published 1-second readings
	var publishedReadings []specs.MeterReadingSpec
	bus.Subscribe(infra.InFlightMeterRead, func(e infra.Event) {
		reading := e.(InFlightMeterReadEvent).Reading
		publishedReadings = append(publishedReadings, reading)
	})

	// === STEP 3: Wire up RatingHandler ===
	// Consumes 1-second readings, computes pricing, prints when threshold reached
	ratingHandler := NewRatingHandler(configRepo)
	bus.Subscribe(infra.InFlightMeterRead, ratingHandler.Handle)

	// === STEP 4: Wire up PostFlightAggregator ===
	// Receives same MeterRecords as in-flight, batches by 10-second windows
	postFlightAgg := &PostFlightAggregator{
		bus:        bus,
		configRepo: configRepo,
	}
	bus.Subscribe(infra.InFlightMeterRecorded, postFlightAgg.Handle)

	// Track published 1-minute readings
	var postFlightReadings []specs.MeterReadingSpec
	bus.Subscribe(infra.PostFlightMeterRead, func(e infra.Event) {
		reading := e.(PostFlightMeterReadEvent).Reading
		postFlightReadings = append(postFlightReadings, reading)
	})

	// === STEP 5: Wire up CustomerBalanceHandler ===
	// Consumes 1-minute readings, prints customer balance updates
	customerBalanceHandler := &CustomerBalanceHandler{}
	bus.Subscribe(infra.PostFlightMeterRead, customerBalanceHandler.Handle)

	// === Generate and publish EventPayloads with batching ===
	fmt.Println("Publishing EventPayloads to bus (simulating high throughput)...")

	startTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	// Generate 30 seconds of events, with 10 events per second (shows batching at both levels)
	events := generateAPIRequestEventsWithBatching(startTime, 30, 10)

	for i, eventPayload := range events {
		bus.Publish(EventPayloadEvent{Payload: eventPayload})
		if (i+1)%10 == 0 {
			secondsElapsed := int(eventPayload.Time.Sub(startTime).Seconds())
			fmt.Printf("  Published %d events (second %d)\n", i+1, secondsElapsed)
		}
	}

	// Flush any partial batches (in production, a timer would handle this)
	inFlightAgg.Flush()
	postFlightAgg.Flush()

	// === Verify and summarize results ===
	fmt.Println()
	assert.Len(t, publishedRecords, 300, "should have 300 meter records published")
	assert.Len(t, publishedReadings, 30, "should have 30 1-second readings (batched from 300 records)")
	assert.Len(t, postFlightReadings, 3, "should have 3 10-second readings (batched from 300 records)")

	expectedRevenue, _ := internal.NewDecimal("0.300") // 300 requests * $0.001
	assert.Equal(t, expectedRevenue.String(), ratingHandler.accumulatedRevenue.String(),
		"should accumulate correct revenue (300 requests * $0.001)")
	assert.True(t, ratingHandler.thresholdReached, "threshold should be reached")

	// Verify each 10-second reading aggregated 100 requests
	for i, reading := range postFlightReadings {
		assert.Equal(t, "100", reading.Measurement.Quantity,
			"reading %d should aggregate 100 requests", i+1)
	}

	fmt.Printf("✓ In-flight: %d EventPayloads → %d MeterRecords → %d 1-second readings\n",
		len(events), len(publishedRecords), len(publishedReadings))
	fmt.Printf("✓ Rating: Accumulated revenue = $%s, threshold reached\n",
		ratingHandler.accumulatedRevenue.String())
	fmt.Printf("✓ Post-flight: %d MeterRecords → %d 10-second readings\n",
		len(publishedRecords), len(postFlightReadings))
}

// === HELPER FUNCTIONS ===

func generateAPIRequestEvents(startTime time.Time, count int) []specs.EventPayloadSpec {
	events := make([]specs.EventPayloadSpec, count)
	for i := 0; i < count; i++ {
		events[i] = specs.EventPayloadSpec{
			ID:          fmt.Sprintf("req-%d", i),
			WorkspaceID: "workspace-prod",
			UniverseID:  "production",
			Type:        "api.request",
			Subject:     "customer:acme-corp",
			Time:        startTime.Add(time.Duration(i) * time.Second),
			Properties: map[string]string{
				"request_count": "1",
				"endpoint":      "/api/v1/users",
				"status_code":   "200",
			},
		}
	}
	return events
}

// generateAPIRequestEventsWithBatching creates events with multiple per second
// to demonstrate batching behavior (simulating high throughput)
func generateAPIRequestEventsWithBatching(
	startTime time.Time,
	durationSeconds int,
	eventsPerSecond int,
) []specs.EventPayloadSpec {
	var events []specs.EventPayloadSpec
	eventID := 0

	for second := 0; second < durationSeconds; second++ {
		// Generate multiple events for this second (simulating high throughput)
		for event := 0; event < eventsPerSecond; event++ {
			// Add microseconds to spread events within the second
			// but still within the same 1-second window
			timestamp := startTime.Add(
				time.Duration(second)*time.Second +
					time.Duration(event)*time.Microsecond,
			)

			events = append(events, specs.EventPayloadSpec{
				ID:          fmt.Sprintf("req-%d", eventID),
				WorkspaceID: "workspace-prod",
				UniverseID:  "production",
				Type:        "api.request",
				Subject:     "customer:acme-corp",
				Time:        timestamp,
				Properties: map[string]string{
					"request_count": "1",
					"endpoint":      "/api/v1/users",
					"status_code":   "200",
				},
			})
			eventID++
		}
	}

	return events
}

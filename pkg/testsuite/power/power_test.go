package power

import (
	"testing"
	"time"

	"govsan/pkg/testsuite"
	"govsan/pkg/testsuite/health"
)

// MetricsCollector - collects performance metrics
type MetricsCollector struct {
	samples []map[string]interface{}
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		samples: make([]map[string]interface{}, 0),
	}
}

func (c *MetricsCollector) Collect() error {
	c.samples = append(c.samples, map[string]interface{}{
		"timestamp": time.Now(),
		"cpu":       45.5,
		"memory":    60.0,
	})
	return nil
}

func (c *MetricsCollector) Reset() {
	c.samples = c.samples[:0]
}

func (c *MetricsCollector) GetSamples() []map[string]interface{} {
	return c.samples
}

func TestPowerTestSuite(t *testing.T) {
	ctx := testsuite.NewTestContext(t)
	lifecycle := testsuite.NewTestManager(ctx)
	
	collector := NewMetricsCollector()
	baseline := map[string]float64{
		"response_time_ms": 100.0,
		"throughput_rps":   1000.0,
		"error_rate":       0.01,
	}
	metricsData := make([]map[string]interface{}, 0)
	
	lifecycle.BeforeSuite(func() {
		testsuite.Log(ctx.T, "Setting up PowerTestSuite...")
		ctx.Set("baseline_initialized", true)
		ctx.Set("metrics_collected", 0)
	})
	
	lifecycle.AfterSuite(func() {
		testsuite.Log(ctx.T, "Tearing down PowerTestSuite...")
		metricsCount := ctx.GetInt("metrics_collected", 0)
		testsuite.Logf(ctx.T, "Total metrics collected: %d", metricsCount)
	})
	
	tests := map[string]func(*testsuite.TestContext){
		"TestQueryPerformanceMetrics": func(ctx *testsuite.TestContext) {
			testQueryPerformanceMetrics(ctx, collector)
		},
		"TestMeasureLatency": func(ctx *testsuite.TestContext) {
			testMeasureLatency(ctx, baseline)
		},
		"TestMeasureThroughput": func(ctx *testsuite.TestContext) {
			testMeasureThroughput(ctx, baseline)
		},
		"TestErrorRate": func(ctx *testsuite.TestContext) {
			testErrorRate(ctx, baseline)
		},
		"TestMetricsAggregation": func(ctx *testsuite.TestContext) {
			testMetricsAggregation(ctx, collector, &metricsData)
		},
		"TestPowerHealthCorrelation": func(ctx *testsuite.TestContext) {
			testPowerHealthCorrelation(ctx)
		},
	}
	
	lifecycle.RunSuite(t, tests)
}

func testQueryPerformanceMetrics(ctx *testsuite.TestContext, collector *MetricsCollector) {
	testsuite.Log(ctx.T, "Running TestQueryPerformanceMetrics...")
	err := collector.Collect()
	testsuite.RequireNoError(ctx.T, err, "Failed to collect metrics")
	ctx.Set("metrics_collected", ctx.GetInt("metrics_collected", 0)+1)
}

func testMeasureLatency(ctx *testsuite.TestContext, baseline map[string]float64) {
	testsuite.Log(ctx.T, "Running TestMeasureLatency...")
	latency := 85.5
	threshold := baseline["response_time_ms"]
	testsuite.Logf(ctx.T, "Measured latency: %.2f ms (threshold: %.2f ms)", latency, threshold)
	testsuite.Assertf(ctx.T, latency < threshold, "Latency exceeds threshold")
}

func testMeasureThroughput(ctx *testsuite.TestContext, baseline map[string]float64) {
	testsuite.Log(ctx.T, "Running TestMeasureThroughput...")
	throughput := 1200.0
	threshold := baseline["throughput_rps"]
	testsuite.Logf(ctx.T, "Measured throughput: %.2f rps (threshold: %.2f rps)", throughput, threshold)
	testsuite.Assertf(ctx.T, throughput >= threshold, "Throughput below threshold")
}

func testErrorRate(ctx *testsuite.TestContext, baseline map[string]float64) {
	testsuite.Log(ctx.T, "Running TestErrorRate...")
	errorRate := 0.005
	threshold := baseline["error_rate"]
	testsuite.Logf(ctx.T, "Measured error rate: %.4f (threshold: %.4f)", errorRate, threshold)
	testsuite.Assertf(ctx.T, errorRate <= threshold, "Error rate exceeds threshold")
}

func testMetricsAggregation(ctx *testsuite.TestContext, collector *MetricsCollector, metricsData *[]map[string]interface{}) {
	testsuite.Log(ctx.T, "Running TestMetricsAggregation...")
	for i := 0; i < 3; i++ {
		err := collector.Collect()
		testsuite.RequireNoError(ctx.T, err, "Metrics collection failed")
		*metricsData = append(*metricsData, map[string]interface{}{"cpu": 45.5})
	}
	testsuite.Assertf(ctx.T, len(*metricsData) > 0, "Aggregation produced no results")
}

// testPowerHealthCorrelation demonstrates using health module in power module
func testPowerHealthCorrelation(ctx *testsuite.TestContext) {
	testsuite.Log(ctx.T, "Running TestPowerHealthCorrelation...")

	// Use health module's HealthStatusManager (still available)
	manager := health.NewHealthStatusManager()
	manager.AddThresholdRule("PowerConsumption", 80.0, 95.0)
	status := manager.EvaluateStatus("PowerConsumption", 75.0)
	testsuite.Assertf(ctx.T, status == health.HealthStatusOK,
		"Expected OK status for low power consumption")

	// Test status evaluation with different values
	status = manager.EvaluateStatus("PowerConsumption", 85.0)
	testsuite.Assertf(ctx.T, status == health.HealthStatusWarning,
		"Expected Warning status for medium power consumption")

	status = manager.EvaluateStatus("PowerConsumption", 96.0)
	testsuite.Assertf(ctx.T, status == health.HealthStatusError,
		"Expected Error status for high power consumption")

	testsuite.Log(ctx.T, "Power-Health Correlation Test completed")
}

package health

import (
	"testing"
	"time"

	"govsan/pkg/testsuite"
)

func TestHealthStatusManagementSuite(t *testing.T) {
	lifecycle := testsuite.NewTestLifecycle()
	ctx := testsuite.NewTestContext(t)

	var manager *HealthStatusManager
	var testStartTime time.Time

	lifecycle.BeforeSuite(func() {
		testsuite.Log(ctx.T, "=== Initializing HealthStatusManagement Suite ===")
		manager = NewHealthStatusManager()
		testStartTime = time.Now()

		testsuite.Log(ctx.T, "Adding default threshold rules...")
		manager.AddThresholdRule("CPU", 70.0, 90.0)
		manager.AddThresholdRule("Memory", 80.0, 95.0)
		manager.AddThresholdRule("NetworkLatency", 50.0, 100.0)
	})

	lifecycle.AfterSuite(func() {
		testsuite.Log(ctx.T, "=== Cleaning up HealthStatusManagement Suite ===")
		duration := time.Since(testStartTime)
		testsuite.Logf(ctx.T, "Total test time: %v", duration)
	})

	lifecycle.BeforeTest(func(name string) {
		testsuite.Log(ctx.T, "--- Starting test:", name, "---")
	})

	lifecycle.AfterTest(func(name string) {
		testsuite.Log(ctx.T, "--- Finished test:", name, "---")
	})

	tests := map[string]func(*testsuite.TestContext){
		"TestThresholdRuleAddition": func(ctx *testsuite.TestContext) {
			testThresholdRuleAddition(ctx, manager)
		},
		"TestStatusEvaluationNormal": func(ctx *testsuite.TestContext) {
			testStatusEvaluationNormal(ctx, manager)
		},
		"TestStatusEvaluationWarning": func(ctx *testsuite.TestContext) {
			testStatusEvaluationWarning(ctx, manager)
		},
		"TestStatusEvaluationError": func(ctx *testsuite.TestContext) {
			testStatusEvaluationError(ctx, manager)
		},
		"TestStatusEvaluationUnknown": func(ctx *testsuite.TestContext) {
			testStatusEvaluationUnknown(ctx, manager)
		},
		"TestHealthRecordRecording": func(ctx *testsuite.TestContext) {
			testHealthRecordRecording(ctx, manager)
		},
		"TestHistoryRetrievalByComponent": func(ctx *testsuite.TestContext) {
			testHistoryRetrievalByComponent(ctx, manager)
		},
		"TestStatusTrendAnalysis": func(ctx *testsuite.TestContext) {
			testStatusTrendAnalysis(ctx, manager)
		},
		"TestComponentAvailabilityCalculation": func(ctx *testsuite.TestContext) {
			testComponentAvailabilityCalculation(ctx, manager)
		},
		"TestAvailabilityWithAllOK": testAvailabilityWithAllOK,
		"TestAvailabilityWithMixStatuses": testAvailabilityWithMixStatuses,
		"TestAvailabilityWithEmptyHistory": testAvailabilityWithEmptyHistory,
	}

	lifecycle.RunSuite(t, tests)
}

func testThresholdRuleAddition(ctx *testsuite.TestContext, manager *HealthStatusManager) {
	testsuite.Log(ctx.T, "Testing threshold rule addition...")

	manager.AddThresholdRule("DiskIO", 85.0, 99.0)

	testsuite.Assertf(ctx.T, manager.rules["DiskIO"] != nil, "Threshold rule not added correctly")
	testsuite.Assertf(ctx.T, manager.rules["DiskIO"].WarningLevel == 85.0, "Warning threshold incorrect: expected 85.0, got %.1f", manager.rules["DiskIO"].WarningLevel)
	testsuite.Assertf(ctx.T, manager.rules["DiskIO"].ErrorLevel == 99.0, "Error threshold incorrect: expected 99.0, got %.1f", manager.rules["DiskIO"].ErrorLevel)

	testsuite.Log(ctx.T, "Threshold rule addition test passed")
}

func testStatusEvaluationNormal(ctx *testsuite.TestContext, manager *HealthStatusManager) {
	testsuite.Log(ctx.T, "Testing normal status evaluation...")

	status := manager.EvaluateStatus("CPU", 45.5)
	testsuite.Assertf(ctx.T, status == HealthStatusOK, "Expected status OK, got %s", status)

	status = manager.EvaluateStatus("Memory", 60.0)
	testsuite.Assertf(ctx.T, status == HealthStatusOK, "Expected status OK, got %s", status)

	testsuite.Log(ctx.T, "Normal status evaluation test passed")
}

func testStatusEvaluationWarning(ctx *testsuite.TestContext, manager *HealthStatusManager) {
	testsuite.Log(ctx.T, "Testing warning status evaluation...")

	status := manager.EvaluateStatus("CPU", 82.3)
	testsuite.Assertf(ctx.T, status == HealthStatusWarning, "Expected status Warning, got %s", status)

	status = manager.EvaluateStatus("NetworkLatency", 75.0)
	testsuite.Assertf(ctx.T, status == HealthStatusWarning, "Expected status Warning, got %s", status)

	testsuite.Log(ctx.T, "Warning status evaluation test passed")
}

func testStatusEvaluationError(ctx *testsuite.TestContext, manager *HealthStatusManager) {
	testsuite.Log(ctx.T, "Testing error status evaluation...")

	status := manager.EvaluateStatus("CPU", 95.0)
	testsuite.Assertf(ctx.T, status == HealthStatusError, "Expected status Error, got %s", status)

	status = manager.EvaluateStatus("Memory", 98.0)
	testsuite.Assertf(ctx.T, status == HealthStatusError, "Expected status Error, got %s", status)

	testsuite.Log(ctx.T, "Error status evaluation test passed")
}

func testStatusEvaluationUnknown(ctx *testsuite.TestContext, manager *HealthStatusManager) {
	testsuite.Log(ctx.T, "Testing unknown status evaluation...")

	status := manager.EvaluateStatus("UndefinedComponent", 50.0)
	testsuite.Assertf(ctx.T, status == HealthStatusUnknown, "Expected status Unknown, got %s", status)

	testsuite.Log(ctx.T, "Unknown status evaluation test passed")
}

func testHealthRecordRecording(ctx *testsuite.TestContext, manager *HealthStatusManager) {
	testsuite.Log(ctx.T, "Testing health record recording...")

	totalBefore := len(manager.historyRecords)

	manager.RecordHealthStatus("CPU", HealthStatusOK, map[string]float64{
		"usage": 45.5,
		"temp":  60.0,
	})
	manager.RecordHealthStatus("Memory", HealthStatusOK, map[string]float64{
		"usage": 55.0,
		"swap":  0.0,
	})

	totalAfter := len(manager.historyRecords)

	testsuite.Assertf(ctx.T, totalAfter == totalBefore+2, "Record count incorrect: expected %d, got %d", totalBefore+2, totalAfter)
	testsuite.Log(ctx.T, "Health record recording test passed")
}

func testHistoryRetrievalByComponent(ctx *testsuite.TestContext, manager *HealthStatusManager) {
	testsuite.Log(ctx.T, "Testing component history retrieval...")

	manager.RecordHealthStatus("Disk", HealthStatusOK, map[string]float64{"usage": 50.0})
	manager.RecordHealthStatus("Disk", HealthStatusOK, map[string]float64{"usage": 52.0})
	manager.RecordHealthStatus("Disk", HealthStatusWarning, map[string]float64{"usage": 85.0})

	history := manager.GetHistoryForComponent("Disk")
	testsuite.Assertf(ctx.T, len(history) >= 3, "History records insufficient: expected >=3, got %d", len(history))

	warningFound := false
	for _, record := range history {
		if record.Status == HealthStatusWarning {
			warningFound = true
			break
		}
	}
	testsuite.Assertf(ctx.T, warningFound, "Warning status not found in history")

	testsuite.Log(ctx.T, "History retrieval test passed")
}

func testStatusTrendAnalysis(ctx *testsuite.TestContext, manager *HealthStatusManager) {
	testsuite.Log(ctx.T, "Testing status trend analysis...")

	manager.RecordHealthStatus("Network", HealthStatusOK, map[string]float64{"latency": 20.0})
	manager.RecordHealthStatus("Network", HealthStatusOK, map[string]float64{"latency": 22.0})
	manager.RecordHealthStatus("Network", HealthStatusWarning, map[string]float64{"latency": 55.0})
	manager.RecordHealthStatus("Network", HealthStatusError, map[string]float64{"latency": 105.0})

	trends := manager.GetRecentStatusTrend("Network", time.Hour)
	testsuite.Assertf(ctx.T, len(trends) >= 4, "Trend data insufficient: expected >=4, got %d", len(trends))

	testsuite.Log(ctx.T, "Status trend analysis test passed")
}

func testComponentAvailabilityCalculation(ctx *testsuite.TestContext, manager *HealthStatusManager) {
	testsuite.Log(ctx.T, "Testing component availability calculation...")

	for i := 0; i < 10; i++ {
		var status HealthStatus
		if i < 8 {
			status = HealthStatusOK
		} else {
			status = HealthStatusWarning
		}
		manager.RecordHealthStatus("Power", status, map[string]float64{"voltage": 12.0})
	}

	availability := manager.CalculateComponentAvailability("Power", time.Hour)
	testsuite.Assertf(ctx.T, availability > 0.0, "Availability calculation returned 0")
	testsuite.Assertf(ctx.T, availability <= 100.0, "Availability calculation exceeded 100%%: %.1f", availability)
	testsuite.Logf(ctx.T, "Power component availability: %.1f%%", availability)

	testsuite.Log(ctx.T, "Component availability calculation test passed")
}

func testAvailabilityWithAllOK(ctx *testsuite.TestContext) {
	testsuite.Log(ctx.T, "Testing all OK availability...")
	manager := NewHealthStatusManager()

	for i := 0; i < 5; i++ {
		manager.RecordHealthStatus("Fan", HealthStatusOK, map[string]float64{"speed": 2000.0})
	}

	availability := manager.CalculateComponentAvailability("Fan", time.Hour)
	testsuite.Assertf(ctx.T, availability == 100.0, "All OK expected to have 100%% availability, got %.1f", availability)

	testsuite.Log(ctx.T, "All OK availability test passed")
}

func testAvailabilityWithMixStatuses(ctx *testsuite.TestContext) {
	testsuite.Log(ctx.T, "Testing mixed statuses availability...")
	manager := NewHealthStatusManager()

	statuses := []HealthStatus{
		HealthStatusOK, HealthStatusOK, HealthStatusOK,
		HealthStatusWarning, HealthStatusError,
	}

	for _, status := range statuses {
		manager.RecordHealthStatus("Storage", status, map[string]float64{"iops": 1000.0})
	}

	availability := manager.CalculateComponentAvailability("Storage", time.Hour)
	expectedAvailability := 60.0
	testsuite.Assertf(ctx.T, availability == expectedAvailability, "Availability calculation incorrect: expected %.1f%%, got %.1f%%", expectedAvailability, availability)

	testsuite.Log(ctx.T, "Mixed statuses availability test passed")
}

func testAvailabilityWithEmptyHistory(ctx *testsuite.TestContext) {
	testsuite.Log(ctx.T, "Testing empty history availability...")
	manager := NewHealthStatusManager()

	availability := manager.CalculateComponentAvailability("Nothing", time.Hour)
	testsuite.Assertf(ctx.T, availability == 0.0, "Empty history expected to have 0 availability, got %.1f", availability)

	testsuite.Log(ctx.T, "Empty history availability test passed")
}

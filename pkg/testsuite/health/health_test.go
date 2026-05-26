package health

import (
	"testing"

	"govsan/pkg/testsuite"
)

func TestHealthTestSuite(t *testing.T) {
	lifecycle := testsuite.NewTestLifecycle()
	ctx := testsuite.NewTestContext(t)
	
	checker := NewHealthChecker()
	healthRecords := make(map[string]HealthStatus)
	
	lifecycle.BeforeSuite(func() {
		testsuite.Log(ctx.T, "Setting up HealthTestSuite...")
		lifecycle.BeforeTest(func(name string) {
			testsuite.Log(ctx.T, "Before health check:", name)
		})
		lifecycle.AfterTest(func(name string) {
			testsuite.Log(ctx.T, "After health check:", name)
		})
		ctx.Set("health_initialized", true)
	})
	
	lifecycle.AfterSuite(func() {
		testsuite.Log(ctx.T, "Tearing down HealthTestSuite...")
		testsuite.Log(ctx.T, "=== Health Summary ===")
		for component, status := range healthRecords {
			testsuite.Logf(ctx.T, "  %s: %s", component, status)
		}
	})
	
	tests := map[string]func(*testsuite.TestContext){
		"TestObjectHealth": func(ctx *testsuite.TestContext) {
			testObjectHealth(ctx, checker, &healthRecords)
		},
		"TestDiskHealth": func(ctx *testsuite.TestContext) {
			testDiskHealth(ctx, checker, &healthRecords)
		},
		"TestNetworkHealth": func(ctx *testsuite.TestContext) {
			testNetworkHealth(ctx, checker, &healthRecords)
		},
		"TestDataEfficiencyHealth": func(ctx *testsuite.TestContext) {
			testDataEfficiencyHealth(ctx, checker, &healthRecords)
		},
		"TestClusterSummary": func(ctx *testsuite.TestContext) {
			testClusterSummary(ctx, checker)
		},
	}
	
	lifecycle.RunSuite(t, tests)
}

func testObjectHealth(ctx *testsuite.TestContext, checker *SimpleHealthChecker, records *map[string]HealthStatus) {
	testsuite.Log(ctx.T, "Running TestObjectHealth...")
	status := checker.CheckObjectHealth()
	testsuite.RequireNoError(ctx.T, status.Error)
	(*records)["object"] = status.Status
}

func testDiskHealth(ctx *testsuite.TestContext, checker *SimpleHealthChecker, records *map[string]HealthStatus) {
	testsuite.Log(ctx.T, "Running TestDiskHealth...")
	status := checker.CheckDiskHealth()
	testsuite.RequireNoError(ctx.T, status.Error)
	(*records)["disk"] = status.Status
}

func testNetworkHealth(ctx *testsuite.TestContext, checker *SimpleHealthChecker, records *map[string]HealthStatus) {
	testsuite.Log(ctx.T, "Running TestNetworkHealth...")
	status := checker.CheckNetworkHealth()
	testsuite.RequireNoError(ctx.T, status.Error)
	(*records)["network"] = status.Status
}

func testDataEfficiencyHealth(ctx *testsuite.TestContext, checker *SimpleHealthChecker, records *map[string]HealthStatus) {
	testsuite.Log(ctx.T, "Running TestDataEfficiencyHealth...")
	status := checker.CheckDataEfficiency()
	testsuite.RequireNoError(ctx.T, status.Error)
	(*records)["data_efficiency"] = status.Status
}

func testClusterSummary(ctx *testsuite.TestContext, checker *SimpleHealthChecker) {
	testsuite.Log(ctx.T, "Running TestClusterSummary...")
	summary := checker.GetClusterSummary()
	testsuite.Assertf(ctx.T, summary.TotalChecks > 0, "No health checks performed")
	testsuite.Logf(ctx.T, "Cluster summary: %d checks, %d OK, %d Warning, %d Error",
		summary.TotalChecks, summary.OKCount, summary.WarningCount, summary.ErrorCount)
}

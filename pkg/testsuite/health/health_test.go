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
		"TestClusterSummary": func(ctx *testsuite.TestContext) {
			testClusterSummary(ctx, checker)
		},
	}

	lifecycle.RunSuite(t, tests)
}

func testClusterSummary(ctx *testsuite.TestContext, checker *SimpleHealthChecker) {
	testsuite.Log(ctx.T, "Running TestClusterSummary...")
	summary := checker.GetClusterSummary()
	testsuite.Assertf(ctx.T, summary.TotalChecks > 0, "No health checks performed")
	testsuite.Logf(ctx.T, "Cluster summary: %d checks, %d OK, %d Warning, %d Error",
		summary.TotalChecks, summary.OKCount, summary.WarningCount, summary.ErrorCount)
}

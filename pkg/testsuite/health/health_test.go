package health

import (
	"testing"

	"github.com/vmware/govmomi/vim25/types"

	"govsan/pkg/testsuite"
)

func TestHealthTestSuite(t *testing.T) {
	ctx := testsuite.NewTestContext(t)
	lifecycle := testsuite.NewTestLifecycle(ctx)

	var checker *RealHealthChecker

	lifecycle.BeforeSuite(func() {
		testsuite.Log(ctx.T, "Setting up HealthTestSuite...")

		// Require real vCenter or vcsim connection
		testsuite.Log(ctx.T, "Connecting to vCenter (real or vcsim)...")
		vc, err := testsuite.SetupVCConnection(ctx)
		if err != nil {
			t.Skipf("Skipping test: Failed to connect to vCenter/vcsim: %v", err)
			return
		}

		if vc == nil {
			t.Skip("Skipping test: No vCenter/vcsim connection available")
			return
		}

		testsuite.Log(ctx.T, "Successfully connected to vCenter/vcsim")

		// Get cluster from context (set by SetupVCConnection)
		clusterVal, ok := ctx.Get("target_cluster")
		if !ok || clusterVal == nil {
			t.Skip("Skipping test: No target cluster found in context")
			return
		}

		cluster, ok := clusterVal.(types.ManagedObjectReference)
		if !ok {
			t.Skip("Skipping test: Invalid cluster type in context")
			return
		}

		// Create RealHealthChecker with actual API calls
		checker, err = NewRealHealthChecker(ctx.Ctx, vc, cluster)
		if err != nil {
			t.Skipf("Skipping test: Failed to create RealHealthChecker: %v", err)
			return
		}

		testsuite.Log(ctx.T, "Using RealHealthChecker with real vSAN API calls")

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

		// Cleanup vCenter connection
		if checker != nil {
			testsuite.Log(ctx.T, "Closing vCenter connection...")
			if err := checker.Close(); err != nil {
				testsuite.Logf(ctx.T, "Error closing connection: %v", err)
			}
		}
	})

	tests := map[string]func(*testsuite.TestContext){
		"TestClusterSummary": func(ctx *testsuite.TestContext) {
			testClusterSummary(ctx, checker)
		},
		"TestVsanClusterHealth": func(ctx *testsuite.TestContext) {
			testVsanClusterHealth(ctx, checker)
		},
	}

	lifecycle.RunSuite(t, tests)
}

func testVsanClusterHealth(ctx *testsuite.TestContext, checker HealthChecker) {
	testsuite.Log(ctx.T, "Running TestVsanClusterHealth...")

	// Perform detailed vSAN health checks
	objectHealth := checker.CheckObjectHealth()
	diskHealth := checker.CheckDiskHealth()
	networkHealth := checker.CheckNetworkHealth()
	dataEfficiency := checker.CheckDataEfficiency()

	testsuite.Logf(ctx.T, "Object Health: %s - %s", objectHealth.Status, objectHealth.Message)
	testsuite.Logf(ctx.T, "Disk Health: %s - %s", diskHealth.Status, diskHealth.Message)
	testsuite.Logf(ctx.T, "Network Health: %s - %s", networkHealth.Status, networkHealth.Message)
	testsuite.Logf(ctx.T, "Data Efficiency: %s - %s", dataEfficiency.Status, dataEfficiency.Message)

	// Assert that we got results
	testsuite.RequireNoError(ctx.T, objectHealth.Error, "Object health check failed")
	testsuite.RequireNoError(ctx.T, diskHealth.Error, "Disk health check failed")
	testsuite.RequireNoError(ctx.T, networkHealth.Error, "Network health check failed")
	testsuite.RequireNoError(ctx.T, dataEfficiency.Error, "Data efficiency check failed")
}

func testClusterSummary(ctx *testsuite.TestContext, checker HealthChecker) {
	testsuite.Log(ctx.T, "Running TestClusterSummary...")
	summary := checker.GetClusterSummary()
	testsuite.Assertf(ctx.T, summary.TotalChecks > 0, "No health checks performed")
	testsuite.Logf(ctx.T, "Cluster summary: %d checks, %d OK, %d Warning, %d Error",
		summary.TotalChecks, summary.OKCount, summary.WarningCount, summary.ErrorCount)
}

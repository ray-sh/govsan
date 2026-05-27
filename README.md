# GovSAN Testing Framework Technical Documentation

## Table of Contents

1. [Framework Execution Mechanism](#1-framework-execution-mechanism)
2. [Before/After Test Hook Customization Guide](#2-beforeafter-test-hook-customization-guide)
3. [Extending Health Check Tests](#3-extending-health-check-tests)
4. [Testing Guide for Real vCenter Environments](#4-testing-guide-for-real-vcenter-environments)

---

## 1. Framework Execution Mechanism

### 1.1 Overall Architecture

The testing framework adopts a **functional composition** design pattern, completely eschewing inheritance. Code reuse is achieved through shared interfaces and context objects.



govsan/pkg/testsuite/
├── core.go              # Core utility package (exported interfaces)
├── config/              # Configuration test subpackage
│   └── config_test.go
├── health/              # Health check subpackage
│   └── health_test.go
└── power/               # Performance test subpackage
└── power_test.go


### 1.2 Core Components

#### 1.2.1 TestContext - Testing Context

go
type TestContext struct {
    T         *testing.T      // Go standard testing object
    Ctx       context.Context // Context for goroutine communication
    data      map[string]interface{} // Thread-safe data storage
    dataLock  sync.RWMutex    // Data read/write lock
    StartTime time.Time       // Test start timestamp
}


Main Methods:

Set(key string, value interface{}): Store test data

Get(key string) (interface{}, bool): Retrieve test data

GetInt(key string, defaultVal int) int: Retrieve integer data

1.2.2 TestManager - Lifecycle Manager
Go
type TestManager struct {
    beforeSuite []func()
    afterSuite  []func()
    beforeTest  []func(name string)
    afterTest   []func(name string)
}


Main Methods:

BeforeSuite(fn func()): Register a hook before the suite runs

AfterSuite(fn func()): Register a hook after the suite completes

BeforeTest(fn func(name string)): Register a hook before each test

AfterTest(fn func(name string)): Register a hook after each test

RunSuite(t *testing.T, tests map[string]func(*TestContext)): Execute the test suite

1.2.3 Helper Tools

Assertions: Assertf, Requiref, RequireNoError

Logging: Log, Logf

1.3 Execution Flow
Test*TestSuite Entry Point
    │
    ├─ Create TestManager
    │
    ├─ Create TestContext
    │
    ├─ Register BeforeSuite Hooks ──→ Initialize resources
    │
    ├─ Register AfterSuite Hooks ──→ Cleanup resources
    │
    ├─ RunSuite() Execution
    │   │
    │   ├─ Execute BeforeSuite hook chain
    │   │
    │   ├─ Iterate through test cases
    │   │   │
    │   │   ├─ t.Run() start sub-test
    │   │   │   │
    │   │   │   ├─ Execute BeforeTest hook chain
    │   │   │   │
    │   │   │   ├─ defer register AfterTest hook
    │   │   │   │
    │   │   │   ├─ Execute test function
    │   │   │   │
    │   │   │   └─ Execute AfterTest hook chain
    │   │   │
    │   │   └─ Next test
    │   │
    │   └─ defer Execute AfterSuite hook chain
    │
    └─ Test End

1.4 Key Lifecycle Nodes
Node	Timing	Purpose
BeforeSuite	Before any tests run	Establish DB connections, initialize environment, load configs
BeforeTest	Before an individual test	Prepare test data, record start time, set breakpoints
AfterTest	After an individual test	Clean up test data, record end time, generate logs
AfterSuite	After all tests complete	Close DB connections, clean up temporary files, generate reports
1.5 Inter-module Interaction
Subpackages (config/health/power)
    │
    ├─ import "govsan/pkg/testsuite"
    │
    ├─ Use NewTestManager() to manage lifecycle
    ├─ Use NewTestContext(t) to create context
    ├─ Use Log/Logf for logging
    ├─ Use Assertf/Requiref for assertions
    │
    └─ Call lifecycle.RunSuite() to execute tests

2. Before/After Test Hook Customization Guide
2.1 How Hooks Work

Hook functions are registered via chaining and are executed sequentially at the corresponding lifecycle node in the order they were registered.

Execution Order:

Registration: Hook1 → Hook2 → Hook3
Execution: Hook1 → Hook2 → Hook3

2.2 Customizable Hook Types
Hook Type	Function Signature	Parameter Description	Registration Method
BeforeSuite	func()	None	lifecycle.BeforeSuite()
AfterSuite	func()	None	lifecycle.AfterSuite()
BeforeTest	func(name string)	name: Test name	lifecycle.BeforeTest()
AfterTest	func(name string)	name: Test name	lifecycle.AfterTest()
2.3 Use Cases
2.3.1 BeforeSuite Hook

Purpose: Resource Initialization

Go
package health

import (
    "testing"
    "govsan/pkg/testsuite"
)

func TestHealthTestSuite(t *testing.T) {
    lifecycle := testsuite.NewTestManager()
    ctx := testsuite.NewTestContext(t)

    var client *VSanClient
    lifecycle.BeforeSuite(func() {
        testsuite.Log(ctx.T, "Establishing VSAN connection...")
        // Initialize connection
        client = NewVSanClient("192.168.1.100")
        err := client.Connect()
        testsuite.RequireNoError(ctx.T, err, "Failed to connect to VSAN")

        // Store in context
        ctx.Set("vsan_client", client)
    })

    // ... Register other hooks
    // ... Execute tests
}

2.3.2 AfterSuite Hook

Purpose: Resource Cleanup

Go
lifecycle.AfterSuite(func() {
    if client != nil {
        testsuite.Log(ctx.T, "Closing VSAN connection...")
        err := client.Disconnect()
        if err != nil {
            testsuite.Log(ctx.T, "Error during disconnect:", err)
        }
    }
})

2.3.3 BeforeTest Hook

Purpose: Test Preparation

Go
lifecycle.BeforeTest(func(name string) {
    testsuite.Log(ctx.T, "===== Starting Test:", name, "=====")

    // Specific preparation based on test name
    if strings.Contains(name, "Disk") {
        testsuite.Log(ctx.T, "Warming up disk check module...")
    }
})

2.3.4 AfterTest Hook

Purpose: Test Cleanup

Go
lifecycle.AfterTest(func(name string) {
    testsuite.Log(ctx.T, "===== Test Finished:", name, "=====")

    // Clean up data generated by this test
    if strings.Contains(name, "Object") {
        testsuite.Log(ctx.T, "Cleaning up object check data...")
    }
})

3. Extending Health Check Tests
3.1 Naming Conventions

Follow Go testing file naming conventions:

Main test file: health_test.go

Sub-module tests: {module}_health_test.go (e.g., disk_health_test.go)

3.2 Directory Structure
pkg/testsuite/health/
├── health_test.go              # Main test suite
├── disk_health_test.go         # Disk health check (New)
├── object_health_test.go       # Object health check (New)
└── network_health_test.go      # Network health check (New)

3.3 Test Case Writing Standards
3.3.1 Creating a New Test File

Example disk_health_test.go:

Go
package health

import (
    "testing"
    "govsan/pkg/testsuite"
)

// ==================== Test Function ====================

func TestDiskHealthTestSuite(t *testing.T) {
    lifecycle := testsuite.NewTestManager()
    ctx := testsuite.NewTestContext(t)

    // Requires a real vCenter or vcsim connection
    vc, err := testsuite.SetupVCConnection(ctx)
    if err != nil || vc == nil {
        t.Skip("Requires vCenter or vcsim connection")
    }

    cluster, _ := ctx.Get("target_cluster")
    checker, _ := NewRealHealthChecker(ctx.Ctx, vc, cluster.(types.ManagedObjectReference))
    diskReports := make(map[string]DiskHealthReport)

    // Register hooks
    lifecycle.BeforeSuite(func() {
        testsuite.Log(ctx.T, "[DiskHealth] Initializing disk check module...")
        ctx.Set("disk_checker", checker)
    })

    lifecycle.BeforeTest(func(name string) {
        testsuite.Log(ctx.T, "[DiskHealth] Preparing:", name)
    })

    lifecycle.AfterTest(func(name string) {
        testsuite.Log(ctx.T, "[DiskHealth] Finished:", name)
    })

    // Define test cases
    tests := map[string]func(*testsuite.TestContext){
        "TestDiskCapacity": func(ctx *testsuite.TestContext) {
            testDiskCapacity(ctx, checker, &diskReports)
        },
        "TestDiskLatency": func(ctx *testsuite.TestContext) {
            testDiskLatency(ctx, checker, &diskReports)
        },
    }

    lifecycle.RunSuite(t, tests)
}

// ==================== Test Case Implementation ====================

type DiskHealthReport struct {
    Path       string
    CapacityGB int
    Status     HealthStatus
}

func testDiskCapacity(ctx *testsuite.TestContext, checker *HealthChecker, reports *map[string]DiskHealthReport) {
    testsuite.Log(ctx.T, "Testing disk capacity...")

    disks, err := checker.ListDisks()
    testsuite.RequireNoError(ctx.T, err, "Failed to list disks")

    for _, disk := range disks {
        cap, err := checker.CheckDiskCapacity(disk.Path)
        testsuite.RequireNoError(ctx.T, err)

        // Assert capacity is above threshold
        testsuite.Assertf(ctx.T, cap >= 100, "Disk %s low capacity (Need 100GB, Got %dGB)", disk.Path, cap)

        // Store report
        (*reports)[disk.Path] = DiskHealthReport{
            Path:       disk.Path,
            CapacityGB: cap,
            Status:     HealthStatusOK,
        }
    }
}

3.4 Registering New Tests

No additional configuration is required; the Go test tool automatically identifies new test files.

How to run:

Bash
# Run all health tests
go test -v ./pkg/testsuite/health/

# Run a specific test suite
go test -v ./pkg/testsuite/health -run TestDiskHealthTestSuite

3.5 Dependency Management

Ensure new tests correctly import the core package:

Go
import "govsan/pkg/testsuite"


Use Go modules to manage dependencies:

Bash
go mod tidy

4. Testing Guide for Real vCenter Environments
4.1 Environment Preparation

Set the following environment variables before running tests:

Bash
# Required variables
export GOVC_URL="[https://your-vcenter.example.com/sdk](https://your-vcenter.example.com/sdk)"
export GOVC_USERNAME="administrator@vsphere.local"
export GOVC_PASSWORD="your-password"

# Optional variables
export GOVC_INSECURE="true"                    # Skip cert verification (common for test envs)
export VSAN_CLUSTER_PATH="/datacenter/cluster" # Specify target cluster path

4.2 Using vcsim Simulator (Recommended for Dev)

If a real vCenter is unavailable, use the simulator provided by govmomi:

Bash
# Start vcsim
vcsim -start

# Point environment variables to the simulator
export GOVC_URL="https://localhost:8989/sdk"
export GOVC_USERNAME="user"
export GOVC_PASSWORD="password"
export GOVC_INSECURE="true"

4.3 Execution Flow
┌─────────────────────────────────────────────────────────────┐
│  1. BeforeSuite Phase                                       │
│     └─ SetupVCConnection() Establish vCenter connection      │
│         ├─ Read env vars (GOVC_URL, GOVC_USERNAME, ...)     │
│         ├─ Establish govmomi.Client connection              │
│         ├─ Initialize vSAN Client                           │
│         └─ Find target cluster and store in TestContext     │
├─────────────────────────────────────────────────────────────┤
│  2. Test Execution Phase                                    │
│     └─ Use RealHealthChecker to call real vSAN APIs         │
│         ├─ CheckObjectHealth() → VsanQueryObjectIdentities  │
│         ├─ CheckDiskHealth()   → VsanClusterGetConfig       │
│         ├─ CheckNetworkHealth() → Query host network info   │
│         └─ CheckDataEfficiency() → Query performance metrics│
├─────────────────────────────────────────────────────────────┤
│  3. AfterSuite Phase                                        │
│     └─ checker.Close() Close vCenter connection             │
└─────────────────────────────────────────────────────────────┘

4.4 Test Output Example
Bash
$ go test -v ./pkg/testsuite/health -run TestHealthTestSuite

=== RUN   TestHealthTestSuite
    health_test.go:18: Setting up HealthTestSuite...
    health_test.go:21: Connecting to vCenter (real or vcsim)...
    health_test.go:25: Successfully connected to vCenter/vcsim
--- PASS: TestHealthTestSuite (15.23s)
PASS
ok      govsan/pkg/testsuite/health     15.251s

Appendix
A. Run Commands
Command	Purpose
go test -v ./pkg/testsuite/...	Run all tests
go test -v ./pkg/testsuite/config/	Run a specific module
go test -v ./pkg/testsuite/health -run TestX	Run a specific test function
B. Project Contact

If you have any questions, please contact the maintenance team.

<task_progress>

[x] Translate README.md to English

[x] Update README.md file
</task_progress>
</write_to_file>
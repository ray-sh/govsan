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

text
govsan/pkg/testsuite/
├── core.go              # Core utility package (exported interfaces)
├── config/              # Configuration test subpackage
│   └── config_test.go
├── health/              # Health check subpackage
│   └── health_test.go
└── power/               # Performance test subpackage
    └── power_test.go

1.2 Core Components
1.2.1 TestContext - Testing Context
Go
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
Plaintext
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
Plaintext
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

3. Extending Health Check Tests
3.1 Naming Conventions

Follow Go testing file naming conventions:

Main test file: health_test.go

Sub-module tests: {module}_health_test.go (e.g., disk_health_test.go)

3.2 Directory Structure
Plaintext
pkg/testsuite/health/
├── health_test.go              # Main test suite
├── disk_health_test.go         # Disk health check (New)
├── object_health_test.go       # Object health check (New)
└── network_health_test.go      # Network health check (New)

4. Testing Guide for Real vCenter Environments
4.1 Environment Preparation

Set the following environment variables before running tests:

Bash
# Required variables
export GOVC_URL="[https://your-vcenter.example.com/sdk](https://your-vcenter.example.com/sdk)"
export GOVC_USERNAME="administrator@vsphere.local"
export GOVC_PASSWORD="your-password"

# Optional variables
export GOVC_INSECURE="true"                    # Skip cert verification
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
Plaintext
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

Appendix
A. Run Commands
Command	Purpose
go test -v ./pkg/testsuite/...	Run all tests
go test -v ./pkg/testsuite/config/	Run a specific module
go test -v ./pkg/testsuite/health -run TestX	Run a specific test function
B. Project Contact

If you have any questions, please contact the maintenance team.

<task_progress>

[x] Fix Markdown rendering in README.md (added missing code blocks and spacing)
</task_progress>
</write_to_file>
# GovSAN 测试框架技术文档

## 目录

1. [框架执行机制说明](#1-框架执行机制说明)
2. [Before/After Test Hook 定制指南](#2-beforeafter-test-hook-定制指南)
3. [健康检查测试扩展步骤](#3-健康检查测试扩展步骤)

---

## 1. 框架执行机制说明

### 1.1 整体架构

测试框架采用**函数式组合**设计模式，完全摒弃继承，通过共享接口和上下文对象实现代码重用。

```
govsan/pkg/testsuite/
├── core.go              # 核心工具包（导出接口）
├── config/              # 配置测试子包
│   └── config_test.go
├── health/              # 健康检查子包
│   └── health_test.go
└── power/               # 性能测试子包
    └── power_test.go
```

### 1.2 核心组件

#### 1.2.1 TestContext - 测试上下文

```go
type TestContext struct {
    T         *testing.T      // Go 标准测试框架的测试对象
    Ctx       context.Context // 上下文对象，用于协程通信
    data      map[string]interface{} // 线程安全的数据存储
    dataLock  sync.RWMutex    // 数据读写锁
    StartTime time.Time       // 测试启动时间戳
}
```

**主要方法**：
- `Set(key string, value interface{})`：存储测试数据
- `Get(key string) (interface{}, bool)`：获取测试数据
- `GetInt(key string, defaultVal int) int`：获取整数数据

#### 1.2.2 TestLifecycle - 生命周期管理器

```go
type TestLifecycle struct {
    beforeSuite []func()
    afterSuite  []func()
    beforeTest  []func(name string)
    afterTest   []func(name string)
}
```

**主要方法**：
- `BeforeSuite(fn func())`：注册套件前钩子
- `AfterSuite(fn func())`：注册套件后钩子
- `BeforeTest(fn func(name string))`：注册测试前钩子
- `AfterTest(fn func(name string))`：注册测试后钩子
- `RunSuite(t *testing.T, tests map[string]func(*TestContext))`：执行测试套件

#### 1.2.3 辅助工具

- **断言**：`Assertf`, `Requiref`, `RequireNoError`
- **日志**：`Log`, `Logf`

### 1.3 完整执行流程

```
Test*TestSuite 入口
    │
    ├─ 创建 TestLifecycle
    │
    ├─ 创建 TestContext
    │
    ├─ 注册 BeforeSuite 钩子 ──→ 初始化资源
    │
    ├─ 注册 AfterSuite 钩子 ──→ 清理资源
    │
    ├─ RunSuite() 执行
    │   │
    │   ├─ 执行 BeforeSuite 钩子链
    │   │
    │   ├─ 遍历测试用例
    │   │   │
    │   │   ├─ t.Run() 启动子测试
    │   │   │   │
    │   │   │   ├─ 执行 BeforeTest 钩子链
    │   │   │   │
    │   │   │   ├─ defer 注册 AfterTest 钩子
    │   │   │   │
    │   │   │   ├─ 执行测试函数
    │   │   │   │
    │   │   │   └─ 执行 AfterTest 钩子链
    │   │   │
    │   │   └─ 下一个测试
    │   │
    │   └─ defer 执行 AfterSuite 钩子链
    │
    └─ 测试结束
```

### 1.4 关键生命周期节点

| 节点 | 执行时机 | 用途 |
|------|---------|------|
| BeforeSuite | 任何测试执行之前 | 建立数据库连接、初始化测试环境、加载配置 |
| BeforeTest | 单个测试执行前 | 准备测试数据、记录开始时间、设置断点 |
| AfterTest | 单个测试执行后 | 清理测试数据、记录结束时间、生成日志 |
| AfterSuite | 所有测试执行后 | 关闭数据库连接、清理临时文件、生成报告 |

### 1.5 模块间交互方式

```
子包 (config/health/power)
    │
    ├─ import "govsan/pkg/testsuite"
    │
    ├─ 使用 NewTestLifecycle() 创建生命周期
    ├─ 使用 NewTestContext(t) 创建上下文
    ├─ 使用 Log/Logf 记录日志
    ├─ 使用 Assertf/Requiref 进行断言
    │
    └─ 调用 lifecycle.RunSuite() 执行测试
```

---

## 2. Before/After Test Hook 定制指南

### 2.1 钩子工作原理

钩子函数通过**链式注册**，在对应生命周期节点按注册顺序依次执行。

**执行顺序**：
```
注册: Hook1 → Hook2 → Hook3
执行: Hook1 → Hook2 → Hook3
```

### 2.2 可定制钩子类型

| 钩子类型 | 函数签名 | 参数说明 | 注册方法 |
|---------|---------|---------|---------|
| BeforeSuite | `func()` | 无 | `lifecycle.BeforeSuite()` |
| AfterSuite | `func()` | 无 | `lifecycle.AfterSuite()` |
| BeforeTest | `func(name string)` | `name`: 测试名称 | `lifecycle.BeforeTest()` |
| AfterTest | `func(name string)` | `name`: 测试名称 | `lifecycle.AfterTest()` |

### 2.3 使用场景

#### 2.3.1 BeforeSuite 钩子

**用途**：资源初始化

```go
package health

import (
    "testing"
    "govsan/pkg/testsuite"
)

func TestHealthTestSuite(t *testing.T) {
    lifecycle := testsuite.NewTestLifecycle()
    ctx := testsuite.NewTestContext(t)

    var client *VSanClient
    lifecycle.BeforeSuite(func() {
        testsuite.Log(ctx.T, "正在建立 VSAN 连接...")
        // 初始化连接
        client = NewVSanClient("192.168.1.100")
        err := client.Connect()
        testsuite.RequireNoError(ctx.T, err, "连接 VSAN 失败")

        // 存储到上下文
        ctx.Set("vsan_client", client)
    })

    // ... 注册其他钩子
    // ... 执行测试
}
```

#### 2.3.2 AfterSuite 钩子

**用途**：资源清理

```go
lifecycle.AfterSuite(func() {
    if client != nil {
        testsuite.Log(ctx.T, "正在关闭 VSAN 连接...")
        err := client.Disconnect()
        if err != nil {
            testsuite.Log(ctx.T, "断开连接时出错:", err)
        }
    }
})
```

#### 2.3.3 BeforeTest 钩子

**用途**：测试准备

```go
lifecycle.BeforeTest(func(name string) {
    testsuite.Log(ctx.T, "===== 开始测试:", name, "=====")

    // 可以根据测试名称做不同的准备
    if strings.Contains(name, "Disk") {
        testsuite.Log(ctx.T, "正在预热磁盘检查模块...")
    }
})
```

#### 2.3.4 AfterTest 钩子

**用途**：测试清理

```go
lifecycle.AfterTest(func(name string) {
    testsuite.Log(ctx.T, "===== 测试结束:", name, "=====")

    // 清理该测试产生的数据
    if strings.Contains(name, "Object") {
        testsuite.Log(ctx.T, "正在清理对象检查数据...")
    }
})
```

### 2.4 完整钩子示例

```go
package config

import (
    "testing"
    "time"
    "govsan/pkg/testsuite"
)

func TestConfigTestSuite(t *testing.T) {
    lifecycle := testsuite.NewTestLifecycle()
    ctx := testsuite.NewTestContext(t)

    var configMap map[string]interface{}

    // ==================== BeforeSuite ====================
    lifecycle.BeforeSuite(func() {
        testsuite.Log(ctx.T, "[BeforeSuite] 初始化配置模块...")
        configMap = make(map[string]interface{})
        ctx.Set("config_map", configMap)
        ctx.Set("start_time", time.Now())
    })

    // ==================== AfterSuite ====================
    lifecycle.AfterSuite(func() {
        startTime, _ := ctx.Get("start_time")
        duration := time.Since(startTime.(time.Time))
        testsuite.Logf(ctx.T, "[AfterSuite] 测试总耗时: %v", duration)
    })

    // ==================== BeforeTest ====================
    lifecycle.BeforeTest(func(name string) {
        testsuite.Log(ctx.T, "[BeforeTest] 准备执行:", name)
        ctx.Set(name+"_start", time.Now())
    })

    // ==================== AfterTest ====================
    lifecycle.AfterTest(func(name string) {
        start, _ := ctx.Get(name + "_start")
        duration := time.Since(start.(time.Time))
        testsuite.Logf(ctx.T, "[AfterTest] %s 耗时: %v", name, duration)
    })

    // 测试用例
    tests := map[string]func(*testsuite.TestContext){
        "TestConfigLoading": func(ctx *testsuite.TestContext) {
            testsuite.Log(ctx.T, "正在加载配置...")
            testsuite.Assertf(ctx.T, true, "加载配置失败")
        },
        "TestConfigUpdate": func(ctx *testsuite.TestContext) {
            testsuite.Log(ctx.T, "正在更新配置...")
            testsuite.Assertf(ctx.T, true, "更新配置失败")
        },
    }

    lifecycle.RunSuite(t, tests)
}
```

---

## 3. 健康检查测试扩展步骤

### 3.1 文件命名规范

遵循 Go 测试文件命名规范：
- **主测试文件**：`health_test.go`
- **子模块测试**：`{模块名}_health_test.go`（如 `disk_health_test.go`）

### 3.2 目录结构

```
pkg/testsuite/health/
├── health_test.go              # 主测试套件
├── disk_health_test.go         # 磁盘健康检查（新）
├── object_health_test.go       # 对象健康检查（新）
└── network_health_test.go      # 网络健康检查（新）
```

### 3.3 测试用例编写标准

#### 3.3.1 新建测试文件

以 `disk_health_test.go` 为例：

```go
package health

import (
    "testing"
    "govsan/pkg/testsuite"
)

// ==================== 测试函数 ====================

func TestDiskHealthTestSuite(t *testing.T) {
    lifecycle := testsuite.NewTestLifecycle()
    ctx := testsuite.NewTestContext(t)

    // 需要真实 vCenter 或 vcsim 连接
    vc, err := testsuite.SetupVCConnection(ctx)
    if err != nil || vc == nil {
        t.Skip("需要 vCenter 或 vcsim 连接")
    }

    cluster, _ := ctx.Get("target_cluster")
    checker, _ := NewRealHealthChecker(ctx.Ctx, vc, cluster.(types.ManagedObjectReference))
    diskReports := make(map[string]DiskHealthReport)

    // 注册钩子
    lifecycle.BeforeSuite(func() {
        testsuite.Log(ctx.T, "[DiskHealth] 初始化磁盘检查模块...")
        ctx.Set("disk_checker", checker)
    })

    lifecycle.BeforeTest(func(name string) {
        testsuite.Log(ctx.T, "[DiskHealth] 准备:", name)
    })

    lifecycle.AfterTest(func(name string) {
        testsuite.Log(ctx.T, "[DiskHealth] 完成:", name)
    })

    // 定义测试用例
    tests := map[string]func(*testsuite.TestContext){
        "TestDiskCapacity": func(ctx *testsuite.TestContext) {
            testDiskCapacity(ctx, checker, &diskReports)
        },
        "TestDiskLatency": func(ctx *testsuite.TestContext) {
            testDiskLatency(ctx, checker, &diskReports)
        },
        "TestDiskErrorRate": func(ctx *testsuite.TestContext) {
            testDiskErrorRate(ctx, checker, &diskReports)
        },
    }

    lifecycle.RunSuite(t, tests)
}

// ==================== 测试用例实现 ====================

type DiskHealthReport struct {
    Path       string
    CapacityGB int
    Status     HealthStatus
}

func testDiskCapacity(ctx *testsuite.TestContext, checker *HealthChecker, reports *map[string]DiskHealthReport) {
    testsuite.Log(ctx.T, "测试磁盘容量...")

    disks, err := checker.ListDisks()
    testsuite.RequireNoError(ctx.T, err, "获取磁盘列表失败")

    for _, disk := range disks {
        cap, err := checker.CheckDiskCapacity(disk.Path)
        testsuite.RequireNoError(ctx.T, err)

        // 断言容量大于阈值
        testsuite.Assertf(ctx.T, cap >= 100, "磁盘 %s 容量不足 (需要 100GB, 实际 %dGB)", disk.Path, cap)

        // 存储报告
        (*reports)[disk.Path] = DiskHealthReport{
            Path:       disk.Path,
            CapacityGB: cap,
            Status:     HealthStatusOK,
        }
    }
}

func testDiskLatency(ctx *testsuite.TestContext, checker *HealthChecker, reports *map[string]DiskHealthReport) {
    testsuite.Log(ctx.T, "测试磁盘延迟...")

    disks, err := checker.ListDisks()
    testsuite.RequireNoError(ctx.T, err, "获取磁盘列表失败")

    for _, disk := range disks {
        latency, err := checker.MeasureDiskLatency(disk.Path)
        testsuite.RequireNoError(ctx.T, err)

        testsuite.Assertf(ctx.T, latency < 100.0, "磁盘 %s 延迟过高 (%.2fms)", disk.Path, latency)
    }
}

func testDiskErrorRate(ctx *testsuite.TestContext, checker *HealthChecker, reports *map[string]DiskHealthReport) {
    testsuite.Log(ctx.T, "测试磁盘错误率...")

    // 实现细节...
}
```

### 3.4 注册新测试到框架

**新测试文件无需额外配置**，Go 测试工具会自动识别。

运行方式：
```bash
# 运行所有 health 测试
go test -v ./pkg/testsuite/health/

# 运行特定测试
go test -v ./pkg/testsuite/health -run TestDiskHealthTestSuite
go test -v ./pkg/testsuite/health -run TestDiskHealthTestSuite/TestDiskCapacity
```

### 3.5 依赖管理

确保新测试正确导入核心包：

```go
import "govsan/pkg/testsuite"
```

使用 Go 模块系统管理依赖：
```bash
cd /home/lei/codes/govsan
go mod tidy
```

### 3.6 多文件测试示例

当一个模块有多个测试文件时：

```
health/
├── health_test.go          # 主套件（含 TestHealthTestSuite）
├── disk_health_test.go     # 磁盘套件（含 TestDiskHealthTestSuite）
└── object_health_test.go   # 对象套件（含 TestObjectHealthTestSuite）
```

运行所有：
```bash
go test -v ./pkg/testsuite/health/
```

运行单个：
```bash
go test -v ./pkg/testsuite/health -run TestDiskHealthTestSuite
```

### 3.7 验证扩展

添加新测试后，执行完整测试验证：

```bash
# 完整测试
cd /home/lei/codes/govsan
go test -v ./pkg/testsuite/...

# 预期输出类似
# ok    govsan/pkg/testsuite/config    0.002s
# ok    govsan/pkg/testsuite/health    0.003s
# ok    govsan/pkg/testsuite/power     0.003s
```

---

## 附录

### A. 运行测试命令

| 命令 | 用途 |
|------|------|
| `go test -v ./pkg/testsuite/...` | 运行所有测试 |
| `go test -v ./pkg/testsuite/config/` | 运行特定模块 |
| `go test -v ./pkg/testsuite/health -run TestX` | 运行特定测试函数 |

### B. 项目联系方式

如有问题，请联系维护团队。

package testsuite

import (
	"context"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// 基础工具 - 可复用函数
// =============================================================================

// TestContext 管理测试数据和状态
type TestContext struct {
	T         *testing.T
	Ctx       context.Context
	data      map[string]interface{}
	dataLock  sync.RWMutex
	StartTime time.Time
}

// NewTestContext 创建测试上下文
func NewTestContext(t *testing.T) *TestContext {
	return &TestContext{
		T:         t,
		Ctx:       context.Background(),
		data:      make(map[string]interface{}),
		StartTime: time.Now(),
	}
}

// Set 存储数据
func (tc *TestContext) Set(key string, value interface{}) {
	tc.dataLock.Lock()
	defer tc.dataLock.Unlock()
	tc.data[key] = value
}

// Get 获取数据
func (tc *TestContext) Get(key string) (interface{}, bool) {
	tc.dataLock.RLock()
	defer tc.dataLock.RUnlock()
	val, ok := tc.data[key]
	return val, ok
}

// GetInt 获取整数
func (tc *TestContext) GetInt(key string, defaultVal int) int {
	if val, ok := tc.Get(key); ok {
		if i, ok := val.(int); ok {
			return i
		}
	}
	return defaultVal
}

// =============================================================================
// 断言工具
// =============================================================================

// Assertf 断言条件，失败记录错误但继续执行
func Assertf(t *testing.T, condition bool, format string, args ...interface{}) {
	if !condition {
		t.Helper()
		t.Errorf(format, args...)
	}
}

// Requiref 断言条件，失败立即停止
func Requiref(t *testing.T, condition bool, format string, args ...interface{}) {
	if !condition {
		t.Helper()
		t.Fatalf(format, args...)
	}
}

// RequireNoError 要求无错误
func RequireNoError(t *testing.T, err error, msg ...string) {
	if err != nil {
		t.Helper()
		if len(msg) > 0 {
			t.Fatalf("unexpected error: %v - %s", err, msg[0])
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

// =============================================================================
// 日志工具
// =============================================================================

// Log 记录日志
func Log(t *testing.T, args ...interface{}) {
	t.Helper()
	t.Log(args...)
}

// Logf 记录格式化日志
func Logf(t *testing.T, format string, args ...interface{}) {
	t.Helper()
	t.Logf(format, args...)
}

// =============================================================================
// 生命周期管理
// =============================================================================

// TestLifecycle 管理测试生命周期钩子
type TestLifecycle struct {
	beforeSuite []func()
	afterSuite  []func()
	beforeTest  []func(name string)
	afterTest   []func(name string)
}

// NewTestLifecycle 创建生命周期管理器
func NewTestLifecycle() *TestLifecycle {
	return &TestLifecycle{}
}

// BeforeSuite 注册套件前钩子
func (tl *TestLifecycle) BeforeSuite(fn func()) {
	tl.beforeSuite = append(tl.beforeSuite, fn)
}

// AfterSuite 注册套件后钩子
func (tl *TestLifecycle) AfterSuite(fn func()) {
	tl.afterSuite = append(tl.afterSuite, fn)
}

// BeforeTest 注册测试前钩子
func (tl *TestLifecycle) BeforeTest(fn func(name string)) {
	tl.beforeTest = append(tl.beforeTest, fn)
}

// AfterTest 注册测试后钩子
func (tl *TestLifecycle) AfterTest(fn func(name string)) {
	tl.afterTest = append(tl.afterTest, fn)
}

// RunSuite 运行测试套件
func (tl *TestLifecycle) RunSuite(t *testing.T, tests map[string]func(*TestContext)) {
	t.Helper()
	
	// 执行套件前钩子
	for _, fn := range tl.beforeSuite {
		fn()
	}
	
	// 执行测试后清理钩子
	defer func() {
		for _, fn := range tl.afterSuite {
			fn()
		}
	}()
	
	// 创建测试上下文
	ctx := NewTestContext(t)
	
	// 运行每个测试
	for name, fn := range tests {
		t.Run(name, func(t *testing.T) {
			ctx.T = t // 更新到子测试的 t
			
			// 测试前钩子
			for _, h := range tl.beforeTest {
				h(name)
			}
			
			// 测试后钩子
			defer func() {
				for _, h := range tl.afterTest {
					h(name)
				}
			}()
			
			// 运行测试
			fn(ctx)
		})
	}
}

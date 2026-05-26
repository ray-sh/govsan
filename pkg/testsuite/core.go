package testsuite

import (
	"context"
	"errors"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/types"
	"govsan/pkg/vsan" // 导入自定义 vsan 包以组装 Service 实例
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

// NewChildContext 基于父上下文创建一个子测试上下文，继承其共享数据（解决数据丢失 Bug）
func NewChildContext(subT *testing.T, parent *TestContext) *TestContext {
	parent.dataLock.RLock()
	defer parent.dataLock.RUnlock()

	// 复制父数据映射表，保证子测试之间并发安全
	newData := make(map[string]interface{})
	for k, v := range parent.data {
		newData[k] = v
	}

	return &TestContext{
		T:         subT,
		Ctx:       parent.Ctx,
		data:      newData,
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
// 断言与日志辅助
// =============================================================================

// Assertf 软断言：不满足条件时报错，但测试继续执行
func Assertf(t *testing.T, cond bool, format string, args ...interface{}) {
	t.Helper()
	if !cond {
		t.Errorf(format, args...)
	}
}

// Requiref 硬断言：不满足条件时立刻终止测试
func Requiref(t *testing.T, cond bool, format string, args ...interface{}) {
	t.Helper()
	if !cond {
		t.Fatalf(format, args...)
	}
}

// RequireNoError 确保无错误：遇到 err 直接终止测试
func RequireNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// Log 打印行号精准定位的测试日志
func Log(t *testing.T, args ...interface{}) {
	t.Helper()
	t.Log(args...)
}

// Logf 格式化打印测试日志
func Logf(t *testing.T, format string, args ...interface{}) {
	t.Helper()
	t.Logf(format, args...)
}

// =============================================================================
// 核心：一键初始化真实 vCenter 客户端并存入上下文
// =============================================================================

// SetupVCConnection 读取环境变量，连接真实 vCenter 并将 vsan.Service 与目标集群注入 Context
// 返回 *govmomi.Client 以供在 AfterSuite 中安全登出
func SetupVCConnection(tc *TestContext) (*govmomi.Client, error) {
	tc.T.Helper()

	// 1. 读取环境变量
	vcURL := os.Getenv("GOVC_URL")
	if vcURL == "" {
		tc.T.Skip("未设置 GOVC_URL 环境变量，跳过真实 VC vSAN 健康测试。")
		return nil, nil
	}

	vcUser := os.Getenv("GOVC_USERNAME")
	vcPass := os.Getenv("GOVC_PASSWORD")
	insecure := os.Getenv("GOVC_INSECURE") == "true"
	clusterPath := os.Getenv("VSAN_CLUSTER_PATH")

	// 2. 解析 URL
	u, err := url.Parse(vcURL)
	if err != nil {
		return nil, err
	}
	if vcUser != "" && vcPass != "" {
		u.User = url.UserPassword(vcUser, vcPass)
	}

	// 3. 建立带有超时控制的连接
	connectCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	govmomiClient, err := govmomi.NewClient(connectCtx, u, insecure)
	if err != nil {
		return nil, err
	}

	// 4. 初始化自定义 vSAN 客户端与 Service
	vsanClient, err := vsan.NewClient(connectCtx, govmomiClient.Client)
	if err != nil {
		govmomiClient.Logout(context.Background())
		return nil, err
	}
	vsanService := vsan.NewService(vsanClient)

	// 5. 寻找目标集群
	finder := find.NewFinder(govmomiClient.Client, true)
	var clusterRef types.ManagedObjectReference

	if clusterPath != "" {
		cluster, err := finder.ClusterComputeResource(connectCtx, clusterPath)
		if err != nil {
			govmomiClient.Logout(context.Background())
			return nil, err
		}
		clusterRef = cluster.Reference()
	} else {
		clusters, err := finder.ClusterComputeResourceList(connectCtx, "*")
		if err != nil {
			govmomiClient.Logout(context.Background())
			return nil, err
		}
		if len(clusters) == 0 {
			govmomiClient.Logout(context.Background())
			return nil, errors.New("vCenter 中没有找到任何集群")
		}
		clusterRef = clusters[0].Reference()
		Logf(tc.T, "自动选择集群: %s", clusters[0].Name())
	}

	// 6. 存入上下文
	tc.Set("vsan_service", vsanService)
	tc.Set("target_cluster", clusterRef)
	tc.Set("initialized", true)

	return govmomiClient, nil
}

// =============================================================================
// 生命周期管理
// =============================================================================

// TestLifecycle 管理测试生命周期钩子
type TestLifecycle struct {
	parentCtx   *TestContext
	beforeSuite []func()
	afterSuite  []func()
	beforeTest  []func(name string)
	afterTest   []func(name string)
}

// NewTestLifecycle 创建绑定了父上下文的生命周期管理器（确保子测试能继承 BeforeSuite 的连接数据）
func NewTestLifecycle(parentCtx *TestContext) *TestLifecycle {
	return &TestLifecycle{
		parentCtx: parentCtx,
	}
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

	// 逐个执行测试用例
	for name, testFn := range tests {
		t.Run(name, func(subT *testing.T) {
			subT.Helper()

			// 核心修复：通过 NewChildContext 深度拷贝父级上下文里的连接会话数据
			var subCtx *TestContext
			if tl.parentCtx != nil {
				subCtx = NewChildContext(subT, tl.parentCtx)
			} else {
				subCtx = NewTestContext(subT)
			}

			// 执行单项测试前钩子
			for _, fn := range tl.beforeTest {
				fn(name)
			}

			defer func() {
				// 执行单项测试后钩子
				for _, fn := range tl.afterTest {
					fn(name)
				}
			}()

			testFn(subCtx)
		})
	}
}
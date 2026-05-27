package vsan

import (
	"context"
	"errors"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/types"
	vsantypes "github.com/vmware/govmomi/vsan/types"
)

var ErrNilClient = errors.New("vsan client is nil")

// Service 负责处理 vSAN 相关的纯业务逻辑
type Service struct {
	client *Client
}

// NewService 依赖注入 vSAN 客户端
func NewService(client *Client) *Service {
	return &Service{
		client: client,
	}
}

// FindCluster 根据路径或通配符寻找集群引用
func (s *Service) FindCluster(ctx context.Context, vimClient *vim25.Client, clusterPath string) (types.ManagedObjectReference, error) {
	finder := find.NewFinder(vimClient, true)

	// 如果指定了路径
	if clusterPath != "" {
		cluster, err := finder.ClusterComputeResource(ctx, clusterPath)
		if err != nil {
			return types.ManagedObjectReference{}, err
		}
		return cluster.Reference(), nil
	}

	// 否则默认查找第一个
	clusters, err := finder.ClusterComputeResourceList(ctx, "*")
	if err != nil {
		return types.ManagedObjectReference{}, err
	}
	if len(clusters) == 0 {
		return types.ManagedObjectReference{}, errors.New("no clusters found in vCenter")
	}

	return clusters[0].Reference(), nil
}

// GetClusterConfig 获取 vSAN 集群的基础配置
func (s *Service) GetClusterConfig(ctx context.Context, cluster types.ManagedObjectReference) (*vsantypes.VsanConfigInfoEx, error) {
	if s.client == nil {
		return nil, ErrNilClient
	}
	// 依托匿名组合，直接调用 govmomi 官方底层对应的方法
	return s.client.VsanClusterGetConfig(ctx, cluster)
}

// ReconfigureVsan 重新配置 vSAN 集群（包含业务逻辑：等待任务完成）
func (s *Service) ReconfigureVsan(ctx context.Context, cluster types.ManagedObjectReference, spec vsantypes.VimVsanReconfigSpec) error {
	if s.client == nil {
		return ErrNilClient
	}

	// 1. 发起底层的 vSAN 集群异步重配置任务
	task, err := s.client.VsanClusterReconfig(ctx, cluster, spec)
	if err != nil {
		return err
	}

	// 2. 阻塞等待任务完成 (原本在 client.go 里的逻辑完美上浮到了这里)
	_, err = task.WaitForResult(ctx, nil)
	return err
}

// GetHostConfig 获取单台 ESXi 主机的 vSAN 配置
func (s *Service) GetHostConfig(ctx context.Context, vsanSystem types.ManagedObjectReference) (*vsantypes.VsanHostConfigInfoEx, error) {
	if s.client == nil {
		return nil, ErrNilClient
	}
	return s.client.VsanHostGetConfig(ctx, vsanSystem)
}

// QueryObjectIdentities 获取集群内容易出问题的 vSAN 对象及健康状况
func (s *Service) QueryObjectIdentities(ctx context.Context, cluster types.ManagedObjectReference) (*vsantypes.VsanObjectIdentityAndHealth, error) {
	if s.client == nil {
		return nil, ErrNilClient
	}
	// 注意：此处需要根据你当前使用的 govmomi 版本匹配具体的方法名
	// 官方 vmodl 生成的方法一般带有 Vsan 前缀，例如 VsanQueryObjectIdentities
	// 如果参数不同，请根据 govmomi 的具体签名进行调整
	return s.client.VsanQueryObjectIdentities(ctx, cluster)
}

// QueryPerf 查询集群级的 vSAN 性能指标数据
func (s *Service) QueryPerf(ctx context.Context, cluster *types.ManagedObjectReference, queries []vsantypes.VsanPerfQuerySpec) ([]vsantypes.VsanPerfEntityMetricCSV, error) {
	if s.client == nil {
		return nil, ErrNilClient
	}
	// 同上，调用底层 VsanPerfQueryPerf 方法
	return s.client.VsanPerfQueryPerf(ctx, cluster, queries)
}
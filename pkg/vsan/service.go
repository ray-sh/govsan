package vsan

import (
	"context"
	"errors"

	"github.com/vmware/govmomi/vim25/types"
	vsantypes "github.com/vmware/govmomi/vsan/types"
)

var ErrNilClient = errors.New("vsan client is nil")

type Service struct {
	client *Client
}

func NewService(client *Client) *Service {
	return &Service{
		client: client,
	}
}

func (s *Service) GetClusterConfig(ctx context.Context, cluster types.ManagedObjectReference) (*vsantypes.VsanConfigInfoEx, error) {
	if s.client == nil {
		return nil, ErrNilClient
	}
	return s.client.GetClusterConfig(ctx, cluster)
}

func (s *Service) ReconfigureVsan(ctx context.Context, cluster types.ManagedObjectReference, spec vsantypes.VimVsanReconfigSpec) error {
	if s.client == nil {
		return ErrNilClient
	}
	return s.client.ReconfigureVsan(ctx, cluster, spec)
}

func (s *Service) GetHostConfig(ctx context.Context, vsanSystem types.ManagedObjectReference) (*vsantypes.VsanHostConfigInfoEx, error) {
	if s.client == nil {
		return nil, ErrNilClient
	}
	return s.client.GetHostConfig(ctx, vsanSystem)
}

func (s *Service) QueryObjectIdentities(ctx context.Context, cluster types.ManagedObjectReference) (*vsantypes.VsanObjectIdentityAndHealth, error) {
	if s.client == nil {
		return nil, ErrNilClient
	}
	return s.client.QueryObjectIdentities(ctx, cluster)
}

func (s *Service) QueryPerf(ctx context.Context, cluster *types.ManagedObjectReference, queries []vsantypes.VsanPerfQuerySpec) ([]vsantypes.VsanPerfEntityMetricCSV, error) {
	if s.client == nil {
		return nil, ErrNilClient
	}
	return s.client.QueryPerf(ctx, cluster, queries)
}

func (s *Service) IsVsanEnabled(ctx context.Context, cluster types.ManagedObjectReference) (bool, error) {
	if s.client == nil {
		return false, ErrNilClient
	}
	config, err := s.GetClusterConfig(ctx, cluster)
	if err != nil {
		return false, err
	}
	if config.Enabled == nil {
		return false, nil
	}
	return *config.Enabled, nil
}
package vsan

import (
	"context"
	"errors"

	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/govmomi/vsan"
	vsantypes "github.com/vmware/govmomi/vsan/types"
)

var ErrNilVIMClient = errors.New("vim client is nil")

type Client struct {
	*vsan.Client
	vimClient *vim25.Client
}

func NewClient(ctx context.Context, vimClient *vim25.Client) (*Client, error) {
	if vimClient == nil {
		return nil, ErrNilVIMClient
	}
	vc, err := vsan.NewClient(ctx, vimClient)
	if err != nil {
		return nil, err
	}
	return &Client{
		Client:    vc,
		vimClient: vimClient,
	}, nil
}

func (c *Client) GetVIMClient() *vim25.Client {
	return c.vimClient
}

func (c *Client) GetClusterConfig(ctx context.Context, cluster types.ManagedObjectReference) (*vsantypes.VsanConfigInfoEx, error) {
	return c.Client.VsanClusterGetConfig(ctx, cluster)
}

func (c *Client) ReconfigureVsan(ctx context.Context, cluster types.ManagedObjectReference, spec vsantypes.VimVsanReconfigSpec) error {
	task, err := c.Client.VsanClusterReconfig(ctx, cluster, spec)
	if err != nil {
		return err
	}
	_, err = task.WaitForResult(ctx, nil)
	return err
}

func (c *Client) GetHostConfig(ctx context.Context, vsanSystem types.ManagedObjectReference) (*vsantypes.VsanHostConfigInfoEx, error) {
	return c.Client.VsanHostGetConfig(ctx, vsanSystem)
}

func (c *Client) QueryObjectIdentities(ctx context.Context, cluster types.ManagedObjectReference) (*vsantypes.VsanObjectIdentityAndHealth, error) {
	return c.Client.VsanQueryObjectIdentities(ctx, cluster)
}

func (c *Client) QueryPerf(ctx context.Context, cluster *types.ManagedObjectReference, queries []vsantypes.VsanPerfQuerySpec) ([]vsantypes.VsanPerfEntityMetricCSV, error) {
	return c.Client.VsanPerfQueryPerf(ctx, cluster, queries)
}
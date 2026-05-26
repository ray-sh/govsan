package vsan

import (
	"context"
	"errors"
	"net/url"
	"os"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vsan"
)

var ErrNilVIMClient = errors.New("vim client is nil")

// Client 结构体：纯粹的数据载体
type Client struct {
	*vsan.Client            // 匿名组合：这让 Client 直接拥有了 VsanClusterGetConfig 等所有底层方法！
	vimClient *vim25.Client
}

// ---------------------------------------------------------
// 核心构造函数（保留，用于单元测试和依赖注入）
// ---------------------------------------------------------
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

// 获取底层发包引擎
func (c *Client) GetVIMClient() *vim25.Client {
	return c.vimClient
}

// ---------------------------------------------------------
// 真实环境连接辅助函数（强烈建议保留）
// ---------------------------------------------------------
func NewClientFromEnv(ctx context.Context) (*Client, error) {
	vcURL := os.Getenv("GOVC_URL")
	if vcURL == "" {
		return nil, errors.New("missing GOVC_URL environment variable")
	}

	u, err := url.Parse(vcURL)
	if err != nil {
		return nil, err
	}

	vcUser := os.Getenv("GOVC_USERNAME")
	vcPass := os.Getenv("GOVC_PASSWORD")
	if vcUser != "" && vcPass != "" {
		u.User = url.UserPassword(vcUser, vcPass)
	}

	insecure := os.Getenv("GOVC_INSECURE") == "true"

	govmomiClient, err := govmomi.NewClient(ctx, u, insecure)
	if err != nil {
		return nil, err
	}

	return NewClient(ctx, govmomiClient.Client)
}
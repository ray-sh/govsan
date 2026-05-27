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

// Client struct: pure data carrier
type Client struct {
	*vsan.Client            // Anonymous composition: allows Client to directly access all low-level methods like VsanClusterGetConfig
	vimClient *vim25.Client
}

// ---------------------------------------------------------
// Core constructor (retained for unit testing and dependency injection)
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

// GetVIMClient returns the underlying VIM client
func (c *Client) GetVIMClient() *vim25.Client {
	return c.vimClient
}

// ---------------------------------------------------------
// Real-world connection helper function (highly recommended to retain)
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
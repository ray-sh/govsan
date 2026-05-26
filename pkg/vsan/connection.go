// Package vsan provides vSAN integration utilities for test suites.
package vsan

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
)

// ConnectionConfig holds vCenter connection configuration
type ConnectionConfig struct {
	URL      string
	Username string
	Password string
	Insecure bool // Skip TLS verification
}

// DefaultConnectionConfig creates a connection config from environment variables
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		URL:      getEnvOrDefault("GOVC_URL", "https://vcenter.example.com/sdk"),
		Username: getEnvOrDefault("GOVC_USERNAME", "administrator@vsphere.local"),
		Password: getEnvOrDefault("GOVC_VC_PASSWORD", ""),
		Insecure: os.Getenv("GOVC_INSECURE") == "true",
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Connect establishes a connection to vCenter
func Connect(ctx context.Context, config *ConnectionConfig) (*govmomi.Client, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("vCenter URL is required")
	}
	if config.Password == "" {
		return nil, fmt.Errorf("vCenter password is required")
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Parse the vCenter URL
	url, err := soap.ParseURL(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vCenter URL: %w", err)
	}

	// Set credentials
	url.User = vim25.NewUserPassword(url.User.URL(), config.Username, config.Password)

	// Connect to vCenter
	client, err := govmomi.NewClient(ctx, url, config.Insecure)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to vCenter: %w", err)
	}

	return client, nil
}

// ConnectWithCredentials connects to vCenter using provided credentials
func ConnectWithCredentials(ctx context.Context, vcURL, username, password string, insecure bool) (*govmomi.Client, error) {
	return Connect(ctx, &ConnectionConfig{
		URL:      vcURL,
		Username: username,
		Password: password,
		Insecure: insecure,
	})
}

// NewVimClient creates a vim25 client for direct vSphere operations
func NewVimClient(ctx context.Context, config *ConnectionConfig) (*vim25.Client, error) {
	client, err := Connect(ctx, config)
	if err != nil {
		return nil, err
	}
	return client.Client, nil
}

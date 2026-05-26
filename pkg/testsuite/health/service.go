// Package health provides reusable health check components for test suites.
package health

import (
	"time"
)

// HealthStatus represents the health status of a component.
type HealthStatus string

const (
	// HealthStatusOK indicates the component is healthy.
	HealthStatusOK HealthStatus = "OK"
	// HealthStatusWarning indicates the component has a warning condition.
	HealthStatusWarning HealthStatus = "Warning"
	// HealthStatusError indicates the component has an error condition.
	HealthStatusError HealthStatus = "Error"
	// HealthStatusUnknown indicates the component status is unknown.
	HealthStatusUnknown HealthStatus = "Unknown"
)

// HealthResult represents the result of a health check.
type HealthResult struct {
	Status    HealthStatus
	Message   string
	Error     error
	Timestamp string
}

// ClusterSummary represents a summary of cluster health.
type ClusterSummary struct {
	TotalChecks   int
	OKCount       int
	WarningCount  int
	ErrorCount    int
	Details       []HealthResult
}

// ThresholdRule represents a rule for evaluating health based on thresholds.
type ThresholdRule struct {
	ComponentType string
	WarningLevel  float64
	ErrorLevel    float64
}

// HealthHistoryRecord represents a historical record of health status.
type HealthHistoryRecord struct {
	Timestamp  time.Time
	Component  string
	Status     HealthStatus
	Metrics    map[string]float64
}

// HealthChecker defines the interface for health check operations.
type HealthChecker interface {
	CheckObjectHealth() *HealthResult
	CheckDiskHealth() *HealthResult
	CheckNetworkHealth() *HealthResult
	CheckDataEfficiency() *HealthResult
	GetClusterSummary() *ClusterSummary
}

// SimpleHealthChecker is a simple implementation of HealthChecker.
type SimpleHealthChecker struct{}

// NewHealthChecker creates a new instance of SimpleHealthChecker.
func NewHealthChecker() *SimpleHealthChecker {
	return &SimpleHealthChecker{}
}

// CheckObjectHealth checks the health of vSAN objects.
func (c *SimpleHealthChecker) CheckObjectHealth() *HealthResult {
	return &HealthResult{Status: HealthStatusOK, Message: "Object health is normal", Error: nil}
}

// CheckDiskHealth checks the health of disks.
func (c *SimpleHealthChecker) CheckDiskHealth() *HealthResult {
	return &HealthResult{Status: HealthStatusOK, Message: "Disk health is normal", Error: nil}
}

// CheckNetworkHealth checks the health of network.
func (c *SimpleHealthChecker) CheckNetworkHealth() *HealthResult {
	return &HealthResult{Status: HealthStatusWarning, Message: "Network latency is elevated", Error: nil}
}

// CheckDataEfficiency checks the health of data efficiency.
func (c *SimpleHealthChecker) CheckDataEfficiency() *HealthResult {
	return &HealthResult{Status: HealthStatusOK, Message: "Data efficiency is optimal", Error: nil}
}

// GetClusterSummary gets the summary of cluster health.
func (c *SimpleHealthChecker) GetClusterSummary() *ClusterSummary {
	return &ClusterSummary{
		TotalChecks: 4, OKCount: 3, WarningCount: 1, ErrorCount: 0,
		Details: []HealthResult{
			*c.CheckObjectHealth(), *c.CheckDiskHealth(),
			*c.CheckNetworkHealth(), *c.CheckDataEfficiency(),
		},
	}
}

// HealthStatusManager manages health status evaluation and history.
type HealthStatusManager struct {
	rules          map[string]*ThresholdRule
	historyRecords []*HealthHistoryRecord
	historyLock    chan struct{}
}

// NewHealthStatusManager creates a new instance of HealthStatusManager.
func NewHealthStatusManager() *HealthStatusManager {
	return &HealthStatusManager{
		rules:          make(map[string]*ThresholdRule),
		historyRecords: make([]*HealthHistoryRecord, 0),
		historyLock:    make(chan struct{}, 1),
	}
}

// AddThresholdRule adds a threshold rule for a component type.
func (m *HealthStatusManager) AddThresholdRule(componentType string, warningLevel, errorLevel float64) {
	m.rules[componentType] = &ThresholdRule{
		ComponentType: componentType,
		WarningLevel:  warningLevel,
		ErrorLevel:    errorLevel,
	}
}

// EvaluateStatus evaluates the health status based on the configured rules.
func (m *HealthStatusManager) EvaluateStatus(componentType string, currentValue float64) HealthStatus {
	rule, exists := m.rules[componentType]
	if !exists {
		return HealthStatusUnknown
	}

	if currentValue >= rule.ErrorLevel {
		return HealthStatusError
	} else if currentValue >= rule.WarningLevel {
		return HealthStatusWarning
	}

	return HealthStatusOK
}

// RecordHealthStatus records a health status in the history.
func (m *HealthStatusManager) RecordHealthStatus(component string, status HealthStatus, metrics map[string]float64) {
	m.lock()
	defer m.unlock()

	record := &HealthHistoryRecord{
		Timestamp: time.Now(),
		Component: component,
		Status:    status,
		Metrics:   make(map[string]float64),
	}

	for k, v := range metrics {
		record.Metrics[k] = v
	}

	m.historyRecords = append(m.historyRecords, record)
}

// GetHistoryForComponent retrieves health history for a specific component.
func (m *HealthStatusManager) GetHistoryForComponent(component string) []*HealthHistoryRecord {
	m.lock()
	defer m.unlock()

	var result []*HealthHistoryRecord
	for _, record := range m.historyRecords {
		if record.Component == component {
			result = append(result, record)
		}
	}
	return result
}

// GetRecentStatusTrend retrieves recent status trend for a component.
func (m *HealthStatusManager) GetRecentStatusTrend(component string, duration time.Duration) []HealthStatus {
	m.lock()
	defer m.unlock()

	cutoff := time.Now().Add(-duration)
	var trends []HealthStatus

	for _, record := range m.historyRecords {
		if record.Component == component && record.Timestamp.After(cutoff) {
			trends = append(trends, record.Status)
		}
	}
	return trends
}

// CalculateComponentAvailability calculates availability based on history.
func (m *HealthStatusManager) CalculateComponentAvailability(component string, duration time.Duration) float64 {
	trends := m.GetRecentStatusTrend(component, duration)
	if len(trends) == 0 {
		return 0.0
	}

	okCount := 0
	for _, status := range trends {
		if status == HealthStatusOK {
			okCount++
		}
	}

	return float64(okCount) / float64(len(trends)) * 100.0
}

func (m *HealthStatusManager) lock() {
	m.historyLock <- struct{}{}
}

func (m *HealthStatusManager) unlock() {
	<-m.historyLock
}

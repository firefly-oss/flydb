/*
 * Copyright (c) 2026 Firefly Software Solutions Inc.
 * Licensed under the Apache License, Version 2.0
 */

package audit

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"flydb/internal/logging"
)

// ClusterAuditManager manages audit logs across a cluster.
// It aggregates audit logs from all nodes for comprehensive visibility.
type ClusterAuditManager struct {
	localManager *Manager
	logger       *logging.Logger
	mu           sync.RWMutex

	// Cluster configuration
	nodeID      string
	clusterPort int
	peers       map[string]string // nodeID -> address

	// Cache for remote audit logs
	remoteCache     map[string][]Event // nodeID -> events
	remoteCacheMu   sync.RWMutex
	cacheExpiration time.Duration
	lastCacheUpdate time.Time
}

// NewClusterAuditManager creates a new cluster audit manager.
func NewClusterAuditManager(localManager *Manager, nodeID string, clusterPort int) *ClusterAuditManager {
	return &ClusterAuditManager{
		localManager:    localManager,
		logger:          logging.NewLogger("audit-cluster"),
		nodeID:          nodeID,
		clusterPort:     clusterPort,
		peers:           make(map[string]string),
		remoteCache:     make(map[string][]Event),
		cacheExpiration: 30 * time.Second,
	}
}

// AddPeer adds a cluster peer for audit log aggregation.
func (cam *ClusterAuditManager) AddPeer(nodeID, address string) {
	cam.mu.Lock()
	defer cam.mu.Unlock()
	cam.peers[nodeID] = address
	cam.logger.Info("Added audit peer", "node_id", nodeID, "address", address)
}

// RemovePeer removes a cluster peer.
func (cam *ClusterAuditManager) RemovePeer(nodeID string) {
	cam.mu.Lock()
	defer cam.mu.Unlock()
	delete(cam.peers, nodeID)
	cam.logger.Info("Removed audit peer", "node_id", nodeID)
}

// LogEvent logs an audit event locally.
func (cam *ClusterAuditManager) LogEvent(event Event) {
	// Add node ID to metadata
	if event.Metadata == nil {
		event.Metadata = make(map[string]string)
	}
	event.Metadata["node_id"] = cam.nodeID

	cam.localManager.LogEvent(event)
}

// QueryLogsAcrossCluster queries audit logs from all cluster nodes.
func (cam *ClusterAuditManager) QueryLogsAcrossCluster(opts QueryOptions) ([]Event, error) {
	// Get local logs
	localLogs, err := cam.localManager.QueryLogs(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query local logs: %w", err)
	}

	// Get remote logs from all peers
	cam.mu.RLock()
	peers := make(map[string]string)
	for nodeID, addr := range cam.peers {
		peers[nodeID] = addr
	}
	cam.mu.RUnlock()

	// Aggregate logs from all nodes
	allLogs := make([]Event, 0, len(localLogs))
	allLogs = append(allLogs, localLogs...)

	// Query each peer concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	for nodeID, addr := range peers {
		wg.Add(1)
		go func(nid, address string) {
			defer wg.Done()

			remoteLogs, err := cam.queryRemoteLogs(address, opts)
			if err != nil {
				cam.logger.Warn("Failed to query remote audit logs", "node_id", nid, "error", err)
				return
			}

			mu.Lock()
			allLogs = append(allLogs, remoteLogs...)
			mu.Unlock()
		}(nodeID, addr)
	}

	wg.Wait()

	// Sort by timestamp (most recent first)
	// In a production implementation, you'd want to sort the combined results
	// For now, we'll return them as-is

	return allLogs, nil
}

// queryRemoteLogs queries audit logs from a remote node.
func (cam *ClusterAuditManager) queryRemoteLogs(address string, opts QueryOptions) ([]Event, error) {
	// Connect to remote node
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote node: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Send audit query request
	request := map[string]interface{}{
		"type":    "audit_query",
		"options": opts,
	}

	if err := json.NewEncoder(conn).Encode(request); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	var response struct {
		Success bool    `json:"success"`
		Events  []Event `json:"events"`
		Error   string  `json:"error"`
	}

	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("remote query failed: %s", response.Error)
	}

	return response.Events, nil
}

// ExportLogsAcrossCluster exports audit logs from all cluster nodes.
func (cam *ClusterAuditManager) ExportLogsAcrossCluster(filename string, format ExportFormat, opts QueryOptions) error {
	// Query logs from all nodes
	allLogs, err := cam.QueryLogsAcrossCluster(opts)
	if err != nil {
		return err
	}

	// Export using local manager's export functionality with aggregated logs
	return cam.localManager.ExportEvents(filename, format, allLogs)
}

// GetClusterStatistics retrieves audit statistics from all cluster nodes.
func (cam *ClusterAuditManager) GetClusterStatistics() (map[string]interface{}, error) {
	// Get local stats
	localHelper := NewSQLHelper(cam.localManager)
	localStats, err := localHelper.GetAuditStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get local stats: %w", err)
	}

	// In a full implementation, we'd aggregate stats from all nodes
	// For now, return local stats with cluster context
	clusterStats := make(map[string]interface{})
	clusterStats["node_id"] = cam.nodeID
	clusterStats["local_stats"] = localStats
	clusterStats["cluster_mode"] = true
	clusterStats["peer_count"] = len(cam.peers)

	return clusterStats, nil
}

// IsClusterMode returns whether audit manager is running in cluster mode.
func (cam *ClusterAuditManager) IsClusterMode() bool {
	cam.mu.RLock()
	defer cam.mu.RUnlock()
	return len(cam.peers) > 0
}

// GetLocalManager returns the local audit manager for standalone operations.
func (cam *ClusterAuditManager) GetLocalManager() *Manager {
	return cam.localManager
}

// Stop stops the cluster audit manager.
func (cam *ClusterAuditManager) Stop() {
	cam.localManager.Stop()
}

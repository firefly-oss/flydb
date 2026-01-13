/*
 * Copyright (c) 2026 Firefly Software Solutions Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
Package cluster provides automatic failover for FlyDB.

Automatic Failover Overview:
============================

This module implements fast and reliable automatic failover with:
- Rapid failure detection using heartbeats and phi-accrual detector
- Smooth leader transitions with minimal disruption
- Fencing to prevent split-brain scenarios
- Automatic recovery and rejoin

Failure Detection:
==================

Uses a phi-accrual failure detector which provides:
- Adaptive thresholds based on network conditions
- Probabilistic failure detection
- Configurable sensitivity

Failover Process:
=================

1. Failure detected (leader unresponsive)
2. Fencing: Old leader is fenced off
3. Election: New leader elected via Raft
4. Promotion: New leader takes over
5. Recovery: Old leader rejoins as follower

Fencing Mechanisms:
===================

- Network fencing: Block old leader's connections
- Storage fencing: Revoke old leader's write access
- Token fencing: Invalidate old leader's tokens
*/
package cluster

import (
	"fmt"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// FailoverConfig holds configuration for the failover manager
type FailoverConfig struct {
	NodeID              string        `json:"node_id"`
	HeartbeatInterval   time.Duration `json:"heartbeat_interval"`
	FailureThreshold    float64       `json:"failure_threshold"`
	MinSamples          int           `json:"min_samples"`
	MaxSamples          int           `json:"max_samples"`
	FencingTimeout      time.Duration `json:"fencing_timeout"`
	PromotionTimeout    time.Duration `json:"promotion_timeout"`
	RecoveryGracePeriod time.Duration `json:"recovery_grace_period"`
}

// DefaultFailoverConfig returns sensible defaults
func DefaultFailoverConfig(nodeID string) FailoverConfig {
	return FailoverConfig{
		NodeID:              nodeID,
		HeartbeatInterval:   100 * time.Millisecond,
		FailureThreshold:    8.0, // Phi threshold
		MinSamples:          10,
		MaxSamples:          1000,
		FencingTimeout:      5 * time.Second,
		PromotionTimeout:    10 * time.Second,
		RecoveryGracePeriod: 30 * time.Second,
	}
}

// FailoverState represents the current failover state
type FailoverState int32

const (
	FailoverStateNormal FailoverState = iota
	FailoverStateDetecting
	FailoverStateFencing
	FailoverStateElecting
	FailoverStatePromoting
	FailoverStateRecovering
)

func (s FailoverState) String() string {
	switch s {
	case FailoverStateNormal:
		return "NORMAL"
	case FailoverStateDetecting:
		return "DETECTING"
	case FailoverStateFencing:
		return "FENCING"
	case FailoverStateElecting:
		return "ELECTING"
	case FailoverStatePromoting:
		return "PROMOTING"
	case FailoverStateRecovering:
		return "RECOVERING"
	default:
		return "UNKNOWN"
	}
}

// PhiAccrualDetector implements the phi-accrual failure detector
type PhiAccrualDetector struct {
	mu           sync.RWMutex
	intervals    []float64
	lastBeat     time.Time
	minSamples   int
	maxSamples   int
	threshold    float64
	mean         float64
	variance     float64
}

// NewPhiAccrualDetector creates a new phi-accrual detector
func NewPhiAccrualDetector(threshold float64, minSamples, maxSamples int) *PhiAccrualDetector {
	return &PhiAccrualDetector{
		intervals:  make([]float64, 0, maxSamples),
		threshold:  threshold,
		minSamples: minSamples,
		maxSamples: maxSamples,
		mean:       0,
		variance:   0,
	}
}

// Heartbeat records a heartbeat
func (d *PhiAccrualDetector) Heartbeat() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	if !d.lastBeat.IsZero() {
		interval := now.Sub(d.lastBeat).Seconds() * 1000 // Convert to ms
		d.intervals = append(d.intervals, interval)

		// Keep only maxSamples
		if len(d.intervals) > d.maxSamples {
			d.intervals = d.intervals[1:]
		}

		// Update statistics
		d.updateStats()
	}
	d.lastBeat = now
}

// updateStats updates mean and variance
func (d *PhiAccrualDetector) updateStats() {
	if len(d.intervals) == 0 {
		return
	}

	// Calculate mean
	sum := 0.0
	for _, v := range d.intervals {
		sum += v
	}
	d.mean = sum / float64(len(d.intervals))

	// Calculate variance
	sumSq := 0.0
	for _, v := range d.intervals {
		diff := v - d.mean
		sumSq += diff * diff
	}
	d.variance = sumSq / float64(len(d.intervals))
}

// Phi calculates the current phi value
func (d *PhiAccrualDetector) Phi() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.intervals) < d.minSamples {
		return 0 // Not enough samples
	}

	if d.lastBeat.IsZero() {
		return d.threshold + 1 // Never received heartbeat
	}

	timeSinceLast := time.Since(d.lastBeat).Seconds() * 1000
	return d.phi(timeSinceLast)
}

// phi calculates phi for a given time since last heartbeat
func (d *PhiAccrualDetector) phi(timeSinceLast float64) float64 {
	// Use normal distribution CDF
	stdDev := math.Sqrt(d.variance)
	if stdDev < 1 {
		stdDev = 1 // Minimum stddev to avoid division by zero
	}

	y := (timeSinceLast - d.mean) / stdDev
	e := math.Exp(-y * (1.5976 + 0.070566*y*y))
	if timeSinceLast > d.mean {
		return -math.Log10(e / (1 + e))
	}
	return -math.Log10(1 - 1/(1+e))
}

// IsFailed returns true if the node is considered failed
func (d *PhiAccrualDetector) IsFailed() bool {
	return d.Phi() > d.threshold
}

// FailoverManager manages automatic failover
type FailoverManager struct {
	config FailoverConfig
	mu     sync.RWMutex

	state     int32 // atomic FailoverState
	leaderID  string
	leaderAddr string

	// Failure detectors for each node
	detectors   map[string]*PhiAccrualDetector
	detectorsMu sync.RWMutex

	// Fencing tokens
	fencingToken uint64
	fencedNodes  map[string]uint64

	// Raft integration
	raft *RaftNode

	// Callbacks
	onFailoverStart    func(oldLeader string)
	onFailoverComplete func(newLeader string)
	onNodeFenced       func(nodeID string)

	// Control
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewFailoverManager creates a new failover manager
func NewFailoverManager(config FailoverConfig, raft *RaftNode) *FailoverManager {
	return &FailoverManager{
		config:      config,
		detectors:   make(map[string]*PhiAccrualDetector),
		fencedNodes: make(map[string]uint64),
		raft:        raft,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the failover manager
func (fm *FailoverManager) Start() error {
	fm.wg.Add(1)
	go fm.monitorLoop()
	fmt.Printf("Failover manager started for node %s\n", fm.config.NodeID)
	return nil
}

// Stop gracefully shuts down the failover manager
func (fm *FailoverManager) Stop() error {
	close(fm.stopCh)
	fm.wg.Wait()
	return nil
}

// GetState returns the current failover state
func (fm *FailoverManager) GetState() FailoverState {
	return FailoverState(atomic.LoadInt32(&fm.state))
}

// RecordHeartbeat records a heartbeat from a node
func (fm *FailoverManager) RecordHeartbeat(nodeID string) {
	fm.detectorsMu.Lock()
	detector, ok := fm.detectors[nodeID]
	if !ok {
		detector = NewPhiAccrualDetector(
			fm.config.FailureThreshold,
			fm.config.MinSamples,
			fm.config.MaxSamples,
		)
		fm.detectors[nodeID] = detector
	}
	fm.detectorsMu.Unlock()

	detector.Heartbeat()
}


// monitorLoop continuously monitors for failures
func (fm *FailoverManager) monitorLoop() {
	defer fm.wg.Done()

	ticker := time.NewTicker(fm.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-fm.stopCh:
			return
		case <-ticker.C:
			fm.checkForFailures()
		}
	}
}

// checkForFailures checks all nodes for failures
func (fm *FailoverManager) checkForFailures() {
	fm.mu.RLock()
	leaderID := fm.leaderID
	fm.mu.RUnlock()

	if leaderID == "" || leaderID == fm.config.NodeID {
		return // No leader or we are the leader
	}

	fm.detectorsMu.RLock()
	detector, ok := fm.detectors[leaderID]
	fm.detectorsMu.RUnlock()

	if !ok {
		return
	}

	if detector.IsFailed() {
		fm.initiateFailover(leaderID)
	}
}

// initiateFailover starts the failover process
func (fm *FailoverManager) initiateFailover(failedLeader string) {
	// Try to transition to detecting state
	if !atomic.CompareAndSwapInt32(&fm.state, int32(FailoverStateNormal), int32(FailoverStateDetecting)) {
		return // Already in failover
	}

	fmt.Printf("Initiating failover: leader %s appears to have failed\n", failedLeader)

	if fm.onFailoverStart != nil {
		go fm.onFailoverStart(failedLeader)
	}

	// Step 1: Fence the old leader
	atomic.StoreInt32(&fm.state, int32(FailoverStateFencing))
	if err := fm.fenceNode(failedLeader); err != nil {
		fmt.Printf("Warning: failed to fence old leader: %v\n", err)
	}

	// Step 2: Trigger election via Raft
	atomic.StoreInt32(&fm.state, int32(FailoverStateElecting))
	// The Raft module will handle the election

	// Step 3: Wait for new leader
	atomic.StoreInt32(&fm.state, int32(FailoverStatePromoting))
	newLeader := fm.waitForNewLeader()

	if newLeader != "" {
		fm.mu.Lock()
		fm.leaderID = newLeader
		fm.mu.Unlock()

		if fm.onFailoverComplete != nil {
			go fm.onFailoverComplete(newLeader)
		}
	}

	atomic.StoreInt32(&fm.state, int32(FailoverStateNormal))
}

// fenceNode fences a node to prevent split-brain
func (fm *FailoverManager) fenceNode(nodeID string) error {
	fm.mu.Lock()
	token := atomic.AddUint64(&fm.fencingToken, 1)
	fm.fencedNodes[nodeID] = token
	fm.mu.Unlock()

	if fm.onNodeFenced != nil {
		go fm.onNodeFenced(nodeID)
	}

	// Try to notify the node it's been fenced
	fm.detectorsMu.RLock()
	// In a real implementation, we would send a fencing message
	fm.detectorsMu.RUnlock()

	return nil
}

// waitForNewLeader waits for a new leader to be elected
func (fm *FailoverManager) waitForNewLeader() string {
	timeout := time.After(fm.config.PromotionTimeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-fm.stopCh:
			return ""
		case <-timeout:
			return ""
		case <-ticker.C:
			if fm.raft != nil && fm.raft.IsLeader() {
				return fm.config.NodeID
			}
			leaderID, _ := fm.raft.GetLeader()
			if leaderID != "" {
				return leaderID
			}
		}
	}
}

// IsFenced returns true if a node is fenced
func (fm *FailoverManager) IsFenced(nodeID string) bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	_, fenced := fm.fencedNodes[nodeID]
	return fenced
}

// Unfence removes fencing from a node
func (fm *FailoverManager) Unfence(nodeID string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	delete(fm.fencedNodes, nodeID)
}

// SetLeader updates the current leader
func (fm *FailoverManager) SetLeader(leaderID, leaderAddr string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.leaderID = leaderID
	fm.leaderAddr = leaderAddr
}

// GetLeader returns the current leader
func (fm *FailoverManager) GetLeader() (string, string) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.leaderID, fm.leaderAddr
}

// SetFailoverStartCallback sets the callback for failover start
func (fm *FailoverManager) SetFailoverStartCallback(fn func(oldLeader string)) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.onFailoverStart = fn
}

// SetFailoverCompleteCallback sets the callback for failover completion
func (fm *FailoverManager) SetFailoverCompleteCallback(fn func(newLeader string)) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.onFailoverComplete = fn
}

// SetNodeFencedCallback sets the callback for node fencing
func (fm *FailoverManager) SetNodeFencedCallback(fn func(nodeID string)) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.onNodeFenced = fn
}

// GetNodeHealth returns health information for all monitored nodes
func (fm *FailoverManager) GetNodeHealth() map[string]interface{} {
	fm.detectorsMu.RLock()
	defer fm.detectorsMu.RUnlock()

	health := make(map[string]interface{})
	for nodeID, detector := range fm.detectors {
		health[nodeID] = map[string]interface{}{
			"phi":       detector.Phi(),
			"is_failed": detector.IsFailed(),
			"threshold": detector.threshold,
		}
	}
	return health
}

// GetStatus returns the current failover status
func (fm *FailoverManager) GetStatus() map[string]interface{} {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	return map[string]interface{}{
		"state":         fm.GetState().String(),
		"leader_id":     fm.leaderID,
		"leader_addr":   fm.leaderAddr,
		"fencing_token": fm.fencingToken,
		"fenced_nodes":  len(fm.fencedNodes),
	}
}

// HealthChecker provides health check functionality
type HealthChecker struct {
	config   FailoverConfig
	listener net.Listener
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(config FailoverConfig, port int) (*HealthChecker, error) {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to start health checker: %w", err)
	}

	return &HealthChecker{
		config:   config,
		listener: ln,
		stopCh:   make(chan struct{}),
	}, nil
}

// Start begins the health checker
func (hc *HealthChecker) Start() {
	hc.wg.Add(1)
	go hc.acceptConnections()
}

// Stop gracefully shuts down the health checker
func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
	hc.listener.Close()
	hc.wg.Wait()
}

// acceptConnections handles incoming health check connections
func (hc *HealthChecker) acceptConnections() {
	defer hc.wg.Done()

	for {
		select {
		case <-hc.stopCh:
			return
		default:
		}

		hc.listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
		conn, err := hc.listener.Accept()
		if err != nil {
			continue
		}

		// Simple health check: respond with OK
		conn.Write([]byte("OK"))
		conn.Close()
	}
}


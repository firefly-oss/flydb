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
Package cluster provides advanced cluster membership management for FlyDB.

Cluster Membership Overview:
============================

This module implements dynamic cluster membership with:
- Automatic node discovery via gossip protocol
- Health monitoring with configurable probes
- Graceful node addition and removal
- Cluster state synchronization

Node Discovery:
===============

Nodes discover each other through:
1. Seed nodes: Initial known nodes to bootstrap
2. Gossip: Nodes share membership information
3. DNS: Optional DNS-based discovery

Health Monitoring:
==================

Each node is monitored via:
- TCP health checks
- Application-level probes
- Resource utilization metrics

Membership Changes:
===================

Changes are coordinated through Raft:
1. Propose membership change
2. Wait for commit
3. Apply change to local state
4. Notify other components
*/
package cluster

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// MemberState represents the state of a cluster member (distinct from NodeState in unified.go)
type MemberState int32

const (
	MemberStateUnknown MemberState = iota
	MemberStateJoining
	MemberStateActive
	MemberStateLeaving
	MemberStateDead
)

func (s MemberState) String() string {
	switch s {
	case MemberStateUnknown:
		return "UNKNOWN"
	case MemberStateJoining:
		return "JOINING"
	case MemberStateActive:
		return "ACTIVE"
	case MemberStateLeaving:
		return "LEAVING"
	case MemberStateDead:
		return "DEAD"
	default:
		return "UNKNOWN"
	}
}

// MemberInfo contains information about a cluster member
type MemberInfo struct {
	ID          string            `json:"id"`
	Addr        string            `json:"addr"`
	ClusterPort int               `json:"cluster_port"`
	DataPort    int               `json:"data_port"`
	State       MemberState       `json:"state"`
	JoinedAt    time.Time         `json:"joined_at"`
	LastSeen    time.Time         `json:"last_seen"`
	Metadata    map[string]string `json:"metadata"`
	Version     string            `json:"version"`
	Partitions  []int             `json:"partitions"`
}

// MembershipConfig holds configuration for the membership manager
type MembershipConfig struct {
	NodeID           string        `json:"node_id"`
	NodeAddr         string        `json:"node_addr"`
	ClusterPort      int           `json:"cluster_port"`
	DataPort         int           `json:"data_port"`
	SeedNodes        []string      `json:"seed_nodes"`
	GossipInterval   time.Duration `json:"gossip_interval"`
	ProbeInterval    time.Duration `json:"probe_interval"`
	ProbeTimeout     time.Duration `json:"probe_timeout"`
	SuspicionTimeout time.Duration `json:"suspicion_timeout"`
	DeadTimeout      time.Duration `json:"dead_timeout"`
}

// DefaultMembershipConfig returns sensible defaults
func DefaultMembershipConfig(nodeID, nodeAddr string) MembershipConfig {
	return MembershipConfig{
		NodeID:           nodeID,
		NodeAddr:         nodeAddr,
		ClusterPort:      9996,
		DataPort:         9999,
		SeedNodes:        []string{},
		GossipInterval:   200 * time.Millisecond,
		ProbeInterval:    1 * time.Second,
		ProbeTimeout:     500 * time.Millisecond,
		SuspicionTimeout: 5 * time.Second,
		DeadTimeout:      30 * time.Second,
	}
}

// GossipMessage represents a gossip protocol message
type GossipMessage struct {
	Type      GossipMessageType `json:"type"`
	SenderID  string            `json:"sender_id"`
	Members   []*MemberInfo     `json:"members,omitempty"`
	Timestamp int64             `json:"timestamp"`
}

// GossipMessageType represents the type of gossip message
type GossipMessageType int

const (
	GossipPing GossipMessageType = iota
	GossipPingReq
	GossipAck
	GossipSync
	GossipJoin
	GossipLeave
)

// MembershipManager manages cluster membership
type MembershipManager struct {
	config MembershipConfig
	mu     sync.RWMutex

	// Local node info
	localNode *MemberInfo

	// All known members
	members   map[string]*MemberInfo
	membersMu sync.RWMutex

	// Suspicion tracking
	suspicions   map[string]time.Time
	suspicionsMu sync.RWMutex

	// Network
	listener net.Listener
	stopCh   chan struct{}
	wg       sync.WaitGroup

	// Raft integration
	raft *RaftNode

	// Callbacks
	onNodeJoin  func(node *MemberInfo)
	onNodeLeave func(node *MemberInfo)
	onNodeDead  func(node *MemberInfo)

	// Sequence number for gossip
	seqNum uint64
}

// NewMembershipManager creates a new membership manager
func NewMembershipManager(config MembershipConfig, raft *RaftNode) *MembershipManager {
	localNode := &MemberInfo{
		ID:          config.NodeID,
		Addr:        config.NodeAddr,
		ClusterPort: config.ClusterPort,
		DataPort:    config.DataPort,
		State:       MemberStateJoining,
		JoinedAt:    time.Now(),
		LastSeen:    time.Now(),
		Metadata:    make(map[string]string),
		Partitions:  []int{},
	}

	return &MembershipManager{
		config:     config,
		localNode:  localNode,
		members:    make(map[string]*MemberInfo),
		suspicions: make(map[string]time.Time),
		raft:       raft,
		stopCh:     make(chan struct{}),
	}
}

// Start begins the membership manager
func (mm *MembershipManager) Start() error {
	addr := fmt.Sprintf(":%d", mm.config.ClusterPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start membership manager: %w", err)
	}
	mm.listener = ln

	// Add self to members
	mm.membersMu.Lock()
	mm.members[mm.config.NodeID] = mm.localNode
	mm.membersMu.Unlock()

	// Start background goroutines
	mm.wg.Add(3)
	go mm.acceptConnections()
	go mm.gossipLoop()
	go mm.probeLoop()

	// Join cluster via seed nodes
	go mm.joinCluster()

	fmt.Printf("Membership manager started on %s\n", addr)
	return nil
}

// Stop gracefully shuts down the membership manager
func (mm *MembershipManager) Stop() error {
	// Announce leaving
	mm.announceLeave()

	close(mm.stopCh)
	if mm.listener != nil {
		mm.listener.Close()
	}
	mm.wg.Wait()
	return nil
}

// joinCluster attempts to join the cluster via seed nodes
func (mm *MembershipManager) joinCluster() {
	for _, seed := range mm.config.SeedNodes {
		if seed == mm.config.NodeAddr {
			continue // Skip self
		}

		if err := mm.sendJoin(seed); err != nil {
			fmt.Printf("Failed to join via seed %s: %v\n", seed, err)
			continue
		}

		// Successfully joined
		mm.localNode.State = MemberStateActive
		return
	}

	// No seed nodes or all failed, we're the first node
	mm.localNode.State = MemberStateActive
}

// sendJoin sends a join request to a seed node
func (mm *MembershipManager) sendJoin(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	msg := GossipMessage{
		Type:      GossipJoin,
		SenderID:  mm.config.NodeID,
		Members:   []*MemberInfo{mm.localNode},
		Timestamp: time.Now().UnixNano(),
	}

	return mm.sendGossipMessage(conn, &msg)
}

// announceLeave announces that this node is leaving
func (mm *MembershipManager) announceLeave() {
	mm.localNode.State = MemberStateLeaving

	mm.membersMu.RLock()
	members := make([]*MemberInfo, 0, len(mm.members))
	for _, m := range mm.members {
		if m.ID != mm.config.NodeID {
			members = append(members, m)
		}
	}
	mm.membersMu.RUnlock()

	msg := GossipMessage{
		Type:      GossipLeave,
		SenderID:  mm.config.NodeID,
		Members:   []*MemberInfo{mm.localNode},
		Timestamp: time.Now().UnixNano(),
	}

	for _, m := range members {
		go func(node *MemberInfo) {
			addr := net.JoinHostPort(node.Addr, fmt.Sprint(node.ClusterPort))
			conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
			if err != nil {
				return
			}
			defer conn.Close()
			mm.sendGossipMessage(conn, &msg)
		}(m)
	}
}

// acceptConnections handles incoming gossip connections
func (mm *MembershipManager) acceptConnections() {
	defer mm.wg.Done()

	for {
		select {
		case <-mm.stopCh:
			return
		default:
		}

		mm.listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
		conn, err := mm.listener.Accept()
		if err != nil {
			continue
		}

		go mm.handleConnection(conn)
	}
}

// handleConnection handles an incoming gossip connection
func (mm *MembershipManager) handleConnection(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	msg, err := mm.readGossipMessage(conn)
	if err != nil {
		return
	}

	switch msg.Type {
	case GossipPing:
		mm.handlePing(conn, msg)
	case GossipSync:
		mm.handleSync(conn, msg)
	case GossipJoin:
		mm.handleJoin(conn, msg)
	case GossipLeave:
		mm.handleLeave(msg)
	}
}

// handlePing handles a ping message
func (mm *MembershipManager) handlePing(conn net.Conn, msg *GossipMessage) {
	// Update sender's last seen
	mm.updateMember(msg.SenderID, func(node *MemberInfo) {
		node.LastSeen = time.Now()
	})

	// Send ack
	ack := GossipMessage{
		Type:      GossipAck,
		SenderID:  mm.config.NodeID,
		Timestamp: time.Now().UnixNano(),
	}
	mm.sendGossipMessage(conn, &ack)
}

// handleSync handles a sync message
func (mm *MembershipManager) handleSync(conn net.Conn, msg *GossipMessage) {
	// Merge received members
	for _, node := range msg.Members {
		mm.mergeMember(node)
	}

	// Send our members back
	mm.membersMu.RLock()
	members := make([]*MemberInfo, 0, len(mm.members))
	for _, m := range mm.members {
		members = append(members, m)
	}
	mm.membersMu.RUnlock()

	reply := GossipMessage{
		Type:      GossipSync,
		SenderID:  mm.config.NodeID,
		Members:   members,
		Timestamp: time.Now().UnixNano(),
	}
	mm.sendGossipMessage(conn, &reply)
}

// handleJoin handles a join message
func (mm *MembershipManager) handleJoin(conn net.Conn, msg *GossipMessage) {
	for _, node := range msg.Members {
		node.State = MemberStateActive
		node.JoinedAt = time.Now()
		node.LastSeen = time.Now()
		mm.addMember(node)
	}

	// Send current membership
	mm.membersMu.RLock()
	members := make([]*MemberInfo, 0, len(mm.members))
	for _, m := range mm.members {
		members = append(members, m)
	}
	mm.membersMu.RUnlock()

	reply := GossipMessage{
		Type:      GossipSync,
		SenderID:  mm.config.NodeID,
		Members:   members,
		Timestamp: time.Now().UnixNano(),
	}
	mm.sendGossipMessage(conn, &reply)
}

// handleLeave handles a leave message
func (mm *MembershipManager) handleLeave(msg *GossipMessage) {
	for _, node := range msg.Members {
		mm.removeMember(node.ID)
	}
}

// gossipLoop periodically gossips with random members
func (mm *MembershipManager) gossipLoop() {
	defer mm.wg.Done()

	ticker := time.NewTicker(mm.config.GossipInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mm.stopCh:
			return
		case <-ticker.C:
			mm.gossipRound()
		}
	}
}

// gossipRound performs one round of gossip
func (mm *MembershipManager) gossipRound() {
	// Select random member to gossip with
	target := mm.selectRandomMember()
	if target == nil {
		return
	}

	addr := net.JoinHostPort(target.Addr, fmt.Sprint(target.ClusterPort))
	conn, err := net.DialTimeout("tcp", addr, mm.config.ProbeTimeout)
	if err != nil {
		mm.markSuspect(target.ID)
		return
	}
	defer conn.Close()

	// Send sync message
	mm.membersMu.RLock()
	members := make([]*MemberInfo, 0, len(mm.members))
	for _, m := range mm.members {
		members = append(members, m)
	}
	mm.membersMu.RUnlock()

	msg := GossipMessage{
		Type:      GossipSync,
		SenderID:  mm.config.NodeID,
		Members:   members,
		Timestamp: time.Now().UnixNano(),
	}

	if err := mm.sendGossipMessage(conn, &msg); err != nil {
		mm.markSuspect(target.ID)
		return
	}

	// Read response
	reply, err := mm.readGossipMessage(conn)
	if err != nil {
		mm.markSuspect(target.ID)
		return
	}

	// Merge received members
	for _, node := range reply.Members {
		mm.mergeMember(node)
	}

	// Clear suspicion
	mm.clearSuspicion(target.ID)
}

// probeLoop periodically probes members for health
func (mm *MembershipManager) probeLoop() {
	defer mm.wg.Done()

	ticker := time.NewTicker(mm.config.ProbeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mm.stopCh:
			return
		case <-ticker.C:
			mm.probeMembers()
		}
	}
}

// probeMembers probes all members for health
func (mm *MembershipManager) probeMembers() {
	mm.membersMu.RLock()
	members := make([]*MemberInfo, 0, len(mm.members))
	for _, m := range mm.members {
		if m.ID != mm.config.NodeID {
			members = append(members, m)
		}
	}
	mm.membersMu.RUnlock()

	for _, m := range members {
		go mm.probeMember(m)
	}

	// Check for dead members
	mm.checkDeadMembers()
}

// probeMember probes a single member
func (mm *MembershipManager) probeMember(node *MemberInfo) {
	addr := net.JoinHostPort(node.Addr, fmt.Sprint(node.ClusterPort))
	conn, err := net.DialTimeout("tcp", addr, mm.config.ProbeTimeout)
	if err != nil {
		mm.markSuspect(node.ID)
		return
	}
	defer conn.Close()

	msg := GossipMessage{
		Type:      GossipPing,
		SenderID:  mm.config.NodeID,
		Timestamp: time.Now().UnixNano(),
	}

	if err := mm.sendGossipMessage(conn, &msg); err != nil {
		mm.markSuspect(node.ID)
		return
	}

	// Wait for ack
	conn.SetReadDeadline(time.Now().Add(mm.config.ProbeTimeout))
	_, err = mm.readGossipMessage(conn)
	if err != nil {
		mm.markSuspect(node.ID)
		return
	}

	mm.clearSuspicion(node.ID)
	mm.updateMember(node.ID, func(n *MemberInfo) {
		n.LastSeen = time.Now()
	})
}

// checkDeadMembers checks for members that should be marked dead
func (mm *MembershipManager) checkDeadMembers() {
	mm.suspicionsMu.RLock()
	suspects := make(map[string]time.Time)
	for id, t := range mm.suspicions {
		suspects[id] = t
	}
	mm.suspicionsMu.RUnlock()

	for id, suspectTime := range suspects {
		if time.Since(suspectTime) > mm.config.DeadTimeout {
			mm.markDead(id)
		}
	}
}

// selectRandomMember selects a random member for gossip
func (mm *MembershipManager) selectRandomMember() *MemberInfo {
	mm.membersMu.RLock()
	defer mm.membersMu.RUnlock()

	candidates := make([]*MemberInfo, 0)
	for _, m := range mm.members {
		if m.ID != mm.config.NodeID && m.State == MemberStateActive {
			candidates = append(candidates, m)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Simple random selection using sequence number
	idx := atomic.AddUint64(&mm.seqNum, 1) % uint64(len(candidates))
	return candidates[idx]
}

// addMember adds a new member
func (mm *MembershipManager) addMember(node *MemberInfo) {
	mm.membersMu.Lock()
	defer mm.membersMu.Unlock()

	if _, exists := mm.members[node.ID]; !exists {
		mm.members[node.ID] = node
		if mm.onNodeJoin != nil {
			go mm.onNodeJoin(node)
		}
	}
}

// removeMember removes a member
func (mm *MembershipManager) removeMember(nodeID string) {
	mm.membersMu.Lock()
	node, exists := mm.members[nodeID]
	if exists {
		delete(mm.members, nodeID)
	}
	mm.membersMu.Unlock()

	if exists && mm.onNodeLeave != nil {
		go mm.onNodeLeave(node)
	}
}

// updateMember updates a member's info
func (mm *MembershipManager) updateMember(nodeID string, fn func(*MemberInfo)) {
	mm.membersMu.Lock()
	defer mm.membersMu.Unlock()

	if node, exists := mm.members[nodeID]; exists {
		fn(node)
	}
}

// mergeMember merges a received member into our list
func (mm *MembershipManager) mergeMember(node *MemberInfo) {
	mm.membersMu.Lock()
	defer mm.membersMu.Unlock()

	existing, exists := mm.members[node.ID]
	if !exists {
		mm.members[node.ID] = node
		if mm.onNodeJoin != nil {
			go mm.onNodeJoin(node)
		}
		return
	}

	// Update if newer
	if node.LastSeen.After(existing.LastSeen) {
		existing.LastSeen = node.LastSeen
		existing.State = node.State
		existing.Metadata = node.Metadata
	}
}

// markSuspect marks a node as suspect
func (mm *MembershipManager) markSuspect(nodeID string) {
	mm.suspicionsMu.Lock()
	defer mm.suspicionsMu.Unlock()

	if _, exists := mm.suspicions[nodeID]; !exists {
		mm.suspicions[nodeID] = time.Now()
	}
}

// clearSuspicion clears suspicion for a node
func (mm *MembershipManager) clearSuspicion(nodeID string) {
	mm.suspicionsMu.Lock()
	defer mm.suspicionsMu.Unlock()
	delete(mm.suspicions, nodeID)
}

// markDead marks a node as dead
func (mm *MembershipManager) markDead(nodeID string) {
	mm.membersMu.Lock()
	node, exists := mm.members[nodeID]
	if exists {
		node.State = MemberStateDead
	}
	mm.membersMu.Unlock()

	mm.suspicionsMu.Lock()
	delete(mm.suspicions, nodeID)
	mm.suspicionsMu.Unlock()

	if exists && mm.onNodeDead != nil {
		go mm.onNodeDead(node)
	}
}

// sendGossipMessage sends a gossip message
func (mm *MembershipManager) sendGossipMessage(conn net.Conn, msg *GossipMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))
	if _, err := conn.Write(lenBuf); err != nil {
		return err
	}
	_, err = conn.Write(data)
	return err
}

// readGossipMessage reads a gossip message
func (mm *MembershipManager) readGossipMessage(conn net.Conn) (*GossipMessage, error) {
	lenBuf := make([]byte, 4)
	if _, err := conn.Read(lenBuf); err != nil {
		return nil, err
	}
	msgLen := binary.BigEndian.Uint32(lenBuf)

	data := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, err
	}

	var msg GossipMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetMembers returns all known members
func (mm *MembershipManager) GetMembers() []*MemberInfo {
	mm.membersMu.RLock()
	defer mm.membersMu.RUnlock()

	members := make([]*MemberInfo, 0, len(mm.members))
	for _, m := range mm.members {
		members = append(members, m)
	}
	return members
}

// GetMember returns a specific member
func (mm *MembershipManager) GetMember(nodeID string) *MemberInfo {
	mm.membersMu.RLock()
	defer mm.membersMu.RUnlock()
	return mm.members[nodeID]
}

// GetActiveMembers returns all active members
func (mm *MembershipManager) GetActiveMembers() []*MemberInfo {
	mm.membersMu.RLock()
	defer mm.membersMu.RUnlock()

	members := make([]*MemberInfo, 0)
	for _, m := range mm.members {
		if m.State == MemberStateActive {
			members = append(members, m)
		}
	}
	return members
}

// SetNodeJoinCallback sets the callback for node joins
func (mm *MembershipManager) SetNodeJoinCallback(fn func(node *MemberInfo)) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.onNodeJoin = fn
}

// SetNodeLeaveCallback sets the callback for node leaves
func (mm *MembershipManager) SetNodeLeaveCallback(fn func(node *MemberInfo)) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.onNodeLeave = fn
}

// SetNodeDeadCallback sets the callback for dead nodes
func (mm *MembershipManager) SetNodeDeadCallback(fn func(node *MemberInfo)) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.onNodeDead = fn
}

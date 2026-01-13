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
Package cluster provides Raft consensus implementation for FlyDB.

Raft Consensus Overview:
========================

Raft is a consensus algorithm designed to be understandable and practical.
It provides strong consistency guarantees and handles network partitions gracefully.

Key Properties:
- Leader Election: Only one leader at a time, elected by majority vote
- Log Replication: Leader replicates log entries to followers
- Safety: Committed entries are never lost
- Availability: System remains available as long as majority is alive

State Machine:
==============

Each node can be in one of three states:
- Follower: Passive, responds to leader/candidate requests
- Candidate: Actively seeking votes to become leader
- Leader: Handles all client requests, replicates to followers

Term-Based Leadership:
======================

Time is divided into terms (monotonically increasing integers).
Each term has at most one leader. Terms act as logical clocks.
*/
package cluster

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// RaftState represents the state of a Raft node
type RaftState int32

const (
	StateFollower RaftState = iota
	StateCandidate
	StateLeader
)

func (s RaftState) String() string {
	switch s {
	case StateFollower:
		return "FOLLOWER"
	case StateCandidate:
		return "CANDIDATE"
	case StateLeader:
		return "LEADER"
	default:
		return "UNKNOWN"
	}
}

// RaftConfig holds configuration for the Raft consensus module
type RaftConfig struct {
	NodeID            string        `json:"node_id"`
	NodeAddr          string        `json:"node_addr"`
	ClusterPort       int           `json:"cluster_port"`
	Peers             []string      `json:"peers"`
	ElectionTimeout   time.Duration `json:"election_timeout"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	EnablePreVote     bool          `json:"enable_pre_vote"`
	DataDir           string        `json:"data_dir"`
}

// DefaultRaftConfig returns a RaftConfig with sensible defaults
func DefaultRaftConfig(nodeID, nodeAddr string) RaftConfig {
	return RaftConfig{
		NodeID:            nodeID,
		NodeAddr:          nodeAddr,
		ClusterPort:       9998,
		Peers:             []string{},
		ElectionTimeout:   1000 * time.Millisecond,
		HeartbeatInterval: 150 * time.Millisecond,
		EnablePreVote:     true,
		DataDir:           "./data/raft",
	}
}

// LogEntry represents a single entry in the Raft log
type LogEntry struct {
	Term    uint64 `json:"term"`
	Index   uint64 `json:"index"`
	Command []byte `json:"command"`
	Type    LogEntryType `json:"type"`
}

// LogEntryType represents the type of log entry
type LogEntryType int

const (
	LogEntryCommand LogEntryType = iota
	LogEntryConfig
	LogEntryNoop
)

// RaftNode implements the Raft consensus algorithm
type RaftNode struct {
	config RaftConfig
	mu     sync.RWMutex

	// Persistent state (saved to stable storage)
	currentTerm uint64
	votedFor    string
	log         []LogEntry

	// Volatile state on all servers
	commitIndex uint64
	lastApplied uint64
	state       int32 // atomic RaftState

	// Volatile state on leaders
	nextIndex  map[string]uint64
	matchIndex map[string]uint64

	// Cluster membership
	peers   map[string]*RaftPeer
	peersMu sync.RWMutex

	// Channels
	applyCh     chan LogEntry
	stopCh      chan struct{}
	heartbeatCh chan struct{}
	voteCh      chan *VoteResult

	// Network
	listener net.Listener
	wg       sync.WaitGroup

	// Leader info
	leaderID   string
	leaderAddr string
	leaderMu   sync.RWMutex

	// Callbacks
	onBecomeLeader   func()
	onBecomeFollower func(leaderID string)
}

// RaftPeer represents a peer node in the Raft cluster
type RaftPeer struct {
	ID            string
	Addr          string
	conn          net.Conn
	mu            sync.Mutex
	lastContact   time.Time
	isHealthy     bool
	failedAttempts int
}

// VoteResult represents the result of a vote request
type VoteResult struct {
	Term        uint64
	VoteGranted bool
	VoterID     string
}

// Raft RPC message types
const (
	RaftMsgRequestVote     byte = 0x10
	RaftMsgRequestVoteResp byte = 0x11
	RaftMsgAppendEntries   byte = 0x12
	RaftMsgAppendEntriesResp byte = 0x13
	RaftMsgPreVote         byte = 0x14
	RaftMsgPreVoteResp     byte = 0x15
	RaftMsgInstallSnapshot byte = 0x16
)

// RequestVoteArgs contains arguments for RequestVote RPC
type RequestVoteArgs struct {
	Term         uint64 `json:"term"`
	CandidateID  string `json:"candidate_id"`
	LastLogIndex uint64 `json:"last_log_index"`
	LastLogTerm  uint64 `json:"last_log_term"`
	PreVote      bool   `json:"pre_vote"`
}

// RequestVoteReply contains the reply for RequestVote RPC
type RequestVoteReply struct {
	Term        uint64 `json:"term"`
	VoteGranted bool   `json:"vote_granted"`
}

// AppendEntriesArgs contains arguments for AppendEntries RPC
type AppendEntriesArgs struct {
	Term         uint64     `json:"term"`
	LeaderID     string     `json:"leader_id"`
	PrevLogIndex uint64     `json:"prev_log_index"`
	PrevLogTerm  uint64     `json:"prev_log_term"`
	Entries      []LogEntry `json:"entries"`
	LeaderCommit uint64     `json:"leader_commit"`
}

// AppendEntriesReply contains the reply for AppendEntries RPC
type AppendEntriesReply struct {
	Term          uint64 `json:"term"`
	Success       bool   `json:"success"`
	ConflictIndex uint64 `json:"conflict_index"`
	ConflictTerm  uint64 `json:"conflict_term"`
}

// NewRaftNode creates a new Raft consensus node
func NewRaftNode(config RaftConfig, applyCh chan LogEntry) *RaftNode {
	rn := &RaftNode{
		config:      config,
		currentTerm: 0,
		votedFor:    "",
		log:         make([]LogEntry, 0),
		commitIndex: 0,
		lastApplied: 0,
		nextIndex:   make(map[string]uint64),
		matchIndex:  make(map[string]uint64),
		peers:       make(map[string]*RaftPeer),
		applyCh:     applyCh,
		stopCh:      make(chan struct{}),
		heartbeatCh: make(chan struct{}, 1),
		voteCh:      make(chan *VoteResult, 100),
	}

	// Initialize as follower
	atomic.StoreInt32(&rn.state, int32(StateFollower))

	// Add initial peers
	for _, peerAddr := range config.Peers {
		rn.peers[peerAddr] = &RaftPeer{
			Addr:      peerAddr,
			isHealthy: true,
		}
	}

	// Append initial no-op entry to simplify log handling
	rn.log = append(rn.log, LogEntry{
		Term:  0,
		Index: 0,
		Type:  LogEntryNoop,
	})

	return rn
}

// Start begins the Raft consensus protocol
func (rn *RaftNode) Start() error {
	// Start network listener
	addr := fmt.Sprintf(":%d", rn.config.ClusterPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start Raft listener: %w", err)
	}
	rn.listener = ln

	fmt.Printf("Raft node %s started on %s\n", rn.config.NodeID, addr)

	// Start background goroutines
	rn.wg.Add(3)
	go rn.acceptConnections()
	go rn.runElectionTimer()
	go rn.applyCommittedEntries()

	return nil
}

// Stop gracefully shuts down the Raft node
func (rn *RaftNode) Stop() error {
	close(rn.stopCh)
	if rn.listener != nil {
		rn.listener.Close()
	}
	rn.wg.Wait()
	return nil
}

// GetState returns the current state of the Raft node
func (rn *RaftNode) GetState() RaftState {
	return RaftState(atomic.LoadInt32(&rn.state))
}

// GetTerm returns the current term
func (rn *RaftNode) GetTerm() uint64 {
	rn.mu.RLock()
	defer rn.mu.RUnlock()
	return rn.currentTerm
}

// IsLeader returns true if this node is the leader
func (rn *RaftNode) IsLeader() bool {
	return rn.GetState() == StateLeader
}

// GetLeader returns the current leader's ID and address
func (rn *RaftNode) GetLeader() (string, string) {
	rn.leaderMu.RLock()
	defer rn.leaderMu.RUnlock()
	return rn.leaderID, rn.leaderAddr
}

// SetLeaderCallback sets the callback for when this node becomes leader
func (rn *RaftNode) SetLeaderCallback(fn func()) {
	rn.mu.Lock()
	defer rn.mu.Unlock()
	rn.onBecomeLeader = fn
}

// SetFollowerCallback sets the callback for when this node becomes follower
func (rn *RaftNode) SetFollowerCallback(fn func(leaderID string)) {
	rn.mu.Lock()
	defer rn.mu.Unlock()
	rn.onBecomeFollower = fn
}

// acceptConnections handles incoming Raft RPC connections
func (rn *RaftNode) acceptConnections() {
	defer rn.wg.Done()

	for {
		select {
		case <-rn.stopCh:
			return
		default:
		}

		rn.listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
		conn, err := rn.listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			continue
		}

		go rn.handleConnection(conn)
	}
}

// handleConnection processes incoming Raft RPC messages
func (rn *RaftNode) handleConnection(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Read message type
	msgType := make([]byte, 1)
	if _, err := conn.Read(msgType); err != nil {
		return
	}

	switch msgType[0] {
	case RaftMsgRequestVote, RaftMsgPreVote:
		rn.handleRequestVote(conn, msgType[0] == RaftMsgPreVote)
	case RaftMsgAppendEntries:
		rn.handleAppendEntries(conn)
	}
}

// runElectionTimer runs the election timeout loop
func (rn *RaftNode) runElectionTimer() {
	defer rn.wg.Done()

	for {
		select {
		case <-rn.stopCh:
			return
		default:
		}

		state := rn.GetState()
		timeout := rn.randomElectionTimeout()

		select {
		case <-rn.stopCh:
			return
		case <-rn.heartbeatCh:
			// Received heartbeat, reset timer
			continue
		case <-time.After(timeout):
			if state != StateLeader {
				rn.startElection()
			}
		}
	}
}

// randomElectionTimeout returns a randomized election timeout
func (rn *RaftNode) randomElectionTimeout() time.Duration {
	base := rn.config.ElectionTimeout
	jitter := time.Duration(rand.Int63n(int64(base)))
	return base + jitter
}

// startElection initiates a leader election
func (rn *RaftNode) startElection() {
	rn.mu.Lock()

	// If pre-vote is enabled, do pre-vote first
	if rn.config.EnablePreVote && rn.GetState() == StateFollower {
		rn.mu.Unlock()
		if !rn.conductPreVote() {
			return // Pre-vote failed, don't start real election
		}
		rn.mu.Lock()
	}

	// Transition to candidate
	atomic.StoreInt32(&rn.state, int32(StateCandidate))
	rn.currentTerm++
	rn.votedFor = rn.config.NodeID
	currentTerm := rn.currentTerm
	lastLogIndex := uint64(len(rn.log) - 1)
	lastLogTerm := rn.log[lastLogIndex].Term

	rn.mu.Unlock()

	fmt.Printf("Node %s starting election for term %d\n", rn.config.NodeID, currentTerm)

	// Vote for self
	votesReceived := 1
	votesNeeded := (len(rn.peers) + 1) / 2 + 1

	// Request votes from all peers
	var votesMu sync.Mutex
	var wg sync.WaitGroup

	rn.peersMu.RLock()
	peers := make([]*RaftPeer, 0, len(rn.peers))
	for _, peer := range rn.peers {
		peers = append(peers, peer)
	}
	rn.peersMu.RUnlock()

	for _, peer := range peers {
		wg.Add(1)
		go func(p *RaftPeer) {
			defer wg.Done()
			reply := rn.sendRequestVote(p.Addr, RequestVoteArgs{
				Term:         currentTerm,
				CandidateID:  rn.config.NodeID,
				LastLogIndex: lastLogIndex,
				LastLogTerm:  lastLogTerm,
				PreVote:      false,
			})

			if reply == nil {
				return
			}

			rn.mu.Lock()
			defer rn.mu.Unlock()

			if reply.Term > rn.currentTerm {
				rn.becomeFollower(reply.Term, "")
				return
			}

			if reply.VoteGranted && rn.GetState() == StateCandidate && rn.currentTerm == currentTerm {
				votesMu.Lock()
				votesReceived++
				if votesReceived >= votesNeeded {
					rn.becomeLeader()
				}
				votesMu.Unlock()
			}
		}(peer)
	}

	wg.Wait()
}


// conductPreVote performs a pre-vote phase before starting a real election
func (rn *RaftNode) conductPreVote() bool {
	rn.mu.RLock()
	currentTerm := rn.currentTerm + 1 // Hypothetical next term
	lastLogIndex := uint64(len(rn.log) - 1)
	lastLogTerm := rn.log[lastLogIndex].Term
	rn.mu.RUnlock()

	votesReceived := 1 // Vote for self
	votesNeeded := (len(rn.peers) + 1) / 2 + 1

	var votesMu sync.Mutex
	var wg sync.WaitGroup

	rn.peersMu.RLock()
	peers := make([]*RaftPeer, 0, len(rn.peers))
	for _, peer := range rn.peers {
		peers = append(peers, peer)
	}
	rn.peersMu.RUnlock()

	for _, peer := range peers {
		wg.Add(1)
		go func(p *RaftPeer) {
			defer wg.Done()
			reply := rn.sendRequestVote(p.Addr, RequestVoteArgs{
				Term:         currentTerm,
				CandidateID:  rn.config.NodeID,
				LastLogIndex: lastLogIndex,
				LastLogTerm:  lastLogTerm,
				PreVote:      true,
			})

			if reply != nil && reply.VoteGranted {
				votesMu.Lock()
				votesReceived++
				votesMu.Unlock()
			}
		}(peer)
	}

	wg.Wait()
	return votesReceived >= votesNeeded
}

// becomeFollower transitions the node to follower state
func (rn *RaftNode) becomeFollower(term uint64, leaderID string) {
	atomic.StoreInt32(&rn.state, int32(StateFollower))
	rn.currentTerm = term
	rn.votedFor = ""

	rn.leaderMu.Lock()
	rn.leaderID = leaderID
	rn.leaderMu.Unlock()

	if rn.onBecomeFollower != nil {
		go rn.onBecomeFollower(leaderID)
	}

	fmt.Printf("Node %s became follower for term %d (leader: %s)\n", rn.config.NodeID, term, leaderID)
}

// becomeLeader transitions the node to leader state
func (rn *RaftNode) becomeLeader() {
	atomic.StoreInt32(&rn.state, int32(StateLeader))

	rn.leaderMu.Lock()
	rn.leaderID = rn.config.NodeID
	rn.leaderAddr = rn.config.NodeAddr
	rn.leaderMu.Unlock()

	// Initialize leader state
	lastLogIndex := uint64(len(rn.log))
	for peerAddr := range rn.peers {
		rn.nextIndex[peerAddr] = lastLogIndex
		rn.matchIndex[peerAddr] = 0
	}

	// Append no-op entry to commit entries from previous terms
	rn.log = append(rn.log, LogEntry{
		Term:  rn.currentTerm,
		Index: lastLogIndex,
		Type:  LogEntryNoop,
	})

	if rn.onBecomeLeader != nil {
		go rn.onBecomeLeader()
	}

	fmt.Printf("Node %s became leader for term %d\n", rn.config.NodeID, rn.currentTerm)

	// Start sending heartbeats
	go rn.sendHeartbeats()
}

// sendHeartbeats sends periodic heartbeats to all followers
func (rn *RaftNode) sendHeartbeats() {
	ticker := time.NewTicker(rn.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rn.stopCh:
			return
		case <-ticker.C:
			if rn.GetState() != StateLeader {
				return
			}
			rn.broadcastAppendEntries()
		}
	}
}

// broadcastAppendEntries sends AppendEntries RPCs to all peers
func (rn *RaftNode) broadcastAppendEntries() {
	rn.peersMu.RLock()
	peers := make([]*RaftPeer, 0, len(rn.peers))
	for _, peer := range rn.peers {
		peers = append(peers, peer)
	}
	rn.peersMu.RUnlock()

	for _, peer := range peers {
		go rn.sendAppendEntriesToPeer(peer)
	}
}

// sendAppendEntriesToPeer sends AppendEntries RPC to a specific peer
func (rn *RaftNode) sendAppendEntriesToPeer(peer *RaftPeer) {
	rn.mu.RLock()
	nextIdx := rn.nextIndex[peer.Addr]
	prevLogIndex := nextIdx - 1
	prevLogTerm := uint64(0)
	if prevLogIndex > 0 && prevLogIndex < uint64(len(rn.log)) {
		prevLogTerm = rn.log[prevLogIndex].Term
	}

	entries := make([]LogEntry, 0)
	if nextIdx < uint64(len(rn.log)) {
		entries = rn.log[nextIdx:]
	}

	args := AppendEntriesArgs{
		Term:         rn.currentTerm,
		LeaderID:     rn.config.NodeID,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      entries,
		LeaderCommit: rn.commitIndex,
	}
	rn.mu.RUnlock()

	reply := rn.sendAppendEntries(peer.Addr, args)
	if reply == nil {
		return
	}

	rn.mu.Lock()
	defer rn.mu.Unlock()

	if reply.Term > rn.currentTerm {
		rn.becomeFollower(reply.Term, "")
		return
	}

	if reply.Success {
		rn.nextIndex[peer.Addr] = nextIdx + uint64(len(entries))
		rn.matchIndex[peer.Addr] = rn.nextIndex[peer.Addr] - 1
		rn.updateCommitIndex()
	} else {
		// Decrement nextIndex and retry
		if reply.ConflictIndex > 0 {
			rn.nextIndex[peer.Addr] = reply.ConflictIndex
		} else if rn.nextIndex[peer.Addr] > 1 {
			rn.nextIndex[peer.Addr]--
		}
	}
}


// updateCommitIndex updates the commit index based on matchIndex values
func (rn *RaftNode) updateCommitIndex() {
	// Find the highest index that a majority of servers have replicated
	matchIndexes := make([]uint64, 0, len(rn.peers)+1)
	matchIndexes = append(matchIndexes, uint64(len(rn.log)-1)) // Leader's own index

	for _, idx := range rn.matchIndex {
		matchIndexes = append(matchIndexes, idx)
	}

	// Sort to find median
	for i := 0; i < len(matchIndexes)-1; i++ {
		for j := i + 1; j < len(matchIndexes); j++ {
			if matchIndexes[i] > matchIndexes[j] {
				matchIndexes[i], matchIndexes[j] = matchIndexes[j], matchIndexes[i]
			}
		}
	}

	// Majority index is at position (n-1)/2 for n nodes
	majorityIdx := matchIndexes[(len(matchIndexes)-1)/2]

	// Only update if the entry at majorityIdx is from current term
	if majorityIdx > rn.commitIndex && rn.log[majorityIdx].Term == rn.currentTerm {
		rn.commitIndex = majorityIdx
	}
}

// applyCommittedEntries applies committed log entries to the state machine
func (rn *RaftNode) applyCommittedEntries() {
	defer rn.wg.Done()

	for {
		select {
		case <-rn.stopCh:
			return
		default:
		}

		rn.mu.Lock()
		if rn.lastApplied < rn.commitIndex {
			for i := rn.lastApplied + 1; i <= rn.commitIndex; i++ {
				if i < uint64(len(rn.log)) {
					entry := rn.log[i]
					if entry.Type == LogEntryCommand && rn.applyCh != nil {
						rn.applyCh <- entry
					}
					rn.lastApplied = i
				}
			}
		}
		rn.mu.Unlock()

		time.Sleep(10 * time.Millisecond)
	}
}

// handleRequestVote handles incoming RequestVote RPCs
func (rn *RaftNode) handleRequestVote(conn net.Conn, isPreVote bool) {
	// Read request length
	lenBuf := make([]byte, 4)
	if _, err := conn.Read(lenBuf); err != nil {
		return
	}
	msgLen := binary.BigEndian.Uint32(lenBuf)

	// Read request body
	body := make([]byte, msgLen)
	if _, err := conn.Read(body); err != nil {
		return
	}

	var args RequestVoteArgs
	if err := json.Unmarshal(body, &args); err != nil {
		return
	}

	rn.mu.Lock()
	defer rn.mu.Unlock()

	reply := RequestVoteReply{
		Term:        rn.currentTerm,
		VoteGranted: false,
	}

	// If RPC term is greater, update current term
	if args.Term > rn.currentTerm && !isPreVote {
		rn.becomeFollower(args.Term, "")
	}

	// Check if we can grant vote
	if args.Term >= rn.currentTerm {
		lastLogIndex := uint64(len(rn.log) - 1)
		lastLogTerm := rn.log[lastLogIndex].Term

		// Check if candidate's log is at least as up-to-date as ours
		logOK := args.LastLogTerm > lastLogTerm ||
			(args.LastLogTerm == lastLogTerm && args.LastLogIndex >= lastLogIndex)

		if isPreVote {
			// For pre-vote, grant if log is OK and we haven't heard from leader recently
			reply.VoteGranted = logOK
		} else {
			// For real vote, also check votedFor
			if (rn.votedFor == "" || rn.votedFor == args.CandidateID) && logOK {
				rn.votedFor = args.CandidateID
				reply.VoteGranted = true
				// Reset election timer
				select {
				case rn.heartbeatCh <- struct{}{}:
				default:
				}
			}
		}
	}

	reply.Term = rn.currentTerm

	// Send reply
	replyData, _ := json.Marshal(reply)
	respType := RaftMsgRequestVoteResp
	if isPreVote {
		respType = RaftMsgPreVoteResp
	}
	conn.Write([]byte{respType})
	binary.Write(conn, binary.BigEndian, uint32(len(replyData)))
	conn.Write(replyData)
}

// handleAppendEntries handles incoming AppendEntries RPCs
func (rn *RaftNode) handleAppendEntries(conn net.Conn) {
	// Read request length
	lenBuf := make([]byte, 4)
	if _, err := conn.Read(lenBuf); err != nil {
		return
	}
	msgLen := binary.BigEndian.Uint32(lenBuf)

	// Read request body
	body := make([]byte, msgLen)
	if _, err := conn.Read(body); err != nil {
		return
	}

	var args AppendEntriesArgs
	if err := json.Unmarshal(body, &args); err != nil {
		return
	}

	rn.mu.Lock()
	defer rn.mu.Unlock()

	reply := AppendEntriesReply{
		Term:    rn.currentTerm,
		Success: false,
	}

	// Reply false if term < currentTerm
	if args.Term < rn.currentTerm {
		rn.sendAppendEntriesReply(conn, reply)
		return
	}

	// If RPC term is greater, update current term
	if args.Term > rn.currentTerm {
		rn.becomeFollower(args.Term, args.LeaderID)
	} else if rn.GetState() != StateFollower {
		rn.becomeFollower(args.Term, args.LeaderID)
	}

	// Reset election timer
	select {
	case rn.heartbeatCh <- struct{}{}:
	default:
	}

	// Update leader info
	rn.leaderMu.Lock()
	rn.leaderID = args.LeaderID
	rn.leaderMu.Unlock()

	// Check if log contains entry at prevLogIndex with prevLogTerm
	if args.PrevLogIndex > 0 {
		if args.PrevLogIndex >= uint64(len(rn.log)) {
			reply.ConflictIndex = uint64(len(rn.log))
			rn.sendAppendEntriesReply(conn, reply)
			return
		}
		if rn.log[args.PrevLogIndex].Term != args.PrevLogTerm {
			// Find first index of conflicting term
			conflictTerm := rn.log[args.PrevLogIndex].Term
			reply.ConflictTerm = conflictTerm
			for i := args.PrevLogIndex; i > 0; i-- {
				if rn.log[i-1].Term != conflictTerm {
					reply.ConflictIndex = i
					break
				}
			}
			rn.sendAppendEntriesReply(conn, reply)
			return
		}
	}

	// Append new entries
	for i, entry := range args.Entries {
		idx := args.PrevLogIndex + 1 + uint64(i)
		if idx < uint64(len(rn.log)) {
			if rn.log[idx].Term != entry.Term {
				// Delete conflicting entry and all that follow
				rn.log = rn.log[:idx]
				rn.log = append(rn.log, entry)
			}
		} else {
			rn.log = append(rn.log, entry)
		}
	}

	// Update commit index
	if args.LeaderCommit > rn.commitIndex {
		lastNewIndex := args.PrevLogIndex + uint64(len(args.Entries))
		if args.LeaderCommit < lastNewIndex {
			rn.commitIndex = args.LeaderCommit
		} else {
			rn.commitIndex = lastNewIndex
		}
	}

	reply.Success = true
	reply.Term = rn.currentTerm
	rn.sendAppendEntriesReply(conn, reply)
}

// sendAppendEntriesReply sends an AppendEntries reply
func (rn *RaftNode) sendAppendEntriesReply(conn net.Conn, reply AppendEntriesReply) {
	replyData, _ := json.Marshal(reply)
	conn.Write([]byte{RaftMsgAppendEntriesResp})
	binary.Write(conn, binary.BigEndian, uint32(len(replyData)))
	conn.Write(replyData)
}

// sendRequestVote sends a RequestVote RPC to a peer
func (rn *RaftNode) sendRequestVote(addr string, args RequestVoteArgs) *RequestVoteReply {
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return nil
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(1 * time.Second))

	// Send message type
	msgType := RaftMsgRequestVote
	if args.PreVote {
		msgType = RaftMsgPreVote
	}
	conn.Write([]byte{msgType})

	// Send request
	data, _ := json.Marshal(args)
	binary.Write(conn, binary.BigEndian, uint32(len(data)))
	conn.Write(data)

	// Read response type
	respType := make([]byte, 1)
	if _, err := conn.Read(respType); err != nil {
		return nil
	}

	// Read response length
	lenBuf := make([]byte, 4)
	if _, err := conn.Read(lenBuf); err != nil {
		return nil
	}
	respLen := binary.BigEndian.Uint32(lenBuf)

	// Read response body
	body := make([]byte, respLen)
	if _, err := conn.Read(body); err != nil {
		return nil
	}

	var reply RequestVoteReply
	if err := json.Unmarshal(body, &reply); err != nil {
		return nil
	}

	return &reply
}

// sendAppendEntries sends an AppendEntries RPC to a peer
func (rn *RaftNode) sendAppendEntries(addr string, args AppendEntriesArgs) *AppendEntriesReply {
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return nil
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))

	// Send message type
	conn.Write([]byte{RaftMsgAppendEntries})

	// Send request
	data, _ := json.Marshal(args)
	binary.Write(conn, binary.BigEndian, uint32(len(data)))
	conn.Write(data)

	// Read response type
	respType := make([]byte, 1)
	if _, err := conn.Read(respType); err != nil {
		return nil
	}

	// Read response length
	lenBuf := make([]byte, 4)
	if _, err := conn.Read(lenBuf); err != nil {
		return nil
	}
	respLen := binary.BigEndian.Uint32(lenBuf)

	// Read response body
	body := make([]byte, respLen)
	if _, err := conn.Read(body); err != nil {
		return nil
	}

	var reply AppendEntriesReply
	if err := json.Unmarshal(body, &reply); err != nil {
		return nil
	}

	return &reply
}

// Propose proposes a new command to be replicated
func (rn *RaftNode) Propose(command []byte) error {
	if !rn.IsLeader() {
		return fmt.Errorf("not the leader")
	}

	rn.mu.Lock()
	defer rn.mu.Unlock()

	entry := LogEntry{
		Term:    rn.currentTerm,
		Index:   uint64(len(rn.log)),
		Command: command,
		Type:    LogEntryCommand,
	}

	rn.log = append(rn.log, entry)

	// Trigger immediate replication
	go rn.broadcastAppendEntries()

	return nil
}

// AddPeer adds a new peer to the cluster
func (rn *RaftNode) AddPeer(id, addr string) {
	rn.peersMu.Lock()
	defer rn.peersMu.Unlock()

	rn.peers[addr] = &RaftPeer{
		ID:        id,
		Addr:      addr,
		isHealthy: true,
	}

	rn.mu.Lock()
	rn.nextIndex[addr] = uint64(len(rn.log))
	rn.matchIndex[addr] = 0
	rn.mu.Unlock()
}

// RemovePeer removes a peer from the cluster
func (rn *RaftNode) RemovePeer(addr string) {
	rn.peersMu.Lock()
	defer rn.peersMu.Unlock()

	delete(rn.peers, addr)

	rn.mu.Lock()
	delete(rn.nextIndex, addr)
	delete(rn.matchIndex, addr)
	rn.mu.Unlock()
}

// GetClusterStatus returns the current cluster status
func (rn *RaftNode) GetClusterStatus() map[string]interface{} {
	rn.mu.RLock()
	defer rn.mu.RUnlock()

	rn.peersMu.RLock()
	peerList := make([]string, 0, len(rn.peers))
	for addr := range rn.peers {
		peerList = append(peerList, addr)
	}
	rn.peersMu.RUnlock()

	leaderID, leaderAddr := rn.GetLeader()

	return map[string]interface{}{
		"node_id":      rn.config.NodeID,
		"state":        rn.GetState().String(),
		"term":         rn.currentTerm,
		"leader_id":    leaderID,
		"leader_addr":  leaderAddr,
		"commit_index": rn.commitIndex,
		"last_applied": rn.lastApplied,
		"log_length":   len(rn.log),
		"peers":        peerList,
	}
}

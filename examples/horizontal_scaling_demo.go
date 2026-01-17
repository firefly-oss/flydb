package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"flydb/internal/cluster"
	"flydb/internal/storage"
)

// DemoNode represents a single node in the cluster demo
type DemoNode struct {
	ClusterMgr *cluster.UnifiedClusterManager
	Store      *storage.UnifiedStorageEngine
}

func (n *DemoNode) Close() {
	if n.ClusterMgr != nil {
		n.ClusterMgr.Stop()
	}
	if n.Store != nil {
		n.Store.Close()
	}
}

func main() {
	fmt.Println("=== FlyDB Horizontal Scaling Demo ===")
	fmt.Println()

	// Create a temporary directory for demo data
	tmpDir, err := os.MkdirTemp("", "flydb-demo-*")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Step 1: Start a 3-node cluster
	fmt.Println("Step 1: Starting 3-node cluster...")
	nodes := startCluster(3, tmpDir)
	defer stopCluster(nodes)

	// Wait for cluster to stabilize and elect a leader
	fmt.Println("Waiting for cluster to stabilize...")
	time.Sleep(5 * time.Second)

	// Step 2: Write data across the cluster
	fmt.Println("\nStep 2: Writing 100 keys across the cluster...")
	writeData(nodes[0], 100)

	// Step 3: Show data distribution (conceptually)
	fmt.Println("\nStep 3: Cluster Status")
	status := nodes[0].ClusterMgr.GetStatus()
	fmt.Printf("  Cluster Health: %s\n", status.Health)
	fmt.Printf("  Total Nodes:    %d\n", status.TotalNodes)

	// Step 4: Read data
	fmt.Println("\nStep 4: Reading data (requests auto-routed to correct nodes)...")
	readData(nodes[1], []string{"user:1", "user:50", "user:99"})

	fmt.Println("\n=== Demo Complete ===")
}

func startCluster(nodeCount int, baseDir string) []*DemoNode {
	nodes := make([]*DemoNode, nodeCount)
	seeds := []string{"127.0.0.1:7946"}

	for i := 0; i < nodeCount; i++ {
		nodeID := fmt.Sprintf("node%d", i+1)
		clusterPort := 7946 + i
		dataPort := 8946 + i
		dataDir := fmt.Sprintf("%s/node%d", baseDir, i+1)
		os.MkdirAll(dataDir, 0755)

		storeCfg := storage.StorageConfig{
			DataDir: dataDir,
		}
		store, err := storage.NewStorageEngine(storeCfg)
		if err != nil {
			log.Fatalf("Failed to create storage for node %d: %v", i+1, err)
		}

		// Create cluster manager
		clusterCfg := cluster.ClusterConfig{
			NodeID:            nodeID,
			NodeAddr:          "127.0.0.1",
			ClusterPort:       clusterPort,
			DataPort:          dataPort,
			Seeds:             seeds,
			PartitionCount:    256,
			ReplicationFactor: 3,
			HeartbeatInterval: 500 * time.Millisecond,
			ElectionTimeout:   2 * time.Second,
			SyncTimeout:       5 * time.Second,
			DataDir:           dataDir + "/cluster",
		}

		clusterMgr := cluster.NewUnifiedClusterManager(clusterCfg)
		clusterMgr.SetStore(store)
		clusterMgr.SetWAL(store.WAL())

		if err := clusterMgr.Start(); err != nil {
			log.Fatalf("Failed to start cluster manager for node %d: %v", i+1, err)
		}

		nodes[i] = &DemoNode{
			ClusterMgr: clusterMgr,
			Store:      store,
		}
		fmt.Printf("  ✓ Node %d started (ID: %s, Cluster Port: %d)\n", i+1, nodeID, clusterPort)
	}

	return nodes
}

func stopCluster(nodes []*DemoNode) {
	for i, node := range nodes {
		if node != nil {
			node.Close()
			fmt.Printf("  ✓ Node %d stopped\n", i+1)
		}
	}
}

func writeData(node *DemoNode, count int) {
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("user:%d", i)
		value := fmt.Sprintf(`{"id":%d,"name":"User %d"}`, i, i)

		if err := node.ClusterMgr.Put(key, []byte(value)); err != nil {
			log.Printf("Failed to write %s: %v", key, err)
		}

		if (i+1)%20 == 0 {
			fmt.Printf("  Written %d keys...\n", i+1)
		}
	}
	fmt.Printf("  ✓ Wrote %d keys\n", count)
}

func readData(node *DemoNode, keys []string) {
	for _, key := range keys {
		value, err := node.ClusterMgr.Get(key)
		if err != nil {
			fmt.Printf("  ✗ Failed to read %s: %v\n", key, err)
		} else {
			fmt.Printf("  ✓ %s = %s\n", key, string(value))
		}
	}
}

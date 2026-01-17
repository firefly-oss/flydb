/*
 * Copyright (c) 2026 FlyDB Authors.
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
flydb-discover - FlyDB Node Discovery Tool

This tool discovers FlyDB nodes on the local network using mDNS (Bonjour/Avahi).
It can be used by install.sh to find existing cluster nodes for joining.

Usage:
    flydb-discover                    # Discover nodes (5 second timeout)
    flydb-discover --timeout 10       # Custom timeout in seconds
    flydb-discover --json             # Output as JSON
    flydb-discover --quiet            # Only output addresses (for scripting)
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"flydb/internal/cluster"
)

const (
	version   = "1.0.0"
	copyright = "Copyright (c) 2026 FlyDB Authors"
)

// ANSI color codes
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
)

func main() {
	timeout := flag.Int("timeout", 5, "Discovery timeout in seconds")
	jsonOutput := flag.Bool("json", false, "Output as JSON")
	quiet := flag.Bool("quiet", false, "Only output cluster addresses (for scripting)")
	help := flag.Bool("help", false, "Show help")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.BoolVar(help, "h", false, "Show help")
	flag.BoolVar(showVersion, "v", false, "Show version information")

	flag.Parse()

	if *help {
		printUsage()
		os.Exit(0)
	}

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	// Suppress mDNS library logging (it logs IPv6 errors that are not critical)
	log.SetOutput(io.Discard)

	// Show banner and welcome message unless in quiet/json mode
	if !*quiet && !*jsonOutput {
		printBanner()
	}

	// Create discovery service (not advertising, just discovering)
	discovery := cluster.NewDiscoveryService(cluster.DiscoveryConfig{
		NodeID:  "discover-client",
		Enabled: false, // Don't advertise, just discover
	})

	// Show scanning message unless in quiet mode
	if !*quiet && !*jsonOutput {
		fmt.Printf("%s%sℹ%s Scanning for FlyDB nodes on the network (timeout: %ds)...\n\n",
			cyan, bold, reset, *timeout)
	}

	// Discover nodes
	nodes, err := discovery.DiscoverNodes(time.Duration(*timeout) * time.Second)
	if err != nil {
		if !*quiet {
			fmt.Fprintf(os.Stderr, "%s%s✗%s Discovery failed: %v\n", red, bold, reset, err)
		}
		os.Exit(1)
	}

	if len(nodes) == 0 {
		if !*quiet && !*jsonOutput {
			fmt.Printf("%s%s⚠%s No FlyDB nodes found on the network.\n\n", yellow, bold, reset)
			fmt.Printf("%s%sTROUBLESHOOTING%s\n\n", bold, cyan, reset)
			fmt.Printf("%s  Common issues:%s\n", dim, reset)
			fmt.Printf("    %s•%s FlyDB nodes are not running with discovery enabled\n", yellow, reset)
			fmt.Printf("    %s•%s mDNS/Bonjour is blocked by firewall (UDP port 5353)\n", yellow, reset)
			fmt.Printf("    %s•%s Nodes are on a different network segment\n\n", yellow, reset)
			fmt.Printf("%s  Try:%s\n", dim, reset)
			fmt.Printf("    %sflydb-discover --timeout 10%s   # Increase timeout\n\n", green, reset)
		}
		os.Exit(0)
	}

	if *jsonOutput {
		outputJSON(nodes)
	} else if *quiet {
		outputQuiet(nodes)
	} else {
		outputHuman(nodes)
	}
}

func printBanner() {
	fmt.Println()
	fmt.Printf("%s%s", cyan, bold)
	fmt.Println("  ███████╗██╗  ██╗   ██╗██████╗ ██████╗ ")
	fmt.Println("  ██╔════╝██║  ╚██╗ ██╔╝██╔══██╗██╔══██╗")
	fmt.Println("  █████╗  ██║   ╚████╔╝ ██║  ██║██████╔╝")
	fmt.Println("  ██╔══╝  ██║    ╚██╔╝  ██║  ██║██╔══██╗")
	fmt.Println("  ██║     ███████╗██║   ██████╔╝██████╔╝")
	fmt.Println("  ╚═╝     ╚══════╝╚═╝   ╚═════╝ ╚═════╝ ")
	fmt.Printf("%s\n", reset)
	fmt.Printf("  %s%sFlyDB Discover%s %sv%s%s\n", green, bold, reset, dim, version, reset)
	fmt.Printf("  %sNetwork Node Discovery Tool%s\n\n", dim, reset)
}



func printVersion() {
	fmt.Println()
	fmt.Printf("  %s%sFlyDB Discover%s %sv%s%s\n", cyan, bold, reset, dim, version, reset)
	fmt.Printf("  %sNetwork Node Discovery Tool%s\n\n", dim, reset)
	fmt.Printf("  %s%s%s\n\n", dim, copyright, reset)
}

func printUsage() {
	// Print banner
	printBanner()

	// Description
	fmt.Printf("%s  Discovers FlyDB nodes on the local network using mDNS (Bonjour/Avahi).%s\n", dim, reset)
	fmt.Printf("%s  Useful for finding existing cluster nodes to join.%s\n\n", dim, reset)

	// Usage
	fmt.Printf("%sUsage:%s flydb-discover [options]\n\n", bold, reset)

	// Options
	fmt.Printf("%s%sOPTIONS%s\n\n", bold, cyan, reset)
	fmt.Printf("    %s--timeout%s <seconds>   Discovery timeout (default: 5)\n", green, reset)
	fmt.Printf("    %s--json%s               Output results as JSON\n", green, reset)
	fmt.Printf("    %s--quiet%s, %s-q%s          Only output addresses (for scripting)\n", green, reset, green, reset)
	fmt.Printf("    %s--version%s, %s-v%s        Show version information\n", green, reset, green, reset)
	fmt.Printf("    %s--help%s, %s-h%s           Show this help message\n\n", green, reset, green, reset)

	// Examples
	fmt.Printf("%s%sEXAMPLES%s\n\n", bold, cyan, reset)
	fmt.Printf("%s    # Discover nodes with default timeout%s\n", dim, reset)
	fmt.Println("    flydb-discover")
	fmt.Println()
	fmt.Printf("%s    # Increase timeout for slower networks%s\n", dim, reset)
	fmt.Println("    flydb-discover --timeout 10")
	fmt.Println()
	fmt.Printf("%s    # Get JSON output for automation%s\n", dim, reset)
	fmt.Println("    flydb-discover --json")
	fmt.Println()
	fmt.Printf("%s    # Get just addresses for scripting%s\n", dim, reset)
	fmt.Println("    flydb-discover --quiet")
	fmt.Println()
	fmt.Printf("%s    # Use in install script to find cluster%s\n", dim, reset)
	fmt.Println("    PEERS=$(flydb-discover --quiet)")
	fmt.Println()

	// Network requirements
	fmt.Printf("%s%sNETWORK REQUIREMENTS%s\n\n", bold, cyan, reset)
	fmt.Printf("    %s•%s mDNS uses UDP port 5353 (multicast)\n", yellow, reset)
	fmt.Printf("    %s•%s Nodes must be on the same network segment\n", yellow, reset)
	fmt.Printf("    %s•%s Firewalls must allow mDNS traffic\n\n", yellow, reset)
}

func outputJSON(nodes []*cluster.DiscoveredNode) {
	type nodeOutput struct {
		NodeID      string `json:"node_id"`
		ClusterID   string `json:"cluster_id,omitempty"`
		ClusterAddr string `json:"cluster_addr"`
		RaftAddr    string `json:"raft_addr,omitempty"`
		HTTPAddr    string `json:"http_addr,omitempty"`
		Version     string `json:"version,omitempty"`
	}

	output := make([]nodeOutput, len(nodes))
	for i, n := range nodes {
		output[i] = nodeOutput{
			NodeID:      n.NodeID,
			ClusterID:   n.ClusterID,
			ClusterAddr: n.ClusterAddr,
			RaftAddr:    n.RaftAddr,
			HTTPAddr:    n.HTTPAddr,
			Version:     n.Version,
		}
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}

func outputQuiet(nodes []*cluster.DiscoveredNode) {
	addrs := make([]string, len(nodes))
	for i, n := range nodes {
		addrs[i] = n.ClusterAddr
	}
	fmt.Println(strings.Join(addrs, ","))
}

func outputHuman(nodes []*cluster.DiscoveredNode) {
	fmt.Printf("%s%s✓%s Found %d FlyDB node(s)\n\n", green, bold, reset, len(nodes))

	for i, n := range nodes {
		// Node header with index and ID
		fmt.Printf("  %s[%d]%s %s%s%s\n",
			dim, i+1, reset,
			bold+cyan, n.NodeID, reset)

		// Cluster address (always present)
		fmt.Printf("      %sCluster Address:%s %s%s%s\n",
			dim, reset,
			green, n.ClusterAddr, reset)

		// Raft address (optional)
		if n.RaftAddr != "" {
			fmt.Printf("      %sRaft Address:%s    %s\n",
				dim, reset, n.RaftAddr)
		}

		// HTTP address (optional)
		if n.HTTPAddr != "" {
			fmt.Printf("      %sHTTP Address:%s    %s\n",
				dim, reset, n.HTTPAddr)
		}

		// Cluster ID (optional)
		if n.ClusterID != "" {
			fmt.Printf("      %sCluster ID:%s      %s\n",
				dim, reset, n.ClusterID)
		}

		// Version (optional)
		if n.Version != "" {
			fmt.Printf("      %sVersion:%s         %s\n",
				dim, reset, n.Version)
		}

		fmt.Println()
	}

	// Helpful tip
	fmt.Printf("%s  Tip: Use --json for machine-readable output%s\n\n", dim, reset)
}

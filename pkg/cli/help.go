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

package cli

import (
	"fmt"
	"strings"
)

// Command represents a CLI command for help display.
type Command struct {
	Name        string
	Aliases     []string
	Description string
	Usage       string
	Examples    []Example
	Flags       []Flag
	SubCommands []Command
}

// Example represents a usage example.
type Example struct {
	Description string
	Command     string
}

// Flag represents a command-line flag.
type Flag struct {
	Name        string
	Short       string
	Description string
	Default     string
	Required    bool
}

// HelpFormatter formats help output.
type HelpFormatter struct {
	AppName    string
	AppVersion string
	Commands   []Command
}

// NewHelpFormatter creates a new help formatter.
func NewHelpFormatter(appName, version string) *HelpFormatter {
	return &HelpFormatter{
		AppName:    appName,
		AppVersion: version,
		Commands:   make([]Command, 0),
	}
}

// AddCommand adds a command to the help formatter.
func (h *HelpFormatter) AddCommand(cmd Command) {
	h.Commands = append(h.Commands, cmd)
}

// PrintVersion prints version information.
func (h *HelpFormatter) PrintVersion() {
	fmt.Printf("%s version %s\n", h.AppName, h.AppVersion)
}

// PrintUsage prints the main usage information.
func (h *HelpFormatter) PrintUsage() {
	fmt.Printf("\n%s\n", Highlight(h.AppName+" - FlyDB Command Line Interface"))
	fmt.Printf("Version: %s\n\n", h.AppVersion)
	
	fmt.Printf("%s\n", Highlight("USAGE:"))
	fmt.Printf("  %s [flags] [command]\n\n", h.AppName)
	
	if len(h.Commands) > 0 {
		fmt.Printf("%s\n", Highlight("COMMANDS:"))
		maxLen := 0
		for _, cmd := range h.Commands {
			if len(cmd.Name) > maxLen {
				maxLen = len(cmd.Name)
			}
		}
		for _, cmd := range h.Commands {
			fmt.Printf("  %-*s  %s\n", maxLen+2, cmd.Name, cmd.Description)
		}
		fmt.Println()
	}
}

// PrintCommandHelp prints help for a specific command.
func (h *HelpFormatter) PrintCommandHelp(cmdName string) {
	for _, cmd := range h.Commands {
		if cmd.Name == cmdName || contains(cmd.Aliases, cmdName) {
			h.printCommand(cmd)
			return
		}
	}
	PrintError("Unknown command: %s", cmdName)
	fmt.Println("Run with --help to see available commands.")
}

func (h *HelpFormatter) printCommand(cmd Command) {
	fmt.Printf("\n%s\n", Highlight(strings.ToUpper(cmd.Name)))
	fmt.Printf("  %s\n\n", cmd.Description)
	
	if len(cmd.Aliases) > 0 {
		fmt.Printf("%s\n", Highlight("ALIASES:"))
		fmt.Printf("  %s\n\n", strings.Join(cmd.Aliases, ", "))
	}
	
	if cmd.Usage != "" {
		fmt.Printf("%s\n", Highlight("USAGE:"))
		fmt.Printf("  %s\n\n", cmd.Usage)
	}
	
	if len(cmd.Flags) > 0 {
		fmt.Printf("%s\n", Highlight("FLAGS:"))
		for _, f := range cmd.Flags {
			flagStr := ""
			if f.Short != "" {
				flagStr = fmt.Sprintf("-%s, --%s", f.Short, f.Name)
			} else {
				flagStr = fmt.Sprintf("    --%s", f.Name)
			}
			
			defaultStr := ""
			if f.Default != "" {
				defaultStr = fmt.Sprintf(" (default: %s)", f.Default)
			}
			
			reqStr := ""
			if f.Required {
				reqStr = colorize(Red, " [required]")
			}
			
			fmt.Printf("  %-20s  %s%s%s\n", flagStr, f.Description, defaultStr, reqStr)
		}
		fmt.Println()
	}
	
	if len(cmd.Examples) > 0 {
		fmt.Printf("%s\n", Highlight("EXAMPLES:"))
		for _, ex := range cmd.Examples {
			fmt.Printf("  %s\n", Dimmed("# "+ex.Description))
			fmt.Printf("  %s\n\n", Info(ex.Command))
		}
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}


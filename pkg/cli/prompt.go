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
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Prompt displays a prompt and reads user input.
func Prompt(message string) string {
	fmt.Print(message)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimSpace(input)
}

// PromptWithDefault displays a prompt with a default value.
func PromptWithDefault(message, defaultVal string) string {
	fmt.Printf("%s [%s%s%s]: ", message, Yellow, defaultVal, Reset)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultVal
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

// PromptYesNo prompts for a yes/no answer.
func PromptYesNo(message string, defaultYes bool) bool {
	var prompt string
	if defaultYes {
		prompt = fmt.Sprintf("%s [%sY%s/n]: ", message, Bold, Reset)
	} else {
		prompt = fmt.Sprintf("%s [y/%sN%s]: ", message, Bold, Reset)
	}
	
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}
	
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultYes
	}
	
	return input == "y" || input == "yes"
}

// PromptPassword prompts for a password (no echo in future versions).
func PromptPassword(message string) string {
	fmt.Printf("%s: ", message)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimSpace(input)
}

// PromptSelect displays a selection menu and returns the selected index.
func PromptSelect(message string, options []string, defaultIndex int) int {
	fmt.Println(message)
	for i, opt := range options {
		marker := "  "
		if i == defaultIndex {
			marker = colorize(Green, "> ")
		}
		fmt.Printf("%s%d) %s\n", marker, i+1, opt)
	}
	
	for {
		input := Prompt(fmt.Sprintf("Select [1-%d]: ", len(options)))
		if input == "" {
			return defaultIndex
		}
		
		var selected int
		if _, err := fmt.Sscanf(input, "%d", &selected); err == nil {
			if selected >= 1 && selected <= len(options) {
				return selected - 1
			}
		}
		PrintWarning("Invalid selection. Please enter a number between 1 and %d.", len(options))
	}
}

// Confirm prompts for confirmation before a destructive operation.
func Confirm(message string) bool {
	fmt.Printf("%s %s\n", WarningIcon(), Warning(message))
	return PromptYesNo("Are you sure you want to continue?", false)
}

// ConfirmDestructive prompts for confirmation with extra warning for destructive operations.
func ConfirmDestructive(message, confirmWord string) bool {
	fmt.Printf("\n%s %s\n", ErrorIcon(), Error("DESTRUCTIVE OPERATION"))
	fmt.Printf("  %s\n\n", message)
	fmt.Printf("Type '%s%s%s' to confirm: ", Bold, confirmWord, Reset)
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	
	return strings.TrimSpace(input) == confirmWord
}


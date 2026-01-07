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
	"sync"
	"time"
)

// SpinnerFrames defines the animation frames for the spinner.
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner provides an animated progress indicator.
type Spinner struct {
	message  string
	frames   []string
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	running  bool
}

// NewSpinner creates a new spinner with the given message.
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message:  message,
		frames:   SpinnerFrames,
		interval: 80 * time.Millisecond,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	s.mu.Unlock()

	go func() {
		defer close(s.done)
		i := 0
		for {
			select {
			case <-s.stop:
				// Clear the spinner line
				fmt.Print("\r\033[K")
				return
			default:
				frame := s.frames[i%len(s.frames)]
				if colorsEnabled {
					fmt.Printf("\r%s%s%s %s", Cyan, frame, Reset, s.message)
				} else {
					fmt.Printf("\r%s %s", frame, s.message)
				}
				i++
				time.Sleep(s.interval)
			}
		}
	}()
}

// Stop stops the spinner animation.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stop)
	<-s.done
}

// StopWithSuccess stops the spinner and shows a success message.
func (s *Spinner) StopWithSuccess(message string) {
	s.Stop()
	PrintSuccess("%s", message)
}

// StopWithError stops the spinner and shows an error message.
func (s *Spinner) StopWithError(message string) {
	s.Stop()
	PrintError("%s", message)
}

// StopWithWarning stops the spinner and shows a warning message.
func (s *Spinner) StopWithWarning(message string) {
	s.Stop()
	PrintWarning("%s", message)
}

// UpdateMessage updates the spinner message while running.
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// ProgressBar provides a simple progress bar.
type ProgressBar struct {
	total   int
	current int
	width   int
	message string
	mu      sync.Mutex
}

// NewProgressBar creates a new progress bar.
func NewProgressBar(total int, message string) *ProgressBar {
	return &ProgressBar{
		total:   total,
		current: 0,
		width:   40,
		message: message,
	}
}

// Update updates the progress bar to the given value.
func (p *ProgressBar) Update(current int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = current
	if p.current > p.total {
		p.current = p.total
	}

	percent := float64(p.current) / float64(p.total)
	filled := int(percent * float64(p.width))
	empty := p.width - filled

	bar := fmt.Sprintf("[%s%s]",
		colorize(Green, repeatChar('█', filled)),
		repeatChar('░', empty))

	fmt.Printf("\r%s %s %3.0f%%", p.message, bar, percent*100)
}

// Complete marks the progress bar as complete.
func (p *ProgressBar) Complete() {
	p.Update(p.total)
	fmt.Println()
}

func repeatChar(char rune, count int) string {
	result := make([]rune, count)
	for i := range result {
		result[i] = char
	}
	return string(result)
}

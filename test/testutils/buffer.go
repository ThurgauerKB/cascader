/*
Copyright 2025 Thurgauer Kantonalbank

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testutils

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/gomega" // nolint:staticcheck
)

// LogBuffer is a thread-safe buffer for capturing logs.
var LogBuffer = NewSyncBuffer()

// SyncBuffer is a thread-safe buffer implementation compatible with zapcore.WriteSyncer.
type SyncBuffer struct {
	buffer *bytes.Buffer
	mu     sync.Mutex
}

// NewSyncBuffer creates a new SyncBuffer instance.
func NewSyncBuffer() *SyncBuffer {
	return &SyncBuffer{
		buffer: bytes.NewBuffer(nil),
	}
}

// Write writes data to the buffer with thread-safety.
func (s *SyncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buffer.Write(p)
}

// String retrieves the string value of the buffer's contents.
func (s *SyncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buffer.String()
}

// Sync is a no-op to satisfy zapcore.WriteSyncer interface.
func (s *SyncBuffer) Sync() error {
	// No-op: sync is not necessary for an in-memory buffer.
	return nil
}

// Reset clears the buffer's contents.
func (s *SyncBuffer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buffer.Reset()
}

// ContainsLogs checks if the expected log is present in the log buffer.
func ContainsLogs(expectedLog string, timeout, interval time.Duration) {
	Eventually(func() bool {
		return strings.Contains(LogBuffer.String(), expectedLog)
	}, timeout, interval).Should(BeTrue(), fmt.Sprintf("Expected log not found: %s", expectedLog))
}

// ContainsNotLogs checks that the expected log is NOT present in the log buffer
// for the entire duration of the timeout, checking every interval.
func ContainsNotLogs(expectedLog string, timeout, interval time.Duration) {
	Consistently(func() bool {
		return strings.Contains(LogBuffer.String(), expectedLog)
	}, timeout, interval).Should(BeFalse(), fmt.Sprintf("Expected log should not be found: %s", expectedLog))
}

// CountLogOccurrences checks how many times the expected string appears in the log buffer.
func CountLogOccurrences(expectedText string, amount int, timeout, interval time.Duration) {
	Eventually(func() bool {
		content := LogBuffer.String()
		return strings.Count(content, expectedText) == amount
	}, timeout, interval).Should(BeTrue(), "Expected text not found")
}

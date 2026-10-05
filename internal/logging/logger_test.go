/*
Copyright 2025 containeroo.ch

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

package logging

import (
	"bytes"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	uzap "go.uber.org/zap"
)

func TestInitLogging(t *testing.T) {
	t.Parallel()

	t.Run("Valid Configuration JSON", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		logger := InitLogging(LogFormatJSON, uzap.InfoLevel, false, &buf)
		assert.NotEqual(t, logr.Logger{}, logger)
	})

	t.Run("Valid Configuration Console", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		logger := InitLogging(LogFormatText, uzap.ErrorLevel, true, &buf)
		assert.NotEqual(t, logr.Logger{}, logger)
	})
}

func TestNewLogger(t *testing.T) {
	t.Parallel()

	t.Run("Setup Logger", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		logger := newLogger(LogFormatJSON, uzap.ErrorLevel, true, &buf)
		assert.NotEqual(t, logr.Logger{}, logger)
	})
}

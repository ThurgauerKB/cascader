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

package logging_test

import (
	"bytes"
	"testing"

	"github.com/thurgauerkb/cascader/internal/config"
	"github.com/thurgauerkb/cascader/internal/logging"

	"github.com/stretchr/testify/assert"
)

// TestSetupLogger tests the SetupLogger function for various configurations.
func TestSetupLogger(t *testing.T) {
	t.Parallel()

	t.Run("Valid JSON Encoder", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			LogDev:             true,
			LogEncoder:         "json",
			LogStacktraceLevel: "error",
		}

		var buf bytes.Buffer
		_, err := logging.SetupLogger(cfg, &buf)
		assert.NoError(t, err)
	})

	t.Run("Valid Console Encoder", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			LogDev:             false,
			LogEncoder:         "console",
			LogStacktraceLevel: "panic",
		}

		var buf bytes.Buffer
		_, err := logging.SetupLogger(cfg, &buf)
		assert.NoError(t, err)
	})

	t.Run("Invalid Log Encoder", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			LogEncoder:         "invalid",
			LogStacktraceLevel: "error",
		}
		var buf bytes.Buffer
		_, err := logging.SetupLogger(cfg, &buf)
		assert.Error(t, err)
	})

	t.Run("Log level Warn", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			LogDev:             false,
			LogEncoder:         "console",
			LogStacktraceLevel: "panic",
		}
		var buf bytes.Buffer
		_, err := logging.SetupLogger(cfg, &buf)
		assert.NoError(t, err)
	})

	t.Run("Log level Error", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			LogDev:             false,
			LogEncoder:         "console",
			LogStacktraceLevel: "panic",
		}
		var buf bytes.Buffer
		_, err := logging.SetupLogger(cfg, &buf)
		assert.NoError(t, err)
	})

	t.Run("Stacktrace Level Info", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			LogDev:             true,
			LogEncoder:         "console",
			LogStacktraceLevel: "info",
		}
		var buf bytes.Buffer
		_, err := logging.SetupLogger(cfg, &buf)
		assert.NoError(t, err)
	})

	t.Run("Stacktrace Level Error", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			LogDev:             true,
			LogEncoder:         "console",
			LogStacktraceLevel: "error",
		}
		var buf bytes.Buffer
		_, err := logging.SetupLogger(cfg, &buf)
		assert.NoError(t, err)
	})

	t.Run("Stacktrace Level Panic", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			LogDev:             true,
			LogEncoder:         "console",
			LogStacktraceLevel: "panic",
		}
		var buf bytes.Buffer
		_, err := logging.SetupLogger(cfg, &buf)
		assert.NoError(t, err)
	})

	t.Run("Invalid Stacktrace Level", func(t *testing.T) {
		t.Parallel()

		cfg := config.Config{
			LogDev:             true,
			LogEncoder:         "console",
			LogStacktraceLevel: "invalid",
		}
		var buf bytes.Buffer
		_, err := logging.SetupLogger(cfg, &buf)
		assert.Error(t, err)
		assert.EqualError(t, err, "invalid stacktrace level: \"invalid\"")
	})
}

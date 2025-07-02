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

package logging

import (
	"bytes"
	"strings"
	"testing"

	"github.com/thurgauerkb/cascader/internal/flag"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	uzap "go.uber.org/zap"
	zapcore "go.uber.org/zap/zapcore"
)

func TestInitLogging(t *testing.T) {
	t.Parallel()

	t.Run("Valid Configuration JSON", func(t *testing.T) {
		t.Parallel()

		opts := flag.Options{
			LogEncoder:         "json",
			LogStacktraceLevel: "info",
			LogDev:             false,
		}
		var buf bytes.Buffer

		logger, err := InitLogging(opts, &buf)
		assert.NoError(t, err)
		assert.NotEqual(t, logr.Logger{}, logger)
	})

	t.Run("Valid Configuration Console", func(t *testing.T) {
		t.Parallel()

		opts := flag.Options{
			LogEncoder:         "console",
			LogStacktraceLevel: "error",
			LogDev:             true,
		}
		var buf bytes.Buffer

		logger, err := InitLogging(opts, &buf)
		assert.NoError(t, err)
		assert.NotEqual(t, logr.Logger{}, logger)
	})

	t.Run("Invalid Log Encoder", func(t *testing.T) {
		t.Parallel()

		opts := flag.Options{
			LogEncoder:         "invalid-encoder",
			LogStacktraceLevel: "info",
			LogDev:             false,
		}
		var buf bytes.Buffer

		logger, err := InitLogging(opts, &buf)
		assert.Error(t, err)
		assert.EqualError(t, err, `invalid log encoder: "invalid-encoder"`)
		assert.Equal(t, logr.Logger{}, logger)
	})

	t.Run("Invalid Stacktrace Level", func(t *testing.T) {
		t.Parallel()

		opts := flag.Options{
			LogEncoder:         "json",
			LogStacktraceLevel: "invalid-level",
			LogDev:             false,
		}
		var buf bytes.Buffer

		logger, err := InitLogging(opts, &buf)
		assert.Error(t, err)
		assert.EqualError(t, err, `invalid stacktrace level: "invalid-level"`)
		assert.Equal(t, logr.Logger{}, logger)
	})
}

func TestSetupLogger(t *testing.T) {
	t.Parallel()

	t.Run("Setup Logger", func(t *testing.T) {
		t.Parallel()
		opts := flag.Options{
			LogDev:             true,
			LogEncoder:         "json",
			LogStacktraceLevel: "error",
		}

		var buf bytes.Buffer
		_, err := setupLogger(opts, &buf)
		assert.NoError(t, err)
	})

	t.Run("Error Setup Logger - invalid encoder", func(t *testing.T) {
		t.Parallel()

		opts := flag.Options{
			LogDev:             false,
			LogEncoder:         "invalid",
			LogStacktraceLevel: "panic",
		}
		var buf bytes.Buffer
		_, err := setupLogger(opts, &buf)
		assert.Error(t, err)
		assert.EqualError(t, err, "invalid log encoder: \"invalid\"")
	})

	t.Run("Error Setup Logger - invalid stacktrace level", func(t *testing.T) {
		t.Parallel()

		opts := flag.Options{
			LogDev:             true,
			LogEncoder:         "console",
			LogStacktraceLevel: "invalid",
		}
		var buf bytes.Buffer
		_, err := setupLogger(opts, &buf)
		assert.Error(t, err)
		assert.EqualError(t, err, "invalid stacktrace level: \"invalid\"")
	})
}

func TestEncoder(t *testing.T) {
	t.Parallel()

	t.Run("Encoder JSON", func(t *testing.T) {
		t.Parallel()

		e := "json"
		enc, err := encoder(e)
		assert.NoError(t, err)

		entry := zapcore.Entry{
			Level:   zapcore.InfoLevel,
			Message: "test message",
		}
		buf, err := enc.EncodeEntry(entry, nil)
		assert.NoError(t, err)

		expectedPrefix := `{"level":"info","msg":"test message"`
		assert.True(t, strings.HasPrefix(buf.String(), expectedPrefix), "encoder output should start with JSON prefix")
	})

	t.Run("Encoder Console", func(t *testing.T) {
		t.Parallel()

		e := "console"
		enc, err := encoder(e)
		assert.NoError(t, err)

		entry := zapcore.Entry{
			Level:   zapcore.InfoLevel,
			Message: "test message",
		}
		buf, err := enc.EncodeEntry(entry, nil)
		assert.NoError(t, err)

		expectedContains := "INFO\ttest message"
		assert.Contains(t, buf.String(), expectedContains, "encoder output should contain console-formatted message")
	})

	t.Run("Invalid Encoder", func(t *testing.T) {
		t.Parallel()

		e := "invalid"
		enc, err := encoder(e)
		assert.Nil(t, enc)
		assert.Error(t, err)
		assert.EqualError(t, err, "invalid log encoder: \"invalid\"")
	})
}

func TestStacktraceLevel(t *testing.T) {
	t.Parallel()

	t.Run("Stacktrace Level Info", func(t *testing.T) {
		t.Parallel()

		level := "info"
		result, err := stacktraceLevel(level)
		assert.NoError(t, err)
		assert.Equal(t, result, uzap.NewAtomicLevelAt(uzap.InfoLevel))
	})

	t.Run("Stacktrace Level Error", func(t *testing.T) {
		t.Parallel()

		level := "error"
		result, err := stacktraceLevel(level)
		assert.NoError(t, err)
		assert.Equal(t, result, uzap.NewAtomicLevelAt(uzap.ErrorLevel))
	})
	t.Run("Stacktrace Level Panic", func(t *testing.T) {
		t.Parallel()

		level := "panic"
		result, err := stacktraceLevel(level)
		assert.NoError(t, err)
		assert.Equal(t, result, uzap.NewAtomicLevelAt(uzap.PanicLevel))
	})

	t.Run("Invalid Stacktrace Level", func(t *testing.T) {
		t.Parallel()

		level := "invalid"
		result, err := stacktraceLevel(level)
		assert.Equal(t, uzap.AtomicLevel{}, result)
		assert.Error(t, err)
		assert.EqualError(t, err, `invalid stacktrace level: "invalid"`)
	})
}

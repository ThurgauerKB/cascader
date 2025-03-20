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
	"fmt"
	"io"

	"github.com/go-logr/logr"
	"github.com/thurgauerkb/cascader/internal/config"

	uzap "go.uber.org/zap"
	zapcore "go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// SetupLogger initializes and configures the logger based on the provided config.
func SetupLogger(cfg config.Config, out io.Writer) (logr.Logger, error) {
	opts := zap.Options{
		Development: cfg.LogDev,
		DestWriter:  out,
	}

	// Set log encoder format
	encoders := map[string]zapcore.Encoder{
		"json":    zapcore.NewJSONEncoder(uzap.NewProductionEncoderConfig()),
		"console": zapcore.NewConsoleEncoder(uzap.NewDevelopmentEncoderConfig()),
	}
	encoder, ok := encoders[cfg.LogEncoder]
	if !ok {
		return logr.Logger{}, fmt.Errorf("invalid log encoder: %q", cfg.LogEncoder)
	}
	opts.Encoder = encoder

	// Set stacktrace level
	levels := map[string]uzap.AtomicLevel{
		"info":  uzap.NewAtomicLevelAt(uzap.InfoLevel),
		"error": uzap.NewAtomicLevelAt(uzap.ErrorLevel),
		"panic": uzap.NewAtomicLevelAt(uzap.PanicLevel),
	}
	level, ok := levels[cfg.LogStacktraceLevel]
	if !ok {
		return logr.Logger{}, fmt.Errorf("invalid stacktrace level: %q", cfg.LogStacktraceLevel)
	}
	opts.StacktraceLevel = level

	return zap.New(zap.UseFlagOptions(&opts)), nil
}

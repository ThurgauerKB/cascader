/*
Copyright 2026 Thurgauer Kantonalbank

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
	"io"

	"github.com/go-logr/logr"
	uzap "go.uber.org/zap"
	zapcore "go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// LogFormat identifies the encoder used for application log output.
type LogFormat string

const (
	// LogFormatText emits human-readable development-style logs.
	LogFormatText LogFormat = "text"
	// LogFormatJSON emits structured JSON logs.
	LogFormatJSON LogFormat = "json"
)

// InitLogging configures the process-wide controller-runtime and klog loggers.
func InitLogging(format LogFormat, stacktraceLevel zapcore.Level, development bool, w io.Writer) logr.Logger {
	logger := newLogger(format, stacktraceLevel, development, w)

	log.SetLogger(logger)
	klog.SetLogger(logger)

	return logger
}

// newLogger builds a logger without registering it as a process-wide logger.
func newLogger(format LogFormat, stacktraceLevel zapcore.Level, development bool, w io.Writer) logr.Logger {
	var encoder zapcore.Encoder
	if format == LogFormatJSON {
		encoder = zapcore.NewJSONEncoder(uzap.NewProductionEncoderConfig())
	} else {
		encoder = zapcore.NewConsoleEncoder(uzap.NewDevelopmentEncoderConfig())
	}

	opts := zap.Options{
		Development:     development,
		DestWriter:      w,
		Encoder:         encoder,
		StacktraceLevel: uzap.NewAtomicLevelAt(stacktraceLevel),
	}

	return zap.New(zap.UseFlagOptions(&opts))
}

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

	"github.com/thurgauerkb/cascader/internal/flag"

	"github.com/go-logr/logr"
	uzap "go.uber.org/zap"
	zapcore "go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	EncoderJSON    string = "json"
	EncoderConsole string = "console"

	LevelInfo  string = "info"
	LevelError string = "error"
	LevelPanic string = "panic"
)

// InitLogging initializes logging based on provided configuration.
func InitLogging(flags flag.Options, w io.Writer) logr.Logger {
	logger := setupLogger(flags, w)

	log.SetLogger(logger)
	klog.SetLogger(logger)

	return logger
}

// setupLogger configures and returns a logr.Logger based on given configuration.
func setupLogger(flags flag.Options, w io.Writer) logr.Logger {
	opts := zap.Options{
		Development:     flags.LogDev,
		DestWriter:      w,
		Encoder:         encoder(flags.LogEncoder),
		StacktraceLevel: stacktraceLevel(flags.LogStacktraceLevel),
	}

	return zap.New(zap.UseFlagOptions(&opts))
}

// encoder returns the appropriate zapcore.Encoder based on name.
func encoder(name string) zapcore.Encoder {
	if name == EncoderConsole {
		return zapcore.NewConsoleEncoder(uzap.NewDevelopmentEncoderConfig())
	}

	return zapcore.NewJSONEncoder(uzap.NewProductionEncoderConfig())
}

// stacktraceLevel returns the appropriate zap.AtomicLevel based on the provided name.
func stacktraceLevel(level string) uzap.AtomicLevel {
	switch level {
	case LevelInfo:
		return uzap.NewAtomicLevelAt(uzap.InfoLevel)
	case LevelError:
		return uzap.NewAtomicLevelAt(uzap.ErrorLevel)
	default:
		return uzap.NewAtomicLevelAt(uzap.PanicLevel)
	}
}

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

	"github.com/thurgauerkb/cascader/internal/config"

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
func InitLogging(cfg config.Config, out io.Writer) (logr.Logger, error) {
	logger, err := setupLogger(cfg, out)
	if err != nil {
		return logr.Logger{}, err
	}

	log.SetLogger(logger)
	klog.SetLogger(logger)

	return logger, nil
}

// setupLogger configures and returns a logr.Logger based on given configuration.
func setupLogger(cfg config.Config, out io.Writer) (logr.Logger, error) {
	encoder, err := encoder(cfg.LogEncoder)
	if err != nil {
		return logr.Logger{}, err
	}

	stackLevel, err := stacktraceLevel(cfg.LogStacktraceLevel)
	if err != nil {
		return logr.Logger{}, err
	}

	opts := zap.Options{
		Development:     cfg.LogDev,
		DestWriter:      out,
		Encoder:         encoder,
		StacktraceLevel: stackLevel,
	}

	return zap.New(zap.UseFlagOptions(&opts)), nil
}

// encoder returns the appropriate zapcore.Encoder based on name.
func encoder(name string) (zapcore.Encoder, error) {
	switch name {
	case EncoderJSON:
		return zapcore.NewJSONEncoder(uzap.NewProductionEncoderConfig()), nil
	case EncoderConsole:
		return zapcore.NewConsoleEncoder(uzap.NewDevelopmentEncoderConfig()), nil
	default:
		return nil, fmt.Errorf("invalid log encoder: %q", name)
	}
}

// stacktraceLevel returns the appropriate zap.AtomicLevel based on the provided name.
func stacktraceLevel(level string) (uzap.AtomicLevel, error) {
	switch level {
	case LevelInfo:
		return uzap.NewAtomicLevelAt(uzap.InfoLevel), nil
	case LevelError:
		return uzap.NewAtomicLevelAt(uzap.ErrorLevel), nil
	case LevelPanic:
		return uzap.NewAtomicLevelAt(uzap.PanicLevel), nil
	default:
		return uzap.AtomicLevel{}, fmt.Errorf("invalid stacktrace level: %q", level)
	}
}

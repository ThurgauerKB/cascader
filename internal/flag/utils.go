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

package flag

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/pflag"
)

// must panics on err and is used to keep config assembly clean.
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// decorateUsageWithEnv adds (env: ENV_NAME) to all flags.
func decorateUsageWithEnv(fs *pflag.FlagSet, envPrefix string) {
	fs.VisitAll(func(f *pflag.Flag) {
		envName := strings.ToUpper(envPrefix + "_" + strings.ReplaceAll(f.Name, "-", "_"))

		// Only append if not already present
		if !strings.Contains(f.Usage, "(env:") {
			f.Usage = fmt.Sprintf("%s (env: %s)", f.Usage, envName)
		}
	})
}

// IsHelpRequested checks if the error is a HelpRequested sentinel and prints it.
func IsHelpRequested(err error, w io.Writer) bool {
	var helpErr *HelpRequested
	if errors.As(err, &helpErr) {
		fmt.Fprint(w, helpErr.Error()) // nolint:errcheck
		return true
	}
	return false
}

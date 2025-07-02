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
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMust(t *testing.T) {
	t.Parallel()

	t.Run("returns value if no error", func(t *testing.T) {
		t.Parallel()

		got := must("value", nil)
		assert.Equal(t, "value", got)
	})

	t.Run("panics if error is not nil", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic but got none")
			}
		}()
		_ = must("fail", errors.New("error"))
	})
}

func TestEnvDesc(t *testing.T) {
	t.Parallel()

	t.Run("appends env var to description", func(t *testing.T) {
		t.Parallel()
		got := envDesc("Enable debug logging", "HEARTBEATS_DEBUG")
		want := "Enable debug logging (env: HEARTBEATS_DEBUG)"
		assert.Equal(t, want, got)
	})

	t.Run("works with empty description", func(t *testing.T) {
		t.Parallel()
		got := envDesc("", "FOO")
		want := " (env: FOO)"
		assert.Equal(t, want, got)
	})

	t.Run("works with empty env var", func(t *testing.T) {
		t.Parallel()
		got := envDesc("Desc only", "")
		want := "Desc only (env: )"
		assert.Equal(t, want, got)
	})
}

func TestIsHelpRequested(t *testing.T) {
	t.Parallel()

	t.Run("returns true and writes message for HelpRequested error", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		helpMsg := "this is the help message\n"
		err := &HelpRequested{Message: helpMsg}

		ok := IsHelpRequested(err, buf)

		assert.True(t, ok)
		assert.Equal(t, helpMsg, buf.String())
	})

	t.Run("returns false and writes nothing for unrelated error", func(t *testing.T) {
		t.Parallel()

		buf := &bytes.Buffer{}
		err := errors.New("some other error")

		ok := IsHelpRequested(err, buf)

		assert.False(t, ok)
		assert.Equal(t, "", buf.String())
	})
}

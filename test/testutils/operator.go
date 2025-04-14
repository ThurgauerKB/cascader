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
	"context"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2" // nolint:staticcheck
	. "github.com/onsi/gomega"    // nolint:staticcheck
)

var (
	operatorCmd    *exec.Cmd
	operatorCancel context.CancelFunc
)

// StartOperatorWithFlags starts the operator process with the given flags and checks that it is ready.
func StartOperatorWithFlags(flags []string) {
	ctx, cancel := context.WithCancel(context.Background())
	operatorCancel = cancel

	cmd := exec.CommandContext(ctx, "../../bin/cascader", flags...)
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// redirect output so ic an be captured
	output := io.MultiWriter(LogBuffer, GinkgoWriter)
	cmd.Stdout = output
	cmd.Stderr = output

	Expect(cmd.Start()).To(Succeed())
	operatorCmd = cmd

	// Wait until Operator is ready
	CountLogOccurrences("\"worker count\":1", 3, 1*time.Minute, 2*time.Second)
}

// StopOperator stops the operator process.
func StopOperator() {
	if operatorCancel != nil {
		operatorCancel()
	}

	if operatorCmd != nil && operatorCmd.Process != nil {
		_ = syscall.Kill(-operatorCmd.Process.Pid, syscall.SIGKILL)
		operatorCmd.Wait() // nolint:errcheck
	}
}

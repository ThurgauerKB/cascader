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

package main

import (
	"os"

	"github.com/thurgauerkb/cascader/internal/app"

	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	Version string = "dev"
	Commit  string = "none"
)

func main() {
	ctx := ctrl.SetupSignalHandler()

	if err := app.Run(ctx, Version, os.Args[1:], os.Stdout); err != nil {
		os.Exit(1)
	}
}

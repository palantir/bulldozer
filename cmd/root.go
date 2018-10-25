// Copyright 2018 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/spf13/cobra"
)

var rootConfig struct {
	Debug bool
}

func IsDebugMode() bool {
	return rootConfig.Debug
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:          "bulldozer",
	Short:        "A bot for auto-merging PRs when all status checks are green and the PR is reviewed",
	Long:         "A bot for auto-merging PRs when all status checks are green and the PR is reviewed",
	SilenceUsage: true,
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&rootConfig.Debug, "debug", "d", false, "enables debug output")
}

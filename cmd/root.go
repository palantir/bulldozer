// Copyright 2017 Palantir Technologies, Inc.
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
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/palantir/bulldozer/version"
)

var (
	cfgFile     string
	versionFlag bool
)

var RootCmd = &cobra.Command{
	Use:   "bulldozer",
	Short: "Auto merge bot",
	Long:  "Bot that merges pull requests when they are green and reviewed",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if versionFlag {
			log.Info(version.Version())
			os.Exit(0)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("No command provided")
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "bulldozer.yml", "config file")
	err := viper.BindPFlag("verbose", RootCmd.PersistentFlags().Lookup("verbose"))
	if err != nil {
		log.Fatal(errors.Wrap(err, "Cannot bind verbose flag"))
	}

	RootCmd.PersistentFlags().BoolVar(&versionFlag, "version", false, "Print version and exit")
}

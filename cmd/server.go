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
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/palantir/bulldozer/auth"
	"github.com/palantir/bulldozer/server"
	"github.com/palantir/bulldozer/server/config"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run Bulldozer as a server",
	Long:  "Run Bulldozer in server mode, the only mode.",
	Run: func(cmd *cobra.Command, args []string) {
		data, err := ioutil.ReadFile(cfgFile)
		if err != nil {
			log.Fatalf("Cannot read config file: %+v", err)
		}
		startup, err := config.Parse(data)
		if err != nil {
			log.Fatalf("Failed to parse config file: %+v", err)
		}

		if startup.LogLevel() == log.DebugLevel {
			cmd.DebugFlags()
			viper.Debug()
		}

		log.SetFormatter(&log.JSONFormatter{})
		db, err := server.InitDB(startup.Database)
		if err != nil {
			log.Fatalf("Cannot init db: %+v", err)
		}

		auth.GitHubOAuth(startup.Github)

		srv := server.New(db, startup)
		if err := srv.SetupSessionStore(); err != nil {
			log.Fatal("Cannot setup session store")
		}

		if err := srv.Start(); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	RootCmd.AddCommand(serverCmd)
}

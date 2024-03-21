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

package server

import (
	"fmt"

	"github.com/c2h5oh/datasize"
	"github.com/die-net/lrucache"
	"github.com/gregjones/httpcache"
	"github.com/palantir/bulldozer/server/handler"
	"github.com/palantir/bulldozer/version"
	"github.com/palantir/go-baseapp/baseapp"
	"github.com/palantir/go-baseapp/baseapp/datadog"
	"github.com/palantir/go-githubapp/appconfig"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"goji.io/pat"
)

type Server struct {
	config *Config
	base   *baseapp.Server
}

// New instantiates a new Server.
// Callers must then invoke Start to run the Server.
func New(c *Config) (*Server, error) {
	logger := baseapp.NewLogger(baseapp.LoggingConfig{
		Level:  c.Logging.Level,
		Pretty: c.Logging.Text,
	})

	serverParams := baseapp.DefaultParams(logger, c.Options.AppName+".")
	base, err := baseapp.NewServer(c.Server, serverParams...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize base server")
	}

	maxSize := int64(50 * datasize.MB)
	if c.Cache.MaxSize != 0 {
		maxSize = int64(c.Cache.MaxSize)
	}

	userAgent := fmt.Sprintf("%s/%s", c.Options.AppName, version.GetVersion())
	clientCreator, err := githubapp.NewDefaultCachingClientCreator(
		c.Github,
		githubapp.WithClientUserAgent(userAgent),
		githubapp.WithClientCaching(true, func() httpcache.Cache { return lrucache.New(maxSize, 0) }),
		githubapp.WithClientMiddleware(
			githubapp.ClientLogging(zerolog.DebugLevel),
			githubapp.ClientMetrics(base.Registry()),
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize Github client creator")
	}

	configPaths := []string{c.Options.ConfigurationPath}
	for _, p := range c.Options.ConfigurationV0Paths {
		if p != c.Options.ConfigurationPath {
			configPaths = append(configPaths, p)
		}
	}

	baseHandler := handler.Base{
		ClientCreator: clientCreator,
		ConfigFetcher: handler.NewConfigFetcher(
			appconfig.NewLoader(
				configPaths,
				appconfig.WithOwnerDefault(c.Options.SharedRepository, []string{
					c.Options.SharedConfigurationPath,
				}),
			),
			c.Options.DefaultRepositoryConfig,
		),

		PushRestrictionUserToken: c.Options.PushRestrictionUserToken,
		DisableUpdateFeature:     c.Options.DisableUpdateFeature,
	}

	queueSize := c.Workers.QueueSize
	if queueSize < 1 {
		queueSize = 100
	}

	workers := c.Workers.Workers
	if workers < 1 {
		workers = 10
	}

	webhookHandler := githubapp.NewEventDispatcher(
		[]githubapp.EventHandler{
			&handler.CheckRun{Base: baseHandler},
			&handler.IssueComment{Base: baseHandler},
			&handler.PullRequest{Base: baseHandler},
			&handler.PullRequestReview{Base: baseHandler},
			&handler.Push{Base: baseHandler},
			&handler.Status{Base: baseHandler},
		},
		c.Github.App.WebhookSecret,
		githubapp.WithErrorCallback(githubapp.MetricsErrorCallback(base.Registry())),
		githubapp.WithScheduler(
			githubapp.QueueAsyncScheduler(
				queueSize, workers,
				githubapp.WithSchedulingMetrics(base.Registry()),
				githubapp.WithAsyncErrorCallback(githubapp.MetricsAsyncErrorCallback(base.Registry())),
			),
		),
	)

	mux := base.Mux()

	// webhook route
	mux.Handle(pat.Post(githubapp.DefaultWebhookRoute), webhookHandler)

	// any additional API routes
	mux.Handle(pat.Get("/api/health"), handler.Health())
	mux.Handle(pat.Get("/api/metrics"), handler.Metrics(base.Registry(), c.Prometheus))

	return &Server{
		config: c,
		base:   base,
	}, nil
}

// Start is blocking and long-running
func (s *Server) Start() error {
	if s.config.Datadog.Address != "" {
		if err := datadog.StartEmitter(s.base, s.config.Datadog); err != nil {
			return err
		}
	}
	return s.base.Start()
}

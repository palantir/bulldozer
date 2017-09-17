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

package config

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/yaml.v2"
)

const (
	EchoLoggingFormat = `{"time":"${time_rfc3339_nano}","id":"${id}","remote_ip":"${remote_ip}","x_forwarded_for":"${header:X-Forwarded-For}",host":"${host}",` +
		`"method":"${method}","uri":"${uri}","status":${status}, "latency":${latency},` +
		`"latency_human":"${latency_human}","bytes_in":${bytes_in},` +
		`"bytes_out":${bytes_out}}` + "\n"
)

var (
	validate *validator.Validate

	Instance *Startup

	DoNotMerge     = []string{"do not merge", "wip", "do-not-merge", "do_not_merge"}
	MergeWhenReady = []string{"merge when ready", "merge-when-ready", "merge_when_ready"}
	UpdateMe       = []string{"update me", "update-me", "update_me"}
)

type Startup struct {
	Server   Rest            `yaml:"rest" validate:"required"`
	Logging  LoggingConfig   `yaml:"logging" validate:"required,dive"`
	Database *DatabaseConfig `yaml:"database" validate:"required,dive"`
	Github   *GithubConfig   `yaml:"github" validate:"required,dive"`
	AssetDir string          `yaml:"assetDir" validate:"required"`
}

type Rest struct {
	Address string `yaml:"address" validate:"required"`
	Port    int    `yaml:"port" validate:"required"`
}

type LoggingConfig struct {
	Level string `yaml:"level" validate:"required"`
}

type DatabaseConfig struct {
	DBName   string `yaml:"name" validate:"required"`
	Host     string `yaml:"host" validate:"required"`
	Username string `yaml:"username" validate:"required"`
	Password string `yaml:"password" validate:"required"`
	SSLMode  string `yaml:"sslmode" validate:"required"`
}

type GithubConfig struct {
	Address       string `yaml:"address" validate:"required"`
	APIURL        string `yaml:"apiURL" validate:"required"`
	CallbackURL   string `yaml:"callbackURL" validate:"required"`
	ClientID      string `yaml:"clientID" validate:"required"`
	ClientSecret  string `yaml:"clientSecret" validate:"required"`
	WebHookURL    string `yaml:"webhookURL" validate:"required"`
	WebhookSecret string `yaml:"webhookSecret" validate:"required"`
}

func Parse(bytes []byte) (*Startup, error) {
	var c Startup
	err := yaml.Unmarshal(bytes, &c)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling yaml")
	}

	err = validateConfig(&c)
	if err != nil {
		return nil, errors.Wrapf(err, "failed basic config validation")
	}

	return &c, nil
}

func (s *Startup) LogLevel() logrus.Level {
	level, _ := logrus.ParseLevel(s.Logging.Level)
	return level
}

func validateConfig(c *Startup) error {
	if err := validate.Struct(c); err != nil {
		return errors.Wrapf(err, "failed validating struct")
	}

	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "require"
	}

	if err := validate.Var(c.Database.SSLMode, "sslmode"); err != nil {
		return errors.Wrapf(err, "invalid sslmode")
	}

	if err := validate.Var(c.Logging.Level, "level"); err != nil {
		return errors.Wrapf(err, "invalid log level")
	}

	return nil
}

func in(test string, strings []string) bool {
	for _, s := range strings {
		if s == test {
			return true
		}
	}
	return false
}

func isSSLMode(fl validator.FieldLevel) bool {
	allowedModes := []string{"disable", "require", "verify-ca", "verify-full"}
	return in(fl.Field().String(), allowedModes)
}

func isAllowedLogLevel(fl validator.FieldLevel) bool {
	allowedLevels := []string{"debug", "info", "warning", "error", "fatal", "panic"}
	return in(fl.Field().String(), allowedLevels)
}

func init() {
	validate = validator.New()
	_ = validate.RegisterValidation("level", isAllowedLogLevel)
	_ = validate.RegisterValidation("sslmode", isSSLMode)
}

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
	"os"

	"github.com/c2h5oh/datasize"
	"github.com/palantir/bulldozer/bulldozer"
	"github.com/palantir/go-baseapp/baseapp"
	"github.com/palantir/go-baseapp/baseapp/datadog"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	DefaultEnvPrefix = "BULLDOZER_"
)

type Config struct {
	Server  baseapp.HTTPConfig `yaml:"server"`
	Github  githubapp.Config   `yaml:"github"`
	Options Options            `yaml:"options"`
	Logging LoggingConfig      `yaml:"logging"`
	Datadog datadog.Config     `yaml:"datadog"`
	Cache   CacheConfig        `yaml:"cache"`
	Workers WorkerConfig       `yaml:"workers"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
	Text  bool   `yaml:"text"`
}

type CacheConfig struct {
	MaxSize datasize.ByteSize `yaml:"max_size"`
}

type WorkerConfig struct {
	Workers   int `yaml:"workers"`
	QueueSize int `yaml:"queue_size"`
}

type Options struct {
	AppName                  string            `yaml:"app_name"`
	ConfigurationPath        string            `yaml:"configuration_path"`
	DefaultRepositoryConfig  *bulldozer.Config `yaml:"default_repository_config"`
	PushRestrictionUserToken string            `yaml:"push_restriction_user_token"`

	ConfigurationV0Paths []string `yaml:"configuration_v0_paths"`
}

func ParseConfig(bytes []byte) (*Config, error) {
	var c Config
	if err := yaml.UnmarshalStrict(bytes, &c); err != nil {
		return nil, errors.Wrapf(err, "failed unmarshaling yaml")
	}

	c.Github.SetValuesFromEnv("")

	envPrefix := DefaultEnvPrefix
	if v, ok := os.LookupEnv("BULLDOZER_ENV_PREFIX"); ok {
		envPrefix = v
	}
	c.Server.SetValuesFromEnv(envPrefix)

	if v, ok := os.LookupEnv(envPrefix + "PUSH_RESTRICTION_USER_TOKEN"); ok {
		c.Options.PushRestrictionUserToken = v
	}

	return &c, nil
}

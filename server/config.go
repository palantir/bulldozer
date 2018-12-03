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
	"github.com/palantir/go-baseapp/baseapp"
	"github.com/palantir/go-baseapp/baseapp/datadog"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/palantir/bulldozer/bulldozer"
)

const (
	DefaultAppName             = "bulldozer"
	DefaultConfigurationV1Path = ".bulldozer.v1.yml"
)

type Config struct {
	Server  baseapp.HTTPConfig `yaml:"server"`
	Github  githubapp.Config   `yaml:"github"`
	Options Options            `yaml:"options"`
	Logging LoggingConfig      `yaml:"logging"`
	Datadog datadog.Config     `yaml:"datadog"`
	Global  bulldozer.Config   `yaml:"global"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
	Text  bool   `yaml:"text"`
}

type Options struct {
	AppName              string   `yaml:"app_name"`
	ConfigurationPath    string   `yaml:"configuration_path"`
	EnableGlobalConfig   bool     `yaml:"enable_global_config"`
	ConfigurationV0Paths []string `yaml:"configuration_v0_paths"`
}

func (o *Options) fillDefaults() {
	if o.AppName == "" {
		o.AppName = DefaultAppName
	}

	if o.ConfigurationPath == "" {
		o.ConfigurationPath = DefaultConfigurationV1Path
	}
}

func ParseConfig(bytes []byte) (*Config, error) {
	var c Config
	err := yaml.UnmarshalStrict(bytes, &c)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshaling yaml")
	}

	return &c, nil
}

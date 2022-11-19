// Copyright 2022 Palantir Technologies, Inc.
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

package handler

import (
	"github.com/palantir/bulldozer/bulldozer"
	"os"
	"strconv"
)

const (
	DefaultSharedRepository = ".github"
	// DefaultConfigurationPath
	// The default configuration path is the same for the repo as
	// for the shared config if not configured
	DefaultConfigurationPath = "bulldozer.yml"
	DefaultAppName           = "bulldozer"
)

type Options struct {
	AppName                  string `yaml:"app_name"`
	PushRestrictionUserToken string `yaml:"push_restriction_user_token"`

	ConfigurationPath       string            `yaml:"configuration_path"`
	SharedRepository        string            `yaml:"shared_repository"`
	SharedConfigurationPath string            `yaml:"shared_configuration_path"`
	DefaultRepositoryConfig *bulldozer.Config `yaml:"default_repository_config"`

	ConfigurationV0Paths []string `yaml:"configuration_v0_paths"`

	DisableUpdateFeature bool `yaml:"disable_update_feature"`
}

func (o *Options) fillDefaults() {
	if o.ConfigurationPath == "" {
		o.ConfigurationPath = DefaultConfigurationPath
	}
	if o.AppName == "" {
		o.AppName = DefaultAppName
	}
	if o.SharedRepository == "" {
		o.SharedRepository = DefaultSharedRepository
	}
	if o.SharedConfigurationPath == "" {
		o.SharedConfigurationPath = DefaultConfigurationPath
	}
}

func (o *Options) SetValuesFromEnv(prefix string) {
	setStringFromEnv("CONFIGURATION_PATH", prefix, &o.ConfigurationPath)
	setStringFromEnv("APP_NAME", prefix, &o.AppName)
	setStringFromEnv("SHARED_REPOSITORY", prefix, &o.SharedRepository)
	setStringFromEnv("SHARED_CONFIGURATION_PATH", prefix, &o.SharedConfigurationPath)
	setBooleanFromEnv("DISABLE_UPDATE_FEATURE", prefix, &o.DisableUpdateFeature)
	setStringFromEnv("PUSH_RESTRICTION_USER_TOKEN", prefix, &o.PushRestrictionUserToken)
	o.fillDefaults()
}

func setStringFromEnv(key, prefix string, value *string) bool {
	if v, ok := os.LookupEnv(prefix + key); ok {
		*value = v
		return true
	}
	return false
}

func setBooleanFromEnv(key, prefix string, value *bool) bool {
	if v, ok := os.LookupEnv(prefix + key); ok {
		*value, _ = strconv.ParseBool(v)
		return true
	}
	return false
}

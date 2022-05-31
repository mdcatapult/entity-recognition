/*
 * Copyright 2022 Medicines Discovery Catapult
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lib

import (
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const configFlag = "config"

type BaseConfig struct {
	LogLevel string `mapstructure:"log_level"`
}

/**
	InitializeConfig standardises config initialization across all apps.

	Usage:

	Config can be specified in a yml file. By default this is located at the defaultPath argument, but can be overridden
	with the --config flag, which should contain a filepath. For example, if defaultPath is "./config/dictionary.yml",
	then a k8s config map with a dictionary.yml key could be mounted to $(pwd)/config so that the config map
	is available at the path.

	Keys which exist on defaultConfig but NOT on the config yaml will also be used.

	Env vars can be used to overwrite config keys IF the config yaml is empty OR the env var has the same name
	as the key you want to overwrite (the env var must be uppercased).

	Args:
	defaultPath is the default relative
	or absolute path to the config file. This is overridden with the --config flag.

	defaultConfig is the default config, defined as a map[string]interface{} within the code itself.
	It should be defined close to the "main" function and should be set up for local development.

	targetStruct should be a pointer to a struct which the config can be unmarshalled to.
**/

func InitializeConfig(defaultPath string, defaultConfig map[string]interface{}, targetStruct interface{}) error {

	// load the config flag argument into viper
	pflag.String(configFlag, defaultPath, "The config file path.")
	pflag.Parse()

	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		return err
	}

	// load the config filepath from viper
	configFile := viper.GetString("config")

	if !filepath.IsAbs(configFile) {
		configFile, err = filepath.Abs(configFile)
		if err != nil {
			return err
		}
	}

	// set viper's default config using defaultConfig
	for k, v := range defaultConfig {
		viper.SetDefault(k, v)
	}

	// set the name for the config file
	viper.SetConfigName(strings.TrimSuffix(filepath.Base(configFile), filepath.Ext(configFile)))
	viper.AddConfigPath(filepath.Dir(configFile))

	// tell viper to prefer env vars over config keys. An env var must ALSO exist as a key in
	// viper's config for viper to be able to read the env var.
	viper.AutomaticEnv()

	// rewrite env var names to use "_" instead of "." when reading env vars
	// this means that the env var DICTIONARY_FORMAT is used in the config struct as Dictionary.Format
	repl := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(repl)

	// now we are ready to read the config into viper - we have told it where to look for it with `addConfigPath()`
	err = viper.ReadInConfig()
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		log.Warn().Err(err).Msg("default settings applied")
	} else if err != nil {
		return err
	}

	var bc BaseConfig
	err = viper.Unmarshal(&bc)
	if err != nil {
		return err
	}

	lvl, err := zerolog.ParseLevel(bc.LogLevel)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(lvl)

	// unmarshal config into struct
	if err := viper.Unmarshal(targetStruct); err != nil {
		return err
	}

	return nil
}

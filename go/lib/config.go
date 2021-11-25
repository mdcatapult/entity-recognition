package lib

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type BaseConfig struct {
	LogLevel string `mapstructure:"log_level"`
}

/**
	InitializeConfig standardises config initialization across all apps. defaultPath is the default relative
	or absolute path to the config file. This is overridden with the --config flag.

	defaultConfig is the default config, defined as a map[string]interface{} within the code itself.
	It should be defined close to the "main" function and should be set up for local development.

	targetStruct should be a pointer to a struct which the config can be unmarshalled to.

	Keys which exist in the config map will be overridden by env vars which have the same, but capitalised name.
	e.g. if "myKey" is in the config map, it will be overridden by $MYKEY.
	Therefore for env vars to work, either they must already have a corresponding key in the config map OR config
	map must be empty.
**/

func InitializeConfig(defaultPath string, defaultConfig map[string]interface{}, targetStruct interface{}) error {

	// load the --config flag argument into viper
	pflag.String("config", defaultPath, "The config file path.")
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

	// tell viper to prefer env vars over config keys. An env var must ALSO exist as a key in the
	// config map for viper to read it.
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

	fmt.Println(viper.AllSettings())
	fmt.Println(targetStruct)

	return nil
}

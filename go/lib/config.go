package lib

import (
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

// InitializeConfig standardises config initialization across all apps. defaultPath is the default relative
// or absolute path to the config file. This is overriden with the --config flag. defaultConfig is the default
// config, defined as a map[string]interface{} within the code itself. It should be defined close to the "main"
// function and should be set up for local development. targetStruct should be a pointer to a struct which the
// config can be unmarshalled to.
func InitializeConfig(defaultPath string, defaultConfig map[string]interface{}, targetStruct interface{}) error {
	pflag.String("config", defaultPath, "The config file path.")
	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		return err
	}

	configFile := viper.GetString("config")
	if !filepath.IsAbs(configFile) {
		configFile, err = filepath.Abs(configFile)
		if err != nil {
			return err
		}
	}

	for k, v := range defaultConfig {
		viper.SetDefault(k, v)
	}

	viper.SetConfigName(strings.TrimSuffix(filepath.Base(configFile), filepath.Ext(configFile)))
	viper.AddConfigPath(filepath.Dir(configFile))
	viper.AutomaticEnv()
	repl := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(repl)
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

	if err := viper.Unmarshal(targetStruct); err != nil {
		return err
	}

	return nil
}

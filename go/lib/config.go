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

// InitializeConfig standardises config initialization across all apps.
func InitializeConfig(defaultPath string, defaults map[string]interface{}) error {
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

	for k, v := range defaults {
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

	return nil
}

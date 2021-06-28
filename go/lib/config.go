package lib

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// InitializeConfig standardises config initialization across all apps.
func InitializeConfig(defaults map[string]interface{}) error {
	pflag.String("config", "config.yml", "The config file path.")
	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		return err
	}

	configFile := viper.GetString("config")
	if !filepath.IsAbs(configFile) {
		_, callingFile, _, _ := runtime.Caller(1)
		callersDirectory := path.Dir(callingFile)
		configFile, err = filepath.Abs(path.Join(callersDirectory, configFile))
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
		log.Fatal().Err(err).Send()
	}
	return nil
}
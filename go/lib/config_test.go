package lib

import (
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"testing"
)

type config struct {
	ConfigKey1 string
	ConfigKey2 struct {
		ConfigKey3 string
	}
	KeyNotInConfigMap string
}

var (
	configValue1 = "configValue1"
	configValue3 = "configValue3"
	configFileName string
)

func TestMain(m *testing.M) {
	configMap := map[string]interface{}{
		"configkey1": configValue1,
		"configkey2": map[string]interface{}{
			"configkey3": configValue3,
		},
	}

	file, err := ioutil.TempFile(".", "*.yml")
	if err != nil {
		panic(err)
	}
	configFileName = file.Name()

	data, err := yaml.Marshal(&configMap)
	if err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(configFileName, data, 0); err != nil {
		panic(err)
	}

	code := m.Run()
	os.Remove(configFileName)
	os.Exit(code)
}

func TestInitializeConfigFromPath(t *testing.T) {
	var parsedConfig config
	err := InitializeConfig(configFileName, map[string]interface{}{}, &parsedConfig)

	assert.NoError(t, err)
	assert.Equal(t, "configValue1", parsedConfig.ConfigKey1)
	assert.Equal(t, "configValue3", parsedConfig.ConfigKey2.ConfigKey3)
}

func TestInitializeConfigEnvOverride(t *testing.T) {
	overrideValue := "anewvalue"
	os.Setenv("CONFIGKEY1", overrideValue)
	os.Setenv("CONFIGKEY2_CONFIGKEY3", overrideValue)
	os.Setenv("KEYNOTINCONFIGMAP", overrideValue)

	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	var parsedConfig config
	err := InitializeConfig(configFileName, map[string]interface{}{}, &parsedConfig)

	assert.NoError(t, err)
	assert.Equal(t, overrideValue, parsedConfig.ConfigKey1)
	assert.Equal(t, overrideValue, parsedConfig.ConfigKey2.ConfigKey3)

	// If an env var does not exist in the config map, viper will not parse it
	assert.Equal(t, "", parsedConfig.KeyNotInConfigMap)
}

func TestInitializeConfigEmptyPath(t *testing.T) {
	overrideValue := "some value"
	os.Setenv("CONFIGKEY1", overrideValue)
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	var parsedConfig config
	err := InitializeConfig("", map[string]interface{}{}, &parsedConfig)
	assert.NoError(t, err)

	// when config path is empty, viper will listen to env vars
	assert.Equal(t, overrideValue, parsedConfig.ConfigKey1)
}

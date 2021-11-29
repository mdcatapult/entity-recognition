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

	filename, err := createConfigFile(configMap, ".", "*.yml")
	if err != nil {
		panic(err)
	}

	code := m.Run()
	os.Remove(filename)
	os.Exit(code)
}

func TestInitializeConfigFromPath(t *testing.T) {
	resetFlags()

	var parsedConfig config
	err := InitializeConfig(configFileName, map[string]interface{}{}, &parsedConfig)

	assert.NoError(t, err)
	assert.Equal(t, configValue1, parsedConfig.ConfigKey1)
	assert.Equal(t, configValue3, parsedConfig.ConfigKey2.ConfigKey3)
}

func TestInitializeConfigEnvOverride(t *testing.T) {
	resetFlags()

	overrideValue := "anewvalue"
	os.Setenv("CONFIGKEY1", overrideValue)
	os.Setenv("CONFIGKEY2_CONFIGKEY3", overrideValue)
	os.Setenv("KEYNOTINCONFIGMAP", overrideValue)

	var parsedConfig config
	err := InitializeConfig(configFileName, map[string]interface{}{}, &parsedConfig)

	assert.NoError(t, err)
	assert.Equal(t, overrideValue, parsedConfig.ConfigKey1)
	assert.Equal(t, overrideValue, parsedConfig.ConfigKey2.ConfigKey3)

	// If an env var does not exist in the config map, viper will not parse it
	assert.Equal(t, "", parsedConfig.KeyNotInConfigMap)
}

func TestInitializeConfigEmptyPath(t *testing.T) {
	resetFlags()

	overrideValue := "some value"
	os.Setenv("CONFIGKEY1", overrideValue)

	var parsedConfig config
	err := InitializeConfig("", map[string]interface{}{}, &parsedConfig)
	assert.NoError(t, err)

	// when config path is empty, viper will listen to env vars
	assert.Equal(t, overrideValue, parsedConfig.ConfigKey1)

	os.Unsetenv("CONFIGKEY1")
}

func TestInitializeConfigWithFlag(t *testing.T) {
	resetFlags()

	overrideConfigPath := "*.yml"
	pflag.Set(configFlag, overrideConfigPath)
	overrideValue := "this is overridden!"
	overrideConfigMap := map[string]interface{}{
		"configkey1": overrideValue,
	}

	filename, err := createConfigFile(overrideConfigMap, ".", overrideConfigPath)
	if err != nil {
		panic(err)
	}

	var parsedConfig config
	err = InitializeConfig(configFileName, map[string]interface{}{}, &parsedConfig)

	assert.NoError(t, err)
	assert.Equal(t, overrideValue, parsedConfig.ConfigKey1)

	os.Remove(filename)
}

func createConfigFile(configMap map[string]interface{}, path, name string) (fileName string, err error) {
	file, err := ioutil.TempFile(path, name)
	if err != nil {
		return"", err
	}
	configFileName = file.Name()

	data, err := yaml.Marshal(&configMap)
	if err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(configFileName, data, 0); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func resetFlags() {
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
}

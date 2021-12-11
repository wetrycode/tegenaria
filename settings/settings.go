package settings

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

type Settings interface {
	GetValue(key string) (error, string)
}

type Logger struct {
	Path  string `yaml:"path"`
	Level string `yaml:"level"`
}

type Configuration struct {
	Log *Logger `ymal:"log"`
}

func (l *Logger) GetValue(key string) (string, error) {
	return "", nil
}

var Config Configuration

func init() {
	_, filename, _, _ := runtime.Caller(0)

	runtimeViper := viper.New()

	runtimeViper.AddConfigPath(filepath.Dir(filepath.Dir(filename)))
	runtimeViper.SetConfigName("settings")
	runtimeViper.SetConfigType("yaml")
	readErr := runtimeViper.ReadInConfig()
	if readErr != nil {
		panic(fmt.Errorf("fatal error config file: %s", readErr))
	}
	err:=runtimeViper.Unmarshal(&Config)
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", readErr))
	}
}

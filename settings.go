package tegenaria

import (
	"fmt"
	"os"

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

var Config *Configuration = &Configuration{
	Log: &Logger{
		Path:  "/var/log",
		Level: "info",
	},
}

func load() bool {
	defer func() {
		if p := recover(); p != nil {
			fmt.Printf("fatal error config file: %s", p)
		}
	}()
	str, _ := os.Getwd()
	runtimeViper := viper.New()

	runtimeViper.AddConfigPath(str)
	runtimeViper.SetConfigName("settings")
	runtimeViper.SetConfigType("yaml")
	readErr := runtimeViper.ReadInConfig()
	if readErr != nil {
		return false
	}
	err := runtimeViper.Unmarshal(Config)
	return err == nil
}
func initSettings() {
	if !load() {
		Config = &Configuration{
			Log: &Logger{
				Path:  "/var/log",
				Level: "info",
			},
		}
	}
	// _, filename, _, _ := runtime.Caller(1)
	// str, _ := os.Getwd()
	// runtimeViper := viper.New()

	// runtimeViper.AddConfigPath(str)
	// runtimeViper.SetConfigName("settings")
	// runtimeViper.SetConfigType("yaml")
	// readErr := runtimeViper.ReadInConfig()
	// if readErr != nil {
	// 	panic(fmt.Errorf("fatal error config file: %s", readErr))
	// }
	// err := runtimeViper.Unmarshal(&Config)
	// if err != nil {
	// 	panic(fmt.Errorf("fatal error config file: %s", readErr))
	// }
}

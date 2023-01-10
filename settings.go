// Copyright 2022 geebytes
// Licensed under the Apache License, Version 2.0 (the 'License');
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an 'AS IS' BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tegenaria

import (
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

var Config *Configuration = &Configuration{
	Log: &Logger{
		Path:  "/var/log",
		Level: "info",
	},
}

func load() bool {
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
}

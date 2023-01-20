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
	"path"
	"runtime"
	"sync"

	"github.com/spf13/viper"
)

type Settings interface {
	// GetValue 获取指定的参数值
	GetValue(key string) (interface{},error)
}


type Configuration struct {
	// Log *Logger `ymal:"log"`
	*viper.Viper
}

var onceConfig sync.Once
var Config *Configuration = nil

func newTegenariaConfig() {
	onceConfig.Do(func() {
		Config = &Configuration{
			viper.New(),
		}
	})

}
func(c *Configuration)GetValue(key string) (interface{},error){
	value:=c.Get(key)
	return value,nil
}
func (c *Configuration) load(dir string) bool {
	c.AddConfigPath(dir)
	c.SetConfigName("settings")
	c.SetConfigType("yaml")
	readErr := c.ReadInConfig()
	if readErr != nil {
		return false
	}
	err := c.Unmarshal(c)
	return err == nil
}
func initSettings() {
	newTegenariaConfig()
	wd, _ := os.Getwd()
	var abPath string

	_, filename, _, ok := runtime.Caller(0)
	if ok {
		abPath = path.Dir(filename)

	}
	Config.load(wd)
	Config.load(abPath)

}

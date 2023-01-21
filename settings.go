// MIT License

// Copyright (c) 2023 wetrycode

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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

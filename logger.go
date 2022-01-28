// Copyright 2022 vforfreedom96@gmail.com
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
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger = logrus.New()
var ProcessId string = uuid.New().String()

type DefaultFieldHook struct {
}

func (hook *DefaultFieldHook) Fire(entry *logrus.Entry) error {
	// u4 := uuid.New()
	name, _ := os.Hostname()
	// entry.Data["uuid"] = u4.String()
	entry.Data["hostname"] = name
	// entry.Data["function"] = entry.Caller.Function
	return nil
}

func (hook *DefaultFieldHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
func GetLogger(Name string) *logrus.Entry {

	log := logger.WithFields(logrus.Fields{
		"logName": Name,
	})

	return log
}
func initLog() {
	// 设置路径
	logger.SetReportCaller(true)

	logger.SetOutput(os.Stdout)
	_, ex := os.LookupEnv("UNITTEST")
	logLevel := Config.Log.Level
	if ex {
		logLevel = "error"
	}
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		panic(fmt.Errorf("fatal error parse level: %s", err))
	}
	logger.SetFormatter(&logrus.TextFormatter{
		ForceQuote:      true,                  //键值对加引号
		TimestampFormat: "2006-01-02 15:04:05", //时间格式
		FullTimestamp:   true,
	})
	logger.SetLevel(logrus.Level(level))
	// logger.Hooks.Add(lfshook.NewHook(pathMap, logger.Formatter))
	logger.Hooks.Add(&DefaultFieldHook{})
}

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
	"fmt"
	"os"
	"strings"

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
	logLevel := Config.GetString("log.level")
	if ex {
		logLevel = "error"
	}
	logLevel = strings.TrimSpace(logLevel)
	if logLevel == "" {
		logLevel = "info"
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

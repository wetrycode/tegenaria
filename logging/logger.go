package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/geebytes/Tegenaria/settings"

	"github.com/google/uuid"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
)

var Logger = logrus.New()
var ProcessId string = uuid.New().String()

type DefaultFieldHook struct {
}

func (hook *DefaultFieldHook) Fire(entry *logrus.Entry) error {
	u4 := uuid.New()
	name, _ := os.Hostname()
	entry.Data["appName"] = "Tegenaria"
	entry.Data["uuid"] = u4.String()
	entry.Data["hostname"] = name
	entry.Data["processId"] = ProcessId
	return nil
}

func (hook *DefaultFieldHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
func GetLogger(loggerName string) *logrus.Entry {
	log := Logger.WithFields(logrus.Fields{
		"logName": loggerName,
	})
	return log
}
func init() {
	// 设置路径
	pathMap := lfshook.PathMap{
		logrus.InfoLevel:  filepath.Join(settings.Config.Log.Path, "info.log"),
		logrus.PanicLevel: filepath.Join(settings.Config.Log.Path, "error.log"),
		logrus.ErrorLevel: filepath.Join(settings.Config.Log.Path, "error.log"),
		logrus.FatalLevel: filepath.Join(settings.Config.Log.Path, "error.log"),
		logrus.DebugLevel: filepath.Join(settings.Config.Log.Path, "debug.log"),
	}
	// 设置日志格式为json格式
	Logger.Formatter = &logrus.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05"}
	Logger.SetOutput(os.Stdout)
	level, err := logrus.ParseLevel(settings.Config.Log.Level)
	if err != nil {
		panic(fmt.Errorf("fatal error parse level: %s", err))
	}
	Logger.SetLevel(logrus.Level(level))
	Logger.Hooks.Add(lfshook.NewHook(pathMap, &logrus.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05"}))
	Logger.Hooks.Add(&DefaultFieldHook{})
}

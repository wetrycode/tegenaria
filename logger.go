package tegenaria

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/google/uuid"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger = logrus.New()
var ProcessId string = uuid.New().String()

type DefaultFieldHook struct {
}

func (hook *DefaultFieldHook) Fire(entry *logrus.Entry) error {
	u4 := uuid.New()
	name, _ := os.Hostname()
	entry.Data["uuid"] = u4.String()
	entry.Data["hostname"] = name
	entry.Data["function"] = entry.Caller.Function
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

	pathMap := lfshook.PathMap{
		logrus.InfoLevel:  filepath.Join(Config.Log.Path, "info.log"),
		logrus.PanicLevel: filepath.Join(Config.Log.Path, "error.log"),
		logrus.ErrorLevel: filepath.Join(Config.Log.Path, "error.log"),
		logrus.FatalLevel: filepath.Join(Config.Log.Path, "error.log"),
		logrus.DebugLevel: filepath.Join(Config.Log.Path, "debug.log"),
	}
	// 设置日志格式为json格式
	logger.Formatter = &logrus.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05", CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
		fileName := path.Base(frame.File) + ":" + strconv.Itoa(frame.Line)
		//return frame.Function, fileName
		return "", fileName
	}}

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
	logger.SetLevel(logrus.Level(level))
	logger.Hooks.Add(lfshook.NewHook(pathMap, logger.Formatter))
	logger.Hooks.Add(&DefaultFieldHook{})
}

package tegenaria

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/sirupsen/logrus"
	"github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func TestLogger(t *testing.T) {
	convey.Convey("test logger", t, func() {
		log := GetLogger("test")
		log.Infof("testtest")
	})
	convey.Convey("test log level empty", t, func() {
		patch := gomonkey.ApplyFunc((*viper.Viper).GetString, func(_ *viper.Viper, _ string)string {
			return ""
		})
		defer patch.Reset()
		initLog()
		convey.So(logger.Level.String(),convey.ShouldContainSubstring,"info")
	})

	convey.Convey("test log level parser error", t, func() {
		patch := gomonkey.ApplyFunc(logrus.ParseLevel, func(_ string)(logrus.Level,error) {
			return logrus.ErrorLevel,errors.New("parse level error")
		})
		defer patch.Reset()
		f:=func(){
			initLog()
		}
		convey.So(f,convey.ShouldPanic)
	})
}

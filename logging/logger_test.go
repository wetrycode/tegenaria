package logger_test

import (
	"testing"

	logger "github.com/geebytes/go-scrapy/logging"
)

func TestLogger(t *testing.T){
	log := logger.GetLogger("test")
	log.Infof("testtest")
}

package tegenaria

import (
	"testing"

)

func TestLogger(t *testing.T) {
	log := GetLogger("test")
	log.Infof("testtest")
}

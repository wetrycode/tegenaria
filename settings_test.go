package tegenaria

import (
	"path/filepath"
	"testing"

	"github.com/smartystreets/goconvey/convey"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

var yamlExample = []byte(`
redis:
  addr: "127.0.0.1:6379"
  username: ""
  password: ""
log:
  level: "error"
`)

func AbsFilePath(t *testing.T, path string) string {
	t.Helper()

	s, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}

	return s
}
func TestSetting(t *testing.T) {
	convey.Convey("test settings load", t, func() {
		config := &Configuration{
			viper.New(),
		}
		fs := afero.NewMemMapFs()
		err := fs.Mkdir(AbsFilePath(t, "/etc/viper"), 0o777)
		convey.So(err, convey.ShouldBeNil)

		file, err := fs.Create(AbsFilePath(t, "/etc/viper/settings.yaml"))
		convey.So(err, convey.ShouldBeNil)

		_, err = fs.Stat("/etc/viper/settings.yaml")

		convey.So(err, convey.ShouldBeNil)

		_, err = file.Write(yamlExample)
		convey.So(err, convey.ShouldBeNil)
		config.SetFs(fs)
		ret := config.load("/etc/viper")
		convey.So(ret, convey.ShouldBeTrue)
		convey.So(config.GetString("redis.addr"), convey.ShouldContainSubstring, "127.0.0.1")
		convey.So(config.GetString("log.level"), convey.ShouldContainSubstring, "error")

	})
}

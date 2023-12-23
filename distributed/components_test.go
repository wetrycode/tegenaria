package distributed

import (
	"path"
	"runtime"
	"testing"

	"github.com/smartystreets/goconvey/convey"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/wetrycode/tegenaria"
)

var yamlExample = []byte(`
redis:
  addr: "127.0.0.1:16379"
  username: ""
  password: ""
  db: 0

influxdb:
  host: "127.0.0.1"
  port: 8086
  bucket: "tegenaria"
  org: "wetrycode"
  token: "9kCeULToVdZVdUGboG5qhiWaMiUUV1XKjKMlrFwPisNMKSENDAK3EBsuh_M-GFwGnNpQZkcZ6kPgltrDOHZo5g=="

log:
  path: "/var/logs/Tegenaria"
  level: "info"
`)

func TestNewDefaultComponents(t *testing.T) {
	convey.Convey("test new a default distributed components", t, func() {
		config := &tegenaria.Configuration{
			Viper: viper.New(),
		}
		var abPath string = ""

		_, filename, _, ok := runtime.Caller(0)
		if ok {
			abPath = path.Dir(filename)

		}
		fs := afero.NewMemMapFs()
		err := fs.Mkdir(tegenaria.AbsFilePathTest(t, abPath), 0o777)
		convey.So(err, convey.ShouldBeNil)
		settings := path.Join(abPath, "settings.yaml")
		file, err := fs.Create(tegenaria.AbsFilePathTest(t, settings))
		convey.So(err, convey.ShouldBeNil)

		_, err = fs.Stat(settings)

		convey.So(err, convey.ShouldBeNil)

		_, err = file.Write(yamlExample)
		convey.So(err, convey.ShouldBeNil)
		config.SetFs(fs)
		c := NewDefaultDistributedComponents()
		convey.So(c.statistic.influxClient.ServerURL(), convey.ShouldNotContainSubstring, "127.0.0.1:8086")
	})
}

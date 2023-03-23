module github.com/wetrycode/tegenaria

go 1.19

require (
	github.com/agiledragon/gomonkey/v2 v2.9.0
	github.com/alicebob/miniredis/v2 v2.30.0
	github.com/bits-and-blooms/bloom/v3 v3.3.1
	github.com/gin-gonic/gin v1.8.2
	github.com/go-kiss/monkey v0.0.0-20210912230757-40cda447d0e3
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.15.2
	github.com/influxdata/influxdb-client-go/v2 v2.12.2
	github.com/json-iterator/go v1.1.12
	github.com/redis/go-redis/v9 v9.0.1
	github.com/shopspring/decimal v1.3.1
	github.com/sirupsen/logrus v1.8.1
	github.com/smartystreets/goconvey v1.7.2
	github.com/sourcegraph/conc v0.2.0
	github.com/spaolacci/murmur3 v1.1.0
	github.com/spf13/cobra v1.4.0
	github.com/spf13/viper v1.14.0
	github.com/wxnacy/wgo v1.0.4
	github.com/yireyun/go-queue v0.0.0-20210520035143-72b190eafcba
	go.uber.org/ratelimit v0.2.0
	golang.org/x/net v0.7.0
	google.golang.org/genproto v0.0.0-20230223222841-637eb2293923
	google.golang.org/grpc v1.53.0
)

require (
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/andybalholm/cascadia v1.3.1 // indirect
	github.com/deepmap/oapi-codegen v1.8.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/goccy/go-json v0.9.11 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20181017120253-0766667cb4d1 // indirect
	github.com/influxdata/line-protocol v0.0.0-20200327222509-2487e7298839 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/pelletier/go-toml/v2 v2.0.6 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/smartystreets/assertions v1.2.0 // indirect
	github.com/yuin/gopher-lua v0.0.0-20220504180219-658193537a64 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/PuerkitoBio/goquery v1.8.0
	github.com/andres-erbsen/clock v0.0.0-20160526145045-9e14626cd129 // indirect
	github.com/bits-and-blooms/bitset v1.3.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.0 // indirect
	github.com/go-playground/universal-translator v0.18.0 // indirect
	github.com/go-playground/validator/v10 v10.11.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huandu/go-tls v1.0.1 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/mitchellh/mapstructure v1.5.0
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/spf13/afero v1.9.2
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.4.1 // indirect
	github.com/ugorji/go/codec v1.2.7 // indirect
	golang.org/x/arch v0.0.0-20210901143047-ebb09ed340f1 // indirect
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	google.golang.org/protobuf v1.28.1
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

// Undo the wrong version number
retract (
	// Undo the wrong version number
	v0.2.6
	// Undo the wrong version number
	v0.2.5
	// Undo the wrong version number
	v0.2.4
	// Undo the wrong version number
	v0.2.3
	// Undo the wrong version number
	v0.2.2
	// Undo the wrong version number
	v0.2.1
	// Undo the wrong version number
	v0.2.0
	// Undo the wrong version number
	v0.1.3
)
